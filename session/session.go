package session

import (
	"encoding/binary"
	"errors"
	"excel_parser/proto"
	"excel_parser/trie"
	"fmt"
	"log"
	"net"
	"time"
)

type Session struct {
	decoder         *decoder
	disconnected    bool
	Conn            net.Conn
	ClientID        string
	Will            proto.Will
	KeepAlive       int // second
	LastPingReq     time.Time
	ProcessStopChan chan byte

	PublishSubsChan chan []byte // this channel brings all subscribing topic's message
	PublishEndChan  chan byte   // write to this channel to stop select PublishSubsChan when disconnected or error

	Subscribing map[string]*SubscribingTopic // subscribing topics. key: topic name, value topic's info
}

func init() {
	log.SetFlags(log.Lshortfile | log.Ltime)
}
func New(conn net.Conn) *Session {
	session := &Session{
		decoder:         NewDecoder(),
		Conn:            conn,
		ProcessStopChan: make(chan byte, 1),
		Subscribing:     make(map[string]*SubscribingTopic),
		PublishSubsChan: make(chan []byte, 100),
		PublishEndChan:  make(chan byte, 1),
	}
	return session
}

func (s *Session) Handle() {
	var (
		n     int
		err   error
		bytes = make([]byte, 1024)
	)

	n, err = s.Read(bytes)
	if err != nil {
		_ = s.Close()
		return
	}

	// connect
	if err = s.processConn(bytes[:n]); err != nil {
		_ = s.Close()
		return
	}

	// connect ack

	log.Println(s.ClientID, "connected")

	go s.processInteractive()
	go s.publish()

	// fixed header maximum length 5 bytes.
	// last byte of headerLen length x: 128 > x > 0, previous bytes y: 256 > y > 0.
	// only listenPublish packet's headerLen bytes length can be over than 1 byte

	for {
		n, err = s.Read(bytes)
		if err != nil {
			_ = s.Close()
			return
		}
		s.decoder.decode(bytes[:n])
	}
}

func (s *Session) processInteractive() {
	var (
		content content
		err     error
	)
loop:
	for {
		select {
		case content = <-s.decoder.publishChan:
			err = s.handlePublish(content.properties, content.body)
		case content = <-s.decoder.publishAckChan:
		case content = <-s.decoder.publishRECChan:
			s.Write(proto.NewCommonACK(proto.PPubRELAlia, binary.BigEndian.Uint16(content.body)))
		case content = <-s.decoder.publishRELChan:
			s.Write(proto.NewCommonACK(proto.PPubCOMPAlia, binary.BigEndian.Uint16(content.body)))
		case content = <-s.decoder.subscribeChan:
			err = s.handleSubscribe(content.body)
		case content = <-s.decoder.unsubscribeChan:
			err = s.handleUnsubscribe(content.body)
		case content = <-s.decoder.pingREQChan:
			err = s.handlePingREQ()
		case content = <-s.decoder.disconnectChan:
			err = s.handleDisconnect()
		case <-s.ProcessStopChan:
			break loop
		}

		if err != nil {
			s.Conn.Close()
			break loop
		}
	}

}

func (s *Session) handleDisconnect() error {
	s.disconnected = true
	return errors.New("disconnect")
}

func (s *Session) handlePingREQ() error {
	s.LastPingReq = time.Now()
	_, _ = s.Write(proto.NewPingRESP())
	return nil
}

func (s *Session) handleUnsubscribe(remain []byte) error {
	us := proto.UnSubscribe{}
	if err := us.Decode(nil, remain); err != nil {
		return err
	}

	// todo unsubscribe
	for _, topic := range us.Topic {
		subscribingTopic := s.Subscribing[topic]
		subscribingTopic.UnsubscribeChan <- s.ClientID
		subscribingTopic.StopChan <- 0
		delete(s.Subscribing, topic)
	}

	ack := proto.NewCommonACK(proto.PUnsubscribeACK, us.PacketID)
	_, _ = s.Write(ack)

	return nil
}

