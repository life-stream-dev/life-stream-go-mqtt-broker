package main

import (
	c "github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/config"
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/database"
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/event"
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/logger"
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/server"
)

func main() {
	config, err := c.ReadConfig()
	if err != nil {
		logger.FatalF("Error occured while reading config %v", err)
		return
	}
	loggerCallback := logger.Init()
	logger.Info("Application initializing...")
	cleaner := event.NewCleaner()
	cleaner.Init(loggerCallback)
	defer cleaner.Clean()
	err = database.ConnectDatabase()
	if err != nil {
		logger.FatalF("Error occured while initializing database, details: %v", err)
		return
	}
	server.StartServer(config.AppPort)
}
