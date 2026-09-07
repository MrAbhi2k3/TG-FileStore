package bot

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

const (
	defaultTelegramAPIBase  = "https://api.telegram.org/bot"
	defaultTelegramFileBase = "https://api.telegram.org/file/bot"
)

type TelegramClient struct {
	token      string
	apiBase    string
	fileBase   string
	httpClient *http.Client
	botUser    *User
	botMu      sync.RWMutex
}

func NewTelegramClient(token string) *TelegramClient {
	return &TelegramClient{
		token:    token,
		apiBase:  defaultTelegramAPIBase + token,
		fileBase: defaultTelegramFileBase + token,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (c *TelegramClient) GetMe(ctx context.Context) (*User, error) {
	c.botMu.RLock()
	if c.botUser != nil {
		defer c.botMu.RUnlock()
		return c.botUser, nil
	}
	c.botMu.RUnlock()

	var resp APIResponse[User]
	if err := c.postJSON(ctx, "/getMe", nil, &resp); err != nil {
		return nil, err
	}
	if !resp.OK {
		return nil, fmt.Errorf("telegram error (%d): %s", resp.ErrorCode, resp.Description)
	}

	c.botMu.Lock()
	c.botUser = &resp.Result
	c.botMu.Unlock()

	return &resp.Result, nil
}

func (c *TelegramClient) SendMessage(ctx context.Context, chatID int64, text string, replyMarkup *InlineKeyboardMarkup) (*Message, error) {
	payload := map[string]any{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "HTML",
		"link_preview_options": map[string]any{
			"is_disabled": true,
		},
	}
	if replyMarkup != nil {
		payload["reply_markup"] = replyMarkup
	}

	var resp APIResponse[Message]
	if err := c.postJSON(ctx, "/sendMessage", payload, &resp); err != nil {
		return nil, err
	}
	if !resp.OK {
		return nil, fmt.Errorf("telegram error (%d): %s", resp.ErrorCode, resp.Description)
	}
	return &resp.Result, nil
}

func (c *TelegramClient) CopyMessage(ctx context.Context, chatID int64, fromChatID int64, messageID int) (int, error) {
	payload := map[string]any{
		"chat_id":      chatID,
		"from_chat_id": fromChatID,
		"message_id":   messageID,
	}

	var resp APIResponse[MessageIDResult]
	if err := c.postJSON(ctx, "/copyMessage", payload, &resp); err != nil {
		return 0, err
	}
	if !resp.OK {
		return 0, fmt.Errorf("telegram error (%d): %s", resp.ErrorCode, resp.Description)
	}
	return resp.Result.MessageID, nil
}

func (c *TelegramClient) ForwardMessage(ctx context.Context, chatID int64, fromChatID int64, messageID int) (*Message, error) {
	payload := map[string]any{
		"chat_id":      chatID,
		"from_chat_id": fromChatID,
		"message_id":   messageID,
	}

	var resp APIResponse[Message]
	if err := c.postJSON(ctx, "/forwardMessage", payload, &resp); err != nil {
		return nil, err
	}
	if !resp.OK {
		return nil, fmt.Errorf("telegram error (%d): %s", resp.ErrorCode, resp.Description)
	}
	return &resp.Result, nil
}

func (c *TelegramClient) DeleteMessage(ctx context.Context, chatID int64, messageID int) error {
	payload := map[string]any{
		"chat_id":    chatID,
		"message_id": messageID,
	}
	var resp APIResponse[bool]
	if err := c.postJSON(ctx, "/deleteMessage", payload, &resp); err != nil {
		return err
	}
	if !resp.OK {
		return fmt.Errorf("telegram error (%d): %s", resp.ErrorCode, resp.Description)
	}
	return nil
}

func (c *TelegramClient) EditMessageText(ctx context.Context, chatID int64, messageID int, text string, replyMarkup *InlineKeyboardMarkup) error {
	payload := map[string]any{
		"chat_id":    chatID,
		"message_id": messageID,
		"text":       text,
		"parse_mode": "HTML",
		"link_preview_options": map[string]any{
			"is_disabled": true,
		},
	}
	if replyMarkup != nil {
		payload["reply_markup"] = replyMarkup
	}

	var resp APIResponse[Message]
	if err := c.postJSON(ctx, "/editMessageText", payload, &resp); err != nil {
		return err
	}
	if !resp.OK {
		return fmt.Errorf("telegram error (%d): %s", resp.ErrorCode, resp.Description)
	}
	return nil
}

func (c *TelegramClient) AnswerCallbackQuery(ctx context.Context, callbackQueryID string, text string, showAlert bool) error {
	payload := map[string]any{
		"callback_query_id": callbackQueryID,
		"show_alert":        showAlert,
	}
	if text != "" {
		payload["text"] = text
	}

	var resp APIResponse[bool]
	if err := c.postJSON(ctx, "/answerCallbackQuery", payload, &resp); err != nil {
		return err
	}
	if !resp.OK {
		return fmt.Errorf("telegram error (%d): %s", resp.ErrorCode, resp.Description)
	}
	return nil
}

func (c *TelegramClient) GetChatMember(ctx context.Context, chatID int64, userID int64) (*ChatMember, error) {
	payload := map[string]any{
		"chat_id": chatID,
		"user_id": userID,
	}
	var resp APIResponse[ChatMember]
	if err := c.postJSON(ctx, "/getChatMember", payload, &resp); err != nil {
		return nil, err
	}
	if !resp.OK {
		return nil, fmt.Errorf("telegram error (%d): %s", resp.ErrorCode, resp.Description)
	}
	return &resp.Result, nil
}

func (c *TelegramClient) GetFile(ctx context.Context, fileID string) (*TelegramFile, error) {
	payload := map[string]any{
		"file_id": fileID,
	}
	var resp APIResponse[TelegramFile]
	if err := c.postJSON(ctx, "/getFile", payload, &resp); err != nil {
		return nil, err
	}
	if !resp.OK {
		return nil, fmt.Errorf("telegram error (%d): %s", resp.ErrorCode, resp.Description)
	}
	return &resp.Result, nil
}

func (c *TelegramClient) ComputeFileSHA256(ctx context.Context, filePath string) (string, error) {
	if filePath == "" {
		return "", errors.New("empty file path")
	}

	url := fmt.Sprintf("%s/%s", c.fileBase, filePath)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download file from telegram: HTTP %d", resp.StatusCode)
	}

	hasher := sha256.New()
	if _, err := io.Copy(hasher, resp.Body); err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func (c *TelegramClient) SetWebhook(ctx context.Context, url string, secretToken string) error {
	payload := map[string]any{
		"url":             url,
		"allowed_updates": []string{"message", "edited_message", "callback_query", "channel_post"},
	}
	if secretToken != "" {
		payload["secret_token"] = secretToken
	}

	var resp APIResponse[bool]
	if err := c.postJSON(ctx, "/setWebhook", payload, &resp); err != nil {
		return err
	}
	if !resp.OK {
		return fmt.Errorf("telegram error (%d): %s", resp.ErrorCode, resp.Description)
	}
	return nil
}

func (c *TelegramClient) DeleteWebhook(ctx context.Context, dropPendingUpdates bool) error {
	payload := map[string]any{
		"drop_pending_updates": dropPendingUpdates,
	}
	var resp APIResponse[bool]
	if err := c.postJSON(ctx, "/deleteWebhook", payload, &resp); err != nil {
		return err
	}
	if !resp.OK {
		return fmt.Errorf("telegram error (%d): %s", resp.ErrorCode, resp.Description)
	}
	return nil
}

func (c *TelegramClient) GetUpdates(ctx context.Context, offset int64, limit int, timeoutSec int) ([]Update, error) {
	payload := map[string]any{
		"offset":          offset,
		"limit":           limit,
		"timeout":         timeoutSec,
		"allowed_updates": []string{"message", "edited_message", "callback_query"},
	}
	var resp APIResponse[[]Update]
	if err := c.postJSON(ctx, "/getUpdates", payload, &resp); err != nil {
		return nil, err
	}
	if !resp.OK {
		return nil, fmt.Errorf("telegram error (%d): %s", resp.ErrorCode, resp.Description)
	}
	return resp.Result, nil
}

func (c *TelegramClient) postJSON(ctx context.Context, endpoint string, body any, out any) error {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(data)
	}

	url := c.apiBase + endpoint
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bodyReader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("[TELEGRAM API ERROR] %s returned status %d: %s", endpoint, resp.StatusCode, string(respBody))
	}

	return json.Unmarshal(respBody, out)
}
