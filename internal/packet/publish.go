package packet

import (
	"encoding/binary"
	"fmt"
	. "github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/connection"
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/database"
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/logger"
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/mqtt"
)

type PublishPacketFlag struct {
	RetryFlag bool
	QoS       byte
	Retain    bool
}

type PublishPacketPayloads struct {
	PacketFlag PublishPacketFlag
	TopicName  FieldPayload
	PacketID   int
	Payload    []byte
}

func NewPublishPacket(packetPayloads *PublishPacketPayloads) []byte {
	packet := make([]byte, 1)
	packet[0] += byte(mqtt.PUBLISH) << 4
	if packetPayloads.PacketFlag.QoS > 0 {
		packet[0] += packetPayloads.PacketFlag.QoS << 1
	}
	payload := make([]byte, 0)
	topicLength := packetPayloads.TopicName.PayloadLength
	payload = append(payload, mqtt.UInt16ToByte(uint16(topicLength))...)
	payload = append(payload, packetPayloads.TopicName.Payload...)
	if packetPayloads.PacketFlag.QoS > 0 {
		payload = append(payload, mqtt.UInt16ToByte(uint16(packetPayloads.PacketID))...)
	}
	payload = append(payload, packetPayloads.Payload...)
	remainLength := len(payload)
	packet = append(packet, mqtt.EncodeRemainingLength(remainLength)...)
	packet = append(packet, payload...)
	return packet
}

func ParsePublishPacket(packet *mqtt.Packet) (*PublishPacketPayloads, error) {
	result := &PublishPacketPayloads{
		PacketFlag: PublishPacketFlag{
			RetryFlag: (packet.Header.Flags&0x08)>>3 == 1,
			QoS:       (packet.Header.Flags & 0x06) >> 1,
			Retain:    packet.Header.Flags&0x01 == 1,
		},
	}
	payload := packet.Payload

	if result.PacketFlag.QoS == 0 && result.PacketFlag.RetryFlag {
		return result, fmt.Errorf("when QoS Level set to 0, retry flag must be set to 0 either")
	}

	if result.PacketFlag.QoS == 3 {
		return result, fmt.Errorf("the QoS Level must not set to 3")
	}

	topicName, err := readPacketPayload(packet.Payload)
	if err != nil {
		return result, fmt.Errorf("error occured when reading topic name, %v", err)
	}
	result.TopicName = topicName

	if result.PacketFlag.QoS > 0 {
		packetId, err := readPacketBytes(packet.Payload, 2)
		if err != nil {
			return result, fmt.Errorf("error occured when reading packet ID, %v", err)
		}
		result.PacketID = int(binary.BigEndian.Uint16(packetId))
	}

	if !payload.CheckRemainingLength() {
		result.Payload = nil
		return result, nil
	}

	content, err := readPacketResumeByte(packet.Payload)
	if err != nil {
		return result, fmt.Errorf("error occured when reading payload, %v", err)
	}
	result.Payload = content

	return result, nil
}

// HandlePublishPacket 处理发布消息
// payload: 发布消息的数据包
// session: 发布者的会话数据
// 返回值: 需要发送给发布者的响应数据包
func HandlePublishPacket(payload *PublishPacketPayloads, session *database.SessionData) []byte {
	// 获取数据库存储实例
	dbStore := database.NewDatabaseStore()

	// 获取主题名称
	topicName := string(payload.TopicName.Payload)

	// 根据QoS级别处理消息
	switch payload.PacketFlag.QoS {
	case 0:
		// QoS 0: 最多一次，不需要确认
		handleQoS0Publish(dbStore, topicName, payload)
		return nil

	case 1:
		// QoS 1: 至少一次，需要PUBACK确认
		handleQoS1Publish(dbStore, topicName, payload, session)
		return NewPubAckPacket(payload.PacketID)

	case 2:
		// QoS 2: 确保一次，需要完整的确认流程
		handleQoS2Publish(dbStore, topicName, payload, session)
		return NewPubRecPacket(payload.PacketID)

	default:
		logger.ErrorF("Invalid QoS level: %d", payload.PacketFlag.QoS)
		return nil
	}
}

