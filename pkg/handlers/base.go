package handlers

import (
	"context"
	"fmt"
	"strings"

	"github.com/MrAbhi2k3/TG-FileStore/pkg/bot"
	"github.com/MrAbhi2k3/TG-FileStore/pkg/database"
	"github.com/MrAbhi2k3/TG-FileStore/pkg/models"
	"github.com/MrAbhi2k3/TG-FileStore/pkg/utils"
)

type BotHandler struct {
	cfg *bot.Config
	tg  *bot.TelegramClient
	db  *database.MongoDB
}

func NewBotHandler(cfg *bot.Config, tg *bot.TelegramClient, db *database.MongoDB) *BotHandler {
	return &BotHandler{
		cfg: cfg,
		tg:  tg,
		db:  db,
	}
}

func (h *BotHandler) GetFileLink(ctx context.Context, token string) string {
	if h.cfg.WebhookURL != "" && !strings.Contains(h.cfg.WebhookURL, "localhost") && !strings.Contains(h.cfg.WebhookURL, "127.0.0.1") {
		base := strings.TrimSuffix(h.cfg.WebhookURL, "/")
		return fmt.Sprintf("%s/f/%s", base, token)
	}
	botUser, _ := h.tg.GetMe(ctx)
	if botUser != nil && botUser.Username != "" {
		return fmt.Sprintf("https://t.me/%s?start=%s", botUser.Username, token)
	}
	return "https://t.me/share/url?url=" + token
}

func (h *BotHandler) GetButtonLink(ctx context.Context, token string) string {
	if h.cfg.WebhookURL != "" && strings.HasPrefix(h.cfg.WebhookURL, "https://") && !strings.Contains(h.cfg.WebhookURL, "localhost") && !strings.Contains(h.cfg.WebhookURL, "127.0.0.1") {
		base := strings.TrimSuffix(h.cfg.WebhookURL, "/")
		return fmt.Sprintf("%s/f/%s", base, token)
	}
	botUser, _ := h.tg.GetMe(ctx)
	if botUser != nil && botUser.Username != "" {
		return fmt.Sprintf("https://t.me/%s?start=%s", botUser.Username, token)
	}
	return "https://t.me/share/url?url=" + token
}

func (h *BotHandler) SendLogNotification(ctx context.Context, u *bot.User, title string, fileName string, size int64, link string) {
	if h.cfg.LogChannelID == 0 {
		return
	}
	userName := "Unknown"
	userID := int64(0)
	if u != nil {
		userID = u.ID
		if u.Username != "" {
			userName = "@" + u.Username
		} else {
			userName = u.FirstName
		}
	}
	logText := fmt.Sprintf(`📢 <b>#NewFileStored</b>

<b>Type:</b> %s
<b>User:</b> %s (<code>%d</code>)
<b>File Name:</b> <code>%s</code>
<b>Size:</b> %s
🔗 <b>Link:</b> %s`,
		title,
		utils.EscapeHTML(userName),
		userID,
		utils.EscapeHTML(fileName),
		utils.FormatFileSize(size),
		link,
	)
	_, _ = h.tg.SendMessage(ctx, h.cfg.LogChannelID, logText, nil)
}

func (h *BotHandler) IsOwner(userID int64) bool {
	return userID == h.cfg.OwnerID
}

func (h *BotHandler) GetStartKeyboard() *bot.InlineKeyboardMarkup {
	return &bot.InlineKeyboardMarkup{
		InlineKeyboard: [][]bot.InlineKeyboardButton{
			{
				{Text: "ℹ️ About", CallbackData: "nav_about"},
				{Text: "❓ Help", CallbackData: "nav_help"},
			},
		},
	}
}

