package testdata

// Basic function with sequential statements.
func BasicSequential() int {
	a := 1
	b := 2
	c := a + b
	return c
}

// Function with no body (interface-like, but concrete empty).
func EmptyFunc() {}

// Multiple return values.
func MultiReturn(x int) (int, error) {
	if x < 0 {
		return 0, nil
	}
	return x * 2, nil
}
