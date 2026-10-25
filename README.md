# Etsy Scraper & Monitor

A Go-based service for monitoring new Etsy listings, enriching them with shop data, and filtering based on custom criteria.

## Features

- **Two Modes**: CLI (single run) and Daemon (continuous polling).
- **Data Enrichment**: Automatically fetches shop-level metrics (listing count, sales, ratings, etc.).
- **Flexible Filtering**: Filter by price, seller age, listing age, shop size, sales, rating, and country.
- **Persistent Storage**: Saves data to SQLite for deduplication and historical analysis.
- **Multiple Exports**: Export matching listings to JSON Lines and CSV.

## Requirements

- Go 1.22+
- Etsy API Credentials (API Key and OAuth 2.0 Access Token)

## Installation

1. Clone the repository.
2. Install dependencies:
   ```bash
   go mod tidy
   ```

## Configuration

1. Copy `.env.example` to `.env` and fill in your Etsy API credentials.
2. Copy `config.example.yaml` to `config.yaml` and adjust filters and settings.

## Usage

### CLI Mode (Single Run)
```bash
go run cmd/etsy-monitor/main.go
```

### Daemon Mode
Set `mode: daemon` in `config.yaml` or use environment variables, then run:
```bash
go run cmd/etsy-monitor/main.go
```

## Output

- **SQLite**: Data is stored in `etsy.db` (configurable).
- **JSONL**: Matching listings are appended to `output/matches.jsonl`.
- **CSV**: Matching listings are appended to `output/matches.csv`.
