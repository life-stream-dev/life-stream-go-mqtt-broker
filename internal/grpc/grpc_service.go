package __

import (
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/connection"
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/database"
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/packet"
	"golang.org/x/net/context"
)

type GRPCService struct{}

func (s *GRPCService) mustEmbedUnimplementedDeviceServer() {
	//TODO implement me
	panic("implement me")
}

func (*GRPCService) GetAllDevices(context.Context, *Empty) (*AllDevicesResponse, error) {
	databaseStore := database.NewDatabaseStore()
	sessions := databaseStore.GetAllSession()
	devices := make([]*Devices, len(sessions))
	i := 0
	for _, value := range sessions {
		devices[i] = &Devices{
			DevicesId: value.ClientID,
		}
		i++
	}
	response := &AllDevicesResponse{
		Devices: devices,
	}
	return response, nil
}

func (s *GRPCService) SetDevicesState(ctx context.Context, device *TargetDevice) (*ExecuteResponse, error) {
	publishPacket := &packet.PublishPacketPayloads{
		PacketFlag: packet.PublishPacketFlag{
			QoS: 0,
		},
		TopicName: packet.FieldPayload{
			PayloadLength: len("control/switch"),
			Payload:       []byte("control/switch"),
		},
	}
	if device.Status {
		publishPacket.Payload = []byte("ON")
	} else {
		publishPacket.Payload = []byte("OFF")
	}
	sender := connection.NewMessageSender()
	err := sender.SendMessage(device.DevicesId, packet.NewPublishPacket(publishPacket))
	if err != nil {
		return &ExecuteResponse{Status: false}, err
	}
	return &ExecuteResponse{Status: true}, nil
}
