package packet

// 控制包类型 CONNECT 相关函数

import (
	"errors"
	. "fmt"
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/logger"
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/mqtt"
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

// HandlerConnectPacket 处理 CONNECT 控制包的可变头和负载
func HandlerConnectPacket(payload []byte) (ConnectPacketPayloads, error) {
	result := ConnectPacketPayloads{}
	currentPtr := 0

	// 协议名长度检查
	if currentPtr+2 > len(payload) {
		return result, errors.New("insufficient bytes for protocol length")
	}
	protocolLength := int(mqtt.ByteToUInt16(payload[currentPtr : currentPtr+2]))
	currentPtr += 2
	logger.Debug("协议字段长度：%d", protocolLength)

	// 协议名检查
	if currentPtr+protocolLength > len(payload) {
		return result, errors.New("protocol name exceeds payload")
	}
	protocolString := string(payload[currentPtr : currentPtr+protocolLength])
	currentPtr += protocolLength
	logger.Debug("协议：%s", protocolString)

	// 协议版本
	if currentPtr >= len(payload) {
		return result, errors.New("insufficient bytes for protocol version")
	}
	protocolVersion := int(payload[currentPtr])
	currentPtr++
	logger.Debug("协议版本：%d", protocolVersion)

	// 连接标志位
	if currentPtr >= len(payload) {
		return result, errors.New("insufficient bytes for connect flags")
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

	logger.Debug("连接标志位：%+v", result.ConnectFlag)

	// Client ID
	clientID, err := readConnectPacketPayload(payload, currentPtr)
	if err != nil {
		return result, Errorf("client ID: %w", err)
	}
	result.ClientIdentifier = clientID
	currentPtr += 2 + clientID.PayloadLength

	// Will Message
	if result.ConnectFlag.WillMessageFlag {
		willTopic, err := readConnectPacketPayload(payload, currentPtr)
		if err != nil {
			return result, Errorf("will topic: %w", err)
		}
		result.WillMessageTopic = willTopic
		currentPtr += 2 + willTopic.PayloadLength

		willContent, err := readConnectPacketPayload(payload, currentPtr)
		if err != nil {
			return result, Errorf("will content: %w", err)
		}
		result.WillMessageContent = willContent
		currentPtr += 2 + willContent.PayloadLength
	}

	// Username
	if result.ConnectFlag.UsernameFlag {
		username, err := readConnectPacketPayload(payload, currentPtr)
		if err != nil {
			return result, Errorf("username: %w", err)
		}
		result.UsernamePayload = username
		currentPtr += 2 + username.PayloadLength
	}

	// Password
	if result.ConnectFlag.PasswordFlag {
		password, err := readConnectPacketPayload(payload, currentPtr)
		if err != nil {
			return result, Errorf("password: %w", err)
		}
		result.PasswordPayload = password
		currentPtr += 2 + password.PayloadLength
	}

	return result, nil
}
