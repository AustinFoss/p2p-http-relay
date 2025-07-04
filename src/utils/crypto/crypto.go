package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"

	"github.com/pion/webrtc/v4"
)

// Given a certain file path read a PEM encoded certificate file
// Or create a new certificate and private key and write it to the path
// Returns the certificate to be passed to webrtc transport
func LoadOrGenerateCert(certPath string) (*webrtc.Certificate, error) {
	// Check if the certificate file exists
	if _, err := os.Stat(certPath); err == nil {
		// If exists load the existing certificate
		certData, err := os.ReadFile(certPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read certificate from file: %v", err)
		}

		// Print the raw certificate data
		fmt.Println("Loaded certificate file contents:")
		fmt.Println(string(certData))

		// Decode the PEM-encoded certificate and private key
		var certBlock, keyBlock *pem.Block
		data := certData
		for {
			block, rest := pem.Decode(data)
			if block == nil {
				break
			}
			if block.Type == "CERTIFICATE" {
				certBlock = block
			} else if strings.HasSuffix(block.Type, "PRIVATE KEY") {
				keyBlock = block
			}
			data = rest
		}

		if certBlock == nil {
			return nil, fmt.Errorf("failed to find CERTIFICATE block in PEM file")
		}
		if keyBlock == nil {
			return nil, fmt.Errorf("failed to find PRIVATE KEY block in PEM file")
		}

		// Parse the CERTIFICATE block to get expiry date
		x509Cert, err := x509.ParseCertificate(certBlock.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse x509 certificate: %v", err)
		}

		// Print the certificate's expiry date
		fmt.Println("Loaded existing certificate from", certPath)
		fmt.Println("Certificate expires on:", x509Cert.NotAfter)

		// Convert the PEM back to a WebRTC certificate
		cert, err := webrtc.CertificateFromPEM(string(certData))
		if err != nil {
			return nil, fmt.Errorf("failed to parse WebRTC certificate: %v", err)
		}

		return cert, nil
	}

	// If no Certificate file was found
	// Generate a new ECDSA key for the certificate
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ECDSA key: %v", err)
	}

	// X.509 certificate template
	serialNumber, err := rand.Int(rand.Reader, big.NewInt(1<<63-1))
	if err != nil {
		return nil, fmt.Errorf("failed to generate serial number: %v", err)
	}
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   "libp2p-webrtc",
			Organization: []string{"libp2p"},
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(365 * 24 * time.Hour), // 1 year expiry
		KeyUsage:  x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
			x509.ExtKeyUsageClientAuth,
		},
		BasicConstraintsValid: true,
		IsCA:                  false,
	}

	// Generate the X.509 certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return nil, fmt.Errorf("failed to create x509 certificate: %v", err)
	}

	// Encode certificate to PEM
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})
	if certPEM == nil {
		return nil, fmt.Errorf("failed to encode certificate to PEM")
	}

	// Encode private key to PEM
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal private key: %v", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: keyDER,
	})
	if keyPEM == nil {
		return nil, fmt.Errorf("failed to encode private key to PEM")
	}

	// Print the generated certificate and key
	fmt.Println("Generated certificate PEM:")
	fmt.Println(string(certPEM))
	fmt.Println("Generated private key PEM:")
	fmt.Println(string(keyPEM))

	// Parse the PEM to get the x509 certificate for expiry date
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, fmt.Errorf("failed to decode generated PEM certificate")
	}
	x509Cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse generated x509 certificate: %v", err)
	}

	// Convert to WebRTC certificate
	cert := webrtc.CertificateFromX509(key, x509Cert)

	// Combine certificate and private key PEMs and save to file
	// TODO Consider saving these as seperate files
	pemData := append(certPEM, keyPEM...)
	if err := os.MkdirAll(filepath.Dir(certPath), 0700); err != nil {
		return nil, fmt.Errorf("failed to create directory")
	}
	if err := os.WriteFile(certPath, pemData, 0600); err != nil {
		return nil, fmt.Errorf("failed to save certificate and key to file: %v", err)
	}

	// Print the certificate's expiry date
	fmt.Println("Generated and saved new certificate to", certPath)
	fmt.Println("Certificate expires on:", x509Cert.NotAfter)

	return &cert, nil
}

// Given a certain file path read a LibP2P private key file
// Or create a new private key and write it to the path
// Returns the private key to be passed to libp2p library
func LoadOrGenerateLibP2pKey(keyFilePath string) (crypto.PrivKey, error) {
	// Check if the key file exists
	if _, err := os.Stat(keyFilePath); os.IsNotExist(err) {
		// Generate a new private key if it doesn’t exist
		privKey, _, err := crypto.GenerateKeyPair(crypto.Ed25519, -1)
		if err != nil {
			return nil, fmt.Errorf("failed to generate private key: %v", err)
		}

		// Serialize the private key to bytes
		keyBytes, err := crypto.MarshalPrivateKey(privKey)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal private key: %v", err)
		}

		// Save the key to file
		if err := os.MkdirAll(filepath.Dir(keyFilePath), 0700); err != nil {
			return nil, fmt.Errorf("failed to create directory")
		}
		err = os.WriteFile(keyFilePath, keyBytes, 0600)
		if err != nil {
			return nil, fmt.Errorf("failed to save private key to file: %v", err)
		}

		fmt.Println("Generated and saved new private key to", keyFilePath)
		return privKey, nil
	}

	// Load the existing key from file
	keyBytes, err := os.ReadFile(keyFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key from file: %v", err)
	}

	// Unmarshal the key bytes into a private key
	privKey, err := crypto.UnmarshalPrivateKey(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal private key: %v", err)
	}

	fmt.Println("Loaded existing private key from", keyFilePath)
	return privKey, nil
}
