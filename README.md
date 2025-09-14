# seqr

Run commands in sequence. Perfect for development workflows.

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

# Run with default .queue.json
./seqr

# Run with custom file
./seqr --file my-queue.json
```

## Configuration

Create a `.queue.json` file:

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
      }
    }
  ]
}
```

## Modes

- `once`: Run command and wait for completion
- `keepAlive`: Start command and keep it running

## Options

- `workDir`: Working directory for command
- `env`: Environment variables
- `args`: Command arguments

## Development

```bash
make build    # Build binary
make install  # Build and install binary to /usr/local/bin
make test     # Run tests
make dev      # Run from source
```

---

*Hackathon project - use at your own risk*