package prom

import (
	gocontext "context"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	promapiv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"

	"github.com/gocrane/crane/pkg/common"
)

type QueryRangeFunc func(ctx gocontext.Context, maxPointsPerSeries int, queryResult model.Value, warnings promapiv1.Warnings, query string, r promapiv1.Range) (model.Value, promapiv1.Warnings, error)

type FakePromAPI struct {
	rangeFunc          QueryRangeFunc
	queryResult        model.Value
	maxPointsPerSeries int
	warnings           promapiv1.Warnings
	err                error
}

func (fpa *FakePromAPI) Alerts(ctx gocontext.Context) (promapiv1.AlertsResult, error) {
	return promapiv1.AlertsResult{}, nil
}

func (fpa *FakePromAPI) AlertManagers(ctx gocontext.Context) (promapiv1.AlertManagersResult, error) {
	return promapiv1.AlertManagersResult{}, nil
}
func (fpa *FakePromAPI) CleanTombstones(ctx gocontext.Context) error {
	return nil
}
func (fpa *FakePromAPI) Config(ctx gocontext.Context) (promapiv1.ConfigResult, error) {
	return promapiv1.ConfigResult{}, nil
}
func (fpa *FakePromAPI) DeleteSeries(ctx gocontext.Context, matches []string, startTime time.Time, endTime time.Time) error {
	return nil
}
func (fpa *FakePromAPI) Flags(ctx gocontext.Context) (promapiv1.FlagsResult, error) {
	return promapiv1.FlagsResult{}, nil
}
func (fpa *FakePromAPI) LabelNames(ctx gocontext.Context, matches []string, startTime time.Time, endTime time.Time) ([]string, promapiv1.Warnings, error) {
	return []string{}, promapiv1.Warnings{}, nil
}
func (fpa *FakePromAPI) LabelValues(ctx gocontext.Context, label string, matches []string, startTime time.Time, endTime time.Time) (model.LabelValues, promapiv1.Warnings, error) {
	return model.LabelValues{}, promapiv1.Warnings{}, nil
}
func (fpa *FakePromAPI) Query(ctx gocontext.Context, query string, ts time.Time) (model.Value, promapiv1.Warnings, error) {
	return fpa.queryResult, fpa.warnings, fpa.err
}

func (fpa *FakePromAPI) QueryRange(ctx gocontext.Context, query string, r promapiv1.Range) (model.Value, promapiv1.Warnings, error) {
	return fpa.rangeFunc(ctx, fpa.maxPointsPerSeries, fpa.queryResult, fpa.warnings, query, r)
}

func (fpa *FakePromAPI) QueryExemplars(ctx gocontext.Context, query string, startTime time.Time, endTime time.Time) ([]promapiv1.ExemplarQueryResult, error) {
	return []promapiv1.ExemplarQueryResult{}, nil
}
func (fpa *FakePromAPI) Buildinfo(ctx gocontext.Context) (promapiv1.BuildinfoResult, error) {
	return promapiv1.BuildinfoResult{}, nil

}
func (fpa *FakePromAPI) Runtimeinfo(ctx gocontext.Context) (promapiv1.RuntimeinfoResult, error) {
	return promapiv1.RuntimeinfoResult{}, nil
}
func (fpa *FakePromAPI) Series(ctx gocontext.Context, matches []string, startTime time.Time, endTime time.Time) ([]model.LabelSet, promapiv1.Warnings, error) {
	return []model.LabelSet{}, promapiv1.Warnings{}, nil
}
func (fpa *FakePromAPI) Snapshot(ctx gocontext.Context, skipHead bool) (promapiv1.SnapshotResult, error) {
	return promapiv1.SnapshotResult{}, nil
}
func (fpa *FakePromAPI) Rules(ctx gocontext.Context) (promapiv1.RulesResult, error) {
	return promapiv1.RulesResult{}, nil
}
func (fpa *FakePromAPI) Targets(ctx gocontext.Context) (promapiv1.TargetsResult, error) {
	return promapiv1.TargetsResult{}, nil
}

