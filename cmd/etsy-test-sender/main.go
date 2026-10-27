package main

import (
	"context"
	"log"
	"time"

	"etsy/internal/config"
	"etsy/internal/storage"
	"etsy/internal/telegram"
)

func main() {
	// 1. Load application configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 2. Initialize database and retrieve latest match record
	s, err := storage.NewSQLite(cfg.SqlitePath)
	if err != nil {
		log.Fatalf("Failed to open DB: %v", err)
	}
	defer s.Close()

	match, err := s.GetLatestMatch()
	if err != nil {
		log.Fatalf("❌ No matches found in DB: %v. Run the monitor first!", err)
	}

	// 3. Initialize message dispatcher
	sender := telegram.NewSender(&cfg.Telegram)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	log.Println("🔌 Connecting to Telegram...")
	if err := sender.Start(ctx); err != nil {
		log.Fatalf("Failed to start sender: %v", err)
	}

	// 4. Run test dispatch against live payload
	log.Printf("🚀 Starting test for shop from DB: %s (%s)", match.Shop.ShopName, match.Shop.Email)
	err = sender.SendMatch(ctx, match)
	if err != nil {
		log.Printf("❌ Test FAILED: %v", err)
	} else {
		log.Println("✅ Test COMPLETED successfully!")
	}
}
