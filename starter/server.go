package starter

import (
	"crypto/tls"
	"excel_parser/config"
	"excel_parser/session"
	"log"
	"net"
)

func Start() error {
	var (
		address = config.Addr()
		listen  net.Listener
		err     error
		tlsMode = config.TlsMode()
	)
	if tlsMode {
		pair, e := tls.LoadX509KeyPair(config.TlsFile())
		if e != nil {
			return e
		}
		listen, err = tls.Listen("tcp", address, &tls.Config{Certificates: []tls.Certificate{pair}})
	} else {
		listen, err = net.Listen("tcp", address)
	}

	if err != nil {
		return err
	}

	log.Println("mqtt server start listen:", address)
	log.Println("mqtt server start up with tls:", tlsMode)

	for {
		conn, err := listen.Accept()
		if err != nil {
			return err
		}
		go session.New(conn)
	}

}
