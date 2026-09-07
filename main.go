package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/MrAbhi2k3/TG-FileStore/pkg/bot"
	"github.com/MrAbhi2k3/TG-FileStore/pkg/database"
	"github.com/MrAbhi2k3/TG-FileStore/pkg/handlers"
)

func main() {
	cfg, err := bot.LoadConfig()
	if err != nil {
		log.Fatalf("Config error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	dbCtx, dbCancel := context.WithTimeout(ctx, 15*time.Second)
	defer dbCancel()

	db, err := database.GetMongoDB(dbCtx, cfg.MongoURI, cfg.DatabaseName)
	if err != nil {
		log.Fatalf("MongoDB error: %v", err)
	}

	tg := bot.NewTelegramClient(cfg.BotToken)
	botUser, err := tg.GetMe(ctx)
	if err != nil {
		log.Fatalf("Telegram error: %v", err)
	}

	bHandler := handlers.NewBotHandler(cfg, tg, db)
	service := bot.NewService(cfg, tg, db, bHandler)

	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/f/", func(w http.ResponseWriter, r *http.Request) {
			token := strings.TrimPrefix(r.URL.Path, "/f/")
			token = strings.TrimSpace(token)
			if token != "" {
				redirectURL := fmt.Sprintf("https://t.me/%s?start=%s", botUser.Username, token)
				http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
				return
			}
			http.NotFound(w, r)
		})
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok","message":"Telegram File Store Bot is running"}`))
		})
		port := os.Getenv("PORT")
		if port == "" {
			port = "3000"
		}
		log.Printf("Local HTTP server listening on port %s", port)
		server := &http.Server{
			Addr:    ":" + port,
			Handler: mux,
		}
		go func() {
			<-ctx.Done()
			_ = server.Close()
		}()
		_ = server.ListenAndServe()
	}()

	_ = tg.DeleteWebhook(ctx, false)

	log.Printf("Bot online: @%s", botUser.Username)
	fmt.Println("👉 Bot Started Yahooooo")

	var offset int64 = 0
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		pollCtx, pollCancel := context.WithTimeout(ctx, 30*time.Second)
		updates, err := tg.GetUpdates(pollCtx, offset, 100, 20)
		pollCancel()

		if err != nil {
			if ctx.Err() != nil {
				return
			}
			time.Sleep(3 * time.Second)
			continue
		}

		for _, update := range updates {
			if update.UpdateID >= offset {
				offset = update.UpdateID + 1
			}

			u := update
			go func(upd bot.Update) {
				processCtx, procCancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer procCancel()
				if err := service.ProcessUpdate(processCtx, &upd); err != nil {
					log.Printf("[ERROR] ProcessUpdate failed: %v", err)
				}
			}(u)
		}
	}
}
