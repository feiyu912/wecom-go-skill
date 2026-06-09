package wecom

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func ParseTimeToEpoch(raw string) (int64, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, fmt.Errorf("missing time value")
	}
	if digitsOnly(value) {
		return strconv.ParseInt(value, 10, 64)
	}

	normalized := strings.ReplaceAll(value, "Z", "+00:00")
	if t, err := time.Parse(time.RFC3339, normalized); err == nil {
		return t.Unix(), nil
	}
	if t, err := time.ParseInLocation("2006-01-02 15:04", normalized, time.Local); err == nil {
		return t.Unix(), nil
	}
	if t, err := time.ParseInLocation("2006-01-02 15:04:05", normalized, time.Local); err == nil {
		return t.Unix(), nil
	}
	return 0, fmt.Errorf("unsupported time format: %s", raw)
}

func DefaultMeetingStart(now time.Time) int64 {
	return now.Truncate(time.Hour).Add(time.Hour).Unix()
}

func digitsOnly(value string) bool {
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
