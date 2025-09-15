package config

// Example demonstrates how to use the normalizer with different command formats
func ExampleNormalizer() {
	normalizer := NewNormalizer()

	// Example 1: String format - "npm run build"
	stringCmd, _ := normalizer.NormalizeCommand("npm run build", "build-app", ModeOnce, "", nil)
	// Result: Command="npm", Args=["run", "build"]

	// Example 2: Array format - ["docker", "run", "nginx"]
	arrayCmd, _ := normalizer.NormalizeCommand([]interface{}{"docker", "run", "nginx"}, "web-server", ModeKeepAlive, "", nil)
	// Result: Command="docker", Args=["run", "nginx"]

	// Example 3: Object format - {"command": "node", "args": ["server.js"]}
	objectFormat := map[string]interface{}{
		"command": "node",
		"args":    []interface{}{"server.js", "--port", "3000"},
	}
	objectCmd, _ := normalizer.NormalizeCommand(objectFormat, "api-server", ModeKeepAlive, "", nil)
	// Result: Command="node", Args=["server.js", "--port", "3000"]

	// All commands are now in the unified Command structure
	_ = stringCmd
	_ = arrayCmd
	_ = objectCmd
}

// ExampleNormalizeConfig demonstrates normalizing an entire configuration
func ExampleNormalizeConfig() {
	normalizer := NewNormalizer()

	// Mixed format configuration
	configData := map[string]interface{}{
		"version": "1.0",
		"commands": []interface{}{
			map[string]interface{}{
				"name":    "build",
				"command": "npm run build", // String format
				"mode":    "once",
			},
			map[string]interface{}{
				"name":    "start",
				"command": []interface{}{"node", "server.js"}, // Array format
				"mode":    "keepAlive",
			},
			map[string]interface{}{
				"name": "docker",
				"command": map[string]interface{}{ // Object format
					"command": "docker",
					"args":    []interface{}{"run", "nginx"},
				},
				"mode": "keepAlive",
			},
		},
	}

	config, _ := normalizer.NormalizeConfig(configData)
	// All commands are now normalized to the unified structure
	_ = config
}
