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
		node.size++
		node = node.next[str]
	}
}

// test/v
func (tr *trie) match(v string) *trieNode {
	split := strings.Split(v, splitter)
	var node = tr.root
	for _, str := range split {
		trNode, ok := node.next[str]
		if !ok {
			return nil
		}
		node = trNode
	}
	return node
}

func (tr *trie) delete(v string) string {
	split := strings.Split(v, splitter)
	var node = tr.root
	for i, str := range split {
		trNode, ok := node.next[str]
		if !ok {
			return ""
		}
		node.size--
		if i == len(split)-1 {
			delete(node.next, str)
		}
		node = trNode
	}
	return v
}

func newNode() *trieNode {
	return &trieNode{
		next: make(map[string]*trieNode),
	}
}
