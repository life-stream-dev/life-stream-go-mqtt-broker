package mqtt

import (
	"errors"
	"fmt"
	"io"
	"net"
)

type FixedHeader struct {
	Type            PacketType
	Flags           byte
	RemainingLength int
}

func ByteToUInt16(bytes []byte) uint16 {
	if len(bytes) == 0 {
		return 0
	}
	if len(bytes) == 1 {
		return uint16(bytes[0])
	}
	return uint16(bytes[0])<<8 | uint16(bytes[1])
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

func ReadPacket(conn net.Conn) (*FixedHeader, []byte, error) {
	// 读取固定头
	typeAndFlags := make([]byte, 1)
	if _, err := io.ReadFull(conn, typeAndFlags); err != nil {
		return nil, nil, err
	}

	// 解析剩余长度
	remaining, err := DecodeRemainingLength(conn)
	if err != nil {
		return nil, nil, err
	}

	// 读取可变头+有效载荷
	payload := make([]byte, remaining)
	if _, err := io.ReadFull(conn, payload); err != nil {
		return nil, nil, err
	}

	header := &FixedHeader{
		Type:            PacketType(typeAndFlags[0] >> 4),
		Flags:           typeAndFlags[0] & 0x0F,
		RemainingLength: remaining,
	}

	if !ValidateFlags(header.Type, header.Flags) {
		return nil, nil, fmt.Errorf("flags %d of %s packet is not valid", header.Flags, header.Type.String())
	}

	return header, payload, nil
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
