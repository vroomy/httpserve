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
	defer engine.Close()

	engine.GET("/ping", svc.Ping)
	engine.Set404(svc.NotFound)

	if err = engine.Listen(8080); err != nil {
		log.Fatal(err)
	}
}
