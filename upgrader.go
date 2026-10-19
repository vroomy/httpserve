package apex

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
	s    *Engine
	port uint16

	proxies []VirtualHost
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

// Close releases the redirect listener and prevents subsequent startup.
func (u *Upgrader) Close() error {
	return u.s.Close()
}
