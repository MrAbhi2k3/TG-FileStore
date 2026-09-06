package handlers

import (
	"context"
	"fmt"
	"strings"

	"github.com/MrAbhi2k3/TG-FileStore/internal/bot"
	"github.com/MrAbhi2k3/TG-FileStore/internal/models"
	"github.com/MrAbhi2k3/TG-FileStore/internal/utils"
)

func (h *BotHandler) HandleCallbackQuery(ctx context.Context, cq *bot.CallbackQuery) error {
	data := cq.Data
	userID := cq.From.ID
	chatID := cq.From.ID
	if cq.Message != nil && cq.Message.Chat != nil {
		chatID = cq.Message.Chat.ID
	}

	if strings.HasPrefix(data, "check_sub") {
		isSub, err := h.CheckForceSubscription(ctx, userID)
		if err != nil || !isSub {
			_ = h.tg.AnswerCallbackQuery(ctx, cq.ID, "❌ You haven't joined yet. Please join the channel first!", true)
			return nil
		}

		_ = h.tg.AnswerCallbackQuery(ctx, cq.ID, "✅ Subscription confirmed!", false)

		parts := strings.SplitN(data, ":", 2)
		if len(parts) == 2 && parts[1] != "" {
			return h.deliverFile(ctx, chatID, parts[1])
		}

		_, _ = h.tg.SendMessage(ctx, chatID, "🎉 <b>Welcome!</b> You now have full access to the bot. Send any file or use /help to begin.", nil)
		return nil
	}

	if data == "batch_create" {
		items, err := h.db.GetBatchQueue(ctx, userID)
		_ = h.db.ClearBatch(ctx, userID)

		if err != nil || len(items) == 0 {
			_ = h.tg.AnswerCallbackQuery(ctx, cq.ID, "⚠️ No files in queue to create a batch.", true)
			return nil
		}

		_ = h.tg.AnswerCallbackQuery(ctx, cq.ID, "Creating batch link...", false)

		token, err := utils.GenerateToken(8)
		if err != nil {
			token = fmt.Sprintf("b%d", cq.Message.MessageID)
		}

		var messageIDs []int
		var totalSize int64
		for _, item := range items {
			messageIDs = append(messageIDs, item.MessageID)
			totalSize += item.FileSize
		}

		batchRecord := &models.FileRecord{
			Token:        token,
			ChannelID:    h.cfg.LogChannelID,
			FileName:     fmt.Sprintf("Batch_%s (%d files)", token, len(items)),
			FileSize:     totalSize,
			FileType:     models.FileTypeDocument,
			UploadedBy:   userID,
			IsBatch:      true,
			MessageIDs:   messageIDs,
		}

		_ = h.db.SaveFile(ctx, batchRecord)

		link := h.GetFileLink(ctx, token)

		h.SendLogNotification(ctx, cq.From, fmt.Sprintf("Batch (%d Files)", len(items)), batchRecord.FileName, totalSize, link)

		successMsg := fmt.Sprintf(`📦 <b>Batch Link Generated!</b>

<b>Total Files:</b> %d
<b>Total Size:</b> %s
🔗 <b>Permanent Batch Link:</b>
%s`,
			len(items),
			utils.FormatFileSize(totalSize),
			link,
		)

		kb := &bot.InlineKeyboardMarkup{
			InlineKeyboard: [][]bot.InlineKeyboardButton{
				{
					{Text: "📥 Open Link", URL: link},
				},
				{
					{Text: "🔗 Share Link", URL: fmt.Sprintf("https://t.me/share/url?url=%s", link)},
				},
			},
		}

		if cq.Message != nil {
			err = h.tg.EditMessageText(ctx, chatID, cq.Message.MessageID, successMsg, kb)
			if err == nil {
				return nil
			}
		}

		_, err = h.tg.SendMessage(ctx, chatID, successMsg, kb)
		return err
	}

	if data == "batch_cancel" {
		_ = h.db.ClearBatch(ctx, userID)
		_ = h.tg.AnswerCallbackQuery(ctx, cq.ID, "Batch session cancelled.", false)
		if cq.Message != nil {
			_ = h.tg.DeleteMessage(ctx, chatID, cq.Message.MessageID)
		}
		_, err := h.tg.SendMessage(ctx, chatID, "🗑️ <b>Batch session cancelled!</b> Queue cleared.", nil)
		return err
	}

	if data == "nav_help" {
		_ = h.tg.AnswerCallbackQuery(ctx, cq.ID, "", false)
		if cq.Message != nil {
			helpText := h.GetHelpText(h.IsOwner(cq.From.ID))
			kb := &bot.InlineKeyboardMarkup{
				InlineKeyboard: [][]bot.InlineKeyboardButton{
					{
						{Text: "ℹ️ About", CallbackData: "nav_about"},
						{Text: "🔙 Back", CallbackData: "nav_home"},
					},
				},
			}
			_ = h.tg.EditMessageText(ctx, cq.Message.Chat.ID, cq.Message.MessageID, helpText, kb)
		}
		return nil
	}

	if data == "nav_about" {
		_ = h.tg.AnswerCallbackQuery(ctx, cq.ID, "", false)
		if cq.Message != nil {
			aboutText := h.GetAboutText(ctx)
			kb := &bot.InlineKeyboardMarkup{
				InlineKeyboard: [][]bot.InlineKeyboardButton{
					{
						{Text: "❓ Help", CallbackData: "nav_help"},
						{Text: "🔙 Back", CallbackData: "nav_home"},
					},
				},
			}
			_ = h.tg.EditMessageText(ctx, cq.Message.Chat.ID, cq.Message.MessageID, aboutText, kb)
		}
		return nil
	}

	if data == "nav_home" {
		_ = h.tg.AnswerCallbackQuery(ctx, cq.ID, "", false)
		if cq.Message != nil {
			welcomeText := `👋 <b>Welcome to FileStore Bot!</b>

I can store files securely in Telegram and generate unique, permanent retrieval links.

📦 <b>How to use:</b>
1. Send or forward any Document, Video, Audio, or Photo to me.
2. Receive a fast, shareable deep link.
3. Access your file anytime using that link.`
			kb := h.GetStartKeyboard()
			_ = h.tg.EditMessageText(ctx, cq.Message.Chat.ID, cq.Message.MessageID, welcomeText, kb)
		}
		return nil
	}

	_ = h.tg.AnswerCallbackQuery(ctx, cq.ID, "", false)
	return nil
}
