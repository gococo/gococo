package greeting

import "fmt"

func Hello(name string) string {
	if name == "" {
		return "Hello, stranger!"
	}
	return fmt.Sprintf("Hello, %s!", name)
}

func Goodbye(name string) string {
	if name == "" {
		return "Goodbye!"
	}
	return fmt.Sprintf("Goodbye, %s!", name)
}
