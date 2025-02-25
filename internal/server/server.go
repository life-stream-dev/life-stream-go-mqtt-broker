package server

import (
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/logger"
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/mqtt"
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/packet"
	"log"
	"net"
	"strconv"
	"time"
)

const (
	KeepAliveTimeout = 30 * time.Second
)

var sem = make(chan struct{}, 10000) // 限制 1 万并发

func handleConnect(conn net.Conn, payload []byte) {
	// 实现 CONNECT 报文解析
	// [示例代码应包含协议版本校验、客户端标识处理等]
	resp := []byte{0x20, 0x02, 0x00, 0x00} // CONNACK
	if _, err := conn.Write(resp); err != nil {
		logger.Error("Write CONNACK error: %v", err)
	}
}

func handleConnection(conn net.Conn) {
	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {
			logger.Error("Close error: %v", err)
		}
	}(conn)

	// 设置读写超时
	err := conn.SetDeadline(time.Now().Add(KeepAliveTimeout))
	if err != nil {
		logger.Error("SetDeadline error: %v", err)
	}

	for {
		header, payload, err := mqtt.ReadPacket(conn)
		logger.Debug("ReadPacket: %+v, %+v, %+v", header, payload, err)
		if err != nil {
			logger.Debug("Read error: %v", err)
			return
		}

		// 重置超时计时器
		err = conn.SetDeadline(time.Now().Add(KeepAliveTimeout))
		if err != nil {
			logger.Debug("SetDeadline error: %v", err)
			return
		}

		switch header.Type {
		case mqtt.CONNECT: // CONNECT
			packet.HandlerConnectPacket(payload)
		default:
			panic("unhandled default case")
		}
	}
}

func StartServer(port int) {
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	logger.Debug("MQTT Server Listen On " + ln.Addr().String())
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			logger.Error("Accept error: %v", err)
			continue
		}

		sem <- struct{}{}
		go func(c net.Conn) {
			handleConnection(c)
			<-sem
		}(conn)
	}
}
