package export

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"etsy/internal/model"
)

// ClearExports removes the old matches files.
func ClearExports(dir string) error {
	jsonlPath := filepath.Join(dir, "matches.jsonl")
	csvPath := filepath.Join(dir, "matches.csv")
	
	_ = os.Remove(jsonlPath)
	_ = os.Remove(csvPath)
	
	return os.MkdirAll(dir, 0755)
}

func SaveToJSONL(dir string, match *model.Match) error {
	path := filepath.Join(dir, "matches.jsonl")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	data, err := json.Marshal(match)
	if err != nil {
		return err
	}

	_, err = f.Write(append(data, '\n'))
	return err
}

func SaveToCSV(dir string, match *model.Match) error {
	path := filepath.Join(dir, "matches.csv")
	exists := true
	if _, err := os.Stat(path); os.IsNotExist(err) {
		exists = false
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	writer := csv.NewWriter(f)
	defer writer.Flush()

	if !exists {
		header := []string{
			"listing_id", "title", "price", "currency_code", "listing_url", "image_url",
			"created_at", "listing_age_minutes", "shop_id", "shop_name", "login", "email",
			"shop_created_at", "shop_age_days", "listing_active_count", "transaction_sold_count",
			"review_average", "review_count", "country", "matched_at",
		}
		if err := writer.Write(header); err != nil {
			return err
		}
	}

	listingAge := time.Since(time.Unix(match.Listing.CreationTimestamp, 0)).Minutes()
	shopAge := time.Since(time.Unix(match.Shop.CreateDate, 0)).Hours() / 24

	record := []string{
		strconv.FormatInt(match.Listing.ListingID, 10),
		match.Listing.Title,
		fmt.Sprintf("%.2f", match.Listing.Price.ToFloat()),
		match.Listing.Price.CurrencyCode,
		match.Listing.URL,
		match.Listing.ImageURL,
		time.Unix(match.Listing.CreationTimestamp, 0).Format(time.RFC3339),
		fmt.Sprintf("%.0f", listingAge),
		strconv.FormatInt(match.Shop.ShopID, 10),
		match.Shop.ShopName,
		match.Shop.Login,
		match.Shop.Email,
		time.Unix(match.Shop.CreateDate, 0).Format(time.RFC3339),
		fmt.Sprintf("%.0f", shopAge),
		strconv.Itoa(match.Shop.ListingActiveCount),
		strconv.Itoa(match.Shop.TransactionSoldCount),
		fmt.Sprintf("%.2f", match.Shop.ReviewAverage),
		strconv.Itoa(match.Shop.ReviewCount),
		match.Shop.ShopLocationCountryISO,
		match.MatchedAt.Format(time.RFC3339),
	}

	return writer.Write(record)
}
