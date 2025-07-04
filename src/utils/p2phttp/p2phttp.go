package p2phttp

import (
	// "context"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

	libp2phttp "github.com/libp2p/go-libp2p/p2p/http"

	"github.com/libp2p/go-libp2p/core/host"

	ma "github.com/multiformats/go-multiaddr"

	"github.com/joho/godotenv"
)

func NewP2pHttpMux(peerID string) *http.ServeMux {

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	mux := http.NewServeMux()

	// Setup PocketBase proxy
	// pbProxy := httputil.NewSingleHostReverseProxy(&url.URL{
	// 	Scheme: "http",
	// 	Host:   "localhost:8090", // Adjust as appropriate
	// })

	// // Setup DoH proxy
	// dohProxy := httputil.NewSingleHostReverseProxy(&url.URL{
	// 	Scheme: "https",
	// 	Host:   "cloudflare-dns.com",
	// })

	// Setup Ethereum RPC proxy to Alchemy
	alchemyApi := os.Getenv("ALCHEMY_API")
	ethRpcURL := &url.URL{
		Scheme: "https",
		Host:   "eth-mainnet.g.alchemy.com",
		Path:   "/v2/" + alchemyApi,
	}

	ethConsensusURL := &url.URL{
		Scheme: "https",
		Host:   "ethereum.operationsolarstorm.org",
	}

	// TODO: Find a way to combine and reduce repetition for consensusProxy & ethRpcProxy

	// Only allow specific headers
	// Remote Service endpoints have been picky when other headers that browser clients include and reject requests
	consensusProxy := httputil.NewSingleHostReverseProxy(ethConsensusURL)

	consensusProxy.Director = func(req *http.Request) {
		req.URL.Scheme = ethConsensusURL.Scheme
		req.URL.Host = ethConsensusURL.Host

		allowed := map[string]bool{
			"Content-Type":   true,
			"Content-Length": true,
			"Accept":         true,
			"User-Agent":     true,
		}

		for k := range req.Header {
			if !allowed[k] {
				req.Header.Del(k)
			}
		}

		req.Header.Set("User-Agent", "MyLibp2pProxy/1.0")
		req.Header.Set("Accept", "*/*")
		req.Host = ethConsensusURL.Host
	}

	ethRpcProxy := httputil.NewSingleHostReverseProxy(ethRpcURL)

	ethRpcProxy.Director = func(req *http.Request) {
		req.URL.Scheme = ethRpcURL.Scheme
		req.URL.Host = ethRpcURL.Host
		req.URL.Path = ethRpcURL.Path

		allowed := map[string]bool{
			"Content-Type":   true,
			"Content-Length": true,
			"Accept":         true,
			"User-Agent":     true,
		}

		// Remove all headers except allowed ones
		for k := range req.Header {
			if !allowed[k] {
				req.Header.Del(k)
			}
		}

		req.Header.Set("User-Agent", "MyLibp2pProxy/1.0")
		req.Header.Set("Accept", "*/*")
		req.Host = ethRpcURL.Host
	}

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		dump, err := httputil.DumpRequest(r, true) // true to include body
		if err != nil {
			log.Printf("Error dumping request: %v", err)
		} else {
			log.Printf("Full HTTP request:\n%s", dump)
		}

		host := r.Host // e.g., pb.<peerId>.libp2p

		switch {
		// case strings.HasPrefix(host, "pb.") && strings.HasSuffix(host, ".libp2p"):
		// 	pbProxy.ServeHTTP(w, r)
		// 	return
		// case strings.HasPrefix(host, "doh.") && strings.HasSuffix(host, ".libp2p"):
		// 	dohProxy.ServeHTTP(w, r)
		// 	return
		case strings.HasPrefix(host, "eth-rpc.") && strings.HasSuffix(host, ".libp2p"):
			ethRpcProxy.ServeHTTP(w, r)
			return
		case strings.HasPrefix(host, "eth-consensus.") && strings.HasSuffix(host, ".libp2p"):
			log.Println("Got Consensus request")
			consensusProxy.ServeHTTP(w, r)
			return
		default:
			http.NotFound(w, r)
		}
	})

	// Well-known endpoint for service discovery
	// TODO: Reconsider moving this as it might conflict with the existing protocol spec
	mux.HandleFunc("/.well-known/libp2p/protocols", func(w http.ResponseWriter, r *http.Request) {
		mapping := map[string]interface{}{
			"hosts": map[string]string{
				"pb." + peerID + ".libp2p":      "PocketBase API proxy",
				"doh." + peerID + ".libp2p":     "DNS over HTTPS proxy",
				"eth-rpc." + peerID + ".libp2p": "Ethereum JSON-RPC proxy (Alchemy)",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mapping)
	})

	return mux
}

func InitP2pHttp(libp2pHost host.Host, mux *http.ServeMux) {

	listenAddr := ma.StringCast("/ip4/127.0.0.1/tcp/0/http")

	server := libp2phttp.Host{
		InsecureAllowHTTP: true, // For local proxy, allow HTTP
		ListenAddrs:       []ma.Multiaddr{listenAddr},
		StreamHost:        libp2pHost,
	}

	server.SetHTTPHandler("/", mux)

	go func() {
		if err := server.Serve(); err != nil {
		}
	}()
}
