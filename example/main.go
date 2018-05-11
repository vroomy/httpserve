package main

import (
	"log"

	"github.com/Hatch1fy/httpserve"
)

func main() {
	var (
		srv *httpserve.Serve
		err error
	)

	srv = httpserve.New()
	defer srv.Close()

	srv.GET("/ping", func(ctx *httpserve.Context) (res httpserve.Response) {
		return httpserve.NewTextResponse(200, []byte("pong"))
	})

	if err = srv.Listen(8080); err != nil {
		log.Fatal(err)
	}
}
