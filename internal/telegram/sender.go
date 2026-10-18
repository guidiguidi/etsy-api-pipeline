package telegram

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"etsy/internal/config"
	"etsy/internal/model"

	"github.com/gotd/td/session"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/message"
	"github.com/gotd/td/tg"
)

type Sender struct {
	cfg    *config.TelegramConfig
	client *telegram.Client
	api    *tg.Client
	sender *message.Sender
	peer   tg.InputPeerClass
}

func NewSender(cfg *config.TelegramConfig) *Sender {
	return &Sender{cfg: cfg}
}

func (s *Sender) Start(ctx context.Context) error {
	if !s.cfg.Enabled {
		return nil
	}

	sessionStorage := &session.FileStorage{Path: s.cfg.SessionPath}
	s.client = telegram.NewClient(s.cfg.APIID, s.cfg.APIHash, telegram.Options{
		SessionStorage: sessionStorage,
	})

	ready := make(chan struct{})
	go func() {
		err := s.client.Run(ctx, func(ctx context.Context) error {
			s.api = tg.NewClient(s.client)
			s.sender = message.NewSender(s.api)
			
			p, err := s.sender.Resolve(s.cfg.BotUsername).AsInputPeer(ctx)
			if err != nil {
				return err
			}
			s.peer = p
			close(ready)
			<-ctx.Done()
			return nil
		})
		if err != nil && ctx.Err() == nil {
			log.Printf("Telegram client error: %v", err)
		}
	}()

	select {
	case <-ready:
		log.Println("✅ Telegram: Connected and ready")
		return nil
	case <-time.After(30 * time.Second):
		return fmt.Errorf("timeout waiting for telegram connection")
	}
}

// waitForButton ждет появления кнопки
func (s *Sender) waitForButton(ctx context.Context, buttonTextOrData string, timeout time.Duration) (int, []byte, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		history, err := s.api.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
			Peer:  s.peer,
			Limit: 3,
		})
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}

		var msgs []tg.MessageClass
		switch h := history.(type) {
		case *tg.MessagesMessagesSlice: msgs = h.Messages
		case *tg.MessagesMessages: msgs = h.Messages
		}

		for _, mClass := range msgs {
			if msg, ok := mClass.(*tg.Message); ok && msg.ReplyMarkup != nil {
				if markup, ok := msg.ReplyMarkup.(*tg.ReplyInlineMarkup); ok {
					for _, row := range markup.Rows {
						for _, btn := range row.Buttons {
							if b, ok := btn.(*tg.KeyboardButtonCallback); ok {
								if strings.Contains(string(b.Data), buttonTextOrData) || strings.Contains(b.Text, buttonTextOrData) {
									return msg.ID, b.Data, nil
								}
							}
						}
					}
				}
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return 0, nil, fmt.Errorf("button '%s' not found", buttonTextOrData)
}

// waitForText ждет появления определенного текста от бота
func (s *Sender) waitForText(ctx context.Context, expectedText string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		history, err := s.api.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
			Peer:  s.peer,
			Limit: 1,
		})
		if err == nil {
			var msgs []tg.MessageClass
			switch h := history.(type) {
			case *tg.MessagesMessagesSlice: msgs = h.Messages
			case *tg.MessagesMessages: msgs = h.Messages
			}
			if len(msgs) > 0 {
				if m, ok := msgs[0].(*tg.Message); ok {
					if strings.Contains(strings.ToLower(m.Message), strings.ToLower(expectedText)) {
						return nil
					}
				}
			}
		}
		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("timeout waiting for text: %s", expectedText)
}

func (s *Sender) click(ctx context.Context, button string) error {
	id, data, err := s.waitForButton(ctx, button, 10*time.Second)
	if err != nil {
		return err
	}
	log.Printf("🔘 Telegram: Clicking [%s]", button)
	_, err = s.api.MessagesGetBotCallbackAnswer(ctx, &tg.MessagesGetBotCallbackAnswerRequest{
		Peer:    s.peer,
		MsgID:   id,
		Data:    data,
	})
	if err != nil && strings.Contains(err.Error(), "BOT_RESPONSE_TIMEOUT") {
		log.Println("⚠️ Telegram: Bot timeout, ignoring...")
		return nil
	}
	return err
}

func (s *Sender) SendMatch(ctx context.Context, match *model.Match) error {
	if !s.cfg.Enabled || s.peer == nil {
		return nil
	}

	log.Printf("🤖 Telegram: Starting flow for %s", match.Shop.ShopName)

	log.Println("💬 Telegram: Sending /start")
	_, _ = s.sender.To(s.peer).Text(ctx, "/start")
	
	if err := s.click(ctx, "create_link"); err != nil { return err }
	if err := s.click(ctx, "create_link_eu"); err != nil { return err }
	if err := s.click(ctx, "create_link_service_etsy_eu"); err != nil { return err }

	_ = s.waitForText(ctx, "Введите название", 10*time.Second)
	title := strings.ReplaceAll(match.Listing.Title, "`", "")
	log.Printf("💬 Telegram: Sending Title: %s", title)
	_, _ = s.sender.To(s.peer).Text(ctx, title)

	_ = s.waitForText(ctx, "Введите цену", 10*time.Second)
	price := fmt.Sprintf("%.2f", match.Listing.Price.ToFloat())
	log.Printf("💬 Telegram: Sending Price: %s", price)
	_, _ = s.sender.To(s.peer).Text(ctx, price)

	_ = s.waitForText(ctx, "изображение", 10*time.Second)
	img := match.Listing.ImageURL
	if img == "" { img = "https://via.placeholder.com/500" }
	log.Printf("💬 Telegram: Sending Image URL: %s", img)
	_, _ = s.sender.To(s.peer).Text(ctx, img)
	
	if err := s.click(ctx, "Daily Mail"); err != nil { return err }

	_ = s.waitForText(ctx, "Введите почту", 10*time.Second)
	if match.Shop.EmailValid {
		log.Printf("💬 Telegram: Sending Email: %s", match.Shop.Email)
		_, _ = s.sender.To(s.peer).Text(ctx, match.Shop.Email)
		log.Printf("🏁 Telegram: SUCCESS for %s", match.Shop.ShopName)
	}
	return nil
}

func (s *Sender) SendFromJSONL(ctx context.Context, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var match model.Match
		if err := json.Unmarshal(scanner.Bytes(), &match); err != nil {
			continue
		}
		log.Printf("📤 Telegram: Resending match from JSONL: %s", match.Shop.ShopName)
		_ = s.SendMatch(ctx, &match)
		time.Sleep(10 * time.Second)
	}
	return scanner.Err()
}
