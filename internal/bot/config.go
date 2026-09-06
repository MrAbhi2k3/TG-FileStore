package bot

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	BotToken          string
	MongoURI          string
	DatabaseName      string
	LogChannelID      int64
	OwnerID           int64
	ForceSubEnabled   bool
	ForceSubChannelID int64
	WebhookURL        string
	WebhookSecret     string
	AutoDeleteSeconds int
	PublicUse         bool
}

func LoadConfig() (*Config, error) {
	loadDotEnv(".env")

	botToken := strings.TrimSpace(os.Getenv("BOT_TOKEN"))
	if botToken == "" {
		return nil, errors.New("BOT_TOKEN is required")
	}

	mongoURI := strings.TrimSpace(os.Getenv("MONGODB_URI"))
	if mongoURI == "" {
		return nil, errors.New("MONGODB_URI is required")
	}

	dbName := strings.TrimSpace(os.Getenv("MONGODB_DATABASE"))
	if dbName == "" {
		dbName = "file_store"
	}

	logChanStr := strings.TrimSpace(os.Getenv("LOG_CHANNEL_ID"))
	if logChanStr == "" {
		return nil, errors.New("LOG_CHANNEL_ID is required")
	}
	logChannelID, err := strconv.ParseInt(logChanStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid LOG_CHANNEL_ID: %w", err)
	}

	ownerStr := strings.TrimSpace(os.Getenv("OWNER_ID"))
	if ownerStr == "" {
		return nil, errors.New("OWNER_ID is required")
	}
	ownerID, err := strconv.ParseInt(ownerStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid OWNER_ID: %w", err)
	}

	forceSubEnabled := true
	if fseStr := strings.TrimSpace(os.Getenv("FORCE_SUB_ENABLED")); fseStr != "" {
		forceSubEnabled = strings.ToLower(fseStr) == "true" || fseStr == "1"
	}

	var forceSubChannelID int64
	if forceSubChanStr := strings.TrimSpace(os.Getenv("FORCE_SUB_CHANNEL_ID")); forceSubChanStr != "" {
		parsed, err := strconv.ParseInt(forceSubChanStr, 10, 64)
		if err == nil {
			forceSubChannelID = parsed
		}
	}

	if forceSubChannelID == 0 {
		forceSubEnabled = false
	}

	webhookURL := strings.TrimSpace(os.Getenv("WEBHOOK_URL"))
	if webhookURL == "" {
		if prodURL := strings.TrimSpace(os.Getenv("VERCEL_PROJECT_PRODUCTION_URL")); prodURL != "" {
			webhookURL = "https://" + prodURL
		} else if vURL := strings.TrimSpace(os.Getenv("VERCEL_URL")); vURL != "" {
			webhookURL = "https://" + vURL
		} else if cfURL := strings.TrimSpace(os.Getenv("CF_PAGES_URL")); cfURL != "" {
			webhookURL = cfURL
		}
	}
	if webhookURL != "" && !strings.HasPrefix(webhookURL, "http://") && !strings.HasPrefix(webhookURL, "https://") {
		webhookURL = "https://" + webhookURL
	}
	webhookURL = strings.TrimSuffix(webhookURL, "/")

	webhookSecret := strings.TrimSpace(os.Getenv("WEBHOOK_SECRET"))

	autoDeleteSec := 0
	if adsStr := strings.TrimSpace(os.Getenv("AUTO_DELETE_SECONDS")); adsStr != "" {
		if val, err := strconv.Atoi(adsStr); err == nil {
			autoDeleteSec = val
		}
	} else if admStr := strings.TrimSpace(os.Getenv("AUTO_DELETE_MINUTES")); admStr != "" {
		if val, err := strconv.Atoi(admStr); err == nil {
			autoDeleteSec = val * 60
		}
	}

	publicUse := true
	if puStr := strings.TrimSpace(os.Getenv("PUBLIC_USE")); puStr != "" {
		publicUse = strings.ToLower(puStr) == "true" || puStr == "1"
	}

	return &Config{
		BotToken:          botToken,
		MongoURI:          mongoURI,
		DatabaseName:      dbName,
		LogChannelID:      logChannelID,
		OwnerID:           ownerID,
		ForceSubEnabled:   forceSubEnabled,
		ForceSubChannelID: forceSubChannelID,
		WebhookURL:        webhookURL,
		WebhookSecret:     webhookSecret,
		AutoDeleteSeconds: autoDeleteSec,
		PublicUse:         publicUse,
	}, nil
}

func loadDotEnv(filepath string) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		parts := strings.SplitN(trimmed, "=", 2)
		if len(parts) == 2 {
			k := strings.TrimSpace(parts[0])
			v := strings.TrimSpace(parts[1])
			v = strings.Trim(v, `"'`)
			if os.Getenv(k) == "" {
				_ = os.Setenv(k, v)
			}
		}
	}
}
