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

func deepCopy(src []byte) []byte {
	dst := make([]byte, len(src))
	copy(dst, src)
	return dst
}

func (coder *decoder) decode(conn net.Conn, in []byte) (content, error) {
	var (
		totalLen   int
		headerLen  int
		contentLen int
	)

	fixedHeader := in[:5]
	remaining, remainingBytesLen, _ := calcRemaining(fixedHeader[1:])
	totalLen = remaining + remainingBytesLen + 1
	headerLen = remainingBytesLen + 1
	received := len(in) - headerLen

	contentLen = totalLen - headerLen
	if received < contentLen {
		leftBytes := make([]byte, contentLen-received)
		left, err := io.ReadFull(conn, leftBytes)
		if err != nil {
			return content{}, err
		}
		if left+received != remainingBytesLen {
			return content{}, err
		}
		in = append(in, leftBytes[:left]...)
	}

	return unPacket(headerLen, in), nil
}

func calcRemaining(remaining []byte) (int, int, error) {
	var (
		value        = 0
		multiplier   = 1
		remainingLen = 0
		err          error
		last         byte
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

	return value, remainingLen, err
}