func (fpa *FakePromAPI) TargetsMetadata(ctx gocontext.Context, matchTarget string, metric string, limit string) ([]promapiv1.MetricMetadata, error) {
	return []promapiv1.MetricMetadata{}, nil
}

func (fpa *FakePromAPI) Metadata(ctx gocontext.Context, metric string, limit string) (map[string][]promapiv1.Metadata, error) {
	return map[string][]promapiv1.Metadata{}, nil
}
func (fpa *FakePromAPI) TSDB(ctx gocontext.Context) (promapiv1.TSDBResult, error) {
	return promapiv1.TSDBResult{}, nil
}

func NewFakeAPI(rangeFunc QueryRangeFunc, tsList []*common.TimeSeries, warnings []string, err error, maxPointsPerSeries int) *FakePromAPI {
	var matrix []*model.SampleStream
	for _, ts := range tsList {
		labels := make(map[model.LabelName]model.LabelValue)
		values := make([]model.SamplePair, 0)
		for _, l := range ts.Labels {
			labels[model.LabelName(l.Name)] = model.LabelValue(l.Value)
		}
		for _, sample := range ts.Samples {
			// sample.Timestamp is unix second unit. prometheus model.Time is millisecond unit.
			values = append(values, model.SamplePair{Timestamp: model.Time(sample.Timestamp * 1000), Value: model.SampleValue(sample.Value)})
		}
		ss := &model.SampleStream{}
		ss.Metric = labels
		ss.Values = values
		matrix = append(matrix, ss)
	}
	return &FakePromAPI{
		rangeFunc:          rangeFunc,
		queryResult:        model.Matrix(matrix),
		maxPointsPerSeries: maxPointsPerSeries,
		warnings:           warnings,
		err:                err,
	}
}

func NewFakeTimeSeries(labels []common.Label, window promapiv1.Range) *common.TimeSeries {
	ts := common.NewTimeSeries()
	ts.SetLabels(labels)
	samples := []common.Sample{}
	for s := window.Start; s.Before(window.End) || s.Equal(window.End); s = s.Add(window.Step) {
		samples = append(samples, common.Sample{Timestamp: s.Unix(), Value: 0})
	}
	ts.Samples = samples
	return ts
}

func IsSelected(sampleStream *model.SampleStream, querySelector model.LabelSet) bool {
	// query selector, all satisfy if queryLabelSet is null, else each k v in queryLabelSet must in sampleStream.Metric
	for k, v := range querySelector {
		if sv, ok := sampleStream.Metric[k]; !ok || sv != v {
			return false
		}
	}
	return true
}

func defaultFakeQueryRange(ctx gocontext.Context, maxPointsPerSeries int, queryResult model.Value, warnings promapiv1.Warnings, query string, r promapiv1.Range) (model.Value, promapiv1.Warnings, error) {
	queryLabels := Query2Labels(query)
	queryLabelSet := make(model.LabelSet)
	for _, label := range queryLabels {
		queryLabelSet[model.LabelName(label.Name)] = model.LabelValue(label.Value)
	}
	typeValue := queryResult.Type()
	switch typeValue {
	case model.ValMatrix:
		var results []*model.SampleStream
		if matrix, ok := queryResult.(model.Matrix); ok {
			for _, sampleStream := range matrix {
				if sampleStream == nil {
					continue
				}

				if !IsSelected(sampleStream, queryLabelSet) {
					continue
				}

				values := make([]model.SamplePair, 0)
				for s := r.Start; s.Before(r.End) || s.Equal(r.End); s = s.Add(r.Step) {
					// sampling
					for _, pair := range sampleStream.Values {
						// left close, right open
						if pair.Timestamp.Unix() >= s.Unix() && pair.Timestamp.Unix() <= r.End.Unix() {
							values = append(values, model.SamplePair{Timestamp: pair.Timestamp, Value: pair.Value})
							break
						}
					}
				}

				fmt.Println(sampleStream.Metric, r, r.Start.Unix(), r.End.Unix(), values)
				ss := &model.SampleStream{}
				ss.Metric = sampleStream.Metric
				ss.Values = values
				if len(ss.Values) > maxPointsPerSeries {
					return nil, warnings, fmt.Errorf("exceeded maximum resolution of %v points per timeseries. Try decreasing the query resolution (?step=XX)", maxPointsPerSeries)
				}
				results = append(results, ss)
			}
			return model.Matrix(results), warnings, nil
		} else {
			return model.Matrix(results), warnings, fmt.Errorf("prometheus value type is %v, but assert failed", typeValue)
		}

	case model.ValVector:
		return queryResult, warnings, fmt.Errorf("not support for vector when use QueryRange")
	case model.ValScalar:
		return queryResult, warnings, fmt.Errorf("not support for scalar when use timeseries")
	case model.ValString:
		return queryResult, warnings, fmt.Errorf("not support for string when use timeseries")
	case model.ValNone:
		return queryResult, warnings, fmt.Errorf("prometheus return value type is none")
	}
	return queryResult, warnings, fmt.Errorf("prometheus return value type is none")
}

