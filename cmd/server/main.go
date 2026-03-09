package main

import (
	"log"

	"status-platform/internal/server"
)

func main() {
	log.Println("StatusForge Unified Server Starting...")

	if err := server.RunServer(); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
