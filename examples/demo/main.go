package main

import (
	"fmt"
	"math"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"demo/calc"
	"demo/convert"
	"demo/validate"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "0"
	}

	http.HandleFunc("/status", handleStatus)

	ln, err := net.Listen("tcp", "127.0.0.1:"+port)
	if err != nil {
		fmt.Fprintf(os.Stderr, "listen: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("LISTEN %s\n", ln.Addr().String())

	go autoLoop()
	http.Serve(ln, nil)
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "running")
}

// autoLoop cycles through different functions each second.
func autoLoop() {
	tick := 0
	for {
		time.Sleep(1 * time.Second)
		tick++
		fmt.Printf("[tick %d] ", tick)

		// Use tick*7 to create varied inputs (hits both even and odd numbers,
		// primes, multiples of 3, etc.) instead of tick itself which would
		// be monotonically increasing and biased.
		n := tick*7 + tick%3 - 1

		switch tick % 10 {
		case 0:
			result := classify(n)
			fmt.Println("classify:", result)
		case 1:
			result := fizzbuzz(n)
			fmt.Println("fizzbuzz:", result)
		case 2:
			result := triangleType(n%5+1, n%7+2, n%3+3)
			fmt.Println("triangle:", result)
		case 3:
			result := grade(float64(n % 100))
			fmt.Println("grade:", result)
		case 4:
			result := transform("hello world", n)
			fmt.Println("transform:", result)
		case 5:
			result := mathOp(float64(n), float64(n%7+1), n%4)
			fmt.Printf("math: %.2f\n", result)
		case 6:
			exerciseCalc(n)
		case 7:
			exerciseConvert(n)
		case 8:
			exerciseValidate(n)
		case 9:
			exerciseStats(n)
		}
	}
}

func exerciseCalc(tick int) {
	a, b := float64(tick), float64(tick%7+1)
	sum := calc.Add(a, b)
	diff := calc.Sub(a, b)
	prod := calc.Multiply(a, b)
	quot := calc.Divide(a, b)
	fmt.Printf("calc: add=%.0f sub=%.0f mul=%.0f div=%.1f\n", sum, diff, prod, quot)

	if tick%3 == 0 {
		fib := calc.Fibonacci(tick % 20)
		fmt.Printf("  fibonacci(%d)=%d\n", tick%20, fib)
	} else {
		fact := calc.Factorial(tick % 12)
		fmt.Printf("  factorial(%d)=%d\n", tick%12, fact)
	}
}

func exerciseConvert(tick int) {
	c := float64(tick % 40)
	f := convert.CelsiusToFahrenheit(c)
	fmt.Printf("convert: %.0f°C = %.1f°F\n", c, f)

	if tick%2 == 0 {
		km := float64(tick * 10)
		miles := convert.KmToMiles(km)
		fmt.Printf("  %.0f km = %.1f miles\n", km, miles)
	} else {
		bytes := int64(tick) * 1024 * 1024 * int64(tick%100+1)
		human := convert.BytesToHuman(bytes)
		fmt.Printf("  %d bytes = %s\n", bytes, human)
	}

	secs := tick * 67
	fmt.Printf("  %d secs = %s\n", secs, convert.SecondsToHuman(secs))
}

func exerciseValidate(tick int) {
	emails := []string{"user@example.com", "bad-email", "test@test.co", "", "a@b"}
	email := emails[tick%len(emails)]
	fmt.Printf("validate: email %q -> %v\n", email, validate.IsEmail(email))

	passwords := []string{"Str0ng!Pass", "weak", "NoSpecial1", "12345678", "Ab1!xxxx"}
	pw := passwords[tick%len(passwords)]
	fmt.Printf("  password %q -> %v\n", pw, validate.IsStrongPassword(pw))

	ips := []string{"192.168.1.1", "256.0.0.1", "10.0.0.1", "1.2.3", "0.0.0.0"}
	ip := ips[tick%len(ips)]
	fmt.Printf("  ipv4 %q -> %v\n", ip, validate.IsIPv4(ip))

	words := []string{"racecar", "hello", "madam", "world", "A man a plan a canal Panama"}
	word := words[tick%len(words)]
	fmt.Printf("  palindrome %q -> %v\n", word, validate.IsPalindrome(word))
}

