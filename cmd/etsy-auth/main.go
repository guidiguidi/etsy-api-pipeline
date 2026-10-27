package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"etsy/internal/config"
	"etsy/internal/etsy"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	apiKey := os.Getenv("ETSY_API_KEY")
	if apiKey == "" {
		log.Fatal("ETSY_API_KEY not found in .env")
	}

	verifier, challenge, err := etsy.GeneratePKCE()
	if err != nil {
		log.Fatalf("Failed to generate PKCE: %v", err)
	}

	// IMPORTANT: Configure Redirect URI in Etsy Developer Console to: https://localhost:3000
	redirectURI := "https://localhost:3000"
	state := "supersecretstate"
	scopes := []string{"listings_r", "shops_r"}

	authURL := etsy.BuildAuthURL(apiKey, redirectURI, state, challenge, scopes)

	codeChan := make(chan string)
	server := &http.Server{Addr: ":3000"}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code != "" {
			fmt.Fprintf(w, "<h1>Success!</h1><p>Code received. Return to terminal.</p>")
			codeChan <- code
		}
	})

	go func() {
		_ = server.ListenAndServe()
	}()

	fmt.Printf("\n--- Etsy OAuth 2.0 Helper ---\n")
	fmt.Printf("1. Opening browser...\n")
	openBrowser(authURL)

	fmt.Printf("\nIf the browser shows an error or fails to redirect automatically,\n")
	fmt.Printf("copy and paste the FULL URL from the address bar here:\n")
	fmt.Print("\nURL or Code: ")

	inputChan := make(chan string)
	go func() {
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		// Attempt to parse redirect URL and extract ?code= query parameter
		if u, err := url.Parse(input); err == nil {
			if c := u.Query().Get("code"); c != "" {
				inputChan <- c
				return
			}
		}

		// If not a full URL or lacks code parameter, assume raw code was provided directly
		inputChan <- input
	}()

	var finalCode string
	select {
	case finalCode = <-codeChan:
		fmt.Println("\n✅ Code received automatically!")
	case finalCode = <-inputChan:
		fmt.Println("\n✅ Code accepted manually!")
	case <-time.After(5 * time.Minute):
		log.Fatal("Timeout.")
	}

	client := etsy.NewClient(apiKey, "", "", 10*time.Second)
	access, refresh, err := client.ExchangeCodeForToken(finalCode, verifier, redirectURI)
	if err != nil {
		log.Fatalf("❌ Exchange failed: %v", err)
	}

	config.SaveTokens(access, refresh)
	fmt.Println("\n🎉 ALL DONE! You can now run the monitor.")

	ctx, _ := context.WithTimeout(context.Background(), 1*time.Second)
	server.Shutdown(ctx)
}

func openBrowser(url string) {
	switch runtime.GOOS {
	case "darwin":
		exec.Command("open", url).Start()
	case "windows":
		exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "linux":
		exec.Command("xdg-open", url).Start()
	}
}
