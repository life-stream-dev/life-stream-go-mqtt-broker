package packet

func NewPingRespPacket() []byte {
	return []byte{0xD0, 0x00}
}
