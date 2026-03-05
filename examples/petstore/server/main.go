package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

func main() {
	transport := flag.String("transport", "stdio", "Transport type: stdio or sse")
	port := flag.Int("port", 8080, "Port for SSE transport")
	baseURL := flag.String("base-url", "", "Base URL of the API")
	flag.Parse()

	apiKey := os.Getenv("MINT_API_KEY")

	srv := NewServer(*baseURL, apiKey)

	switch *transport {
	case "stdio":
		if err := srv.ServeStdio(); err != nil {
			log.Fatal(err)
		}
	case "sse":
		addr := fmt.Sprintf(":%d", *port)
		log.Printf("Starting SSE server on %s", addr)
		if err := srv.ServeSSE(addr); err != nil {
			log.Fatal(err)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown transport: %s\n", *transport)
		os.Exit(1)
	}
}
