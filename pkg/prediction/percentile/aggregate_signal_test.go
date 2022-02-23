package percentile

import (
	"testing"

	"github.com/gocrane/crane/pkg/prediction"
	"github.com/stretchr/testify/assert"
)

func TestAggregateSignals_Add(t *testing.T) {
	a := newAggregateSignals()
	qc := prediction.QueryExprWithCaller{
		QueryExpr: "hello",
		Caller:    "link",
	}
	assert.True(t, a.Add(qc))
	assert.False(t, a.Add(qc))

	qc.Caller = "snake"
	assert.False(t, a.Add(qc))

	qc.QueryExpr = "hey"
	assert.True(t, a.Add(qc))

	qc.Caller = "link"
	assert.False(t, a.Add(qc))
}

func TestAggregateSignals_Delete(t *testing.T) {
	a := newAggregateSignals()
	qc := prediction.QueryExprWithCaller{
		QueryExpr: "hello",
		Caller:    "link",
	}
	a.Add(qc)
	qc.QueryExpr = "hey"
	a.Add(qc)
	qc.Caller = "snake"
	a.Add(qc)

	signal := &aggregateSignal{}
	a.SetSignal("hello", "k1", signal)
	signal2 := &aggregateSignal{}
	a.SetSignal("hey", "k1", signal2)

	qc = prediction.QueryExprWithCaller{
		QueryExpr: "hello",
		Caller:    "link",
	}
	assert.True(t, a.Delete(qc))
	assert.Nil(t, a.GetSignal("hello", "k1"))

	qc.QueryExpr = "hey"
	assert.False(t, a.Delete(qc))
	assert.NotNil(t, a.GetSignal("hey", "k1"))
	assert.Equal(t, signal2, a.GetSignal("hey", "k1"))

	qc.Caller = "snake"
	assert.True(t, a.Delete(qc))
	assert.Nil(t, a.GetSignal("key", "k1"))
}

func TestAggregateSignals_SetSignal(t *testing.T) {
	a := newAggregateSignals()
	signal := &aggregateSignal{}
	a.SetSignal("hello", "link", signal)
	assert.Nil(t, a.GetSignal("hello", "link"))
	a.Add(prediction.QueryExprWithCaller{
		QueryExpr: "hello",
		Caller:    "link",
	})
	a.SetSignal("hello", "link", signal)
	assert.Equal(t, signal, a.GetSignal("hello", "link"))
}

func TestAggregateSignals_GetSignal(t *testing.T) {
	a := newAggregateSignals()
	a.Add(prediction.QueryExprWithCaller{
		QueryExpr: "hello",
		Caller:    "link",
	})
	signal := &aggregateSignal{}
	a.SetSignal("hello", "k1", signal)
	assert.Equal(t, signal, a.GetSignal("hello", "k1"))

	signal2 := &aggregateSignal{totalSamplesCount: 1}
	a.SetSignal("hello", "k2", signal2)
	assert.Equal(t, signal2, a.GetSignal("hello", "k2"))
}

func TestAggregateSignals_GetOrStoreSignal(t *testing.T) {
	a := newAggregateSignals()
	qc := prediction.QueryExprWithCaller{
		QueryExpr: "hello",
		Caller:    "link",
	}
	a.Add(qc)
	signal := &aggregateSignal{}
	//a.SetSignal("hello", "k1", signal)
	assert.Equal(t, signal, a.GetOrStoreSignal("hello", "k1", signal))
	assert.Equal(t, signal, a.GetSignal("hello", "k1"))
}
