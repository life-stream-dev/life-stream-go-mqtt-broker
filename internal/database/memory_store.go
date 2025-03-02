package database

import "errors"

type MemorySessionStore struct {
	sessions map[string]*SessionData
}

var MemoryStore *MemorySessionStore

func NewMemorySessionStore() *MemorySessionStore {
	if MemoryStore == nil {
		MemoryStore = &MemorySessionStore{sessions: make(map[string]*SessionData)}
	}
	return MemoryStore
}

func (mss *MemorySessionStore) Get(clientID string) (*SessionData, error) {
	session, err := mss.sessions[clientID]
	if !err {
		return nil, errors.New("session not found")
	}
	return session, nil
}

func (mss *MemorySessionStore) Save(session *SessionData) error {
	mss.sessions[session.ClientID] = session
	return nil
}

func (mss *MemorySessionStore) Delete(clientID string) error {
	delete(mss.sessions, clientID)
	return nil
}
