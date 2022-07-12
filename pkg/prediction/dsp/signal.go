package dsp

import (
	"fmt"
	"math/cmplx"
	"math/rand"
	"sort"
	"time"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-echarts/go-echarts/v2/types"
	"github.com/mjibson/go-dsp/fft"
)

var (
	// MaxAutoCorrelationPeakSearchIntervalSeconds is the maximum search interval for validating if
	// a periodicity hint is a peak of the ACF.
	MaxAutoCorrelationPeakSearchIntervalSeconds = (time.Hour * 4).Seconds()
)

// Signal represents a discrete signal.
type Signal struct {
	// SampleRate is the sampling rate in hertz
	SampleRate float64
	// Samples store all samples
	Samples []float64
}

// Truncate truncates the signal to a length of multiple of d.
func (s *Signal) Truncate(d time.Duration) (*Signal, int /*multiple*/) {
	if s.Duration() < d.Seconds() {
		return nil, 0
	}

	n := int(d.Seconds() * s.SampleRate)
	m := 0
	i := len(s.Samples)
	for i-n >= 0 {
		i -= n
		m++
	}

	return &Signal{
		SampleRate: s.SampleRate,
		Samples:    s.Samples[i:],
	}, m
}

// Num returns the number of samples in the signal.
func (s *Signal) Num() int {
	return len(s.Samples)
}

// Duration returns the signal duration in seconds.
func (s *Signal) Duration() float64 {
	duration := float64(len(s.Samples)) / s.SampleRate
	return duration
}

// Min returns the minimum sample value.
func (s *Signal) Min() float64 {
	if len(s.Samples) == 0 {
		return 0
	}
	min := s.Samples[0]
	for i := 1; i < len(s.Samples); i++ {
		if s.Samples[i] < min {
			min = s.Samples[i]
		}
	}
	return min
}

// Max returns the maximum sample value.
func (s *Signal) Max() float64 {
	if len(s.Samples) == 0 {
		return 0
	}
	max := s.Samples[0]
	for i := 1; i < len(s.Samples); i++ {
		if s.Samples[i] > max {
			max = s.Samples[i]
		}
	}
	return max
}

// Normalize normalizes the signal between -1 and 1 and return a new signal instance.
func (s *Signal) Normalize() (*Signal, error) {
	if len(s.Samples) == 0 {
		return &Signal{
			SampleRate: s.SampleRate,
			Samples:    s.Samples,
		}, nil
	}

	min := s.Min()
	max := s.Max()

	if min == max {
		return nil, fmt.Errorf("cannot normalize signal")
	}

	normalized := make([]float64, len(s.Samples))

	for i := 0; i < len(s.Samples); i++ {
		normalized[i] = 2*((s.Samples[i]-min)/(max-min)) - 1
	}

	return &Signal{
		SampleRate: s.SampleRate,
		Samples:    normalized,
	}, nil
}

// Denormalize denormalizes the signal between min and max.
func (s *Signal) Denormalize(min, max float64) (*Signal, error) {
	if !(min < max) {
		return nil, fmt.Errorf("cannot denormalize signal")
	}

	if len(s.Samples) < 2 {
		return s, nil
	}

	denormalized := make([]float64, len(s.Samples))

	for i := 0; i < len(s.Samples); i++ {
		denormalized[i] = (s.Samples[i]+1)/2*(max-min) + min
	}
	return &Signal{
		SampleRate: s.SampleRate,
		Samples:    denormalized,
	}, nil
}

// Filter filters out frequency components whose amplitudes are less than the threshold and returns a new signal
func (s *Signal) Filter(threshold float64) *Signal {
	X := fft.FFTReal(s.Samples)
	sampleLength := float64(len(s.Samples))

	var frequencies []float64
	for k := range X {
		// Calculate which frequencies the spectrum contains
		frequencies = append(frequencies, float64(k)*s.SampleRate/sampleLength) //nolint // SA4010: this result of append is never used, except maybe in other appends
	}

	for k := range X {
		// Calculate the modulus since the result of FFT is an array of complex number with both real and imaginary parts
		amplitude := cmplx.Abs(X[k]) / sampleLength
		if amplitude < threshold {
			X[k] = 0.0
		}
	}

	x := fft.IFFT(X)

	samples := make([]float64, len(x))
	for i := range x {
		samples[i] = real(x[i])
	}

	return &Signal{
		SampleRate: s.SampleRate,
		Samples:    samples,
	}
}

func (s *Signal) FindPeriod() time.Duration {
	x := make([]float64, len(s.Samples))
	copy(x, s.Samples)
	N := len(s.Samples)

	// Use FFT to generate the periodogram of a random permutation of the samples, record
	// its maximum power argmax|X(f)|. Repeat above operation 100 times, and use the 99th
	// largest power as the threshold.
	var maxPowers []float64
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < 100; i++ {
		rand.Shuffle(len(x), func(i, j int) {
			x[i], x[j] = x[j], x[i]
		})
		X := fft.FFTReal(x)
		pmax := 0.
		for k := 1; k < len(X)/2; k++ {
			p := cmplx.Abs(X[k])
			if p > pmax {
				pmax = p
			}
		}
		maxPowers = append(maxPowers, pmax)
	}

	sort.Float64s(maxPowers)
	pThreshold := maxPowers[98]

	X := fft.FFTReal(s.Samples)
	var hints []int
	for k := 2; k < len(X)/2; k++ {
		p := cmplx.Abs(X[k])
		// If a frequency has more power than the threshold, regard its period as
		// a candidate fundamental period for the further verification.
		if p > pThreshold {
			if len(hints) == 0 || hints[len(hints)-1] > N/k {
				hints = append(hints, N/k)
			}
		}
	}

	// Use auto correlation function (ACF) to verify the candidate periods.
	// The value of the fundamental period should be the 'highest peak' in the graph of ACF.
	cor := AutoCorrelation(s.Samples)
	maxCorVal := 0.
	j := -1
	maxR := int(MaxAutoCorrelationPeakSearchIntervalSeconds * s.SampleRate)
	for i := range hints {
		r := min(maxR, hints[i]/2)
		if isPeak(cor, hints[i], r) && maxCorVal < cor[hints[i]] {
			j = i
			maxCorVal = cor[hints[j]]
		}
	}

	if j >= 0 {
		return time.Duration(float64(hints[j])/s.SampleRate) * time.Second
	} else {
		return -1
	}
}

func (s *Signal) String() string {
	return fmt.Sprintf("SampleRate: %.5fHz, Samples: %v, Duration: %.1fs", s.SampleRate, len(s.Samples), s.Duration())
}

func (s *Signal) Plot(color string, o ...charts.GlobalOpts) *charts.Line {
	x := make([]string, 0)
	y := make([]opts.LineData, 0)
	for i := 0; i < s.Num(); i++ {
		x = append(x, fmt.Sprintf("%.1f", float64(i)/s.SampleRate))
		y = append(y, opts.LineData{Value: s.Samples[i], Symbol: "none"})
	}

	line := charts.NewLine()
	line.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{Width: "3000px", Theme: types.ThemeRoma}),
		charts.WithTitleOpts(opts.Title{Title: s.String()}))
	if o != nil {
		line.SetGlobalOptions(o...)
	}
	if color != "" {
		line.SetXAxis(x).AddSeries("sample value", y, charts.WithAreaStyleOpts(
			opts.AreaStyle{
				Color:   color,
				Opacity: 0.1,
			}),
			charts.WithLineStyleOpts(opts.LineStyle{Color: color}))
	} else {
		line.SetXAxis(x).AddSeries("sample value", y)
	}

	return line
}