//func TestMain(m *testing.M) {
//	flag.Set("alsologtostderr", "false")
//	flag.Set("log_dir", "/tmp")
//	flag.Set("v", "5")
//	flag.Parse()
//
//	ret := m.Run()
//	os.Exit(ret)
//}

func Labels2Query(labels []common.Label) string {
	var res []string
	for _, label := range labels {
		res = append(res, label.String())
	}
	sort.Strings(res)
	return strings.Join(res, ",")
}

func Query2Labels(query string) []common.Label {
	var labels []common.Label
	labelStrList := strings.Split(query, ",")
	for _, labelStr := range labelStrList {
		splits := strings.Split(labelStr, "=")
		if len(splits) >= 2 {
			labels = append(labels, common.Label{Name: strings.Trim(splits[0], " "), Value: strings.Trim(splits[1], " ")})
		}
	}
	return labels
}

func TestMergeSortedTimeSeries(t *testing.T) {

	now := time.Now()
	end := now.Truncate(time.Minute)
	start := end.Add(-20 * time.Minute)
	step := time.Minute

	mid := end.Add(-10 * time.Minute)

	tc1_ts1 := NewFakeTimeSeries([]common.Label{{Name: "ts", Value: "1"}}, promapiv1.Range{Start: start, End: mid, Step: step})
	tc1_ts2 := NewFakeTimeSeries([]common.Label{{Name: "ts", Value: "1"}}, promapiv1.Range{Start: mid, End: end, Step: step})
	tc1_wantTs := NewFakeTimeSeries([]common.Label{{Name: "ts", Value: "1"}}, promapiv1.Range{Start: start, End: end, Step: step})

	end2_1 := now.Truncate(time.Minute)
	start2_1 := end.Add(-20 * time.Minute)
	end2_2 := end.Add(-10 * time.Minute)
	start2_2 := end.Add(-30 * time.Minute)
	step2 := time.Minute

	tc2_ts1 := NewFakeTimeSeries([]common.Label{{Name: "ts", Value: "1"}}, promapiv1.Range{Start: start2_1, End: end2_1, Step: step2})
	tc2_ts2 := NewFakeTimeSeries([]common.Label{{Name: "ts", Value: "1"}}, promapiv1.Range{Start: start2_2, End: end2_2, Step: step2})
	tc2_wantTs := NewFakeTimeSeries([]common.Label{{Name: "ts", Value: "1"}}, promapiv1.Range{Start: start2_2, End: end2_1, Step: step2})

	testCases := []struct {
		desc               string
		ts1                *common.TimeSeries
		ts2                *common.TimeSeries
		expectedTimeSeries *common.TimeSeries
	}{
		{
			desc:               "tc1",
			ts1:                tc1_ts1,
			ts2:                tc1_ts2,
			expectedTimeSeries: tc1_wantTs,
		},
		{
			desc:               "tc2",
			ts1:                tc2_ts1,
			ts2:                tc2_ts2,
			expectedTimeSeries: tc2_wantTs,
		},
	}

	for _, tc := range testCases {
		goTs := MergeSortedTimeSeries(tc.ts1, tc.ts2)
		if !reflect.DeepEqual(goTs, tc.expectedTimeSeries) {
			fmt.Println("got:")
			PrintTs(goTs)
			fmt.Println("expected:")
			PrintTs(tc.expectedTimeSeries)
			t.Fatalf("tc %v failed", tc.desc)
		}
	}
}

