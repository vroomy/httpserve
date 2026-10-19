package main

import (
	"log"

	"github.com/vroomy/apex"
)

func main() {
	var (
		engine *apex.Engine
		err    error
	)

	engine = apex.New()
	defer engine.Close()

	if err = engine.GET("/ping", func(ctx *apex.Context) {
		ctx.WriteString(200, "text/plain", "pong")
	}); err != nil {
		log.Fatal(err)
	}

	engine.Set404(func(ctx *apex.Context) {
		ctx.WriteString(404, "text/plain", "Oh shoot, this page doesn't exist")
	})

	if err = engine.Listen(8080); err != nil {
		log.Fatal(err)
	}
}
