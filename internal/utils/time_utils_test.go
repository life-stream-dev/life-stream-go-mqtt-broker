package utils

import (
	"testing"
	"time"
)

func TestParseStringTime(t *testing.T) {
	tests := []struct {
		timeString string
		expected   time.Duration
	}{
		{"10s", 10 * time.Second},
		{"20M", 20 * time.Minute},
		{"48h", 48 * time.Hour},
		{"2d", 2 * time.Hour * 24},
	}

	for _, test := range tests {
		result := ParseStringTime(test.timeString)
		if result != test.expected {
			t.Errorf("ParseStringTime(%s): expected %v, got %v", test.timeString, test.expected, result)
		}
	}
}