func TestQueryRangeSync(t *testing.T) {
	now := time.Now()
	end := now.Truncate(time.Minute)
	start := end.Add(-20 * time.Minute)
	step := time.Minute
	origin_ts1 := NewFakeTimeSeries([]common.Label{{Name: "ts", Value: "1"}}, promapiv1.Range{Start: start, End: now, Step: step})
	origin_ts2 := NewFakeTimeSeries([]common.Label{{Name: "ts", Value: "2"}}, promapiv1.Range{Start: start, End: now, Step: step})

	maxPointsPerSeries := 10
	fakeApi := NewFakeAPI(defaultFakeQueryRange, []*common.TimeSeries{origin_ts1, origin_ts2}, promapiv1.Warnings{}, nil, maxPointsPerSeries)

	end1 := now.Truncate(time.Minute)
	start1 := end1.Add(-21 * time.Minute)
	step1 := time.Minute
	tc1_wantTs1 := NewFakeTimeSeries([]common.Label{{Name: "ts", Value: "1"}}, promapiv1.Range{Start: start, End: end1, Step: step1})
	tc1_wantTs2 := NewFakeTimeSeries([]common.Label{{Name: "ts", Value: "2"}}, promapiv1.Range{Start: start, End: end1, Step: step1})

	end2 := now.Truncate(time.Minute)
	start2 := end2.Add(-9 * time.Minute)
	step2 := time.Minute
	tc2_wantTs1 := NewFakeTimeSeries([]common.Label{{Name: "ts", Value: "1"}}, promapiv1.Range{Start: start2, End: end2, Step: step2})
	tc2_wantTs2 := NewFakeTimeSeries([]common.Label{{Name: "ts", Value: "2"}}, promapiv1.Range{Start: start2, End: end2, Step: step2})

	end3 := now.Truncate(time.Minute)
	start3 := end3.Add(-10 * time.Minute)
	step3 := time.Minute
	tc3_wantTs1 := NewFakeTimeSeries([]common.Label{{Name: "ts", Value: "1"}}, promapiv1.Range{Start: start3, End: end3, Step: step3})
	tc3_wantTs2 := NewFakeTimeSeries([]common.Label{{Name: "ts", Value: "2"}}, promapiv1.Range{Start: start3, End: end3, Step: step3})

	end4 := now.Truncate(time.Minute)
	start4 := end4.Add(-11 * time.Minute)
	step4 := time.Minute
	tc4_wantTs := NewFakeTimeSeries([]common.Label{{Name: "ts", Value: "1"}}, promapiv1.Range{Start: start4, End: end4, Step: step4})

	end5 := now.Truncate(time.Minute)
	start5 := end5.Add(-10 * time.Minute)
	step5 := time.Minute
	tc5_wantTs := NewFakeTimeSeries([]common.Label{{Name: "ts", Value: "2"}}, promapiv1.Range{Start: start5, End: end5, Step: step5})

	end6 := now.Truncate(time.Minute)
	start6 := end6.Add(-11 * time.Minute)
	step6 := time.Minute
	tc6_wantTs := NewFakeTimeSeries([]common.Label{{Name: "ts", Value: "1"}}, promapiv1.Range{Start: start6, End: end6, Step: step6})

	end7 := now.Truncate(time.Minute)
	start7 := end7.Add(-12 * time.Minute)
	step7 := time.Minute
	tc7_wantTs := NewFakeTimeSeries([]common.Label{{Name: "ts", Value: "2"}}, promapiv1.Range{Start: start7, End: end7, Step: step7})

	//now := time.Now()
	//end := now.Truncate(time.Minute)
	//start := end.Add(-24*16*time.Hour)
	//step := time.Minute
	//origin_ts1 := NewFakeTimeSeries([]common.Label{{Name: "ts", Value: "1"}}, promapiv1.Range{Start: start,End: now,Step: step})
	//origin_ts2 := NewFakeTimeSeries([]common.Label{{Name: "ts", Value: "2"}}, promapiv1.Range{Start: start,End: now,Step: step})
	//
	//maxPointsPerSeries := 11000
	//fakeApi := NewFakeAPI(defaultFakeQueryRange, []*common.TimeSeries{origin_ts1, origin_ts2}, promapiv1.Warnings{}, nil, maxPointsPerSeries)

	//end1 := now.Truncate(time.Minute)
	//start1 := end1.Add(-24*17*time.Hour)
	//step1 := time.Minute
	//tc1_wantTs1 := NewFakeTimeSeries([]common.Label{{Name: "ts", Value: "1"}}, promapiv1.Range{Start: start1,End: end1,Step: step1})
	//tc1_wantTs2 := NewFakeTimeSeries([]common.Label{{Name: "ts", Value: "2"}}, promapiv1.Range{Start: start1,End: end1,Step: step1})
	//
	//end2 := now.Truncate(time.Minute)
	//start2 := end2.Add(-24*15*time.Hour)
	//step2 := time.Minute
	//tc2_wantTs1 := NewFakeTimeSeries([]common.Label{{Name: "ts", Value: "1"}}, promapiv1.Range{Start: start2,End: end2,Step: step2})
	//tc2_wantTs2 := NewFakeTimeSeries([]common.Label{{Name: "ts", Value: "2"}}, promapiv1.Range{Start: start2,End: end2,Step: step2})
	//
	//
	//end3 := now.Truncate(time.Minute)
	//start3 := end3.Add(-24*16*time.Hour)
	//step3 := time.Minute
	//tc3_wantTs1 := NewFakeTimeSeries([]common.Label{{Name: "ts", Value: "1"}}, promapiv1.Range{Start: start3,End: end3,Step: step3})
	//tc3_wantTs2 := NewFakeTimeSeries([]common.Label{{Name: "ts", Value: "2"}}, promapiv1.Range{Start: start3,End: end3,Step: step3})
	//
	//
	//end4 := now.Truncate(time.Minute)
	//start4 := end4.Add(-10080*time.Minute)
	//step4 := time.Minute
	//tc4_wantTs := NewFakeTimeSeries([]common.Label{{Name: "ts", Value: "1"}}, promapiv1.Range{Start: start4,End: end4,Step: step4})
	//
	//end5 := now.Truncate(time.Minute)
	//start5 := end5.Add(-11000*time.Minute)
	//step5 := time.Minute
	//tc5_wantTs := NewFakeTimeSeries([]common.Label{{Name: "ts", Value: "2"}}, promapiv1.Range{Start: start5,End: end5,Step: step5})
	//
	//
	//end6 := now.Truncate(time.Minute)
	//start6 := now.Add(-10999*time.Minute)
	//step6 := time.Minute
	//tc6_wantTs := NewFakeTimeSeries([]common.Label{{Name: "ts", Value: "1"}}, promapiv1.Range{Start: start6,End: end6,Step: step6})
	//
	//end7 := now.Truncate(time.Minute)
	//start7 := end7.Add(-11001*time.Minute)
	//step7 := time.Minute
	//tc7_wantTs := NewFakeTimeSeries([]common.Label{{Name: "ts", Value: "2"}}, promapiv1.Range{Start: start7,End: end7,Step: step7})
	//

	testCases := []struct {
		desc               string
		api                *FakePromAPI
		start, end         time.Time
		step               time.Duration
		quetyLabels        []common.Label
		expectedTimeSeries []*common.TimeSeries
		expectedError      error
		expectedWarnings   promapiv1.Warnings
	}{
		{
			desc:               "1.",
			api:                fakeApi,
			start:              start1,
			end:                end1,
			step:               step1,
			quetyLabels:        []common.Label{},
			expectedTimeSeries: []*common.TimeSeries{tc1_wantTs1, tc1_wantTs2},
			expectedError:      nil,
		},
		{
			desc:               "2.",
			api:                fakeApi,
			start:              start2,
			end:                end2,
			step:               step2,
			quetyLabels:        []common.Label{},
			expectedTimeSeries: []*common.TimeSeries{tc2_wantTs1, tc2_wantTs2},
			expectedError:      nil,
		},
		{
			desc:               "3.",
			api:                fakeApi,
			start:              start3,
			end:                end3,
			step:               step3,
			quetyLabels:        []common.Label{},
			expectedTimeSeries: []*common.TimeSeries{tc3_wantTs1, tc3_wantTs2},
			expectedError:      nil,
		},
		{
			desc:               "4.",
			api:                fakeApi,
			start:              start4,
			end:                end4,
			step:               step4,
			quetyLabels:        []common.Label{{Name: "ts", Value: "1"}},
			expectedTimeSeries: []*common.TimeSeries{tc4_wantTs},
			expectedError:      nil,
		},
		{
			desc:               "5.",
			api:                fakeApi,
			start:              start5,
			end:                end5,
			step:               step5,
			quetyLabels:        []common.Label{{Name: "ts", Value: "2"}},
			expectedTimeSeries: []*common.TimeSeries{tc5_wantTs},
			expectedError:      nil,
		},
		{
			desc:               "6.",
			api:                fakeApi,
			start:              start6,
			end:                end6,
			step:               step6,
			quetyLabels:        []common.Label{{Name: "ts", Value: "1"}},
			expectedTimeSeries: []*common.TimeSeries{tc6_wantTs},
			expectedError:      nil,
		},
		{
			desc:               "7.",
			api:                fakeApi,
			start:              start7,
			end:                end7,
			step:               step7,
			quetyLabels:        []common.Label{{Name: "ts", Value: "2"}},
			expectedTimeSeries: []*common.TimeSeries{tc7_wantTs},
			expectedError:      nil,
		},
	}
	for _, tc := range testCases {
		ctx := NewContextByAPI(tc.api, tc.api.maxPointsPerSeries)
		gotTsList, gotErr := ctx.QueryRangeSync(gocontext.TODO(), Labels2Query(tc.quetyLabels), tc.start, tc.end, tc.step)
		if !EqualTimeSeries(gotTsList, tc.expectedTimeSeries) {
			fmt.Println(tc.api.queryResult.String())
			t.Logf("tc %v start: %v, end: %v, step: %v", tc.desc, tc.start.Unix(), tc.end.Unix(), tc.step)
			fmt.Println("got:")
			PrintTsList(gotTsList)
			fmt.Println("expected:")
			PrintTsList(tc.expectedTimeSeries)
			t.Fatalf("tc %v failed EqualTimeSeries", tc.desc)

		}
		if !reflect.DeepEqual(gotErr, tc.expectedError) {
			t.Fatalf("tc %v failed, gotErr: %v, expectedError: %v", tc.desc, gotErr, tc.expectedError)
		}
	}
}

