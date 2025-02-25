package mqtt

import "testing"

func TestValidateFlags(t *testing.T) {
	tests := []struct {
		pt     PacketType
		flags  byte
		expect bool
	}{
		{CONNECT, 0x00, true},  // 合法
		{CONNECT, 0x01, false}, // 非法
		{PUBREL, 0x02, true},   // 合法
		{PUBREL, 0x03, false},  // 非法
		{PUBLISH, 0x0F, true},  // 允许所有标志位
	}

	for _, tt := range tests {
		result := ValidateFlags(tt.pt, tt.flags)
		if result != tt.expect {
			t.Errorf("类型=%X 标志=%04b 期望=%v 实际=%v",
				tt.pt, tt.flags, tt.expect, result)
		}
	}
}
