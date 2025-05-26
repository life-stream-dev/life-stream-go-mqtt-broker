// Package database 实现了MQTT服务器的数据存储功能
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

// DBStore 实现了数据库存储接口
type DBStore struct {
	client       *mongo.Client                          // MongoDB客户端
	db           *mongo.Database                        // 数据库实例
	sessions     map[string]*SessionData                // 内存中的会话数据
	willMessages map[string]*WillMessage                // 遗嘱消息
	sessionCache *expirable.LRU[string, *SessionData]   // 会话缓存
	topicCache   *expirable.LRU[string, *TopicTreeNode] // 主题树缓存
}

var (
	store              *DBStore
	ClientIdEmptyError = errors.New("client_id is empty")
)

// NewDatabaseStore 创建新的数据库存储实例
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

// handleErr 处理数据库操作错误
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

// GetSession 获取客户端会话数据
func (ds *DBStore) GetSession(clientID string) *SessionData {
	// 首先检查内存中的会话
	if session, ok := ds.sessions[clientID]; ok {
		return session
	}
	// 然后检查缓存
	if session, ok := ds.sessionCache.Get(clientID); ok {
		return session
	}
	// 最后从数据库查询
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

// SaveSession 保存客户端会话数据
func (ds *DBStore) SaveSession(sessionData *SessionData) bool {
	// 临时会话只保存在内存中
	if sessionData.TempSession {
		delete(ds.sessions, sessionData.ClientID)
		ds.sessions[sessionData.ClientID] = sessionData
		return true
	}

	// 从缓存中移除
	ds.sessionCache.Remove(sessionData.ClientID)
	ctx, cancel := context.WithTimeout(context.Background(), OperationTimeout)
	defer cancel()

	if sessionData.ClientID == "" {
		handleErr(ClientIdEmptyError)
		return false
	}

	// 保存到数据库
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

// DeleteSession 删除客户端会话数据
func (ds *DBStore) DeleteSession(clientID string) bool {
	// 从内存中删除
	if _, ok := ds.sessions[clientID]; ok {
		delete(ds.sessions, clientID)
		return true
	}

	// 从缓存中删除
	ds.sessionCache.Remove(clientID)

	ctx, cancel := context.WithTimeout(context.Background(), OperationTimeout)
	defer cancel()

	if clientID == "" {
		handleErr(ClientIdEmptyError)
		return false
	}

	// 从数据库中删除
	filter := bson.D{{"client_id", clientID}}
	result, err := Database.Collection(SessionCollectionName).DeleteOne(ctx, filter)

	if err != nil {
		handleErr(err)
		return false
	}

	logger.DebugF("Session deleted: client_id=%s, deleted=%d", clientID, result.DeletedCount)

	return true
}

// GetWillMessage 获取遗嘱消息
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

// SaveWillMessage 保存遗嘱消息
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

// DeleteWillMessage 删除遗嘱消息
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
