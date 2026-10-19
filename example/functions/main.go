package main

import (
	"log"

	"github.com/vroomy/apex"
)

func main() {
	var err error
	engine := apex.New()
	if err = engine.GET("/ping", servePong); err != nil {
		log.Fatal(err)
	}

	engine.Set404(serve404)

	if err = engine.Listen(8080); err != nil {
		log.Fatal(err)
	}
}

func servePong(ctx *apex.Context) {
	ctx.WriteString(200, "text/plain", "pong")
}

func serve404(ctx *apex.Context) {
	ctx.WriteString(404, "text/plain", "Oh shoot, this page doesn't exist")
}
