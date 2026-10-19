package main

import (
	"log"

	"github.com/vroomy/apex"
)

func main() {
	var (
		srv *apex.Serve
		svc Service
		err error
	)

	srv = apex.New()
	defer srv.Close()

	srv.GET("/ping", svc.Ping)
	srv.Set404(svc.NotFound)

	if err = srv.Listen(8080); err != nil {
		log.Fatal(err)
	}
}
