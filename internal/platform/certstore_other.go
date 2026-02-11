//go:build !windows && !darwin

package platform

import "crypto/x509"

type stubCertStore struct{}

// NewCertStore returns a stub CertStore that returns ErrNotSupported on unsupported platforms.
func NewCertStore() CertStore { return &stubCertStore{} }

func (s *stubCertStore) SystemRoots() ([]*x509.Certificate, error) {
	return nil, ErrNotSupported
}
