package database

import "sync"

type PacketIDManager struct {
	mu        sync.Mutex
	currentID uint16
	released  map[uint16]struct{}
}

var PacketManager *PacketIDManager

func NewPacketIDManager() *PacketIDManager {
	if PacketManager == nil {
		PacketManager = &PacketIDManager{
			currentID: 1, // 起始值为1
			released:  make(map[uint16]struct{}),
		}
	}
	return PacketManager
}

// NextID 获取下一个可用ID
func (m *PacketIDManager) NextID() uint16 {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 优先使用已释放的ID
	for id := range m.released {
		delete(m.released, id)
		return id
	}

	// 分配新ID
	id := m.currentID
	m.currentID++
	if m.currentID == 0 { // 溢出处理
		m.currentID = 1
	}
	return id
}

// ReleaseID 释放ID（收到确认后调用）
func (m *PacketIDManager) ReleaseID(id uint16) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.released[id] = struct{}{}
}

// EncodePacketID 编码为字节流（大端序）
func EncodePacketID(id uint16) []byte {
	return []byte{byte(id >> 8), byte(id & 0xFF)}
}

// DecodePacketID 从字节流解码
func DecodePacketID(data []byte) uint16 {
	if len(data) < 2 {
		return 0
	}
	return uint16(data[0])<<8 | uint16(data[1])
}
