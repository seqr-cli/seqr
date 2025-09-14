# seqr - AI-Safe Command Queue Runner

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

## Feature 1: Command Queue Configuration

**User Story:** As a developer using AI tools, I want to define a sequence of commands in a configuration file, so that I can ensure they always execute in the correct order without manual intervention.

### Acceptance Criteria
- System parses .queue.json files into Go structs
- Maintains execution order from configuration  
- Displays clear error messages for malformed files
- Supports optional parameters (working directory, flags)

### Design

**Architecture Pattern:** Simple data pipeline with validation gateway

```
.queue.json → [Parser] → [Validator] → Validated Config
```

**What:** Configuration layer that transforms JSON into validated internal representation
**Why:** JSON provides universal readability for AI tools while validation ensures system reliability
**How:** Single-pass parsing with immediate validation to fail fast on invalid configurations

**Data Flow:**
1. File system reads JSON configuration
2. Parser deserializes into structured data
3. Validator applies business rules (mode validation, required fields)
4. Clean configuration object passed to execution layer

**Error Boundaries:** All parsing and validation errors caught at this layer, preventing invalid data from reaching execution engine

**Maintainability:** Clear separation allows configuration format changes without affecting execution logic

### Tasks
- [x] **1.1** Create Go module with standard project structure (cmd/, internal/, pkg/)
- [x] **1.2** Define configuration data structures with JSON binding
- [x] **1.3** Implement file loading with proper error handling
- [x] **1.4** Build validation layer with comprehensive rule checking
- [x] **1.5** Create unit tests covering valid/invalid configurations and edge cases
- [x] **1.6** Ensure that all rules and requirements defined in spec.md, as well as the specifics outlined for feature 1, are followed for tasks 1.1 to 1.5. Fix any discrepancies
---

## ⚡ Feature 2: Sequential Execution with Error Handling

**User Story:** As a developer, I want commands to execute sequentially with proper error handling, so that failed commands don't cause subsequent commands to run in an invalid state.

### Acceptance Criteria
- Commands run one at a time in defined order
- Execution stops if any command fails
- System proceeds only after successful command completion
- Clear failure reporting with command identification

### Design

**Architecture Pattern:** Sequential state machine with fail-fast semantics

```
Command Queue → [Executor] → Success → Next Command
                     ↓
                   Failure → Stop & Report
```

**What:** Linear execution engine that maintains strict ordering and error boundaries
**Why:** AI workflows require predictable behavior - partial execution creates undefined system states
**How:** State machine approach where each command completion determines next state

**Execution States:**
- **Ready:** Waiting to execute next command
- **Running:** Command in progress
- **Success:** Command completed, advance to next
- **Failed:** Command failed, terminate sequence

**Error Philosophy:** Fail-fast with clear error context - no recovery attempts, no partial states

### Tasks
- [x] **2.1** Design execution engine interface with clear state management
- [x] **2.2** Implement sequential command processor with state tracking
- [x] **2.3** Build error handling with detailed failure context
- [x] **2.4** Add execution reporting and status output
- [x] **2.5** Create comprehensive test suite for all execution paths
- [x] **2.6** Ensure that all rules and requirements defined in spec.md, as well as the specifics outlined for feature 2, are followed for tasks 2.1 to 2.5. Fix any discrepancies

---

## 🔄 Feature 3: Multiple Command Modes

**User Story:** As a developer, I want to run different types of commands including one-time tasks and persistent services, so that I can handle both batch operations and long-running processes in the same workflow.

### Acceptance Criteria
- **"once" mode**: Execute and wait for completion before proceeding
- **"keepAlive" mode**: Start process and keep running while proceeding
- Manage multiple keepAlive processes simultaneously
- Detect and report unexpected process exits

### Design

**Architecture Pattern:** Dual-mode process orchestrator with lifecycle management

```
Command → [Mode Router] → Once: Execute & Wait
                       → KeepAlive: Execute & Track

Background: [Process Monitor] → Status Updates
```

**What:** Process management system that handles synchronous and asynchronous execution patterns
**Why:** AI workflows need both setup tasks (synchronous) and services (asynchronous) in single pipeline
**How:** Mode-based routing with separate lifecycle management for each execution type

**Process Lifecycle:**
- **Once Mode:** Start → Wait → Complete → Continue
- **KeepAlive Mode:** Start → Track → Monitor → (Continue Immediately)

**Monitoring Strategy:** Background goroutine tracks all keepAlive processes, reports status changes, handles unexpected exits

### Tasks
- [x] **3.1** Design process manager with mode-based execution routing
- [x] **3.2** Implement synchronous execution for "once" mode commands
- [x] **3.3** Build asynchronous execution with process tracking for "keepAlive" mode
- [x] **3.4** Create background monitoring system for process health
- [x] **3.5** Add process status reporting and lifecycle event handling
- [x] **3.6** Test both execution modes with real commands and edge cases
- [x] **3.7** Ensure that all rules and requirements defined in spec.md, as well as the specifics outlined for feature 3, are followed for tasks 3.1 to 3.6. Fix any discrepancies

---

## 💻 Feature 4: Simple CLI Interface

**User Story:** As a developer, I want a simple CLI interface with helpful options, so that I can easily run my command queues and get appropriate feedback.

### Acceptance Criteria
- Default behavior: look for .queue.json in current directory
- `-f <file>` flag: specify alternative queue file
- `-v` flag: enable verbose output with execution details
- Display running status and success/failure for each command
- Provide execution summary upon completion

