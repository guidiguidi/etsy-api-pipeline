package email

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/smtp"
	"os"
	"strings"
	"time"
)

type Validator struct {
	Timeout    time.Duration
	Mode       string 
	APIKey     string
	HTTPClient *http.Client
}

func NewValidator(timeout time.Duration, mode, apiKey string) *Validator {
	return &Validator{
		Timeout: timeout,
		Mode:    mode,
		APIKey:  apiKey,
		HTTPClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (v *Validator) Validate(email string) (bool, error) {
	if v.Mode == "api" && v.APIKey != "" {
		return v.validateViaAPI(email)
	}
	return v.validateViaSMTP(email)
}

func (v *Validator) validateViaAPI(email string) (bool, error) {
	url := fmt.Sprintf("https://emailvalidation.abstractapi.com/v1/?api_key=%s&email=%s", v.APIKey, email)
	resp, err := v.HTTPClient.Get(url)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	var result struct {
		Deliverability string `json:"deliverability"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, err
	}
	return result.Deliverability == "DELIVERABLE", nil
}

func (v *Validator) validateViaSMTP(email string) (bool, error) {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false, fmt.Errorf("invalid format")
	}
	domain := parts[1]

	// 1. Query domain MX DNS records
	mxRecords, err := net.LookupMX(domain)
	if err != nil || len(mxRecords) == 0 {
		return false, fmt.Errorf("no MX records")
	}
	mxHost := mxRecords[0].Host

	// 2. Establish direct SMTP connection
	addr := mxHost + ":25"
	conn, err := net.DialTimeout("tcp", addr, v.Timeout)
	if err != nil {
		return false, err
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, mxHost)
	if err != nil {
		return false, err
	}
	defer client.Quit()

	// 3. Issue HELO/EHLO with local hostname
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "localhost"
	}
	if err := client.Hello(hostname); err != nil {
		return false, err
	}

	// 4. Issue MAIL FROM handshake
	if err := client.Mail("test@example.com"); err != nil {
		return false, err
	}

	// 5. Validate mailbox existence via RCPT TO (status 250 = exists)
	if err := client.Rcpt(email); err != nil {
		// In Go smtp.Client, Rcpt returns error on non-250 response
		return false, nil
	}

	return true, nil
}

func GenerateEmail(shopName string) string {
	cleanName := strings.ReplaceAll(strings.ToLower(shopName), " ", "")
	return fmt.Sprintf("%s@gmail.com", cleanName)
}
