package session

import (
	"errors"
	"excel_parser/calc"
	"excel_parser/proto"
	"log"
	"net"
	"time"
)

type Session struct {
	decoder     *decoder
	Conn        net.Conn
	ClientID    string
	Will        proto.Will
	KeepAlive   int // unit second
	LastPingReq time.Time

	PublishSubsChan chan []byte // this channel brings all subscribing topic's message
	PublishEndChan  chan byte   // write to this channel to stop select PublishSubsChan when disconnected or error

	Subscribing map[string]SubscribingTopic // subscribing topics. key: topic name, value topic's info
}

func init() {
	log.SetFlags(log.Lshortfile | log.Ltime)
}
func New(conn net.Conn) *Session {
	s := new(Session)
	s.decoder = NewDecoder()
	s.Conn = conn
	s.Subscribing = make(map[string]SubscribingTopic)

	s.PublishSubsChan = make(chan []byte, 100)
	s.PublishEndChan = make(chan byte, 1)
	return s
}

func (s *Session) Handle() {
	bytes := make([]byte, 1024)

	n, err := s.Read(bytes)
	if err != nil {
		_ = s.Close()
		return
	}

	// connect
	if err = s.processConn(bytes[:n]); err != nil {
		if err == proto.UnSupportedLevelErr {
			_, _ = s.Write(proto.NewConnACK(proto.ConnUnsupportedVersion))
		}
		_ = s.Close()
		return
	}

	// connect ack
	_, _ = s.Write(proto.NewConnACK(proto.ConnAccept))

	log.Println(s.ClientID, "connected")

	go s.publish()

	// todo unpack first, headerLen length can be 4 bytes.
	// fixed header maximum length 5 bytes.
	// last byte of headerLen length x: 128 > x > 0, previous bytes y: 256 > y > 0.
	// only publish packet's headerLen bytes length can be orver than 1 byte
	for {
		n, err := s.Read(bytes)
		log.Println(bytes[:n])
		if err != nil {
			_ = s.Close()
			return
		}
		_ = s.decoder.decode(bytes[:n])
		if s.decoder.end {
			if err := s.processInteractive(s.decoder.readAll(), s.decoder.headerLen); err != nil {
				_ = s.Close()
			}
			s.decoder = NewDecoder()
		}
	}
}

func (s *Session) processInteractive(received []byte, headerLen int) error {
	bits := calc.Bytes2Bits(received[0])
	ctrlPacket := proto.CalcControlPacket(bits[:4])
	log.Println("control packet", ctrlPacket)

	if ctrlPacket == proto.PPubACK || ctrlPacket == 4 {
		return nil
	}

	if ctrlPacket == proto.PPubCOMP || ctrlPacket == 7 {

	}

	remain := received[headerLen:]

	if ctrlPacket == proto.PPubREC || ctrlPacket == 5 {
		s.Write(proto.NewCommonACK(proto.PPubREL, remain[0], remain[1]))
		return nil
	}

	if ctrlPacket == proto.PPubREL || ctrlPacket == 6 {
		s.Write(proto.NewCommonACK(proto.PPubCOMP, remain[0], remain[1]))
		return nil
	}

	if ctrlPacket == proto.PPublish {
		return s.handlePublish(bits[4:], remain)
	}

	if ctrlPacket == proto.PSubscribe {
		return s.handleSubscribe(remain)
	}

	if ctrlPacket == proto.PUnsubscribe {
		return s.handleUnsubscribe(remain)
	}

	if ctrlPacket == proto.PPingREQ {
		return s.handlePingREQ()
	}

	if ctrlPacket == proto.PDisconnect {
		return s.handleDisconnect()
	}

	return nil
}

func (s *Session) handleDisconnect() error {
	_ = s.Close()
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

	ack := proto.NewCommonACK(proto.PUnsubscribeACK, us.MSBPacketID, us.LSBPacketID)
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

	for _, topic := range subscribe.Topic {
		receiverChan := make(chan []byte)
		stopChan := make(chan byte)
		sessionCloseChan := b.GetTopic(topic, s.ClientID, receiverChan)
		subscribingTopic := NewSubscribeTopic(sessionCloseChan, receiverChan, stopChan, topic, max, s.PublishSubsChan)
		s.Subscribing[topic] = subscribingTopic
	}

	ack := proto.NewSubscribeACK(subscribe.Qos, subscribe.MSBPacketID, subscribe.LSBPacketID)
	_, _ = s.Write(ack)

	return nil
}

func (s *Session) handlePublish(properties []uint8, remain []byte) error {
	publish := proto.Publish{}
	if err := publish.Decode(properties, remain); err != nil {
		return err
	}

	// todo publish, if retain == 1 store message
	b.Publish(publish.Topic, publish.Payload)

	if publish.Qos == proto.AtLeaseOne {
		_, _ = s.Write(proto.NewCommonACK(proto.PPubACK, publish.MSBPacketID, publish.LSBPacketID))
		return nil
	}

	if publish.Qos == proto.MustOne {
		_, _ = s.Write(proto.NewCommonACK(proto.PPubREC, publish.MSBPacketID, publish.LSBPacketID))
		return nil
	}

	return nil
}

func (s *Session) processConn(received []byte) error {
	bits := calc.Bytes2Bits(received[0])
	ctrlPacket := proto.CalcControlPacket(bits[:4])

	if ctrlPacket != proto.PConnect {
		return proto.UnConnErr
	}

	connect := proto.Connect{}
	if err := connect.Decode(nil, received[4:]); err != nil {
		return err
	}

	s.Will = connect.Will
	s.ClientID = connect.ClientID
	s.KeepAlive = connect.KeepAlive
	return nil
}

func (s *Session) Write(b []byte) (int, error) {
	return s.Conn.Write(b)
}

func (s *Session) Read(b []byte) (int, error) {
	if s.KeepAlive != 0 {
		_ = s.Conn.SetReadDeadline(time.Now().Add(time.Second * time.Duration(s.KeepAlive*2)))
	}
	return s.Conn.Read(b)
}

func (s *Session) Close() error {
	s.PublishEndChan <- 0
	for key, topic := range s.Subscribing {
		topic.StopChan <- 0
		topic.UnsubscribeChan <- s.ClientID
		delete(s.Subscribing, key)
	}
	log.Println(s.ClientID, "closed")
	return s.Conn.Close()
}

func (s *Session) publish() {
loop:
	for {
		select {
		case resp := <-s.PublishSubsChan:
			_, _ = s.Write(resp)
		case <-s.PublishEndChan:
			log.Println("stop publish to: ", s.ClientID)
			break loop
		}
	}
}

type SubscribingTopic struct {
	UnsubscribeChan chan string // write session id to the channel to broker to stop listening topic's message
	StopChan        chan byte   // a signal channel to stop select Receiver
}

// subscribe a new topic. receiver channel will get new message
func NewSubscribeTopic(unsubChan chan string, receiver chan []byte, stopChan chan byte, topic string, qos proto.QOS, publishMsg chan []byte) SubscribingTopic {

	// todo qos
	s := SubscribingTopic{
		UnsubscribeChan: unsubChan,
		StopChan:        stopChan,
	}

	go func() {
	loop:
		for {
			select {
			case msg := <-receiver:
				publish, err := proto.NewPublish(0, qos, 0, topic, msg)
				if err != nil {
					log.Println(err)
					continue
				}
				publishMsg <- publish
			case <-stopChan:
				log.Println("stop listen topic: ", topic)
				break loop
			}
		}
	}()

	return s
}
