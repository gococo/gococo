package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"time"
)

func main() {
	http.HandleFunc("/", handleRoot)
	http.HandleFunc("/compute", handleCompute)

	fmt.Println("listening on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Printf("server error: %v\n", err)
	}
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello from gococo example! Time: %s\n", time.Now().Format(time.RFC3339))
}

func handleCompute(w http.ResponseWriter, r *http.Request) {
	n := rand.Intn(100)
	result := compute(n)
	fmt.Fprintf(w, "compute(%d) = %d\n", n, result)
}

func compute(n int) int {
	if n <= 1 {
		return n
	}
	if n%2 == 0 {
		return compute(n/2) + 1
	}
	return compute(3*n+1) + 1
}
