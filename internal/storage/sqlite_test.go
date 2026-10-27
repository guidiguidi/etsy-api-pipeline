package storage

import (
	"testing"
	"time"

	"etsy/internal/model"
)

func TestSQLite_InitAndSave(t *testing.T) {
	s, err := NewSQLite(":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory sqlite: %v", err)
	}
	defer s.Close()

	listing := &model.Listing{
		ListingID:         123456,
		ShopID:            789,
		Title:             "Vintage Leather Jacket",
		URL:               "https://etsy.com/listing/123456",
		State:             "active",
		CreationTimestamp: time.Now().Unix(),
		Price: model.Money{
			Amount:       9500,
			Divisor:      100,
			CurrencyCode: "USD",
		},
	}

	if err := s.SaveListing(listing); err != nil {
		t.Fatalf("failed to save listing: %v", err)
	}

	// Test processing tracking
	processed, err := s.IsProcessed(123456)
	if err != nil {
		t.Fatalf("failed to check if processed: %v", err)
	}
	if processed {
		t.Errorf("expected listing 123456 to not be processed yet")
	}

	if err := s.MarkProcessed(123456); err != nil {
		t.Fatalf("failed to mark processed: %v", err)
	}

	processed, err = s.IsProcessed(123456)
	if err != nil {
		t.Fatalf("failed to re-check if processed: %v", err)
	}
	if !processed {
		t.Errorf("expected listing 123456 to be marked as processed")
	}
}

func TestSQLite_SaveShop(t *testing.T) {
	s, err := NewSQLite(":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory sqlite: %v", err)
	}
	defer s.Close()

	shop := &model.Shop{
		ShopID:             999,
		ShopName:           "VintageVault",
		ListingActiveCount: 42,
		ReviewCount:        150,
		ReviewAverage:      4.95,
	}

	if err := s.SaveShop(shop); err != nil {
		t.Fatalf("failed to save shop: %v", err)
	}
}
