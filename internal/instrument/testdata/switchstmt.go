package testdata

import "fmt"

// Simple switch.
func SimpleSwitch(x int) string {
	switch x {
	case 1:
		return "one"
	case 2:
		return "two"
	case 3:
		return "three"
	default:
		return "other"
	}
}

// Switch with no tag (expression switch).
func ExprSwitch(x int) string {
	switch {
	case x < 0:
		return "negative"
	case x == 0:
		return "zero"
	case x > 0:
		return "positive"
	}
	return "unreachable"
}

// Switch with init statement.
func SwitchWithInit(s string) int {
	switch n := len(s); {
	case n == 0:
		return 0
	case n < 5:
		return 1
	default:
		return 2
	}
}

// Switch with fallthrough.
func SwitchFallthrough(x int) string {
	result := ""
	switch x {
	case 1:
		result += "a"
		fallthrough
	case 2:
		result += "b"
	case 3:
		result += "c"
	}
	return result
}

// Switch with multi-value case.
func SwitchMultiCase(x int) string {
	switch x {
	case 1, 2, 3:
		return "low"
	case 4, 5, 6:
		return "mid"
	default:
		return "high"
	}
}

// Type switch.
func TypeSwitch(v interface{}) string {
	switch v.(type) {
	case int:
		return "int"
	case string:
		return "string"
	case bool:
		return "bool"
	default:
		return fmt.Sprintf("unknown: %T", v)
	}
}

// Type switch with assignment.
func TypeSwitchAssign(v interface{}) string {
	switch t := v.(type) {
	case int:
		return fmt.Sprintf("int: %d", t)
	case string:
		return fmt.Sprintf("string: %s", t)
	default:
		return fmt.Sprintf("other: %v", t)
	}
}

// Empty switch.
func EmptySwitch(x int) {
	switch x {
	}
}

// Empty type switch.
func EmptyTypeSwitch(v interface{}) {
	switch v.(type) {
	}
}
