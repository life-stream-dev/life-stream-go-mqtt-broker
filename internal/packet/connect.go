package packet

// 控制包类型 CONNECT 相关函数

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/database"
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/logger"
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/mqtt"
)

type ConnectRespType byte

const (
	Accepted ConnectRespType = iota
	UnacceptableProtocol
	IdentifierRejected
	ServerUnavailable
	AuthenticationFailed
	NotAuthorized
)

// ConnectPacketFlag CONNECT控制包连接标志位
type ConnectPacketFlag struct {
	UsernameFlag    bool
	PasswordFlag    bool
	RemainFlag      bool
	QoSLevel        byte
	WillMessageFlag bool
	CleanSession    bool
}

type ConnectPacketPayloads struct {
	ConnectFlag        ConnectPacketFlag
	ClientIdentifier   FieldPayload
	UsernamePayload    FieldPayload
	PasswordPayload    FieldPayload
	WillMessageTopic   FieldPayload
	WillMessageContent FieldPayload
	KeepAlive          int
}

func NewConnectAckPacket(sessionStatus bool, returnCode ConnectRespType) []byte {
	if returnCode != Accepted {
		return []byte{0x20, 0x02, 0x00, byte(returnCode)}
	}
	if sessionStatus {
		return []byte{0x20, 0x02, 0x01, byte(Accepted)}
	}
	return []byte{0x20, 0x02, 0x00, byte(Accepted)}
}

// ParseConnectPacket 处理 CONNECT 控制包的可变头和负载
func ParseConnectPacket(packet *mqtt.Packet) (*ConnectPacketPayloads, []byte, error) {
	payload := packet.Payload
	result := ConnectPacketPayloads{}

	protocolString, err := readPacketPayload(payload)
	if err != nil {
		return &result, nil, errors.New("unable to check protocol string")
	}
	if string(protocolString.Payload) != "MQTT" {
		return &result, nil, fmt.Errorf("incorrect Protocol String: %s", string(protocolString.Payload))
	}

	// 协议版本
	if !payload.CheckRemainingLength() {
		return &result, nil, errors.New("insufficient bytes for protocol version")
	}
	protocolVersion, err := readPacketByte(payload)
	if err != nil {
		return &result, nil, fmt.Errorf("unable to read protocol version, details: %v", err)
	}
	if protocolVersion != 0x04 {
		return &result, NewConnectAckPacket(false, UnacceptableProtocol), errors.New("protocol version does not match")
	}

	// 连接标志位
	if !payload.CheckRemainingLength() {
		return &result, nil, errors.New("insufficient bytes for connect flags")
	}
	connectFlag, err := readPacketByte(payload)
	if err != nil {
		return &result, nil, fmt.Errorf("unable to read connect flag, details: %v", err)
	}

	// 解析标志位
	result.ConnectFlag = ConnectPacketFlag{
		UsernameFlag:    (connectFlag&0x80)>>7 == 1,
		PasswordFlag:    (connectFlag&0x40)>>6 == 1,
		RemainFlag:      (connectFlag&0x20)>>5 == 1,
		QoSLevel:        (connectFlag & 0x18) >> 3, // 0x18 = 00011000
		WillMessageFlag: (connectFlag&0x04)>>2 == 1,
		CleanSession:    (connectFlag&0x02)>>1 == 1,
	}

	if !result.ConnectFlag.WillMessageFlag && (result.ConnectFlag.RemainFlag || result.ConnectFlag.QoSLevel != 0) {
		return &result, nil, errors.New("when will message flag is not set, remain flag must not be set and QoSLevel must be 0")
	}

	// Keep Alive Time
	data, err := readPacketBytes(payload, 2)
	if err != nil {
		return &result, nil, errors.New("unable to read keep alive time")
	}
	keepAlive := int(binary.BigEndian.Uint16(data))
	result.KeepAlive = keepAlive

	// Client ID
	clientID, err := readPacketPayload(payload)
	if err != nil {
		return &result, nil, fmt.Errorf("client ID: %w", err)
	}
	if clientID.PayloadLength == 0 {
		return &result, NewConnectAckPacket(false, IdentifierRejected), errors.New("client ID is empty")
	}
	result.ClientIdentifier = clientID

	// Will Message
	if result.ConnectFlag.WillMessageFlag {
		willTopic, err := readPacketPayload(payload)
		if err != nil {
			return &result, nil, fmt.Errorf("will topic: %w", err)
		}
		result.WillMessageTopic = willTopic

		willContent, err := readPacketPayload(payload)
		if err != nil {
			return &result, nil, fmt.Errorf("will content: %w", err)
		}
		result.WillMessageContent = willContent
	}

	// Username
	if result.ConnectFlag.UsernameFlag {
		username, err := readPacketPayload(payload)
		if err != nil {
			return &result, nil, fmt.Errorf("username: %w", err)
		}
		result.UsernamePayload = username
	}

	// Password
	if result.ConnectFlag.PasswordFlag {
		password, err := readPacketPayload(payload)
		if err != nil {
			return &result, nil, fmt.Errorf("password: %w", err)
		}
		result.PasswordPayload = password
	}

	return &result, nil, nil
}

func HandlerConnectPacket(payloads *ConnectPacketPayloads) ([]byte, *database.SessionData, error) {
	databaseStore := database.NewDatabaseStore()
	memoryStore := database.NewMemoryStore()
	clientId := string(payloads.ClientIdentifier.Payload)
	if payloads.ConnectFlag.CleanSession {
		if !databaseStore.DeleteSession(clientId) {
			return NewConnectAckPacket(false, ServerUnavailable), nil, fmt.Errorf("unable to delete session during clean session")
		}
		session := database.NewSessionData(clientId)
		session.TempSession = true
		if !memoryStore.SaveSession(session) {
			return NewConnectAckPacket(false, ServerUnavailable), nil, fmt.Errorf("unable to save session")
		}
		return NewConnectAckPacket(false, Accepted), session, nil
	}
	session := databaseStore.GetSession(clientId)
	if session == nil {
		logger.ErrorF("unable to get session from database")
		session := database.NewSessionData(clientId)
		if !databaseStore.SaveSession(session) {
			return NewConnectAckPacket(false, ServerUnavailable), nil, fmt.Errorf("unable to save session")
		}
		return NewConnectAckPacket(false, Accepted), session, nil
	}
	logger.InfoF("[%s]Session has been found in database", session.ClientID)
	return NewConnectAckPacket(true, Accepted), session, nil
}
