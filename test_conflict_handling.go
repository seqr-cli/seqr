package main

import (
	"fmt"
	"os"

	"github.com/seqr-cli/seqr/internal/config"
)

func main() {
	fmt.Println("Testing file conflict handling...")

	// Create a test file first
	testFile := "test-conflict.json"
	testContent := `{"existing": "content"}`

	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		fmt.Printf("Failed to create test file: %v\n", err)
		return
	}
	defer os.Remove(testFile) // Clean up

	fmt.Printf("Created test file: %s\n", testFile)

	// Now try to generate templates which should trigger conflict handling
	generator := config.NewTemplateGenerator()

	// This will prompt for user input when it encounters the existing file
	err = generator.GenerateAllTemplates()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println("Template generation completed!")
}
