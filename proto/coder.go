package proto

import (
	"bytes"
	"encoding/binary"
	"errors"
	"excel_parser/calc"
	"math"
	"math/rand"
	"strings"
)

type QOS int
type MQTTVersion byte
type ConnRespCode byte
type MQTTControlPacket byte

// Support MQTT Level 3.1/3.1.1/5
var supportMQTTLevel = map[MQTTVersion]bool{
	4: true,
}
var (
	UnConnErr           = errors.New("first control packet could only be CONNECT")
	IncompletePacketErr = errors.New("incomplete packet")
	UnSupportedLevelErr = errors.New("support protocol MQTT v3.1.1 only")
	InvalidErr          = errors.New("error packet")
	QosOutOfBoundErr    = errors.New("qos support 0,1,2 only")
)

const ProtocolName = "MQTT"
const MaximumRemainLen = 128*128*128*128 - 1 // 268,435,454

const (
	ConnAccept             ConnRespCode = 0
	ConnUnsupportedVersion ConnRespCode = 1
	ConnClientReject       ConnRespCode = 2
	ConnServerUnavailable  ConnRespCode = 3
	ConnBadUserPwd         ConnRespCode = 4
	ConnNotAuthorized      ConnRespCode = 5
)

const (
	AtMaxOne   QOS = 0
	AtLeaseOne QOS = 1
	MustOne    QOS = 2
	BADQoS     QOS = 128
)

// MQTT Control Packet
const (
	PConnect     MQTTControlPacket = 1
	PConnectAlia MQTTControlPacket = 16
	PConnACK     MQTTControlPacket = 32 // 2

	PPublish     MQTTControlPacket = 3
	PPubACK      MQTTControlPacket = 4 // 4
	PPubACKAlia  MQTTControlPacket = 64
	PPubREC      MQTTControlPacket = 5 // 5
	PPubRECAlia  MQTTControlPacket = 80
	PPubREL      MQTTControlPacket = 6 // 6
	PPubRELAlia  MQTTControlPacket = 98
	PPubCOMP     MQTTControlPacket = 7   //7
	PPubCOMPAlia MQTTControlPacket = 112 //7

	PSubscribe      MQTTControlPacket = 8
	PSubACK         MQTTControlPacket = 144 //9
	PUnsubscribe    MQTTControlPacket = 10  // 162
	PUnsubscribeACK MQTTControlPacket = 176 // 11

	PPingREQ    MQTTControlPacket = 12  // 192
	PPingRESP   MQTTControlPacket = 208 // 13
	PDisconnect MQTTControlPacket = 14  // 14

)

type connectFlags struct {
	userNameFlag   bool
	PwdFlag        bool
	willRetainFlag bool
	willQos        QOS
	willFlag       bool
	cleanSession   bool
	reserved       bool
}

type publishFlags struct {
	dupFlag bool
	qos     QOS
	retain  bool
}

type Protocol interface {
	Decode([]byte, []byte) error
}
type Will struct {
	Topic   string
	Payload []byte
	Qos     QOS
	Retain  bool
}

type Connect struct {
	ClientID     string
	KeepAlive    int
	WillFlag     bool
	Will         Will
	User         string
	Pwd          string
	CleanSession bool
	Reserved     bool
}

