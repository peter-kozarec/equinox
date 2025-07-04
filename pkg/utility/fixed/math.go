package fixed

func Mean(points []Point) Point {
	if len(points) == 0 {
		return Zero
	}
	sum := Zero
	for _, point := range points {
		sum = sum.Add(point)
	}
	return sum.DivInt(len(points))
}

func DownsideDev(points []Point, riskFreeRate Point) Point {
	if len(points) == 0 {
		return Zero
	}

	sum := Zero
	count := 0
	for _, point := range points {
		if point.Lt(riskFreeRate) {
			diff := point.Sub(riskFreeRate)
			sum = sum.Add(diff.Mul(diff))
			count++
		}
	}

	if count <= 1 {
		return Zero
	}

	return sum.DivInt(count).Sqrt()
}

func SampleDownsideDev(points []Point, riskFreeRate Point) Point {
	if len(points) == 0 {
		return Zero
	}

	sum := Zero
	count := 0
	for _, point := range points {
		if point.Lt(riskFreeRate) {
			diff := point.Sub(riskFreeRate)
			sum = sum.Add(diff.Mul(diff))
			count++
		}
	}

	if count <= 1 {
		return Zero
	}

	return sum.DivInt(count - 1).Sqrt()
}

func StdDev(points []Point, mean Point) Point {
	if len(points) <= 1 {
		return Zero
	}

	sum := Zero
	for _, point := range points {
		diff := point.Sub(mean)
		sum = sum.Add(diff.Mul(diff))
	}

	return sum.DivInt(len(points)).Sqrt()
}

func SampleStdDev(points []Point, mean Point) Point {
	if len(points) <= 1 {
		return Zero
	}

	sum := Zero
	for _, point := range points {
		diff := point.Sub(mean)
		sum = sum.Add(diff.Mul(diff))
	}

	return sum.DivInt(len(points) - 1).Sqrt()
}

func Variance(points []Point, mean Point) Point {
	if len(points) <= 1 {
		return Zero
	}

	sum := Zero
	for _, point := range points {
		diff := point.Sub(mean)
		sum = sum.Add(diff.Mul(diff))
	}

	return sum.DivInt(len(points))
}

func SampleVariance(points []Point, mean Point) Point {
	if len(points) <= 1 {
		return Zero
	}

	sum := Zero
	for _, point := range points {
		diff := point.Sub(mean)
		sum = sum.Add(diff.Mul(diff))
	}

	return sum.DivInt(len(points) - 1)
}

func SharpeRatio(points []Point, riskFreeRate Point) Point {
	if len(points) == 0 {
		return Zero
	}

	mean := Mean(points)
	volatility := StdDev(points, mean)

	if volatility.IsZero() {
		return Zero
	}

	return mean.Sub(riskFreeRate).Div(volatility)
}

func SortinoRatio(points []Point, riskFreeRate Point) Point {
	if len(points) == 0 {
		return Zero
	}

	mean := Mean(points)
	downsideDeviation := DownsideDev(points, riskFreeRate)

	if downsideDeviation.IsZero() {
		return Zero
	}

	return mean.Sub(riskFreeRate).Div(downsideDeviation)
}
