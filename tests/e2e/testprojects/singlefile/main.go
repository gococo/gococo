package main

import (
	"fmt"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "0"
	}

	http.HandleFunc("/branch-a", handleA)
	http.HandleFunc("/branch-b", handleB)
	http.HandleFunc("/quit", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "bye")
		go func() { os.Exit(0) }()
	})

	ln, err := listenOnPort(port)
	if err != nil {
		fmt.Fprintf(os.Stderr, "listen: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("LISTEN %s\n", ln.Addr().String())
	http.Serve(ln, nil)
}

func handleA(w http.ResponseWriter, r *http.Request) {
	result := branchA(10)
	fmt.Fprintf(w, "a=%d", result)
}

func handleB(w http.ResponseWriter, r *http.Request) {
	result := branchB(5)
	fmt.Fprintf(w, "b=%d", result)
}

func branchA(n int) int {
	if n > 5 {
		return n * 2
	}
	return n
}

func branchB(n int) int {
	sum := 0
	for i := 0; i < n; i++ {
		if i%2 == 0 {
			sum += i
		} else {
			sum -= i
		}
	}
	return sum
}

// neverCalled is intentionally not reachable to test uncovered code.
func neverCalled() string {
	return "should not appear in coverage"
}
