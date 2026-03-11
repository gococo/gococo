package calc

import (
	"math"
	"sort"
)

func Mean(nums []float64) float64 {
	if len(nums) == 0 {
		return 0
	}
	sum := 0.0
	for _, n := range nums {
		sum += n
	}
	return sum / float64(len(nums))
}

func Median(nums []float64) float64 {
	if len(nums) == 0 {
		return 0
	}
	sorted := make([]float64, len(nums))
	copy(sorted, nums)
	sort.Float64s(sorted)
	mid := len(sorted) / 2
	if len(sorted)%2 == 0 {
		return (sorted[mid-1] + sorted[mid]) / 2
	}
	return sorted[mid]
}

func StdDev(nums []float64) float64 {
	if len(nums) <= 1 {
		return 0
	}
	mean := Mean(nums)
	sumSq := 0.0
	for _, n := range nums {
		diff := n - mean
		sumSq += diff * diff
	}
	return math.Sqrt(sumSq / float64(len(nums)-1))
}

func MinMax(nums []float64) (float64, float64) {
	if len(nums) == 0 {
		return 0, 0
	}
	min, max := nums[0], nums[0]
	for _, n := range nums[1:] {
		if n < min {
			min = n
		}
		if n > max {
			max = n
		}
	}
	return min, max
}
