package session

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"excel_parser/broker"
	"excel_parser/config"
	"excel_parser/database"
	"excel_parser/proto"
	"fmt"
	"golang.org/x/sync/semaphore"
	"io"
	"log"
	"net"
	"os"
	"time"
)

var concurrentOutputSemaphore *semaphore.Weighted

type Session struct {
	Conn         net.Conn
	disconnected bool // true if recv disconnect command
	terminated   bool // true connection closed
	ClientID     string
	KeepAlive    int // second
	LastPingReq  time.Time
	Will         proto.Will

	Subscribing map[string]string // subscribing topics. key: topic name, value: topic name

	publishChan chan []byte // contains subscribing topic's publish packet
}

var (
	clogger     *log.Logger
	closeSignal = []byte{0}
)

func init() {
	c, _ := os.OpenFile("connected.txt", os.O_TRUNC|os.O_CREATE|os.O_WRONLY, os.ModePerm)
	clogger = log.New(io.MultiWriter(c, os.Stdout), "", 0)
	log.SetFlags(log.Lshortfile | log.Ltime)
	concurrentOutputSemaphore = semaphore.NewWeighted(20)
}

func New(conn net.Conn) {
	session := &Session{
		Conn:        conn,
		Subscribing: make(map[string]string),
		publishChan: make(chan []byte, 100),
	}
	handle(session)
}

func handle(s *Session) {
	var (
		n    int
		err  error
		recv = make([]byte, 1024)
	)

	n, err = s.Read(recv)
	if err != nil {
		_ = s.Conn.Close()
		return
	}

	// connect
	if err = s.processConn(recv[:n]); err != nil {
		_ = s.Conn.Close()
		return
	}

	log.Println(s.ClientID)

	go s.listenSubscribe()

	// fixed header maximum length 5 bytes.
	// last byte of headerLen length x: 128 > x > 0, previous bytes y: 256 > y > 0.
	// only listenPublish packet's headerLen bytes length can be over than 1 byte
	var decoded content
	for {
		n, err = s.Read(recv)
		if err != nil {
			_ = s.Close()
			return
		}
		decoded, err = decode(s.Conn, recv[:n])
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
		s.handleSubscribe(content.body)
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

	for _, topic := range us.Topic {
		broker.Unsubscribe(topic, s.ClientID)
		delete(s.Subscribing, topic)
	}

	ack := proto.NewCommonACK(proto.PUnsubscribeACK, us.PacketID)
	_, _ = s.Write(ack)

	return
}

func (s *Session) handleSubscribe(remain []byte) {

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

	for _, topic := range subscribe.Topic {
		s.Subscribing[topic] = topic
		broker.Subscribe(topic, s.ClientID, s.publishChan, max)
	}
}

func (s *Session) handlePublish(properties []uint8, remain []byte) {
	publish := proto.Publish{}
	if err := publish.Decode(properties, remain); err != nil {
		_ = s.Conn.Close()
		return
	}

	if config.SavePublish() {
		database.Save(s.ClientID, publish.Payload)
	}

	broker.Publish(publish.Topic, publish.Retain, publish.Payload)

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

	if config.EnableAuth() {
		if !database.Auth(connect.User, connect.Pwd) {
			_, _ = s.Write(proto.NewConnACK(proto.ConnBadUserPwd))
			return errors.New("auth failed")
		}
	}

	s.Will = connect.Will
	s.ClientID = connect.ClientID
	s.KeepAlive = connect.KeepAlive

	sessionStore.put(s.ClientID, s)

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

	if s.terminated {
		return errors.New("session has been closed")
	}

	s.terminated = true

	if s.disconnected {
		broker.Publish(s.Will.Topic, s.Will.Retain, s.Will.Payload)
	}

	sessionStore.delete(s.ClientID)

	for topic := range s.Subscribing {
		broker.Unsubscribe(topic, s.ClientID)
		delete(s.Subscribing, topic)
		clogger.Printf("%s is unsubscribing topic: %s", s.ClientID, topic)
	}

	clogger.Printf("%s is disconnected", s.ClientID)
	return s.Conn.Close()
}

func (s *Session) listenSubscribe() {
loop:
	for {
		select {
		case msg := <-s.publishChan:
			for concurrentOutputSemaphore.Acquire(context.Background(), 1) != nil {
			}

			if bytes.Compare(msg, closeSignal) == 0 {
				break loop
			}

			err := s.Conn.SetWriteDeadline(time.Now().Add(time.Second * 5))
			_, err = s.Conn.Write(msg)
			if err != nil {
				log.Printf("failed to write to %s, content: [%v], err: [%s]", s.ClientID, msg, err)
			}

			concurrentOutputSemaphore.Release(1)
		}
	}

	_ = s.Close()
}
