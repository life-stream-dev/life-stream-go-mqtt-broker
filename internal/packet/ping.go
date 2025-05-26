package packet

import (
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/connection"
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/logger"
	"net"
)

func NewPingRespPacket() []byte {
	return []byte{0xD0, 0x00}
}

func HandlePingReq(conn net.Conn, connID string) {
	resp := NewPingRespPacket()
	if err := connection.Send(conn, resp, connID); err != nil {
		logger.WarnF("[%s] Fail to send PINGRESP packet, details: %v", connID, err)
	}
}
