package session

import (
	"fmt"
	"testing"
)

func TestCalc(t *testing.T) {
	b := []byte{
		128, 128, 1, 1,
	}
	remaining, i, _ := calcRemaining(b)
	fmt.Println(remaining, i)
}
