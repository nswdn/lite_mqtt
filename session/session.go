package session

import (
	"excel_parser/calc"
	"excel_parser/proto"
	"log"
	"net"
)

type Session struct {
	Conn     net.Conn
	ClientID string
}

func New(conn net.Conn) *Session {
	s := new(Session)
	s.Conn = conn
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

	for {
		n, err := s.Read(bytes)
		if err != nil {
			_ = s.Close()
			return
		}
		go s.processInteractive(bytes[:n])
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
