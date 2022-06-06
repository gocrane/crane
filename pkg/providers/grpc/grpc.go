package grpc

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/metricnaming"
	"github.com/gocrane/crane/pkg/metricquery"
	"github.com/gocrane/crane/pkg/providers"
	"github.com/gocrane/crane/pkg/providers/grpc/pb"
)

var _ providers.Interface = &grpcClient{}

func NewProvider(config *providers.GrpcConfig) providers.Interface {
	return &grpcClient{
		addr:    config.Address,
		timeout: config.Timeout,
	}
}

type grpcClient struct {
	addr    string
	timeout time.Duration
}

func (g *grpcClient) QueryTimeSeries(namer metricnaming.MetricNamer, startTime time.Time, endTime time.Time, step time.Duration) ([]*common.TimeSeries, error) {
	m, err := grpcMetric(namer)
	if err != nil {
		return nil, err
	}

	req := &pb.QueryTimeSeriesRequest{
		Metric:    m,
		StartTime: startTime.Unix(),
		EndTime:   endTime.Unix(),
		Step:      int64(step / time.Second),
	}
	conn, err := grpc.Dial(g.addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	c := pb.NewHistoryClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), g.timeout)
	defer cancel()
	resp, err := c.QueryTimeSeries(ctx, req)
	if err != nil {
		return nil, err
	}
	return commonTimeSeriesList(resp.TimeSeriesList), nil
}

func (g *grpcClient) QueryLatestTimeSeries(namer metricnaming.MetricNamer) ([]*common.TimeSeries, error) {
	return nil, fmt.Errorf("not supported")
}

func commonTimeSeriesList(tsList []*pb.TimeSeries) []*common.TimeSeries {
	var res []*common.TimeSeries = make([]*common.TimeSeries, len(tsList))
	for i := range tsList {
		res[i] = &common.TimeSeries{
			Labels:  make([]common.Label, len(tsList[i].Labels)),
			Samples: make([]common.Sample, len(tsList[i].Samples)),
		}
		for j := range tsList[i].Labels {
			res[i].Labels[j] = common.Label{
				Name:  tsList[i].Labels[j].Name,
				Value: tsList[i].Labels[j].Value,
			}
		}
		for j := range tsList[i].Samples {
			res[i].Samples[j] = common.Sample{
				Timestamp: tsList[i].Samples[j].Timestamp,
				Value:     tsList[i].Samples[j].Value,
			}
		}
	}
	return res
}

func grpcMetric(namer metricnaming.MetricNamer) (*pb.Metric, error) {
	q, err := namer.QueryBuilder().Builder(metricquery.GrpcMetricSource).BuildQuery()
	if err != nil {
		return nil, err
	}
	m := &pb.Metric{
		MetricName: q.GenericQuery.Metric.MetricName,
	}
	switch q.GenericQuery.Metric.Type {
	case metricquery.ContainerMetricType:
		c := q.GenericQuery.Metric.Container
		m.Info = &pb.Metric_Container{
			Container: &pb.Container{
				Namespace:    c.Namespace,
				WorkloadName: c.WorkloadName,
				ApiVersion:   c.APIVersion,
				WorkloadKind: c.WorkloadKind,
				Name:         c.Name,
			},
		}
	default:
		return nil, fmt.Errorf("%s not supported", q.GenericQuery.Metric.Type)
	}
	return m, nil
}
