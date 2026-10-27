package worker

import (
	"context"
	"log"
	"time"

	"etsy/internal/config"
	"etsy/internal/email"
	"etsy/internal/etsy"
	"etsy/internal/export"
	"etsy/internal/filters"
	"etsy/internal/model"
	"etsy/internal/storage"
	"etsy/internal/telegram"
)

type Worker struct {
	cfg            *config.Config
	client         *etsy.Client
	storage        *storage.SQLite
	emailValidator *email.Validator
	tgSender       *telegram.Sender
}

func NewWorker(cfg *config.Config, client *etsy.Client, storage *storage.SQLite) *Worker {
	return &Worker{
		cfg:            cfg,
		client:         client,
		storage:        storage,
		emailValidator: email.NewValidator(time.Duration(cfg.HTTPTimeoutSec)*time.Second, cfg.EmailVerifyMode, cfg.EmailAPIKey),
		tgSender:       telegram.NewSender(&cfg.Telegram),
	}
}

func (w *Worker) RunOnce() error {
	log.Println("🚀 Starting scan run...")
	
	if err := export.ClearExports(w.cfg.OutputDir); err != nil {
		log.Printf("Warning: could not clear old exports: %v", err)
	}

	listings, err := w.client.GetActiveListings(100, 0)
	if err != nil {
		return err
	}

	log.Printf("📦 Fetched %d listings. Starting validation...", len(listings))

	var matches []*model.Match

	for i, l := range listings {
		if w.cfg.Debug {
			log.Printf("📦 Validating listing %d/%d...", i+1, len(listings))
		}
		match := w.processAndFilter(l)
		if match != nil {
			matches = append(matches, match)
		}
		// Throttling delay to prevent rate-limiting on frequent SMTP handshakes
		time.Sleep(2 * time.Second)
	}

	if len(matches) == 0 {
		log.Println("😴 No valid matches found in this run.")
		return nil
	}

	log.Printf("🎯 Found %d VALID matches. Starting Telegram queue...", len(matches))

	// Process queued candidate records sequentially
	for i, m := range matches {
		log.Printf("📤 Telegram Queue [%d/%d]: Processing %s", i+1, len(matches), m.Shop.ShopName)
		if err := w.tgSender.SendMatch(context.Background(), m); err != nil {
			log.Printf("❌ Telegram Queue Error: %v", err)
		}
	}

	log.Println("🏁 Scan run completed.")
	return nil
}

// processAndFilter evaluates listing criteria, validates email, and returns Match on success
func (w *Worker) processAndFilter(l model.Listing) *model.Match {
	if w.cfg.Debug {
		log.Printf("🔍 [DEBUG] Processing listing %d from shop %d", l.ListingID, l.ShopID)
	}
	processed, _ := w.storage.IsProcessed(l.ListingID)
	if processed {
		return nil
	}

	defer w.storage.MarkProcessed(l.ListingID)

	_ = w.storage.SaveListing(&l)

	shop, err := w.client.GetShop(l.ShopID)
	if err != nil {
		if w.cfg.Debug {
			log.Printf("🔍 [DEBUG] Failed to get shop %d: %v", l.ShopID, err)
		}
		return nil
	}

	shop.Login = shop.ShopName
	shop.Email = email.GenerateEmail(shop.ShopName)

	if w.cfg.Debug {
		log.Printf("🔍 [DEBUG] Validating email %s for shop %s", shop.Email, shop.ShopName)
	}

	valid, err := w.emailValidator.Validate(shop.Email)
	if err != nil {
		if w.cfg.Debug {
			log.Printf("🔍 [DEBUG] Email validation error: %v", err)
		}
		valid = false
	}
	shop.EmailValid = valid

	_ = w.storage.SaveShop(shop)

	if !valid {
		if w.cfg.Debug {
			log.Printf("🔍 [DEBUG] Skipping %s: Email not valid", shop.ShopName)
		}
		return nil
	}

	if !filters.Pass(&w.cfg.Filters, &l, shop) {
		if w.cfg.Debug {
			log.Printf("🔍 [DEBUG] Skipping %s: Failed filters", shop.ShopName)
		}
		return nil
	}

	match := &model.Match{
		Listing:   l,
		Shop:      *shop,
		MatchedAt: time.Now(),
	}

	// Persist match record to SQLite and export output files
	_ = w.storage.SaveMatch(l.ListingID, shop.ShopID)
	_ = export.SaveToJSONL(w.cfg.OutputDir, match)
	_ = export.SaveToCSV(w.cfg.OutputDir, match)

	log.Printf("✅ Validated: %s (%s) - MATCH", shop.ShopName, shop.Email)
	return match
}

func (w *Worker) StartDaemon() {
	ctx := context.Background()
	if err := w.tgSender.Start(ctx); err != nil {
		log.Printf("Warning: could not start Telegram sender: %v", err)
	}

	ticker := time.NewTicker(time.Duration(w.cfg.ScanIntervalSec) * time.Second)
	defer ticker.Stop()

	for {
		if err := w.RunOnce(); err != nil {
			log.Printf("Scan run failed: %v", err)
		}
		log.Printf("😴 Waiting %d seconds for next scan...", w.cfg.ScanIntervalSec)
		<-ticker.C
	}
}
