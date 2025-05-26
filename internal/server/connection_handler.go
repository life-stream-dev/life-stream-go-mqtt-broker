// Package server 实现了MQTT服务器的连接处理功能
package server

import (
	. "github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/connection"
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/database"
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/logger"
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/mqtt"
	. "github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/packet"
	"net"
	"time"
)

// ConnectionHandler 处理MQTT客户端连接
type ConnectionHandler struct {
	conn          net.Conn              // 网络连接
	connId        string                // 连接ID
	keepAlive     time.Duration         // 心跳间隔
	clientSession *database.SessionData // 客户端会话数据
}

// handleFirstPacket 处理连接建立后的第一个报文（必须是CONNECT）
func (c *ConnectionHandler) handleFirstPacket() error {
	connManager := GetConnectionManager()

	// 设置读取超时
	_ = c.conn.SetReadDeadline(time.Now().Add(time.Minute))
	packet, err := mqtt.ReadPacket(c.conn)
	if err != nil {
		logger.WarnF("[%s] Fail to read first packet, details: %v", c.connId, err)
		return err
	}

	// 验证第一个报文必须是CONNECT
	if packet.Header.Type != mqtt.CONNECT {
		logger.ErrorF("[%s] Invalid first packet type, expected %s packet, but got %s packet", c.connId, mqtt.CONNECT.String(), packet.Header.Type.String())
		return err
	}

	// 解析CONNECT报文
	clientInfo, resp, err := ParseConnectPacket(packet)

	// 发送响应
	if resp != nil {
		if err := Send(c.conn, resp, c.connId); err != nil {
			return err
		}
	}

	if err != nil {
		logger.ErrorF("[%s] Fail to parse CONNECT packet, details: %v", c.connId, err)
		return err
	}

	logger.InfoF("First packet response %v", resp)

	// 处理CONNECT报文
	resp, c.clientSession, err = HandlerConnectPacket(clientInfo)

	// 发送响应
	if err := Send(c.conn, resp, c.connId); err != nil {
		return err
	}

	if err != nil {
		logger.ErrorF("[%s] Fail to handle CONNECT packet, details: %v", c.connId, err)
		return err
	}

	connManager.AddConnection(c.clientSession.ClientID, &Connection{
		Conn:   c.conn,
		ConnID: c.connId,
	})

	// 设置心跳间隔
	c.keepAlive = time.Duration(clientInfo.KeepAlive) * time.Second
	if c.keepAlive == 0 {
		logger.WarnF("[%s] Keep alive set to 0, heartbeat disable", c.connId)
		_ = c.conn.SetReadDeadline(time.Time{})
	}

	return nil
}

// handlePacket 处理后续的MQTT报文
func (c *ConnectionHandler) handlePacket() {
	for {
		// 设置读取超时
		if c.keepAlive != 0 {
			_ = c.conn.SetReadDeadline(time.Now().Add(c.keepAlive + 10*time.Second))
		}

		// 读取报文
		packet, err := mqtt.ReadPacket(c.conn)
		if err != nil {
			HandleReadError(c.connId, err)
			return
		}

		_ = c.conn.SetReadDeadline(time.Time{})

		logger.DebugF("[%s] Receive %s package, data %+v", c.connId, packet.Header.Type, packet.Payload)

		// 根据报文类型处理
		switch packet.Header.Type {
		case mqtt.CONNECT:
			logger.ErrorF("[%s] Duplicate CONNECT package", c.connId)
			return
		case mqtt.PUBLISH:
			result, err := ParsePublishPacket(packet)
			if err != nil {
				logger.ErrorF("[%s] Fail to handle publish packet, details: %v", c.connId, err)
				return
			}
			if result.Payload == nil {
				logger.WarnF("[%s] Receive a zero length payload packet, ", c.connId)
				break
			}
			HandlePublishPacket(result, c.clientSession)
		case mqtt.SUBSCRIBE:
			result, err := ParseSubscribePacket(packet)
			if err != nil {
				logger.ErrorF("[%s] Fail to handle subscribe packet, details: %v", c.connId, err)
				return
			}
			resp := HandleSubscribePacket(result, c.clientSession)
			err = Send(c.conn, resp, c.connId)
			if err != nil {
				logger.ErrorF("[%s] Fail to send subscribe ack packet, details: %v", c.connId, err)
				return
			}
		case mqtt.UNSUBSCRIBE:
			result, err := ParseUnSubscribePacket(packet)
			if err != nil {
				logger.ErrorF("[%s] Fail to handle unsubscribe packet, details: %v", c.connId, err)
				return
			}
			resp, err := HandleUnSubscribePacket(result, c.clientSession)
			if err != nil {
				logger.ErrorF("[%s] Fail to handle unsubscribe packet, details: %v", c.connId, err)
				return
			}
			err = Send(c.conn, resp, c.connId)
			if err != nil {
				logger.ErrorF("[%s] Fail to send unsubscribe ack packet, details: %v", c.connId, err)
				return
			}
		case mqtt.PINGREQ:
			HandlePingReq(c.conn, c.connId)
		case mqtt.DISCONNECT:
			HandleDisconnectPacket(c.clientSession)
			logger.InfoF("[%s] Client disconnect", c.connId)
			return
		default:
			logger.WarnF("[%s] %s package has not been supported", c.connId, packet.Header.Type.String())
			return
		}
	}
}

// handleConnection 处理完整的连接生命周期
func (c *ConnectionHandler) handleConnection() {
	// 确保连接最终被关闭
	defer func() {
		GetConnectionManager().RemoveConnection(c.clientSession.ClientID)
		logger.DebugF("[%s] Connection closed", c.connId)
		if err := c.conn.Close(); err != nil && !IsNetClosedError(err) {
			logger.WarnF("[%s] Error occured while closing connection, details: %v", c.connId, err)
		}
	}()

	// 处理第一个报文
	if err := c.handleFirstPacket(); err != nil {
		return
	}

	// 处理后续报文
	c.handlePacket()
}
