# seqr - Enhanced Command Queue Runner

## 🛠️ Technology Stack & Standards

### Core Technologies
- **Language:** Go 1.25+ (latest version of Go, for speed, cross-platform support, and excellent concurrency)
- **CLI Framework:** Standard library `flag` package (minimal dependencies)
- **Configuration:** JSON with standard library `encoding/json`
- **Process Management:** Standard library `os/exec` and `os/signal`
- **Testing:** Standard library `testing` package with table-driven tests

### Architecture Principles
- **Minimal dependencies** - use standard library where possible to reduce maintenance burden
- **Clear boundaries** - separate concerns with well-defined interfaces
- **Testable design** - all components easily unit testable in isolation
- **Modular structure** - features can be modified independently
- **Simple data flow** - linear, predictable data transformations

### Distribution & Publishing
- **Repository:** GitHub public repository with proper README and documentation
- **Primary Installation:** Pre-built binaries for Windows, macOS, and Linux (no dependencies)
- **Secondary Options:** `go install` for Go developers, potential homebrew/chocolatey packages
- **Releases:** GitHub releases with semantic versioning and automated builds
- **Documentation:** Installation instructions, usage examples, and .queue.json schema

### Coding Standards

#### Core Principles
- **No explanatory comments** - code should be self-documenting through clear naming
- **Minimal comments** - only for complex business logic or non-obvious decisions
- **Follow Go conventions** - gofmt, golint, effective Go practices
- **Single responsibility** - each function/struct has one clear purpose
- **Error handling** - explicit error returns, no panics in normal operation
- **Naming** - descriptive names that eliminate need for comments

#### Maintainability Best Practices
- **Small functions** - maximum 20-30 lines, single level of abstraction
- **Interface segregation** - small, focused interfaces over large ones
- **Dependency injection** - avoid global state, pass dependencies explicitly
- **Table-driven tests** - comprehensive test coverage with clear test cases
- **Package organization** - logical separation by domain, not by layer
- **Immutable data** - prefer value types and immutable structs where possible
- **Fail fast** - validate inputs early, return errors immediately
- **Consistent patterns** - use same patterns throughout codebase for similar problems

## Feature 1: Queue Configuration Flexibility

**User Story:** As a developer, I want to support multiple formats for .queue.json files including objects with command and arguments, arrays of commands, and simple string commands, so that I can accommodate various use cases and improve usability.

### Acceptance Criteria
- Support object format with command and arguments properties
- Support array format with command strings  
- Support simple string commands for basic use cases
- Maintain backward compatibility with existing configurations
- Provide clear validation and error messages for invalid formats

### Design

**Architecture Pattern:** Configuration parser with multiple format support

```
.queue.json → [Parser] → Unified Config Structure
```

**What:** Configuration system that accepts multiple JSON formats and converts them to internal representation
**Why:** Different use cases benefit from different configuration styles
**How:** Parse different formats and normalize to common internal structure

### Tasks
- [x] **1.1** Extend configuration parser to support object, array, and string formats
- [x] **1.2** Implement format detection and validation
- [x] **1.3** Create normalization to unified internal structure
- [x] **1.4** Add error handling for invalid formats
- [x] **1.5** Create unit tests for all supported formats
---

## ⚡ Feature 2: Enhanced Verbose Logging

**User Story:** As a developer debugging command execution, I want detailed output under the --verbose flag that logs the exact output of each command, including live streaming of outputs during execution, so that I can enable real-time monitoring and debugging.

### Acceptance Criteria
- Log exact output of each command under --verbose flag
- Stream command outputs live during execution
- Include timestamps and command identification
- Distinguish between stdout and stderr clearly
- Maintain backward compatibility with quiet mode

### Design

**Architecture Pattern:** Real-time output streaming with verbose logging

```
Command Output → [Output Capture] → Live Stream + Verbose Log
```

