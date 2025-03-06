package database

const (
	SessionCollectionName     = "sessions"
	WillMessageCollectionName = "will_messages"
	TopicCollectionName       = "topics"
)

type Topic struct {
	TopicName string `bson:"topic_name"`
}

type Subscription struct {
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
	GetSession(clientID string) (*SessionData, error)
	SaveSession(session *SessionData) error
	DeleteSession(clientID string) error
}

type WillMessageStore interface {
	GetWillMessage(clientID string) (*WillMessage, error)
	SaveWillMessage(willMessage *WillMessage) error
	DeleteWillMessage(clientID string) error
}

func NewTopic() *Topic {
	return &Topic{}
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
