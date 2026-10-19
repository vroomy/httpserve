package apex

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTLSWalkErrorsAndShortNames(t *testing.T) {
	certificates := make(tlsCerts)
	if err := certificates.walk("missing", nil, os.ErrNotExist); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("walk() = %v, want filesystem cause", err)
	}

	shortFile := filepath.Join(t.TempDir(), "x")
	if err := os.WriteFile(shortFile, nil, 0600); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(shortFile)
	if err != nil {
		t.Fatal(err)
	}

	if err = certificates.walk("x", info, nil); err != nil {
		t.Fatal(err)
	}

	if len(certificates) != 0 {
		t.Fatalf("non-certificate file retained: %v", certificates)
	}
}

func writeTestCertificate(t *testing.T, directory, name string) {
	t.Helper()
	var public, private, err = ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	var template = &x509.Certificate{
		SerialNumber: big.NewInt(1),
		DNSNames:     []string{"localhost"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	var certificate []byte
	if certificate, err = x509.CreateCertificate(rand.Reader, template, template, public, private); err != nil {
		t.Fatal(err)
	}

	var key []byte
	if key, err = x509.MarshalPKCS8PrivateKey(private); err != nil {
		t.Fatal(err)
	}

	if err = os.WriteFile(filepath.Join(directory, name+".crt"), pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certificate}), 0600); err != nil {
		t.Fatal(err)
	}

	if err = os.WriteFile(filepath.Join(directory, name+".key"), pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: key}), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestTLSCertificateDiscovery(t *testing.T) {
	var directory = t.TempDir()
	var nested = filepath.Join(directory, "nested")
	if err := os.Mkdir(nested, 0700); err != nil {
		t.Fatal(err)
	}

	writeTestCertificate(t, directory, "first")
	writeTestCertificate(t, nested, "second")
	if err := os.WriteFile(filepath.Join(directory, "ignored.txt"), nil, 0600); err != nil {
		t.Fatal(err)
	}

	var discovered, err = newTLSCerts(directory)
	if err != nil || len(discovered) != 2 {
		t.Fatalf("discovered certificates = %v/%v", discovered, err)
	}

	var certificates = discovered[filepath.Join(directory, "first")]
	if certificates == nil || !certificates.isValid() {
		t.Fatalf("certificate pair = %#v", certificates)
	}

	var loaded, loadErr = discovered.Certificates()
	if loadErr != nil || len(loaded) != 2 {
		t.Fatalf("Certificates() = %v/%v", loaded, loadErr)
	}

	for _, certificate := range loaded {
		if len(certificate.Certificate) != 1 || certificate.PrivateKey == nil {
			t.Fatal("loaded certificate missing key material")
		}
	}
}

func TestTLSCertificateFailures(t *testing.T) {
	var missing = filepath.Join(t.TempDir(), "missing")
	if _, err := newTLSCerts(missing); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing directory error = %v", err)
	}

	var cases = []struct {
		name string
		pair *tlsCert
	}{
		{"empty pair", &tlsCert{}},
		{"missing key", &tlsCert{cert: "missing.crt"}},
		{"missing certificate", &tlsCert{key: "missing.key"}},
		{"unreadable pair", &tlsCert{key: missing + ".key", cert: missing + ".crt"}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			var _, err = (tlsCerts{"test": test.pair}).Certificates()
			if err == nil {
				t.Fatal("invalid certificate pair was accepted")
			}
		})
	}

	var directory = t.TempDir()
	for _, name := range []string{"invalid.crt", "invalid.key"} {
		if err := os.WriteFile(filepath.Join(directory, name), []byte("invalid PEM"), 0600); err != nil {
			t.Fatal(err)
		}
	}

	var discovered, err = newTLSCerts(directory)
	if err != nil {
		t.Fatal(err)
	}

	if _, err = discovered.Certificates(); err == nil {
		t.Fatal("invalid PEM was accepted")
	}

	var empty, emptyErr = (tlsCerts{}).Certificates()
	if emptyErr != nil || len(empty) != 0 {
		t.Fatalf("empty certificate set = %v/%v", empty, emptyErr)
	}
}
