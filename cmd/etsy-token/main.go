package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"etsy/internal/config"
	"github.com/joho/godotenv"
	"encoding/json"
)

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func main() {
	_ = godotenv.Load()
	apiKey := os.Getenv("ETSY_API_KEY")
	if apiKey == "" {
		log.Fatal("ETSY_API_KEY not found in .env")
	}

	// Берем только Keystring
	clientID := strings.Split(apiKey, ":")[0]

	var code, verifier string
	fmt.Print("Введите Code из браузера: ")
	fmt.Scanln(&code)
	fmt.Print("Введите Code Verifier: ")
	fmt.Scanln(&verifier)

	redirectURI := "http://localhost:3000"

	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("client_id", clientID)
	data.Set("redirect_uri", redirectURI)
	data.Set("code", code)
	data.Set("code_verifier", verifier)

	resp, err := http.PostForm("https://api.etsy.com/v3/public/oauth/token", data)
	if err != nil {
		log.Fatalf("Ошибка запроса: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Ошибка от Etsy (%s): %s", resp.Status, string(body))
	}

	var tr TokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		log.Fatalf("Ошибка парсинга JSON: %v", err)
	}

	fmt.Println("\n--- Успех! ---")
	if err := config.SaveTokens(tr.AccessToken, tr.RefreshToken); err != nil {
		fmt.Printf("Ошибка сохранения: %v\n", err)
	} else {
		fmt.Println("Токены успешно сохранены в .env")
	}
}
