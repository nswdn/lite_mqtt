package session

import (
	"bytes"
	"errors"
	"excel_parser/calc"
	"excel_parser/proto"
)

type decoder struct {
	store     *bytes.Buffer
	decoding  bool
	totalLen  int
	headerLen int

	newMsgChan    chan newMessage
	decodeEndChan chan struct{}
	stopChan      chan struct{}

	processedChan chan content
}

type newMessage struct {
	headerLen int
	content   []byte
}

type content struct {
	ctrlPacket proto.MQTTControlPacket
	properties []uint8
	body       []byte
}

func NewDecoder() *decoder {
	d := &decoder{
		store:         bytes.NewBuffer(nil),
		newMsgChan:    make(chan newMessage, 1),
		decodeEndChan: make(chan struct{}),
		stopChan:      make(chan struct{}),
		processedChan: make(chan content, 10),
	}

	go d.processPacket()
	return d
}

func (coder *decoder) processPacket() {
	var (
		bits       []byte
		ctrlPacket proto.MQTTControlPacket
		body       []byte
		dst        []byte
	)
loop:
	for {
		select {
		case packet := <-coder.newMsgChan:
			bits = calc.Bytes2Bits(packet.content[0])
			ctrlPacket = proto.CalcControlPacket(bits[:4])
			body = packet.content[packet.headerLen:]
			dst = deepCopy(body)
			coder.processedChan <- content{ctrlPacket, bits[4:], dst}
			coder.decoding = false
			coder.decodeEndChan <- struct{}{}
		case <-coder.stopChan:
			break loop
		}
	}
}

func unPacket(header int, packet []byte) content {
	var (
		bits       []byte
		ctrlPacket proto.MQTTControlPacket
		body       []byte
		dst        []byte
	)
	bits = calc.Bytes2Bits(packet[0])
	ctrlPacket = proto.CalcControlPacket(bits[:4])
	body = packet[header:]
	dst = deepCopy(body)
	return content{ctrlPacket, bits[4:], dst}
}

func deepCopy(src []byte) []byte {
	dst := make([]byte, len(src))
	copy(dst, src)
	return dst
}

func (coder *decoder) decode(in []byte) (content, error) {
	coder.store.Write(in)
	for {
		if coder.store.Len() >= 2 {
			if !coder.decoding {
				fixedHeader := coder.store.Next(5)
				remaining, remainingBytesLen, _ := calcRemaining(fixedHeader[1:])
				coder.totalLen = remaining + remainingBytesLen + 1
				coder.headerLen = remainingBytesLen + 1
				newStore := bytes.NewBuffer(fixedHeader)
				newStore.Write(coder.store.Bytes())
				coder.store = newStore
				if coder.totalLen == 1 {
					coder.totalLen += 1
				}
			}

			if coder.store.Len() < coder.totalLen {
				return content{}, errors.New("invalid length")
			}

			fullPacket := coder.store.Next(coder.totalLen)
			return unPacket(coder.headerLen, fullPacket), nil

		}
		break
	}

	return content{}, errors.New("invalid length")
}

func (coder *decoder) Close() {
	coder.stopChan <- struct{}{}
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
