package packet

import (
	"encoding/binary"
	"fmt"
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/database"
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/mqtt"
)

type UnSubscribePacketPayloads struct {
	PacketID      int
	Subscriptions []*database.Subscription
}

func NewUnSubAckPacket(packetId int, state SubscribeState) []byte {
	packet := make([]byte, 1)
	packet[0] += byte(mqtt.UNSUBACK) << 4

	payload := make([]byte, 0)
	payload = append(payload, mqtt.UInt16ToByte(uint16(packetId))...)
	payload = append(payload, byte(state))

	packet = append(packet, mqtt.EncodeRemainingLength(len(payload))...)
	packet = append(packet, payload...)
	return packet
}

func ParseUnSubscribePacket(packet *mqtt.Packet) (*UnSubscribePacketPayloads, error) {
	result := &UnSubscribePacketPayloads{
		PacketID:      -1,
		Subscriptions: make([]*database.Subscription, 0),
	}

	packetId, err := readPacketBytes(packet.Payload, 2)
	if err != nil {
		return result, fmt.Errorf("error occured when reading packet ID, details: %v", err)
	}
	result.PacketID = int(binary.BigEndian.Uint16(packetId))

	for packet.Payload.CurrentPtr != packet.Payload.ContextLen {
		subscript := &database.Subscription{}
		topicFilter, err := readPacketPayload(packet.Payload)
		if err != nil {
			return result, fmt.Errorf("error occured when reading topic filter, details: %v", err)
		}
		subscript.TopicName = string(topicFilter.Payload)
		result.Subscriptions = append(result.Subscriptions, subscript)
	}

	return result, nil
}

func HandleUnSubscribePacket(payload *UnSubscribePacketPayloads, session *database.SessionData) ([]byte, error) {
	for _, subscription := range payload.Subscriptions {
		session.RemoveSubscription(subscription)
	}
	return NewUnSubAckPacket(payload.PacketID, SuccessQos0), nil
}
