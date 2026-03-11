package testdata

import "strings"

// Short-circuit boolean.
func ShortCircuit(a, b bool) bool {
	return a && b || !a
}

// Multi-assign.
func MultiAssign() (int, int) {
	a, b := 1, 2
	a, b = b, a
	return a, b
}

// Variadic function.
func Variadic(nums ...int) int {
	sum := 0
	for _, n := range nums {
		sum += n
	}
	return sum
}

// Slice operations.
func SliceOps() []int {
	s := make([]int, 0, 10)
	s = append(s, 1, 2, 3)
	s = append(s, []int{4, 5}...)
	return s[1:3]
}

// Map operations.
func MapOps() map[string]int {
	m := map[string]int{
		"a": 1,
		"b": 2,
	}
	m["c"] = 3
	delete(m, "a")
	if _, ok := m["a"]; !ok {
		m["a"] = 0
	}
	return m
}

// Type assertion.
func TypeAssert(v interface{}) (string, bool) {
	s, ok := v.(string)
	if !ok {
		return "", false
	}
	return strings.ToUpper(s), true
}

// Complex init.
func ComplexInit() []int {
	s := []int{
		1,
		2,
		3,
		func() int { return 4 }(),
	}
	return s
}

// Nested function calls.
func NestedCalls() string {
	return strings.Join(
		strings.Split("a.b.c", "."),
		"/",
	)
}

// Multiline expression across blocks.
func MultilineExpr(x int) int {
	result := x*2 +
		x*3 +
		x*4
	return result
}

// Blank identifier.
func BlankIdent() {
	_ = 42
	_, _ = MultiAssign()
}

// Const and var blocks in function.
func LocalConstVar() int {
	const (
		a = 1
		b = 2
	)
	var (
		x = a
		y = b
	)
	return x + y
}

// Go expression with function literal.
func GoExprFuncLit(ch chan int) {
	go func() {
		ch <- 1
	}()
}

// Assign with function literal.
func AssignFuncLit() int {
	x := func() int {
		return 42
	}()
	return x
}
