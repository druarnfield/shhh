//go:build darwin

package platform

import (
	"crypto/sha256"
	"testing"
)

func TestCertStore_SystemRoots(t *testing.T) {
	store := NewCertStore()
	certs, err := store.SystemRoots()
	if err != nil {
		t.Fatalf("SystemRoots() error: %v", err)
	}

	if len(certs) == 0 {
		t.Fatal("SystemRoots() returned 0 certificates")
	}
	t.Logf("SystemRoots() returned %d certificates", len(certs))

	// Check each cert is a CA or self-signed.
	for i, cert := range certs {
		if !cert.IsCA && cert.Issuer.String() != cert.Subject.String() {
			t.Errorf("cert[%d] %q is neither CA nor self-signed", i, cert.Subject)
		}
	}

	// Check no duplicate fingerprints.
	seen := make(map[[sha256.Size]byte]struct{})
	for _, cert := range certs {
		fp := sha256.Sum256(cert.Raw)
		if _, dup := seen[fp]; dup {
			t.Errorf("duplicate certificate: %q", cert.Subject)
		}
		seen[fp] = struct{}{}
	}
}
