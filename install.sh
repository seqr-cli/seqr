#!/bin/bash

# seqr installation script
# Usage: curl -sSL https://raw.githubusercontent.com/seqr-cli/seqr/main/install.sh | bash

set -e

# Configuration
REPO="seqr-cli/seqr"
BINARY_NAME="seqr"
INSTALL_DIR="/usr/local/bin"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Detect OS and architecture
detect_platform() {
    local os arch
    
    case "$(uname -s)" in
        Darwin*)
            os="darwin"
            ;;
        Linux*)
            os="linux"
            ;;
        CYGWIN*|MINGW*|MSYS*)
            os="windows"
            ;;
        *)
            log_error "Unsupported operating system: $(uname -s)"
            exit 1
            ;;
    esac
    
    case "$(uname -m)" in
        x86_64|amd64)
            arch="amd64"
            ;;
        arm64|aarch64)
            arch="arm64"
            ;;
        *)
            log_error "Unsupported architecture: $(uname -m)"
            exit 1
            ;;
    esac
    
    echo "${os}-${arch}"
}

# Get the latest release version
get_latest_version() {
    local version
    version=$(curl -s "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    
    if [ -z "$version" ]; then
        log_error "Failed to get latest version from GitHub API"
        exit 1
    fi
    
    echo "$version"
}

# Download and install seqr
install_seqr() {
    local platform version binary_name download_url temp_dir
    
    platform=$(detect_platform)
    version=$(get_latest_version)
    
    log_info "Detected platform: $platform"
    log_info "Latest version: $version"
    
    # Determine binary name and download URL
    if [[ "$platform" == *"windows"* ]]; then
        binary_name="${BINARY_NAME}-${platform}.exe"
        download_url="https://github.com/${REPO}/releases/download/${version}/${binary_name}.zip"
    else
        binary_name="${BINARY_NAME}-${platform}"
        download_url="https://github.com/${REPO}/releases/download/${version}/${binary_name}.tar.gz"
    fi
    
    log_info "Downloading $binary_name..."
    
    # Create temporary directory
    temp_dir=$(mktemp -d)
    cd "$temp_dir"
    
    # Download the binary
    if ! curl -sL "$download_url" -o "archive"; then
        log_error "Failed to download $download_url"
        exit 1
    fi
    
    # Extract the binary
    if [[ "$platform" == *"windows"* ]]; then
        unzip -q archive
        binary_path="$binary_name"
    else
        tar -xzf archive
        binary_path="$binary_name"
    fi
    
    # Check if binary exists
    if [ ! -f "$binary_path" ]; then
        log_error "Binary not found in archive"
        exit 1
    fi
    
    # Make binary executable
    chmod +x "$binary_path"
    
    # Install binary
    log_info "Installing to $INSTALL_DIR/$BINARY_NAME..."
    
    if [ -w "$INSTALL_DIR" ]; then
        mv "$binary_path" "$INSTALL_DIR/$BINARY_NAME"
    else
        log_info "Requesting sudo access to install to $INSTALL_DIR..."
        sudo mv "$binary_path" "$INSTALL_DIR/$BINARY_NAME"
    fi
    
    # Cleanup
    cd - > /dev/null
    rm -rf "$temp_dir"
    
    log_success "seqr installed successfully!"
    log_info "Run 'seqr --help' to get started"
    
    # Verify installation
    if command -v seqr >/dev/null 2>&1; then
        log_success "Installation verified: $(seqr --version)"
    else
        log_warning "seqr installed but not found in PATH. You may need to restart your shell or add $INSTALL_DIR to your PATH."
    fi
}

# Check dependencies
check_dependencies() {
    local missing_deps=()
    
    if ! command -v curl >/dev/null 2>&1; then
        missing_deps+=("curl")
    fi
    
    if ! command -v tar >/dev/null 2>&1; then
        missing_deps+=("tar")
    fi
    
    if [[ "${#missing_deps[@]}" -gt 0 ]]; then
        log_error "Missing required dependencies: ${missing_deps[*]}"
        log_error "Please install the missing dependencies and try again"
        exit 1
    fi
}

# Main installation function
main() {
    log_info "Installing seqr - Sequential Command Queue Runner"
    echo
    
    check_dependencies
    install_seqr
    
    echo
    log_success "Installation complete!"
    echo
    echo "Quick start:"
    echo "  1. Create a .queue.json file in your project"
    echo "  2. Run 'seqr' to execute your command queue"
    echo "  3. Use 'seqr --help' for more options"
    echo
    echo "Example .queue.json:"
    echo '  {'
    echo '    "version": "1.0",'
    echo '    "commands": ['
    echo '      {'
    echo '        "name": "hello",'
    echo '        "command": "echo",'
    echo '        "args": ["Hello, World!"],'
    echo '        "mode": "once"'
    echo '      }'
    echo '    ]'
    echo '  }'
}

# Run main function
main "$@"