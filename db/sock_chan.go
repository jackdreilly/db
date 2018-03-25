package db

import (
	"net"
)

func SocketChannels(listener net.Listener) <-chan net.Conn {
	connChan := make(chan net.Conn)
	go func() {
		defer close(connChan)
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			connChan <- conn
		}
	}()
	return connChan
}
