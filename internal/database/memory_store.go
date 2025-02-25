package database

import "errors"

type MemorySessionStore struct {
	sessions map[string]*SessionData
}

func NewMemorySessionStore() *MemorySessionStore {
	return &MemorySessionStore{sessions: make(map[string]*SessionData)}
}

func (mss *MemorySessionStore) Get(clientID string) (*SessionData, error) {
	session, err := mss.sessions[clientID]
	if !err {
		return nil, errors.New("session not found")
	}
	return session, nil
}

func (mss *MemorySessionStore) Save(clientID string, session *SessionData) error {
	if clientID != session.ClientID {
		return errors.New("clientID does not match")
	}
	mss.sessions[clientID] = session
	return nil
}

func (mss *MemorySessionStore) Delete(clientID string) error {
	delete(mss.sessions, clientID)
	return nil
}
