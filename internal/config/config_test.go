package config

import (
	"fmt"
	"testing"
)

func TestReadConfig(t *testing.T) {
	config, err := ReadConfig()
	if err != nil {
		fmt.Printf("Error reading configuration file: %v\n", err)
	}
	fmt.Printf("%+v\n", config)
}
