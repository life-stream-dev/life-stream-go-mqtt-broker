package packet

import (
	"fmt"
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

func HandlePublishPacket(packet *mqtt.Packet) (*PublishPacketPayloads, error) {
	//payload := packet.Payload
	result := &PublishPacketPayloads{
		PacketFlag: PublishPacketFlag{
			RetryFlag: (packet.Header.Flags&0x08)>>3 == 1,
			QoS:       (packet.Header.Flags & 0x06) >> 1,
			Retain:    packet.Header.Flags&0x01 == 1,
		},
	}

	if result.PacketFlag.QoS == 0 && result.PacketFlag.RetryFlag {
		return result, fmt.Errorf("when QoS Level set to 0, retry flag must be set to 0 either")
	}

	if result.PacketFlag.QoS == 3 {
		return result, fmt.Errorf("the QoS Level must not set to 3")
	}

	payloadLength := packet.Header.RemainingLength

	topicName, err := readPacketPayload(packet.Payload)
	if err != nil {
		return result, fmt.Errorf("error occured when reading topic name, details: %v", err)
	}
	result.TopicName = topicName
	payloadLength -= 2 + topicName.PayloadLength

	if result.PacketFlag.QoS > 0 {
		packetId, err := readPacketPayload(packet.Payload)
		if err != nil {
			return result, fmt.Errorf("error occured when reading packet ID, details: %v", err)
		}
		result.PacketID = int(mqtt.ByteToUInt16(packetId.Payload))
		payloadLength -= 2
	}

	payload, err := readPacketBytes(packet.Payload, payloadLength)
	if err != nil {
		return result, fmt.Errorf("error occured when reading payload, details: %v", err)
	}
	result.Payload = payload

	logger.Debug(string(result.TopicName.Payload))
	logger.Debug(string(result.Payload))

	return result, nil
}
