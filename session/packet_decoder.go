package session

import (
	"bytes"
	"errors"
)

type decoder struct {
	decoding  bool
	totalLen  int
	readLen   int
	end       bool
	headerLen int
	readBytes *bytes.Buffer
}

func NewDecoder() *decoder {
	return &decoder{
		readBytes: bytes.NewBuffer(nil),
	}
}

func (coder *decoder) decode(in []byte) error {
	if !coder.decoding {
		// process fixed header
		buffer := bytes.NewBuffer(in)
		_, _ = buffer.ReadByte()

		var remainBytesLen int
		var remainBytes = make([]byte, 4)
		for i := 0; i < 4; i++ {
			r, _ := buffer.ReadByte()
			if r == 0 {
				_ = buffer.UnreadByte()
				break
			}
			remainBytesLen++
			remainBytes[i] = r
		}

		var remaining int
		var err error

		if remaining, err = calcRemaining(remainBytes); err != nil {
			return err
		}
		coder.headerLen = remainBytesLen + 1
		coder.totalLen = remaining + coder.headerLen
		coder.decoding = true
	}

	if (coder.readLen + len(in)) <= coder.totalLen {
		if (coder.readLen + len(in)) == coder.totalLen {
			coder.end = true
		}
		coder.readBytes.Write(in)
		coder.readLen += len(in)
		return nil
	}

	coder.readBytes.Write(in)
	coder.readLen += len(in)

	return nil
}

func (coder *decoder) readAll() []byte {
	return coder.readBytes.Bytes()
}

func calcRemaining(remaining []byte) (int, error) {
	var (
		value      = 0
		multiplier = 1
		error      error
	)

	for _, i := range remaining {
		value += int(i&127) * multiplier

		if (i & 128) == 0 {
			break
		}

		multiplier *= 128
		if multiplier > 128*128*128 {
			error = errors.New("Malformed Remaining Length")
		}
	}

	return value, error
}
