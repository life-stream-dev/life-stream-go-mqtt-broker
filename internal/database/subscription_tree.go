package database

import (
	"context"
	"fmt"
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"slices"
	"strings"
	"time"
)

// TopicTreeNode 主题订阅树节点
type TopicTreeNode struct {
	ID    primitive.ObjectID `bson:"_id"`   // MongoDB文档ID
	Path  string             `bson:"path"`  // 物化路径（如 "sport/football"）
	Level string             `bson:"level"` // 当前层级名称（如 "football"）

	// 直接子节点（精确匹配）
	// 示例：sport/football 的直接子节点是 live（对应路径 sport/football/live）
	Children map[string]primitive.ObjectID `bson:"children"` // key=子层级名称, value=子节点ID

	// 通配符子节点
	WildcardPlus primitive.ObjectID `bson:"wildcard_plus"` // "+" 通配符子节点（单层）
	WildcardHash []Subscription     `bson:"wildcard_hash"` // "#" 通配符订阅列表（多层）

	// 终端订阅者（当前路径的精确匹配订阅）
	Terminals []Subscription `bson:"terminals"` // 精确匹配的订阅者
}

func createNode(path string, level string) *TopicTreeNode {
	result := &TopicTreeNode{
		ID:           primitive.NewObjectID(),
		Path:         path,
		Level:        level,
		Children:     map[string]primitive.ObjectID{},
		Terminals:    []Subscription{},
		WildcardHash: []Subscription{},
	}
	result.save()
	return result
}

func (ds *DBStore) getOrCreateNode(path string, level string) *TopicTreeNode {
	result := ds.getNodeByPath(path)
	if result != nil {
		return result
	}
	return createNode(path, level)
}

func (ds *DBStore) getNodeByPath(path string) *TopicTreeNode {
	if node, ok := ds.topicCache.Get(path); ok {
		return node
	}
	ctx, cancel := context.WithTimeout(context.Background(), OperationTimeout)
	defer cancel()

	filter := bson.D{{"path", path}}
	var topicNode TopicTreeNode

	startTime := time.Now()
	err := Subscriptions.FindOne(ctx, filter).Decode(&topicNode)
	logger.DebugF("query topic tree node cost: %v", time.Since(startTime))

	if err != nil {
		handleErr(err)
		return nil
	}

	ds.topicCache.Add(path, &topicNode)
	return &topicNode
}

func (ds *DBStore) getNodeByID(id primitive.ObjectID) *TopicTreeNode {
	if node, ok := ds.topicCache.Get(id.String()); ok {
		return node
	}
	ctx, cancel := context.WithTimeout(context.Background(), OperationTimeout)
	defer cancel()

	filter := bson.D{{"_id", id}}
	var topicNode TopicTreeNode

	startTime := time.Now()
	err := Subscriptions.FindOne(ctx, filter).Decode(&topicNode)
	logger.DebugF("query topic tree node cost: %v", time.Since(startTime))

	if err != nil {
		handleErr(err)
		return nil
	}

	ds.topicCache.Add(id.String(), &topicNode)
	return &topicNode
}

