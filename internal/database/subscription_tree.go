// Package database 实现了MQTT服务器的数据存储功能
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

// TopicTreeNode 表示主题订阅树的节点
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

// createNode 创建新的主题树节点
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

// getOrCreateNode 获取或创建主题树节点
func (ds *DBStore) getOrCreateNode(path string, level string) *TopicTreeNode {
	result := ds.getNodeByPath(path)
	if result != nil {
		return result
	}
	return createNode(path, level)
}

// getNodeByPath 根据路径获取主题树节点
func (ds *DBStore) getNodeByPath(path string) *TopicTreeNode {
	// 首先检查缓存
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

// getNodeByID 根据ID获取主题树节点
func (ds *DBStore) getNodeByID(id primitive.ObjectID) *TopicTreeNode {
	// 首先检查缓存
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

// save 保存主题树节点到数据库
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

// equal 比较两个订阅是否相等
func (sc *Subscription) equal(subscription Subscription) bool {
	return subscription.ClientID == sc.ClientID && subscription.TopicName == sc.TopicName
}

// equalId 比较两个订阅的客户端ID是否相等
func (sc *Subscription) equalId(subscription Subscription) bool {
	return subscription.ClientID == sc.ClientID
}

// ClearSubscription 清除会话的所有订阅
func (ds *DBStore) ClearSubscription(session *SessionData) {

}

// DeleteSubscription 删除订阅
func (ds *DBStore) DeleteSubscription(subscription *Subscription) bool {
	levels := strings.Split(subscription.TopicName, "/")

	// 处理通配符订阅
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

	// 处理普通订阅
	node := ds.getNodeByPath(subscription.TopicName)
	if node != nil {
		node.Terminals = slices.DeleteFunc(node.Terminals, subscription.equal)
		node.save()
		return true
	}
	return false
}

// InsertSubscription 插入新的订阅
func (ds *DBStore) InsertSubscription(subscription *Subscription) error {
	levels := strings.Split(subscription.TopicName, "/")

	var currentNode *TopicTreeNode = nil
	var parentNode *TopicTreeNode = nil

	for i, level := range levels {
		path := strings.Join(levels[:i+1], "/")
		currentNode = ds.getOrCreateNode(path, level)

		if parentNode == nil {
			// 根节点处理
		} else if level == "+" {
			// 处理单层通配符
			parentNode.WildcardPlus = currentNode.ID
			parentNode.save()
		} else if level == "#" {
			// 处理多层通配符
			if i != len(levels)-1 {
				return fmt.Errorf("'#' must be the last level, topic: %s", subscription.TopicName)
			}
			parentNode.WildcardHash = append(parentNode.WildcardHash, *subscription)
			parentNode.save()
			return nil
		} else {
			// 处理普通层级
			if _, ok := parentNode.Children[level]; !ok {
				parentNode.Children[level] = currentNode.ID
				parentNode.save()
			}
		}

		// 如果是最后一个层级，添加终端订阅
		if i == len(levels)-1 && !slices.ContainsFunc(currentNode.Terminals, subscription.equal) {
			currentNode.Terminals = append(currentNode.Terminals, *subscription)
			currentNode.save()
		}

		parentNode = currentNode
	}

	return nil
}

// MatchTopic 匹配主题订阅
func (ds *DBStore) MatchTopic(publishTopic string) ([]Subscription, error) {
	// 拆分发布主题为层级数组
	levels := strings.Split(publishTopic, "/")
	var results []Subscription

	// 初始化队列：从根节点开始
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
				childNode := ds.getNodeByID(childID)
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

	return results, nil
}
