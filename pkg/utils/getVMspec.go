package utils

import (
	"fmt"
	"strconv"

	"k8s.io/apimachinery/pkg/api/resource"
)

func GetVMSpec(cpu, mem *resource.Quantity, specs []ResourceSpec) (resource.Quantity, resource.Quantity) {
	var cpuCores float64
	var memInGBi float64
	var specCList []float64
	// var specs []ResourceSpec

	val := float64(cpu.MilliValue()) / 1000.

	for i := 0; i < len(specs); i++ {
		specC, _ := strconv.ParseFloat(specs[i].CPU, 64)
		specCList = append(specCList, specC)
	}

	m := map[float64][]ResourceSpec{}
	for _, s := range specs {
		sCPU, _ := strconv.ParseFloat(s.CPU, 64)
		m[sCPU] = append(m[sCPU], s)

	}
	for i := 0; i < len(specs); i++ {
		if val <= specCList[i] {
			l := len(m[specCList[i]])
			m1CPU, _ := strconv.ParseFloat(m[specCList[i]][l-1].Memory, 64)
			if float64(mem.Value())/(1024.*1024.*1024.) <= m1CPU {
				cpuCores = specCList[i]
				break
			}
		}
	}
	if cpuCores > 0.0 {
		val = float64(mem.Value()) / (1024. * 1024. * 1024.)
		memList := m[cpuCores]
		for i := range memList {
			m1Mem, _ := strconv.ParseFloat(memList[i].Memory, 64)
			if val <= m1Mem {
				memInGBi = m1Mem
				break
			}
		}
	}
	return resource.MustParse(fmt.Sprintf("%.2f", cpuCores)), resource.MustParse(fmt.Sprintf("%.2fGi", memInGBi))
}