func (s *Session) handleSubscribe(remain []byte) error {
	subscribe := proto.Subscribe{}
	if err := subscribe.Decode(nil, remain); err != nil {
		return err
	}

	// todo topic and qos, if subscribe on failure max = 128
	var max proto.QOS
	for _, q := range subscribe.Qos {
		if q > max {
			max = q
		}
	}

	go s.asyncSubscribe(max, &subscribe)

	ack := proto.NewSubscribeACK(subscribe.Qos, subscribe.PacketID)
	_, _ = s.Write(ack)

	return nil
}

func (s *Session) asyncSubscribe(max proto.QOS, subscribe *proto.Subscribe) {
	for _, topic := range subscribe.Topic {
		receiverChan := make(chan []byte, 100)
		stopChan := make(chan byte)
		sessionCloseChan := trie.GetTopic(topic, s.ClientID, receiverChan)
		subscribingTopic := NewSubscribeTopic(sessionCloseChan, receiverChan, s.PublishSubsChan, stopChan, topic, max)
		s.Subscribing[topic] = subscribingTopic
	}
}

func (s *Session) handlePublish(properties []uint8, remain []byte) error {
	publish := proto.Publish{}
	if err := publish.Decode(properties, remain); err != nil {
		return err
	}

	// todo listenPublish, if retain == 1 store message
	trie.Publish(publish.Topic, publish.Retain, publish.Payload)

	if publish.Qos == proto.AtLeaseOne {
		_, _ = s.Write(proto.NewCommonACK(proto.PPubACKAlia, publish.PacketID))
		return nil
	}

	if publish.Qos == proto.MustOne {
		_, _ = s.Write(proto.NewCommonACK(proto.PPubRECAlia, publish.PacketID))
		return nil
	}

	return nil
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
	if err != nil {
		fmt.Println(err)
	}
	return write, err
}

func (s *Session) Read(b []byte) (int, error) {
	if s.KeepAlive != 0 {
		_ = s.Conn.SetReadDeadline(time.Now().Add(time.Second * time.Duration(s.KeepAlive*2)))
	}
	return s.Conn.Read(b)
}

func (s *Session) Close() error {
	if s.disconnected {
		log.Println(s.Will.Payload)
		trie.Publish(s.Will.Topic, s.Will.Retain, s.Will.Payload)
	}

	s.PublishEndChan <- 0
	for key, topic := range s.Subscribing {
		topic.StopChan <- 0
		topic.UnsubscribeChan <- s.ClientID
		delete(s.Subscribing, key)
	}
	s.ProcessStopChan <- 1
	s.decoder.Close()
	return s.Conn.Close()
}

func (s *Session) publish() {
loop:
	for {
		select {
		case resp := <-s.PublishSubsChan:
			_, _ = s.Write(resp)
		case <-s.PublishEndChan:
			log.Println("stop listenPublish to: ", s.ClientID)
			break loop
		}
	}
}

type SubscribingTopic struct {
	UnsubscribeChan chan string // write session id to the channel to broker to stop listening topic's message
	StopChan        chan byte   // a signal channel to stop select Receiver
}

// subscribe a new topic. receiver channel will get new message
func NewSubscribeTopic(unsubChan chan string, receiver, publishMsg chan []byte, stopChan chan byte, topic string, qos proto.QOS) *SubscribingTopic {

	// todo qos
	s := &SubscribingTopic{
		UnsubscribeChan: unsubChan,
		StopChan:        stopChan,
	}

	go listenSubscribe(receiver, publishMsg, stopChan, topic, qos)

	return s
}

func listenSubscribe(receiveChan, publishChan chan []byte, stopChan chan byte, topic string, qos proto.QOS) {
	var (
		msg, publish []byte
		err          error
	)
loop:
	for {
		select {
		case msg = <-receiveChan:
			publish, err = proto.NewPublish(0, qos, 0, topic, msg)
			if err != nil {
				log.Println(err)
				continue
			}
			publishChan <- publish
		case <-stopChan:
			break loop
		}
	}
}
