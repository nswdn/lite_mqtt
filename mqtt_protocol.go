package main

import (
	"bytes"
	"errors"
	"excel_parser/proto"
	"log"
	"net"
	"time"
)

type MQTTControlPacket byte
type MQTTVersion byte

const ProtocolName = "MQTT"

// Support MQTT Level 3.1/3.1.1/5
var supportMQTTLevel = map[MQTTVersion]bool{
	4: true,
}

// MQTT Control Packet
const (
	CONNECT MQTTControlPacket = 1
	CONNACK MQTTControlPacket = 32 // 2

	Publish MQTTControlPacket = 3
	PubAck  MQTTControlPacket = 64 // 4
	PubREC  MQTTControlPacket = 80 // 5
	PubREL  MQTTControlPacket = 6
	PubCOMP MQTTControlPacket = 112 //7

	Subscribe      MQTTControlPacket = 8
	SubAck         MQTTControlPacket = 144 //9
	Unsubscribe    MQTTControlPacket = 162 // 10
	UnsubscribeACK MQTTControlPacket = 176 // 11

	KeepAlive MQTTControlPacket = 192 // 12
)

var (
	UnConnErr           = errors.New("first control packet could only be CONNECT")
	IncompletePacketErr = errors.New("incomplete packet")
	UnSupportedLevelErr = errors.New("support protocol MQTT v3.1.1 only")
	InvalidErr          = errors.New("error packet")
)

type connectFlags struct {
	userNameFlag   bool
	PwdFlag        bool
	willRetainFlag bool
	willQos        proto.QOS
	willFlag       bool
	cleanSession   int
	reserved       bool
}

type publishFlags struct {
	dupFlag bool
	qos     proto.QOS
	retain  bool
}

type MQTT struct {
	client        net.Conn
	ControlPacket MQTTControlPacket

	WillFlag bool
	Will     proto.Will

	Topic        string
	ClientID     string
	KeepAlive    int
	User         string
	Pwd          string
	CleanSession int
	Reserved     bool
}

func Decode(client net.Conn, received []byte) (*MQTT, error) {

	var decoded = new(MQTT)

	if decoded.client == nil {
		decoded.client = client

	}
	bits := Bytes2Bits(received[0])
	ctrlPacket := calcControlPacket(bits[:4])
	log.Println("control packet: ", ctrlPacket)

	shouldRemainingLength := received[1]
	remainingLength := len(received) - 2
	if shouldRemainingLength != byte(remainingLength) {
		return nil, IncompletePacketErr
	}

	if ctrlPacket == Publish {
		if err := decoded.handlePublish(bits[4:], received[2:]); err != nil {
			return nil, err
		}
	}

	if ctrlPacket == PubREL {
		if calcControlPacket(bits[4:]) != 2 {
			return nil, InvalidErr
		}
		decoded.handlePubREL(received[2:])
	}

	if ctrlPacket == Subscribe {
		if calcControlPacket(bits[4:]) != 2 {
			return nil, InvalidErr
		}
		if err := decoded.handleSubscribe(received[2:]); err != nil {
			return nil, err
		}
	}

	return decoded, nil
}

func (mq *MQTT) handlePublish(properties []uint8, remain []byte) error {
	log.Println(properties)

	// dup, qos, retain
	flag := parsePublishFlag(properties)
	log.Println("publish flag: ", flag)

	// topic
	buffer := bytes.NewBuffer(remain)
	_, _ = buffer.ReadByte()
	lsbTopicLen, _ := buffer.ReadByte()
	topic := make([]byte, lsbTopicLen)
	_, _ = buffer.Read(topic)

	// unique identifier, exist in Qos level 1&2 only
	if flag.qos != 0 {
		msbPacketId, _ := buffer.ReadByte()
		lsbPacketId, _ := buffer.ReadByte()
		log.Println("packet id: ", lsbPacketId)

		ack := []byte{
			0,
			2,
			msbPacketId,
			lsbPacketId,
		}

		if flag.qos == proto.AtLeaseOne {
			ack[0] = byte(PubAck)
		} else {
			ack[0] = byte(PubREC)
		}

		_, _ = mq.client.Write(ack)

	}

	// todo publish
	payload := buffer.Bytes()
	log.Printf("publish to %s, msg: %s\n", topic, payload)

	return nil
}

func (mq *MQTT) handlePubREL(packetIDs []byte) {

	// todo check publish success
	msbPacketId := packetIDs[0]
	lsbPacketId := packetIDs[1]

	ack := []byte{
		byte(PubCOMP),
		2,
		msbPacketId,
		lsbPacketId,
	}

	_, _ = mq.client.Write(ack)
}

func (mq *MQTT) handleSubscribe(remain []byte) error {
	buffer := bytes.NewBuffer(remain)
	msbPacketId, _ := buffer.ReadByte()
	lsbPacketId, _ := buffer.ReadByte()
	log.Println(msbPacketId, lsbPacketId)

	ack := make([]byte, 4)
	ack[0] = byte(SubAck)
	ack[1] = 2
	ack[2] = msbPacketId
	ack[3] = lsbPacketId

	// get topic and it's qos
	for {
		if _, err := buffer.ReadByte(); err != nil {
			break
		}
		lsbTopicLen, _ := buffer.ReadByte()
		topic := make([]byte, lsbTopicLen)
		_, _ = buffer.Read(topic)
		qos, _ := buffer.ReadByte()
		// todo upper 6 bits to of qos reserved for future version use
		bits := Bytes2Bits(qos)
		qos = bits[6]*2 + bits[7]
		if qos > 2 {
			return InvalidErr
		}
		ack = append(ack, qos)
		ack[1] = ack[1] + 1
		// todo on failures qos = 128
		log.Println(string(topic), qos)
	}

	_, _ = mq.client.Write(ack)

	// todo temp a client subscribed topics listen topics when they got new publish.
	go func() {
		buffer := bytes.NewBuffer(nil)
		buffer.WriteByte(50)
		buffer.WriteByte(16)
		buffer.WriteByte(0)
		buffer.WriteByte(4)
		buffer.WriteString("test")
		buffer.WriteByte(0)
		buffer.WriteByte(1)
		buffer.WriteString("response")
		for {
			time.Sleep(50 * time.Millisecond)
			_, _ = mq.client.Write(buffer.Bytes())
		}
	}()
	return nil
}

func Bytes2Bits(data byte) []uint8 {
	dst := make([]uint8, 0)
	for i := 0; i < 8; i++ {
		move := uint(7 - i)
		dst = append(dst, (data>>move)&1)
	}
	return dst
}

func parsePublishFlag(bits []uint8) publishFlags {
	var flag publishFlags

	flag.dupFlag = bits[0] == 1
	flag.qos = proto.QOS(bits[1]*2 + bits[2])
	flag.retain = bits[3] == 1

	return flag
}

func calcControlPacket(binaryArr []uint8) MQTTControlPacket {
	var res MQTTControlPacket

	if binaryArr[0] == 1 {
		res += 8
	}

	if binaryArr[1] == 1 {
		res += 4
	}

	if binaryArr[2] == 1 {
		res += 2
	}

	if binaryArr[3] == 1 {
		res += 1
	}

	return res
}
