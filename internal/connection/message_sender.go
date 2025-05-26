// Package connection 实现了MQTT服务器的消息发送功能
package connection

import (
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/logger"
	"net"
)

// MessageSender 消息发送器接口
type MessageSender interface {
	SendMessage(clientID string, data []byte) error
}

// DefaultMessageSender 默认的消息发送器实现
type DefaultMessageSender struct{}

// NewMessageSender 创建新的消息发送器
func NewMessageSender() MessageSender {
	return &DefaultMessageSender{}
}

// SendMessage 发送消息到指定客户端
func (s *DefaultMessageSender) SendMessage(clientID string, data []byte) error {
	connManager := GetConnectionManager()
	conn, ok := connManager.GetConnection(clientID)
	if !ok {
		return nil
	}
	return Send(conn.Conn, data, conn.ConnID)
}

// Send 发送数据到客户端
func Send(conn net.Conn, data []byte, connID string) error {
	total := 0
	for total < len(data) {
		n, err := conn.Write(data[total:])
		if err != nil {
			logger.ErrorF("[%s] Fail to send data, details: %v", connID, err)
			return err
		}
		total += n
	}
	logger.DebugF("[%s] Send %d bytes to client, data %v", connID, total, data)
	return nil
}
