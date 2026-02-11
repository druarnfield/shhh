//go:build windows

package platform

import (
	"crypto/sha256"
	"crypto/x509"
	"fmt"
	"syscall"
	"unsafe"
)

type windowsCertStore struct{}

// NewCertStore returns a CertStore that reads from Windows certificate stores.
func NewCertStore() CertStore { return &windowsCertStore{} }

func (w *windowsCertStore) SystemRoots() ([]*x509.Certificate, error) {
	storeNames := []string{"ROOT", "CA"}

	seen := make(map[[sha256.Size]byte]struct{})
	var certs []*x509.Certificate

	for _, name := range storeNames {
		store, err := syscall.CertOpenSystemStore(0, syscall.StringToUTF16Ptr(name))
		if err != nil {
			continue
		}
		defer syscall.CertCloseStore(store, 0)

		var ctx *syscall.CertContext
		for {
			ctx, err = syscall.CertEnumCertificatesInStore(store, ctx)
			if err != nil {
				break
			}

			// Copy the DER bytes from the cert context.
			der := unsafe.Slice(ctx.EncodedCert, ctx.Length)
			buf := make([]byte, len(der))
			copy(buf, der)

			cert, err := x509.ParseCertificate(buf)
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
		return nil, fmt.Errorf("no certificates found in Windows certificate stores")
	}

	return certs, nil
}
