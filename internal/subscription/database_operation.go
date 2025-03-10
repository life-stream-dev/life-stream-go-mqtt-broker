package subscription

import (
	"context"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/database"
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

var nodeCache = expirable.NewLRU[string, *TopicTreeNode](256, nil, time.Hour)

func getNodeByPath(path string) *TopicTreeNode {
	if node, ok := nodeCache.Get(path); ok {
		return node
	}
	ctx, cancel := context.WithTimeout(context.Background(), database.OperationTimeout)
	defer cancel()

	filter := bson.D{{"path", path}}
	var topicNode TopicTreeNode

	startTime := time.Now()
	err := database.Subscriptions.FindOne(ctx, filter).Decode(&topicNode)
	logger.DebugF("query topic tree node cost: %v", time.Since(startTime))

	if err != nil {
		database.HandleErr(err)
		return nil
	}

	nodeCache.Add(path, &topicNode)
	return &topicNode
}

func getNodeByID(id primitive.ObjectID) *TopicTreeNode {
	if node, ok := nodeCache.Get(id.String()); ok {
		return node
	}
	ctx, cancel := context.WithTimeout(context.Background(), database.OperationTimeout)
	defer cancel()

	filter := bson.D{{"_id", id}}
	var topicNode TopicTreeNode

	startTime := time.Now()
	err := database.Subscriptions.FindOne(ctx, filter).Decode(&topicNode)
	logger.DebugF("query topic tree node cost: %v", time.Since(startTime))

	if err != nil {
		database.HandleErr(err)
		return nil
	}

	nodeCache.Add(id.String(), &topicNode)
	return &topicNode
}

func (topic *TopicTreeNode) save() {
	nodeCache.Remove(topic.Path)

	ctx, cancel := context.WithTimeout(context.Background(), database.OperationTimeout)
	defer cancel()

	filter := bson.D{{"_id", topic.ID}}
	opts := options.Replace().SetUpsert(true)

	result, err := database.Database.Collection(database.SubscriptionCollectionName).ReplaceOne(ctx, filter, topic, opts)

	if err != nil {
		database.HandleErr(err)
		return
	}

	logger.DebugF("Topic saved successfully, path=%s, level=%s, matched=%d, modified=%d, upserted=%v",
		topic.Path,
		topic.Level,
		result.MatchedCount,
		result.ModifiedCount,
		result.UpsertedID != nil,
	)
}
