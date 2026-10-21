package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"etsy/internal/config"

	"github.com/gotd/td/session"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
)

// terminalAuth реализует интерфейс auth.UserAuthenticator
type terminalAuth struct {
	scan *bufio.Scanner
}

func (a terminalAuth) Phone(ctx context.Context) (string, error) {
	fmt.Print("Введите ваш номер телефона (например, +79991234567): ")
	a.scan.Scan()
	return strings.TrimSpace(a.scan.Text()), nil
}

func (a terminalAuth) Password(ctx context.Context) (string, error) {
	fmt.Print("Введите пароль (2FA), если он есть: ")
	a.scan.Scan()
	return strings.TrimSpace(a.scan.Text()), nil
}

func (a terminalAuth) Code(ctx context.Context, sentCode *tg.AuthSentCode) (string, error) {
	fmt.Print("Введите код из Telegram: ")
	a.scan.Scan()
	return strings.TrimSpace(a.scan.Text()), nil
}

func (a terminalAuth) AcceptTermsOfService(ctx context.Context, tos tg.HelpTermsOfService) error {
	return nil
}

func (a terminalAuth) SignUp(ctx context.Context) (auth.UserInfo, error) {
	return auth.UserInfo{}, fmt.Errorf("регистрация новых пользователей не поддерживается")
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	fmt.Println("--- Telegram Auth Helper ---")

	sessionStorage := &session.FileStorage{
		Path: cfg.Telegram.SessionPath,
	}

	client := telegram.NewClient(cfg.Telegram.APIID, cfg.Telegram.APIHash, telegram.Options{
		SessionStorage: sessionStorage,
	})

	authenticator := terminalAuth{
		scan: bufio.NewScanner(os.Stdin),
	}

	ctx := context.Background()
	if err := client.Run(ctx, func(ctx context.Context) error {
		if err := client.Auth().IfNecessary(ctx, auth.NewFlow(authenticator, auth.SendCodeOptions{})); err != nil {
			return err
		}

		fmt.Println("✅ Авторизация успешна! Сессия сохранена в:", cfg.Telegram.SessionPath)
		return nil
	}); err != nil {
		log.Fatalf("Ошибка: %v", err)
	}
}
