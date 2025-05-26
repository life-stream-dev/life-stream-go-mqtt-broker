// Package database 实现了MQTT服务器的数据存储功能
package database

import "github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/logger"

// SessionData 表示MQTT客户端的会话数据
type SessionData struct {
	ClientID       string              `bson:"client_id"`       // 客户端ID
	TempSession    bool                `bson:"temp_session"`    // 是否为临时会话
	Subscriptions  map[string]byte     `bson:"subscriptions"`   // 订阅的主题和QoS级别
	PendingPublish map[uint16]string   `bson:"pending_publish"` // 待确认的QoS 1/2消息
	PendingPubrel  map[uint16]struct{} `bson:"pending_pubrel"`  // 等待PUBREL的QoS 2消息
	InflightQoS2   map[uint16]string   `bson:"inflight_qos2"`   // 已发送但未完成的QoS 2消息
}

// NewSessionData 创建新的会话数据实例
func NewSessionData(clientID string) *SessionData {
	return &SessionData{
		ClientID:       clientID,
		TempSession:    false,
		Subscriptions:  make(map[string]byte),
		PendingPublish: make(map[uint16]string),
		PendingPubrel:  make(map[uint16]struct{}),
		InflightQoS2:   make(map[uint16]string),
	}
}

// Save 保存会话数据到数据库
func (session *SessionData) Save() bool {
	return store.SaveSession(session)
}

// AddSubscription 添加主题订阅
func (session *SessionData) AddSubscription(subscription *Subscription) {
	subscription.ClientID = session.ClientID
	err := store.InsertSubscription(subscription)
	if err != nil {
		logger.ErrorF("Error while inserting subscription %v", err)
	}
	session.Subscriptions[subscription.TopicName] = subscription.QoSLevel
}

// RemoveSubscription 移除主题订阅
func (session *SessionData) RemoveSubscription(subscription *Subscription) {
	subscription.ClientID = session.ClientID
	store.DeleteSubscription(subscription)
	delete(session.Subscriptions, subscription.TopicName)
}

// RemoveAllSubscriptions 移除所有主题订阅
func (session *SessionData) RemoveAllSubscriptions() {
	for k, v := range session.Subscriptions {
		store.DeleteSubscription(&Subscription{
			ClientID:  session.ClientID,
			TopicName: k,
			QoSLevel:  v,
		})
	}
}
