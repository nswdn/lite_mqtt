package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"excel_parser/proto"
	"fmt"
	"io/ioutil"
	"testing"
)

func TestConv(t *testing.T) {

	bytes := make([]byte, 4)

	binary.BigEndian.PutUint32(bytes, 3)
	fmt.Println(bytes)
}

func TestTlS(t *testing.T) {

	pool := x509.NewCertPool()
	file, _ := ioutil.ReadFile("C:\\Users\\Administrator\\Desktop\\ca\\AlanDevTest.crt")
	pool.AppendCertsFromPEM(file)

	dial, e := tls.Dial("tcp", "127.0.0.1:45323", &tls.Config{ClientCAs: pool, InsecureSkipVerify: true})
	if e != nil {
		fmt.Println(e)
		return
	}
	n, err := dial.Write([]byte{1, 2, 3, 3})
	fmt.Println(n, err)

	dial.Close()
}

func TestWriteToClosedChan(t *testing.T) {
	ints := make(chan int, 1)
	i := cap(ints)
	fmt.Println(i)

	i, ok := <-ints
	fmt.Println(i, ok)
}

//multiplier = 1
//value = 0
//do
//encodedByte = 'next byte from stream'
//value += (encodedByte AND 127) * multiplier
//multiplier *= 128
//if (multiplier > 128*128*128)
//throw Error(Malformed Remaining Length)
//while ((encodedByte AND 128) != 0)
func TestCalcRemainLen(t *testing.T) {
	var value = 0
	var multiplier = 1

	l := []int{
		254, 255, 255, 127,
	}

	var lo int
	for _, i := range l {
		value += (i & 127) * multiplier

		if (i & 128) == 0 {
			break
		}

		multiplier *= 128
		if multiplier > 128*128*128 {
			panic("Malformed Remaining Length")
		}
	}

	fmt.Println(value, lo)
}

func TestCalcRemainByte(t *testing.T) {
	var input = 268435455

	var (
		first = 0
		next  = input
		ret   [4]byte
		k     = 0
	)

	if input > proto.MaximumRemainLen {
		panic("Malformed Remaining Length")
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

	fmt.Println(ret[:k])
}

func TestAnd(t *testing.T) {
	fmt.Println(128 & 128)
}

func TestBuffer(t *testing.T) {

	buffer := bytes.NewBuffer(nil)
	buffer.WriteByte(1)
	fmt.Println(buffer.Next(1))
	fmt.Println(buffer.Next(1))

	buffer.Truncate(0)

	newBuffer := bytes.NewBuffer(nil)
	fmt.Println(newBuffer)
}
