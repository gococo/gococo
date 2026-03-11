package main

import (
	"fmt"
	"net"
	"net/http"
	"os"

	"testproject/multipkg/calc"
	"testproject/multipkg/greeting"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "0"
	}

	http.HandleFunc("/add", func(w http.ResponseWriter, r *http.Request) {
		result := calc.Add(3, 4)
		fmt.Fprintf(w, "%d", result)
	})
	http.HandleFunc("/greet", func(w http.ResponseWriter, r *http.Request) {
		msg := greeting.Hello("world")
		fmt.Fprint(w, msg)
	})
	http.HandleFunc("/quit", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "bye")
		go func() { os.Exit(0) }()
	})

	ln, err := net.Listen("tcp", "127.0.0.1:"+port)
	if err != nil {
		fmt.Fprintf(os.Stderr, "listen: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("LISTEN %s\n", ln.Addr().String())
	http.Serve(ln, nil)
}