func exerciseStats(tick int) {
	data := make([]float64, tick%10+3)
	for i := range data {
		data[i] = float64((tick + i*7) % 100)
	}
	mean := calc.Mean(data)
	median := calc.Median(data)
	sd := calc.StdDev(data)
	min, max := calc.MinMax(data)
	fmt.Printf("stats: mean=%.1f median=%.1f sd=%.1f min=%.0f max=%.0f\n",
		mean, median, sd, min, max)
}

// classify categorizes a number with nested conditions.
func classify(n int) string {
	if n <= 0 {
		return "non-positive"
	}
	if n%2 == 0 {
		if n%4 == 0 {
			if n%8 == 0 {
				return "divisible by 8"
			}
			return "divisible by 4"
		}
		return "even"
	}
	if n%3 == 0 {
		if n%9 == 0 {
			return "divisible by 9"
		}
		return "divisible by 3"
	}
	if isPrime(n) {
		return "prime"
	}
	return "odd composite"
}

func isPrime(n int) bool {
	if n < 2 {
		return false
	}
	for i := 2; i*i <= n; i++ {
		if n%i == 0 {
			return false
		}
	}
	return true
}

func fizzbuzz(n int) string {
	result := ""
	if n%3 == 0 {
		result += "Fizz"
	}
	if n%5 == 0 {
		result += "Buzz"
	}
	if n%7 == 0 {
		result += "Woof"
	}
	if result == "" {
		return fmt.Sprintf("%d", n)
	}
	return result
}

func triangleType(a, b, c int) string {
	if a+b <= c || a+c <= b || b+c <= a {
		return "invalid"
	}
	if a == b && b == c {
		return "equilateral"
	}
	if a == b || b == c || a == c {
		return "isosceles"
	}
	aa, bb, cc := a*a, b*b, c*c
	if aa+bb == cc || aa+cc == bb || bb+cc == aa {
		return "right"
	}
	return "scalene"
}

func grade(score float64) string {
	switch {
	case score >= 97:
		return "A+"
	case score >= 93:
		return "A"
	case score >= 90:
		return "A-"
	case score >= 87:
		return "B+"
	case score >= 83:
		return "B"
	case score >= 80:
		return "B-"
	case score >= 77:
		return "C+"
	case score >= 73:
		return "C"
	case score >= 70:
		return "C-"
	case score >= 60:
		return "D"
	default:
		return "F"
	}
}

func transform(s string, tick int) string {
	op := tick % 5
	switch op {
	case 0:
		return strings.ToUpper(s)
	case 1:
		return strings.ToLower(s)
	case 2:
		return reverseString(s)
	case 3:
		shifted := make([]byte, len(s))
		for i, c := range []byte(s) {
			if c >= 'a' && c <= 'z' {
				shifted[i] = 'a' + (c-'a'+3)%26
			} else if c >= 'A' && c <= 'Z' {
				shifted[i] = 'A' + (c-'A'+3)%26
			} else {
				shifted[i] = c
			}
		}
		return string(shifted)
	case 4:
		vowels, consonants := 0, 0
		for _, c := range strings.ToLower(s) {
			if strings.ContainsRune("aeiou", c) {
				vowels++
			} else if c >= 'a' && c <= 'z' {
				consonants++
			}
		}
		return fmt.Sprintf("v=%d c=%d", vowels, consonants)
	}
	return s
}

func reverseString(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

func mathOp(a, b float64, op int) float64 {
	switch op {
	case 0:
		return a + b
	case 1:
		if b != 0 {
			return a / b
		}
		return 0
	case 2:
		return math.Sqrt(a*a + b*b)
	case 3:
		return math.Sin(a) * math.Cos(b)
	}
	return 0
}

// neverCalled exists to show uncovered code in the UI.
func neverCalled() {
	fmt.Println("this code is never reached")
	for i := 0; i < 10; i++ {
		if i%2 == 0 {
			fmt.Println("even:", i)
		} else {
			fmt.Println("odd:", i)
		}
	}
}
