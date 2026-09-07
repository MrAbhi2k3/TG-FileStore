package bot

import (
	"context"
	"strings"

	"github.com/MrAbhi2k3/TG-FileStore/pkg/database"
)

type BotHandlerInterface interface {
	TrackUser(ctx context.Context, u *User)
	HandleCommand(ctx context.Context, msg *Message) error
	HandleFile(ctx context.Context, msg *Message) error
	HandleCallbackQuery(ctx context.Context, cq *CallbackQuery) error
}

type Service struct {
	cfg     *Config
	tg      *TelegramClient
	db      *database.MongoDB
	handler BotHandlerInterface
}

func NewService(cfg *Config, tg *TelegramClient, db *database.MongoDB, handler BotHandlerInterface) *Service {
	return &Service{
		cfg:     cfg,
		tg:      tg,
		db:      db,
		handler: handler,
	}
}

func (s *Service) Config() *Config {
	return s.cfg
}

func (s *Service) ProcessUpdate(ctx context.Context, update *Update) error {
	if update == nil {
		return nil
	}

	if update.CallbackQuery != nil {
		s.handler.TrackUser(ctx, update.CallbackQuery.From)
		return s.handler.HandleCallbackQuery(ctx, update.CallbackQuery)
	}

	msg := update.Message
	if msg == nil {
		msg = update.EditedMessage
	}
	if msg == nil {
		return nil
	}

	if msg.From != nil {
		s.handler.TrackUser(ctx, msg.From)
	}

	if strings.HasPrefix(strings.TrimSpace(msg.Text), "/") {
		return s.handler.HandleCommand(ctx, msg)
	}

	return s.handler.HandleFile(ctx, msg)
}
