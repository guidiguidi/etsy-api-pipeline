package main

import (
	"context"
	"log"
	"path/filepath"

	"etsy/internal/config"
	"etsy/internal/telegram"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if !cfg.Telegram.Enabled {
		log.Fatal("Telegram is disabled in config")
	}

	sender := telegram.NewSender(&cfg.Telegram)
	
	// Path to processed results file
	matchesFile := filepath.Join(cfg.OutputDir, "matches.jsonl")
	
	log.Printf("Starting sender for file: %s", matchesFile)
	
	ctx := context.Background()
	if err := sender.SendFromJSONL(ctx, matchesFile); err != nil {
		log.Fatalf("Sender failed: %v", err)
	}
	
	log.Println("All matches processed successfully")
}
