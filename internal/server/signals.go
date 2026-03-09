// cmd/server/signals.go
// Package server provides signal handling utilities.
package server

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// SetupShutdownSignalHandler sets up a signal handler for graceful shutdown
func SetupShutdownSignalHandler(cancel context.CancelFunc) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Printf("[SHUTDOWN] Received signal: %s", sig.String())
		log.Printf("[SHUTDOWN] Initiating graceful shutdown...")

		cancel() // Cancel the main context

		// Wait for 30 seconds to allow goroutines to finish
		gracefulTimer := time.NewTimer(30 * time.Second)
		<-gracefulTimer.C

		log.Printf("[SHUTDOWN] Force exit after timeout")
		os.Exit(1)
	}()
}
