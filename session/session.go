package session

import (
	"encoding/binary"
	"errors"
	"excel_parser/proto"
	"excel_parser/trie"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sync"
	"time"
)

type SubscribingTopic struct {
	ReceiveChan chan []byte // a signal channel to stop select Receiver
}

type Session struct {
	Conn         net.Conn
	decoder      *decoder
	disconnected bool
	ClientID     string
	KeepAlive    int // second
	LastPingReq  time.Time
	Will         proto.Will

	wg              *sync.WaitGroup
	ProcessStopChan chan struct{}

	mutex       sync.Mutex
	Subscribing map[string]*SubscribingTopic // subscribing topics. key: topic name, value topic's info

	closed bool
}

var (
	clogger *log.Logger
)

func init() {
	c, _ := os.OpenFile("connected.txt", os.O_TRUNC|os.O_CREATE|os.O_WRONLY, os.ModePerm)
	clogger = log.New(io.MultiWriter(c, os.Stdout), "", 0)
	log.SetFlags(log.Lshortfile | log.Ltime)
}

func New(conn net.Conn) {
	session := &Session{
		wg:              &sync.WaitGroup{},
		Conn:            conn,
		Subscribing:     make(map[string]*SubscribingTopic),
		ProcessStopChan: make(chan struct{}),
	}
	session.Handle()
}

func (s *Session) Handle() {
	var (
		n     int
		err   error
		bytes = make([]byte, 1024)
	)

	n, err = s.Read(bytes)
	if err != nil {
		_ = s.Conn.Close()
		return
	}

	// connect
	if err = s.processConn(bytes[:n]); err != nil {
		_ = s.Conn.Close()
		return
	}

	s.decoder = NewDecoder()

	// fixed header maximum length 5 bytes.
	// last byte of headerLen length x: 128 > x > 0, previous bytes y: 256 > y > 0.
	// only listenPublish packet's headerLen bytes length can be over than 1 byte
	for {
		n, err = s.Read(bytes)
		if err != nil {
			_ = s.Close()
			return
		}
		decoded, err := s.decoder.decode(bytes[:n])
		if err != nil {
			continue
		}
		s.processInteractive(decoded)
	}
}

func (s *Session) processInteractive(content content) {

	switch content.ctrlPacket {
	case proto.PPublish:
		s.handlePublish(content.properties, content.body)
	case proto.PPubACK:
	case proto.PPubREC:
		_, _ = s.Write(proto.NewCommonACK(proto.PPubRELAlia, binary.BigEndian.Uint16(content.body)))
	case proto.PPubREL:
		_, _ = s.Write(proto.NewCommonACK(proto.PPubCOMPAlia, binary.BigEndian.Uint16(content.body)))
	case proto.PPubCOMP:
	case proto.PSubscribe:
		clogger.Println(&s)
		go s.handleSubscribe(content.body)
	case proto.PUnsubscribe:
		s.handleUnsubscribe(content.body)
	case proto.PPingREQ:
		s.handlePingREQ()
	case proto.PDisconnect:
		s.handleDisconnect()
	default:
		_ = s.Conn.Close()
	}
}

func (s *Session) handleDisconnect() {
	s.disconnected = true
	_ = s.Conn.Close()
	return
}

func (s *Session) handlePingREQ() {
	s.LastPingReq = time.Now()
	_, _ = s.Write(proto.NewPingRESP())
}

func (s *Session) handleUnsubscribe(remain []byte) {
	us := proto.UnSubscribe{}
	if err := us.Decode(nil, remain); err != nil {
		_ = s.Conn.Close()
		return
	}

	s.mutex.Lock()
	for _, topic := range us.Topic {
		subscribingTopic := s.Subscribing[topic]
		trie.Unsubscribe([]string{topic}, s.ClientID)
		close(subscribingTopic.ReceiveChan)
		delete(s.Subscribing, topic)
	}
	s.mutex.Unlock()

	ack := proto.NewCommonACK(proto.PUnsubscribeACK, us.PacketID)
	_, _ = s.Write(ack)

	return
}

