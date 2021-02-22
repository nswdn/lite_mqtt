package trie

import (
	"fmt"
	"testing"
)

func TestTrie(t *testing.T) {
	client1 := make(chan []byte, 1)
	client2 := make(chan []byte, 1)

	for i := 0; i < 2; i++ {
		select {
		case r := <-client1:
			fmt.Println(r)
		case r := <-client2:
			fmt.Println(r)
		}
	}
}
