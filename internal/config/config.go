// Package config 实现了MQTT服务器的配置管理功能
package config

import (
	"encoding/json"
	"errors"
	"os"
)

// Config 定义了MQTT服务器的配置结构
type Config struct {
	Database struct {
		Host               string `json:"host"`                 // 数据库主机地址
		Port               uint64 `json:"port"`                 // 数据库端口
		Username           string `json:"username"`             // 数据库用户名
		Password           string `json:"password"`             // 数据库密码
		Database           string `json:"database"`             // 数据库名称
		UseTLS             bool   `json:"use_tls"`              // 是否使用TLS
		ConnectTimeout     string `json:"connect_timeout"`      // 连接超时时间
		SocketTimeout      string `json:"socket_timeout"`       // Socket超时时间
		ConnectIdleTimeout string `json:"connect_idle_timeout"` // 连接空闲超时时间
		OperationTimeout   string `json:"operation_timeout"`    // 操作超时时间
		Heartbeat          string `json:"heartbeat"`            // 心跳间隔
		MinPoolSize        uint64 `json:"min_pool_size"`        // 最小连接池大小
		MaxPoolSize        uint64 `json:"max_pool_size"`        // 最大连接池大小
	} `json:"database"`
	DebugMode bool   `json:"debug_mode"` // 是否启用调试模式
	AppName   string `json:"app_name"`   // 应用名称
	AppPort   int    `json:"app_port"`   // 应用端口
}

var (
	config      Config
	initialized = false
)

// ReadConfig 从配置文件读取配置
func ReadConfig() (Config, error) {
	// 读取配置文件
	bytes, err := os.ReadFile("config.json")

	if err != nil {
		// 如果配置文件不存在，创建默认配置
		writer, _ := os.OpenFile("config.json", os.O_RDONLY|os.O_CREATE, 0777)
		data, _ := json.MarshalIndent(config, "", "\t")
		_, _ = writer.Write(data)
		_ = writer.Close()
		return config, errors.New("the configuration file does not exist and has been created. Please try again after editing the configuration file")
	}

	// 解析JSON配置
	err = json.Unmarshal(bytes, &config)

	if err != nil {
		return config, errors.New("the configuration file does not contain valid JSON")
	}

	initialized = true
	return config, nil
}

// GetConfig 获取配置，如果未初始化则先读取配置
func GetConfig() (Config, error) {
	if initialized {
		return config, nil
	}
	return ReadConfig()
}
