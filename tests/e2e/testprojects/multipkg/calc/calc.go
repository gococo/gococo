package calc

func Add(a, b int) int {
	return a + b
}

func Sub(a, b int) int {
	return a - b
}

func Multiply(a, b int) int {
	if a == 0 || b == 0 {
		return 0
	}
	return a * b
}

// Divide performs integer division with error handling.
func Divide(a, b int) (int, error) {
	if b == 0 {
		panic("division by zero")
	}
	return a / b, nil
}
