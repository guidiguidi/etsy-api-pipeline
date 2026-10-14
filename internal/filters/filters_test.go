package filters

import (
	"testing"

	"etsy/internal/config"
	"etsy/internal/model"
)

func TestPass(t *testing.T) {
	cfg := &config.Filters{
		Price: config.PriceFilter{
			Enabled: true,
			Min:     10.0,
			Max:     50.0,
		},
		ShopListingCount: config.ShopListingCountFilter{
			Enabled: true,
			Min:     5,
		},
	}

	listing := &model.Listing{
		Price: model.Money{
			Amount:       2000,
			Divisor:      100,
			CurrencyCode: "USD",
		},
	}

	shop := &model.Shop{
		ListingActiveCount: 10,
	}

	if !Pass(cfg, listing, shop) {
		t.Errorf("Expected Pass to return true, got false")
	}

	listing.Price.Amount = 500
	if Pass(cfg, listing, shop) {
		t.Errorf("Expected Pass to return false for low price, got true")
	}

	listing.Price.Amount = 2000
	shop.ListingActiveCount = 2
	if Pass(cfg, listing, shop) {
		t.Errorf("Expected Pass to return false for low listing count, got true")
	}
}
