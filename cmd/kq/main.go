package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("KogniKwery (kq) - Local-First Cognitive Data Engine")
		fmt.Println("Usage: kq <command> [flags]")
		return
	}

	switch os.Args[1] {
	case "version":
		fmt.Println("kq v0.1.0-dev (linux/amd64)")
	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
	}
}
