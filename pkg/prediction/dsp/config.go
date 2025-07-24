package dsp

import (
	"fmt"
	"time"

	"github.com/gocrane/api/prediction/v1alpha1"

	"github.com/gocrane/crane/pkg/utils"
)

var defaultInternalConfig = internalConfig{
	historyResolution: time.Minute,
	historyDuration:   time.Hour * 24 * 15,
	estimators:        defaultEstimators,
}

var defaultEstimators = []Estimator{
	&fftEstimator{minNumOfSpectrumItems: 3, lowAmplitudeThreshold: 1.0, marginFraction: 0.01},
	&fftEstimator{minNumOfSpectrumItems: 3, lowAmplitudeThreshold: 1.0, marginFraction: 0.10},
	&fftEstimator{minNumOfSpectrumItems: 3, lowAmplitudeThreshold: 1.0, marginFraction: 0.15},
	&fftEstimator{minNumOfSpectrumItems: 3, lowAmplitudeThreshold: 1.0, marginFraction: 0.20},
	&fftEstimator{minNumOfSpectrumItems: 50, lowAmplitudeThreshold: 0.05, marginFraction: 0.01},
	&fftEstimator{minNumOfSpectrumItems: 50, lowAmplitudeThreshold: 0.05, marginFraction: 0.10},
	&fftEstimator{minNumOfSpectrumItems: 50, lowAmplitudeThreshold: 0.05, marginFraction: 0.15},
	&fftEstimator{minNumOfSpectrumItems: 50, lowAmplitudeThreshold: 0.05, marginFraction: 0.20},
}

type internalConfig struct {
	historyResolution time.Duration
	historyDuration   time.Duration
	estimators        []Estimator
}

func (i internalConfig) String() string {
	return fmt.Sprintf("DSP internal Config: {historyResolution: %s, historyDuration: %v, estimators: %v",
		i.historyResolution.String(), i.historyDuration.String(), i.estimators)
}

func makeInternalConfig(d *v1alpha1.DSP) (*internalConfig, error) {
	historyResolution, err := utils.ParseDuration(d.SampleInterval)
	if err != nil {
		return nil, err
	}
	if historyResolution > time.Hour {
		return nil, fmt.Errorf("historyResolution is too long")
	}

	historyDuration, err := utils.ParseDuration(d.HistoryLength)
	if err != nil {
		return nil, err
	}
	if historyDuration < time.Hour*48 {
		return nil, fmt.Errorf("historyDuration is too short")
	}

	// parse estimators
	var estimators []Estimator

	for _, e := range d.Estimators.MaxValueEstimators {
		marginFraction, err := utils.ParseFloat(e.MarginFraction, defaultMaxValueMarginFraction)
		if err != nil {
			return nil, err
		}
		estimators = append(estimators, &maxValueEstimator{marginFraction})
	}

	for _, e := range d.Estimators.FFTEstimators {
		marginFraction, err := utils.ParseFloat(e.MarginFraction, defaultFFTMarginFraction)
		if err != nil {
			return nil, err
		}

		highFrequencyThreshold, err := utils.ParseFloat(e.HighFrequencyThreshold, defaultHighFrequencyThreshold)
		if err != nil {
			return nil, err
		}

		lowAmplitudeThreshold, err := utils.ParseFloat(e.LowAmplitudeThreshold, defaultLowAmplitudeThreshold)
		if err != nil {
			return nil, err
		}

		maxNumOfSpectrumItems := defaultMaxNumOfSpectrumItems
		if e.MaxNumOfSpectrumItems != nil {
			maxNumOfSpectrumItems = int(*e.MaxNumOfSpectrumItems)
		}

		minNumOfSpectrumItems := defaultMinNumOfSpectrumItems
		if e.MinNumOfSpectrumItems != nil {
			minNumOfSpectrumItems = int(*e.MinNumOfSpectrumItems)
		}

		estimators = append(estimators, &fftEstimator{
			minNumOfSpectrumItems,
			maxNumOfSpectrumItems,
			highFrequencyThreshold,
			lowAmplitudeThreshold,
			marginFraction,
		})
	}

	if len(estimators) == 0 {
		estimators = defaultEstimators
	}

	return &internalConfig{historyResolution, historyDuration, estimators}, nil
}
