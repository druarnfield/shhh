package platform

import "crypto/x509"

// CertStore provides access to the operating system's trusted root certificate store.
type CertStore interface {
	// SystemRoots returns all trusted root certificates from the OS cert store.
	SystemRoots() ([]*x509.Certificate, error)
}
