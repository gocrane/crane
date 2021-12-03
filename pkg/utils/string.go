package utils

import "strconv"

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
