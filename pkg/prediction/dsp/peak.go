package dsp

// Use linear regression to fit x in [i-k, i+k]
func isPeak(x []float64, i int, k int) bool {
	n := len(x)
	if i < 1 || i > n-1 {
		return false
	}

	l := max(0, i-k)
	r := min(n-1, i+k)

	p := []point{}
	for j := l; j <= i; j++ {
		p = append(p, point{x: float64(j), y: x[j]})
	}
	slope, _ := linearRegressionLSE(p)
	if slope <= 0 {
		return false
	}

	p = []point{}
	for j := i; j <= r; j++ {
		p = append(p, point{x: float64(j), y: x[j]})
	}
	slope, _ = linearRegressionLSE(p)
	if slope >= 0 {
		return false
	}

	return true
}

func min(x, y int) int {
	if x <= y {
		return x
	}
	return y
}

func max(x, y int) int {
	if x >= y {
		return x
	}
	return y
}

type point struct {
	x, y float64
}

// y_hat = ax + b
func linearRegressionLSE(points []point) (float64, float64) {
	var sumX, sumY, sumXY, sumXX float64
	for _, p := range points {
		sumX += p.x
		sumY += p.y
		sumXY += p.x * p.y
		sumXX += p.x * p.x
	}
	n := float64(len(points))
	a := (n*sumXY - sumX*sumY) / (n*sumXX - sumX*sumX)
	b := (sumY - a*sumX) / n
	return a, b
}