func (topic *TopicTreeNode) save() {
	store.topicCache.Remove(topic.Path)

	ctx, cancel := context.WithTimeout(context.Background(), OperationTimeout)
	defer cancel()

	filter := bson.D{{"_id", topic.ID}}
	opts := options.Replace().SetUpsert(true)

	result, err := Database.Collection(SubscriptionCollectionName).ReplaceOne(ctx, filter, topic, opts)

	if err != nil {
		handleErr(err)
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

func (sc *Subscription) equal(subscription Subscription) bool {
	return subscription.ClientID == sc.ClientID && subscription.TopicName == sc.TopicName
}

func (sc *Subscription) equalId(subscription Subscription) bool {
	return subscription.ClientID == sc.ClientID
}

func (ds *DBStore) ClearSubscription(session *SessionData) {

}

func (ds *DBStore) DeleteSubscription(subscription *Subscription) bool {
	levels := strings.Split(subscription.TopicName, "/")

	if slices.Contains(levels, "#") {
		path := strings.Join(levels[:len(levels)-1], "/")
		node := ds.getNodeByPath(path)
		if node != nil {
			node.WildcardHash = slices.DeleteFunc(node.WildcardHash, subscription.equal)
			node.save()
			return true
		}
		return false
	}

	node := ds.getNodeByPath(subscription.TopicName)
	if node != nil {
		node.Terminals = slices.DeleteFunc(node.Terminals, subscription.equal)
		node.save()
		return true
	}
	return false
}

func (ds *DBStore) InsertSubscription(subscription *Subscription) error {
	levels := strings.Split(subscription.TopicName, "/")

	var currentNode *TopicTreeNode = nil
	var parentNode *TopicTreeNode = nil

	for i, level := range levels {
		path := strings.Join(levels[:i+1], "/")
		currentNode = ds.getOrCreateNode(path, level)

		if parentNode == nil {

		} else if level == "+" {
			parentNode.WildcardPlus = currentNode.ID
			parentNode.save()
		} else if level == "#" {
			if i != len(levels)-1 {
				return fmt.Errorf("'#' must be the last level, topic: %s", subscription.TopicName)
			}
			parentNode.WildcardHash = append(parentNode.WildcardHash, *subscription)
			parentNode.save()
			return nil
		} else {
			if _, ok := parentNode.Children[level]; !ok {
				parentNode.Children[level] = currentNode.ID
				parentNode.save()
			}
		}

		if i == len(levels)-1 && !slices.ContainsFunc(currentNode.Terminals, subscription.equal) {
			currentNode.Terminals = append(currentNode.Terminals, *subscription)
			currentNode.save()
		}

		parentNode = currentNode
	}

	return nil
}

func (ds *DBStore) MatchTopic(publishTopic string) ([]Subscription, error) {
	// 拆分发布主题为层级数组
	levels := strings.Split(publishTopic, "/")
	var results []Subscription

	// 初始化队列：从根节点开始（假设根节点 path 为空）
	rootNode := ds.getNodeByPath(levels[0])
	rootPlus := ds.getNodeByPath("+")
	var queue []*TopicTreeNode
	if rootPlus != nil {
		queue = append(queue, rootPlus)
	}
	if rootNode != nil {
		queue = append(queue, rootNode)
	}

	for _, currentLevel := range levels[1:] {
		var nextQueue []*TopicTreeNode

		// 遍历当前层所有可能匹配的节点
		for _, node := range queue {
			// 1. 收集当前节点的 # 通配符订阅
			results = append(results, node.WildcardHash...)

			// 2. 精确匹配子节点
			if childID, ok := node.Children[currentLevel]; ok {
				childNode := ds.getNodeByID(childID) // 从 MongoDB 获取子节点
				nextQueue = append(nextQueue, childNode)
			}

			// 3. 处理 + 通配符子节点
			if !node.WildcardPlus.IsZero() {
				plusNode := ds.getNodeByID(node.WildcardPlus)
				nextQueue = append(nextQueue, plusNode)
			}
		}

		// 4. 更新队列为下一层节点
		queue = nextQueue

		// 提前终止：队列为空时无需继续
		if len(queue) == 0 {
			break
		}
	}

	// 5. 收集终端节点的精确订阅
	for _, node := range queue {
		results = append(results, node.Terminals...)
	}

	// 6. 去重（假设 Subscription 有唯一标识如 ClientID+Topic）
	seen := make(map[string]bool)
	finalResults := make([]Subscription, 0)
	for _, sub := range results {
		key := sub.ClientID + "|" + sub.TopicName
		if !seen[key] {
			finalResults = append(finalResults, sub)
			seen[key] = true
		}
	}

	return finalResults, nil
}
