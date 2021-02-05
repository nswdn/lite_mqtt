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
	decodeEndChan chan byte
	stopChan      chan byte

	publishChan     chan content
	publishAckChan  chan content
	publishRECChan  chan content
	publishRELChan  chan content
	subscribeChan   chan content
	unsubscribeChan chan content
	pingREQChan     chan content
	disconnectChan  chan content
}

type newMessage struct {
	headerLen int
	content   []byte
}

type content struct {
	properties []uint8
	body       []byte
}

func NewDecoder() *decoder {
	d := &decoder{
		store:         bytes.NewBuffer(nil),
		decodeEndChan: make(chan byte, 1),
		newMsgChan:    make(chan newMessage, 1),
		stopChan:      make(chan byte, 1),

		publishChan:     make(chan content, 1),
		publishAckChan:  make(chan content, 1),
		publishRECChan:  make(chan content, 1),
		publishRELChan:  make(chan content, 1),
		subscribeChan:   make(chan content, 1),
		unsubscribeChan: make(chan content, 1),
		pingREQChan:     make(chan content, 1),
		disconnectChan:  make(chan content, 1),
	}

	go d.processPacket()
	return d
}

func (coder *decoder) processPacket() {
loop:
	for {
		select {
		case packet := <-coder.newMsgChan:

			bits := calc.Bytes2Bits(packet.content[0])
			ctrlPacket := proto.CalcControlPacket(bits[:4])
			body := packet.content[packet.headerLen:]
			coder.selectChannel(ctrlPacket, content{bits, body})
			coder.decoding = false
			coder.decodeEndChan <- 1
		case <-coder.stopChan:
			break loop
		}
	}
}

func (coder *decoder) selectChannel(packet proto.MQTTControlPacket, content content) {
	switch packet {
	case proto.PPublish:
		coder.publishChan <- content
	case proto.PPubACK:
		coder.publishAckChan <- content
	case proto.PPubREC:
		coder.publishRECChan <- content
	case proto.PPubREL:
		coder.publishRELChan <- content
	case proto.PSubscribe:
		coder.subscribeChan <- content
	case proto.PUnsubscribe:
		coder.unsubscribeChan <- content
	case proto.PPingREQ:
		coder.pingREQChan <- content
	case proto.PDisconnect:
		coder.disconnectChan <- content

	}
}

func (coder *decoder) decode(in []byte) {
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
				return
			}
			fullPacket := coder.store.Next(coder.totalLen)
			coder.newMsgChan <- newMessage{coder.headerLen, fullPacket}
			<-coder.decodeEndChan
		}
		break
	}
}

func (coder *decoder) Close() {
	coder.stopChan <- 1
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
