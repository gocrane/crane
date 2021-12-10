package collect

import (
	"fmt"
	"sync"
)

type NodeLocal struct {
	Name        string
	StatusCache sync.Map
}

func NewNodeLocal() *NodeLocal {
	n := NodeLocal{
		Name:        "nodelocal",
		StatusCache: sync.Map{},
	}
	return &n
}

func (n *NodeLocal) GetName() string {
	return n.Name
}

func (e *NodeLocal) Collect() {
	fmt.Println("node local collecting")
}

func (e *NodeLocal) List() sync.Map {
	fmt.Println("node local listing")
	return e.StatusCache
}
