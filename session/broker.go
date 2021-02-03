package session

import (
	"log"
	"sync"
)

type Topic struct {
	Name        string
	Bus         chan string
	Subscribers map[string]chan []byte
	mutex       sync.Mutex
}

type Broker struct {
	Topics  map[string]*Topic
	rwMutex sync.RWMutex
}

var b = &Broker{
	Topics: make(map[string]*Topic),
}

func (broker *Broker) GetTopic(topicName, clientID string, receiver chan []byte) chan string {
	broker.rwMutex.Lock()
	defer broker.rwMutex.Unlock()

	topic := broker.Topics[topicName]
	if topic == nil {
		topic = newTopic(topicName)
		topic.Subscribers = map[string]chan []byte{
			clientID: receiver,
		}
	}

	topic.addReceiver(clientID, receiver)
	broker.Topics[topicName] = topic

	return topic.Bus
}

func (broker *Broker) Publish(topicName string, payload []byte) {
	broker.rwMutex.RLock()
	defer broker.rwMutex.RUnlock()

	topic, ok := broker.Topics[topicName]
	if !ok {
		return
	}
	for _, v := range topic.Subscribers {
		v <- payload
	}
}

func newTopic(name string) *Topic {
	t := new(Topic)
	t.Name = name
	t.Bus = make(chan string, 100)
	go t.listenUnsub()
	return t
}

func (to *Topic) addReceiver(clientID string, receiver chan []byte) {
	to.Subscribers[clientID] = receiver
}

func (to *Topic) listenUnsub() {
	for {
		select {
		case clientID := <-to.Bus:
			to.mutex.Lock()
			delete(to.Subscribers, clientID)
			to.mutex.Unlock()
			log.Printf("%s unsubscribe topic: %s", clientID, to.Name)
		}
	}
}
