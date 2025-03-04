package main

import (
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/config"
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/database"
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/event"
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/logger"
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/server"
)

func main() {
	_, err := config.ReadConfig()
	if err != nil {
		logger.FatalF("Error occured while reading config %v", err)
		return
	}
	loggerCallback := logger.Init()
	logger.Debug("Application initializing...")
	cleaner := event.NewCleaner()
	cleaner.Init(loggerCallback)
	defer cleaner.Clean()
	err = database.ConnectDatabase()
	if err != nil {
		logger.FatalF("Error occured while initializing database, details: %v", err)
		return
	}
	server.StartServer(1883)
}
