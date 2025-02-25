package main

import (
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/logger"
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/server"
	"log/slog"
)

func main() {
	logger.Init()
	slog.Debug("Application initializing...")
	server.StartServer(1883)
}
