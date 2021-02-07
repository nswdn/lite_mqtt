package trie

import (
	"sync"
)

type Topic struct {
	Name        string
	Bus         chan string              // listen unsubscribe write clientID in
	Subscribers map[string]chan<- []byte // key: clientID, value: channel to notify new subscribe message
	mutex       sync.Mutex
}

type publishMessage struct {
	topic   string
	payload []byte
}

type Broker struct {
	topicTrie   *trie
	rwMutex     sync.RWMutex
	publishChan chan *publishMessage
}

var b = &Broker{
	topicTrie: &trie{
		root: &trieNode{
			size: 0,
			next: make(map[string]*trieNode),
		},
	},
	publishChan: make(chan *publishMessage, 1),
}

func init() {
	go b.listenPublish()
}

func (broker *Broker) GetTopic(topicName, clientID string, receiver chan<- []byte) chan string {
	broker.rwMutex.Lock()
	defer broker.rwMutex.Unlock()

	result := broker.topicTrie.insert(topicName, clientID, receiver)
	return result.topic.Bus
}

func (broker *Broker) listenPublish() {
	var (
		message     *publishMessage
		match       *trieNode
		receiveChan chan<- []byte
	)

	for {
		select {
		case message = <-broker.publishChan:
			broker.rwMutex.RLock()
			match = broker.topicTrie.match(message.topic)
			for _, receiveChan = range match.topic.Subscribers {
				receiveChan <- message.payload
			}
			broker.rwMutex.RUnlock()
		}
	}

}

func (to *Topic) listenUnsub() {
	var clientID string
	for {
		select {
		case clientID = <-to.Bus:
			to.mutex.Lock()
			delete(to.Subscribers, clientID)
			to.mutex.Unlock()
		}
	}
}

func newTopic(name string, id string, receiver chan<- []byte) *Topic {
	t := &Topic{
		Name:        name,
		Bus:         make(chan string, 100),
		Subscribers: make(map[string]chan<- []byte),
	}
	if id != empty {
		t.Subscribers[id] = receiver
	}
	go t.listenUnsub()
	return t
}