### Design

**Architecture Pattern:** Minimal CLI facade with smart defaults

```
CLI Args → [Parser] → [Validator] → [Orchestrator] → Output
```

**What:** Thin command-line interface that bridges user input to execution engine
**Why:** Simple interface reduces cognitive load for both humans and AI tools
**How:** Convention-over-configuration approach with sensible defaults and minimal required input

**Interface Philosophy:**
- **Zero-config default:** `seqr` just works with `.queue.json`
- **Progressive disclosure:** Advanced options available but not required
- **Clear feedback:** Every action produces clear, actionable output

**Output Strategy:** 
- **Quiet by default:** Show only essential information
- **Verbose on demand:** Detailed execution information when requested
- **Error clarity:** Specific, actionable error messages

### Tasks
- [x] **4.1** Design CLI interface with minimal required arguments
- [x] **4.2** Implement argument parsing with validation and defaults
- [ ] **4.3** Create output formatting system (quiet/verbose modes)
- [ ] **4.4** Build help system with clear usage examples
- [ ] **4.5** Add execution summary and progress indicators
- [ ] **4.6** Test CLI with various argument combinations and error scenarios
- [ ] **4.7** Ensure that all rules and requirements defined in spec.md, as well as the specifics outlined for feature 4, are followed for tasks 4.1 to 4.6. Fix any discrepancies

---

## 🛑 Feature 5: Signal Handling

**User Story:** As a developer, I want proper signal handling for running processes, so that I can cleanly terminate all processes when needed.

### Acceptance Criteria
- Ctrl+C gracefully terminates all running processes
- Forward signals to all keepAlive processes
- Wait for clean shutdown before exiting
- Force kill processes that don't terminate within reasonable time

### Design

**Architecture Pattern:** Graceful shutdown coordinator with escalating termination

```
Signal → [Handler] → Graceful Shutdown → Timeout → Force Kill
                           ↓
                    [Process Tree] → Forward Signal → Wait
```

**What:** Signal management system that ensures clean termination of all managed processes
**Why:** Prevents orphaned processes, resource leaks, and corrupted state in long-running services
**How:** Hierarchical shutdown with escalating force levels and timeout boundaries

**Shutdown Sequence:**
1. **Signal Reception:** Catch SIGINT/SIGTERM from user or system
2. **Graceful Phase:** Forward signals to all child processes
3. **Wait Phase:** Allow processes time to clean up (5-second timeout)
4. **Force Phase:** Kill unresponsive processes
5. **Exit Phase:** Clean application termination

**Process Tree Management:** Track parent-child relationships to ensure complete cleanup

### Tasks
- [ ] **5.1** Design signal handling system with proper goroutine management
- [ ] **5.2** Implement graceful shutdown coordinator with timeout handling
- [ ] **5.3** Build signal forwarding mechanism for child processes
- [ ] **5.4** Create escalating termination strategy (graceful → force)
- [ ] **5.5** Add process tree tracking for complete cleanup
- [ ] **5.6** Test signal handling with complex process hierarchies and edge cases
- [ ] **5.7** Ensure that all rules and requirements defined in spec.md, as well as the specifics outlined for feature 5, are followed for tasks 5.1 to 5.6. Fix any discrepancies

---

## 📦 Feature 6: Distribution & Publishing

**User Story:** As a developer, I want to easily install and distribute seqr across different platforms, so that it can be used in various development environments and shared with the community.

### Acceptance Criteria
- Pre-built binaries available for Windows, macOS, and Linux (no Go installation required)
- GitHub repository with proper documentation and examples
- Multiple installation methods: direct download, `go install` for Go users, package managers
- Automated releases with semantic versioning
- Clear installation and usage documentation with copy-paste commands

### Design

**Architecture Pattern:** Multi-platform build pipeline with automated distribution

```
Source Code → [CI/CD] → Cross-Platform Builds → GitHub Releases
                                ↓
                        Package Managers (go install, homebrew)
```

**What:** Complete distribution system that makes seqr easily installable across platforms
**Why:** Tool adoption requires frictionless installation and clear documentation
**How:** Automated build pipeline with multiple distribution channels

**Build Strategy:**
- **Cross-compilation:** Single codebase builds for all target platforms as standalone binaries
- **Zero dependencies:** Statically linked binaries that work without Go installation
- **Automated releases:** CI/CD pipeline triggered by version tags
- **Multiple channels:** Direct binary downloads (primary), `go install` (Go developers), package managers (future)

**Documentation Philosophy:**
- **Quick start:** Get users running in under 2 minutes
- **Examples first:** Show real .queue.json files before explaining theory
- **Progressive depth:** Basic usage → advanced features → customization

### Tasks

- [ ] **6.0** Clean-up the directory and remove unneeded or redundant files/folders 
- [ ] **6.1** Set up GitHub repository with proper structure and README
- [ ] **6.2** Create cross-platform build configuration
- [ ] **6.3** Implement GitHub Actions for automated testing and releases
- [ ] **6.4** Build release pipeline with cross-platform binary generation
- [ ] **6.5** Create comprehensive documentation with installation instructions
- [ ] **6.6** Add example .queue.json files for common use cases
- [ ] **6.7** Set up semantic versioning and changelog automation
- [ ] **6.8** Ensure that all rules and requirements defined in spec.md, as well as the specifics outlined for feature 6, are followed for tasks 6.1 to 6.7. Fix any discrepancies