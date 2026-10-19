package apex

import (
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestCloseRacingWithListen(t *testing.T) {
	for i := 0; i < 50; i++ {
		server := New()
		result := make(chan error, 1)
		go func() { result <- server.Listen(0) }()
		if err := server.Close(); err != nil {
			t.Fatal(err)
		}

		select {
		case err := <-result:
			if !errors.Is(err, http.ErrServerClosed) {
				t.Fatalf("Listen() = %v, want closed server", err)
			}
		case <-time.After(time.Second):
			t.Fatal("Close allowed a late listener to remain open")
		}
	}
}

func TestUpgraderCloseBeforeListen(t *testing.T) {
	var (
		u   *Upgrader
		err error
	)

	if u, err = NewUpgrader(443); err != nil {
		t.Fatal(err)
	}

	if err = u.Close(); err != nil {
		t.Fatal(err)
	}

	if err := u.Listen(0); !errors.Is(err, http.ErrServerClosed) {
		t.Fatalf("Listen() = %v, want closed server", err)
	}
}
