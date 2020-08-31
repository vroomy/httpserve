package main

import (
	"log"

	"github.com/vroomy/common"
	"github.com/vroomy/httpserve"
)

func main() {
	var (
		srv *httpserve.Serve
		err error
	)

	srv = httpserve.New()
	defer srv.Close()

	srv.GET("/ping", func(ctx common.Context) (res common.Response) {
		return httpserve.NewTextResponse(200, []byte("pong"))
	})

	srv.Set404(func(ctx common.Context) (res common.Response) {
		return httpserve.NewTextResponse(404, []byte("Oh shoot, this page doesn't exist"))
	})

	if err = srv.Listen(8080); err != nil {
		log.Fatal(err)
	}
}
