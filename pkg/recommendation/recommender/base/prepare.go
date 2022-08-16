package base

import (
	"fmt"
	"strconv"
	"time"

	"github.com/lithammer/fuzzysearch/fuzzy"

	"github.com/gocrane/crane/pkg/providers"
	"github.com/gocrane/crane/pkg/providers/prom"
	"github.com/gocrane/crane/pkg/recommendation/config"
	"github.com/gocrane/crane/pkg/recommendation/framework"
)

// CheckDataProviders in PrePrepare phase, will create data source provider via your recommendation config.
func (br *BaseRecommender) CheckDataProviders(ctx *framework.RecommendationContext) error {
	// 1. load data provider from recommendation config, override the default data source
	configSet := br.Recommender.Config
	// replicas recommender only need history data provider
	// History data source
	// metricserver can't collect history data
	// default is prometheus, you can override the provider to grpc or override the prometheus config
	// TODO(xieydd) Load cache data source provider if config is not changed.
	configKeys := config.GetKeysOfMap(configSet)
	promKeys := fuzzy.FindFold(string(providers.PrometheusDataSource), configKeys)
	dataSourceKeys := fuzzy.FindFold(providers.DataSourceTypeKey, configKeys)
	//grpcKeys := fuzzy.FindFold(string(providers.GrpcDataSource), configKeys)

	dataSourceType := dataSourceKeys[0]

	if dataSourceType != string(providers.PrometheusDataSource) {
		return fmt.Errorf("in replicas recommender, only suppport prometheus history data source")
	}

	if len(promKeys) != 0 {
		return fmt.Errorf("in replicas recommender, you need set prometheus config %v for history data provider", providers.PrometheusConfigKeys)
	}

	mustSetConfig := []string{"prometheus-address", "prometheus-auth-username", "prometheus-auth-password", "prometheus-auth-bearertoken"}
	if config.SlicesContainSlice(promKeys, mustSetConfig) {
		return fmt.Errorf("in replicas recommender, you need set prometheus config %v for history data provider", mustSetConfig)
	}
	timeOut := 3 * time.Minute
	if value, ok := configSet["prometheus-timeout"]; ok {
		timeOut, _ = time.ParseDuration(value)
	}

	aliveTime := 60 * time.Second
	if value, ok := configSet["prometheus-keepalive"]; ok {
		aliveTime, _ = time.ParseDuration(value)
	}

	concurrency := 10
	if value, ok := configSet["prometheus-query-concurrency"]; ok {
		concurrency, _ = strconv.Atoi(value)
	}

	maxPoints := 11000
	if value, ok := configSet["prometheus-maxpoints"]; ok {
		maxPoints, _ = strconv.Atoi(value)
	}
	promConfig := providers.PromConfig{
		Address:            configSet["prometheus-address"],
		Timeout:            timeOut,
		KeepAlive:          aliveTime,
		InsecureSkipVerify: configSet["prometheus-insecure-skip-verify"] == "true",
		Auth: providers.ClientAuth{
			Username:    configSet["prometheus-auth-username"],
			Password:    configSet["prometheus-auth-password"],
			BearerToken: configSet["prometheus-auth-bearertoken"],
		},
		QueryConcurrency:            concurrency,
		BRateLimit:                  configSet["prometheus-bratelimit"] == "true",
		MaxPointsLimitPerTimeSeries: maxPoints,
	}
	promDataProvider, err := prom.NewProvider(&promConfig)
	if err != nil {
		return err
	}
	ctx.DataProviders = map[providers.DataSourceType]providers.History{
		providers.PrometheusDataSource: promDataProvider,
	}

	// will be use in future.
	// if no data provider config set, use default history data provider
	/*
		if len(dataSourceKeys) > 0 {
			switch dataSourceType {
			case string(providers.PrometheusDataSource):
				if len(promKeys) != 0 {
					return fmt.Errorf("in replicas recommender, you need set prometheus config %v for history data provider", providers.PrometheusConfigKeys)
				}

				mustSetConfig := []string{"prometheus-address", "prometheus-auth-username", "prometheus-auth-password", "prometheus-auth-bearertoken"}
				if recommendation.SlicesContainSlice(promKeys, mustSetConfig) {
					return fmt.Errorf("in replicas recommender, you need set prometheus config %v for history data provider", mustSetConfig)
				}
				timeOut := 3 * time.Minute
				if value, ok := configSet["prometheus-timeout"]; ok {
					timeOut, _ = time.ParseDuration(value)
				}

				aliveTime := 60 * time.Second
				if value, ok := configSet["prometheus-keepalive"]; ok {
					aliveTime, _ = time.ParseDuration(value)
				}

				concurrency := 10
				if value, ok := configSet["prometheus-query-concurrency"]; ok {
					concurrency, _ = strconv.Atoi(value)
				}

				maxPoints := 11000
				if value, ok := configSet["prometheus-maxpoints"]; ok {
					maxPoints, _ = strconv.Atoi(value)
				}
				promConfig := providers.PromConfig{
					Address:            configSet["prometheus-address"],
					Timeout:            timeOut,
					KeepAlive:          aliveTime,
					InsecureSkipVerify: configSet["prometheus-insecure-skip-verify"] == "true",
					Auth: providers.ClientAuth{
						Username:    configSet["prometheus-auth-username"],
						Password:    configSet["prometheus-auth-password"],
						BearerToken: configSet["prometheus-auth-bearertoken"],
					},
					QueryConcurrency:            concurrency,
					BRateLimit:                  configSet["prometheus-bratelimit"] == "true",
					MaxPointsLimitPerTimeSeries: maxPoints,
				}
				promDataProvider, err := prom.NewProvider(&promConfig)
				if err != nil {
					return err
				}
				ctx.DataProviders = map[providers.DataSourceType]providers.Interface{
					providers.PrometheusDataSource: promDataProvider,
				}
			case string(providers.GrpcDataSource):
				// not support grpc yet
				if len(grpcKeys) != 0 {
					return fmt.Errorf("in replicas recommender, you need set grpc config %v for history data provider", providers.PrometheusConfigKeys)
				}

				timeOut := time.Minute
				if value, ok := configSet["grpc-ds-timeout"]; ok {
					timeOut, _ = time.ParseDuration(value)
				}

				address := "localhost:50051"
				if value, ok := configSet["grpc-ds-address"]; ok {
					address = value
				}

				grpcConfig := providers.GrpcConfig{
					Address: address,
					Timeout: timeOut,
				}
				grpcDataProvider := grpc.NewProvider(&grpcConfig)
				ctx.DataProviders = map[providers.DataSourceType]providers.Interface{
					providers.GrpcDataSource: grpcDataProvider,
				}
			default:
				return fmt.Errorf("replicas recommender only support %v and %v provider", providers.PrometheusDataSource, providers.GrpcDataSource)
			}
		}
	*/

	// 2. if not set data provider, will use default
	// do nothing
	return nil
}

func (br *BaseRecommender) CollectData(ctx *framework.RecommendationContext) error {
	return nil
}

func (br *BaseRecommender) PostProcessing(ctx *framework.RecommendationContext) error {
	return nil
}
