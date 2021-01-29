package proto

import (
	"bytes"
	"errors"
	"excel_parser/calc"
	"log"
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
	PConnect MQTTControlPacket = 1
	PConnACK MQTTControlPacket = 32 // 2

	PPublish MQTTControlPacket = 3
	PPubACK  MQTTControlPacket = 64  // 4
	PPubREC  MQTTControlPacket = 80  // 5
	PPubREL  MQTTControlPacket = 98  // 6
	PPubCOMP MQTTControlPacket = 112 //7

	PSubscribe      MQTTControlPacket = 8
	PSubACK         MQTTControlPacket = 144 //9
	PUnsubscribe    MQTTControlPacket = 162 // 10
	PUnsubscribeACK MQTTControlPacket = 176 // 11

	PPingREQ    MQTTControlPacket = 12  // 12
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
	protocol := make([]byte, 4)
	buffer := bytes.NewBuffer(remain)
	_, _ = buffer.Read(protocol)
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

	_, _ = buffer.ReadByte()
	lsbKeepAlive, _ := buffer.ReadByte()
	conn.KeepAlive = int(lsbKeepAlive)

	_, _ = buffer.ReadByte()
	lsbClientLen, _ := buffer.ReadByte()

	client := make([]byte, lsbClientLen)
	_, _ = buffer.Read(client)
	conn.ClientID = string(client)

	if connectFlag.willFlag {
		_, _ = buffer.ReadByte()
		lsbTopicLen, _ := buffer.ReadByte()
		topic := make([]byte, lsbTopicLen)
		_, _ = buffer.Read(topic)

		_, _ = buffer.ReadByte()
		lsbMsgLen, _ := buffer.ReadByte()
		msg := make([]byte, lsbMsgLen)
		_, _ = buffer.Read(msg)

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
		_, _ = buffer.ReadByte()
		lsbUserLen, _ := buffer.ReadByte()
		userName := make([]byte, lsbUserLen)
		_, _ = buffer.Read(userName)
		conn.User = string(userName)
	}

	if connectFlag.PwdFlag {
		_, _ = buffer.ReadByte()
		lsbPwdLen, _ := buffer.ReadByte()
		pwd := make([]byte, lsbPwdLen)
		_, _ = buffer.Read(pwd)
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
	Topic       string
	Dup         bool
	Qos         QOS
	Retain      bool
	MSBPacketID byte
	LSBPacketID byte
	Payload     []byte
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
	_, _ = buffer.ReadByte()
	lsbTopicLen, _ := buffer.ReadByte()
	topic := make([]byte, lsbTopicLen)
	_, _ = buffer.Read(topic)
	pub.Topic = string(topic)

	// unique identifier, exist in Qos level 1&2 only
	if flag.qos != 0 {
		msbPacketId, _ := buffer.ReadByte()
		lsbPacketId, _ := buffer.ReadByte()
		log.Println("packet id: ", lsbPacketId)
		pub.MSBPacketID = msbPacketId
		pub.LSBPacketID = lsbPacketId
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

func NewCommonACK(ctrlPacket MQTTControlPacket, MSBPacketID byte, LSBPacketID byte) []byte {
	ack := []byte{
		byte(ctrlPacket),
		2,
		MSBPacketID,
		LSBPacketID,
	}
	return ack
}

type Subscribe struct {
	Topic       []string
	Qos         []QOS
	MSBPacketID byte
	LSBPacketID byte
}

// properties: nil
// remain: remain: packet without fixed header
// QosOutOfBoundErr
func (sub *Subscribe) Decode(properties []byte, remain []byte) error {
	buffer := bytes.NewBuffer(remain)
	msbPacketId, _ := buffer.ReadByte()
	lsbPacketId, _ := buffer.ReadByte()
	sub.MSBPacketID = msbPacketId
	sub.LSBPacketID = lsbPacketId
	sub.Topic = []string{}
	sub.Qos = []QOS{}
	// get topic and it's qos
	for {
		if _, err := buffer.ReadByte(); err != nil {
			break
		}
		lsbTopicLen, _ := buffer.ReadByte()
		topic := make([]byte, lsbTopicLen)
		_, _ = buffer.Read(topic)
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

func NewSubscribeACK(maxQos []QOS, MSBPacketID byte, LSBPacketID byte) []byte {
	ack := []byte{
		byte(PSubACK),
		2 + byte(len(maxQos)),
		MSBPacketID,
		LSBPacketID,
	}

	for _, qos := range maxQos {
		ack = append(ack, byte(qos))
	}

	return ack
}

type UnSubscribe struct {
	MSBPacketID byte
	LSBPacketID byte
	Topic       []string
}

// properties: nil
// remain: remain: packet without fixed header
// InvalidErr without payload
func (us *UnSubscribe) Decode(properties []uint8, remain []byte) error {
	if len(remain) == 2 {
		return InvalidErr
	}

	buffer := bytes.NewBuffer(remain)
	MSBPacketID, _ := buffer.ReadByte()
	LSBPacketID, _ := buffer.ReadByte()
	us.MSBPacketID = MSBPacketID
	us.LSBPacketID = LSBPacketID
	us.Topic = []string{}

	for {
		if _, err := buffer.ReadByte(); err != nil {
			break
		}
		lsb, _ := buffer.ReadByte()
		topic := make([]byte, lsb)
		_, _ = buffer.Read(topic)
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
