# CLI Package

This package implements the command-line interface for seqr with minimal required arguments and smart defaults.

## Design Principles

- **Minimal dependencies**: Uses only the standard library `flag` package
- **Smart defaults**: Works out of the box with `.queue.json` in current directory
- **Progressive disclosure**: Advanced options available but not required
- **Clear feedback**: Specific, actionable error messages

## Architecture

```
CLI Args → [Parser] → [Validator] → [Orchestrator] → Output
```

The CLI follows a simple pipeline:
1. **Parser**: Parses command-line arguments using standard library flags
2. **Validator**: Validates options (minimal validation during parse)
3. **Orchestrator**: Coordinates config loading and executor creation
4. **Output**: Provides clear feedback and execution results

## Interface

### Core Types

- `CLIOptions`: Holds all command-line configuration options
- `CLI`: Main CLI struct implementing the Interface
- `Interface`: Contract for CLI implementations

### Command-Line Options

| Flag | Description | Default |
|------|-------------|---------|
| `-f <file>` | Path to queue configuration file | `.queue.json` |
| `-v` | Enable verbose output with execution details | `false` |
| `-h`, `-help` | Show help message | `false` |

### Usage Examples

```bash
# Default behavior - look for .queue.json in current directory
seqr

# Specify custom configuration file
seqr -f my-queue.json

# Enable verbose output
seqr -v

# Combine options
seqr -f queue.json -v

# Show help
seqr -h
```

## Error Handling

The CLI provides clear, actionable error messages:

- **Parse errors**: Invalid command-line arguments
- **Config errors**: Missing or malformed configuration files
- **Execution errors**: Command failures with detailed context

## Signal Handling

The CLI properly handles interrupt signals (Ctrl+C) for graceful shutdown:
- Catches SIGINT and SIGTERM
- Cancels execution context
- Stops executor and all running processes
- Exits cleanly

## Testing

The package includes comprehensive tests covering:
- Argument parsing with various flag combinations
- Help display functionality
- Error handling scenarios
- Integration with executor and config packages