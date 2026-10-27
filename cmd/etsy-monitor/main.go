package main

import (
	"log"
	"time"

	"etsy/internal/config"
	"etsy/internal/etsy"
	"etsy/internal/storage"
	"etsy/internal/worker"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if cfg.EtsyAPIKey == "" {
		log.Fatal("ETSY_API_KEY is required")
	}

	s, err := storage.NewSQLite(cfg.SqlitePath)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	defer s.Close()

	client := etsy.NewClient(cfg.EtsyAPIKey, cfg.EtsyAccessToken, cfg.EtsyRefreshToken, time.Duration(cfg.HTTPTimeoutSec)*time.Second)
	w := worker.NewWorker(cfg, client, s)

	if cfg.Mode == "daemon" {
		log.Println("Starting in daemon mode...")
		w.StartDaemon()
	} else {
		log.Println("Starting in CLI mode (single run)...")
		if err := w.RunOnce(); err != nil {
			log.Fatalf("Run failed: %v", err)
		}
	}
}
