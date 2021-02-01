package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
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
		128, 127,
	}

	for _, i := range l {

		value += (i & 127) * multiplier

		if (i & 128) == 0 {
			break
		}

		multiplier *= 128
		if multiplier > 128*128*128*128 {
			return
		}
	}

	fmt.Println(value)

}
