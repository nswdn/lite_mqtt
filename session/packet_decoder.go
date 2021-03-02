package session

import (
	"errors"
	"excel_parser/calc"
	"excel_parser/proto"
	"io"
	"net"
)

type decoder struct {
}

type content struct {
	ctrlPacket proto.MQTTControlPacket
	properties []uint8
	body       []byte
}

func NewDecoder() *decoder {
	d := &decoder{}

	//go d.processPacket()
	return d
}

func unPacket(header int, packet []byte) content {
	var (
		bits       []byte
		ctrlPacket proto.MQTTControlPacket
		body       []byte
	)
	bits = calc.Bytes2Bits(packet[0])
	ctrlPacket = proto.CalcControlPacket(bits[:4])
	body = packet[header:]
	return content{ctrlPacket, bits[4:], body}
}

func (coder *decoder) decode(conn net.Conn, in []byte) (con content, err error) {
	var headerLen, remaining, remainingBytesLen, received int

	expectedFixedHeader := in[:5]

	remaining, remainingBytesLen, err = calcRemaining(expectedFixedHeader[1:])
	if err != nil {
		return
	}

	headerLen = remainingBytesLen + 1
	received = len(in) - headerLen

	if received < remaining {
		var left int
		leftBytes := make([]byte, remaining-received)
		// todo need read time out
		left, err = io.ReadFull(conn, leftBytes)
		if err != nil {
			return
		}
		if left+received != remaining {
			return
		}
		in = append(in, leftBytes[:left]...)
	}

	return unPacket(headerLen, in), nil
}

// value: packet total length without fixed header(control packet length + remaining bytes length)
func calcRemaining(remaining []byte) (value, remainingLen int, err error) {
	var (
		multiplier = 1
		last       byte
	)

	for _, i := range remaining {
		v := int(i&127) * multiplier
		value += v

		if last == 128 {
			remainingLen++
		}
		last = i
		if v > 0 {
			remainingLen++
		}

		if (i & 128) == 0 {
			break
		}

		multiplier *= 128
		if multiplier > 128*128*128 {
			err = errors.New("Malformed Remaining Length")
		}
	}
	return
}
