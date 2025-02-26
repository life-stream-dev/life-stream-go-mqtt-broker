package config

import (
	"encoding/json"
	"errors"
	"os"
)

type Config struct {
	Database struct {
		Host     string `json:"host"`
		Port     int    `json:"port"`
		Username string `json:"username"`
		Password string `json:"password"`
	} `json:"database"`
	DebugMode bool `json:"debug_mode"`
}

var config Config
var initialized = false

func ReadConfig() (Config, error) {
	bytes, err := os.ReadFile("config.json")

	if err != nil {
		writer, _ := os.OpenFile("config.json", os.O_RDONLY|os.O_CREATE, 0777)
		data, _ := json.MarshalIndent(config, "", "\t")
		_, _ = writer.Write(data)
		_ = writer.Close()
		return config, errors.New("the configuration file does not exist and has been created. Please try again after editing the configuration file")
	}

	err = json.Unmarshal(bytes, &config)

	if err != nil {
		return config, errors.New("the configuration file does not contain valid JSON")
	}

	initialized = true
	return config, nil
}

func GetConfig() (Config, error) {
	if initialized {
		return config, nil
	}
	return ReadConfig()
}
