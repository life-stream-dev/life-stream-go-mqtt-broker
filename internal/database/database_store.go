package database

import (
	"context"
	"errors"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

type DBStore struct {
	client       *mongo.Client
	db           *mongo.Database
	sessions     map[string]*SessionData
	willMessages map[string]*WillMessage
	sessionCache *expirable.LRU[string, *SessionData]
	topicCache   *expirable.LRU[string, *TopicTreeNode]
}

var (
	store              *DBStore
	ClientIdEmptyError = errors.New("client_id is empty")
)

func NewDatabaseStore() *DBStore {
	if store == nil {
		store = &DBStore{
			client:       Client,
			db:           Database,
			sessions:     make(map[string]*SessionData),
			willMessages: make(map[string]*WillMessage),
			sessionCache: expirable.NewLRU[string, *SessionData](256, nil, time.Hour),
			topicCache:   expirable.NewLRU[string, *TopicTreeNode](256, nil, time.Hour),
		}
	}
	return store
}

func handleErr(err error) {
	if mongo.IsDuplicateKeyError(err) {
		logger.ErrorF("unique key conflicts: %s", err.Error())
		return
	}
	if errors.Is(err, mongo.ErrNoDocuments) {
		logger.ErrorF("document does not exist: %s", err.Error())
		return
	}
	if errors.Is(err, ClientIdEmptyError) {
		logger.ErrorF("Client id can not be empty: %s", err.Error())
		return
	}
	logger.ErrorF("database operation failed: %s", err.Error())
}

func (ds *DBStore) GetSession(clientID string) *SessionData {
	if session, ok := ds.sessions[clientID]; ok {
		return session
	}
	if session, ok := ds.sessionCache.Get(clientID); ok {
		return session
	}
	ctx, cancel := context.WithTimeout(context.Background(), OperationTimeout)
	defer cancel()

	if clientID == "" {
		handleErr(ClientIdEmptyError)
		return nil
	}

	filter := bson.D{{"client_id", clientID}}
	var session SessionData

	startTime := time.Now()
	err := Database.Collection(SessionCollectionName).FindOne(ctx, filter).Decode(&session)
	logger.DebugF("session query cost: %v", time.Since(startTime))

	if err != nil {
		handleErr(err)
		return nil
	}

	ds.sessionCache.Add(clientID, &session)
	return &session
}

func (ds *DBStore) SaveSession(sessionData *SessionData) bool {
	if sessionData.TempSession {
		delete(ds.sessions, sessionData.ClientID)
		ds.sessions[sessionData.ClientID] = sessionData
		return true
	}

	ds.sessionCache.Remove(sessionData.ClientID)
	ctx, cancel := context.WithTimeout(context.Background(), OperationTimeout)
	defer cancel()

	if sessionData.ClientID == "" {
		handleErr(ClientIdEmptyError)
		return false
	}

	filter := bson.D{{"client_id", sessionData.ClientID}}
	opts := options.Replace().SetUpsert(true)

	result, err := Database.Collection(SessionCollectionName).ReplaceOne(ctx, filter, sessionData, opts)

	if err != nil {
		handleErr(err)
		return false
	}

	logger.DebugF("Session saved: client_id=%s, matched=%d, modified=%d, upserted=%v",
		sessionData.ClientID,
		result.MatchedCount,
		result.ModifiedCount,
		result.UpsertedID != nil,
	)

	ds.sessionCache.Add(sessionData.ClientID, sessionData)
	return true
}

func (ds *DBStore) DeleteSession(clientID string) bool {
	if _, ok := ds.sessions[clientID]; ok {
		delete(ds.sessions, clientID)
		return true
	}

	ds.sessionCache.Remove(clientID)

	ctx, cancel := context.WithTimeout(context.Background(), OperationTimeout)
	defer cancel()

	if clientID == "" {
		handleErr(ClientIdEmptyError)
		return false
	}

	filter := bson.D{{"client_id", clientID}}
	result, err := Database.Collection(SessionCollectionName).DeleteOne(ctx, filter)

	if err != nil {
		handleErr(err)
		return false
	}

	logger.DebugF("Session deleted: client_id=%s, deleted=%d", clientID, result.DeletedCount)

	return true
}

func (ds *DBStore) GetWillMessage(clientID string) *WillMessage {
	ctx, cancel := context.WithTimeout(context.Background(), OperationTimeout)
	defer cancel()

	if clientID == "" {
		handleErr(ClientIdEmptyError)
		return nil
	}

	filter := bson.D{{"client_id", clientID}}
	var message WillMessage

	startTime := time.Now()
	err := Database.Collection(WillMessageCollectionName).FindOne(ctx, filter).Decode(&message)
	logger.DebugF("will message query cost: %v", time.Since(startTime))

	if err != nil {
		handleErr(err)
		return nil
	}

	return &message
}

func (ds *DBStore) SaveWillMessage(willMessage *WillMessage) bool {
	ctx, cancel := context.WithTimeout(context.Background(), OperationTimeout)
	defer cancel()

	if willMessage.ClientID == "" {
		handleErr(ClientIdEmptyError)
		return false
	}

	filter := bson.D{{"client_id", willMessage.ClientID}}
	opts := options.Replace().SetUpsert(true)

	result, err := Database.Collection(WillMessageCollectionName).ReplaceOne(ctx, filter, willMessage, opts)

	if err != nil {
		handleErr(err)
		return false
	}

	logger.DebugF("Session saved: client_id=%s, matched=%d, modified=%d, upserted=%v",
		willMessage.ClientID,
		result.MatchedCount,
		result.ModifiedCount,
		result.UpsertedID != nil,
	)

	return true
}

func (ds *DBStore) DeleteWillMessage(clientID string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), OperationTimeout)
	defer cancel()

	if clientID == "" {
		handleErr(ClientIdEmptyError)
		return false
	}

	filter := bson.D{{"client_id", clientID}}
	result, err := Database.Collection(WillMessageCollectionName).DeleteOne(ctx, filter)

	if err != nil {
		handleErr(err)
		return false
	}

	logger.DebugF("Session deleted: client_id=%s, deleted=%d", clientID, result.DeletedCount)

	return true
}
