package arima

import "github.com/peter-kozarec/equinox/pkg/utility/fixed"

func solveLinearSystem(A [][]fixed.Point, b []fixed.Point) []fixed.Point {
	n := len(b)
	if n == 0 || len(A) != n {
		return nil
	}

	// Create augmented matrix [A | b]
	augmented := make([][]fixed.Point, n)
	for i := 0; i < n; i++ {
		if len(A[i]) != n {
			return nil // ensure A is square
		}
		augmented[i] = make([]fixed.Point, n+1)
		for j := 0; j < n; j++ {
			augmented[i][j] = A[i][j]
		}
		augmented[i][n] = b[i]
	}

	// Gaussian elimination with partial pivoting
	for k := 0; k < n; k++ {
		// Find row with largest absolute pivot
		maxRow := k
		for i := k + 1; i < n; i++ {
			if augmented[i][k].Abs().Gt(augmented[maxRow][k].Abs()) {
				maxRow = i
			}
		}

		// Swap rows if needed
		if maxRow != k {
			augmented[k], augmented[maxRow] = augmented[maxRow], augmented[k]
		}

		// Check for singular matrix
		if augmented[k][k].IsZero() {
			return nil
		}

		// Eliminate below
		for i := k + 1; i < n; i++ {
			factor := augmented[i][k].Div(augmented[k][k])
			for j := k; j < n+1; j++ {
				augmented[i][j] = augmented[i][j].Sub(factor.Mul(augmented[k][j]))
			}
		}
	}

	// Back substitution
	x := make([]fixed.Point, n)
	for i := n - 1; i >= 0; i-- {
		sum := augmented[i][n]
		for j := i + 1; j < n; j++ {
			sum = sum.Sub(augmented[i][j].Mul(x[j]))
		}
		if augmented[i][i].IsZero() {
			return nil
		}
		x[i] = sum.Div(augmented[i][i])
	}

	return x
}

func solveNormalEquations(X [][]fixed.Point, y []fixed.Point) []fixed.Point {
	if len(X) == 0 || len(X[0]) == 0 {
		return nil
	}

	rows := len(X)
	cols := len(X[0])

	// Compute X'X
	XtX := make([][]fixed.Point, cols)
	for i := 0; i < cols; i++ {
		XtX[i] = make([]fixed.Point, cols)
		for j := 0; j < cols; j++ {
			var sum fixed.Point
			for k := 0; k < rows; k++ {
				sum = sum.Add(X[k][i].Mul(X[k][j]))
			}
			XtX[i][j] = sum
		}
	}

	// Compute X'y
	Xty := make([]fixed.Point, cols)
	for i := 0; i < cols; i++ {
		var sum fixed.Point
		for k := 0; k < rows; k++ {
			sum = sum.Add(X[k][i].Mul(y[k]))
		}
		Xty[i] = sum
	}

	// Solve (X'X)β = X'y
	return solveLinearSystem(XtX, Xty)
}

func binomialCoefficient(n, k uint) fixed.Point {
	if k > n {
		return fixed.Zero
	}
	if k == 0 || k == n {
		return fixed.One
	}

	// Use multiplicative formula to avoid overflow
	result := fixed.One
	for i := uint(0); i < k; i++ {
		result = result.MulInt(int(n - i))
		result = result.DivInt(int(i + 1))
	}
	return result
}
