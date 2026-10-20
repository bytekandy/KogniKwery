package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/bytekandy/kognikwery/internal/ai"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("KogniKwery (kq) - Local-First Cognitive Data Engine")
		fmt.Println("Commands:")
		fmt.Println("  kq version               Show binary version")
		fmt.Println("  kq ping-ai [flags]       Test connection to local LLM server")
		return
	}

	switch os.Args[1] {
	case "version":
		fmt.Println("kq v0.1.0-dev (linux/amd64)")

	case "ping-ai":
		pingCmd := flag.NewFlagSet("ping-ai", flag.ExitOnError)
		endpoint := pingCmd.String("endpoint", "http://localhost:11434/v1/chat/completions", "OpenAI-compatible LLM endpoint")
		model := pingCmd.String("model", "gemma3:4b", "Local model name")
		pingCmd.Parse(os.Args[2:])

		client := ai.NewClient(*endpoint, *model)
		fmt.Printf("Pinging model '%s' at %s...\n", client.Model, client.Endpoint)

		reply, err := client.Ask("You are a helpful assistant.", "Reply with the exact word: PONG")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("LLM Response: %s\n", reply)

	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}
