package mqtt

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestRemainingLength(t *testing.T) {
	tests := []struct {
		input  int
		expect []byte
	}{
		{64, []byte{0x40}},
		{321, []byte{0xC1, 0x02}},
		{268435455, []byte{0xFF, 0xFF, 0xFF, 0x7F}},
	}

	for _, tt := range tests {
		encoded := EncodeRemainingLength(tt.input)
		if !bytes.Equal(encoded, tt.expect) {
			t.Errorf("输入=%d 期望=%x 实际=%x", tt.input, tt.expect, encoded)
		}

		decoded, _ := DecodeRemainingLength(bytes.NewReader(encoded))
		if decoded != tt.input {
			t.Errorf("输入=%d 解码后=%d", tt.input, decoded)
		}
	}
}

func TestByteToUInt16(t *testing.T) {
	tests := []struct {
		input  []byte
		expect uint16
	}{
		{[]byte{0x00, 0x00}, 0},
		{[]byte{0x01, 0x00}, 256},
		{[]byte{0xAF, 0x89}, 44937},
	}
	for _, tt := range tests {
		number := binary.BigEndian.Uint16(tt.input)
		if number != tt.expect {
			t.Errorf("输入=%x 期望=%d 实际=%d", tt.input, tt.expect, number)
		}
	}
}
