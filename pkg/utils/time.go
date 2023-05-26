package utils

import (
	"regexp"
	"strconv"
	"time"
)

// ParseDuration parse a string to time.Duration
func ParseDuration(s string) (time.Duration, error) {
	if matched, _ := regexp.MatchString("(\\d+)d", s); matched {
		if nDays, err := strconv.ParseInt(s[:len(s)-1], 10, 64); err == nil {
			return time.Hour * 24 * time.Duration(nDays), nil
		}
	}
	return time.ParseDuration(s)
}

// ParseTimestamp parse a string to time.Time
func ParseTimestamp(ts string) (time.Time, error) {
	i, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return time.Now(), err
	}
	return time.Unix(i, 0), nil
}

func NowUTC() time.Time {
	now := time.Now()
	loc, _ := time.LoadLocation("UTC")
	return now.In(loc)
}
