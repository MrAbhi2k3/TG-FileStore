package handlers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/MrAbhi2k3/TG-FileStore/pkg/bot"
	"github.com/MrAbhi2k3/TG-FileStore/pkg/utils"
)

func (h *BotHandler) HandleCommand(ctx context.Context, msg *bot.Message) error {
	text := strings.TrimSpace(msg.Text)
	parts := strings.Fields(text)
	if len(parts) == 0 {
		return nil
	}

	cmd := strings.ToLower(parts[0])
	if atIndex := strings.Index(cmd, "@"); atIndex != -1 {
		cmd = cmd[:atIndex]
	}

	args := parts[1:]

	switch cmd {
	case "/start":
		return h.handleStart(ctx, msg, args)
	case "/help":
		return h.handleHelp(ctx, msg)
	case "/about":
		return h.handleAbout(ctx, msg)
	case "/id":
		return h.handleID(ctx, msg)
	case "/ping":
		return h.handlePing(ctx, msg)
	case "/batch":
		return h.handleBatch(ctx, msg)

	case "/stats":
		if !h.IsOwner(msg.From.ID) {
			return nil
		}
		return h.handleStats(ctx, msg)
	case "/users":
		if !h.IsOwner(msg.From.ID) {
			return nil
		}
		return h.handleUsers(ctx, msg)
	case "/files":
		if !h.IsOwner(msg.From.ID) {
			return nil
		}
		return h.handleFiles(ctx, msg)
	case "/dbstats":
		if !h.IsOwner(msg.From.ID) {
			return nil
		}
		return h.handleDBStats(ctx, msg)
	case "/delete":
		if !h.IsOwner(msg.From.ID) {
			return nil
		}
		return h.handleDeleteByHash(ctx, msg, args)
	case "/delete_token":
		if !h.IsOwner(msg.From.ID) {
			return nil
		}
		return h.handleDeleteByToken(ctx, msg, args)
	case "/broadcast":
		if !h.IsOwner(msg.From.ID) {
			return nil
		}
		return h.handleBroadcast(ctx, msg, false)
	case "/broadcast_forward":
		if !h.IsOwner(msg.From.ID) {
			return nil
		}
		return h.handleBroadcast(ctx, msg, true)
	case "/setforcesub":
		if !h.IsOwner(msg.From.ID) {
			return nil
		}
		return h.handleSetForceSub(ctx, msg, args)

	default:
		return nil
	}
}

func (h *BotHandler) handleStart(ctx context.Context, msg *bot.Message, args []string) error {
	chatID := msg.Chat.ID
	userID := msg.From.ID

	if len(args) > 0 && strings.TrimSpace(args[0]) != "" {
		payload := strings.TrimSpace(args[0])

		isSub, err := h.CheckForceSubscription(ctx, userID)
		if err != nil || !isSub {
			return h.SendForceSubPrompt(ctx, chatID, payload)
		}

		return h.deliverFile(ctx, chatID, payload)
	}

	welcomeText := `👋 <b>Welcome to FileStore Bot!</b>

I can store files securely in Telegram and generate unique, permanent retrieval links.

📦 <b>How to use:</b>
1. Send or forward any Document, Video, Audio, or Photo to me.
2. Receive a fast, shareable deep link.
3. Access your file anytime using that link.`

	kb := h.GetStartKeyboard()
	_, err := h.tg.SendMessage(ctx, chatID, welcomeText, kb)
	return err
}

