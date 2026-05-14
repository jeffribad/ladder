package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	defaultPort    = "8080"
	defaultTimeout = 30 * time.Second
	readTimeout    = 30 * time.Second // increased from 15s to handle slower clients
	writeTimeout   = 60 * time.Second // increased from 30s for larger proxied responses
	idleTimeout    = 120 * time.Second // increased from 60s to reduce reconnect overhead
)

// version is set at build time via ldflags.
var version = "dev"

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	app, err := NewApp()
	if err != nil {
		log.Fatalf("failed to initialize app: %v", err)
	}

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      app.Router(),
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
	}

	// Start server in a goroutine so we can handle graceful shutdown.
	go func() {
		log.Printf("ladder %s listening on %s", version, srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server.
	// Also handle SIGHUP so the process can be signalled without killing it
	// (useful when running under systemd or in a tmux session).
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	<-quit

	log.Println("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("forced shutdown: %v", err)
	}

	log.Println("server stopped")
}
