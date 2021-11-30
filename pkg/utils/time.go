package utils

import (
	"regexp"
	"strconv"
	"time"
)

// ParseDuration ParseDuration
func ParseDuration(s string) (time.Duration, error) {
	if matched, _ := regexp.MatchString("(\\d+)d", s); matched {
		if nDays, err := strconv.ParseInt(s[:len(s)-1], 10, 64); err == nil {
			return time.Hour * 24 * time.Duration(nDays), nil
		}
	}
	return time.ParseDuration(s)
}
