package ebpf

import (
	"sync"

	"k8s.io/klog/v2"
)

type EBPF struct {
	Name        string
	StatusCache sync.Map
}

func (e *EBPF) GetName() string {
	return e.Name
}

func NewEBPF() *EBPF {
	e := EBPF{
		Name:        "ebpf",
		StatusCache: sync.Map{},
	}
	return &e
}

func (e *EBPF) Collect() {
	klog.V(4).Infof("Ebpf collecting")
	return
}
