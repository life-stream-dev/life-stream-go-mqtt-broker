// Package connection 实现了MQTT服务器的连接管理功能
package connection

import (
	"errors"
	"io"
	"net"
	"os"
	"sync"

	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/logger"
)

// Connection 表示一个客户端连接
type Connection struct {
	Conn   net.Conn
	ConnID string
}

// ConnectionManager 连接管理器
type ConnectionManager struct {
	connections sync.Map
}

var (
	instance *ConnectionManager
	once     sync.Once
)

// GetConnectionManager 获取连接管理器实例
func GetConnectionManager() *ConnectionManager {
	once.Do(func() {
		instance = &ConnectionManager{}
	})
	return instance
}

// AddConnection 添加连接
func (cm *ConnectionManager) AddConnection(clientID string, conn *Connection) {
	cm.connections.Store(clientID, conn)
	logger.InfoF("Client %s connected", clientID)
}

// RemoveConnection 移除连接
func (cm *ConnectionManager) RemoveConnection(clientID string) {
	cm.connections.Delete(clientID)
	logger.InfoF("Client %s disconnected", clientID)
}

// GetConnection 获取连接
func (cm *ConnectionManager) GetConnection(clientID string) (*Connection, bool) {
	if value, ok := cm.connections.Load(clientID); ok {
		return value.(*Connection), true
	}
	return nil, false
}

func IsNetClosedError(err error) bool {
	if errors.Is(err, net.ErrClosed) {
		return true
	}
	var opErr *net.OpError
	ok := errors.As(err, &opErr)
	return ok && opErr.Timeout()
}

func HandleReadError(connID string, err error) {
	switch {
	case errors.Is(err, io.EOF):
		logger.InfoF("[%s] Client close connection", connID)
	case os.IsTimeout(err):
		logger.WarnF("[%s] Reading timeout", connID)
	default:
		logger.ErrorF("[%s] Error occured while reading packet, details: %v", connID, err)
	}
}
