package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/MrAbhi2k3/TG-FileStore/pkg/bot"
	"github.com/MrAbhi2k3/TG-FileStore/pkg/database"
	"github.com/MrAbhi2k3/TG-FileStore/pkg/handlers"
)

var (
	botService *bot.Service
	botInitErr error
	initOnce   sync.Once
)

func initBot() (*bot.Service, error) {
	initOnce.Do(func() {
		cfg, err := bot.LoadConfig()
		if err != nil {
			botInitErr = err
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		db, err := database.GetMongoDB(ctx, cfg.MongoURI, cfg.DatabaseName)
		if err != nil {
			botInitErr = err
			return
		}

		tgClient := bot.NewTelegramClient(cfg.BotToken)
		bHandler := handlers.NewBotHandler(cfg, tgClient, db)
		botService = bot.NewService(cfg, tgClient, db, bHandler)
	})

	return botService, botInitErr
}

func Handler(w http.ResponseWriter, r *http.Request) {
	service, err := initBot()
	if err != nil {
		log.Printf("[ERROR] Service initialization failed: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if service.Config().WebhookURL == "" && r.Host != "" {
		scheme := "https"
		if r.TLS == nil && !strings.Contains(r.Host, "vercel.app") && !strings.Contains(r.Host, "pages.dev") {
			scheme = "http"
		}
		service.Config().WebhookURL = scheme + "://" + r.Host
	}

	if strings.HasPrefix(r.URL.Path, "/f/") {
		token := strings.TrimPrefix(r.URL.Path, "/f/")
		token = strings.TrimSpace(token)
		if token != "" {
			ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
			defer cancel()

			client := bot.NewTelegramClient(service.Config().BotToken)
			user, err := client.GetMe(ctx)
			if err == nil && user != nil && user.Username != "" {
				redirectURL := fmt.Sprintf("https://t.me/%s?start=%s", user.Username, token)
				http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
				return
			}
		}
	}

	if r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","message":"Telegram File Store Bot is alive"}`))
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if secret := service.Config().WebhookSecret; secret != "" {
		if r.Header.Get("X-Telegram-Bot-Api-Secret-Token") != secret {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	var update bot.Update
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
	defer cancel()

	_ = service.ProcessUpdate(ctx, &update)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"ok":true}`))
}
