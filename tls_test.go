package apex

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
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
