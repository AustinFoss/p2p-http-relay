package libp2p

import (
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/transport"

	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
	libp2pwebrtc "github.com/libp2p/go-libp2p/p2p/transport/webrtc"
	webtransport "github.com/libp2p/go-libp2p/p2p/transport/webtransport"

	"github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/relay"

	"github.com/pion/webrtc/v4"
)

func InitLibP2p(privKey crypto.PrivKey, webRTCCert *webrtc.Certificate, addrFilePath string) (host.Host, error) {

	listenUDP := func(network string, laddr *net.UDPAddr) (net.PacketConn, error) {
		return net.ListenUDP(network, laddr)
	}

	tptWebRTC, err := libp2pwebrtc.New(privKey, nil, nil, nil, listenUDP, libp2pwebrtc.WithCustomCert(webRTCCert))
	if err != nil {
		log.Fatal("Err creating WebRTC transport: ", err)
	}

	libp2p, err := libp2p.New(
		libp2p.Identity(privKey),
		libp2p.ListenAddrStrings(
			"/ip4/0.0.0.0/tcp/37285",
			"/ip4/0.0.0.0/udp/37385/quic-v1/webtransport",
			"/ip4/0.0.0.0/udp/37485/webrtc-direct",
		),
		libp2p.Transport(tcp.NewTCPTransport),
		libp2p.Transport(webtransport.New),
		libp2p.Transport(func() transport.Transport { return tptWebRTC }),
		libp2p.EnableRelay(),
	)

	if err != nil {
		fmt.Println("Err creating LibP2P: ", err)
	}
	resources := relay.DefaultResources()
	resources.MaxReservations = 256
	_, err = relay.New(libp2p, relay.WithResources(resources))
	if err != nil {
		panic(err)
	}

	libp2p.Network().Notify(&network.NotifyBundle{
		ConnectedF: func(n network.Network, conn network.Conn) {
			log.Printf("Peer connected: PeerID=%s, Multiaddr=%s", conn.RemotePeer(), conn.RemoteMultiaddr())
		},
	})

	fmt.Println("Host ID:", libp2p.ID())

	addrStrings := make([]string, len(libp2p.Addrs()))
	for i, addr := range libp2p.Addrs() {
		addrStrings[i] = addr.String()
	}
	fmt.Println("Host Addresses:", addrStrings)
	addrStrings = append(addrStrings, libp2p.ID().String())

	if err := writeAddressesToFile(addrStrings, addrFilePath); err != nil {
		panic(err)
	}

	return libp2p, err

}

func writeAddressesToFile(addrs []string, addrFilePath string) error {
	content := strings.Join(addrs, "\n")

	if err := os.MkdirAll(filepath.Dir(addrFilePath), 0700); err != nil {
		return fmt.Errorf("failed to create directory: ")
	}
	err := os.WriteFile(addrFilePath, []byte(content), 0644)
	if err != nil {
		return fmt.Errorf("failed to write host addresses to file: %v", err)
	}
	fmt.Println("Host addresses written to", addrFilePath)
	return nil
}
