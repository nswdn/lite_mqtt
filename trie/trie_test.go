package trie

import (
	"fmt"
	"testing"
)

func TestTrie(t *testing.T) {
	tr.insert("asd/w")
	tr.insert("asd/d")
	tr.insert("asd/s")
	tr.insert("qwe/v")
	tr.insert("asd/s/s")
	tr.delete("asd/s/s")
	match := tr.match("qwe/v")
	match2 := tr.match("asd/s")
	fmt.Println(tr.root, match, match2)
}
