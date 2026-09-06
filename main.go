package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/MrAbhi2k3/TG-FileStore/internal/bot"
	"github.com/MrAbhi2k3/TG-FileStore/internal/database"
	"github.com/MrAbhi2k3/TG-FileStore/internal/handlers"
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
				_ = service.ProcessUpdate(processCtx, &upd)
			}(u)
		}
	}
}
