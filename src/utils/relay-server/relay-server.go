package relay_server

import (
	"encoding/json"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"fmt"

	"github.com/libp2p/go-libp2p/core/host"
	libp2phttp "github.com/libp2p/go-libp2p/p2p/http"
	ma "github.com/multiformats/go-multiaddr"

	"p2p-http-relay/src/utils/config"
)

func VerifyTokenWithPocketBase(token string, pocketBaseURL string) (bool, error) {
    // Remove "Bearer " prefix if present
    cleanToken := strings.TrimPrefix(strings.TrimSpace(token), "Bearer ")

    // Create HTTP client
    client := &http.Client{}

    // Construct the PocketBase auth verification endpoint
    // Typically, PocketBase uses /api/collections/users/auth-refresh for token validation
    endpoint := fmt.Sprintf("%s/api/collections/users/auth-refresh", strings.TrimSuffix(pocketBaseURL, "/"))

    // Create request
    req, err := http.NewRequest("POST", endpoint, nil)
    if err != nil {
        return false, fmt.Errorf("failed to create request: %v", err)
    }

    // Set authorization header
    req.Header.Set("Authorization", cleanToken)

    // Make request to PocketBase
    resp, err := client.Do(req)
    if err != nil {
        return false, fmt.Errorf("failed to verify token: %v", err)
    }
    defer resp.Body.Close()

    // Check response status
    if resp.StatusCode == http.StatusOK {
        return true, nil
    }

    // Handle different status codes
    switch resp.StatusCode {
    case http.StatusUnauthorized, http.StatusForbidden:
        return false, nil // Token is invalid
    default:
        return false, fmt.Errorf("unexpected response status: %d", resp.StatusCode)
    }
}

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

			host := r.Host
			path := r.URL.Path

			// Check if the request matches a whitelisted service and path
			bypassAuth := false
			var matchedService *ProxyService
			for serviceName, proxyService := range proxyServices {
					hostPrefix := proxyService.Config.HostPrefix
					if strings.HasPrefix(host, hostPrefix+".") && strings.HasSuffix(host, ".libp2p") {
							matchedService = proxyService
							// Check if the request path matches any bypass paths
							for _, bypassPath := range proxyService.Config.BypassAuthPaths {
									if path == bypassPath {
											bypassAuth = true
											log.Printf("Bypassing auth for service: %s, path: %s", serviceName, path)
											break
									}
							}
							break
					}
			}

			// Only verify token if auth is not bypassed
			if !bypassAuth {
					proxyAuth := r.Header.Get("Proxy-Authorization")
					if proxyAuth != "" && strings.HasPrefix(strings.ToLower(proxyAuth), "bearer ") {
							log.Printf("Bearer token detected in Proxy-Authorization header for request to %s", r.Host)
							
							pocketBaseURL := "http://localhost:3003" // Replace with your PocketBase URL
							isValid, err := VerifyTokenWithPocketBase(proxyAuth, pocketBaseURL)
							if err != nil {
									log.Printf("Token verification error: %v", err)
									http.Error(w, "Token verification failed", http.StatusInternalServerError)
									return
							}
							if !isValid {
									log.Printf("Invalid token for request to %s", r.Host)
									http.Error(w, "Invalid or unauthorized token", http.StatusUnauthorized)
									return
							}
							log.Printf("Token verified successfully for request to %s", r.Host)
					} else {
							log.Printf("No valid bearer token provided for request to %s", r.Host)
							http.Error(w, "Bearer token required", http.StatusUnauthorized)
							return
					}
			}

			// Route to matched service
			if matchedService != nil {
					log.Printf("Routing request to service: %s", matchedService.Config.HostPrefix)
					matchedService.Proxy.ServeHTTP(w, r)
					return
			}

			http.NotFound(w, r)
	})
// 	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
// 		dump, err := httputil.DumpRequest(r, true)
// 		if err != nil {
// 			log.Printf("Error dumping request: %v", err)
// 		} else {
// 			log.Printf("Full HTTP request:\n%s", dump)
// 		}
//
// 		if proxyAuth := r.Header.Get("Proxy-Authorization"); proxyAuth != "" {
// 				if strings.HasPrefix(strings.ToLower(proxyAuth), "bearer ") {
// 						log.Printf("Bearer token detected in Proxy-Authorization header for request to %s", r.Host)
// 				}
// 		}
//
// 		host := r.Host // e.g., eth-rpc.<peerId>.libp2p
//
// 		// Find matching service based on host prefix
// 		for serviceName, proxyService := range proxyServices {
// 			hostPrefix := proxyService.Config.HostPrefix
// 			if strings.HasPrefix(host, hostPrefix+".") && strings.HasSuffix(host, ".libp2p") {
// 				log.Printf("Routing request to service: %s", serviceName)
// 				proxyService.Proxy.ServeHTTP(w, r)
// 				return
// 			}
// 		}
//
// 		http.NotFound(w, r)
// 	})

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
