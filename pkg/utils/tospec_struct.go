package utils

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

type ResourceSpec struct {
	CPU    string
	Memory string
}

var ResourceSpecs []ResourceSpec

func ToSpecStrurt(s string) ([]ResourceSpec, error) {
	var arr1 []string
	arr := strings.Split(s, ",")
	sort.Strings(arr)
	fmt.Println(arr)
	for i := 0; i < len(arr); i++ {
		reg := regexp.MustCompile(`[0-9.]+`)
		if reg != nil {
			arr1 = reg.FindAllString(arr[i], -1)
		}

		ResourceSpecs1 := &ResourceSpec{
			CPU:    arr1[0],
			Memory: arr1[1],
		}
		ResourceSpecs = append(ResourceSpecs, *ResourceSpecs1)
	}
	return ResourceSpecs, nil
}
