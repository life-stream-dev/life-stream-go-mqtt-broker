package server

import (
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/logger"
	"net"
	"strconv"
)

var sem = make(chan struct{}, 10000)

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
			connection := &ConnectionHandler{
				conn:      conn,
				connId:    conn.RemoteAddr().String(),
				keepAlive: 60,
			}
			connection.handleConnection()
			<-sem
		}(conn)
	}
}
