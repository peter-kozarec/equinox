package fixed

func DownsideDev(points []Point, riskFreeRate Point) Point {
	var sum Point
	var count int
	for _, point := range points {
		if point.Lt(riskFreeRate) {
			diff := point.Sub(riskFreeRate)
			sum = sum.Add(diff.Mul(diff))
			count++
		}
	}
	if count == 0 {
		return Zero
	}
	if count == 1 {
		return Zero
	}
	return sum.DivInt(count).Sqrt()
}

func Mean(points []Point) Point {
	if len(points) > 0 {
		sum := Zero
		for _, point := range points {
			sum = sum.Add(point)
		}
		return sum.DivInt(len(points))
	}
	return Point{}
}

func SharpeRatio(points []Point, riskFreeRate Point) Point {
	mean := Mean(points)
	volatility := StdDev(points, mean)
	if volatility.IsZero() {
		return Zero
	}
	return mean.Sub(riskFreeRate).Div(volatility)
}

func SortinoRatio(points []Point, riskFreeRate Point) Point {
	mean := Mean(points)
	downsideDeviation := DownsideDev(points, riskFreeRate)
	if downsideDeviation.IsZero() {
		return Zero
	}
	return mean.Sub(riskFreeRate).Div(downsideDeviation)
}

func StdDev(points []Point, mean Point) Point {
	if len(points) > 0 {
		if len(points) == 1 {
			return Zero
		}
		sum := Zero
		for _, point := range points {
			diff := point.Sub(mean)
			sum = sum.Add(diff.Mul(diff))
		}
		return sum.DivInt(len(points)).Sqrt()
	}
	return Point{}
}
