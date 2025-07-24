package dsp

import (
	"fmt"
	"math/cmplx"
	"sort"
	"time"

	"github.com/mjibson/go-dsp/fft"
)

const (
	defaultHighFrequencyThreshold = 1.0 / (60.0 * 60.0)
	defaultLowAmplitudeThreshold  = 1.0
	defaultMinNumOfSpectrumItems  = 3
	defaultMaxNumOfSpectrumItems  = 100
	defaultFFTMarginFraction      = 0.0
	defaultMaxValueMarginFraction = 0.0
	defaultFFTMinValue            = 0.01
)

type Estimator interface {
	GetEstimation(signal *Signal, periodLength time.Duration) *Signal
	String() string
}

func NewMaxValueEstimator(marginFraction float64) Estimator {
	return &maxValueEstimator{marginFraction}
}

func NewFFTEstimator(minNumOfSpectrumItems, maxNumOfSpectrumItems int, highFrequencyThreshold, lowAmplitudeThreshold, marginFraction float64) Estimator {
	return &fftEstimator{
		minNumOfSpectrumItems:  minNumOfSpectrumItems,
		maxNumOfSpectrumItems:  maxNumOfSpectrumItems,
		highFrequencyThreshold: highFrequencyThreshold,
		lowAmplitudeThreshold:  lowAmplitudeThreshold,
		marginFraction:         marginFraction,
	}
}

type maxValueEstimator struct {
	marginFraction float64
}

type fftEstimator struct {
	minNumOfSpectrumItems  int
	maxNumOfSpectrumItems  int
	highFrequencyThreshold float64
	lowAmplitudeThreshold  float64
	marginFraction         float64
}

func (m *maxValueEstimator) GetEstimation(signal *Signal, periodLength time.Duration) *Signal {
	nSamplesPerPeriod := int(periodLength.Seconds() * signal.SampleRate)
	estimation := make([]float64, 0, nSamplesPerPeriod)

	nSamples := len(signal.Samples)
	nPeriods := nSamples / nSamplesPerPeriod

	for i := nSamples - nSamplesPerPeriod; i < nSamples; i++ {
		maxValue := signal.Samples[i]
		for j := 1; j < nPeriods; j++ {
			if maxValue < signal.Samples[i-nSamplesPerPeriod*j] {
				maxValue = signal.Samples[i-nSamplesPerPeriod*j]
			}
		}
		estimation = append(estimation, maxValue*(1.0+m.marginFraction))
	}

	return &Signal{
		SampleRate: signal.SampleRate,
		Samples:    estimation,
	}
}

func (m *maxValueEstimator) String() string {
	marginFraction := m.marginFraction
	if marginFraction == 0.0 {
		marginFraction = defaultMaxValueMarginFraction
	}
	return fmt.Sprintf("Max Value Estimator {marginFraction: %f}", marginFraction)
}

func (f *fftEstimator) GetEstimation(signal *Signal, periodLength time.Duration) *Signal {
	minNumOfSpectrumItems, maxNumOfSpectrumItems := f.minNumOfSpectrumItems, f.maxNumOfSpectrumItems
	if minNumOfSpectrumItems == 0 {
		minNumOfSpectrumItems = defaultMinNumOfSpectrumItems
	}
	if maxNumOfSpectrumItems == 0 {
		maxNumOfSpectrumItems = defaultMaxNumOfSpectrumItems
	}

	highFrequencyThreshold, lowAmplitudeThreshold, marginFraction := f.highFrequencyThreshold, f.lowAmplitudeThreshold, f.marginFraction
	if f.highFrequencyThreshold == 0 {
		highFrequencyThreshold = defaultHighFrequencyThreshold
	}
	if f.lowAmplitudeThreshold == 0 {
		lowAmplitudeThreshold = defaultLowAmplitudeThreshold
	}
	if f.marginFraction == 0 {
		marginFraction = defaultFFTMarginFraction
	}

	X := fft.FFTReal(signal.Samples)

	sampleLength := float64(len(signal.Samples))

	var frequencies []float64
	for k := range X {
		// Calculate which frequencies the spectrum contains
		frequencies = append(frequencies, float64(k)*signal.SampleRate/sampleLength)
	}

	var amplitudes sort.Float64Slice
	for k := range X {
		amplitudes = append(amplitudes, cmplx.Abs(X[k])/sampleLength)
	}
	//Use the first half slice since the spectrum is conjugate symmetric
	amplitudes = amplitudes[1 : len(amplitudes)/2]

	// Sort the amplitudes in descending order
	sort.Sort(sort.Reverse(amplitudes))

	var minAmplitude float64
	if len(amplitudes) >= maxNumOfSpectrumItems {
		minAmplitude = amplitudes[maxNumOfSpectrumItems-1]
	} else {
		minAmplitude = amplitudes[len(amplitudes)-1]
	}
	if minAmplitude < lowAmplitudeThreshold {
		minAmplitude = lowAmplitudeThreshold
	}
	if len(amplitudes) >= minNumOfSpectrumItems && amplitudes[minNumOfSpectrumItems-1] < minAmplitude {
		minAmplitude = amplitudes[minNumOfSpectrumItems-1]
	}

	for k := range X {
		// Calculate the modulus since the result of FFT is an array of complex number with both real and imaginary parts
		amplitude := cmplx.Abs(X[k]) / sampleLength

		// Filter out the noise, which is of high frequency and low amplitude
		if amplitude < minAmplitude && frequencies[k] > highFrequencyThreshold {
			X[k] = 0
		}
	}

	x := fft.IFFT(X)
	nSamples := len(x)
	nSamplesPerPeriod := int(periodLength.Seconds() * signal.SampleRate)

	samples := make([]float64, nSamplesPerPeriod)
	for i := nSamples - nSamplesPerPeriod; i < nSamples; i++ {
		a := real(x[i])
		if a <= 0.0 {
			a = defaultFFTMinValue
		}
		samples[i+nSamplesPerPeriod-nSamples] = a * (1.0 + marginFraction)
	}

	return &Signal{
		SampleRate: signal.SampleRate,
		Samples:    samples,
	}
}

func (f *fftEstimator) String() string {
	minNumOfSpectrumItems, maxNumOfSpectrumItems := f.minNumOfSpectrumItems, f.maxNumOfSpectrumItems
	if minNumOfSpectrumItems == 0 {
		minNumOfSpectrumItems = defaultMinNumOfSpectrumItems
	}
	if maxNumOfSpectrumItems == 0 {
		maxNumOfSpectrumItems = defaultMaxNumOfSpectrumItems
	}

	highFrequencyThreshold, lowAmplitudeThreshold, marginFraction := f.highFrequencyThreshold, f.lowAmplitudeThreshold, f.marginFraction
	if highFrequencyThreshold == 0.0 {
		highFrequencyThreshold = defaultHighFrequencyThreshold
	}
	if lowAmplitudeThreshold == 0.0 {
		lowAmplitudeThreshold = defaultLowAmplitudeThreshold
	}
	if marginFraction == 0.0 {
		marginFraction = defaultFFTMarginFraction
	}
	return fmt.Sprintf("FFT Estimator {minNumOfSpectrumItems: %d, maxNumOfSpectrumItems: %d, highFrequencyThreshold: %f, lowAmplitudeThreshold: %f, marginFraction: %f}",
		minNumOfSpectrumItems, maxNumOfSpectrumItems, highFrequencyThreshold, lowAmplitudeThreshold, marginFraction)
}
