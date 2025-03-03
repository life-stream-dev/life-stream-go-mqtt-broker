package server

import (
	"errors"
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/logger"
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/mqtt"
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/packet"
	"io"
	"net"
	"os"
	"strconv"
	"time"
)

var sem = make(chan struct{}, 10000)

func send(conn net.Conn, data []byte, connID string) error {
	total := 0
	for total < len(data) {
		n, err := conn.Write(data[total:])
		if err != nil {
			logger.ErrorF("[%s] Fail to send data, details: %v", connID, err)
			return err
		}
		total += n
	}
	logger.DebugF("[%s] Send %d bytes to client", connID, total)
	return nil
}

func handlePingReq(conn net.Conn, connID string) {
	resp := packet.NewPingRespPacket()
	if err := send(conn, resp, connID); err != nil {
		logger.WarnF("[%s] Fail to send PINGRESP packet, details: %v", connID, err)
	}
}

func isNetClosedError(err error) bool {
	if errors.Is(err, net.ErrClosed) {
		return true
	}
	var opErr *net.OpError
	ok := errors.As(err, &opErr)
	return ok && opErr.Timeout()
}

func handleReadError(connID string, err error) {
	switch {
	case errors.Is(err, io.EOF):
		logger.InfoF("[%s] Client close connection", connID)
	case os.IsTimeout(err):
		logger.WarnF("[%s] Reading timeout", connID)
	default:
		logger.ErrorF("[%s] Error occured while reading packet, details: %v", connID, err)
	}
}

func handleConnection(conn net.Conn) {
	connID := conn.RemoteAddr().String()

	defer func() {
		logger.DebugF("[%s] Connection closed", connID)
		if err := conn.Close(); err != nil && !isNetClosedError(err) {
			logger.WarnF("[%s] Error occured while closing connection, details: %v", connID, err)
		}
	}()

	_ = conn.SetReadDeadline(time.Now().Add(time.Minute))
	header, payload, err := mqtt.ReadPacket(conn)
	if err != nil {
		logger.WarnF("[%s] Fail to read first packet, details: %v", connID, err)
		return
	}

	if header.Type != mqtt.CONNECT {
		logger.ErrorF("[%s] Invalid first packet type, expected %s packet, but got %s packet", connID, mqtt.CONNECT.String(), header.Type.String())
		return
	}

	clientInfo, resp, err := packet.HandleConnectPacket(payload)
	if err != nil {
		logger.ErrorF("[%s] Fail to parse CONNECT packet, details: %v", connID, err)
		return
	}
	logger.InfoF("First packet response %v", resp)

	if err := send(conn, resp, connID); err != nil {
		return
	}

	keepAlive := time.Duration(clientInfo.KeepAlive) * time.Second
	if keepAlive == 0 {
		keepAlive = 5 * time.Minute
	}

	for {
		_ = conn.SetReadDeadline(time.Now().Add(keepAlive + 10*time.Second))

		header, payload, err := mqtt.ReadPacket(conn)
		if err != nil {
			handleReadError(connID, err)
			return
		}

		_ = conn.SetReadDeadline(time.Now().Add(keepAlive + 10*time.Second))

		logger.DebugF("[%s] Receive %s package, data %+v", connID, header.Type, payload)

		switch header.Type {
		case mqtt.CONNECT:
			logger.ErrorF("[%s] Duplicate CONNECT package", connID)
			return
		case mqtt.PINGREQ:
			handlePingReq(conn, connID)
		case mqtt.DISCONNECT:
			logger.InfoF("[%s] Client disconnect", connID)
			return
		default:
			logger.WarnF("[%s] %s package has not been supported", connID, header.Type.String())
			return
		}
	}
}

func StartServer(port int) {
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		logger.FatalF("MQTT Server Start error: %v", err)
	}
	logger.InfoF("MQTT Server Listen On " + ln.Addr().String())
	defer func() {
		err := ln.Close()
		if err != nil {
			logger.ErrorF("Server close error: %v", err)
		}
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			logger.ErrorF("Accept connection error: %v", err)
			continue
		}

		logger.DebugF("Accepted new connection from %s", conn.RemoteAddr().String())

		sem <- struct{}{}
		go func(c net.Conn) {
			handleConnection(c)
			<-sem
		}(conn)
	}
}
