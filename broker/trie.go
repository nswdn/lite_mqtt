package broker

import (
	"strings"
)

var tr = &trie{
	root: &trieNode{
		size: 0,
		next: make(map[string]*trieNode),
	},
}

const (
	splitter = "/"
	empty    = ""
)

type trieNode struct {
	topic *Topic
	size  int
	next  map[string]*trieNode
}

type trie struct {
	root *trieNode
}

// test/w
func (tr *trie) insert(subMsg subscriber) *trieNode {
	split := strings.Split(subMsg.topic, splitter)

	match := tr.match(subMsg.topic)
	if match != nil {
		match.topic.Subscribers[subMsg.clientID] = subMsg
		return match
	}

	var node = tr.root

	for _, str := range split {
		_, ok := node.next[str]
		if !ok {
			node.next[str] = newNode()
		}
		node.size++
		node = node.next[str]
	}

	if node.topic == nil {
		node.topic = newTopic(subMsg.topic, subMsg.clientID, subMsg)
	}
	node.topic.put(subMsg.clientID, subMsg)
	return node
}

// test/v
func (tr *trie) match(topic string) *trieNode {
	split := strings.Split(topic, splitter)
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

// Deprecated
func (tr *trie) delete(topic, clientID string) {
	split := strings.Split(topic, splitter)
	var node = tr.root
	for i, str := range split {
		trNode, ok := node.next[str]
		if !ok {
			return
		}
		node.size--
		if i == len(split) {
			trNode.topic.unsubscribe(clientID)
		}
		node = trNode
	}
}

func newNode() *trieNode {
	return &trieNode{
		next: make(map[string]*trieNode),
	}
}
