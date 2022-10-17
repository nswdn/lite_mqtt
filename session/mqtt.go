package session

import "excel_parser/proto"

type MQTT interface {
	GetHeader() Header
	GetBody() Body
}

type MQTTMessage struct {
	MQTT

	Header Header
	Body   Body
}

type Header interface {
	GetProperties()
	GetControlPacket() proto.MQTTControlPacket
}

type Body interface {
}

func (m *MQTTMessage) GetHeader() Header {
	return m.Header
}

func (m *MQTTMessage) GetBody() Body {
	return m.Body
}
