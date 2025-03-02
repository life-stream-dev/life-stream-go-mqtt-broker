# life-stream-go-mqtt-broker

# 如何构建本仓库？

1. 安装Go
2. 安装task

```shell
go install github.com/go-task/task/v3/cmd/task@latest
```

记得在PATH环境变量中添加task可执行文件目录

3. 运行命令

```shell
task run
```

## 词汇缩写查找表

| 英文缩写 | 英文全称                   | 中文名        |  
|:-----|:-----------------------|:-----------|
| DSS  | Database Session Store | 数据库会话存储    |
| MSS  | Memory Session Store   | 内存会话存储     |
| PIM  | Packet Id Manager      | 数据包 ID 管理器 |

## 配置文件示例

```json5
{
  "database": {
    "host": "127.0.0.1",
    "port": 27017,
    "username": "root",
    "password": "1234",
    "database": "lifestream",
    "use_tls": false,
    "connect_timeout": "10s",
    "socket_timeout": "5s",
    "connect_idle_timeout": "30m",
    "operation_timeout": "5s",
    "heartbeat": "10s",
    "min_pool_size": 5,
    "max_pool_size": 50
  },
  "app_name": "lifestream",
  "debug_mode": true
}
```