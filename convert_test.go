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