// properties: nil
// remain: packet without fixed header
// UnSupportedLevelErr if protocol is not mqtt:v3.1.1
func (conn *Connect) Decode(properties []byte, remain []byte) error {
	buffer := bytes.NewBuffer(remain)
	protocol := buffer.Next(4)
	if !strings.EqualFold(string(protocol), ProtocolName) {
		return UnSupportedLevelErr
	}

	readByte, _ := buffer.ReadByte()
	version := MQTTVersion(readByte)
	_, ok := supportMQTTLevel[version]
	if !ok {
		return UnSupportedLevelErr
	}

	flag, _ := buffer.ReadByte()
	bits := calc.Bytes2Bits(flag)
	connectFlag := parseConnectFlag(bits)

	keepAliveBytes := buffer.Next(2)
	keepAlive := binary.BigEndian.Uint16(keepAliveBytes)
	conn.KeepAlive = int(keepAlive)

	clientLenByte := buffer.Next(2)
	clientLen := binary.BigEndian.Uint16(clientLenByte)
	client := buffer.Next(int(clientLen))
	conn.ClientID = string(client)

	// will
	if connectFlag.willFlag {
		topicLenBytes := buffer.Next(2)
		topicLen := binary.BigEndian.Uint16(topicLenBytes)
		topic := buffer.Next(int(topicLen))

		payloadLenBytes := buffer.Next(2)
		payloadLen := binary.BigEndian.Uint16(payloadLenBytes)
		msg := buffer.Next(int(payloadLen))

		w := Will{
			Topic:   string(topic),
			Payload: msg,
			Qos:     connectFlag.willQos,
			Retain:  connectFlag.willRetainFlag,
		}

		conn.WillFlag = true
		conn.Will = w
	}

	if connectFlag.userNameFlag {
		userNameLenBytes := buffer.Next(2)
		userNameLen := binary.BigEndian.Uint16(userNameLenBytes)
		userName := buffer.Next(int(userNameLen))
		conn.User = string(userName)
	}

	if connectFlag.PwdFlag {
		pwdLenBytes := buffer.Next(2)
		pwdLen := binary.BigEndian.Uint16(pwdLenBytes)
		pwd := buffer.Next(int(pwdLen))
		conn.Pwd = string(pwd)
	}

	conn.Reserved = connectFlag.reserved
	conn.CleanSession = connectFlag.cleanSession

	return nil
}

type ConnACK struct {
	Success int
}

func NewConnACK(code ConnRespCode) []byte {
	return []byte{
		byte(PConnACK),
		2,
		0,
		byte(code),
	}
}

type Publish struct {
	Topic    string
	Dup      bool
	Qos      QOS
	Retain   bool
	PacketID uint16
	Payload  []byte
}

func NewPublish(dup byte, qos QOS, retain byte, topicName string, payload []byte) ([]byte, error) {
	fixedHeader := 48 + dup*8 + byte(qos)*2 + retain
	packetID := rand.Intn(math.MaxUint16)
	packetBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(packetBytes, uint16(packetID))

	buffer := bytes.NewBuffer(nil)
	buffer.WriteByte(fixedHeader)
	body := bytes.NewBuffer(nil)
	body.WriteByte(0)
	body.WriteByte(byte(len(topicName)))
	body.WriteString(topicName)
	body.Write(packetBytes)
	body.Write(payload)

	remainingBytes, err := calcRemainingBytes(body.Len())
	if err != nil {
		return nil, err
	}

	buffer.Write(remainingBytes)
	buffer.Write(body.Bytes())

	return buffer.Bytes(), nil
}

// properties: DUP flag, QoS level, RETAIN
// remain: remain: packet without fixed header
func (pub *Publish) Decode(properties []uint8, remain []byte) error {
	// dup, qos, retain
	flag := parsePublishFlag(properties)
	if flag.qos >= 3 {
		return QosOutOfBoundErr
	}
	pub.Dup = flag.dupFlag
	pub.Qos = flag.qos
	pub.Retain = flag.retain

	// topic
	buffer := bytes.NewBuffer(remain)
	topicLenBytes := buffer.Next(2)
	topicLen := binary.BigEndian.Uint16(topicLenBytes)
	topic := buffer.Next(int(topicLen))
	pub.Topic = string(topic)

	// unique identifier, exist in Qos level 1&2 only
	if flag.qos != 0 {
		packetIDBytes := buffer.Next(2)
		if packetIDBytes == nil {
			return IncompletePacketErr
		}
		packetID := binary.BigEndian.Uint16(packetIDBytes)
		pub.PacketID = packetID
	}

	// todo publish
	payload := buffer.Bytes()
	pub.Payload = payload

	return nil
}

// a set of publishACK/REC/REL/COMP & unsubscribeACK
type CommonACK struct {
	MSBPacketID byte
	LSBPacketID byte
}

