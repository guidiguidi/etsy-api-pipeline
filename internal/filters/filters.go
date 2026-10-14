package filters

import (
	"strings"
	"time"

	"etsy/internal/config"
	"etsy/internal/model"
)

func Pass(cfg *config.Filters, listing *model.Listing, shop *model.Shop) bool {
	price := listing.Price.ToFloat()
	if cfg.Price.Enabled {
		if price < cfg.Price.Min || (cfg.Price.Max > 0 && price > cfg.Price.Max) {
			return false
		}
	}

	if cfg.ListingAge.Enabled {
		age := time.Since(time.Unix(listing.CreationTimestamp, 0))
		if age.Minutes() > float64(cfg.ListingAge.MaxMinutes) {
			return false
		}
	}

	if cfg.ShopListingCount.Enabled {
		if shop.ListingActiveCount < cfg.ShopListingCount.Min {
			return false
		}
	}

	if cfg.ShopSales.Enabled {
		if shop.TransactionSoldCount < cfg.ShopSales.Min {
			return false
		}
	}

	if cfg.Rating.Enabled {
		if shop.ReviewAverage < cfg.Rating.MinAverage || shop.ReviewCount < cfg.Rating.MinReviews {
			return false
		}
	}

	if cfg.Country.Enabled {
		country := strings.ToUpper(shop.ShopLocationCountryISO)
		if len(cfg.Country.Allow) > 0 {
			allowed := false
			for _, a := range cfg.Country.Allow {
				if country == strings.ToUpper(a) {
					allowed = true
					break
				}
			}
			if !allowed {
				return false
			}
		}
		if len(cfg.Country.Deny) > 0 {
			for _, d := range cfg.Country.Deny {
				if country == strings.ToUpper(d) {
					return false
				}
			}
		}
	}

	if cfg.SellerRegistered.Enabled {
		shopCreated := time.Unix(shop.CreateDate, 0)
		if cfg.SellerRegistered.Before != "" {
			before, err := time.Parse("2006-01-02", cfg.SellerRegistered.Before)
			if err == nil && !shopCreated.Before(before) {
				return false
			}
		}
		if cfg.SellerRegistered.After != "" {
			after, err := time.Parse("2006-01-02", cfg.SellerRegistered.After)
			if err == nil && !shopCreated.After(after) {
				return false
			}
		}
	}

	return true
}
