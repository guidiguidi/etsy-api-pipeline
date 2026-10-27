package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

type Config struct {
	EtsyAPIKey        string `yaml:"-"`
	EtsyAccessToken   string `yaml:"-"`
	EtsyRefreshToken  string `yaml:"-"`
	ScanIntervalSec   int    `yaml:"scan_interval_sec"`
	SqlitePath        string `yaml:"sqlite_path"`
	OutputDir         string `yaml:"output_dir"`
	WorkerConcurrency int    `yaml:"worker_concurrency"`
	HTTPTimeoutSec    int    `yaml:"http_timeout_sec"`
	LogLevel          string `yaml:"log_level"`
	Debug             bool   `yaml:"debug"`
	EmailVerifyMode   string `yaml:"email_verify_mode"` // "smtp" or "api"
	EmailAPIKey       string `yaml:"email_api_key"`
	Mode              string `yaml:"mode"`
	Filters           Filters `yaml:"filters"`
	Telegram          TelegramConfig `yaml:"telegram"`
}

type TelegramConfig struct {
	Enabled     bool   `yaml:"enabled"`
	APIID       int    `yaml:"api_id"`
	APIHash     string `yaml:"api_hash"`
	SessionPath string `yaml:"session_path"`
	BotUsername string `yaml:"bot_username"`
}

type Filters struct {
	Price            PriceFilter            `yaml:"price"`
	SellerRegistered SellerRegisteredFilter `yaml:"seller_registered"`
	ListingAge       ListingAgeFilter       `yaml:"listing_age"`
	ShopListingCount ShopListingCountFilter `yaml:"shop_listing_count"`
	ShopSales        ShopSalesFilter        `yaml:"shop_sales"`
	Rating           RatingFilter           `yaml:"rating"`
	Country          CountryFilter          `yaml:"country"`
}

type PriceFilter struct {
	Enabled bool    `yaml:"enabled"`
	Min     float64 `yaml:"min"`
	Max     float64 `yaml:"max"`
}

type SellerRegisteredFilter struct {
	Enabled bool   `yaml:"enabled"`
	Before  string `yaml:"before"`
	After   string `yaml:"after"`
}

type ListingAgeFilter struct {
	Enabled    bool `yaml:"enabled"`
	MaxMinutes int  `yaml:"max_minutes"`
}

type ShopListingCountFilter struct {
	Enabled bool `yaml:"enabled"`
	Min     int  `yaml:"min"`
}

type ShopSalesFilter struct {
	Enabled bool `yaml:"enabled"`
	Min     int  `yaml:"min"`
}

type RatingFilter struct {
	Enabled    bool    `yaml:"enabled"`
	MinAverage float64 `yaml:"min_average"`
	MinReviews int     `yaml:"min_reviews"`
}

type CountryFilter struct {
	Enabled bool     `yaml:"enabled"`
	Allow   []string `yaml:"allow"`
	Deny    []string `yaml:"deny"`
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		EtsyAPIKey:      os.Getenv("ETSY_API_KEY"),
		EtsyAccessToken: os.Getenv("ETSY_ACCESS_TOKEN"),
		EtsyRefreshToken: os.Getenv("ETSY_REFRESH_TOKEN"),
		ScanIntervalSec: getEnvInt("SCAN_INTERVAL_SEC", 60),
		SqlitePath:      getEnv("SQLITE_PATH", "etsy.db"),
		OutputDir:       getEnv("OUTPUT_DIR", "output"),
		WorkerConcurrency: getEnvInt("WORKER_CONCURRENCY", 5),
		HTTPTimeoutSec:  getEnvInt("HTTP_TIMEOUT_SEC", 30),
		LogLevel:        getEnv("LOG_LEVEL", "info"),
		Debug:           os.Getenv("DEBUG") == "true",
		EmailVerifyMode: getEnv("EMAIL_VERIFY_MODE", "smtp"),
		EmailAPIKey:     os.Getenv("EMAIL_API_KEY"),
		Telegram: TelegramConfig{
			APIID:       getEnvInt("TG_API_ID", 2040),
			APIHash:     getEnv("TG_API_HASH", "b18441a1ff607e10a989891a5462e627"),
			BotUsername: getEnv("TG_BOT_USERNAME", "Kartatai_RO_BOT"),
			SessionPath: getEnv("TG_SESSION_PATH", "tg.session"),
		},
	}

	yamlFile, err := os.ReadFile("config.yaml")
	if err == nil {
		err = yaml.Unmarshal(yamlFile, cfg)
		if err != nil {
			return nil, err
		}
	}

	return cfg, nil
}

func SaveTokens(accessToken, refreshToken string) error {
	env, _ := godotenv.Read()
	if env == nil {
		env = make(map[string]string)
	}

	env["ETSY_ACCESS_TOKEN"] = accessToken
	env["ETSY_REFRESH_TOKEN"] = refreshToken

	return godotenv.Write(env, ".env")
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value, ok := os.LookupEnv(key); ok {
		i, err := strconv.Atoi(value)
		if err == nil {
			return i
		}
	}
	return fallback
}
