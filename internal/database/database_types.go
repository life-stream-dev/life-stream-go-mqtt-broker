package database

const (
	SessionCollectionName      = "sessions"
	WillMessageCollectionName  = "will_messages"
	SubscriptionCollectionName = "subscriptions"
)

var collectionsList = []string{SessionCollectionName, WillMessageCollectionName, SubscriptionCollectionName}

type Subscription struct {
	ClientID  string `bson:"client_id"`
	TopicName string `bson:"topic_name"`
	QoSLevel  byte   `bson:"qos_level"`
}

type WillMessage struct {
	ClientID string `bson:"client_id"`
	Topic    []byte `bson:"topic"`
	QoS      byte   `bson:"qo_s"`
	Content  []byte `bson:"content"`
	Retained bool   `bson:"retained"`
}

type SessionStore interface {
	GetSession(clientID string) *SessionData
	SaveSession(session *SessionData) bool
	DeleteSession(clientID string) bool
}

type WillMessageStore interface {
	GetWillMessage(clientID string) *WillMessage
	SaveWillMessage(willMessage *WillMessage) bool
	DeleteWillMessage(clientID string) bool
}

func NewWillMessage(clientID string, topic []byte, content []byte, qos byte, retained bool) *WillMessage {
	return &WillMessage{
		ClientID: clientID,
		Topic:    topic,
		QoS:      qos,
		Content:  content,
		Retained: retained,
	}
}
