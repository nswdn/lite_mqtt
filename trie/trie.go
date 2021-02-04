package trie

import "strings"

var tr *trie

const splitter = "/"

type trieNode struct {
	size int
	next map[string]*trieNode
}

type trie struct {
	root *trieNode
}

func init() {
	tr = &trie{
		root: &trieNode{
			size: 0,
			next: make(map[string]*trieNode),
		},
	}
}

// test/w
func (tr *trie) insert(v string) {
	split := strings.Split(v, splitter)
	var node = tr.root

	for _, str := range split {
		_, ok := node.next[str]
		if !ok {
			node.next[str] = newNode()
		}
		node = node.next[str]
	}
}

func newNode() *trieNode {
	return &trieNode{
		next: make(map[string]*trieNode),
	}
}
