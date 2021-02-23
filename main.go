package main

import (
	"excel_parser/session"
	"net"
	"net/http"
	_ "net/http/pprof"
)

func main() {
	go func() {
		http.ListenAndServe("0.0.0.0:8080", nil)
	}()

	//pair, e := tls.LoadX509KeyPair("C:\\Users\\Administrator\\Desktop\\ca\\ca.crt", "C:\\Users\\Administrator\\Desktop\\ca\\ca.key")
	//if e != nil {
	//	panic(e)
	//}
	var (
		tlsMode = false
		listen  net.Listener
		err     error
	)

	var addr = "0.0.0.0:45323"
	if tlsMode {
		//listen, err = tls.Listen("tcp", addr, &tls.Config{Certificates: []tls.Certificate{pair}})
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
		go session.New(conn)
	}

}
