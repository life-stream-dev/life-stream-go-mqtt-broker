package utils

import (
	"github.com/life-stream-dev/life-stream-go-mqtt-broker/internal/logger"
	"strconv"
	"strings"
	"time"
)

func ParseStringTime(timeString string) time.Duration {
	timeString = strings.ToLower(timeString)
	if cutString, _, found := strings.Cut(timeString, "s"); found {
		number, err := strconv.Atoi(cutString)
		if err != nil {
			logger.ErrorF("Error parsing time string: %s", err.Error())
			return 0
		}
		return time.Duration(number) * time.Second
	}
	if cutString, _, found := strings.Cut(timeString, "m"); found {
		number, err := strconv.Atoi(cutString)
		if err != nil {
			logger.ErrorF("Error parsing time string: %s", err.Error())
			return 0
		}
		return time.Duration(number) * time.Minute
	}
	if cutString, _, found := strings.Cut(timeString, "h"); found {
		number, err := strconv.Atoi(cutString)
		if err != nil {
			logger.ErrorF("Error parsing time string: %s", err.Error())
			return 0
		}
		return time.Duration(number) * time.Hour
	}
	if cutString, _, found := strings.Cut(timeString, "d"); found {
		number, err := strconv.Atoi(cutString)
		if err != nil {
			logger.ErrorF("Error parsing time string: %s", err.Error())
			return 0
		}
		return time.Duration(number) * time.Hour * 24
	}
	logger.ErrorF("invalid time format: %s", timeString)
	return 0
}
