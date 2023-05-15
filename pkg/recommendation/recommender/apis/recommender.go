package apis

import (
	"strconv"
	"time"
)

func (r Recommender) GetConfigFloat(key string, def float64) (float64, error) {
	if value, exists := r.Config[key]; exists {
		return strconv.ParseFloat(value, 64)
	}
	return def, nil
}

func (r Recommender) GetConfigInt(key string, def int64) (int64, error) {
	if value, exists := r.Config[key]; exists {
		return strconv.ParseInt(value, 10, 64)
	}
	return def, nil
}

func (r Recommender) GetConfigBool(key string, def bool) (bool, error) {
	if value, exists := r.Config[key]; exists {
		return strconv.ParseBool(value)
	}
	return def, nil
}

func (r Recommender) GetConfigString(key string, def string) string {
	if value, exists := r.Config[key]; exists {
		return value
	}
	return def
}

func (r Recommender) GetConfigDuration(key string, def time.Duration) (time.Duration, error) {
	if value, exists := r.Config[key]; exists {
		return time.ParseDuration(value)
	}
	return def, nil
}
