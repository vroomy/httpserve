package main

import (
	"log"

	"github.com/vroomy/apex"
)

func main() {
	var (
		engine *apex.Engine
		svc    Service
		err    error
	)

	engine = apex.New()
	if err = engine.GET("/ping", svc.Ping); err != nil {
		log.Fatal(err)
	}

	engine.Set404(svc.NotFound)

	if err = engine.Listen(8080); err != nil {
		log.Fatal(err)
	}
}
