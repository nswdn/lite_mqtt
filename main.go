package main

import (
	"excel_parser/session"
	"net"
)

func main() {
	listen, err := net.Listen("tcp", "127.0.0.1:45323")
	if err != nil {
		return
	}

	for {
		conn, err := listen.Accept()
		if err != nil {
			return
		}
		s := session.New(conn)
		go s.Handle()
	}

}
