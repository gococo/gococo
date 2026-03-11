package convert

import (
	"fmt"
	"strings"
)

func CelsiusToFahrenheit(c float64) float64 {
	return c*9/5 + 32
}

func FahrenheitToCelsius(f float64) float64 {
	return (f - 32) * 5 / 9
}

func KmToMiles(km float64) float64 {
	return km * 0.621371
}

func MilesToKm(miles float64) float64 {
	return miles / 0.621371
}

func BytesToHuman(bytes int64) string {
	if bytes < 0 {
		return "invalid"
	}
	units := []string{"B", "KB", "MB", "GB", "TB"}
	size := float64(bytes)
	i := 0
	for size >= 1024 && i < len(units)-1 {
		size /= 1024
		i++
	}
	if i == 0 {
		return fmt.Sprintf("%d B", bytes)
	}
	return fmt.Sprintf("%.1f %s", size, units[i])
}

func SecondsToHuman(secs int) string {
	if secs < 0 {
		return "invalid"
	}
	if secs == 0 {
		return "0s"
	}

	parts := []string{}
	if days := secs / 86400; days > 0 {
		parts = append(parts, fmt.Sprintf("%dd", days))
		secs %= 86400
	}
	if hours := secs / 3600; hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
		secs %= 3600
	}
	if mins := secs / 60; mins > 0 {
		parts = append(parts, fmt.Sprintf("%dm", mins))
		secs %= 60
	}
	if secs > 0 {
		parts = append(parts, fmt.Sprintf("%ds", secs))
	}
	return strings.Join(parts, " ")
}
