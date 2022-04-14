package utils

import "strconv"

func ParseFloat(str string, defaultValue float64) (float64, error) {
	if len(str) == 0 {
		return defaultValue, nil
	}
	return strconv.ParseFloat(str, 64)
}
