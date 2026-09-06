package handlers

import (
	"context"
	"fmt"

	"github.com/MrAbhi2k3/TG-FileStore/internal/bot"
	"github.com/MrAbhi2k3/TG-FileStore/internal/models"
	"github.com/MrAbhi2k3/TG-FileStore/internal/utils"
)

type ExtractedMedia struct {
	FileID       string
	FileUniqueID string
	FileName     string
	FileSize     int64
	MimeType     string
	FileType     models.FileType
}

func ExtractMediaFromMessage(msg *bot.Message) *ExtractedMedia {
	if msg.Document != nil {
		name := msg.Document.FileName
		if name == "" {
			name = "document"
		}
		return &ExtractedMedia{
			FileID:       msg.Document.FileID,
			FileUniqueID: msg.Document.FileUniqueID,
			FileName:     name,
			FileSize:     msg.Document.FileSize,
			MimeType:     msg.Document.MimeType,
			FileType:     models.FileTypeDocument,
		}
	}

	if msg.Video != nil {
		name := msg.Video.FileName
		if name == "" {
			name = "video.mp4"
		}
		return &ExtractedMedia{
			FileID:       msg.Video.FileID,
			FileUniqueID: msg.Video.FileUniqueID,
			FileName:     name,
			FileSize:     msg.Video.FileSize,
			MimeType:     msg.Video.MimeType,
			FileType:     models.FileTypeVideo,
		}
	}

	if msg.Audio != nil {
		name := msg.Audio.FileName
		if name == "" {
			if msg.Audio.Title != "" {
				name = msg.Audio.Title + ".mp3"
			} else {
				name = "audio.mp3"
			}
		}
		return &ExtractedMedia{
			FileID:       msg.Audio.FileID,
			FileUniqueID: msg.Audio.FileUniqueID,
			FileName:     name,
			FileSize:     msg.Audio.FileSize,
			MimeType:     msg.Audio.MimeType,
			FileType:     models.FileTypeAudio,
		}
	}

	if len(msg.Photo) > 0 {
		largest := msg.Photo[len(msg.Photo)-1]
		return &ExtractedMedia{
			FileID:       largest.FileID,
			FileUniqueID: largest.FileUniqueID,
			FileName:     fmt.Sprintf("photo_%s.jpg", largest.FileUniqueID),
			FileSize:     largest.FileSize,
			MimeType:     "image/jpeg",
			FileType:     models.FileTypePhoto,
		}
	}

	if msg.Animation != nil {
		name := msg.Animation.FileName
		if name == "" {
			name = "animation.gif"
		}
		return &ExtractedMedia{
			FileID:       msg.Animation.FileID,
			FileUniqueID: msg.Animation.FileUniqueID,
			FileName:     name,
			FileSize:     msg.Animation.FileSize,
			MimeType:     msg.Animation.MimeType,
			FileType:     models.FileTypeAnimation,
		}
	}

	if msg.Voice != nil {
		return &ExtractedMedia{
			FileID:       msg.Voice.FileID,
			FileUniqueID: msg.Voice.FileUniqueID,
			FileName:     "voice.ogg",
			FileSize:     msg.Voice.FileSize,
			MimeType:     msg.Voice.MimeType,
			FileType:     models.FileTypeVoice,
		}
	}

	if msg.Sticker != nil {
		return &ExtractedMedia{
			FileID:       msg.Sticker.FileID,
			FileUniqueID: msg.Sticker.FileUniqueID,
			FileName:     "sticker.webp",
			FileSize:     msg.Sticker.FileSize,
			MimeType:     "image/webp",
			FileType:     models.FileTypeSticker,
		}
	}

	return nil
}

