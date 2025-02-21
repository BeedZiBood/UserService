package main

import (
	"UserService/internal/config"
	"UserService/internal/entrypoint"
	"log"
)

func main() {
	cfg := config.MustLoad()

	ep, err := entrypoint.New(&cfg)
	if err != nil {
		log.Fatalf("cannot create entrypoint: %s", err)
		return
	}

	ep.Run()
}
