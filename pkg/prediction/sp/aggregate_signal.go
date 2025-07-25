package sp

import (
	"k8s.io/klog/v2"
	"math"
	"time"

	vpa "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/recommender/util"

	"github.com/gocrane/crane/pkg/common"
)

// 样本信息结构
type sampleInfo struct {
	time   time.Time
	value  float64
	weight float64
}

type aggregateSignal struct {
	histogram         vpa.Histogram
	firstSampleTime   time.Time
	lastSampleTime    time.Time
	minSampleWeight   float64
	totalSamplesCount int
	sampleInterval    time.Duration
	creationTime      time.Time
	labels            []common.Label
	historyLength     time.Duration
	// 样本队列（按时间排序，最早的在最前面）,默认按照时间加入，所以不需要排序
	sampleQueue []sampleInfo
}

func (a *aggregateSignal) addSample(sampleTime time.Time, sampleValue float64) {
	// 计算样本权重，突出大值的影响，覆盖毛刺情况
	weight := math.Max(a.minSampleWeight, sampleValue)

	// 添加新样本到直方图
	a.histogram.AddSample(sampleValue, weight, sampleTime)

	// 将新样本插入队列（按时间排序）
	newSample := sampleInfo{
		time:   sampleTime,
		value:  sampleValue,
		weight: weight,
	}

	a.sampleQueue = append(a.sampleQueue, newSample)

	// 更新最后样本时间
	if a.lastSampleTime.Before(sampleTime) {
		a.lastSampleTime = sampleTime
	}

	// 更新最早样本时间
	if a.firstSampleTime.IsZero() || a.firstSampleTime.After(sampleTime) {
		a.firstSampleTime = sampleTime
	}

	// 增加样本计数
	a.totalSamplesCount++

	// 计算过期时间点（当前时间 - 历史长度）
	expirationTime := time.Now().Add(-a.historyLength)

	// 删除过期样本（从队列头部开始，直到遇到未过期的样本）
	removedCount := 0
	for i := 0; i < len(a.sampleQueue); i++ {
		if a.sampleQueue[i].time.Before(expirationTime) {
			// 从直方图中移除过期样本
			a.histogram.SubtractSample(
				a.sampleQueue[i].value,
				a.sampleQueue[i].weight,
				a.sampleQueue[i].time,
			)
			removedCount++
		} else {
			// 遇到第一个未过期样本，停止删除
			break
		}
	}

	// 更新队列和计数
	if removedCount > 0 {
		klog.V(4).Infof("Removed %v expired samples from the queue.", removedCount)
		a.sampleQueue = a.sampleQueue[removedCount:]
		a.totalSamplesCount -= removedCount
	}

	// 更新最早样本时间
	if len(a.sampleQueue) > 0 {
		a.firstSampleTime = a.sampleQueue[0].time
	} else {
		a.firstSampleTime = time.Time{}
	}
}

// largest is 290 years, so it can not be overflow
func (a *aggregateSignal) GetAggregationWindowLength() time.Duration {
	return time.Duration(a.totalSamplesCount) * a.sampleInterval
}

func newAggregateSignal(c *internalConfig) *aggregateSignal {
	var histogram vpa.Histogram
	if c.histogramDecayHalfLife == 0 {
		histogram = vpa.NewHistogram(c.histogramOptions)
	} else {
		histogram = vpa.NewDecayingHistogram(c.histogramOptions, c.histogramDecayHalfLife)
	}
	return &aggregateSignal{
		histogram:       histogram,
		minSampleWeight: c.minSampleWeight,
		creationTime:    time.Now(),
		sampleInterval:  c.sampleInterval,
		historyLength:   c.historyLength,
		sampleQueue:     []sampleInfo{}, // 初始化样本队列
	}
}
