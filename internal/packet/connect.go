package packet

// 控制包类型 CONNECT 相关函数

import (
	"errors"
	. "fmt"
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

type ConnectPacketPayload struct {
	PayloadLength int
	Payload       []byte
}

type ConnectPacketPayloads struct {
	ConnectFlag        ConnectPacketFlag
	ClientIdentifier   ConnectPacketPayload
	UsernamePayload    ConnectPacketPayload
	PasswordPayload    ConnectPacketPayload
	WillMessageTopic   ConnectPacketPayload
	WillMessageContent ConnectPacketPayload
	KeepAlive          int
}

func readConnectPacketPayload(payload []byte, startByte int) (ConnectPacketPayload, error) {
	if startByte+1 >= len(payload) {
		return ConnectPacketPayload{}, errors.New("insufficient bytes for length")
	}
	length := int(mqtt.ByteToUInt16(payload[startByte : startByte+2]))
	end := startByte + 2 + length
	if end > len(payload) {
		return ConnectPacketPayload{}, Errorf("payload length %d exceeds buffer (len=%d)", length, len(payload))
	}

	return ConnectPacketPayload{
		PayloadLength: length,
		Payload:       payload[startByte+2 : end],
	}, nil
}

func NewConnectAckPacket(sessionStatus bool, returnCode ConnectRespType) []byte {
	if sessionStatus {
		return []byte{0x20, 0x02, 0x01, byte(returnCode)}
	}
	return []byte{0x20, 0x02, 0x00, byte(returnCode)}
}

// HandleConnectPacket 处理 CONNECT 控制包的可变头和负载
func HandleConnectPacket(payload []byte) (ConnectPacketPayloads, []byte, error) {
	result := ConnectPacketPayloads{}
	currentPtr := 0

	protocolString, err := readConnectPacketPayload(payload, currentPtr)
	if err != nil {
		return result, nil, errors.New("unable to check protocol string")
	}
	if string(protocolString.Payload) != "MQTT" {
		return result, nil, Errorf("incorrect Protocol String: %s", string(protocolString.Payload))
	}
	currentPtr += 2 + protocolString.PayloadLength

	// 协议版本
	if currentPtr >= len(payload) {
		return result, nil, errors.New("insufficient bytes for protocol version")
	}
	protocolVersion := int(payload[currentPtr])
	currentPtr++
	if protocolVersion != 0x04 {
		return result, NewConnectAckPacket(false, UnacceptableProtocol), errors.New("protocol version does not match")
	}

	// 连接标志位
	if currentPtr >= len(payload) {
		return result, nil, errors.New("insufficient bytes for connect flags")
	}
	connectFlag := payload[currentPtr]
	currentPtr++

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
	keepAlive := int(mqtt.ByteToUInt16(payload[currentPtr : currentPtr+2]))
	result.KeepAlive = keepAlive
	currentPtr += 2

	// Client ID
	clientID, err := readConnectPacketPayload(payload, currentPtr)
	if err != nil {
		return result, nil, Errorf("client ID: %w", err)
	}
	result.ClientIdentifier = clientID
	currentPtr += 2 + clientID.PayloadLength

	// Will Message
	if result.ConnectFlag.WillMessageFlag {
		willTopic, err := readConnectPacketPayload(payload, currentPtr)
		if err != nil {
			return result, nil, Errorf("will topic: %w", err)
		}
		result.WillMessageTopic = willTopic
		currentPtr += 2 + willTopic.PayloadLength

		willContent, err := readConnectPacketPayload(payload, currentPtr)
		if err != nil {
			return result, nil, Errorf("will content: %w", err)
		}
		result.WillMessageContent = willContent
		currentPtr += 2 + willContent.PayloadLength
	}

	// Username
	if result.ConnectFlag.UsernameFlag {
		username, err := readConnectPacketPayload(payload, currentPtr)
		if err != nil {
			return result, nil, Errorf("username: %w", err)
		}
		result.UsernamePayload = username
		currentPtr += 2 + username.PayloadLength
	}

	// Password
	if result.ConnectFlag.PasswordFlag {
		password, err := readConnectPacketPayload(payload, currentPtr)
		if err != nil {
			return result, nil, Errorf("password: %w", err)
		}
		result.PasswordPayload = password
		currentPtr += 2 + password.PayloadLength
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
