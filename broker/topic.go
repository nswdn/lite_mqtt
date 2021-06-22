package broker

import "log"

type retainMessage []byte

var emptyRetain retainMessage = nil

type Topic struct {
	Name        string
	Subscribers map[string]subscriber // key: clientID, value: receiver connection
	retain      retainMessage
}

func (to *Topic) unsubscribe(clientID string) {
	//to.mutex.Lock()
	//defer to.mutex.Unlock()
	delete(to.Subscribers, clientID)
}

func (to *Topic) put(clientID string, receiver subscriber) {
	//to.mutex.Lock()
	//defer to.mutex.Unlock()
	to.Subscribers[clientID] = receiver
}

func newTopic(name string, id string, receiver subscriber) *Topic {
	t := &Topic{
		Name:        name,
		Subscribers: make(map[string]subscriber),
	}

	if id != empty {
		t.Subscribers[id] = receiver
	}

	log.Println("new topic name:", name)
	return t
}
