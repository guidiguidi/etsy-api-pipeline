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

	// 1. Получаем MX записи (как в твоем скрипте через dns.resolver)
	mxRecords, err := net.LookupMX(domain)
	if err != nil || len(mxRecords) == 0 {
		return false, fmt.Errorf("no MX records")
	}
	mxHost := mxRecords[0].Host

	// 2. Подключаемся (как в твоем smtplib.SMTP)
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

	// 3. Представляемся именем нашего хоста (как server.local_hostname)
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "localhost"
	}
	if err := client.Hello(hostname); err != nil {
		return false, err
	}

	// 4. Указываем MAIL FROM (как в твоем скрипте)
	if err := client.Mail("test@example.com"); err != nil {
		return false, err
	}

	// 5. Проверяем адрес через RCPT TO (код 250 = успех)
	if err := client.Rcpt(email); err != nil {
		// В Go метод Rcpt возвращает ошибку, если код не 250
		return false, nil
	}

	return true, nil
}

func GenerateEmail(shopName string) string {
	cleanName := strings.ReplaceAll(strings.ToLower(shopName), " ", "")
	return fmt.Sprintf("%s@gmail.com", cleanName)
}