func NewCommonACK(ctrlPacket MQTTControlPacket, packetID uint16) []byte {

	packet := make([]byte, 2)
	binary.BigEndian.PutUint16(packet, packetID)
	ack := []byte{
		byte(ctrlPacket),
		2,
		packet[0],
		packet[1],
	}
	return ack
}

type Subscribe struct {
	Topic    []string
	Qos      []QOS
	PacketID uint16
}

// properties: nil
// remain: remain: packet without fixed header
// QosOutOfBoundErr
func (sub *Subscribe) Decode(properties []byte, remain []byte) error {
	buffer := bytes.NewBuffer(remain)
	packetBytes := buffer.Next(2)
	sub.PacketID = binary.BigEndian.Uint16(packetBytes)
	sub.Topic = []string{}
	sub.Qos = []QOS{}
	// get topic and it's qos
	for {
		topicLenBytes := buffer.Next(2)
		if len(topicLenBytes) == 0 {
			break
		}
		topicLen := binary.BigEndian.Uint16(topicLenBytes)
		topic := buffer.Next(int(topicLen))
		sub.Topic = append(sub.Topic, string(topic))

		qos, _ := buffer.ReadByte()
		// todo upper 6 bits to of qos reserved for future version use
		bits := calc.Bytes2Bits(qos)
		qos = bits[6]*2 + bits[7]
		if qos > 2 {
			return QosOutOfBoundErr
		}
		sub.Qos = append(sub.Qos, QOS(qos))
	}
	return nil
}

type SubscribeACK struct {
	MSBPacketID byte
	LSBPacketID byte
	Payload     []byte
}

func NewSubscribeACK(maxQos []QOS, packetID uint16) []byte {
	packet := make([]byte, 2)
	binary.BigEndian.PutUint16(packet, packetID)
	ack := []byte{
		byte(PSubACK),
		2 + byte(len(maxQos)),
		packet[0],
		packet[1],
	}

	for _, qos := range maxQos {
		ack = append(ack, byte(qos))
	}

	return ack
}

type UnSubscribe struct {
	PacketID uint16
	Topic    []string
}

// properties: nil
// remain: remain: packet without fixed header
// InvalidErr without payload
func (us *UnSubscribe) Decode(properties []uint8, remain []byte) error {
	if len(remain) == 2 {
		return InvalidErr
	}

	buffer := bytes.NewBuffer(remain)
	packetBytes := buffer.Next(2)
	us.PacketID = binary.BigEndian.Uint16(packetBytes)
	us.Topic = []string{}

	for {
		topicLenBytes := buffer.Next(2)
		if len(topicLenBytes) == 0 {
			break
		}
		topicLen := binary.BigEndian.Uint16(topicLenBytes)
		topic := buffer.Next(int(topicLen))
		us.Topic = append(us.Topic, string(topic))
	}

	return nil
}

type KeepAlive struct {
}

func NewPingRESP() []byte {
	return []byte{
		byte(PPingRESP),
		0,
	}
}

func parseConnectFlag(bits []uint8) connectFlags {
	var flag connectFlags
	flag.userNameFlag = bits[0] == 1
	flag.PwdFlag = bits[1] == 1
	flag.willRetainFlag = bits[2] == 1
	flag.willQos = QOS(bits[3]*2 + bits[4])
	flag.willFlag = bits[5] == 1
	flag.cleanSession = int(bits[6]) == 1
	flag.reserved = bits[7] == 1
	return flag
}

func parsePublishFlag(bits []uint8) publishFlags {
	var flag publishFlags

	flag.dupFlag = bits[0] == 1
	flag.qos = QOS(bits[1]*2 + bits[2])
	flag.retain = bits[3] == 1

	return flag
}

func CalcControlPacket(binaryArr []uint8) MQTTControlPacket {
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

func calcRemainingBytes(input int) ([]byte, error) {
	var (
		first = 0
		next  = input
		ret   [4]byte
		k     = 0
	)

	if input > MaximumRemainLen {
		return nil, errors.New("Malformed Remaining Length")
	}

	for next > 0 {
		first = next % 128
		next = next / 128
		ret[k] = byte(first)
		if next > 0 {
			ret[k] |= 0x80
		}
		k++
	}

	return ret[:k], nil
}
