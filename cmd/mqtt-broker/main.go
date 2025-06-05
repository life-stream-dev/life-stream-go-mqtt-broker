package main

import (
	c "github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/config"
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/database"
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/event"
	__ "github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/grpc"
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/logger"
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"log"
	"net"
	"strconv"
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
	listener, err := net.Listen("tcp", ":"+strconv.Itoa(config.GrpcPort))
	if err != nil {
		logger.FatalF("Fail to open grpc port: %v", err)
		return
	}
	grpcServer := grpc.NewServer()
	__.RegisterDeviceServer(grpcServer, &__.GRPCService{})
	reflection.Register(grpcServer)
	go func() {
		err := grpcServer.Serve(listener)
		if err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()
	server.StartServer(config.AppPort)
}
