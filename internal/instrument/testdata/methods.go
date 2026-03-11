package testdata

import "fmt"

// Interface and implementation.
type Animal interface {
	Sound() string
	Name() string
}

type Dog struct{ name string }

func (d *Dog) Sound() string { return "woof" }
func (d *Dog) Name() string  { return d.name }

type Cat struct{ name string }

func (c Cat) Sound() string { return "meow" }
func (c Cat) Name() string  { return c.name }

// Value receiver vs pointer receiver.
type Counter struct {
	n int
}

func (c Counter) Value() int     { return c.n }
func (c *Counter) Increment()    { c.n++ }
func (c *Counter) Add(delta int) { c.n += delta }

// Method with complex logic.
func (c *Counter) AddIf(delta int, cond bool) {
	if cond {
		c.n += delta
	} else {
		c.n -= delta
	}
}

// Stringer interface.
type Point struct {
	X, Y int
}

func (p Point) String() string {
	return fmt.Sprintf("(%d, %d)", p.X, p.Y)
}

// Embedded struct.
type Base struct {
	ID int
}

func (b *Base) GetID() int { return b.ID }

type Derived struct {
	Base
	Extra string
}

func (d *Derived) Describe() string {
	return fmt.Sprintf("id=%d extra=%s", d.GetID(), d.Extra)
}