**What:** Enhanced logging system that captures and displays command output in real-time when verbose flag is used
**Why:** Real-time feedback is essential for debugging long-running processes
**How:** Capture stdout/stderr and stream to console with timestamps when --verbose is enabled

### Tasks
- [ ] **2.1** Add --verbose flag support to CLI
- [ ] **2.2** Implement real-time output capture from commands
- [ ] **2.3** Add timestamps and command identification to verbose output
- [ ] **2.4** Distinguish stdout/stderr in verbose display
- [ ] **2.5** Maintain live streaming during command execution
- [ ] **2.6** Test verbose logging with various command outputs

---

## 🔄 Feature 3: Template Generation

**User Story:** As a new seqr user, I want a --init CLI flag that generates example queue configuration files with predefined structures, so that I can enable quick setup without learning the format from scratch.

### Acceptance Criteria
- Generate example .queue.json files with --init flag
- Include examples of all supported configuration formats
- Provide helpful comments explaining configuration options
- Handle existing files gracefully with user prompts

### Design

**Architecture Pattern:** Template generator with predefined examples

```
--init → [Template Generator] → .queue.json examples
```

**What:** CLI command that generates example configuration files with documentation
**Why:** Reduces onboarding friction and provides immediate working examples
**How:** Predefined templates that demonstrate different configuration formats

### Tasks
- [ ] **3.1** Add --init flag to CLI interface
- [ ] **3.2** Create example templates for different configuration formats
- [ ] **3.3** Implement template generation with helpful comments
- [ ] **3.4** Handle existing file conflicts with user prompts
- [ ] **3.5** Test template generation and validate generated files

---

## 💻 Feature 4: Graceful Process Termination

**User Story:** As a developer running long-running processes, I want a --kill CLI option to safely stop running queue executions, so that I can ensure proper cleanup of processes and resources.

### Acceptance Criteria
- Safely terminate running queue executions with --kill flag
- Ensure proper cleanup of all child processes and resources
- Attempt graceful shutdown before forcing termination
- Provide clear feedback on termination progress and results

### Design

**Architecture Pattern:** Process termination with cleanup

```
--kill → [Signal Handler] → Graceful Shutdown → Force Kill (if needed)
```

**What:** CLI option that safely terminates running seqr processes and their children
**Why:** Long-running processes need clean shutdown to prevent orphaned processes
**How:** Send termination signals with escalation from graceful to forced termination

### Tasks
- [ ] **4.1** Add --kill flag to CLI interface
- [ ] **4.2** Implement process tracking for running executions
- [ ] **4.3** Add graceful shutdown with SIGTERM
- [ ] **4.4** Implement force termination with SIGKILL after timeout
- [ ] **4.5** Ensure cleanup of all child processes
- [ ] **4.6** Test termination scenarios with various process types

---

## 🛑 Feature 5: Live Output Streaming

**User Story:** As a developer working with background services, I want real-time display of command outputs during execution, allowing the streaming to exit while keeping live processes running in the background, so that I can run other commands concurrently while maintaining necessary background processes.

### Acceptance Criteria
- Display command outputs in real-time during execution
- Allow users to exit streaming view while keeping processes running
- Support concurrent execution of other commands
- Maintain process state when switching between streaming and background modes
- Notify users of unexpected process terminations

### Design

**Architecture Pattern:** Background process management with live streaming

```
Commands → [Process Manager] → Background Processes + Live Output Stream
```

**What:** Process execution system that supports background processes with real-time output streaming
**Why:** Users need to monitor long-running services while maintaining ability to execute other commands
**How:** Detach processes to background while streaming their output, allow exit from streaming without killing processes

### Tasks
- [ ] **5.1** Implement background process execution for keepAlive commands
- [ ] **5.2** Add real-time output streaming during execution
- [ ] **5.3** Allow exit from streaming while keeping processes running
- [ ] **5.4** Support concurrent command execution
- [ ] **5.5** Add process monitoring and status notifications
- [ ] **5.6** Test background processes with live streaming