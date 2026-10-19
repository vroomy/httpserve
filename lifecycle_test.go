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
	upgrader := NewUpgrader(443)
	if err := upgrader.Close(); err != nil {
		t.Fatal(err)
	}

	if err := upgrader.Listen(0); !errors.Is(err, http.ErrServerClosed) {
		t.Fatalf("Listen() = %v, want closed server", err)
	}
}
