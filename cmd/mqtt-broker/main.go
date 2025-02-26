package main

import (
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/config"
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/logger"
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/server"
	"log/slog"
)

func main() {
	_, err := config.ReadConfig()
	if err != nil {
		logger.Error("Error occured while reading config %v", err)
		return
	}
	logger.Init()
	slog.Debug("Application initializing...")
	server.StartServer(1883)
}
