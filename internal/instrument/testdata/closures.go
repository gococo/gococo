package testdata

import "sort"

// Function literal in assignment.
func ClosureAssign() func() int {
	x := 42
	return func() int {
		return x
	}
}

// Immediately invoked function literal.
func IIFE() int {
	return func() int {
		return 42
	}()
}

// Function literal as argument.
func ClosureAsArg() {
	s := []int{3, 1, 2}
	sort.Slice(s, func(i, j int) bool {
		return s[i] < s[j]
	})
}

// Multiple closures in one function.
func MultiClosure() (func() int, func() int) {
	a := func() int { return 1 }
	b := func() int { return 2 }
	return a, b
}

// Closure capturing loop variable.
func ClosureInLoop(n int) []func() int {
	funcs := make([]func() int, n)
	for i := 0; i < n; i++ {
		i := i // shadow
		funcs[i] = func() int {
			return i
		}
	}
	return funcs
}

// Nested closures.
func NestedClosure() func() func() int {
	return func() func() int {
		return func() int {
			return 42
		}
	}
}

// Closure with multiple statements.
func ClosureMultiStmt() int {
	f := func(a, b int) int {
		sum := a + b
		if sum > 10 {
			return sum * 2
		}
		return sum
	}
	return f(3, 4)
}
