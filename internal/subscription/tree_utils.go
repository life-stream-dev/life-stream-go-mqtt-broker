package subscription

import (
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/database"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func createNode(path string, level string) *TopicTreeNode {
	result := &TopicTreeNode{
		ID:           primitive.NewObjectID(),
		Path:         path,
		Level:        level,
		Children:     map[string]primitive.ObjectID{},
		Terminals:    []database.Subscription{},
		WildcardHash: []database.Subscription{},
	}
	result.save()
	return result
}

func getOrCreateNode(path string, level string) *TopicTreeNode {
	result := getNodeByPath(path)
	if result != nil {
		return result
	}
	return createNode(path, level)
}
