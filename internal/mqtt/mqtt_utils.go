package mqtt

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
)

func UInt16ToByte(number uint16) []byte {
	result := make([]byte, 2)
	binary.BigEndian.PutUint16(result, number)
	return result
}

func ReadBytes(r io.Reader, l int) (byte, error) {
	bytes := make([]byte, l)
	_, err := r.Read(bytes)
	if err != nil {
		return 0, err
	}
	return bytes[0], nil
}

func ReadByte(r io.Reader) (byte, error) {
	return ReadBytes(r, 1)
}

func ReadPacket(conn net.Conn) (*Packet, error) {
	// 读取固定头
	typeAndFlags := make([]byte, 1)
	if _, err := io.ReadFull(conn, typeAndFlags); err != nil {
		return nil, err
	}

	// 解析剩余长度
	remaining, err := DecodeRemainingLength(conn)
	if err != nil {
		return nil, err
	}

	// 读取可变头+有效载荷
	payload := make([]byte, remaining)
	if _, err := io.ReadFull(conn, payload); err != nil {
		return nil, err
	}

	header := &FixedHeader{
		Type:            PacketType(typeAndFlags[0] >> 4),
		Flags:           typeAndFlags[0] & 0x0F,
		RemainingLength: remaining,
	}

	if !ValidateFlags(header.Type, header.Flags) {
		return nil, fmt.Errorf("flags %d of %s packet is not valid", header.Flags, header.Type.String())
	}

	return &Packet{
		Header: header,
		Payload: &Payload{
			Context:    payload,
			ContextLen: len(payload),
			CurrentPtr: 0,
		},
	}, nil
}

func DecodeRemainingLength(r io.Reader) (int, error) {
	multiplier := 1
	value := 0
	for i := 0; i < 4; i++ { // 最多读取4字节
		encodedByte, err := ReadByte(r)
		if err != nil {
			return 0, err
		}
		value += int(encodedByte&127) * multiplier
		multiplier *= 128
		if (encodedByte & 128) == 0 {
			return value, nil
		}
	}
	return 0, errors.New("the remaining length exceeds the 4 byte limit")
}

func EncodeRemainingLength(x int) []byte {
	var buf [4]byte
	i := 0
	for x > 0 && i < 4 {
		buf[i] = byte(x % 128)
		if x /= 128; x > 0 {
			buf[i] |= 128
		}
		i++
	}
	return buf[:i]
}

func ValidateFlags(pt PacketType, flags byte) bool {
	allowed := allowedFlags[pt]
	// 检查标志位是否在允许范围内
	return (flags & ^allowed) == 0
}

func (p *Payload) CheckRemainingLength() bool {
	return p.CurrentPtr < p.ContextLen
}
