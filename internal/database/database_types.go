package database

const (
	SessionCollectionName     = "sessions"
	WillMessageCollectionName = "will_messages"
	TopicCollectionName       = "topics"
)

type SessionData struct {
	ClientID       string              `bson:"client_id"`
	Subscriptions  map[string]byte     `bson:"subscriptions"`   // 主题: QoS
	PendingPublish map[uint16]string   `bson:"pending_publish"` // 未确认的 QoS 1/2 消息（PacketID -> Message）
	PendingPubrel  map[uint16]struct{} `bson:"pending_pubrel"`  // 等待 PUBREL 的 QoS 2 消息
	InflightQoS2   map[uint16]string   `bson:"inflight_qos2"`   // 已发送但未完成的 QoS 2 消息
}

type Topic struct {
	TopicName string `bson:"topic_name"`
}

type WillMessage struct {
	ClientID string `bson:"client_id"`
	Topic    string `bson:"topic"`
	QoS      byte   `bson:"qo_s"`
	Content  []byte `bson:"content"`
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

func NewWillMessage() *WillMessage {
	return &WillMessage{}
}

func NewSessionData(clientID string) *SessionData {
	return &SessionData{
		ClientID:       clientID,
		Subscriptions:  make(map[string]byte),
		PendingPublish: make(map[uint16]string),
		PendingPubrel:  make(map[uint16]struct{}),
	}
}
