#!/bin/bash

# P2P HTTP Relay Installer
# This script installs the p2p-http-relay binary and sets up configuration

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
BINARY_NAME="p2p-http-relay"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="$HOME/.p2p-http-relay"
EXAMPLE_CONFIG="config.example.toml"

echo -e "${GREEN}P2P HTTP Relay Installer${NC}"
echo "================================"

# Check if running as root for system-wide install
if [[ $EUID -eq 0 ]]; then
    echo -e "${YELLOW}Warning: Running as root. This will install system-wide.${NC}"
    read -p "Continue? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Installation cancelled."
        exit 1
    fi
fi

# Check if binary exists
if [[ ! -f "dist/$BINARY_NAME" ]]; then
    echo -e "${RED}Error: Binary not found at dist/$BINARY_NAME${NC}"
    echo "Please run 'go build -o dist/$BINARY_NAME src/main.go' first"
    exit 1
fi

# Create config directory
echo -e "${GREEN}Creating configuration directory...${NC}"
mkdir -p "$CONFIG_DIR"

# Copy binary to install directory
echo -e "${GREEN}Installing binary to $INSTALL_DIR...${NC}"
if [[ $EUID -eq 0 ]]; then
    cp "dist/$BINARY_NAME" "$INSTALL_DIR/"
    chmod +x "$INSTALL_DIR/$BINARY_NAME"
else
    sudo cp "dist/$BINARY_NAME" "$INSTALL_DIR/"
    sudo chmod +x "$INSTALL_DIR/$BINARY_NAME"
fi

# Copy example config if it exists
if [[ -f "$EXAMPLE_CONFIG" ]]; then
    echo -e "${GREEN}Copying example configuration...${NC}"
    cp "$EXAMPLE_CONFIG" "$CONFIG_DIR/config.toml"
    echo -e "${YELLOW}Please edit $CONFIG_DIR/config.toml with your settings${NC}"
else
    echo -e "${YELLOW}No example config found. Please create $CONFIG_DIR/config.toml manually${NC}"
fi

# Create .env example
echo -e "${GREEN}Creating .env example file...${NC}"
cat > "$CONFIG_DIR/.env.example" << 'EOF'
# Environment variables for P2P HTTP Relay
# Copy this file to .env and replace with your actual values

EOF

echo -e "${GREEN}Installation completed successfully!${NC}"
echo ""
echo -e "${GREEN}Next steps:${NC}"
echo "1. Edit configuration: $CONFIG_DIR/config.toml"
echo "2. Create .env file: cp $CONFIG_DIR/.env.example $CONFIG_DIR/.env"
echo "3. Edit .env file with your API keys"
echo "4. Run the service: $BINARY_NAME"
echo ""
echo -e "${GREEN}Configuration directory: $CONFIG_DIR${NC}"
echo -e "${GREEN}Binary location: $INSTALL_DIR/$BINARY_NAME${NC}" 