package testdata

// Simple if.
func SimpleIf(x int) int {
	if x > 0 {
		return x
	}
	return -x
}

// If-else.
func IfElse(x int) string {
	if x > 0 {
		return "positive"
	} else {
		return "non-positive"
	}
}

// If-else-if chain.
func IfElseIfChain(x int) string {
	if x > 100 {
		return "big"
	} else if x > 10 {
		return "medium"
	} else if x > 0 {
		return "small"
	} else {
		return "non-positive"
	}
}

// If with init statement.
func IfWithInit(m map[string]int) int {
	if v, ok := m["key"]; ok {
		return v
	}
	return -1
}

// Nested if.
func NestedIf(x, y int) string {
	if x > 0 {
		if y > 0 {
			return "both positive"
		}
		return "x positive only"
	}
	return "x non-positive"
}
