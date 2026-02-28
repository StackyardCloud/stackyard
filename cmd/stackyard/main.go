package main

import (
	"errors"
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/stackyard/stackyard/internal/server"
)

func main() {
	var addr string
	flag.StringVar(&addr, "addr", envOrDefault("STACKYARD_ADDR", ":4566"), "HTTP listen address")
	flag.Parse()

	cfg := server.Config{
		Addr:      addr,
		AccessKey: envOrDefault("STACKYARD_ACCESS_KEY", "stackyard"),
		SecretKey: envOrDefault("STACKYARD_SECRET_KEY", "stackyard"),
		LogLevel:  envOrDefault("STACKYARD_LOG_LEVEL", "info"),
	}
	srv := server.New(cfg)
	log.Printf("stackyard listening on %s", addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server failed: %v", err)
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
