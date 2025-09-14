package config

// ExampleConfig returns a sample configuration for documentation and testing
func ExampleConfig() *Config {
	return &Config{
		Version: "1.0",
		Commands: []Command{
			{
				Name:    "setup-database",
				Command: "docker",
				Args:    []string{"run", "-d", "--name", "test-db", "postgres:13"},
				Mode:    ModeKeepAlive,
				Env: map[string]string{
					"POSTGRES_PASSWORD": "testpass",
				},
			},
			{
				Name:    "run-migrations",
				Command: "npm",
				Args:    []string{"run", "migrate"},
				Mode:    ModeOnce,
				WorkDir: "./backend",
			},
			{
				Name:    "start-server",
				Command: "npm",
				Args:    []string{"start"},
				Mode:    ModeKeepAlive,
				WorkDir: "./backend",
				Env: map[string]string{
					"NODE_ENV": "development",
					"PORT":     "3000",
				},
			},
		},
	}
}
