package session

type Topic struct {
	Bus         chan string
	Subscribers map[string]chan []byte
}

type Broker struct {
	Topics map[string]*Topic
}

var b *Broker = &Broker{}

func GetTopic(topicName, clientID string, receiver chan []byte) chan string {
	topic := b.Topics[topicName]
	if topic == nil {
		topic = newTopic()
		topic.Subscribers = map[string]chan []byte{
			clientID: receiver,
		}
	}

	topic.Subscribers[clientID] = receiver
	return topic.Bus
}

func Publish(topicName string, payload []byte) {
	for _, v := range b.Topics[topicName].Subscribers {
		v <- payload
	}
}

func newTopic() *Topic {
	t := new(Topic)
	t.Bus = make(chan string, 100)
	go t.listenUnsub()
	return t
}

func (to *Topic) listenUnsub() {
	for {
		select {
		case clientID := <-to.Bus:
			delete(to.Subscribers, clientID)
		}
	}
}
