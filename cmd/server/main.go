package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/jdb/oasis-formatter/internal/api"
	"github.com/jdb/oasis-formatter/internal/translator"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	staticDir := os.Getenv("STATIC_DIR")
	if staticDir == "" {
		candidates := []string{
			"web/dist",
			filepath.Join("/app", "web", "dist"),
		}
		for _, c := range candidates {
			if info, err := os.Stat(c); err == nil && info.IsDir() {
				staticDir = c
				break
			}
		}
	}

	provider := translator.NewGTXProvider()
	handler := api.NewHandler(provider, staticDir)

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      5 * time.Minute, // Allow long-lived SSE translation streams
		IdleTimeout:       120 * time.Second,
	}

	log.Printf("oasis-formatter listening on :%s (staticDir=%q)", port, staticDir)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}
