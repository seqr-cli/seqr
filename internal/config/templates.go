package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// TemplateGenerator handles creation of example configuration files
type TemplateGenerator struct {
	OutputDir string
}

// NewTemplateGenerator creates a new template generator
func NewTemplateGenerator() *TemplateGenerator {
	return &TemplateGenerator{
		OutputDir: ".",
	}
}

// GenerateAllTemplates creates example files for all supported configuration formats
func (tg *TemplateGenerator) GenerateAllTemplates() error {
	templates := []struct {
		filename string
		content  string
		desc     string
	}{
		{
			filename: "example-string-format.queue.json",
			content:  tg.getStringFormatTemplate(),
			desc:     "String format - commands as simple strings (e.g., \"npm install\")",
		},
		{
			filename: "example-array-format.queue.json",
			content:  tg.getArrayFormatTemplate(),
			desc:     "Array format - commands as arrays (e.g., [\"npm\", \"install\"])",
		},
		{
			filename: "example-object-format.queue.json",
			content:  tg.getObjectFormatTemplate(),
			desc:     "Object format - commands as objects with separate command/args",
		},
		{
			filename: "example-mixed-format.queue.json",
			content:  tg.getMixedFormatTemplate(),
			desc:     "Mixed format - demonstrates all three formats in one file",
		},
		{
			filename: "example-fullstack.queue.json",
			content:  tg.getFullstackTemplate(),
			desc:     "Full-stack example - complete development environment setup",
		},
	}

	fmt.Printf("Generating example configuration files...\n\n")

	for _, template := range templates {
		if err := tg.writeTemplate(template.filename, template.content, template.desc); err != nil {
			return fmt.Errorf("failed to create %s: %w", template.filename, err)
		}
	}

	fmt.Printf("\n✨ Example files generated successfully!\n\n")
	fmt.Printf("📖 Format Guide:\n")
	fmt.Printf("   • String format: \"npm install\" - Simple and familiar\n")
	fmt.Printf("   • Array format: [\"npm\", \"install\"] - Handles spaces cleanly\n")
	fmt.Printf("   • Object format: {\"command\": \"npm\", \"args\": [\"install\"]} - Most structured\n\n")
	fmt.Printf("🚀 Usage: seqr -f <filename>\n")
	fmt.Printf("   Example: seqr -f example-string-format.queue.json\n")

	return nil
}

// writeTemplate writes a template file with conflict handling
func (tg *TemplateGenerator) writeTemplate(filename, content, description string) error {
	fullPath := filepath.Join(tg.OutputDir, filename)

	// Check if file already exists
	if _, err := os.Stat(fullPath); err == nil {
		fmt.Printf("⚠️  %s already exists, skipping...\n", filename)
		return nil
	}

	// Write the file
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	fmt.Printf("✅ Created %s\n   %s\n", filename, description)
	return nil
}

// getStringFormatTemplate returns a template using string command format
func (tg *TemplateGenerator) getStringFormatTemplate() string {
	config := map[string]interface{}{
		"version": "1.0",
		"commands": []map[string]interface{}{
			{
				"name":    "install-deps",
				"command": "npm install",
				"mode":    "once",
			},
			{
				"name":    "build-project",
				"command": "npm run build",
				"mode":    "once",
			},
			{
				"name":    "start-server",
				"command": "npm start",
				"mode":    "keepAlive",
				"env": map[string]string{
					"NODE_ENV": "production",
					"PORT":     "3000",
				},
			},
		},
	}

	jsonBytes, _ := json.MarshalIndent(config, "", "  ")

	return string(jsonBytes)
}

// getArrayFormatTemplate returns a template using array command format
func (tg *TemplateGenerator) getArrayFormatTemplate() string {
	config := map[string]interface{}{
		"version": "1.0",
		"commands": []map[string]interface{}{
			{
				"name":    "setup-docker",
				"command": []string{"docker", "run", "-d", "--name", "redis", "-p", "6379:6379", "redis:alpine"},
				"mode":    "keepAlive",
			},
			{
				"name":    "wait-for-redis",
				"command": []string{"sleep", "2"},
				"mode":    "once",
			},
			{
				"name":    "run-tests",
				"command": []string{"npm", "test", "--", "--coverage"},
				"mode":    "once",
				"workDir": "./backend",
			},
			{
				"name":    "start-app",
				"command": []string{"node", "server.js", "--port", "8080"},
				"mode":    "keepAlive",
				"workDir": "./backend",
				"env": map[string]string{
					"REDIS_URL": "redis://localhost:6379",
				},
			},
		},
	}

	jsonBytes, _ := json.MarshalIndent(config, "", "  ")

	return string(jsonBytes)
}

