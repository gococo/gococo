package calc

import "math"

func Add(a, b float64) float64 {
	return a + b
}

func Sub(a, b float64) float64 {
	return a - b
}

func Multiply(a, b float64) float64 {
	if a == 0 || b == 0 {
		return 0
	}
	return a * b
}

func Divide(a, b float64) float64 {
	if b == 0 {
		return math.NaN()
	}
	return a / b
}

func Power(base, exp float64) float64 {
	if exp == 0 {
		return 1
	}
	if exp < 0 {
		return 1 / Power(base, -exp)
	}
	return base * Power(base, exp-1)
}

func Factorial(n int) int {
	if n <= 1 {
		return 1
	}
	return n * Factorial(n-1)
}

func Fibonacci(n int) int {
	if n <= 0 {
		return 0
	}
	if n == 1 {
		return 1
	}
	a, b := 0, 1
	for i := 2; i <= n; i++ {
		a, b = b, a+b
	}
	return b
}
