package database

import (
	"context"
	"errors"
	"fmt"
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

type DBStore struct {
	client *mongo.Client
	db     *mongo.Database
}

var (
	DbStore            *DBStore
	ClientIdEmptyError = errors.New("client_id is empty")
)

func NewDatabaseStore() *DBStore {
	if DbStore == nil {
		DbStore = &DBStore{client: Client, db: Database}
	}
	return DbStore
}

func (ds *DBStore) GetSession(clientID string) (*SessionData, error) {
	ctx, cancel := context.WithTimeout(context.Background(), OperationTimeout)
	defer cancel()

	if clientID == "" {
		return nil, ClientIdEmptyError
	}

	filter := bson.D{{"client_id", clientID}}
	var session SessionData

	startTime := time.Now()
	err := Database.Collection(SessionCollectionName).FindOne(ctx, filter).Decode(&session)
	logger.DebugF("session query cost: %v", time.Since(startTime))

	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil, fmt.Errorf("unique key conflicts: %w", err)
		}
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("document does not exist: %w", err)
		}
		return nil, fmt.Errorf("database operation failed: %w", err)
	}
	return &session, nil
}

func (ds *DBStore) SaveSession(sessionData *SessionData) error {
	ctx, cancel := context.WithTimeout(context.Background(), OperationTimeout)
	defer cancel()

	if sessionData.ClientID == "" {
		return ClientIdEmptyError
	}

	filter := bson.D{{"client_id", sessionData.ClientID}}
	opts := options.Replace().SetUpsert(true)

	result, err := Database.Collection(SessionCollectionName).ReplaceOne(ctx, filter, sessionData, opts)

	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return fmt.Errorf("unique key conflicts: %w", err)
		}
		if errors.Is(err, mongo.ErrNoDocuments) {
			return fmt.Errorf("document does not exist: %w", err)
		}
		return fmt.Errorf("database operation failed: %w", err)
	}

	logger.InfoF("Session saved: client_id=%s, matched=%d, modified=%d, upserted=%v",
		sessionData.ClientID,
		result.MatchedCount,
		result.ModifiedCount,
		result.UpsertedID != nil,
	)

	return nil
}

func (ds *DBStore) DeleteSession(clientID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), OperationTimeout)
	defer cancel()

	if clientID == "" {
		return ClientIdEmptyError
	}

	filter := bson.D{{"client_id", clientID}}
	result, err := Database.Collection(SessionCollectionName).DeleteOne(ctx, filter)

	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return fmt.Errorf("unique key conflicts: %w", err)
		}
		if errors.Is(err, mongo.ErrNoDocuments) {
			return fmt.Errorf("document does not exist: %w", err)
		}
		return fmt.Errorf("database operation failed: %w", err)
	}

	logger.InfoF("Session deleted: client_id=%s, deleted=%d", clientID, result.DeletedCount)

	return nil
}

func (ds *DBStore) GetWillMessage(clientID string) (*WillMessage, error) {
	ctx, cancel := context.WithTimeout(context.Background(), OperationTimeout)
	defer cancel()

	if clientID == "" {
		return nil, ClientIdEmptyError
	}

	filter := bson.D{{"client_id", clientID}}
	var message WillMessage

	startTime := time.Now()
	err := Database.Collection(WillMessageCollectionName).FindOne(ctx, filter).Decode(&message)
	logger.DebugF("will message query cost: %v", time.Since(startTime))

	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil, fmt.Errorf("unique key conflicts: %w", err)
		}
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("document does not exist: %w", err)
		}
		return nil, fmt.Errorf("database operation failed: %w", err)
	}

	return &message, nil
}

func (ds *DBStore) SaveWillMessage(willMessage *WillMessage) error {
	ctx, cancel := context.WithTimeout(context.Background(), OperationTimeout)
	defer cancel()

	if willMessage.ClientID == "" {
		return ClientIdEmptyError
	}

	filter := bson.D{{"client_id", willMessage.ClientID}}
	opts := options.Replace().SetUpsert(true)

	result, err := Database.Collection(WillMessageCollectionName).ReplaceOne(ctx, filter, willMessage, opts)

	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return fmt.Errorf("unique key conflicts: %w", err)
		}
		if errors.Is(err, mongo.ErrNoDocuments) {
			return fmt.Errorf("document does not exist: %w", err)
		}
		return fmt.Errorf("database operation failed: %w", err)
	}

	logger.InfoF("Session saved: client_id=%s, matched=%d, modified=%d, upserted=%v",
		willMessage.ClientID,
		result.MatchedCount,
		result.ModifiedCount,
		result.UpsertedID != nil,
	)

	return nil
}

func (ds *DBStore) DeleteWillMessage(clientID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), OperationTimeout)
	defer cancel()

	if clientID == "" {
		return ClientIdEmptyError
	}

	filter := bson.D{{"client_id", clientID}}
	result, err := Database.Collection(WillMessageCollectionName).DeleteOne(ctx, filter)

	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return fmt.Errorf("unique key conflicts: %w", err)
		}
		if errors.Is(err, mongo.ErrNoDocuments) {
			return fmt.Errorf("document does not exist: %w", err)
		}
		return fmt.Errorf("database operation failed: %w", err)
	}

	logger.InfoF("Session deleted: client_id=%s, deleted=%d", clientID, result.DeletedCount)

	return nil
}
