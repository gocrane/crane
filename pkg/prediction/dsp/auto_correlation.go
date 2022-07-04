package dsp

import (
	"math"
	"math/cmplx"

	"github.com/mjibson/go-dsp/fft"
	"github.com/montanaflynn/stats"
)

func AutoCorrelation(samples []float64) []float64 {
	N := len(samples)

	if N == 0 {
		return []float64{}
	}

	x := make([]float64, N)
	mean, _ := stats.Mean(samples)
	std, _ := stats.StdDevP(samples)
	for i := range x {
		x[i] = (samples[i] - mean) / std
	}

	f := fft.FFTReal(x)
	var p []float64
	for i := range f {
		p = append(p, math.Pow(cmplx.Abs(f[i]), 2))
	}
	pi := fft.IFFTReal(p)

	var ac []float64
	for i := range pi {
		ac = append(ac, real(pi[i])/float64(N))
	}
	return ac
}
