package livetiming

import (
	"strconv"
	"strings"
	"time"
)

func atoiSafe(value string) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0
	}
	return parsed
}

func normalizeCarNumber(value string) string {
	return strings.TrimLeft(strings.TrimSpace(value), "0")
}

func timeToMs(value string) int64 {
	value = strings.TrimSpace(value)
	if value == "" || value == "-" {
		return 0
	}

	parts := strings.Split(value, ":")
	switch len(parts) {
	case 1:
		seconds, err := strconv.ParseFloat(parts[0], 64)
		if err != nil {
			return 0
		}
		return int64(seconds * 1000)
	case 2:
		minutes := atoiSafe(parts[0])
		seconds, err := strconv.ParseFloat(parts[1], 64)
		if err != nil {
			return 0
		}
		return int64((time.Duration(minutes) * time.Minute).Milliseconds()) + int64(seconds*1000)
	case 3:
		hours := atoiSafe(parts[0])
		minutes := atoiSafe(parts[1])
		seconds, err := strconv.ParseFloat(parts[2], 64)
		if err != nil {
			return 0
		}
		total := time.Duration(hours)*time.Hour + time.Duration(minutes)*time.Minute
		return total.Milliseconds() + int64(seconds*1000)
	default:
		return 0
	}
}

func parseLiveTimingMs(value string) int64 {
	value = strings.TrimSpace(value)
	if value == "" || value == "-" {
		return 0
	}
	if strings.Contains(value, ":") {
		return timeToMs(value)
	}
	return atoi64Safe(value)
}

func atoi64Safe(value string) int64 {
	parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil {
		return 0
	}
	return parsed
}

func sanitizeLapTime(value int64) int64 {
	switch {
	case value <= 0:
		return 0
	case value == 3599999:
		return 0
	case value == 215999999:
		return 0
	case value > 12*60*60*1000:
		return 0
	default:
		return value
	}
}
