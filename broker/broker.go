package broker

import (
	"excel_parser/proto"
	"log"
)

//var (
//	blocking    int32 = 0
//	dispatching int32 = 1
//)

type publishMessage struct {
	topic   string
	payload []byte
	retain  bool
}

type subscriber struct {
	topic        string
	clientID     string
	receiverChan chan []byte
	qos          proto.QOS
}

type unsubscribeMessage struct {
	topic    string
	clientID string
}

type Dispatcher interface {
	handlePublish(publishMessage)
	handleSubscribe(subscriber)
	handleUnsubscribe(unsubscribeMessage)
	removeCache()
	putRemoveCache(string, string)
	popRemoveCache(string, string)
}

type Broker struct {
	topicTrie   *trie
	publishChan chan publishMessage
	subChan     chan subscriber
	unsubChan   chan unsubscribeMessage
	//dispatching   int32                        // 0: false, 1: true
	//stashingCache map[string]map[string]string // client to be removed from topic key: topic name, value: a map to contain clientIDs, key: clientID
}

var b *Broker

func init() {
	b = &Broker{
		topicTrie:   tr,
		publishChan: make(chan publishMessage, 1000),
		subChan:     make(chan subscriber, 1000),
		unsubChan:   make(chan unsubscribeMessage, 1000),
		//dispatching:   0,
		//stashingCache: map[string]map[string]string{},
	}

	go b.dispatcherHandle()
}

func (broker *Broker) dispatcherHandle() {
	for {
		select {
		case publishMsg := <-broker.publishChan:
			broker.handlePublish(publishMsg)
		case follower := <-broker.subChan:
			broker.handleSubscribe(follower)
		case unsubMsg := <-broker.unsubChan:
			broker.handleUnsubscribe(unsubMsg)
		}
	}
}

func (broker *Broker) handlePublish(publishMsg publishMessage) {
	var match *trieNode
	var follower subscriber

	if match = broker.topicTrie.match(publishMsg.topic); match != nil {
		if publishMsg.retain {
			if len(publishMsg.payload) > 0 {
				match.topic.retain = publishMsg.payload
			} else {
				match.topic.retain = emptyRetain
			}
		}

		for _, follower = range match.topic.Subscribers {
			publish, err := proto.NewPublish(0, follower.qos, 0, publishMsg.topic, publishMsg.payload)
			if err != nil {
				log.Printf("[topic: %s]: failed to generate retain msg", follower.topic)
				continue
			}
			follower.receiverChan <- publish
		}
	}
}

func (broker *Broker) handleSubscribe(follower subscriber) {

	result := b.topicTrie.insert(follower)
	if result.topic.retain != nil {
		publish, err := proto.NewPublish(0, follower.qos, 0, result.topic.Name, result.topic.retain)
		if err != nil {
			log.Printf("[topic: %s]: failed to generate retain msg", follower.topic)
		}
		follower.receiverChan <- publish
	}
}

func (broker *Broker) handleUnsubscribe(unsubMsg unsubscribeMessage) {
	if match := b.topicTrie.match(unsubMsg.topic); match != nil {
		delete(match.topic.Subscribers, unsubMsg.clientID)
	}
}

func Subscribe(topicName, clientID string, receiverChan chan []byte, qos proto.QOS) {
	b.subChan <- subscriber{topic: topicName, clientID: clientID, receiverChan: receiverChan, qos: qos}
}

func Unsubscribe(topicName string, clientID string) {
	b.unsubChan <- unsubscribeMessage{clientID: clientID, topic: topicName}
}

func Publish(topic string, retain bool, payload []byte) {
	b.publishChan <- publishMessage{topic, payload, retain}
}

// Remove all unsub clients that cached before.
// Resolved a large number of clients disconnected from the server,
// this causes the server dispatcherHandle routine to be blocked for a long time and
// could not respond new clients' command.
//func (broker *Broker) removeCache() {
//	ticker := time.NewTicker(time.Second * 10)
//	defer ticker.Stop()
//
//	var cleanedCount int
//	var totalCleaned int
//
//	for range ticker.C {
//		broker.Lock()
//		for topicName, clientIDMap := range broker.stashingCache {
//			topic := broker.topicTrie.match(topicName)
//			for clientID := range clientIDMap {
//				topic.topic.unsubscribe(clientID)
//				cleanedCount++
//				totalCleaned++
//				delete(broker.stashingCache[topicName], clientID)
//			}
//		}
//		log.Printf("cleaned [%d] disconencted subscribers, total cleaned [%d]", cleanedCount, totalCleaned)
//		broker.Unlock()
//		cleanedCount = 0
//	}
//}
//
//// Mark client to remove status
//func (broker *Broker) putRemoveCache(topicName, clientID string) {
//	cache, ok := broker.stashingCache[topicName]
//	if !ok {
//		cache = map[string]string{}
//		broker.stashingCache[topicName] = cache
//	}
//	cache[clientID] = clientID
//}
//
//// Unmark client
//func (broker *Broker) popRemoveCache(topicName, clientID string) {
//	cache, ok := broker.stashingCache[topicName]
//	if ok {
//		delete(cache, clientID)
//	}
//}
//func (broker *Broker) Lock() {
//	for !atomic.CompareAndSwapInt32(&broker.dispatching, blocking, dispatching) {
//		continue
//	}
//}
//func (broker *Broker) Unlock() {
//	for !atomic.CompareAndSwapInt32(&broker.dispatching, dispatching, blocking) {
//		continue
//	}
//}
