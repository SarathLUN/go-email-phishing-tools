package main

import (
	"github.com/SarathLUN/go-email-phishing-tools/internal/app"
	"log"
	"os"
)

func main() {
	// Setup logging
	log.SetOutput(os.Stdout)                     // Log to stdout
	log.SetFlags(log.LstdFlags | log.Lshortfile) // Add timestamp and file/line number

	log.Println("Starting email-phishing-tools CLI...")

	// Execute the Cobra application defined in the app package
	app.Execute()

	log.Println("email-phishing-tools CLI finished.")
}
