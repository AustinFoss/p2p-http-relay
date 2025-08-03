package relay_server

import (
	"encoding/json"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/libp2p/go-libp2p/core/host"
	libp2phttp "github.com/libp2p/go-libp2p/p2p/http"
	ma "github.com/multiformats/go-multiaddr"

	"p2p-http-relay/src/utils/config"
)

// ProxyService represents a configured proxy service
type ProxyService struct {
	Config config.ServiceConfig
	Proxy  *httputil.ReverseProxy
}

// NewP2pHttpMux creates a new HTTP mux with services loaded from TOML config
func NewP2pHttpMux(peerID string, configPath string) *http.ServeMux {
	// Load configuration
	servicesConfig, err := config.LoadServicesConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load services config: %v", err)
	}

	mux := http.NewServeMux()

	// Create proxy services from configuration
	proxyServices := make(map[string]*ProxyService)
	enabledServices := servicesConfig.GetEnabledServices()

	for serviceName, serviceConfig := range enabledServices {
		proxy, err := createProxyFromConfig(serviceConfig)
		if err != nil {
			log.Printf("Failed to create proxy for service %s: %v", serviceName, err)
			continue
		}

		proxyServices[serviceName] = &ProxyService{
			Config: serviceConfig,
			Proxy:  proxy,
		}

		log.Printf("Loaded service: %s -> %s://%s%s",
			serviceConfig.HostPrefix,
			serviceConfig.Scheme,
			serviceConfig.Host,
			serviceConfig.Path)
	}

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		dump, err := httputil.DumpRequest(r, true)
		if err != nil {
			log.Printf("Error dumping request: %v", err)
		} else {
			log.Printf("Full HTTP request:\n%s", dump)
		}

		host := r.Host // e.g., eth-rpc.<peerId>.libp2p

		// Find matching service based on host prefix
		for serviceName, proxyService := range proxyServices {
			hostPrefix := proxyService.Config.HostPrefix
			if strings.HasPrefix(host, hostPrefix+".") && strings.HasSuffix(host, ".libp2p") {
				log.Printf("Routing request to service: %s", serviceName)
				proxyService.Proxy.ServeHTTP(w, r)
				return
			}
		}

		http.NotFound(w, r)
	})

	// Well-known endpoint for service discovery
	mux.HandleFunc("/.well-known/libp2p/protocols", func(w http.ResponseWriter, r *http.Request) {
		hosts := make(map[string]string)
		for serviceName, proxyService := range proxyServices {
			hostKey := proxyService.Config.HostPrefix + "." + peerID + ".libp2p"
			hosts[hostKey] = serviceName + " proxy"
		}

		mapping := map[string]interface{}{
			"hosts": hosts,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mapping)
	})

	return mux
}

// createProxyFromConfig creates a reverse proxy from a service configuration
func createProxyFromConfig(serviceConfig config.ServiceConfig) (*httputil.ReverseProxy, error) {
	// Create the target URL
	targetURL := &url.URL{
		Scheme: serviceConfig.Scheme,
		Host:   serviceConfig.Host,
		Path:   serviceConfig.Path,
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// Set up the director function
	proxy.Director = func(req *http.Request) {
		req.URL.Scheme = targetURL.Scheme
		req.URL.Host = targetURL.Host

		// Only set path if it's not empty
		if targetURL.Path != "" {
			req.URL.Path = targetURL.Path
		}

		// Filter headers based on allowed headers
		allowedHeaders := make(map[string]bool)
		for _, header := range serviceConfig.AllowedHeaders {
			allowedHeaders[header] = true
		}

		// Remove all headers except allowed ones
		for k := range req.Header {
			if !allowedHeaders[k] {
				req.Header.Del(k)
			}
		}

		// Set required headers
		req.Header.Set("User-Agent", serviceConfig.UserAgent)
		req.Header.Set("Accept", "*/*")
		req.Host = targetURL.Host
	}

	return proxy, nil
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
			log.Printf("Server error: %v", err)
		}
	}()
}
