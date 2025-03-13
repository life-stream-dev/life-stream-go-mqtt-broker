package database

import "github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/logger"

type SessionData struct {
	ClientID       string              `bson:"client_id"`
	TempSession    bool                `bson:"temp_session"`
	Subscriptions  map[string]byte     `bson:"subscriptions"`   // 主题: QoS
	PendingPublish map[uint16]string   `bson:"pending_publish"` // 未确认的 QoS 1/2 消息（PacketID -> Message）
	PendingPubrel  map[uint16]struct{} `bson:"pending_pubrel"`  // 等待 PUBREL 的 QoS 2 消息
	InflightQoS2   map[uint16]string   `bson:"inflight_qos2"`   // 已发送但未完成的 QoS 2 消息
}

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

func (session *SessionData) Save() bool {
	return store.SaveSession(session)
}

func (session *SessionData) AddSubscription(subscription *Subscription) {
	subscription.ClientID = session.ClientID
	err := store.InsertSubscription(subscription)
	if err != nil {
		logger.ErrorF("Error while inserting subscription %v", err)
	}
	session.Subscriptions[subscription.TopicName] = subscription.QoSLevel
}

func (session *SessionData) RemoveSubscription(subscription *Subscription) {
	subscription.ClientID = session.ClientID
	store.DeleteSubscription(subscription)
	delete(session.Subscriptions, subscription.TopicName)
}

func (session *SessionData) RemoveAllSubscriptions() {
	for k, v := range session.Subscriptions {
		store.DeleteSubscription(&Subscription{
			ClientID:  session.ClientID,
			TopicName: k,
			QoSLevel:  v,
		})
	}
}
