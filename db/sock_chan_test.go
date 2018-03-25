package db

import (
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
)

func TestSocketChannels(t *testing.T) {
	var l net.Listener
	l, err := net.Listen("tcp", ":8082")
	assert.Nil(t, err)
	sc := SocketChannels(l)
	connections := 0
	go func() {
		for i := 0; i < 3; i++ {
			net.Dial("tcp", ":8082")
		}
		l.Close()
	}()
	for c := range sc {
		assert.Nil(t, c.Close())
		connections++
	}
	assert.Equal(t, 3, connections)
}
