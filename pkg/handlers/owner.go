package handlers

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/MrAbhi2k3/TG-FileStore/pkg/bot"
	"github.com/MrAbhi2k3/TG-FileStore/pkg/utils"
)

func (h *BotHandler) handleStats(ctx context.Context, msg *bot.Message) error {
	totalUsers, _ := h.db.GetTotalUsers(ctx)
	fileStats, _ := h.db.GetFileStats(ctx)

	totalFiles := int64(0)
	totalSize := int64(0)
	totalDownloads := int64(0)
	if fileStats != nil {
		totalFiles = fileStats.TotalFiles
		totalSize = fileStats.TotalSize
		totalDownloads = fileStats.TotalDownloads
	}

	statsText := fmt.Sprintf(`📊 <b>Bot Statistics</b>

👤 <b>Total Users:</b> %d
📦 <b>Total Files:</b> %d
💾 <b>Stored Size:</b> %s
⬇️ <b>Total Downloads:</b> %d

🟢 <b>Bot Status:</b> Online
🔒 <b>Force Sub:</b> %v`,
		totalUsers,
		totalFiles,
		utils.FormatFileSize(totalSize),
		totalDownloads,
		h.cfg.ForceSubEnabled,
	)

	_, err := h.tg.SendMessage(ctx, msg.Chat.ID, statsText, nil)
	return err
}

func (h *BotHandler) handleUsers(ctx context.Context, msg *bot.Message) error {
	totalUsers, err := h.db.GetTotalUsers(ctx)
	if err != nil {
		_, err = h.tg.SendMessage(ctx, msg.Chat.ID, "❌ Failed to query users.", nil)
		return err
	}

	text := fmt.Sprintf("👥 <b>Registered Users:</b> %d\n\nUse <code>/broadcast</code> to communicate with all active users.", totalUsers)
	_, err = h.tg.SendMessage(ctx, msg.Chat.ID, text, nil)
	return err
}

func (h *BotHandler) handleFiles(ctx context.Context, msg *bot.Message) error {
	files, err := h.db.GetRecentFiles(ctx, 10)
	if err != nil {
		_, err = h.tg.SendMessage(ctx, msg.Chat.ID, "❌ Failed to retrieve recent files.", nil)
		return err
	}

	if len(files) == 0 {
		_, err := h.tg.SendMessage(ctx, msg.Chat.ID, "📦 No files stored yet.", nil)
		return err
	}

	var sb strings.Builder
	sb.WriteString("📦 <b>Recent Files (Latest 10):</b>\n\n")
	for i, f := range files {
		sb.WriteString(fmt.Sprintf("%d. <code>%s</code> (%s)\n   Token: <code>%s</code> | Downloads: %d\n",
			i+1,
			utils.EscapeHTML(f.FileName),
			utils.FormatFileSize(f.FileSize),
			f.Token,
			f.Downloads,
		))
	}

	_, err = h.tg.SendMessage(ctx, msg.Chat.ID, sb.String(), nil)
	return err
}

func (h *BotHandler) handleDBStats(ctx context.Context, msg *bot.Message) error {
	pingErr := h.db.Client.Ping(ctx, nil)
	status := "✅ Connected & Healthy"
	if pingErr != nil {
		status = fmt.Sprintf("❌ Error: %s", pingErr.Error())
	}

	text := fmt.Sprintf(`🗄 <b>Database Status</b>

<b>Database:</b> <code>%s</code>
<b>Connection:</b> %s`,
		h.cfg.DatabaseName,
		status,
	)

	_, err := h.tg.SendMessage(ctx, msg.Chat.ID, text, nil)
	return err
}

func (h *BotHandler) handleDeleteByHash(ctx context.Context, msg *bot.Message, args []string) error {
	if len(args) == 0 {
		_, err := h.tg.SendMessage(ctx, msg.Chat.ID, "Usage: <code>/delete &lt;hash&gt;</code>", nil)
		return err
	}

	deleted, err := h.db.DeleteFileByHash(ctx, args[0])
	if err != nil {
		_, err = h.tg.SendMessage(ctx, msg.Chat.ID, "❌ Error deleting file.", nil)
		return err
	}
	if deleted == nil {
		_, err = h.tg.SendMessage(ctx, msg.Chat.ID, "❌ File hash not found in database.", nil)
		return err
	}

	_ = h.tg.DeleteMessage(ctx, deleted.ChannelID, deleted.MessageID)

	reply := fmt.Sprintf("🗑 <b>File Deleted:</b>\nName: <code>%s</code>\nHash: <code>%s</code>", utils.EscapeHTML(deleted.FileName), deleted.Hash)
	_, err = h.tg.SendMessage(ctx, msg.Chat.ID, reply, nil)
	return err
}

