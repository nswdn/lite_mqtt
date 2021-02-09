package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"excel_parser/proto"
	"fmt"
	"io/ioutil"
	"math/rand"
	"testing"
	"time"
)

func TestConv(t *testing.T) {
	i := make([]byte, 1024)
	rand.Read(i)
	buffer := bytes.NewBuffer(i)
	buffer.Reset()
	fmt.Println(buffer)
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
		128, 255, 1, 1,
	}

	var lo int
	var last int
	for _, i := range l {
		v := (i & 127) * multiplier
		if last == 128 {
			lo++
		}
		if v > 0 {
			lo++
		}
		last = i
		value += v
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

	i := make([]byte, 102400)
	buffer := bytes.NewBuffer(i)
	buffer.WriteByte(1)
	fmt.Println(buffer.Next(1))
	fmt.Println(buffer.Next(1))

	buffer.Truncate(0)

	newBuffer := bytes.NewBuffer(nil)
	fmt.Println(newBuffer)
}

func TestUint16(t *testing.T) {
	i := []byte{
		2, 0,
	}

	u := binary.BigEndian.Uint16(i)
	fmt.Println(u)

	i2 := make([]byte, 2)
	binary.BigEndian.PutUint16(i2, u)
	fmt.Println(i2)
}

func TestCalc(t *testing.T) {
	i := []byte{
		2, 2, 2, 0,
	}
	buffer := bytes.NewBuffer(i)
	next := buffer.Next(4)
	fmt.Println(next)

}

func TestClose(t *testing.T) {
	c := make(chan struct{}, 1)
	go func() {
		time.Sleep(time.Second)
		close(c)
	}()

	select {
	case s := <-c:
		fmt.Println(s)
		fmt.Println("1")
	}

}

func TestNoPayload(t *testing.T) {
	buffer := bytes.NewBuffer(nil)
	buffer.Write([]byte{1})
	buffer.Next(1)
	next := buffer.Bytes()
	fmt.Println(next)
}

func TestBlockChan(t *testing.T) {
	c := make(chan struct{})
	go func() {
		select {
		case <-c:
			fmt.Println(2)
		}
	}()
	c <- struct{}{}
	fmt.Println(1)
}
