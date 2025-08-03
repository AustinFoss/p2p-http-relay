package main

import (
	// "flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"p2p-http-relay/src/utils/crypto"
	"p2p-http-relay/src/utils/libp2p"
	relay_server "p2p-http-relay/src/utils/relay-server"
)

const keyFilePath = "tmp/libp2p_private_key"
const certPath = "tmp/webrtc_cert_key.pem"
const addrFilePath = "tmp/host_addresses.txt"

// getConfigPath returns the path to the configuration file
func getConfigPath() string {
	// First, try to find config.toml in current directory (for development)
	if _, err := os.Stat("config.toml"); err == nil {
		return "config.toml"
	}

	// Then try the user's config directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Printf("Warning: Could not get home directory, using current directory")
		return "config.toml"
	}

	configPath := filepath.Join(homeDir, ".p2p-http-relay", "config.toml")
	if _, err := os.Stat(configPath); err == nil {
		return configPath
	}

	// Fallback to current directory
	return "config.toml"
}

func main() {

	// Define a command-line flag for the remote node's multiaddr
	// flag.Parse()

	// Load or generate the LibP2P private key to provide a constant PeerID across multiple restarts
	privKey, err := crypto.LoadOrGenerateLibP2pKey(keyFilePath)
	if err != nil {
		log.Fatal("LoadOrGenerateLibP2pKey(): ", err)
	}

	// Load or generate the WebRTC Certifiacte to provide a constant WebRTC-Direct MultiAddr across multiple restarts
	cert, err := crypto.LoadOrGenerateCert(certPath)
	if err != nil {
		log.Fatal("LoadOrGenerateCert(): ", err)
	}

	// Initialize the LibP2P Node
	libp2p, err := libp2p.InitLibP2p(privKey, cert, addrFilePath)
	if err != nil {
		log.Fatal("Failed to InitLibP2p(): ", err)
	}

	// Initialize the P2PHTTP listener service
	configPath := getConfigPath()
	mux := relay_server.NewP2pHttpMux(libp2p.ID().String(), configPath)
	relay_server.InitP2pHttp(libp2p, mux)

	// Keep the main thread alive
	fmt.Println("Server running. Press Ctrl+C to stop.")
	select {} // Blocks indefinitely
}
