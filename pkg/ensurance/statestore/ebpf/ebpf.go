package ebpf

import (
	"sync"

	"github.com/gocrane/crane/pkg/log"
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
	log.Logger().V(4).Info("Ebpf collecting")
	return
}
