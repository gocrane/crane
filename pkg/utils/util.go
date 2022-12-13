package utils

import (
	"fmt"
	"math"
	"sort"
)

func GetUint64withDefault(i *uint64, value uint64) uint64 {
	if i != nil {
		return *i
	}

	return value
}

func GetInt64withDefault(i *int64, value int64) int64 {
	if i != nil {
		return *i
	}

	return value
}

func GetUint32withDefault(i *uint32, value uint32) uint32 {
	if i != nil {
		return *i
	}

	return value
}

func GetInt32withDefault(i *int32, value int32) int32 {
	if i != nil {
		return *i
	}

	return value
}

func GetUint64FromMaps(key string, maps map[string]uint64) uint64 {
	if v, ok := maps[key]; ok {
		return v
	}

	return 0
}

func Uint32P(value uint32) *uint32 {
	var i = value
	return &i
}

func Uint64P(value uint64) *uint64 {
	var i = value
	return &i
}

func Int32P(value int32) *int32 {
	var i = value
	return &i
}

func Bool2Int32(b bool) int32 {
	if b {
		return 1
	}

	return 0
}

const float64EqualityThreshold = 1e-9

func AlmostEqual(a, b float64) bool {
	return math.Abs(a-b) <= float64EqualityThreshold
}

func StringPtr(str string) *string {
	return &str
}

func Bool2Uint(b bool) uint {
	if b {
		return 1
	}
	return 0
}

func CmpFloat(p1, p2 float64) int32 {
	if AlmostEqual(p1, p2) {
		return 0
	}
	if p1 < p2 {
		return -1
	}
	return 1
}

type SortMap struct {
	key   string
	value string
}

type SortMaps []SortMap

func (a SortMaps) Len() int { return len(a) }

func (a SortMaps) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

func (a SortMaps) Less(i, j int) bool { return a[i].key < a[j].key }

func MapSortToArray(m map[string]string) []string {
	if m == nil || len(m) == 0 {
		return []string{}
	}

	sortMap := make([]SortMap, 0, len(m))
	for label, value := range m {
		sortMap = append(sortMap, SortMap{key: label, value: value})
	}
	// sort to have deterministic string representation
	sort.Sort(SortMaps(sortMap))

	var array []string
	for _, i := range sortMap {
		array = append(array, fmt.Sprintf("%s=\"%s\"", i.key, i.value))
	}
	return array
}
