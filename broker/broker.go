package broker

import (
	"log"
)

type retainMessage []byte

var emptyRetain retainMessage = nil

type Topic struct {
	Name        string
	Subscribers map[string]chan<- []byte // key: clientID, value: channel to notify new subscribe message
	retain      retainMessage
}

type publishMessage struct {
	topic   string
	payload []byte
	retain  bool
}

type subscribeMessage struct {
	topic    string
	clientID string
	receiver chan<- []byte
}

type unsubscribeMessage struct {
	topic    string
	clientID string
}

type Broker struct {
	topicTrie   *trie
	publishChan chan publishMessage
	subChan     chan subscribeMessage
	unsubChan   chan unsubscribeMessage
}

var b *Broker

func init() {
	b = &Broker{
		topicTrie:   tr,
		publishChan: make(chan publishMessage, 100),
		subChan:     make(chan subscribeMessage, 100),
		unsubChan:   make(chan unsubscribeMessage, 100),
	}

	go b.dispatcherHandle()
}

func (broker *Broker) dispatcherHandle() {
	var (
		publishMsg  publishMessage
		subMsg      subscribeMessage
		unsubMsg    unsubscribeMessage
		match       *trieNode
		publishChan chan<- []byte
	)

	for {
		select {
		case publishMsg = <-broker.publishChan:
			if match = broker.topicTrie.match(publishMsg.topic); match != nil {
				if publishMsg.retain {
					if len(publishMsg.payload) > 0 {
						match.topic.retain = publishMsg.payload
					} else {
						match.topic.retain = emptyRetain
					}
				}
				for _, publishChan = range match.topic.Subscribers {
					publishChan <- publishMsg.payload
				}
			}
		case subMsg = <-broker.subChan:
			result := b.topicTrie.insert(subMsg.topic, subMsg.clientID, subMsg.receiver)
			retainMsg := result.topic.retain
			if result.topic.retain != nil {
				subMsg.receiver <- retainMsg
			}
		case unsubMsg = <-broker.unsubChan:
			if match := b.topicTrie.match(unsubMsg.topic); match != nil {
				match.topic.unsubscribe(unsubMsg.clientID)
			}
		}
	}
}

func Subscribe(topicName, clientID string, receiver chan<- []byte) {
	b.subChan <- subscribeMessage{topic: topicName, clientID: clientID, receiver: receiver}
}

func Unsubscribe(topicName string, clientID string) {
	b.unsubChan <- unsubscribeMessage{clientID: clientID, topic: topicName}
}

func Publish(topic string, retain bool, payload []byte) {
	b.publishChan <- publishMessage{topic, payload, retain}
}

func (to *Topic) unsubscribe(clientID string) {
	delete(to.Subscribers, clientID)
}

func (to *Topic) put(clientID string, receiver chan<- []byte) {
	to.Subscribers[clientID] = receiver
}

func newTopic(name string, id string, receiver chan<- []byte) *Topic {
	t := &Topic{
		Name:        name,
		Subscribers: make(map[string]chan<- []byte),
	}

	if id != empty {
		t.Subscribers[id] = receiver
	}

	log.Println("new topic name:", name)
	return t
}
