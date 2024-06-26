package httpserve

import (
	"time"

	"golang.org/x/crypto/acme/autocert"
)

// Config is the basic configuration
type Config struct {
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	MaxHeaderBytes int
}

type AutoCertConfig struct {
	DirCache string

	// For TLS hosts, use either Hosts or HostPolicy depending on your use-case:
	//	- Use Hosts when your list is static
	//	- Host HostPolicy when your list is dynamic
	// Hosts is a static list of acceptable TLS hosts
	Hosts []string
	// HostPolicy is a TLS host policy that can be dynamically updated
	HostPolicy autocert.HostPolicy
}

func (a *AutoCertConfig) hostPolicy() autocert.HostPolicy {
	if a.HostPolicy != nil {
		return a.HostPolicy
	}

	return autocert.HostWhitelist(a.Hosts...)
}
