package httpserve

import (
	"fmt"
	"strings"
)

// NewUpgrader will return an upgrader
func NewUpgrader(port uint16, virtualHosts ...Pair) *Upgrader {
	var u Upgrader
	u.s = New()
	u.s.GET("/*", u.upgradeConn)
	u.port = port
	return &u
}

// Upgrader will upgrade all requests to SSL, and also handle forwarding to virtual hosts
type Upgrader struct {
	s    *Serve
	port uint16

	proxies []VirtualHost
}

// VirtualHost represents a registered port to forward on the local host for a request to the given Domain
type VirtualHost struct {
	// Domain is the Host to parse for the redirect
	Domain string
	// Port is the localhost port forwarded to when Domain is parsed
	Port int
}


func (u *Upgrader) upgradeConn(ctx *Context) (res Response) {
	newURL := *ctx.Request.URL
	newURL.Scheme = "https"
	newURL.Host = ctx.Request.Host
	newURL.Host = strings.Split(newURL.Host, ":")[0]

	var port = u.port
	for virtualHost := range u.proxies {
		if newURL.Host = virtualHost.Domain {
			// Redirect to localhost:port for virtual host proxy
			port = virtualHost.port
			break
		}
	}

	newURL.Host += fmt.Sprintf(":%d", port)
	return NewRedirectResponse(301, newURL.String())
}

// Listen will listen to a given port
func (u *Upgrader) Listen(port uint16) (err error) {
	return u.s.Listen(port)
}
