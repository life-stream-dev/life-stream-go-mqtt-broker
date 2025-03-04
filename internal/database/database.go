package database

import (
	"context"
	"crypto/tls"
	"fmt"
	c "github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/config"
	event2 "github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/event"
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/logger"
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/utils"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"net/url"
	"time"
)

var Client *mongo.Client
var Database *mongo.Database
var OperationTimeout time.Duration

type DBCloseCallback struct {
}

func NewDBCloseCallback() *DBCloseCallback {
	return &DBCloseCallback{}
}

func (dc *DBCloseCallback) Invoke(ctx context.Context) error {
	logger.InfoF("Closing database connection")
	ctx, cancel := context.WithTimeout(context.Background(), OperationTimeout)
	defer cancel()
	return Client.Disconnect(ctx)
}

func ConnectDatabase() error {
	logger.DebugF("Connecting to database...")
	config, err := c.GetConfig()
	if err != nil {
		return fmt.Errorf("error occured while connecting to database: %v", err)
	}

	OperationTimeout = utils.ParseStringTime(config.Database.OperationTimeout)

	// 编码特殊字符
	encodedUser := url.QueryEscape(config.Database.Username)
	encodedPass := url.QueryEscape(config.Database.Password)
	databaseUrl := fmt.Sprintf("mongodb://%s:%s@%s:%d/?authSource=admin",
		encodedUser, encodedPass,
		config.Database.Host,
		config.Database.Port,
	)

	clientOptions := options.Client().ApplyURI(databaseUrl).SetAppName(config.AppName)
	// 连接池配置
	clientOptions.SetMinPoolSize(config.Database.MinPoolSize) // 最小连接数
	clientOptions.SetMaxPoolSize(config.Database.MaxPoolSize) // 最大连接数
	clientOptions.SetMaxConnIdleTime(utils.ParseStringTime(config.Database.ConnectIdleTimeout))
	// 超时限制
	clientOptions.SetConnectTimeout(utils.ParseStringTime(config.Database.ConnectTimeout))
	clientOptions.SetSocketTimeout(utils.ParseStringTime(config.Database.SocketTimeout))
	// 心跳包
	clientOptions.SetHeartbeatInterval(utils.ParseStringTime(config.Database.Heartbeat))
	// TLS
	if config.Database.UseTLS {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: false,
		}
		clientOptions.SetTLSConfig(tlsConfig)
	}
	// 连接池监控
	clientOptions.SetPoolMonitor(&event.PoolMonitor{
		Event: func(evt *event.PoolEvent) {
			switch evt.Type {
			case event.ConnectionCreated:
				logger.DebugF("Database connection created: %+v", evt)
			case event.ConnectionClosed:
				logger.DebugF("Database connection closed: %+v", evt)
			}
		},
	})

	// 创建客户端
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	Client, err = mongo.Connect(ctx, clientOptions)
	if err != nil {
		return fmt.Errorf("error occured while connecting to database: %v", err)
	}

	// 验证连接
	if err = Client.Ping(ctx, nil); err != nil {
		_ = Client.Disconnect(ctx)
		return fmt.Errorf("error occured while pinging database: %v", err)
	}

	Database = Client.Database(config.Database.Database)

	event2.NewCleaner().Add(NewDBCloseCallback())
	return nil
}
