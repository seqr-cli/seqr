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
		"_comment": "String Format Example - Commands as simple strings (easiest to read and write)",
		"version":  "1.0",
		"_usage": map[string]interface{}{
			"description": "String format allows you to write commands exactly as you would in terminal",
			"examples": []string{
				"\"npm install\" - Simple command",
				"\"npm run build --production\" - Command with flags",
				"\"echo 'Hello World'\" - Command with quotes",
			},
			"pros": []string{
				"Most familiar - looks like terminal commands",
				"Easiest to read and write",
				"Good for simple commands without complex arguments",
			},
			"cons": []string{
				"Shell parsing can be tricky with quotes and spaces",
				"Less structured than object format",
			},
		},
		"commands": []map[string]interface{}{
			{
				"_comment": "Install dependencies - runs once and waits for completion",
				"name":     "install-deps",
				"command":  "npm install",
				"mode":     "once",
			},
			{
				"_comment": "Build the project - runs once after dependencies are installed",
				"name":     "build-project",
				"command":  "npm run build",
				"mode":     "once",
			},
			{
				"_comment": "Start the server - runs in background (keepAlive mode)",
				"name":     "start-server",
				"command":  "npm start",
				"mode":     "keepAlive",
				"env": map[string]string{
					"NODE_ENV": "production",
					"PORT":     "3000",
				},
				"_env_comment": "Environment variables are passed to the command process",
			},
		},
		"_modes_explained": map[string]string{
			"once":      "Command runs once and seqr waits for it to complete before continuing",
			"keepAlive": "Command starts and runs in background while seqr continues to next command",
		},
	}

	jsonBytes, _ := json.MarshalIndent(config, "", "  ")

	return string(jsonBytes)
}

// getArrayFormatTemplate returns a template using array command format
func (tg *TemplateGenerator) getArrayFormatTemplate() string {
	config := map[string]interface{}{
		"_comment": "Array Format Example - Commands as arrays of strings (best for complex arguments)",
		"version":  "1.0",
		"_usage": map[string]interface{}{
			"description": "Array format splits command and arguments into separate array elements",
			"examples": []string{
				"[\"npm\", \"install\"] - Command with no arguments",
				"[\"docker\", \"run\", \"-p\", \"3000:3000\", \"myapp\"] - Command with multiple flags",
				"[\"echo\", \"Hello World\"] - Handles spaces without quotes",
			},
			"pros": []string{
				"No shell parsing issues - arguments are explicit",
				"Handles spaces and special characters cleanly",
				"More predictable than string format",
			},
			"cons": []string{
				"More verbose than string format",
				"Requires splitting command from arguments manually",
			},
		},
		"commands": []map[string]interface{}{
			{
				"_comment": "Start Redis container - runs in background for the application",
				"name":     "setup-docker",
				"command":  []string{"docker", "run", "-d", "--name", "redis", "-p", "6379:6379", "redis:alpine"},
				"mode":     "keepAlive",
				"_note":    "Each array element is a separate argument - no shell parsing needed",
			},
			{
				"_comment": "Wait for Redis to be ready before continuing",
				"name":     "wait-for-redis",
				"command":  []string{"sleep", "2"},
				"mode":     "once",
			},
			{
				"_comment":      "Run tests with coverage in backend directory",
				"name":          "run-tests",
				"command":       []string{"npm", "test", "--", "--coverage"},
				"mode":          "once",
				"workDir":       "./backend",
				"_workdir_note": "workDir changes the working directory for this command only",
			},
			{
				"_comment": "Start the application server in background",
				"name":     "start-app",
				"command":  []string{"node", "server.js", "--port", "8080"},
				"mode":     "keepAlive",
				"workDir":  "./backend",
				"env": map[string]string{
					"REDIS_URL": "redis://localhost:6379",
				},
				"_env_note": "Environment variables are available to the command process",
			},
		},
		"_field_explanations": map[string]string{
			"name":    "Unique identifier for the command (used in logs and error messages)",
			"command": "Array where first element is executable, rest are arguments",
			"mode":    "Execution mode: 'once' (wait for completion) or 'keepAlive' (run in background)",
			"workDir": "Optional: Directory to run command in (relative to queue file location)",
			"env":     "Optional: Environment variables to set for the command",
		},
	}

	jsonBytes, _ := json.MarshalIndent(config, "", "  ")

	return string(jsonBytes)
}

