// Package mqtt 实现了MQTT协议的核心类型定义和常量
package mqtt

// PacketType 定义了MQTT控制报文的类型
type PacketType byte

// MQTT 控制报文类型常量定义
const (
	CONNECT     PacketType = iota + 1 // 客户端请求连接到服务器
	CONNACK                           // 连接确认
	PUBLISH                           // 发布消息
	PUBACK                            // 发布确认
	PUBREC                            // 发布收到（QoS 2第一步）
	PUBREL                            // 发布释放（QoS 2第二步）
	PUBCOMP                           // 发布完成（QoS 2第三步）
	SUBSCRIBE                         // 订阅请求
	SUBACK                            // 订阅确认
	UNSUBSCRIBE                       // 取消订阅
	UNSUBACK                          // 取消订阅确认
	PINGREQ                           // 心跳请求
	PINGRESP                          // 心跳响应
	DISCONNECT                        // 断开连接
)

// PacketTypeMap 将PacketType映射到其字符串表示
var PacketTypeMap = map[PacketType]string{
	CONNECT:     "CONNECT",
	CONNACK:     "CONNACK",
	PUBLISH:     "PUBLISH",
	PUBACK:      "PUBACK",
	PUBREC:      "PUBREC",
	PUBREL:      "PUBREL",
	PUBCOMP:     "PUBCOMP",
	SUBSCRIBE:   "SUBSCRIBE",
	SUBACK:      "SUBACK",
	UNSUBSCRIBE: "UNSUBSCRIBE",
	UNSUBACK:    "UNSUBACK",
	PINGREQ:     "PINGREQ",
	PINGRESP:    "PINGRESP",
	DISCONNECT:  "DISCONNECT",
}

// String 返回PacketType的字符串表示
func (packetType PacketType) String() string {
	return PacketTypeMap[packetType]
}

// allowedFlags 定义了每种报文类型允许的标志位组合
var allowedFlags = map[PacketType]byte{
	CONNECT:     0x00, // 0000
	CONNACK:     0x00, // 0000
	PUBLISH:     0x0F, // 1111（允许所有标志位组合）
	PUBACK:      0x00, // 0000
	PUBREC:      0x00, // 0000
	PUBREL:      0x02, // 0010
	PUBCOMP:     0x00, // 0000
	SUBSCRIBE:   0x02, // 0010
	SUBACK:      0x00, // 0000
	UNSUBSCRIBE: 0x02, // 0010
	UNSUBACK:    0x00, // 0000
	PINGREQ:     0x00, // 0000
	PINGRESP:    0x00, // 0000
	DISCONNECT:  0x00, // 0000
}

// FixedHeader 定义了MQTT固定头部结构
type FixedHeader struct {
	Type            PacketType // 报文类型
	Flags           byte       // 标志位
	RemainingLength int        // 剩余长度
}

// Payload 定义了MQTT报文负载结构
type Payload struct {
	Context    []byte // 负载内容
	ContextLen int    // 负载长度
	CurrentPtr int    // 当前读取位置
}

// Packet 定义了完整的MQTT报文结构
type Packet struct {
	Header  *FixedHeader // 固定头部
	Payload *Payload     // 可变头部和有效载荷
}
