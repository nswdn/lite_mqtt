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
	go b.publish()
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

func (broker *Broker) publish() {
	var message *publishMessage
	for {
		select {
		case message = <-broker.publishChan:
			broker.rwMutex.RLock()
			topic, ok := broker.Topics[message.topic]
			if !ok {
				broker.rwMutex.RUnlock()
				return
			}
			for _, v := range topic.Subscribers {
				v <- message.payload
			}
			broker.rwMutex.RUnlock()
		}
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
