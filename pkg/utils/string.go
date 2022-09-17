package utils

import (
	"strconv"
	"strings"
)

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

// get string array by seps
func StrSplitAny(s string, seps string) []string {
	splitter := func(r rune) bool {
		return strings.ContainsRune(seps, r)
	}
	return strings.FieldsFunc(s, splitter)
}