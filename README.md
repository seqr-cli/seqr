# seqr

AI-Safe Command Queue Runner. Execute commands sequentially from a JSON configuration file. Supports both one-time commands and long-running background processes.

## Installation

### Via Go (recommended for Go users)
```bash
go install github.com/seqr-cli/seqr/cmd/seqr@latest
```

### Via installer script
```bash
curl -sSL https://raw.githubusercontent.com/seqr-cli/seqr/main/install.sh | bash
```

## Usage

```bash
# Build
go build -o seqr ./cmd/seqr

# Create a new .queue.json file
./seqr --init

# Run commands from .queue.json
./seqr

# Run commands from custom file
./seqr -f my-queue.json

# Run with verbose output
./seqr -v

# Custom file with verbose output
./seqr -f queue.json -v

# Show status of running processes
./seqr --status

# Kill all running processes
./seqr --kill

# Show help
./seqr --help

# Show version
./seqr --version
```

## Configuration

Create a `.queue.json` file. The configuration supports several formats:

### Full Format
```json
{
  "version": "1.0",
  "commands": [
    {
      "name": "start-db",
      "command": "docker",
      "args": ["run", "-d", "-p", "5432:5432", "postgres"],
      "mode": "keepAlive"
    },
    {
      "name": "migrate",
      "command": "npm",
      "args": ["run", "migrate"],
      "mode": "once",
      "workDir": "./backend"
    },
    {
      "name": "start-server",
      "command": "npm",
      "args": ["start"],
      "mode": "keepAlive",
      "workDir": "./backend",
      "env": {
        "NODE_ENV": "development"
      },
      "concurrent": false
    }
  ]
}
```

### Simple Formats
- **Array of commands**: `[{"command": "echo", "args": ["hello"]}]`
- **Single command**: `{"command": "echo", "args": ["hello"]}`
- **Simple string**: `"echo hello"`
- **Array of strings**: `["echo hello", "echo world"]`

## Modes

- `once`: Run command and wait for completion
- `keepAlive`: Start command and keep it running

## Options

- `workDir`: Working directory for command
- `env`: Environment variables (object)
- `args`: Command arguments (array)
- `concurrent`: Run command concurrently with others (boolean, default: false)

## CLI Flags

- `-f, --file`: Path to queue configuration file (default: .queue.json)
- `-v, --verbose`: Enable verbose output with execution details
- `-h, --help`: Show help message
- `--version`: Show version information
- `--init`: Generate example queue configuration files
- `--kill`: Kill running seqr processes
- `--status`: Show status of running seqr processes

## Development

```bash
make build    # Build binary
make install  # Build and install binary to /usr/local/bin
make test     # Run tests
make dev      # Run from source
```

---

*Hackathon project - use at your own risk*