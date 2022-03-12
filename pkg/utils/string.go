package utils

import (
	"strconv"
	"strings"
)

func ContainsString(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

func ParseFloat(str string, defaultValue float64) (float64, error) {
	if len(str) == 0 {
		return defaultValue, nil
	}
	return strconv.ParseFloat(str, 64)
}

// parsePercentage parse the percent string value
func ParsePercentage(input string) (float64, error) {
	if len(input) == 0 {
		return 0, nil
	}
	value, err := strconv.ParseFloat(strings.TrimRight(input, "%"), 64)
	if err != nil {
		return 0, err
	}
	return value / 100, nil
}
