package main

import (
	"log"

	"github.com/vroomy/apex"
)

func main() {
	var (
		srv *apex.Serve
		err error
	)

	srv = apex.New()
	defer srv.Close()

	if err = srv.GET("/ping", func(ctx *apex.Context) {
		ctx.WriteString(200, "text/plain", "pong")
	}); err != nil {
		log.Fatal(err)
	}

	srv.Set404(func(ctx *apex.Context) {
		ctx.WriteString(404, "text/plain", "Oh shoot, this page doesn't exist")
	})

	if err = srv.Listen(8080); err != nil {
		log.Fatal(err)
	}
}
