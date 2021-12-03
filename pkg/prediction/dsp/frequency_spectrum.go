package dsp

import (
	"math"
	"math/cmplx"
	"sort"
	"time"

	"github.com/mjibson/go-dsp/fft"
)

var PeriodicityAmplitudeThreshold = 0.

type FrequencySpectrum struct {
	Amplitudes  []float64
	Frequencies []float64
}

// FrequencySpectrum returns the frequency spectrum of the signal.
func (s *Signal) FrequencySpectrum() *FrequencySpectrum {
	// Use FFT to convert the signal from time to frequency domain
	fs := fft.FFTReal(s.Samples)

	sampleLength := float64(len(s.Samples))

	var amplitudes []float64
	for i := range fs {
		// Calculate the modulus since the result of FFT is an array of complex number with both real and imaginary parts
		amplitudes = append(amplitudes, cmplx.Abs(fs[i])/sampleLength)
	}
	spectrumLength := float64(len(amplitudes))

	//Use the first half slice since the spectrum is conjugate symmetric
	amplitudes = amplitudes[0 : len(amplitudes)/2]

	var frequencies []float64
	for i := range amplitudes {
		// Calculate which frequencies the spectrum contains
		frequencies = append(frequencies, float64(i)*s.SampleRate/spectrumLength)
	}

	return &FrequencySpectrum{
		// Ignore zero frequency term since it only affects the signal's relative position
		// on y-axis in time domain and has nothing to do with periodicity.
		Amplitudes:  amplitudes[1:],
		Frequencies: frequencies[1:],
	}
}

// Frequencies returns the signal frequency components in hertz in descending order.
func (s *Signal) Frequencies() []float64 {
	f := s.FrequencySpectrum()
	sort.Sort(sort.Reverse(f))
	//for i := 0; i < 20; i++ {
	//	klog.Infof("Cycle length: %f, Amplitude: %f", 1.0 / f.Frequencies[i], f.Amplitudes[i])
	//}
	//klog.Info()
	return f.Frequencies
}

// IsPeriodic checks whether the signal is periodic and its period is approximately
// equal to the given value
func (s *Signal) IsPeriodic(cycleDuration time.Duration) bool {
	//s.Frequencies()
	// The signal length must be at least double of the period
	si, m := s.Truncate(cycleDuration)
	if m < 2 {
		return false
	}

	secondsPerCycle := cycleDuration.Seconds()
	for _, freq := range si.Frequencies() {
		t := 1.0 / freq
		if t > secondsPerCycle {
			return false
		}
		epsilon := math.Abs(t-secondsPerCycle) / t
		if epsilon < 1e-3 {
			return true
		}
	}
	return false
}

func (f *FrequencySpectrum) Len() int {
	return len(f.Amplitudes)
}

func (f *FrequencySpectrum) Less(i, j int) bool {
	return f.Amplitudes[i] < f.Amplitudes[j]
}

func (f *FrequencySpectrum) Swap(i, j int) {
	f.Amplitudes[i], f.Amplitudes[j] = f.Amplitudes[j], f.Amplitudes[i]
	f.Frequencies[i], f.Frequencies[j] = f.Frequencies[j], f.Frequencies[i]
}
