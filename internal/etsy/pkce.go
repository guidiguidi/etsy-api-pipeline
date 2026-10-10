package etsy

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"net/url"
	"strings"
)

func GeneratePKCE() (string, string, error) {
	data := make([]byte, 32)
	if _, err := rand.Read(data); err != nil {
		return "", "", err
	}
	verifier := base64.RawURLEncoding.EncodeToString(data)

	hash := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(hash[:])

	return verifier, challenge, nil
}

func BuildAuthURL(clientID, redirectURI, state, challenge string, scopes []string) string {
	u, _ := url.Parse("https://www.etsy.com/oauth/connect")
	q := u.Query()
	q.Set("response_type", "code")
	q.Set("redirect_uri", redirectURI)
	q.Set("scope", strings.Join(scopes, "+"))
	q.Set("client_id", clientID)
	q.Set("state", state)
	q.Set("code_challenge", challenge)
	q.Set("code_challenge_method", "S256")
	u.RawQuery = q.Encode()
	// Etsy ожидает плюсы в scopes, а Encode() делает %2B. Поправим:
	return strings.ReplaceAll(u.String(), "%2B", "+")
}
