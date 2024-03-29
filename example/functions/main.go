package main

import (
	"log"

	"github.com/vroomy/httpserve"
)

func main() {
	var (
		srv *httpserve.Serve
		err error
	)

	srv = httpserve.New()
	defer srv.Close()

	if err = srv.GET("/ping", func(ctx *httpserve.Context) {
		ctx.WriteString(200, "text/plain", "pong")
	}); err != nil {
		log.Fatal(err)
	}

	srv.Set404(func(ctx *httpserve.Context) {
		ctx.WriteString(404, "text/plain", "Oh shoot, this page doesn't exist")
	})

	if err = srv.Listen(8080); err != nil {
		log.Fatal(err)
	}
}
