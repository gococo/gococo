package testdata

// Package-level init function.
var initOrder []string

func init() {
	initOrder = append(initOrder, "first")
}

func init() {
	initOrder = append(initOrder, "second")
}

// Package-level var with function call.
var computed = computeOnInit()

func computeOnInit() int {
	return 42
}

// Package-level var block.
var (
	globalA = 1
	globalB = globalA + 1
)

func GetInitOrder() []string { return initOrder }
func GetComputed() int       { return computed }
func GetGlobals() (int, int) { return globalA, globalB }