func (h *BotHandler) HandleFile(ctx context.Context, msg *bot.Message) error {
	chatID := msg.Chat.ID
	userID := msg.From.ID

	if !h.cfg.PublicUse && !h.IsOwner(userID) {
		_, err := h.tg.SendMessage(ctx, chatID, "⛔ File storing is restricted to the bot owner.", nil)
		return err
	}

	isSub, err := h.CheckForceSubscription(ctx, userID)
	if err != nil || !isSub {
		return h.SendForceSubPrompt(ctx, chatID, "")
	}

	media := ExtractMediaFromMessage(msg)
	if media == nil {
		return nil
	}

	var fileHash string
	const maxDownloadSize = 20 * 1024 * 1024

	if media.FileSize > 0 && media.FileSize <= maxDownloadSize {
		tgFile, err := h.tg.GetFile(ctx, media.FileID)
		if err == nil && tgFile != nil && tgFile.FilePath != "" {
			computedHash, hashErr := h.tg.ComputeFileSHA256(ctx, tgFile.FilePath)
			if hashErr == nil && computedHash != "" {
				fileHash = computedHash
			}
		}
	}

	if fileHash == "" {
		fileHash = utils.ComputeIdentityHash(media.FileUniqueID, media.FileSize, media.FileName)
	}

	existing, err := h.db.FindFileByHash(ctx, fileHash)
	if err != nil {
		_, _ = h.tg.SendMessage(ctx, chatID, "❌ Database error during duplicate check. Please try again.", nil)
		return err
	}

	var record *models.FileRecord
	if existing != nil {
		record = existing
	} else {
		storageMsgID, copyErr := h.tg.CopyMessage(ctx, h.cfg.LogChannelID, chatID, msg.MessageID)
		if copyErr != nil {
			errMsg := fmt.Sprintf("❌ <b>Upload Failed</b>\n\nCould not copy file to storage channel. Ensure bot is an administrator in the storage channel (ID: <code>%d</code>).", h.cfg.LogChannelID)
			_, _ = h.tg.SendMessage(ctx, chatID, errMsg, nil)
			return copyErr
		}

		token, err := utils.GenerateToken(8)
		if err != nil {
			token = fmt.Sprintf("tok%d", msg.MessageID)
		}

		record = &models.FileRecord{
			Hash:         fileHash,
			Token:        token,
			MessageID:    storageMsgID,
			ChannelID:    h.cfg.LogChannelID,
			FileName:     media.FileName,
			FileSize:     media.FileSize,
			MimeType:     media.MimeType,
			FileType:     media.FileType,
			FileID:       media.FileID,
			UniqueFileID: media.FileUniqueID,
			UploadedBy:   userID,
		}

		if err := h.db.SaveFile(ctx, record); err != nil {
			_, _ = h.tg.SendMessage(ctx, chatID, "❌ Failed to save file metadata to database.", nil)
			return err
		}
	}

	link := h.GetFileLink(ctx, record.Token)

	if h.db.IsInBatchSession(ctx, userID) {
		_ = h.db.AddToBatchQueue(ctx, &models.BatchQueueItem{
			UserID:    userID,
			MessageID: record.MessageID,
			FileName:  record.FileName,
			FileSize:  record.FileSize,
		})
		count := h.db.GetBatchQueueCount(ctx, userID)

		statusText := fmt.Sprintf(`📦 <b>File Added to Batch!</b>

<b>Name:</b> <code>%s</code>
<b>Size:</b> %s
<b>Files in Batch Queue:</b> %d`,
			utils.EscapeHTML(record.FileName),
			utils.FormatFileSize(record.FileSize),
			count,
		)

		kb := &bot.InlineKeyboardMarkup{
			InlineKeyboard: [][]bot.InlineKeyboardButton{
				{
					{Text: fmt.Sprintf("📦 Create Batch Link (%d)", count), CallbackData: "batch_create"},
				},
				{
					{Text: "❌ Cancel Batch", CallbackData: "batch_cancel"},
				},
			},
		}

		_, err = h.tg.SendMessage(ctx, chatID, statusText, kb)
		return err
	}

	h.SendLogNotification(ctx, msg.From, "Single File", record.FileName, record.FileSize, link)

	statusText := fmt.Sprintf(`📦 <b>File Stored!</b>

<b>Name:</b> <code>%s</code>
<b>Size:</b> %s
🔗 <b>Permanent Link:</b>
%s`,
		utils.EscapeHTML(record.FileName),
		utils.FormatFileSize(record.FileSize),
		link,
	)

	kb := &bot.InlineKeyboardMarkup{
		InlineKeyboard: [][]bot.InlineKeyboardButton{
			{
				{Text: "📥 Open Link", URL: link},
				{Text: "🔗 Share Link", URL: fmt.Sprintf("https://t.me/share/url?url=%s", link)},
			},
		},
	}

	_, err = h.tg.SendMessage(ctx, chatID, statusText, kb)
	return err
}
