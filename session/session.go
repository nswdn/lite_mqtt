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
	Conn        net.Conn
	ClientID    string
	Will        proto.Will
	KeepAlive   int // unit second
	LastPingReq time.Time

	Subscribed map[string]chan string
}

func New(conn net.Conn) *Session {
	s := new(Session)
	s.Conn = conn
	s.Subscribed = map[string]chan string{}
	return s
}

func (s *Session) Handle() {
	bytes := make([]byte, 1024)

	n, err := s.Conn.Read(bytes)
	if err != nil {
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

	go s.checkLastPingReq()

	for {
		n, err := s.Read(bytes)
		if err != nil {
			_ = s.Close()
			return
		}
		if err := s.processInteractive(bytes[:n]); err != nil {
			_ = s.Close()
			return
		}

		log.Println(bytes[:n])
	}
}

func (s *Session) processInteractive(received []byte) error {
	bits := calc.Bytes2Bits(received[0])
	ctrlPacket := proto.CalcControlPacket(bits[:4])
	log.Println("control packet: ", ctrlPacket)

	shouldRemainingLength := received[1]
	remainingLength := len(received) - 2
	if shouldRemainingLength != byte(remainingLength) {
		return proto.IncompletePacketErr
	}

	remain := received[2:]
	if ctrlPacket == proto.PPublish {
		return s.handlePublish(bits[4:], remain)
	}

	if ctrlPacket == proto.PPubACK || ctrlPacket == 4 {

	}

	if ctrlPacket == proto.PPubREC || ctrlPacket == 5 {
		s.Write(proto.NewCommonACK(proto.PPubREL, remain[0], remain[1]))
	}

	if ctrlPacket == proto.PPubREL || ctrlPacket == 6 {
		s.Write(proto.NewCommonACK(proto.PPubCOMP, remain[0], remain[1]))
	}

	if ctrlPacket == proto.PPubCOMP || ctrlPacket == 7 {

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

	ack := proto.NewSubscribeACK(subscribe.Qos, subscribe.MSBPacketID, subscribe.LSBPacketID)
	_, _ = s.Write(ack)

	return nil
}

func (s *Session) handlePublish(properties []uint8, remain []byte) error {
	publish := proto.Publish{}
	if err := publish.Decode(properties, remain); err != nil {
		return err
	}
	// todo publish
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
	s.KeepAlive = connect.KeepAlive
	return nil
}

func (s *Session) Write(b []byte) (int, error) {
	return s.Conn.Write(b)
}

func (s *Session) Read(b []byte) (int, error) {
	return s.Conn.Read(b)
}

func (s *Session) Close() error {
	return s.Conn.Close()
}

func (s *Session) checkLastPingReq() {
	var (
		now     time.Time
		elapsed time.Time
		ticker  *time.Ticker
	)

	now = time.Now()
	s.LastPingReq = now
	// elapsed = now - keepalive
	// if last before elapsed
	ticker = time.NewTicker(time.Duration(s.KeepAlive) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		elapsed = now.Add(-(time.Duration(s.KeepAlive) * time.Second))

		// timeout
		if s.LastPingReq.Before(elapsed) {
			_ = s.Close()
			return
		}

		now = time.Now()
	}

}
