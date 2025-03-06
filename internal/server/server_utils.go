package server

import (
	"errors"
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/logger"
	pa "github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/packet"
	"io"
	"net"
	"os"
)

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
	logger.DebugF("[%s] Send %d bytes to client, data %v", connID, total, data)
	return nil
}

func handlePingReq(conn net.Conn, connID string) {
	resp := pa.NewPingRespPacket()
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
