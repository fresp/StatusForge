package main

import (
	"log"

	"github.com/fresp/Statora/internal/server"
)

func main() {
	log.Println("Statora Unified Server Starting...")

	if err := server.RunServer(); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
