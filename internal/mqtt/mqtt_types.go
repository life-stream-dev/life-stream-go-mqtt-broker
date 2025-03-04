package mqtt

type PacketType byte

// MQTT 控制报文类型
const (
	CONNECT PacketType = iota + 1
	CONNACK
	PUBLISH
	PUBACK
	PUBREC
	PUBREL
	PUBCOMP
	SUBSCRIBE
	SUBACK
	UNSUBSCRIBE
	UNSUBACK
	PINGREQ
	PINGRESP
	DISCONNECT
)

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

func (packetType PacketType) String() string {
	return PacketTypeMap[packetType]
}

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

type FixedHeader struct {
	Type            PacketType
	Flags           byte
	RemainingLength int
}

type Payload struct {
	Context    []byte
	ContextLen int
	CurrentPtr int
}

type Packet struct {
	Header  *FixedHeader
	Payload *Payload
}
