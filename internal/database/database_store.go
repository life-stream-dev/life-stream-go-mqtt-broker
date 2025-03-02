package database

import (
	"context"
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/logger"
	"go.mongodb.org/mongo-driver/bson"
)

const CollectionName = "sessions"

type CachedDatabaseStore struct {
	sessions map[string]*SessionData
}

var CachedStore *CachedDatabaseStore

func NewCachedDatabaseStore() *CachedDatabaseStore {
	if CachedStore == nil {
		CachedStore = &CachedDatabaseStore{sessions: make(map[string]*SessionData)}
	}
	return CachedStore
}

func (cds *CachedDatabaseStore) Get(clientID string) (*SessionData, error) {
	ctx, cancel := context.WithTimeout(context.Background(), OperationTimeout)
	defer cancel()
	filter := bson.D{{"ClientID", clientID}}
	var result SessionData
	err := Database.Collection(CollectionName).FindOne(ctx, filter).Decode(&result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (cds *CachedDatabaseStore) Set(data *SessionData) error {
	ctx, cancel := context.WithTimeout(context.Background(), OperationTimeout)
	defer cancel()
	insertOne, err := Database.Collection(CollectionName).InsertOne(ctx, data)
	if err != nil {
		return err
	}
	logger.DebugF("Inserted new session data, %v", insertOne.InsertedID)
	return nil
}

func (cds *CachedDatabaseStore) Delete(clientID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), OperationTimeout)
	defer cancel()
	filter := bson.D{{"ClientID", clientID}}
	_, err := Database.Collection(CollectionName).DeleteOne(ctx, filter)
	return err
}