// getObjectFormatTemplate returns a template using object command format
func (tg *TemplateGenerator) getObjectFormatTemplate() string {
	config := map[string]interface{}{
		"version": "1.0",
		"commands": []map[string]interface{}{
			{
				"name": "database-setup",
				"command": map[string]interface{}{
					"command": "docker",
					"args":    []string{"run", "-d", "--name", "postgres", "-p", "5432:5432", "postgres:13"},
				},
				"mode": "keepAlive",
				"env": map[string]string{
					"POSTGRES_PASSWORD": "devpass",
					"POSTGRES_DB":       "myapp",
				},
			},
			{
				"name": "wait-for-db",
				"command": map[string]interface{}{
					"command": "sleep",
					"args":    []string{"3"},
				},
				"mode": "once",
			},
			{
				"name": "run-migrations",
				"command": map[string]interface{}{
					"command": "npm",
					"args":    []string{"run", "db:migrate"},
				},
				"mode":    "once",
				"workDir": "./api",
			},
			{
				"name": "start-api",
				"command": map[string]interface{}{
					"command": "npm",
					"args":    []string{"run", "dev"},
				},
				"mode":    "keepAlive",
				"workDir": "./api",
				"env": map[string]string{
					"DATABASE_URL": "postgresql://postgres:devpass@localhost:5432/myapp",
					"NODE_ENV":     "development",
				},
			},
		},
	}

	jsonBytes, _ := json.MarshalIndent(config, "", "  ")

	return string(jsonBytes)
}

// getMixedFormatTemplate returns a template mixing different command formats
func (tg *TemplateGenerator) getMixedFormatTemplate() string {
	config := map[string]interface{}{
		"version": "1.0",
		"commands": []map[string]interface{}{
			{
				"name":    "simple-echo",
				"command": "echo Hello from string format!",
				"mode":    "once",
			},
			{
				"name":    "list-files",
				"command": []string{"ls", "-la"},
				"mode":    "once",
			},
			{
				"name": "complex-docker",
				"command": map[string]interface{}{
					"command": "docker",
					"args":    []string{"run", "--rm", "-v", "${PWD}:/workspace", "alpine", "echo", "Hello from object format!"},
				},
				"mode": "once",
			},
			{
				"name":    "background-service",
				"command": []string{"python", "-m", "http.server", "8000"},
				"mode":    "keepAlive",
			},
		},
	}

	jsonBytes, _ := json.MarshalIndent(config, "", "  ")

	return string(jsonBytes)
}

// getFullstackTemplate returns a comprehensive full-stack development template
func (tg *TemplateGenerator) getFullstackTemplate() string {
	config := map[string]interface{}{
		"version": "1.0",
		"commands": []map[string]interface{}{
			{
				"name":    "setup-database",
				"command": []string{"docker", "run", "-d", "--name", "dev-postgres", "-p", "5432:5432", "postgres:13"},
				"mode":    "keepAlive",
				"env": map[string]string{
					"POSTGRES_PASSWORD": "devpass",
					"POSTGRES_DB":       "fullstack_app",
				},
			},
			{
				"name":    "setup-redis",
				"command": "docker run -d --name dev-redis -p 6379:6379 redis:alpine",
				"mode":    "keepAlive",
			},
			{
				"name":    "wait-for-services",
				"command": []string{"sleep", "5"},
				"mode":    "once",
			},
			{
				"name": "install-backend-deps",
				"command": map[string]interface{}{
					"command": "npm",
					"args":    []string{"install"},
				},
				"mode":    "once",
				"workDir": "./backend",
			},
			{
				"name":    "install-frontend-deps",
				"command": "npm install",
				"mode":    "once",
				"workDir": "./frontend",
			},
			{
				"name":    "run-migrations",
				"command": []string{"npm", "run", "db:migrate"},
				"mode":    "once",
				"workDir": "./backend",
			},
			{
				"name": "start-backend",
				"command": map[string]interface{}{
					"command": "npm",
					"args":    []string{"run", "dev"},
				},
				"mode":    "keepAlive",
				"workDir": "./backend",
				"env": map[string]string{
					"NODE_ENV":     "development",
					"PORT":         "3001",
					"DATABASE_URL": "postgresql://postgres:devpass@localhost:5432/fullstack_app",
					"REDIS_URL":    "redis://localhost:6379",
				},
			},
			{
				"name":    "start-frontend",
				"command": "npm run dev",
				"mode":    "keepAlive",
				"workDir": "./frontend",
				"env": map[string]string{
					"VITE_API_URL": "http://localhost:3001",
				},
			},
		},
	}

	jsonBytes, _ := json.MarshalIndent(config, "", "  ")

	return string(jsonBytes)
}
