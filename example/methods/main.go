package main

import (
	"log"

	"github.com/vroomy/common"
	"github.com/vroomy/httpserve"
)

func main() {
	var (
		srv *httpserve.Serve
		svc Service
		err error
	)

	srv = httpserve.New()
	defer srv.Close()

	srv.GET("/ping", svc.Ping)
	srv.Set404(svc.NotFound)

	if err = srv.Listen(8080); err != nil {
		log.Fatal(err)
	}
}

// Service manages a web service
type Service struct{}

// Ping is the ping endpoint handler
func (s *Service) Ping(ctx common.Context) {
	ctx.WriteString(200, "text/plain", "pong")
}

// NotFound is the 404 handler
func (s *Service) NotFound(ctx common.Context) {
	ctx.WriteString(404, "text/plain", "Oh shoot, this page doesn't exist")
}
