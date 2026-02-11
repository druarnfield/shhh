//go:build darwin

package platform

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os/exec"
)

type darwinCertStore struct{}

// NewCertStore returns a CertStore that reads from macOS system keychains.
func NewCertStore() CertStore { return &darwinCertStore{} }

func (d *darwinCertStore) SystemRoots() ([]*x509.Certificate, error) {
	keychains := []string{
		"/System/Library/Keychains/SystemRootCertificates.keychain",
		"/Library/Keychains/System.keychain",
	}

	seen := make(map[[sha256.Size]byte]struct{})
	var certs []*x509.Certificate

	for _, kc := range keychains {
		out, err := exec.Command("security", "find-certificate", "-a", "-p", kc).Output()
		if err != nil {
			// Keychain may not exist; skip gracefully.
			continue
		}

		rest := out
		for {
			var block *pem.Block
			block, rest = pem.Decode(rest)
			if block == nil {
				break
			}
			if block.Type != "CERTIFICATE" {
				continue
			}

			cert, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				continue
			}

			fp := sha256.Sum256(cert.Raw)
			if _, dup := seen[fp]; dup {
				continue
			}
			seen[fp] = struct{}{}
			certs = append(certs, cert)
		}
	}

	if len(certs) == 0 {
		return nil, fmt.Errorf("no certificates found in system keychains")
	}

	return certs, nil
}
