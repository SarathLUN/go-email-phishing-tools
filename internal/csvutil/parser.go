package csvutil

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

// ParsedTarget represents the raw data read from a CSV row.
type ParsedTarget struct {
	FullName string
	Email    string
	Line     int // Original line number for error reporting
}

// ParseTargetsCSV reads a CSV file and returns a slice of ParsedTarget structs.
// It expects columns named "full_name" and "email" (case-insensitive).
func ParseTargetsCSV(filePath string) ([]*ParsedTarget, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file '%s': %w", filePath, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.TrimLeadingSpace = true // Handle potential whitespace

	// Read header
	header, err := reader.Read()
	if err != nil {
		if err == io.EOF {
			return nil, fmt.Errorf("csv file '%s' is empty or has no header", filePath)
		}
		return nil, fmt.Errorf("failed to read CSV header from '%s': %w", filePath, err)
	}

	// Find column indices (case-insensitive)
	nameIndex, emailIndex := -1, -1
	for i, colName := range header {
		cleanName := strings.ToLower(strings.TrimSpace(colName))
		if cleanName == "full_name" {
			nameIndex = i
		} else if cleanName == "email" {
			emailIndex = i
		}
	}

	if nameIndex == -1 || emailIndex == -1 {
		return nil, fmt.Errorf("csv file '%s' must contain 'full_name' and 'email' columns (case-insensitive)", filePath)
	}

	var targets []*ParsedTarget
	line := 1 // Start counting lines after header

	for {
		line++
		record, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break // End of file
			}
			log.Printf("Warning: Error reading CSV record on line %d in '%s': %v. Skipping line.", line, filePath, err)
			continue // Skip malformed lines
		}

		if len(record) <= nameIndex || len(record) <= emailIndex {
			log.Printf("Warning: Skipping line %d in '%s' due to insufficient columns (expected at least %d).", line, filePath, max(nameIndex, emailIndex)+1)
			continue
		}

		fullName := strings.TrimSpace(record[nameIndex])
		email := strings.TrimSpace(record[emailIndex])

		// Basic validation
		if fullName == "" {
			log.Printf("Warning: Skipping line %d in '%s' due to empty full_name.", line, filePath)
			continue
		}
		if email == "" || !strings.Contains(email, "@") { // Very basic email format check
			log.Printf("Warning: Skipping line %d in '%s' due to invalid or empty email: '%s'.", line, filePath, email)
			continue
		}

		targets = append(targets, &ParsedTarget{
			FullName: fullName,
			Email:    email,
			Line:     line,
		})
	}

	if len(targets) == 0 {
		log.Printf("No valid target records found in CSV file '%s'.", filePath)
	}

	log.Printf("Successfully parsed %d potential targets from '%s'.", len(targets), filePath)
	return targets, nil
}

// max returns the greater of two integers.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
