# seqr

🚀 **AI-Safe Command Queue Runner** with **Live Monitoring & Colored Output**

Execute commands sequentially from a JSON configuration file. Supports both one-time commands and long-running background processes with real-time monitoring, colored output, and persistent logging.

## Installation

### Via Go (recommended for Go users)
```bash
go install github.com/seqr-cli/seqr/cmd/seqr@latest
```

### Via installer script
```bash
curl -sSL https://raw.githubusercontent.com/seqr-cli/seqr/main/install.sh | bash
```

## ✨ Features

- 🎨 **Colored Output**: Automatic command type detection with color-coded output
- 👀 **Live Monitoring**: Watch running processes with real-time output
- 📝 **Persistent Logging**: Background processes log output for later review
- 🔄 **Process Management**: Graceful termination with status tracking
- ⚡ **Concurrent Execution**: Run commands in parallel when specified
- 🎯 **Cross-Platform**: Works on macOS, Linux, and Windows

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

# Run with verbose output (shows live colored output)
./seqr -v

# Custom file with verbose output
./seqr -f queue.json -v

# 🎯 NEW: Watch live processes and their output
./seqr --watch

# Show status of running processes
./seqr --status

# Kill all running processes
./seqr --kill

# Show help
./seqr --help

# Show version
./seqr --version
```

### Example Output with Colors

```bash
[15:04:05.000] [docker] [start-db] ✓ Starting PostgreSQL...
[15:04:05.283] [docker] [start-db] ✓ PostgreSQL is ready
[15:04:05.310] [npm] [migrate] ❌ Migration completed
[15:04:05.315] [npm] [start-server] ✓ Server running on http://localhost:3000
```

### Live Watching

```bash
$ seqr --watch
🔍 Watching 2 running seqr process(es):

📊 PID 12345: start-server
   Command: npm [start]
   Mode: keepAlive
   Started: 2025-01-15 15:04:05 (2m30s ago)
   Status: Running
   Recent Output:
     [15:04:05.000] [npm] [start-server] ✓ Server running on http://localhost:3000
     [15:06:05.000] [npm] [start-server] ✓ Request processed: GET /api/users

📊 PID 12346: start-db
   Command: docker [run, postgres]
   Mode: keepAlive
   Started: 2025-01-15 15:04:05 (2m30s ago)
   Status: Running

🎯 Live output will appear below as processes generate it:
💡 Press Ctrl+C to stop watching
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

## 🎨 Colored Output & Command Types

seqr automatically detects command types and applies appropriate colors:

- 🐳 **docker**: Blue - Container operations
- ⚛️ **vite**: Purple - Frontend development
- 🟢 **node**: Green - Node.js applications
- 📦 **npm/yarn/pnpm**: Red/Cyan/Purple - Package managers
- ⚙️ **exec**: White - Other executables

### Color Support

- **stdout**: Green checkmark (✓)
- **stderr**: Red X mark (❌)
- **Timestamps**: Gray
- **Command names**: Cyan
- **Command types**: Color-coded

Colors are automatically disabled when:
- `NO_COLOR` environment variable is set
- Output is not a TTY
- `TERM=dumb` is detected

## 📝 Persistent Logging

Background processes automatically log their output to persistent files:

- **Location**: `~/.seqr/logs/` (or temp directory as fallback)
- **Format**: `[timestamp] [type] [name] [icon] message`
- **Retention**: Logs persist even after processes are killed
- **Cleanup**: Old logs are automatically cleaned up (7+ days)

### Log Files

```bash
~/.seqr/logs/
├── start-server.log
├── start-db.log
└── migrate.log
```

## 👀 Live Process Monitoring

The `--watch` command provides comprehensive monitoring:

### Active Processes
- Real-time status of running processes
- Process IDs, commands, and uptime
- Recent output from each process
- Live output streaming

### Historical Logs
- Access logs from stopped processes
- File sizes and modification times
- Persistent storage across sessions

### Example Watch Output

```bash
$ seqr --watch
🔍 Watching 2 running seqr process(es):

📊 PID 12345: start-server
   Command: npm [start]
   Mode: keepAlive
   Started: 2025-01-15 15:04:05 (5m ago)
   Status: Running
   Recent Output:
     [15:04:05.000] [npm] [start-server] ✓ Server listening on port 3000
     [15:09:05.000] [npm] [start-server] ✓ API request: GET /users

📁 Log files available from 1 stopped process(es):
   📄 migrate (2025-01-15 15:03:45, 2.1 KB)

🎯 Live output will appear below as processes generate it:
💡 Press Ctrl+C to stop watching
```

## CLI Flags

- `-f, --file`: Path to queue configuration file (default: .queue.json)
- `-v, --verbose`: Enable verbose output with execution details and colors
- `-h, --help`: Show help message
- `--version`: Show version information
- `--init`: Generate example queue configuration files
- `--kill`: Kill running seqr processes gracefully
- `--status`: Show status of running seqr processes
- `--watch`: 🎯 **NEW** - Watch live processes and their real-time output

## Development

```bash
make build    # Build binary
make install  # Build and install binary to /usr/local/bin
make test     # Run tests
make dev      # Run from source
```

### Testing the New Features

```bash
# Test colored output
./seqr -v -f examples/fullstack-dev.queue.json

# Test watching (in another terminal)
./seqr --watch

# Test process killing
./seqr --kill

# Check persistent logs
ls -la ~/.seqr/logs/
cat ~/.seqr/logs/start-server.log
```

## 🔧 Architecture

### Core Components

- **CLI Layer**: Command parsing and user interaction
- **Executor**: Command execution with streaming and monitoring
- **Process Manager**: Background process lifecycle management
- **Background Logger**: Persistent output logging
- **Color System**: Cross-platform colored terminal output

### Process Lifecycle

1. **Command Parsing** → Detect command type and apply colors
2. **Execution** → Stream output with real-time formatting
3. **Background Logging** → Persist output to disk
4. **Monitoring** → Track process status and health
5. **Termination** → Graceful shutdown with cleanup

### File Structure

```
~/.seqr/
└── logs/           # Persistent process logs
    ├── process1.log
    └── process2.log

/tmp/seqr-processes.json  # Runtime process tracking
```

## 🤝 Contributing

This project welcomes contributions! Areas for improvement:

- Real-time process output streaming in `--watch`
- Log rotation and compression
- Web-based monitoring interface
- Plugin system for custom command types
- Integration with popular dev tools

---

*🚀 Enhanced with live monitoring, colors, and persistent logging - Production ready!*