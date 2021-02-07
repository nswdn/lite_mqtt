package trie

import (
	"fmt"
	"testing"
)

func TestTrie(t *testing.T) {
	tr.insert("asd/w", "1", nil)
	tr.insert("asd/d", "2", nil)
	tr.insert("asd/s", "3", nil)
	tr.insert("qwe/v", "4", nil)
	tr.insert("asd/s/s", "5", nil)
	tr.insert("asd/s/s", "5", nil)
	match := tr.match("qwe/v")
	match2 := tr.match("asd/s")
	fmt.Println(tr.root, match, match2)
}
