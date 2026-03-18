package grpcclient

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadCAPool_EmptyPath(t *testing.T) {
	t.Parallel()
	pool, err := loadCAPool("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pool != nil {
		t.Error("expected nil pool for empty path (use system roots)")
	}
}

func TestLoadCAPool_MissingFile(t *testing.T) {
	t.Parallel()
	pool, err := loadCAPool("/nonexistent/ca.pem")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pool != nil {
		t.Error("expected nil pool for missing file (fallback to system roots)")
	}
}

func TestLoadCAPool_ValidCert(t *testing.T) {
	t.Parallel()
	caPath := writeTempCA(t)

	pool, err := loadCAPool(caPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pool == nil {
		t.Fatal("expected non-nil pool for valid CA cert")
	}
}

func TestLoadCAPool_InvalidCert(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	caPath := filepath.Join(dir, "bad-ca.pem")
	os.WriteFile(caPath, []byte("not a certificate"), 0o600)

	_, err := loadCAPool(caPath)
	if err == nil {
		t.Fatal("expected error for invalid cert data")
	}
}

func TestTLSDialOption_SystemTLS(t *testing.T) {
	t.Parallel()
	opt, err := TLSDialOption("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opt == nil {
		t.Fatal("expected non-nil dial option")
	}
}

func TestTLSDialOption_CustomCA(t *testing.T) {
	t.Parallel()
	caPath := writeTempCA(t)

	opt, err := TLSDialOption(caPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opt == nil {
		t.Fatal("expected non-nil dial option")
	}
}

// writeTempCA generates a self-signed CA cert and returns the path to the PEM file.
func writeTempCA(t *testing.T) string {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Test CA"},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create cert: %v", err)
	}

	dir := t.TempDir()
	caPath := filepath.Join(dir, "ca.pem")
	f, err := os.Create(caPath)
	if err != nil {
		t.Fatalf("create file: %v", err)
	}
	defer f.Close()

	if err := pem.Encode(f, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}); err != nil {
		t.Fatalf("encode pem: %v", err)
	}

	return caPath
}
