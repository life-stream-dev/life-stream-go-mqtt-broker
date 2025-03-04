package packet

import (
	"errors"
	"fmt"
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/mqtt"
)

type FieldPayload struct {
	PayloadLength int
	Payload       []byte
}

func readPacketByte(payload *mqtt.Payload) (byte, error) {
	startByte := payload.CurrentPtr
	if startByte >= payload.ContextLen {
		return 0, errors.New("invalid packet context length")
	}
	payload.CurrentPtr++
	return payload.Context[startByte], nil
}

func readPacketBytes(payload *mqtt.Payload, length int) ([]byte, error) {
	if length <= 0 {
		return nil, errors.New("invalid reading length, except > 0")
	}
	if length == 1 {
		bytes, err := readPacketByte(payload)
		return []byte{bytes}, err
	}
	startByte := payload.CurrentPtr
	contextLen := payload.ContextLen
	if startByte >= contextLen {
		return nil, errors.New("invalid packet context length")
	}
	end := startByte + length
	if end > contextLen {
		return nil, errors.New("invalid packet context length")
	}
	data := payload.Context[startByte:end]
	payload.CurrentPtr = end
	return data, nil
}

func readPacketPayload(payload *mqtt.Payload) (FieldPayload, error) {
	startByte := payload.CurrentPtr
	contextLen := payload.ContextLen
	if startByte+1 >= contextLen {
		return FieldPayload{}, errors.New("insufficient bytes for length")
	}
	length := int(mqtt.ByteToUInt16(payload.Context[startByte : startByte+2]))
	end := startByte + 2 + length
	if end > contextLen {
		return FieldPayload{}, fmt.Errorf("payload length %d exceeds buffer (len=%d)", length, contextLen)
	}
	payload.CurrentPtr += 2 + length
	return FieldPayload{
		PayloadLength: length,
		Payload:       payload.Context[startByte+2 : end],
	}, nil
}
