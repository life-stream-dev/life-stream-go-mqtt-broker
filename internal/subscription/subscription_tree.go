package subscription

import (
	"fmt"
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/database"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"slices"
	"strings"
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
	WildcardPlus primitive.ObjectID      `bson:"wildcard_plus"` // "+" 通配符子节点（单层）
	WildcardHash []database.Subscription `bson:"wildcard_hash"` // "#" 通配符订阅列表（多层）

	// 终端订阅者（当前路径的精确匹配订阅）
	Terminals []database.Subscription `bson:"terminals"` // 精确匹配的订阅者
}

func DeleteSubscription(subscription database.Subscription) bool {
	levels := strings.Split(subscription.TopicName, "/")

	if slices.Contains(levels, "#") {
		path := strings.Join(levels[:len(levels)-1], "/")
		node := getNodeByPath(path)
		if node != nil {
			node.WildcardHash = slices.DeleteFunc(node.WildcardHash, func(sub database.Subscription) bool {
				return sub == subscription
			})
			node.save()
			return true
		}
		return false
	}

	node := getNodeByPath(subscription.TopicName)
	if node != nil {
		node.Terminals = slices.DeleteFunc(node.Terminals, func(sub database.Subscription) bool {
			return sub == subscription
		})
		node.save()
		return true
	}
	return false
}

func InsertSubscription(subscription database.Subscription) error {
	levels := strings.Split(subscription.TopicName, "/")

	var currentNode *TopicTreeNode = nil
	var parentNode *TopicTreeNode = nil

	for i, level := range levels {
		path := strings.Join(levels[:i+1], "/")
		currentNode = getOrCreateNode(path, level)

		if parentNode == nil {

		} else if level == "+" {
			parentNode.WildcardPlus = currentNode.ID
			parentNode.save()
		} else if level == "#" {
			if i != len(levels)-1 {
				return fmt.Errorf("'#' must be the last level, topic: %s", subscription.TopicName)
			}
			parentNode.WildcardHash = append(parentNode.WildcardHash, subscription)
			parentNode.save()
			return nil
		} else {
			if _, ok := parentNode.Children[level]; !ok {
				parentNode.Children[level] = currentNode.ID
				parentNode.save()
			}
		}

		if i == len(levels)-1 && !slices.Contains(currentNode.Terminals, subscription) {
			currentNode.Terminals = append(currentNode.Terminals, subscription)
			currentNode.save()
		}

		parentNode = currentNode
	}

	return nil
}

func MatchTopic(publishTopic string) ([]database.Subscription, error) {
	// 拆分发布主题为层级数组
	levels := strings.Split(publishTopic, "/")
	var results []database.Subscription

	// 初始化队列：从根节点开始（假设根节点 path 为空）
	rootNode := getNodeByPath(levels[0])
	rootPlus := getNodeByPath("+")
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
				childNode := getNodeByID(childID) // 从 MongoDB 获取子节点
				nextQueue = append(nextQueue, childNode)
			}

			// 3. 处理 + 通配符子节点
			if !node.WildcardPlus.IsZero() {
				plusNode := getNodeByID(node.WildcardPlus)
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
	finalResults := make([]database.Subscription, 0)
	for _, sub := range results {
		key := sub.ClientID + "|" + sub.TopicName
		if !seen[key] {
			finalResults = append(finalResults, sub)
			seen[key] = true
		}
	}

	return finalResults, nil
}
