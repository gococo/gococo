package testdata

import "fmt"

// Simple defer.
func SimpleDefer() string {
	result := "start"
	defer func() {
		result = "deferred"
	}()
	result = "middle"
	return result
}

// Multiple defers (LIFO order).
func MultiDefer() []int {
	var order []int
	defer func() { order = append(order, 1) }()
	defer func() { order = append(order, 2) }()
	defer func() { order = append(order, 3) }()
	return order
}

// Defer in loop.
func DeferInLoop(n int) {
	for i := 0; i < n; i++ {
		defer fmt.Sprintf("%d", i)
	}
}

// Panic and recover.
func PanicRecover() (result string) {
	defer func() {
		if r := recover(); r != nil {
			result = fmt.Sprintf("recovered: %v", r)
		}
	}()
	panic("test panic")
}

// Panic without recover (just tests instrumentation, not execution).
func WillPanic() {
	panic("boom")
}

// Named return with defer.
func NamedReturnDefer() (n int) {
	defer func() {
		n++
	}()
	return 1 // n will be 2
}

// Defer with method value.
type Closer struct{ closed bool }

func (c *Closer) Close() { c.closed = true }

func DeferMethodValue() *Closer {
	c := &Closer{}
	defer c.Close()
	return c
}