func (h *BotHandler) handleDeleteByToken(ctx context.Context, msg *bot.Message, args []string) error {
	if len(args) == 0 {
		_, err := h.tg.SendMessage(ctx, msg.Chat.ID, "Usage: <code>/delete_token &lt;token&gt;</code>", nil)
		return err
	}

	deleted, err := h.db.DeleteFileByToken(ctx, args[0])
	if err != nil {
		_, err = h.tg.SendMessage(ctx, msg.Chat.ID, "❌ Error deleting file.", nil)
		return err
	}
	if deleted == nil {
		_, err = h.tg.SendMessage(ctx, msg.Chat.ID, "❌ File token not found in database.", nil)
		return err
	}

	_ = h.tg.DeleteMessage(ctx, deleted.ChannelID, deleted.MessageID)

	reply := fmt.Sprintf("🗑 <b>File Deleted:</b>\nName: <code>%s</code>\nToken: <code>%s</code>", utils.EscapeHTML(deleted.FileName), deleted.Token)
	_, err = h.tg.SendMessage(ctx, msg.Chat.ID, reply, nil)
	return err
}

func (h *BotHandler) handleSetForceSub(ctx context.Context, msg *bot.Message, args []string) error {
	if len(args) == 0 {
		status := "Enabled"
		if !h.cfg.ForceSubEnabled {
			status = "Disabled"
		}
		text := fmt.Sprintf("Force Subscription status: <b>%s</b>\nChannel: <code>%d</code>\n\nUsage: <code>/setforcesub on</code> or <code>/setforcesub off</code>", status, h.cfg.ForceSubChannelID)
		_, err := h.tg.SendMessage(ctx, msg.Chat.ID, text, nil)
		return err
	}

	val := strings.ToLower(args[0])
	if val == "on" || val == "true" || val == "enable" {
		h.cfg.ForceSubEnabled = true
		_, err := h.tg.SendMessage(ctx, msg.Chat.ID, "✅ Force Subscription has been <b>ENABLED</b>.", nil)
		return err
	} else if val == "off" || val == "false" || val == "disable" {
		h.cfg.ForceSubEnabled = false
		_, err := h.tg.SendMessage(ctx, msg.Chat.ID, "⚠️ Force Subscription has been <b>DISABLED</b>.", nil)
		return err
	}

	_, err := h.tg.SendMessage(ctx, msg.Chat.ID, "Usage: <code>/setforcesub on|off</code>", nil)
	return err
}

func (h *BotHandler) handleBroadcast(ctx context.Context, msg *bot.Message, asForward bool) error {
	targetMsg := msg.ReplyToMessage
	if targetMsg == nil {
		_, err := h.tg.SendMessage(ctx, msg.Chat.ID, "⚠️ Please reply to the message you wish to broadcast with <code>/broadcast</code> or <code>/broadcast_forward</code>.", nil)
		return err
	}

	userIDs, err := h.db.GetAllUserIDs(ctx)
	if err != nil {
		_, err = h.tg.SendMessage(ctx, msg.Chat.ID, "❌ Failed to fetch user IDs from database.", nil)
		return err
	}

	if len(userIDs) == 0 {
		_, err := h.tg.SendMessage(ctx, msg.Chat.ID, "⚠️ No registered users found to broadcast to.", nil)
		return err
	}

	startMsg, _ := h.tg.SendMessage(ctx, msg.Chat.ID, fmt.Sprintf("🚀 Starting broadcast to %d users...", len(userIDs)), nil)

	var (
		successful int64
		failed     int64
		blocked    int64
		semaphore  = make(chan struct{}, 5)
		wg         sync.WaitGroup
		mu         sync.Mutex
	)

	for _, uid := range userIDs {
		if uid == h.cfg.OwnerID {
			continue
		}

		wg.Add(1)
		semaphore <- struct{}{}

		go func(destID int64) {
			defer wg.Done()
			defer func() { <-semaphore }()

			var sendErr error
			if asForward {
				_, sendErr = h.tg.ForwardMessage(ctx, destID, msg.Chat.ID, targetMsg.MessageID)
			} else {
				_, sendErr = h.tg.CopyMessage(ctx, destID, msg.Chat.ID, targetMsg.MessageID)
			}

			mu.Lock()
			defer mu.Unlock()
			if sendErr != nil {
				failed++
				errStr := sendErr.Error()
				if strings.Contains(errStr, "403") || strings.Contains(errStr, "blocked") || strings.Contains(errStr, "deactivated") {
					blocked++
					_ = h.db.MarkUserBlocked(ctx, destID)
				}
			} else {
				successful++
			}
		}(uid)
	}

	wg.Wait()

	resultText := fmt.Sprintf(`📢 <b>Broadcast Completed!</b>

👥 <b>Total Targets:</b> %d
✅ <b>Delivered:</b> %d
❌ <b>Failed:</b> %d
🚫 <b>Blocked/Inactive:</b> %d`,
		len(userIDs),
		successful,
		failed,
		blocked,
	)

	if startMsg != nil {
		_ = h.tg.DeleteMessage(ctx, msg.Chat.ID, startMsg.MessageID)
	}
	_, err = h.tg.SendMessage(ctx, msg.Chat.ID, resultText, nil)
	return err
}
