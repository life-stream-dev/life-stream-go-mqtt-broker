package packet

// 控制包类型 CONNECT 相关函数

import (
	"errors"
	"fmt"
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/database"
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
	if sessionStatus {
		return []byte{0x20, 0x02, 0x01, byte(returnCode)}
	}
	return []byte{0x20, 0x02, 0x00, byte(returnCode)}
}

// HandleConnectPacket 处理 CONNECT 控制包的可变头和负载
func HandleConnectPacket(packet *mqtt.Packet) (ConnectPacketPayloads, []byte, error) {
	payload := packet.Payload
	result := ConnectPacketPayloads{}

	protocolString, err := readPacketPayload(payload)
	if err != nil {
		return result, nil, errors.New("unable to check protocol string")
	}
	if string(protocolString.Payload) != "MQTT" {
		return result, nil, fmt.Errorf("incorrect Protocol String: %s", string(protocolString.Payload))
	}

	// 协议版本
	if !payload.CheckRemainingLength() {
		return result, nil, errors.New("insufficient bytes for protocol version")
	}
	protocolVersion, err := readPacketByte(payload)
	if err != nil {
		return result, nil, fmt.Errorf("unable to read protocol version, details: %v", err)
	}
	if protocolVersion != 0x04 {
		return result, NewConnectAckPacket(false, UnacceptableProtocol), errors.New("protocol version does not match")
	}

	// 连接标志位
	if !payload.CheckRemainingLength() {
		return result, nil, errors.New("insufficient bytes for connect flags")
	}
	connectFlag, err := readPacketByte(payload)
	if err != nil {
		return result, nil, fmt.Errorf("unable to read connect flag, details: %v", err)
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
		return result, nil, errors.New("when will message flag is not set, remain flag must not be set and QoSLevel must be 0")
	}

	// Keep Alive Time
	data, err := readPacketBytes(payload, 2)
	if err != nil {
		return result, nil, errors.New("unable to read keep alive time")
	}
	keepAlive := int(mqtt.ByteToUInt16(data))
	result.KeepAlive = keepAlive

	// Client ID
	clientID, err := readPacketPayload(payload)
	if err != nil {
		return result, nil, fmt.Errorf("client ID: %w", err)
	}
	result.ClientIdentifier = clientID

	// Will Message
	if result.ConnectFlag.WillMessageFlag {
		willTopic, err := readPacketPayload(payload)
		if err != nil {
			return result, nil, fmt.Errorf("will topic: %w", err)
		}
		result.WillMessageTopic = willTopic

		willContent, err := readPacketPayload(payload)
		if err != nil {
			return result, nil, fmt.Errorf("will content: %w", err)
		}
		result.WillMessageContent = willContent
	}

	// Username
	if result.ConnectFlag.UsernameFlag {
		username, err := readPacketPayload(payload)
		if err != nil {
			return result, nil, fmt.Errorf("username: %w", err)
		}
		result.UsernamePayload = username
	}

	// Password
	if result.ConnectFlag.PasswordFlag {
		password, err := readPacketPayload(payload)
		if err != nil {
			return result, nil, fmt.Errorf("password: %w", err)
		}
		result.PasswordPayload = password
	}

	if result.ConnectFlag.CleanSession {
		clientID := string(result.ClientIdentifier.Payload)
		err := database.NewDatabaseStore().DeleteSession(clientID)
		if err != nil {
			return result, nil, errors.New("unable to delete session")
		}
		session := database.NewSessionData(clientID)
		err = database.NewMemoryStore().SaveSession(session)
		if err != nil {
			return result, nil, errors.New("unable to save session")
		}
		if result.ConnectFlag.WillMessageFlag {
			willMessage := database.NewWillMessage(
				clientID,
				result.WillMessageTopic.Payload,
				result.WillMessageContent.Payload,
				result.ConnectFlag.QoSLevel,
				result.ConnectFlag.RemainFlag,
			)
			err := database.NewMemoryStore().SaveWillMessage(willMessage)
			if err != nil {
				return result, nil, errors.New("unable to save will message")
			}
		}
	} else {
		clientID := string(result.ClientIdentifier.Payload)
		_, err := database.NewDatabaseStore().GetSession(clientID)
		if err != nil {
			return result, nil, errors.New("unable to get session")
		}

	}

	return result, NewConnectAckPacket(false, Accepted), nil
}