// getObjectFormatTemplate returns a template using object command format
func (tg *TemplateGenerator) getObjectFormatTemplate() string {
	config := map[string]interface{}{
		"_comment": "Object Format Example - Commands as structured objects (most explicit and structured)",
		"version":  "1.0",
		"_usage": map[string]interface{}{
			"description": "Object format separates command executable from arguments in a structured way",
			"structure": map[string]string{
				"command": "Object with 'command' (executable) and 'args' (array of arguments)",
				"example": "{\"command\": \"docker\", \"args\": [\"run\", \"-p\", \"3000:3000\", \"myapp\"]}",
			},
			"pros": []string{
				"Most structured and explicit format",
				"Clear separation between executable and arguments",
				"Easy to programmatically generate and modify",
				"No ambiguity about command parsing",
			},
			"cons": []string{
				"Most verbose format",
				"Requires more typing than string or array formats",
			},
		},
		"commands": []map[string]interface{}{
			{
				"_comment": "Set up PostgreSQL database container for development",
				"name":     "database-setup",
				"command": map[string]interface{}{
					"command": "docker",
					"args":    []string{"run", "-d", "--name", "postgres", "-p", "5432:5432", "postgres:13"},
				},
				"mode": "keepAlive",
				"env": map[string]string{
					"POSTGRES_PASSWORD": "devpass",
					"POSTGRES_DB":       "myapp",
				},
				"_explanation": "Starts PostgreSQL in Docker container with environment variables for setup",
			},
			{
				"_comment": "Wait for database to be ready before running migrations",
				"name":     "wait-for-db",
				"command": map[string]interface{}{
					"command": "sleep",
					"args":    []string{"3"},
				},
				"mode": "once",
				"_tip": "Always wait for services to be ready before depending on them",
			},
			{
				"_comment": "Run database migrations to set up schema",
				"name":     "run-migrations",
				"command": map[string]interface{}{
					"command": "npm",
					"args":    []string{"run", "db:migrate"},
				},
				"mode":                 "once",
				"workDir":              "./api",
				"_workdir_explanation": "Runs in ./api directory where package.json with db:migrate script is located",
			},
			{
				"_comment": "Start the API server in development mode",
				"name":     "start-api",
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
				"_env_explanation": "DATABASE_URL connects to the PostgreSQL container started above",
			},
		},
		"_object_format_guide": map[string]interface{}{
			"command_structure": map[string]string{
				"command": "The executable name (e.g., 'docker', 'npm', 'node')",
				"args":    "Array of arguments passed to the command",
			},
			"equivalent_examples": map[string]interface{}{
				"string_format": "\"npm run build\"",
				"array_format":  []string{"npm", "run", "build"},
				"object_format": map[string]interface{}{
					"command": "npm",
					"args":    []string{"run", "build"},
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
		"_comment": "Mixed Format Example - Demonstrates all three command formats in one file",
		"version":  "1.0",
		"_format_guide": map[string]interface{}{
			"description": "You can mix different command formats in the same queue file",
			"when_to_use_each": map[string]string{
				"string_format": "Simple commands without complex arguments or spaces",
				"array_format":  "Commands with multiple arguments or spaces in arguments",
				"object_format": "When you want maximum clarity and structure",
			},
			"recommendation": "Choose the format that makes each command most readable",
		},
		"commands": []map[string]interface{}{
			{
				"_format":     "string",
				"_comment":    "String format - simple command, easy to read",
				"name":        "simple-echo",
				"command":     "echo Hello from string format!",
				"mode":        "once",
				"_why_string": "Simple command with no complex arguments - string format is clearest",
			},
			{
				"_format":    "array",
				"_comment":   "Array format - command with flags, no shell parsing issues",
				"name":       "list-files",
				"command":    []string{"ls", "-la"},
				"mode":       "once",
				"_why_array": "Flags are explicit as separate array elements",
			},
			{
				"_format":  "object",
				"_comment": "Object format - complex command with many arguments",
				"name":     "complex-docker",
				"command": map[string]interface{}{
					"command": "docker",
					"args":    []string{"run", "--rm", "-v", "${PWD}:/workspace", "alpine", "echo", "Hello from object format!"},
				},
				"mode":        "once",
				"_why_object": "Complex Docker command with volume mounts - object format makes structure clear",
			},
			{
				"_format":    "array",
				"_comment":   "Array format - background service with module flag",
				"name":       "background-service",
				"command":    []string{"python", "-m", "http.server", "8000"},
				"mode":       "keepAlive",
				"_why_array": "Python module syntax (-m flag) is clearer as separate arguments",
			},
		},
		"_execution_flow": []string{
			"1. simple-echo runs once and completes",
			"2. list-files runs once and completes",
			"3. complex-docker runs once and completes",
			"4. background-service starts and runs in background",
			"5. seqr keeps running to maintain background-service",
		},
		"_tips": []string{
			"Use 'seqr -v' to see detailed execution logs",
			"Background services (keepAlive) will restart if they crash",
			"Press Ctrl+C to stop all processes and exit seqr",
		},
	}

	jsonBytes, _ := json.MarshalIndent(config, "", "  ")

	return string(jsonBytes)
}

// getFullstackTemplate returns a comprehensive full-stack development template
func (tg *TemplateGenerator) getFullstackTemplate() string {
	config := map[string]interface{}{
		"_comment":     "Full-Stack Development Example - Complete development environment setup",
		"version":      "1.0",
		"_description": "This template sets up a complete full-stack development environment with database, cache, backend API, and frontend application",
		"_prerequisites": []string{
			"Docker installed and running",
			"Node.js and npm installed",
			"Backend code in ./backend directory with package.json",
			"Frontend code in ./frontend directory with package.json",
		},
		"_architecture": map[string]string{
			"database": "PostgreSQL running in Docker container on port 5432",
			"cache":    "Redis running in Docker container on port 6379",
			"backend":  "Node.js API server running on port 3001",
			"frontend": "Frontend development server (e.g., Vite, Create React App) with API proxy",
		},
		"commands": []map[string]interface{}{
			{
				"_step":    1,
				"_comment": "Start PostgreSQL database container for persistent data storage",
				"name":     "setup-database",
				"command":  []string{"docker", "run", "-d", "--name", "dev-postgres", "-p", "5432:5432", "postgres:13"},
				"mode":     "keepAlive",
				"env": map[string]string{
					"POSTGRES_PASSWORD": "devpass",
					"POSTGRES_DB":       "fullstack_app",
				},
				"_explanation": "Creates PostgreSQL container with database 'fullstack_app' and password 'devpass'",
			},
			{
				"_step":        2,
				"_comment":     "Start Redis container for caching and session storage",
				"name":         "setup-redis",
				"command":      "docker run -d --name dev-redis -p 6379:6379 redis:alpine",
				"mode":         "keepAlive",
				"_explanation": "Redis provides fast in-memory caching for the application",
			},
			{
				"_step":     3,
				"_comment":  "Wait for database and Redis containers to be fully ready",
				"name":      "wait-for-services",
				"command":   []string{"sleep", "5"},
				"mode":      "once",
				"_why_wait": "Services need time to initialize before accepting connections",
			},
			{
				"_step":    4,
				"_comment": "Install backend dependencies (package.json in ./backend)",
				"name":     "install-backend-deps",
				"command": map[string]interface{}{
					"command": "npm",
					"args":    []string{"install"},
				},
				"mode":         "once",
				"workDir":      "./backend",
				"_explanation": "Installs all npm packages required by the backend API",
			},
			{
				"_step":        5,
				"_comment":     "Install frontend dependencies (package.json in ./frontend)",
				"name":         "install-frontend-deps",
				"command":      "npm install",
				"mode":         "once",
				"workDir":      "./frontend",
				"_explanation": "Installs all npm packages required by the frontend application",
			},
			{
				"_step":        6,
				"_comment":     "Run database migrations to set up schema and initial data",
				"name":         "run-migrations",
				"command":      []string{"npm", "run", "db:migrate"},
				"mode":         "once",
				"workDir":      "./backend",
				"_explanation": "Sets up database tables and initial data (requires db:migrate script in backend/package.json)",
			},
			{
				"_step":    7,
				"_comment": "Start the backend API server in development mode",
				"name":     "start-backend",
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
				"_explanation": "Starts backend server with connections to database and Redis",
			},
			{
				"_step":    8,
				"_comment": "Start the frontend development server with hot reload",
				"name":     "start-frontend",
				"command":  "npm run dev",
				"mode":     "keepAlive",
				"workDir":  "./frontend",
				"env": map[string]string{
					"VITE_API_URL": "http://localhost:3001",
				},
				"_explanation": "Starts frontend dev server configured to proxy API requests to backend",
			},
		},
		"_execution_summary": []string{
			"Steps 1-2: Start infrastructure services (database, cache)",
			"Step 3: Wait for services to be ready",
			"Steps 4-5: Install dependencies for both backend and frontend",
			"Step 6: Set up database schema",
			"Steps 7-8: Start both backend and frontend servers",
		},
		"_after_setup": map[string]interface{}{
			"backend_url":  "http://localhost:3001",
			"frontend_url": "Check console output for frontend dev server URL (usually http://localhost:3000 or http://localhost:5173)",
			"database":     "postgresql://postgres:devpass@localhost:5432/fullstack_app",
			"redis":        "redis://localhost:6379",
		},
		"_troubleshooting": map[string][]string{
			"port_conflicts": []string{
				"If ports 5432, 6379, or 3001 are in use, stop existing services",
				"Use 'docker ps' to see running containers",
				"Use 'docker stop <container-name>' to stop conflicting containers",
			},
			"docker_issues": []string{
				"Ensure Docker is running before starting seqr",
				"If containers fail to start, try 'docker system prune' to clean up",
			},
			"dependency_issues": []string{
				"Ensure package.json files exist in ./backend and ./frontend directories",
				"Check that npm is installed and accessible",
			},
		},
	}

	jsonBytes, _ := json.MarshalIndent(config, "", "  ")

	return string(jsonBytes)
}
