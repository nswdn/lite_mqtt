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

func (ss *sessionHolder) put(clientID string, session *Session) {

	ss.mutex.Lock()

	s, ok := ss.onlineSession[clientID]
	ss.onlineSession[clientID] = session

	ss.mutex.Unlock()

	if ok {
		_ = s.Close()
	}
}

func (ss *sessionHolder) delete(clientID string) {
	ss.mutex.Lock()
	delete(ss.onlineSession, clientID)
	ss.mutex.Unlock()
}
