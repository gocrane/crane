package dsp

import (
	"testing"

	"k8s.io/apimachinery/pkg/labels"

	"github.com/gocrane/crane/pkg/metricnaming"
	"github.com/gocrane/crane/pkg/metricquery"
	"github.com/gocrane/crane/pkg/prediction"
	"github.com/stretchr/testify/assert"
)

func TestAggregateSignals_Add(t *testing.T) {
	a := newAggregateSignals()
	namerHello := &metricnaming.GeneralMetricNamer{
		Metric: &metricquery.Metric{
			Type: metricquery.PromQLMetricType,
			Prom: &metricquery.PromNamerInfo{
				QueryExpr: "hello",
				Selector:  labels.Nothing(),
			},
		}}
	qc := prediction.QueryExprWithCaller{
		MetricNamer: namerHello,
		Caller:      "link",
	}
	assert.True(t, a.Add(qc))
	assert.False(t, a.Add(qc))

	qc.Caller = "snake"
	assert.False(t, a.Add(qc))

	namerHey := &metricnaming.GeneralMetricNamer{
		Metric: &metricquery.Metric{
			Type: metricquery.PromQLMetricType,
			Prom: &metricquery.PromNamerInfo{
				QueryExpr: "hey",
				Selector:  labels.Nothing(),
			},
		}}
	qc.MetricNamer = namerHey
	assert.True(t, a.Add(qc))

	qc.Caller = "link"
	assert.False(t, a.Add(qc))
}

func TestAggregateSignals_Delete(t *testing.T) {
	namerHello := &metricnaming.GeneralMetricNamer{
		Metric: &metricquery.Metric{
			Type: metricquery.PromQLMetricType,
			Prom: &metricquery.PromNamerInfo{
				QueryExpr: "hello",
				Selector:  labels.Nothing(),
			},
		}}
	namerHey := &metricnaming.GeneralMetricNamer{
		Metric: &metricquery.Metric{
			Type: metricquery.PromQLMetricType,
			Prom: &metricquery.PromNamerInfo{
				QueryExpr: "hey",
				Selector:  labels.Nothing(),
			},
		}}

	a := newAggregateSignals()
	qc := prediction.QueryExprWithCaller{
		MetricNamer: namerHello,
		Caller:      "link",
	}
	a.Add(qc)
	qc.MetricNamer = namerHey
	a.Add(qc)
	qc.Caller = "snake"
	a.Add(qc)

	signal := &aggregateSignal{}
	a.SetSignal(namerHello.BuildUniqueKey(), "k1", signal)
	signal2 := &aggregateSignal{}
	a.SetSignal(namerHey.BuildUniqueKey(), "k1", signal2)

	qc = prediction.QueryExprWithCaller{
		MetricNamer: namerHello,
		Caller:      "link",
	}
	assert.True(t, a.Delete(qc))
	assert.Nil(t, a.GetSignal(namerHello.BuildUniqueKey(), "k1"))

	qc.MetricNamer = namerHey
	assert.False(t, a.Delete(qc))
	assert.NotNil(t, a.GetSignal(namerHey.BuildUniqueKey(), "k1"))
	assert.Equal(t, signal2, a.GetSignal(namerHey.BuildUniqueKey(), "k1"))

	qc.Caller = "snake"
	assert.True(t, a.Delete(qc))
	assert.Nil(t, a.GetSignal("key", "k1"))
}

func TestAggregateSignals_SetSignal(t *testing.T) {
	namerHello := &metricnaming.GeneralMetricNamer{
		Metric: &metricquery.Metric{
			Type: metricquery.PromQLMetricType,
			Prom: &metricquery.PromNamerInfo{
				QueryExpr: "hello",
				Selector:  labels.Nothing(),
			},
		}}
	queryExpr := namerHello.BuildUniqueKey()
	a := newAggregateSignals()
	signal := &aggregateSignal{}
	a.SetSignal(queryExpr, "link", signal)
	assert.Nil(t, a.GetSignal(queryExpr, "link"))
	a.Add(prediction.QueryExprWithCaller{
		MetricNamer: namerHello,
		Caller:      "link",
	})
	a.SetSignal(queryExpr, "link", signal)
	assert.Equal(t, signal, a.GetSignal(queryExpr, "link"))
}

func TestAggregateSignals_GetSignal(t *testing.T) {
	namerHello := &metricnaming.GeneralMetricNamer{
		Metric: &metricquery.Metric{
			Type: metricquery.PromQLMetricType,
			Prom: &metricquery.PromNamerInfo{
				QueryExpr: "hello",
				Selector:  labels.Nothing(),
			},
		}}
	queryExpr := namerHello.BuildUniqueKey()

	a := newAggregateSignals()
	a.Add(prediction.QueryExprWithCaller{
		MetricNamer: namerHello,
		Caller:      "link",
	})
	signal := &aggregateSignal{}
	a.SetSignal(queryExpr, "k1", signal)
	assert.Equal(t, signal, a.GetSignal(queryExpr, "k1"))

	signal2 := &aggregateSignal{}
	a.SetSignal(queryExpr, "k2", signal2)
	assert.Equal(t, signal2, a.GetSignal(queryExpr, "k2"))
}

func TestAggregateSignals_GetOrStoreSignal(t *testing.T) {
	namerHello := &metricnaming.GeneralMetricNamer{
		Metric: &metricquery.Metric{
			Type: metricquery.PromQLMetricType,
			Prom: &metricquery.PromNamerInfo{
				QueryExpr: "hello",
				Selector:  labels.Nothing(),
			},
		}}
	queryExpr := namerHello.BuildUniqueKey()

	a := newAggregateSignals()
	qc := prediction.QueryExprWithCaller{
		MetricNamer: namerHello,
		Caller:      "link",
	}
	a.Add(qc)
	signal := &aggregateSignal{}
	a.SetSignal(queryExpr, "k1", signal)
	assert.Equal(t, signal, a.GetOrStoreSignal(queryExpr, "k1", signal))
	assert.Equal(t, signal, a.GetSignal(queryExpr, "k1"))
}
