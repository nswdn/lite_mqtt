package main

import (
	"encoding/binary"
	"fmt"
	"testing"
)

func TestConv(t *testing.T) {

	bytes := make([]byte, 4)

	binary.BigEndian.PutUint32(bytes, 3)
	fmt.Println(bytes)
}
