package trie

import (
	"fmt"
	"testing"
)

func TestTrie(t *testing.T) {
	client1 := make(chan []byte, 1)
	client2 := make(chan []byte, 1)

	go func() {
		b.GetTopic("asd/a", "client1", client1)
		b.GetTopic("asd/b", "client2", client2)
		b.publishChan <- &publishMessage{
			topic:   "asd/a",
			payload: []byte{123},
		}
		b.publishChan <- &publishMessage{
			topic:   "asd/c",
			payload: []byte{123},
		}
	}()

	for i := 0; i < 2; i++ {
		select {
		case r := <-client1:
			fmt.Println(r)
		case r := <-client2:
			fmt.Println(r)
		}
	}
}
