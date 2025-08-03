package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/joho/godotenv"
)

// ServiceConfig represents a single HTTP service configuration
type ServiceConfig struct {
	Enabled        bool              `toml:"enabled"`
	HostPrefix     string            `toml:"host_prefix"`
	Scheme         string            `toml:"scheme"`
	Host           string            `toml:"host"`
	Path           string            `toml:"path"`
	UserAgent      string            `toml:"user_agent"`
	AllowedHeaders []string          `toml:"allowed_headers"`
	EnvVars        map[string]string `toml:"env_vars,omitempty"`
}

// ServicesConfig represents the entire services configuration
type ServicesConfig struct {
	Services map[string]ServiceConfig `toml:"services"`
}

// LoadServicesConfig loads the services configuration from a TOML file
func LoadServicesConfig(configPath string) (*ServicesConfig, error) {
	// Load .env file if it exists (optional)
	loadEnvFile() // Don't fail if .env doesn't exist

	config := &ServicesConfig{}

	// Read the TOML file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse the TOML data
	if err := toml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse TOML: %w", err)
	}

	// Process environment variable substitutions
	if err := config.processEnvironmentVars(); err != nil {
		return nil, fmt.Errorf("failed to process environment variables: %w", err)
	}

	return config, nil
}

// loadEnvFile loads environment variables from .env file if it exists
func loadEnvFile() {
	// Try to load .env file from current directory first
	if err := godotenv.Load(); err == nil {
		log.Println("Loaded environment variables from .env file")
		return
	}

	// Try to load from user's config directory
	homeDir, err := os.UserHomeDir()
	if err == nil {
		envPath := filepath.Join(homeDir, ".p2p-http-relay", ".env")
		if err := godotenv.Load(envPath); err == nil {
			log.Printf("Loaded environment variables from %s", envPath)
			return
		}
	}

	// If no .env file found, that's okay - just log it
	log.Println("No .env file found, using system environment variables only")
}

// processEnvironmentVars replaces environment variable placeholders in paths
func (sc *ServicesConfig) processEnvironmentVars() error {
	for serviceName, service := range sc.Services {
		if len(service.EnvVars) > 0 {
			// Process path for environment variables
			processedPath := service.Path
			hasUnsetVars := false

			for placeholder, envVar := range service.EnvVars {
				envValue := os.Getenv(envVar)
				if envValue == "" {
					log.Printf("Warning: environment variable %s is not set for service %s, disabling this service", envVar, serviceName)
					hasUnsetVars = true
					break
				}
				placeholderStr := "{" + placeholder + "}"
				processedPath = strings.ReplaceAll(processedPath, placeholderStr, envValue)
			}

			if hasUnsetVars {
				// Disable the service if environment variables are missing
				service.Enabled = false
				sc.Services[serviceName] = service
				continue
			}

			// Update the service with processed path
			service.Path = processedPath
			sc.Services[serviceName] = service
		}
	}
	return nil
}

// GetEnabledServices returns only the enabled services
func (sc *ServicesConfig) GetEnabledServices() map[string]ServiceConfig {
	enabled := make(map[string]ServiceConfig)
	for name, service := range sc.Services {
		if service.Enabled {
			enabled[name] = service
		}
	}
	return enabled
}
