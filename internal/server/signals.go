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
func SetupShutdownSignalHandler(cancel context.CancelFunc, timeout time.Duration) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Printf("[SHUTDOWN] Received signal: %s", sig.String())
		log.Printf("[SHUTDOWN] Initiating graceful shutdown...")

		// Trigger cancellation (worker, etc)
		cancel()

		// Force exit after timeout (safety net)
		if timeout > 0 {
			timer := time.NewTimer(timeout)
			defer timer.Stop()

			<-timer.C
			log.Printf("[SHUTDOWN] Force exit after %s timeout", timeout)
			os.Exit(1)
		}
	}()
}