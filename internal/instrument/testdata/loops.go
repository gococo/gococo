package testdata

// Classic for loop.
func ForLoop(n int) int {
	sum := 0
	for i := 0; i < n; i++ {
		sum += i
	}
	return sum
}

// While-style for.
func WhileLoop(n int) int {
	i := 0
	for i < n {
		i++
	}
	return i
}

// Infinite for with break.
func InfiniteForBreak() int {
	count := 0
	for {
		count++
		if count >= 10 {
			break
		}
	}
	return count
}

// For with continue.
func ForContinue(n int) int {
	sum := 0
	for i := 0; i < n; i++ {
		if i%2 == 0 {
			continue
		}
		sum += i
	}
	return sum
}

// Range over slice.
func RangeSlice(s []int) int {
	sum := 0
	for _, v := range s {
		sum += v
	}
	return sum
}

// Range over map.
func RangeMap(m map[string]int) int {
	sum := 0
	for _, v := range m {
		sum += v
	}
	return sum
}

// Range over string.
func RangeString(s string) int {
	count := 0
	for range s {
		count++
	}
	return count
}

// Range over channel.
func RangeChannel(ch <-chan int) int {
	sum := 0
	for v := range ch {
		sum += v
	}
	return sum
}

// Nested loops.
func NestedLoops(n int) int {
	count := 0
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			count++
		}
	}
	return count
}

// Labeled break.
func LabeledBreak(matrix [][]int, target int) bool {
Outer:
	for _, row := range matrix {
		for _, v := range row {
			if v == target {
				break Outer
			}
		}
	}
	return false
}

// Labeled continue.
func LabeledContinue(matrix [][]int) int {
	sum := 0
Next:
	for _, row := range matrix {
		for _, v := range row {
			if v < 0 {
				continue Next
			}
			sum += v
		}
	}
	return sum
}
