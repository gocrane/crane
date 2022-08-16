package executor

import (
	"container/heap"
	"strconv"
	"testing"

	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/stretchr/testify/assert"
)

func (w Watermark) verify(t *testing.T, i int) {
	t.Helper()
	n := w.Len()
	j1 := 2*i + 1
	j2 := 2*i + 2
	if j1 < n {
		if w.Less(j1, i) {
			t.Errorf("heap invariant invalidated [%d] = %d > [%d] = %d", i, w[i].Value(), j1, w[j1].Value())
			return
		}
		w.verify(t, j1)
	}
	if j2 < n {
		if w.Less(j2, i) {
			t.Errorf("heap invariant invalidated [%d] = %d > [%d] = %d", i, w[i].Value(), j1, w[j2].Value())
			return
		}
		w.verify(t, j2)
	}
}

// TestPopSmallest make sure that we can get the smallest value
func TestPopSmallest(t *testing.T) {
	h := Watermark{}

	for i := 20; i > 0; i-- {
		heap.Push(&h, resource.MustParse(strconv.Itoa(i)+"m"))
		assert.Equal(t, strconv.Itoa(i)+"m", h.PopSmallest().String())
	}

	h.verify(t, 0)

	t.Logf("watetline is %s", h.String())
	for i := 1; h.Len() > 0; i++ {
		heap.Pop(&h)
		h.verify(t, 0)
	}
}
