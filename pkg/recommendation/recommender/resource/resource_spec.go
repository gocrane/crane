package resource

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"
)

const DefaultSpecs = "0.25c0.25g,0.25c0.5g,0.25c1g,0.5c0.5g,0.5c1g,1c1g,1c2g,1c4g,1c8g,2c2g,2c4g,2c8g,2c16g,4c4g,4c8g,4c16g,4c32g,8c8g,8c16g,8c32g,8c64g,16c32g,16c64g,16c128g,32c64g,32c128g,32c256g,64c128g,64c256g"

type Specification struct {
	CPU    float64
	Memory float64
}

func GetResourceSpecifications(s string) ([]Specification, error) {
	var resourceSpecs []Specification
	arr := strings.Split(s, ",")
	for i := 0; i < len(arr); i++ {
		var arrResource []string
		reg := regexp.MustCompile(`[0-9.]+`)
		if reg != nil {
			arrResource = reg.FindAllString(arr[i], -1)
		}

		if len(arrResource) != 2 {
			return nil, fmt.Errorf("specification %s format error", arr[i])
		}

		cpu, err := strconv.ParseFloat(arrResource[0], 64)
		if err != nil {
			return nil, fmt.Errorf("specification %s cpu format error", arr[i])
		}

		memory, err := strconv.ParseFloat(arrResource[1], 64)
		if err != nil {
			return nil, fmt.Errorf("specification %s memory format error", arr[i])
		}

		resourceSpec := Specification{
			CPU:    cpu,
			Memory: memory,
		}
		resourceSpecs = append(resourceSpecs, resourceSpec)
	}

	sort.Slice(resourceSpecs, func(i, j int) bool {
		if resourceSpecs[i].CPU > resourceSpecs[j].CPU {
			return false
		} else if resourceSpecs[i].CPU < resourceSpecs[j].CPU {
			return true
		} else {
			return resourceSpecs[i].Memory < resourceSpecs[j].Memory
		}
	})

	return resourceSpecs, nil
}

func GetNormalizedResource(cpu, mem *resource.Quantity, specs []Specification) (resource.Quantity, resource.Quantity) {
	var cpuCores float64
	var memInGBi float64

	for i := 0; i < len(specs); i++ {
		if cpuCores > 0 {
			break
		}

		if specs[i].CPU >= float64(cpu.MilliValue())/1000 {
			for j := i; j < len(specs); j++ {
				if specs[i].CPU != specs[j].CPU {
					break
				}
				if specs[j].Memory >= float64(mem.Value())/(1024.*1024.*1024.) {
					cpuCores = specs[j].CPU
					memInGBi = specs[j].Memory
					break
				}
			}
		}
	}

	return resource.MustParse(fmt.Sprintf("%.2f", cpuCores)), resource.MustParse(fmt.Sprintf("%.2fGi", memInGBi))
}
