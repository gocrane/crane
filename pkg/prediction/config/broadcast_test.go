package config

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBroadcaster(t *testing.T) {
	b := NewBroadcaster()

	r1 := b.Listen()

	b.Write("first")
	assert.Equal(t, "first", r1.Read().(string))

	r2 := b.Listen()

	b.Write(99)
	assert.Equal(t, 99, r1.Read().(int))
	assert.Equal(t, 99, r2.Read().(int))

	b.Write("hello")
	b.Write("finops")
	assert.Equal(t, "hello", r1.Read().(string))
	assert.Equal(t, "finops", r1.Read().(string))
	assert.Equal(t, "hello", r2.Read().(string))
	assert.Equal(t, "finops", r2.Read().(string))
}
