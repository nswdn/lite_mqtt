package main

import (
	"crypto/tls"
	"excel_parser/session"
	"net"
)

var sessions []*session.Session

func main() {
	pair, e := tls.LoadX509KeyPair("C:\\Users\\Administrator\\Desktop\\ca\\ca.crt", "C:\\Users\\Administrator\\Desktop\\ca\\ca.key")
	if e != nil {
		panic(e)
	}
	var (
		tlsMode = true
		listen  net.Listener
		err     error
	)

	var addr = "172.168.5.102:45323"
	if tlsMode {
		listen, err = tls.Listen("tcp", addr, &tls.Config{Certificates: []tls.Certificate{pair}})
	} else {
		listen, err = net.Listen("tcp", addr)
	}

	if err != nil {
		panic(err)
	}

	for {
		conn, err := listen.Accept()
		if err != nil {
			return
		}
		s := session.New(conn)
		sessions = append(sessions, s)
		go s.Handle()
	}

}
