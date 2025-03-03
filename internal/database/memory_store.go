package database

import "errors"

type MemoryStore struct {
	sessions     map[string]*SessionData
	willMessages map[string]*WillMessage
}

var Store *MemoryStore

func NewMemoryStore() *MemoryStore {
	if Store == nil {
		Store = &MemoryStore{sessions: make(map[string]*SessionData)}
	}
	return Store
}

func (ms *MemoryStore) GetSession(clientID string) (*SessionData, error) {
	session, err := ms.sessions[clientID]
	if !err {
		return nil, errors.New("session not found")
	}
	return session, nil
}

func (ms *MemoryStore) SaveSession(session *SessionData) error {
	ms.sessions[session.ClientID] = session
	return nil
}

func (ms *MemoryStore) DeleteSession(clientID string) error {
	delete(ms.sessions, clientID)
	return nil
}

func (ms *MemoryStore) GetWillMessage(clientID string) (*WillMessage, error) {
	message, err := ms.willMessages[clientID]
	if !err {
		return nil, errors.New("message not found")
	}
	return message, nil
}

func (ms *MemoryStore) SaveWillMessage(willMessage *WillMessage) error {
	ms.willMessages[willMessage.ClientID] = willMessage
	return nil
}

func (ms *MemoryStore) DeleteWillMessage(clientID string) error {
	delete(ms.willMessages, clientID)
	return nil
}
