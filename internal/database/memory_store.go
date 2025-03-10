package database

import (
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/logger"
)

type MemoryStore struct {
	sessions     map[string]*SessionData
	willMessages map[string]*WillMessage
}

var Store *MemoryStore

func NewMemoryStore() *MemoryStore {
	if Store == nil {
		Store = &MemoryStore{
			sessions:     make(map[string]*SessionData),
			willMessages: make(map[string]*WillMessage),
		}
	}
	return Store
}

func (ms *MemoryStore) GetSession(clientID string) *SessionData {
	session, err := ms.sessions[clientID]
	if !err {
		logger.ErrorF("Session does not exist for clientID %s", clientID)
		return nil
	}
	return session
}

func (ms *MemoryStore) SaveSession(session *SessionData) bool {
	ms.sessions[session.ClientID] = session
	return true
}

func (ms *MemoryStore) DeleteSession(clientID string) bool {
	delete(ms.sessions, clientID)
	return true
}

func (ms *MemoryStore) GetWillMessage(clientID string) *WillMessage {
	message, err := ms.willMessages[clientID]
	if !err {
		logger.ErrorF("WillMessage does not exist for clientID %s", clientID)
		return nil
	}
	return message
}

func (ms *MemoryStore) SaveWillMessage(willMessage *WillMessage) bool {
	ms.willMessages[willMessage.ClientID] = willMessage
	return true
}

func (ms *MemoryStore) DeleteWillMessage(clientID string) bool {
	delete(ms.willMessages, clientID)
	return true
}
