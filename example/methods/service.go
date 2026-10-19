package main

import "github.com/vroomy/apex"

// Service manages a web service
type Service struct{}

// Ping is the ping endpoint handler
func (s *Service) Ping(ctx *apex.Context) {
	ctx.WriteString(200, "text/plain", "pong")
}

// NotFound is the 404 handler
func (s *Service) NotFound(ctx *apex.Context) {
	ctx.WriteString(404, "text/plain", "Oh shoot, this page doesn't exist")
}