func (h *BotHandler) deliverFile(ctx context.Context, chatID int64, query string) error {
	record, err := h.db.FindFileByToken(ctx, query)
	if err != nil {
		_, _ = h.tg.SendMessage(ctx, chatID, "❌ An error occurred while retrieving the file.", nil)
		return err
	}
	if record == nil {
		record, err = h.db.FindFileByHash(ctx, query)
		if err != nil {
			_, _ = h.tg.SendMessage(ctx, chatID, "❌ An error occurred while retrieving the file.", nil)
			return err
		}
	}

	if record == nil {
		msg := "❌ <b>File not found.</b>\n\nThe link may be invalid, expired, or deleted by the administrator."
		_, _ = h.tg.SendMessage(ctx, chatID, msg, nil)
		return nil
	}

	if record.IsBatch && len(record.MessageIDs) > 0 {
		prepMsg := fmt.Sprintf("📦 <b>Batch Found!</b>\n\nTotal Files: <b>%d</b>\n\n<i>Sending files...</i>", len(record.MessageIDs))
		prepSent, _ := h.tg.SendMessage(ctx, chatID, prepMsg, nil)

		for _, mid := range record.MessageIDs {
			deliveredMsgID, copyErr := h.tg.CopyMessage(ctx, chatID, record.ChannelID, mid)
			if copyErr == nil && h.cfg.AutoDeleteSeconds > 0 && deliveredMsgID > 0 {
				go func(tChatID int64, fMsgID int, delaySec int) {
					time.Sleep(time.Duration(delaySec) * time.Second)
					delCtx, delCancel := context.WithTimeout(context.Background(), 10*time.Second)
					defer delCancel()
					_ = h.tg.DeleteMessage(delCtx, tChatID, fMsgID)
				}(chatID, deliveredMsgID, h.cfg.AutoDeleteSeconds)
			}
			time.Sleep(300 * time.Millisecond)
		}

		if prepSent != nil {
			_ = h.tg.DeleteMessage(ctx, chatID, prepSent.MessageID)
		}
		_ = h.db.IncrementDownloads(ctx, record.Token)
		return nil
	}

	prepMsg := fmt.Sprintf("📦 <b>File Found!</b>\n\n<b>Name:</b> <code>%s</code>\n<b>Size:</b> %s\n\n<i>Sending your file...</i>",
		utils.EscapeHTML(record.FileName),
		utils.FormatFileSize(record.FileSize),
	)
	prepSent, _ := h.tg.SendMessage(ctx, chatID, prepMsg, nil)

	deliveredMsgID, copyErr := h.tg.CopyMessage(ctx, chatID, record.ChannelID, record.MessageID)

	if prepSent != nil {
		_ = h.tg.DeleteMessage(ctx, chatID, prepSent.MessageID)
	}

	if copyErr != nil {
		_, _ = h.tg.SendMessage(ctx, chatID, "❌ Failed to copy file from storage. The original post may have been removed.", nil)
		return copyErr
	}

	if h.cfg.AutoDeleteSeconds > 0 && deliveredMsgID > 0 {
		warningText := fmt.Sprintf("⏳ <i>This file will be automatically deleted in %d seconds to save storage. Please save/forward it now.</i>", h.cfg.AutoDeleteSeconds)
		warningMsg, _ := h.tg.SendMessage(ctx, chatID, warningText, nil)

		go func(targetChatID int64, fileMsgID int, warnMsgID int, delaySec int) {
			time.Sleep(time.Duration(delaySec) * time.Second)
			delCtx, delCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer delCancel()
			_ = h.tg.DeleteMessage(delCtx, targetChatID, fileMsgID)
			if warnMsgID > 0 {
				_ = h.tg.DeleteMessage(delCtx, targetChatID, warnMsgID)
			}
		}(chatID, deliveredMsgID, func() int {
			if warningMsg != nil {
				return warningMsg.MessageID
			}
			return 0
		}(), h.cfg.AutoDeleteSeconds)
	}

	_ = h.db.IncrementDownloads(ctx, record.Token)
	return nil
}

func (h *BotHandler) handleHelp(ctx context.Context, msg *bot.Message) error {
	helpText := h.GetHelpText(h.IsOwner(msg.From.ID))
	kb := &bot.InlineKeyboardMarkup{
		InlineKeyboard: [][]bot.InlineKeyboardButton{
			{
				{Text: "ℹ️ About", CallbackData: "nav_about"},
				{Text: "🔙 Back", CallbackData: "nav_home"},
			},
		},
	}
	_, err := h.tg.SendMessage(ctx, msg.Chat.ID, helpText, kb)
	return err
}

func (h *BotHandler) handleAbout(ctx context.Context, msg *bot.Message) error {
	aboutText := h.GetAboutText(ctx)
	kb := &bot.InlineKeyboardMarkup{
		InlineKeyboard: [][]bot.InlineKeyboardButton{
			{
				{Text: "❓ Help", CallbackData: "nav_help"},
				{Text: "🔙 Back", CallbackData: "nav_home"},
			},
		},
	}
	_, err := h.tg.SendMessage(ctx, msg.Chat.ID, aboutText, kb)
	return err
}

func (h *BotHandler) handleID(ctx context.Context, msg *bot.Message) error {
	text := fmt.Sprintf("🆔 <b>User ID:</b> <code>%d</code>\n💬 <b>Chat ID:</b> <code>%d</code>", msg.From.ID, msg.Chat.ID)
	_, err := h.tg.SendMessage(ctx, msg.Chat.ID, text, nil)
	return err
}

func (h *BotHandler) handlePing(ctx context.Context, msg *bot.Message) error {
	_, err := h.tg.SendMessage(ctx, msg.Chat.ID, "🏓 <b>Pong!</b> Bot is online and healthy.", nil)
	return err
}

func (h *BotHandler) handleBatch(ctx context.Context, msg *bot.Message) error {
	if !h.cfg.PublicUse && !h.IsOwner(msg.From.ID) {
		_, err := h.tg.SendMessage(ctx, msg.Chat.ID, "⛔ File storing is restricted to the bot owner.", nil)
		return err
	}

	_ = h.db.StartBatchSession(ctx, msg.From.ID)
	text := `📦 <b>Batch Mode Activated!</b>

Send or forward all the files you want to include in this batch.
After each file, you can tap:
• <b>📦 Create Batch Link</b> - Finish and generate a single link
• <b>❌ Cancel Queue</b> - Discard all files in the batch`

	_, err := h.tg.SendMessage(ctx, msg.Chat.ID, text, nil)
	return err
}
