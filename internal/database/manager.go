package database

type SessionData struct {
	ClientID       string
	Subscriptions  map[string]byte     // 主题: QoS
	PendingPublish map[uint16]string   // 未确认的 QoS 1/2 消息（PacketID -> Message）
	PendingPubrel  map[uint16]struct{} // 等待 PUBREL 的 QoS 2 消息
	InflightQoS2   map[uint16]string   // 已发送但未完成的 QoS 2 消息
}

type SessionStore interface {
	Get(clientID string) (*SessionData, error)
	Save(clientID string, session *SessionData) error
	Delete(clientID string) error
}

func NewSessionData(clientID string) *SessionData {
	return &SessionData{
		ClientID:       clientID,
		Subscriptions:  make(map[string]byte),
		PendingPublish: make(map[uint16]string),
		PendingPubrel:  make(map[uint16]struct{}),
	}
}
