package storage

import (
	"database/sql"
	"fmt"
	"time"

	"etsy/internal/model"
	_ "github.com/mattn/go-sqlite3"
)

type SQLite struct {
	db *sql.DB
}

func NewSQLite(path string) (*SQLite, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	s := &SQLite{db: db}
	if err := s.Init(); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *SQLite) Init() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS listings_raw (
			listing_id INTEGER PRIMARY KEY,
			shop_id INTEGER,
			title TEXT,
			url TEXT,
			price REAL,
			currency_code TEXT,
			creation_timestamp INTEGER,
			state TEXT,
			image_url TEXT,
			data TEXT,
			created_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS shops_raw (
			shop_id INTEGER PRIMARY KEY,
			shop_name TEXT,
			login TEXT,
			email TEXT,
			email_valid INTEGER,
			create_date INTEGER,
			listing_active_count INTEGER,
			review_count INTEGER,
			review_average REAL,
			transaction_sold_count INTEGER,
			country_iso TEXT,
			data TEXT,
			created_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS matches (
			listing_id INTEGER PRIMARY KEY,
			shop_id INTEGER,
			matched_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS processed_listing_ids (
			listing_id INTEGER PRIMARY KEY,
			processed_at DATETIME
		)`,
	}

	for _, q := range queries {
		if _, err := s.db.Exec(q); err != nil {
			return fmt.Errorf("failed to execute init query: %w", err)
		}
	}

	// Миграция: добавляем колонки, если таблица уже существовала без них
	s.db.Exec("ALTER TABLE shops_raw ADD COLUMN login TEXT")
	s.db.Exec("ALTER TABLE shops_raw ADD COLUMN email TEXT")
	s.db.Exec("ALTER TABLE shops_raw ADD COLUMN email_valid INTEGER")

	return nil
}

func (s *SQLite) SaveListing(l *model.Listing) error {
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO listings_raw (listing_id, shop_id, title, url, price, currency_code, creation_timestamp, state, image_url, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		l.ListingID, l.ShopID, l.Title, l.URL, l.Price.ToFloat(), l.Price.CurrencyCode, l.CreationTimestamp, l.State, l.ImageURL, time.Now(),
	)
	return err
}

func (s *SQLite) SaveShop(sh *model.Shop) error {
	emailValidInt := 0
	if sh.EmailValid {
		emailValidInt = 1
	}
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO shops_raw (shop_id, shop_name, login, email, email_valid, create_date, listing_active_count, review_count, review_average, transaction_sold_count, country_iso, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		sh.ShopID, sh.ShopName, sh.Login, sh.Email, emailValidInt, sh.CreateDate, sh.ListingActiveCount, sh.ReviewCount, sh.ReviewAverage, sh.TransactionSoldCount, sh.ShopLocationCountryISO, time.Now(),
	)
	return err
}

func (s *SQLite) SaveMatch(listingID, shopID int64) error {
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO matches (listing_id, shop_id, matched_at) VALUES (?, ?, ?)`,
		listingID, shopID, time.Now(),
	)
	return err
}

func (s *SQLite) IsProcessed(listingID int64) (bool, error) {
	var exists bool
	err := s.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM processed_listing_ids WHERE listing_id = ?)`, listingID).Scan(&exists)
	return exists, err
}

func (s *SQLite) MarkProcessed(listingID int64) error {
	_, err := s.db.Exec(`INSERT OR REPLACE INTO processed_listing_ids (listing_id, processed_at) VALUES (?, ?)`, listingID, time.Now())
	return err
}

func (s *SQLite) GetLatestMatch() (*model.Match, error) {
	query := `
		SELECT l.listing_id, l.shop_id, l.title, l.url, l.price, l.currency_code, l.image_url,
		       s.shop_id, s.shop_name, s.email, s.email_valid
		FROM matches m
		JOIN listings_raw l ON m.listing_id = l.listing_id
		JOIN shops_raw s ON m.shop_id = s.shop_id
		ORDER BY m.matched_at DESC LIMIT 1`

	var m model.Match
	var priceVal float64
	err := s.db.QueryRow(query).Scan(
		&m.Listing.ListingID, &m.Listing.ShopID, &m.Listing.Title, &m.Listing.URL, &priceVal, &m.Listing.Price.CurrencyCode, &m.Listing.ImageURL,
		&m.Shop.ShopID, &m.Shop.ShopName, &m.Shop.Email, &m.Shop.EmailValid,
	)
	if err != nil {
		return nil, err
	}
	m.Listing.Price.Amount = int64(priceVal * 100)
	m.Listing.Price.Divisor = 100
	return &m, nil
}

func (s *SQLite) Close() error {
	return s.db.Close()
}
