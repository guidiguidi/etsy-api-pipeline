package model

import (
	"time"
)

type Listing struct {
	ListingID         int64          `json:"listing_id"`
	ShopID            int64          `json:"shop_id"`
	Title             string         `json:"title"`
	URL               string         `json:"url"`
	Price             Money          `json:"price"`
	CreationTimestamp int64          `json:"creation_timestamp"`
	State             string         `json:"state"`
	ImageURL          string         `json:"image_url"`
	Images            []ListingImage `json:"images"`
	Views             int            `json:"views"`
	Description       string         `json:"description"`
	Tags              []string       `json:"tags"`
	Materials         []string       `json:"materials"`
	TaxonomyID        int64          `json:"taxonomy_id"`
	ShippingOrigin    string         `json:"shipping_origin"`
	CreatedAt         time.Time      `json:"created_at"`
}

type ListingImage struct {
	URLFull string `json:"url_fullxfull"`
}

type Money struct {
	Amount       int64  `json:"amount"`
	Divisor      int64  `json:"divisor"`
	CurrencyCode string `json:"currency_code"`
}

func (m Money) ToFloat() float64 {
	if m.Divisor <= 0 {
		return float64(m.Amount)
	}
	return float64(m.Amount) / float64(m.Divisor)
}

type Shop struct {
	ShopID                 int64     `json:"shop_id"`
	ShopName               string    `json:"shop_name"`
	Login                  string    `json:"login"`
	Email                  string    `json:"email"`
	CreateDate             int64     `json:"create_date"`
	ListingActiveCount     int       `json:"listing_active_count"`
	ReviewCount            int       `json:"review_count"`
	ReviewAverage          float64   `json:"review_average"`
	TransactionSoldCount   int       `json:"transaction_sold_count"`
	ShopLocationCountryISO string    `json:"shop_location_country_iso"`
	IsVacation             bool      `json:"is_vacation"`
	CurrencyCode           string    `json:"currency_code"`
	UserID                 int64     `json:"user_id"`
	EmailValid             bool      `json:"email_valid"`
	CreatedAt              time.Time `json:"created_at"`
}

type Match struct {
	Listing   Listing   `json:"listing"`
	Shop      Shop      `json:"shop"`
	MatchedAt time.Time `json:"matched_at"`
}
