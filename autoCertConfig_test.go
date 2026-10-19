package apex

import (
	"context"
	"errors"
	"testing"
)

func TestAutoCertHostPolicy(t *testing.T) {
	var config = AutoCertConfig{Hosts: []string{"example.com"}}
	var policy = config.hostPolicy()
	if err := policy(context.Background(), "example.com"); err != nil {
		t.Fatalf("allowed host rejected: %v", err)
	}

	if err := policy(context.Background(), "other.example"); err == nil {
		t.Fatal("unlisted host accepted")
	}

	var failure = errors.New("custom policy")
	var received string
	config.HostPolicy = func(_ context.Context, host string) error {
		received = host
		return failure
	}
	if err := config.hostPolicy()(context.Background(), "example.com"); !errors.Is(err, failure) || received != "example.com" {
		t.Fatalf("custom policy = %q/%v", received, err)
	}
}
