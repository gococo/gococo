package testdata

import (
	"errors"
	"fmt"
)

// Single-line if.
func SingleLineIf(x int) int {
	if x > 0 { return x }
	return 0
}

// Multiline function signature.
func MultilineSig(
	a int,
	b int,
	c int,
) int {
	return a + b + c
}

// Empty for body.
func EmptyForBody(ch <-chan struct{}) {
	for range ch {
	}
}

// Empty if body.
func EmptyIfBody(x int) {
	if x > 0 {
	}
}

// Goto statement.
func WithGoto(x int) int {
	if x < 0 {
		goto negative
	}
	return x

negative:
	return -x
}

// Labeled statement that is not a loop.
func LabeledNonLoop(x int) int {
start:
	x++
	if x < 10 {
		goto start
	}
	return x
}

// Multiple return paths.
func MultiReturn2(x int) (string, error) {
	if x < 0 {
		return "", errors.New("negative")
	}
	if x == 0 {
		return "zero", nil
	}
	if x > 100 {
		return "", fmt.Errorf("too big: %d", x)
	}
	return fmt.Sprintf("%d", x), nil
}

// Nested blocks (bare blocks).
func NestedBareBlocks() int {
	x := 0
	{
		y := 1
		{
			z := 2
			x = y + z
		}
	}
	return x
}

// Complex boolean expression in if.
func ComplexBoolIf(a, b, c bool) string {
	if (a && b) || (b && c) || (a && c) {
		return "majority true"
	}
	return "majority false"
}

// Switch with block in case.
func SwitchCaseBlock(x int) int {
	switch {
	case x > 0:
		{
			y := x * 2
			return y
		}
	default:
		return 0
	}
}

// Long function with many blocks.
func ManyBlocks(x int) string {
	if x == 0 {
		return "zero"
	}
	if x == 1 {
		return "one"
	}
	if x == 2 {
		return "two"
	}
	if x == 3 {
		return "three"
	}
	if x == 4 {
		return "four"
	}
	if x == 5 {
		return "five"
	}
	if x == 6 {
		return "six"
	}
	if x == 7 {
		return "seven"
	}
	if x == 8 {
		return "eight"
	}
	if x == 9 {
		return "nine"
	}
	return "big"
}

// Empty function with defer only.
func DeferOnly() {
	defer func() {}()
}

// Expression statement (non-call).
func ExprStmt() {
	_ = func() {}
}
