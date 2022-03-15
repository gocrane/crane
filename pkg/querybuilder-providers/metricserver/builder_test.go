package metricserver

import (
	"reflect"
	"testing"

	v1 "k8s.io/api/core/v1"

	"github.com/gocrane/crane/pkg/metricquery"
)

func TestNewPromQueryBuilder(t *testing.T) {
	metric := &metricquery.Metric{
		Type: metricquery.WorkloadMetricType,
		Workload: &metricquery.WorkloadNamerInfo{
			Namespace:  "",
			Name:       "test",
			Kind:       "Deployment",
			APIVersion: "v1",
		},
	}
	builder := NewMetricServerQueryBuilder(metric)
	_, err := builder.BuildQuery()
	if err != nil {
		t.Log(err)
	}
}

func TestBuildQuery(t *testing.T) {
	testCases := []struct {
		desc   string
		metric *metricquery.Metric
		want   *metricquery.MetricServerQuery
		err    error
	}{
		{
			desc: "tc1-workload-cpu",
			metric: &metricquery.Metric{
				MetricName: v1.ResourceCPU.String(),
				Type:       metricquery.WorkloadMetricType,
				Workload: &metricquery.WorkloadNamerInfo{
					Namespace:  "default",
					Name:       "test",
					Kind:       "Deployment",
					APIVersion: "v1",
				},
			},
			want: &metricquery.MetricServerQuery{
				Metric: &metricquery.Metric{
					MetricName: v1.ResourceCPU.String(),
					Type:       metricquery.WorkloadMetricType,
					Workload: &metricquery.WorkloadNamerInfo{
						Namespace:  "default",
						Name:       "test",
						Kind:       "Deployment",
						APIVersion: "v1",
					},
				},
			},
		},
		{
			desc: "tc2-workload-mem",
			metric: &metricquery.Metric{
				MetricName: v1.ResourceMemory.String(),
				Type:       metricquery.WorkloadMetricType,
				Workload: &metricquery.WorkloadNamerInfo{
					Namespace:  "default",
					Name:       "test",
					Kind:       "Deployment",
					APIVersion: "v1",
				},
			},
			want: &metricquery.MetricServerQuery{
				Metric: &metricquery.Metric{
					MetricName: v1.ResourceMemory.String(),
					Type:       metricquery.WorkloadMetricType,
					Workload: &metricquery.WorkloadNamerInfo{
						Namespace:  "default",
						Name:       "test",
						Kind:       "Deployment",
						APIVersion: "v1",
					},
				},
			},
		},
		{
			desc: "tc3-container-cpu",
			metric: &metricquery.Metric{
				MetricName: v1.ResourceCPU.String(),
				Type:       metricquery.ContainerMetricType,
				Container: &metricquery.ContainerNamerInfo{
					Namespace:     "default",
					PodName:       "pod-xxx",
					ContainerName: "container",
				},
			},
			want: &metricquery.MetricServerQuery{
				Metric: &metricquery.Metric{
					MetricName: v1.ResourceCPU.String(),
					Type:       metricquery.ContainerMetricType,
					Container: &metricquery.ContainerNamerInfo{
						Namespace:     "default",
						PodName:       "pod-xxx",
						ContainerName: "container",
					},
				},
			},
		},
		{
			desc: "tc4-container-mem",
			metric: &metricquery.Metric{
				MetricName: v1.ResourceMemory.String(),
				Type:       metricquery.ContainerMetricType,
				Container: &metricquery.ContainerNamerInfo{
					Namespace:     "default",
					PodName:       "pod-xxx",
					ContainerName: "container",
				},
			},
			want: &metricquery.MetricServerQuery{
				Metric: &metricquery.Metric{
					MetricName: v1.ResourceMemory.String(),
					Type:       metricquery.ContainerMetricType,
					Container: &metricquery.ContainerNamerInfo{
						Namespace:     "default",
						PodName:       "pod-xxx",
						ContainerName: "container",
					},
				},
			},
		},
		{
			desc: "tc5-node-cpu",
			metric: &metricquery.Metric{
				MetricName: v1.ResourceCPU.String(),
				Type:       metricquery.NodeMetricType,
				Node: &metricquery.NodeNamerInfo{
					Name: "test",
				},
			},
			want: &metricquery.MetricServerQuery{
				Metric: &metricquery.Metric{
					MetricName: v1.ResourceCPU.String(),
					Type:       metricquery.NodeMetricType,
					Node: &metricquery.NodeNamerInfo{
						Name: "test",
					},
				},
			},
		},
		{
			desc: "tc6-node-mem",
			metric: &metricquery.Metric{
				MetricName: v1.ResourceMemory.String(),
				Type:       metricquery.NodeMetricType,
				Node: &metricquery.NodeNamerInfo{
					Name: "test",
				},
			},
			want: &metricquery.MetricServerQuery{
				Metric: &metricquery.Metric{
					MetricName: v1.ResourceMemory.String(),
					Type:       metricquery.NodeMetricType,
					Node: &metricquery.NodeNamerInfo{
						Name: "test",
					},
				},
			},
		},
		{
			desc: "tc7-pod-cpu",
			metric: &metricquery.Metric{
				MetricName: v1.ResourceCPU.String(),
				Type:       metricquery.PodMetricType,
				Pod: &metricquery.PodNamerInfo{
					Namespace: "default",
					Name:      "test",
				},
			},
			want: &metricquery.MetricServerQuery{
				Metric: &metricquery.Metric{
					MetricName: v1.ResourceCPU.String(),
					Type:       metricquery.PodMetricType,
					Pod: &metricquery.PodNamerInfo{
						Namespace: "default",
						Name:      "test",
					},
				},
			},
		},
		{
			desc: "tc8-pod-mem",
			metric: &metricquery.Metric{
				MetricName: v1.ResourceMemory.String(),
				Type:       metricquery.PodMetricType,
				Pod: &metricquery.PodNamerInfo{
					Namespace: "default",
					Name:      "test",
				},
			},
			want: &metricquery.MetricServerQuery{
				Metric: &metricquery.Metric{
					MetricName: v1.ResourceMemory.String(),
					Type:       metricquery.PodMetricType,
					Pod: &metricquery.PodNamerInfo{
						Namespace: "default",
						Name:      "test",
					},
				},
			},
		},
		{
			desc: "tc9-prom",
			metric: &metricquery.Metric{
				MetricName: "http_requests",
				Type:       metricquery.PromQLMetricType,
				Prom: &metricquery.PromNamerInfo{
					QueryExpr: `irate(http_requests{}[3m])`,
				},
			},
			want: &metricquery.MetricServerQuery{
				Metric: &metricquery.Metric{
					MetricName: "http_requests",
					Type:       metricquery.PromQLMetricType,
					Prom: &metricquery.PromNamerInfo{
						QueryExpr: `irate(http_requests{}[3m])`,
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		builder := NewMetricServerQueryBuilder(tc.metric)
		query, err := builder.BuildQuery()
		if !reflect.DeepEqual(err, tc.err) {
			t.Fatalf("tc %v failed, got error: %v, want error: %v", tc.desc, err, tc.err)
		}
		if !reflect.DeepEqual(query.MetricServer, tc.want) {
			t.Fatalf("tc %v failed, got: %v, want: %v", tc.desc, query.MetricServer, tc.want)
		}
	}
}
