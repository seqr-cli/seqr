package config

import (
	"fmt"
	"os"
)

// DemonstrateEnhancedErrorHandling shows examples of the improved error messages
func DemonstrateEnhancedErrorHandling() {
	fmt.Println("=== Enhanced Error Handling Demonstration ===")

	examples := []struct {
		name string
		json string
	}{
		{
			name: "Empty Configuration",
			json: "",
		},
		{
			name: "Invalid JSON Syntax",
			json: `{"version": "1.0"`,
		},
		{
			name: "Missing Commands Field",
			json: `{"version": "1.0"}`,
		},
		{
			name: "Empty Command String",
			json: `{
				"version": "1.0",
				"commands": [
					{
						"name": "test",
						"command": "",
						"mode": "once"
					}
				]
			}`,
		},
		{
			name: "Invalid Command Type",
			json: `{
				"version": "1.0",
				"commands": [
					{
						"name": "test",
						"command": 123,
						"mode": "once"
					}
				]
			}`,
		},
		{
			name: "Multiple Configuration Errors",
			json: `{
				"version": 123,
				"commands": [
					"invalid_command",
					{
						"name": 456,
						"command": "echo"
					}
				]
			}`,
		},
	}

	for _, example := range examples {
		fmt.Printf("--- %s ---\n", example.name)
		_, err := ParseJSON([]byte(example.json))
		if err != nil {
			fmt.Printf("Error: %s\n\n", err.Error())
		} else {
			fmt.Println("No error (unexpected)")
		}
	}
}

// RunDemo runs the demonstration if this file is executed directly
func RunDemo() {
	if len(os.Args) > 1 && os.Args[1] == "demo" {
		DemonstrateEnhancedErrorHandling()
	}
}