func (h *BotHandler) GetAboutText(ctx context.Context) string {
	botUser, _ := h.tg.GetMe(ctx)
	botUsername := "FileStoreBot"
	if botUser != nil && botUser.Username != "" {
		botUsername = botUser.Username
	}

	return fmt.Sprintf(`╭────[ <b>🔅FɪʟᴇSᴛᴏʀᴇBᴏᴛ🔅</b>]────⍟
│
├🔸 <b>My Name:</b> <a href="https://t.me/%s">%s</a>
│
├🔸 <b>Language:</b> <a href="https://go.dev">Go (Golang 1.22+)</a>
│
├🔹 <b>Library:</b> <a href="https://core.telegram.org/bots/api">Telegram Bot API</a>
│
├🔹 <b>Hosted On:</b> <a href="https://vercel.com">Vercel</a>
│
├🔸 <b>Developer:</b> <a href="https://github.com/mrabhi2k3">MrAbhi2k3</a>
│
├🔹 <b>Bot Support:</b> <a href="https://t.me/%s">Support</a>
│
├🔸 <b>Bot Updates:</b> <a href="https://github.com/MrAbhi2k3/TG-FileStore">Updates</a>
│
╰──────[ 😎 ]───────────⍟`,
		botUsername,
		botUsername,
		botUsername,
	)
}

func (h *BotHandler) GetHelpText(isOwner bool) string {
	helpText := `📖 <b>Available Commands</b>

<b>Commands:</b>
• /start - Start the bot or open the main menu
• /id - View your Telegram User and Chat ID
• /ping - Check bot response and latency
• /about - Learn about this bot
• /help - Show this help menu

<b>📦 How to Store Files:</b>
Send or forward any document, video, audio, or photo directly to this chat.`

	if isOwner {
		helpText += `

👑 <b>Owner Commands:</b>
• /stats - Global bot and storage statistics
• /users - Active users overview
• /files - List recent stored files
• /dbstats - Database status
• /delete &lt;hash&gt; - Delete a file by hash
• /delete_token &lt;token&gt; - Delete a file by token
• /broadcast - Broadcast replied message to all users
• /broadcast_forward - Broadcast as forwarded message
• /setforcesub &lt;on/off&gt; - Toggle Force Subscription`
	}

	return helpText
}

func (h *BotHandler) TrackUser(ctx context.Context, u *bot.User) {
	if u == nil {
		return
	}
	record := &models.UserRecord{
		UserID:    u.ID,
		Username:  u.Username,
		FirstName: u.FirstName,
		LastName:  u.LastName,
	}
	_ = h.db.TrackUser(ctx, record)
}

func (h *BotHandler) CheckForceSubscription(ctx context.Context, userID int64) (bool, error) {
	if !h.cfg.ForceSubEnabled || h.cfg.ForceSubChannelID == 0 {
		return true, nil
	}

	if h.IsOwner(userID) {
		return true, nil
	}

	member, err := h.tg.GetChatMember(ctx, h.cfg.ForceSubChannelID, userID)
	if err != nil {
		return false, err
	}

	switch member.Status {
	case "creator", "administrator", "member", "restricted":
		return true, nil
	default:
		return false, nil
	}
}

func (h *BotHandler) SendForceSubPrompt(ctx context.Context, chatID int64, retryPayload string) error {
	var inviteLink string
	if h.cfg.ForceSubChannelID != 0 {
		inviteLink = fmt.Sprintf("https://t.me/c/%d", -h.cfg.ForceSubChannelID)
	}

	var kbRows [][]bot.InlineKeyboardButton
	if inviteLink != "" {
		kbRows = append(kbRows, []bot.InlineKeyboardButton{
			{Text: "📢 Join Channel", URL: inviteLink},
		})
	}

	cbData := "check_sub"
	if retryPayload != "" {
		cbData = "check_sub:" + retryPayload
	}

	kbRows = append(kbRows, []bot.InlineKeyboardButton{
		{Text: "✅ I've Joined", CallbackData: cbData},
	})

	markup := &bot.InlineKeyboardMarkup{InlineKeyboard: kbRows}
	text := "🔒 <b>Access Denied</b>\n\nYou need to join our updates channel first to retrieve files.\n\nAfter joining, press <b>✅ I've Joined</b> below."

	_, err := h.tg.SendMessage(ctx, chatID, text, markup)
	return err
}
