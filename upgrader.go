package httpserve

import (
	"fmt"
	"strings"
)

// NewUpgrader will return an upgrader
func NewUpgrader(port uint16) *Upgrader {
	var u Upgrader
	u.s = New()
	u.s.GET("/*x", u.upgradeConn)
	u.port = port
	return &u
}

// Upgrader will upgrade all requests to SSL
type Upgrader struct {
	s    *Serve
	port uint16
}

func (u *Upgrader) upgradeConn(ctx *Context) (res Response) {
	newURL := *ctx.Request.URL
	newURL.Scheme = "https"
	newURL.Host = ctx.Request.Host
	newURL.Host = strings.Split(newURL.Host, ":")[0]
	newURL.Host += fmt.Sprintf(":%d", u.port)
	return NewRedirectResponse(301, newURL.String())
}

// Listen will listen to a given port
func (u *Upgrader) Listen(port uint16) (err error) {
	return u.s.Listen(port)
}
