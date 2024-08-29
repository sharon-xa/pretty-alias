package main

import (
	"fmt"
	"io"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load(".env")

	if os.Getenv("PRETTY_ALIAS_DEBUG") == "" {
		log.SetOutput(io.Discard) // Disable logging by default
	} else {

		logFile, err := os.OpenFile("debug.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalln("Failed to open log file:", err)
		}

		log.SetOutput(logFile)
		defer logFile.Close()

		log.Println("Program started")
	}

	p := tea.NewProgram(&model{}, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
