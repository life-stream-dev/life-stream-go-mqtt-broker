package main

import (
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/config"
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/logger"
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/server"
)

func main() {
	_, err := config.ReadConfig()
	if err != nil {
		logger.ErrorF("ErrorF occured while reading config %v", err)
		return
	}
	loggerCallback := logger.Init()
	logger.Debug("Application initializing...")
	cleaner := server.NewCleaner()
	cleaner.Init(loggerCallback)
	server.StartServer(1883)
}
