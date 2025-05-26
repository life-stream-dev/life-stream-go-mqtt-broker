// Package database 实现了MQTT服务器的数据存储功能
package database

import (
	"context"
	"crypto/tls"
	"fmt"
	c "github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/config"
	event2 "github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/event"
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/logger"
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"net/url"
	"slices"
	"time"
)

// 全局数据库连接变量
var (
	Client           *mongo.Client
	Database         *mongo.Database
	Sessions         *mongo.Collection
	Subscriptions    *mongo.Collection
	OperationTimeout time.Duration
)

// DBCloseCallback 实现了数据库关闭回调接口
type DBCloseCallback struct {
}

// NewDBCloseCallback 创建新的数据库关闭回调实例
func NewDBCloseCallback() *DBCloseCallback {
	return &DBCloseCallback{}
}

// Invoke 执行数据库关闭操作
func (dc *DBCloseCallback) Invoke(ctx context.Context) error {
	logger.InfoF("Closing database connection")
	ctx, cancel := context.WithTimeout(context.Background(), OperationTimeout)
	defer cancel()
	return Client.Disconnect(ctx)
}

// ConnectDatabase 连接到MongoDB数据库
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

	// 配置MongoDB客户端选项
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
	// TLS配置
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

	// 获取数据库和集合
	Database = Client.Database(config.Database.Database)
	collections, err := Database.ListCollectionNames(context.Background(), bson.D{})
	if err != nil {
		return fmt.Errorf("error occured while listing collections: %v", err)
	}

	// 创建必要的集合
	for _, collection := range collectionsList {
		if !slices.Contains(collections, collection) {
			logger.DebugF("Creating collection: %s", collection)
			if err := Database.CreateCollection(context.Background(), collection); err != nil {
				return fmt.Errorf("error occured while creating collection %s, details: %v", collection, err)
			}
		}
	}

	// 获取集合引用
	Sessions = Database.Collection(SessionCollectionName)
	Subscriptions = Database.Collection(SubscriptionCollectionName)

	// 删除现有索引
	_, err = Sessions.Indexes().DropAll(context.Background())
	if err != nil {
		return fmt.Errorf("error occured while dropping database indexes: %v", err)
	}

	_, err = Subscriptions.Indexes().DropAll(context.Background())
	if err != nil {
		return fmt.Errorf("error occured while dropping database indexes: %v", err)
	}

	// 创建会话集合索引
	_, err = Sessions.Indexes().CreateOne(
		context.Background(),
		mongo.IndexModel{
			Keys:    bson.D{{Key: "client_id", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("sessions_client_id_unique"),
		},
	)

	// 创建订阅集合索引
	_, err = Subscriptions.Indexes().CreateMany(
		context.Background(), []mongo.IndexModel{
			{
				Keys:    bson.D{{Key: "path", Value: 1}},
				Options: options.Index().SetUnique(true).SetName("subscriptions_path_unique"),
			},
			{
				Keys:    bson.D{{Key: "_id", Value: 1}},
				Options: options.Index().SetName("subscriptions_id_unique"),
			},
		},
	)

	if err != nil {
		return fmt.Errorf("error occured while creating database indexes: %v", err)
	}

	// 注册数据库关闭回调
	event2.NewCleaner().Add(NewDBCloseCallback())
	return nil
}
