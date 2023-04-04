package httpserve

import (
	"fmt"
	"strings"
)

// NewUpgrader will return an upgrader
func NewUpgrader(port uint16, virtualHosts ...VirtualHost) *Upgrader {
	var u Upgrader
	u.s = New()
	u.s.GET("/*", u.upgradeConn)
	u.port = port
	u.proxies = virtualHosts
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
	Port uint16
}

func (u *Upgrader) upgradeConn(ctx *Context) {
	newURL := *ctx.Request().URL
	newURL.Scheme = "https"
	newURL.Host = ctx.Request().Host
	newURL.Host = strings.Split(newURL.Host, ":")[0]

	var port = u.port
	for _, virtualHost := range u.proxies {
		if virtualHost.Port != u.port && newURL.Host == virtualHost.Domain {
			// Redirect to localhost:port for virtual host proxy
			port = virtualHost.Port
			break
		}
	}

	newURL.Host += fmt.Sprintf(":%d", port)
	ctx.Redirect(301, newURL.String())
}

// Listen will listen to a given port
func (u *Upgrader) Listen(port uint16) (err error) {
	return u.s.Listen(port)
}