func (s *Session) handleSubscribe(remain []byte) {
	clogger.Println(&s)

	subscribe := proto.Subscribe{}
	if err := subscribe.Decode(nil, remain); err != nil {
		_ = s.Conn.Close()
		return
	}

	// todo topic and qos, if subscribe on failure max = 128
	var max proto.QOS
	for _, q := range subscribe.Qos {
		if q > max {
			max = q
		}
	}

	ack := proto.NewSubscribeACK(subscribe.Qos, subscribe.PacketID)
	_, _ = s.Write(ack)
	s.subscribe(max, subscribe)
}

func (s *Session) subscribe(max proto.QOS, subscribe proto.Subscribe) {

	if s.closed {
		return
	}

	receiverChan := make(chan []byte, 100)
	trie.Subscribe(subscribe.Topic[0], s.ClientID, receiverChan)
	s.mutex.Lock()
	s.Subscribing[subscribe.Topic[0]] = &SubscribingTopic{
		ReceiveChan: receiverChan,
	}
	s.mutex.Unlock()
	s.listenSubscribe(receiverChan, subscribe.Topic[0], max)

	//for _, topic := range subscribe.Topic {
	//	receiverChan := make(chan []byte, 100)
	//	trie.Subscribe(topic, s.ClientID, receiverChan)
	//	s.Subscribing[topic] = &SubscribingTopic{
	//		ReceiveChan: receiverChan,
	//	}
	//	t := topic
	//	go s.listenSubscribe(receiverChan, t, max)
	//}
}

func (s *Session) handlePublish(properties []uint8, remain []byte) {
	publish := proto.Publish{}
	if err := publish.Decode(properties, remain); err != nil {
		_ = s.Conn.Close()
		return
	}

	trie.Publish(publish.Topic, publish.Retain, publish.Payload)

	if publish.Qos == proto.AtLeaseOne {
		_, _ = s.Write(proto.NewCommonACK(proto.PPubACKAlia, publish.PacketID))
	}

	if publish.Qos == proto.MustOne {
		_, _ = s.Write(proto.NewCommonACK(proto.PPubRECAlia, publish.PacketID))
	}

}

func (s *Session) processConn(received []byte) error {

	if proto.MQTTControlPacket(received[0]) != proto.PConnectAlia {
		return errors.New(fmt.Sprintf("invalid control packet: [%b]", received[0]))
	}

	connect := proto.Connect{}
	if err := connect.Decode(nil, received[4:]); err != nil {
		return err
	}

	s.Will = connect.Will
	s.ClientID = connect.ClientID
	s.KeepAlive = connect.KeepAlive

	_, _ = s.Write(proto.NewConnACK(proto.ConnAccept))
	return nil
}

func (s *Session) Write(b []byte) (int, error) {
	write, err := s.Conn.Write(b)
	return write, err
}

func (s *Session) Read(b []byte) (int, error) {
	if s.KeepAlive != 0 {
		_ = s.Conn.SetReadDeadline(time.Now().Add(time.Second * time.Duration(s.KeepAlive*2)))
	}
	return s.Conn.Read(b)
}

func (s *Session) Close() error {
	var i = 0
	s.mutex.Lock()
	s.closed = true

	topicNames := make([]string, len(s.Subscribing))
	for topicName := range s.Subscribing {
		topicNames[i] = topicName
		i++
	}
	trie.Unsubscribe(topicNames, s.ClientID)

	for topicName, topic := range s.Subscribing {
		close(topic.ReceiveChan)
		delete(s.Subscribing, topicName)
	}

	s.mutex.Unlock()

	s.decoder.Close()
	if !s.disconnected {
		trie.Publish(s.Will.Topic, s.Will.Retain, s.Will.Payload)
	}
	return s.Conn.Close()
}

func (s *Session) listenSubscribe(receiveChan chan []byte, topic string, qos proto.QOS) {

loop:
	for {
		select {
		case msg, ok := <-receiveChan:
			if !ok {
				break loop
			}
			publish, err := proto.NewPublish(0, qos, 0, topic, msg)
			if err != nil {
				log.Println(err)
				continue
			}
			_, _ = s.Conn.Write(publish)
		}
	}
}
