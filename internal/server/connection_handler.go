package server

import (
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/logger"
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/mqtt"
	pa "github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/packet"
	"net"
	"time"
)

type ConnectionHandler struct {
	conn      net.Conn
	connId    string
	keepAlive time.Duration
}

func (c *ConnectionHandler) handleFirstPacket() error {
	_ = c.conn.SetReadDeadline(time.Now().Add(time.Minute))
	packet, err := mqtt.ReadPacket(c.conn)
	if err != nil {
		logger.WarnF("[%s] Fail to read first packet, details: %v", c.connId, err)
		return err
	}

	if packet.Header.Type != mqtt.CONNECT {
		logger.ErrorF("[%s] Invalid first packet type, expected %s packet, but got %s packet", c.connId, mqtt.CONNECT.String(), packet.Header.Type.String())
		return err
	}

	clientInfo, resp, err := pa.ParseConnectPacket(packet)

	if resp != nil {
		if err := send(c.conn, resp, c.connId); err != nil {
			return err
		}
	}

	if err != nil {
		logger.ErrorF("[%s] Fail to parse CONNECT packet, details: %v", c.connId, err)
		return err
	}

	logger.InfoF("First packet response %v", resp)

	resp, err = pa.HandlerConnectPacket(clientInfo)

	if err := send(c.conn, resp, c.connId); err != nil {
		return err
	}

	if err != nil {
		logger.ErrorF("[%s] Fail to handle CONNECT packet, details: %v", c.connId, err)
		return err
	}

	c.keepAlive = time.Duration(clientInfo.KeepAlive) * time.Second
	if c.keepAlive == 0 {
		logger.WarnF("[%s] Keep alive set to 0, heartbeat disable", c.connId)
		_ = c.conn.SetReadDeadline(time.Time{})
	}

	return nil
}

func (c *ConnectionHandler) handlePacket() {
	for {
		if c.keepAlive != 0 {
			_ = c.conn.SetReadDeadline(time.Now().Add(c.keepAlive + 10*time.Second))
		}

		packet, err := mqtt.ReadPacket(c.conn)
		if err != nil {
			handleReadError(c.connId, err)
			return
		}

		_ = c.conn.SetReadDeadline(time.Time{})

		logger.DebugF("[%s] Receive %s package, data %+v", c.connId, packet.Header.Type, packet.Payload)

		switch packet.Header.Type {
		case mqtt.CONNECT:
			logger.ErrorF("[%s] Duplicate CONNECT package", c.connId)
			return
		case mqtt.PUBLISH:
			_, err := pa.ParsePublishPacket(packet)
			if err != nil {
				logger.ErrorF("[%s] Fail to handle publish packet, details: %v", c.connId, err)
				return
			}
		case mqtt.SUBSCRIBE:
			result, err := pa.ParseSubscribePacket(packet)
			if err != nil {
				logger.ErrorF("[%s] Fail to handle subscribe packet, details: %v", c.connId, err)
				return
			}
			resp, err := pa.HandleSubscribePacket(result)
			if err != nil {
				logger.ErrorF("[%s] Fail to handle subscribe packet, details: %v", c.connId, err)
				return
			}
			err = send(c.conn, resp, c.connId)
			if err != nil {
				logger.ErrorF("[%s] Fail to send subscribe ack packet, details: %v", c.connId, err)
				return
			}
		case mqtt.UNSUBSCRIBE:
			result, err := pa.ParseUnSubscribePacket(packet)
			if err != nil {
				logger.ErrorF("[%s] Fail to handle unsubscribe packet, details: %v", c.connId, err)
				return
			}
			resp, err := pa.HandleUnSubscribePacket(result)
			if err != nil {
				logger.ErrorF("[%s] Fail to handle unsubscribe packet, details: %v", c.connId, err)
				return
			}
			err = send(c.conn, resp, c.connId)
			if err != nil {
				logger.ErrorF("[%s] Fail to send unsubscribe ack packet, details: %v", c.connId, err)
				return
			}
		case mqtt.PINGREQ:
			handlePingReq(c.conn, c.connId)
		case mqtt.DISCONNECT:
			logger.InfoF("[%s] Client disconnect", c.connId)
			return
		default:
			logger.WarnF("[%s] %s package has not been supported", c.connId, packet.Header.Type.String())
			return
		}
	}
}

func (c *ConnectionHandler) handleConnection() {
	defer func() {
		logger.DebugF("[%s] Connection closed", c.connId)
		if err := c.conn.Close(); err != nil && !isNetClosedError(err) {
			logger.WarnF("[%s] Error occured while closing connection, details: %v", c.connId, err)
		}
	}()

	if err := c.handleFirstPacket(); err != nil {
		return
	}

	c.handlePacket()
}
