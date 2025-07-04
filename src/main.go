package main

import (
	// "flag"
	"fmt"
	"log"

	"p2phttp-server/src/utils/crypto"
	"p2phttp-server/src/utils/libp2p"
	"p2phttp-server/src/utils/p2phttp"
)

const keyFilePath = "tmp/libp2p_private_key"
const certPath = "tmp/webrtc_cert_key.pem"
const addrFilePath = "tmp/host_addresses.txt"

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
	mux := p2phttp.NewP2pHttpMux(libp2p.ID().String())
	p2phttp.InitP2pHttp(libp2p, mux)

	// Keep the main thread alive
	fmt.Println("Server running. Press Ctrl+C to stop.")
	select {} // Blocks indefinitely
}