// handleQoS0Publish 处理QoS 0的发布消息
func handleQoS0Publish(dbStore *database.DBStore, topicName string, payload *PublishPacketPayloads) {
	// 查找匹配的订阅者
	subscriptions, err := dbStore.MatchTopic(topicName)
	if err != nil {
		logger.ErrorF("Failed to match topic %s: %v", topicName, err)
		return
	}

	// 创建消息发送器
	sender := NewMessageSender()

	// 向所有订阅者发送消息
	for _, sub := range subscriptions {
		// 创建新的发布消息
		publishPacket := &PublishPacketPayloads{
			PacketFlag: PublishPacketFlag{
				QoS: sub.QoSLevel,
			},
			TopicName: FieldPayload{
				PayloadLength: len(topicName),
				Payload:       []byte(topicName),
			},
			Payload: payload.Payload,
		}

		// 发送消息给订阅者
		if err := sender.SendMessage(sub.ClientID, NewPublishPacket(publishPacket)); err != nil {
			logger.ErrorF("Failed to send message to client %s: %v", sub.ClientID, err)
		}
	}
}

// handleQoS1Publish 处理QoS 1的发布消息
func handleQoS1Publish(dbStore *database.DBStore, topicName string, payload *PublishPacketPayloads, session *database.SessionData) {
	// 保存消息到待确认队列
	session.PendingPublish[uint16(payload.PacketID)] = topicName
	session.Save()

	// 查找匹配的订阅者
	subscriptions, err := dbStore.MatchTopic(topicName)
	if err != nil {
		logger.ErrorF("Failed to match topic %s: %v", topicName, err)
		return
	}

	// 创建消息发送器
	sender := NewMessageSender()

	// 向所有订阅者发送消息
	for _, sub := range subscriptions {
		// 创建新的发布消息
		publishPacket := &PublishPacketPayloads{
			PacketFlag: PublishPacketFlag{
				QoS: sub.QoSLevel,
			},
			TopicName: FieldPayload{
				PayloadLength: len(topicName),
				Payload:       []byte(topicName),
			},
			Payload: payload.Payload,
		}

		// 为QoS 1/2消息分配新的PacketID
		if sub.QoSLevel > 0 {
			publishPacket.PacketID = int(database.NewPacketIDManager().NextID())
		}

		// 发送消息给订阅者
		if err := sender.SendMessage(sub.ClientID, NewPublishPacket(publishPacket)); err != nil {
			logger.ErrorF("Failed to send message to client %s: %v", sub.ClientID, err)
		}
	}
}

// handleQoS2Publish 处理QoS 2的发布消息
func handleQoS2Publish(dbStore *database.DBStore, topicName string, payload *PublishPacketPayloads, session *database.SessionData) {
	// 保存消息到待确认队列
	session.PendingPublish[uint16(payload.PacketID)] = topicName
	session.Save()

	// 查找匹配的订阅者
	subscriptions, err := dbStore.MatchTopic(topicName)
	if err != nil {
		logger.ErrorF("Failed to match topic %s: %v", topicName, err)
		return
	}

	// 创建消息发送器
	sender := NewMessageSender()

	// 向所有订阅者发送消息
	for _, sub := range subscriptions {
		// 创建新的发布消息
		publishPacket := &PublishPacketPayloads{
			PacketFlag: PublishPacketFlag{
				QoS: sub.QoSLevel,
			},
			TopicName: FieldPayload{
				PayloadLength: len(topicName),
				Payload:       []byte(topicName),
			},
			Payload: payload.Payload,
		}

		// 为QoS 1/2消息分配新的PacketID
		if sub.QoSLevel > 0 {
			publishPacket.PacketID = int(database.NewPacketIDManager().NextID())
		}

		// 发送消息给订阅者
		if err := sender.SendMessage(sub.ClientID, NewPublishPacket(publishPacket)); err != nil {
			logger.ErrorF("Failed to send message to client %s: %v", sub.ClientID, err)
		}
	}
}

// NewPubAckPacket 创建PUBACK响应包
func NewPubAckPacket(packetID int) []byte {
	packet := make([]byte, 1)
	packet[0] = byte(mqtt.PUBACK) << 4

	payload := make([]byte, 0)
	payload = append(payload, mqtt.UInt16ToByte(uint16(packetID))...)

	packet = append(packet, mqtt.EncodeRemainingLength(len(payload))...)
	packet = append(packet, payload...)
	return packet
}

// NewPubRecPacket 创建PUBREC响应包
func NewPubRecPacket(packetID int) []byte {
	packet := make([]byte, 1)
	packet[0] = byte(mqtt.PUBREC) << 4

	payload := make([]byte, 0)
	payload = append(payload, mqtt.UInt16ToByte(uint16(packetID))...)

	packet = append(packet, mqtt.EncodeRemainingLength(len(payload))...)
	packet = append(packet, payload...)
	return packet
}
