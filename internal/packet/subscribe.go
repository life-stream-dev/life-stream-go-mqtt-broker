package packet

import (
	"encoding/binary"
	"fmt"
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/database"
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/mqtt"
)

type SubscribeState byte

const (
	SuccessQos0 SubscribeState = iota
	SuccessQos1
	SuccessQos2
	Failure SubscribeState = 0x80
)

type SubscribePacketPayloads struct {
	PacketID      int
	Subscriptions []*database.Subscription
}

func NewSubAckPacket(packetId int, state SubscribeState) []byte {
	packet := make([]byte, 1)
	packet[0] += byte(mqtt.SUBACK) << 4

	payload := make([]byte, 0)
	payload = append(payload, mqtt.UInt16ToByte(uint16(packetId))...)
	payload = append(payload, byte(state))

	packet = append(packet, mqtt.EncodeRemainingLength(len(payload))...)
	packet = append(packet, payload...)
	return packet
}

func ParseSubscribePacket(packet *mqtt.Packet) (*SubscribePacketPayloads, error) {
	result := &SubscribePacketPayloads{
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
		qos, err := readPacketByte(packet.Payload)
		if err != nil {
			return result, fmt.Errorf("error occured when reading qos level, details: %v", err)
		}
		subscript.TopicName = string(topicFilter.Payload)
		subscript.QoSLevel = qos & 0x02
		result.Subscriptions = append(result.Subscriptions, subscript)
	}

	return result, nil
}

func HandleSubscribePacket(payload *SubscribePacketPayloads) ([]byte, error) {
	return NewSubAckPacket(payload.PacketID, SuccessQos0), nil
}
