package session

import (
	"log"
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
	Topics      map[string]*Topic
	rwMutex     sync.RWMutex
	publishChan chan *publishMessage
}

var b = &Broker{
	Topics:      make(map[string]*Topic),
	publishChan: make(chan *publishMessage, 1),
}

func init() {
	go b.listenPublish()
}

func (broker *Broker) GetTopic(topicName, clientID string, receiver chan<- []byte) chan string {
	broker.rwMutex.Lock()
	defer broker.rwMutex.Unlock()

	topic, ok := broker.Topics[topicName]
	if !ok {
		topic = newTopic(topicName)
	}

	topic.Subscribers[clientID] = receiver
	broker.Topics[topicName] = topic

	return topic.Bus
}

func (broker *Broker) listenPublish() {
	var (
		message *publishMessage
		topic   *Topic
		ok      bool
		v       chan<- []byte
	)

	for {
		select {
		case message = <-broker.publishChan:
			broker.rwMutex.RLock()
			topic, ok = broker.Topics[message.topic]
			if !ok {
				broker.rwMutex.RUnlock()
				return
			}
			for _, v = range topic.Subscribers {
				v <- message.payload
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
			log.Printf("%s unsubscribe topic: %s", clientID, to.Name)
		}
	}
}

func newTopic(name string) *Topic {
	t := new(Topic)
	t.Name = name
	t.Subscribers = make(map[string]chan<- []byte)
	t.Bus = make(chan string, 100)
	go t.listenUnsub()
	return t
}
