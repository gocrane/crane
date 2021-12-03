package prom

import (
	"bytes"
	gocontext "context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gocrane/crane/pkg/common"
	"github.com/gocrane/crane/pkg/providers"
	"github.com/gocrane/crane/pkg/utils/log"
)

const (
	ClusterContextName = "cluster"
)

var logger = log.Logger()

type prom struct {
	ctx *context
}

// NewProvider return a prometheus data provider
func NewProvider(config *providers.PromConfig) (providers.Interface, error) {
	//klog.Infof("NewDataPromSource")

	client, err := NewPrometheusClient(config.Address, config.Timeout, config.KeepAlive,
		config.QueryConcurrency, config.InsecureSkipVerify, config.BRateLimit, config.Auth)
	if err != nil {
		return nil, err
	}

	ctx := NewNamedContext(client, ClusterContextName)

	return &prom{ctx: ctx}, nil
}

func (p *prom) GetTimeSeries(metricName string, conditions []common.QueryCondition, startTime time.Time, endTime time.Time, step time.Duration) ([]*common.TimeSeries, error) {
	if err := checkQueryConditions(conditions); err != nil {
		return []*common.TimeSeries{}, err
	}

	queryExpr := fmt.Sprintf("%s{%s}", metricName, getQueryConditionStr(conditions))
	logger.V(5).Info("GetLatestTimeSeries", "queryExpr", queryExpr)

	timeSeries, err := p.ctx.QueryRangeSync(gocontext.TODO(), queryExpr, startTime, endTime, step)
	if err != nil {
		logger.Error(err, "Failed to QueryTimeSeries")
		return []*common.TimeSeries{}, err
	}

	return timeSeries, nil
}

func (p *prom) GetLatestTimeSeries(metricName string, conditions []common.QueryCondition) ([]*common.TimeSeries, error) {
	if err := checkQueryConditions(conditions); err != nil {
		return []*common.TimeSeries{}, err
	}

	queryExpr := fmt.Sprintf("%s{%s}", metricName, getQueryConditionStr(conditions))
	logger.V(5).Info("GetLatestTimeSeries", "queryExpr", queryExpr)

	timeSeries, err := p.ctx.QuerySync(gocontext.TODO(), queryExpr)
	if err != nil {
		logger.Error(err, "Failed to QueryTimeSeries")
		return []*common.TimeSeries{}, err
	}

	return timeSeries, nil
}

func (p *prom) QueryTimeSeries(queryExpr string, startTime time.Time, endTime time.Time, step time.Duration) ([]*common.TimeSeries, error) {
	timeSeries, err := p.ctx.QueryRangeSync(gocontext.TODO(), queryExpr, startTime, endTime, step)
	if err != nil {
		logger.Error(err, "Failed to QueryTimeSeries")
		return nil, err
	}

	return timeSeries, nil
}

func (p *prom) QueryLatestTimeSeries(queryExpr string) ([]*common.TimeSeries, error) {
	timeSeries, err := p.ctx.QuerySync(gocontext.TODO(), queryExpr)
	if err != nil {
		logger.Error(err, "Failed to QueryLatestTimeSeries")
		return nil, err
	}

	return timeSeries, nil
}

func getQueryConditionStr(conditions []common.QueryCondition) string {
	var buf bytes.Buffer

	for _, cond := range conditions {
		if len(cond.Value) == 0 {
			continue
		}

		if buf.Len() > 0 {
			buf.WriteString(",")
		}

		buf.WriteString(cond.Key)

		if len(cond.Value) == 1 {
			buf.WriteString(string(cond.Operator))
			buf.WriteString(cond.Value[0])
		} else {
			var op common.Operator
			switch cond.Operator {
			case common.OperatorEqual, common.OperatorIn:
				op = common.OperatorRegexMatch
			case common.OperatorNotEqual:
				op = common.OperatorNotRegexMatch
			default:
				op = cond.Operator
			}
			buf.WriteString(string(op))

			buf.WriteString("\"" + strings.Join(cond.Value, "|") + "\"")
		}
	}

	return buf.String()
}

func checkQueryConditions(conditions []common.QueryCondition) error {
	for _, cond := range conditions {
		if cond.Key == "" {
			return errors.New("query condition key can not be empty")
		}

		if len(cond.Value) == 0 {
			return errors.New("query condition value length can not be zero")
		}

		for _, val := range cond.Value {
			if val == "" {
				return errors.New("query condition value can not be empty")
			}
		}
	}

	return nil
}
