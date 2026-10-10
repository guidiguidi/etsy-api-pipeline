package etsy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"etsy/internal/config"
	"etsy/internal/model"

	"golang.org/x/time/rate"
)

const baseURL = "https://api.etsy.com/v3"

type Client struct {
	apiKey       string // Формат "keystring:shared_secret"
	accessToken  string // Формат "access_token"
	refreshToken string
	httpClient   *http.Client
	limiter      *rate.Limiter
}

func NewClient(apiKey, accessToken, refreshToken string, timeout time.Duration) *Client {
	return &Client{
		apiKey:       apiKey,
		accessToken:  accessToken,
		refreshToken: refreshToken,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		limiter: rate.NewLimiter(rate.Limit(5), 5),
	}
}

// ExchangeCodeForToken обменивает временный code на access и refresh токены
func (c *Client) ExchangeCodeForToken(code, verifier, redirectURI string) (string, string, error) {
	clientID := strings.Split(c.apiKey, ":")[0]

	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("client_id", clientID)
	data.Set("redirect_uri", redirectURI)
	data.Set("code", code)
	data.Set("code_verifier", verifier)

	resp, err := http.PostForm(baseURL+"/public/oauth/token", data)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("exchange failed (%d): %s", resp.StatusCode, string(body))
	}

	var res struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}

	if err := json.Unmarshal(body, &res); err != nil {
		return "", "", err
	}

	// Обновляем состояние клиента чистым токеном
	c.accessToken = res.AccessToken
	c.refreshToken = res.RefreshToken

	return res.AccessToken, res.RefreshToken, nil
}

// RefreshAccessToken обновляет протухший токен
func (c *Client) RefreshAccessToken() error {
	clientID := strings.Split(c.apiKey, ":")[0]

	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("client_id", clientID)
	data.Set("refresh_token", c.refreshToken)

	resp, err := http.PostForm(baseURL+"/public/oauth/token", data)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("refresh failed: %s", string(body))
	}

	var res struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return err
	}

	c.accessToken = res.AccessToken
	if res.RefreshToken != "" {
		c.refreshToken = res.RefreshToken
	}

	return config.SaveTokens(c.accessToken, c.refreshToken)
}

// doRequest вешает нужные заголовки и делает рефреш при 401
func (c *Client) doRequest(req *http.Request) (*http.Response, error) {
	if err := c.limiter.Wait(context.Background()); err != nil {
		return nil, err
	}

	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("User-Agent", "EtsyScraper/1.0")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	// Дебаг логирование
	if strings.Contains(c.apiKey, "secret") { // Простая проверка на наличие конфига
		// Мы не можем легко получить доступ к Config отсюда без изменения структуры,
		// но мы можем выводить статус ответа для наглядности
		if resp.StatusCode >= 400 {
			body, _ := io.ReadAll(resp.Body)
			resp.Body = io.NopCloser(bytes.NewBuffer(body)) // Возвращаем тело назад
			fmt.Printf("🔍 [DEBUG] API Error %d: %s\n", resp.StatusCode, string(body))
		}
	}

	if resp.StatusCode == http.StatusUnauthorized && c.refreshToken != "" {
		resp.Body.Close()
		fmt.Println("🔄 Access token expired, refreshing...")
		if err := c.RefreshAccessToken(); err != nil {
			return nil, err
		}
		// Повтор с новым токеном
		req.Header.Set("Authorization", "Bearer "+c.accessToken)
		return c.httpClient.Do(req)
	}

	return resp, nil
}

func (c *Client) GetActiveListings(limit, offset int) ([]model.Listing, error) {
	// Добавляем sort_on=created, sort_order=desc и includes=Images
	endpoint := fmt.Sprintf("%s/application/listings/active?limit=%d&offset=%d&sort_on=created&sort_order=desc&includes=Images", baseURL, limit, offset)
	req, _ := http.NewRequest("GET", endpoint, nil)

	resp, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("api error (%d): %s", resp.StatusCode, string(body))
	}

	var lResp struct {
		Results []model.Listing `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&lResp); err != nil {
		return nil, err
	}

	// Извлекаем URL первой картинки
	for i := range lResp.Results {
		if len(lResp.Results[i].Images) > 0 {
			lResp.Results[i].ImageURL = lResp.Results[i].Images[0].URLFull
		}
	}

	return lResp.Results, nil
}

func (c *Client) GetShop(shopID int64) (*model.Shop, error) {
	endpoint := fmt.Sprintf("%s/application/shops/%d", baseURL, shopID)
	req, _ := http.NewRequest("GET", endpoint, nil)

	resp, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("api error (%d): %s", resp.StatusCode, string(body))
	}

	var shop model.Shop
	if err := json.NewDecoder(resp.Body).Decode(&shop); err != nil {
		return nil, err
	}

	return &shop, nil
}
