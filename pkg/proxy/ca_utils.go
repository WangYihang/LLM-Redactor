package proxy

import (
	"crypto/tls"
	"crypto/x509"
	"os"
	"path/filepath"

	"github.com/elazarl/goproxy"
)

// GenerateAndSetCA generates a fresh CA and sets it for the session.
func GenerateAndSetCA(sessionDir string) (string, error) {
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		return "", err
	}

	certPEM, keyPEM, err := GenerateCA()
	if err != nil {
		return "", err
	}

	certPath := filepath.Join(sessionDir, "ca.crt")
	if err := os.WriteFile(certPath, certPEM, 0644); err != nil {
		return "", err
	}

	keyPath := filepath.Join(sessionDir, "ca.key")
	if err := os.WriteFile(keyPath, keyPEM, 0600); err != nil {
		return "", err
	}

	// Set goproxy.GoproxyCa to our newly generated CA
	ca, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return "", err
	}
	if ca.Leaf == nil {
		if leaf, err := x509.ParseCertificate(ca.Certificate[0]); err == nil {
			ca.Leaf = leaf
		}
	}
	goproxy.GoproxyCa = ca

	return certPath, nil
}
