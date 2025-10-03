# seqr

A simple, reliable process runner for multi-command workflows with live output, persistent logs, and graceful lifecycle management.

- Queue defined in a single JSON file
- Live watch mode with real-time, colorized output
- Persistent logs per process
- Graceful start, status, and kill controls

## Quick start

```bash
# Install from source
make install

# Initialize example queue config
seqr --init

# Run a queue
seqr -f .queue.json

# Watch live output
seqr --watch
```

## CLI

- `-f, --file` Path to queue configuration file (default: .queue.json)
- `-v, --verbose` Verbose output with execution details and colors
- `-h, --help` Show help
- `--version` Show version
- `--init` Generate example queue configs
- `--kill` Gracefully stop running seqr processes
- `--status` Show status of running processes
- `--watch` Watch live processes and their real-time output

## Example queue

```json
{
  "name": "fullstack-dev",
  "concurrency": 3,
  "env": {
    "NODE_ENV": "development"
  },
  "processes": [
    {
      "name": "start-server",
      "cmd": "npm run dev",
      "cwd": "./server",
      "restart": true
    },
    {
      "name": "build-web",
      "cmd": "npm run build",
      "cwd": "./web"
    },
    {
      "name": "watch-web",
      "cmd": "npm run start",
      "cwd": "./web",
      "restart": true,
      "dependsOn": ["build-web"]
    }
  ]
}
```

## Common workflows

```bash
# Run with verbose output
seqr -v -f examples/fullstack-dev.queue.json

# Watch in another terminal
seqr --watch

# Kill all running processes managed by seqr
seqr --kill

# Inspect logs
ls -la ~/.seqr/logs/
tail -f ~/.seqr/logs/start-server.log
```

## Architecture

- CLI layer: Command parsing and user interaction
- Executor: Starts commands, streams output in real time
- Process manager: Tracks background processes and lifecycle
- Background logger: Persists output to disk
- Color system: Cross-platform colorized terminal output

Process lifecycle:
1. Command parsing
2. Execution with streaming
3. Persistent logging
4. Monitoring for health and status
5. Graceful termination and cleanup

## File layout

```
~/.seqr/
└── logs/
    ├── process1.log
    └── process2.log

/tmp/seqr-processes.json  Runtime process tracking
```

## Development

```bash
make build    # Build binary
make install  # Build and install to /usr/local/bin
make test     # Run tests
make dev      # Run from source
```

## Contributing

We welcome contributions. Focus areas:
- Real-time output improvements for `--watch`
- Log rotation and compression
- Web monitoring interface
- Plugin system for custom command types
- Integrations with popular dev tools

## Why seqr

- Single source of truth: One JSON file to describe your workflow
- Developer ergonomics: Live watch and readable logs
- Operations-friendly: Status, kill, and persistent logging
- Minimal footprint: Small, focused tool that plays well with existing scripts