func LabelsKey(labels []common.Label) string {
	var res []string
	for _, label := range labels {
		res = append(res, fmt.Sprintf("%v=%v", label.Name, label.Value))
	}
	sort.Strings(res)
	return strings.Join(res, ",")
}

func PrintTs(ts *common.TimeSeries) {
	fmt.Printf("key: %v, samples: %v\n", Labels2Query(ts.Labels), ts.Samples)
}

func PrintTsList(tsList []*common.TimeSeries) {
	fmt.Println("------------------------")
	for _, ts := range tsList {
		fmt.Printf("key: %v, len: %v, samples: %v\n", Labels2Query(ts.Labels), len(ts.Samples), ts.Samples)
	}
}

func EqualTimeSeries(tsList1, tsList2 []*common.TimeSeries) bool {
	tsMap1 := make(map[string]*common.TimeSeries)
	if len(tsList1) != len(tsList2) {
		return false
	}
	for _, ts := range tsList1 {
		tsMap1[LabelsKey(ts.Labels)] = ts
	}

	for _, ts2 := range tsList2 {
		if ts1, ok := tsMap1[LabelsKey(ts2.Labels)]; ok {
			if !reflect.DeepEqual(ts1, ts2) {
				return false
			}
		} else {
			return false
		}
	}
	return true
}
