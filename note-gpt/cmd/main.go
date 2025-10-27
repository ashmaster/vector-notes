package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"note-gpt/internal"
	"note-gpt/pkg"
)

func main() {
	config, err := internal.LoadConfig()
	if err != nil {
		fmt.Println("Failed to load configuration. Exiting.")
		return
	}
	geminiClient, err := pkg.NewGeminiClient(config.GeminiAPIKey)
	if err != nil {
		fmt.Printf("Error initializing Gemini client: %v\n", err)
		return
	}
	defer geminiClient.Close()

	vectorDb, err := pkg.NewVector(config.PineconeAPIKey, config.PineconeHost, "notes-index", config.EmbeddingUrl)
	if err != nil {
		fmt.Printf("Error initializing vector database: %v\n", err)
		return
	}

	// Interactive mode
	flag.Parse()
	fmt.Println("Welcome to note-gpt! Type 'exit' to quit.")
	app := internal.NewApp(vectorDb, geminiClient)
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "exit" || input == "quit" {
			fmt.Println("Goodbye!")
			break
		}

		if input == "" {
			continue
		}

		response, err := app.HandleQuery(input)
		if err != nil {
			fmt.Printf("Error handling query: %v\n", err)
			continue
		}
		fmt.Println(response)
	}
}
