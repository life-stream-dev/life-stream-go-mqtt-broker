// Package server 实现了MQTT服务器的核心功能
package server

import (
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/logger"
	"net"
	"strconv"
)

// sem 用于控制并发连接数的信号量
var sem = make(chan struct{}, 10000)

// StartServer 启动MQTT服务器
// port: 服务器监听的端口号
func StartServer(port int) {
	// 创建TCP监听器
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		logger.FatalF("MQTT Server Start error: %v", err)
	}
	logger.InfoF("MQTT Server Listen On " + ln.Addr().String())

	// 确保在函数退出时关闭监听器
	defer func() {
		err := ln.Close()
		if err != nil {
			logger.ErrorF("Server close error: %v", err)
		}
	}()

	// 循环接受新的连接
	for {
		conn, err := ln.Accept()
		if err != nil {
			logger.ErrorF("Accept connection error: %v", err)
			continue
		}

		logger.DebugF("Accepted new connection from %s", conn.RemoteAddr().String())

		// 使用信号量控制并发连接数
		sem <- struct{}{}
		go func(c net.Conn) {
			// 创建连接处理器
			connection := &ConnectionHandler{
				conn:      conn,
				connId:    conn.RemoteAddr().String(),
				keepAlive: 60,
			}
			// 处理连接
			connection.handleConnection()
			// 释放信号量
			<-sem
		}(conn)
	}
}
