package session

import "sync"

var sessionStore *sessionHolder

type sessionHolder struct {
	onlineSession map[string]*Session
	mutex         sync.Mutex
}

func init() {
	sessionStore = &sessionHolder{
		onlineSession: make(map[string]*Session),
	}
}

// A client id could not online second times, offline the first.
func (ss *sessionHolder) put(clientID string, session *Session) {

	ss.mutex.Lock()

	s, ok := ss.onlineSession[clientID]
	ss.onlineSession[clientID] = session

	ss.mutex.Unlock()

	if ok {
		_ = s.Conn.Close()
	}
}

func (ss *sessionHolder) delete(clientID string) {
	ss.mutex.Lock()
	delete(ss.onlineSession, clientID)
	ss.mutex.Unlock()
}
