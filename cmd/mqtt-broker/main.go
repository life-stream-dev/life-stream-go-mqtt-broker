package main

import (
	. "fmt"
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/handlers"
)

func main() {
	Println("Starting server...")
	handlers.StartServer()
}
