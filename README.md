# 🔅 FileStore Bot 🔅

A high-performance Telegram File Store Bot written in Go using only the official Telegram Bot API and MongoDB. Designed for serverless deployment on Vercel or running directly as a standalone binary.

---

## Deploy to Vercel

[![Deploy with Vercel](https://vercel.com/button)](https://vercel.com/new/clone?repository-url=https%3A%2F%2Fgithub.com%2FMrAbhi2k3%2FTG-FileStore&env=BOT_TOKEN,MONGODB_URI,LOG_CHANNEL_ID,OWNER_ID&envDescription=Required%20Telegram%20bot%20and%20MongoDB%20credentials&project-name=tg-filestore&repository-name=tg-filestore)

### Step-by-Step Deployment:
1. Click the **Deploy with Vercel** button above or import your repository on [vercel.com](https://vercel.com).
2. Fill in the Environment Variables:
   - `BOT_TOKEN`: Your Telegram Bot API token from [@BotFather](https://t.me/BotFather)
   - `MONGODB_URI`: Your MongoDB connection string (e.g. MongoDB Atlas)
   - `LOG_CHANNEL_ID`: Private Telegram channel ID for file storage (e.g. `-1001692309008`)
   - `OWNER_ID`: Your numeric Telegram ID
3. Deploy!
4. Register your webhook once with Telegram (replace `<BOT_TOKEN>` and `<VERCEL_URL>`):
   ```bash
   curl -F "url=https://<YOUR-VERCEL-DOMAIN>/api/webhook" https://api.telegram.org/bot<BOT_TOKEN>/setWebhook
   ```

---

## Features

- Uses Telegram as cloud storage using `copyMessage`
- MongoDB for file metadata and user records
- Webhook redirect URL protection (`/f/:token`)
- Automatic deployment URL detection on Vercel and Cloudflare
- Dynamic in-place inline button menus (About & Help)
- Public / Private upload mode (`PUBLIC_USE=true/false`)
- Owner administrative controls (`/stats`, `/broadcast`, `/delete`, etc.)
- Channel Force Subscription with auto-bypass if unconfigured
- Configurable automatic file deletion timer (`AUTO_DELETE_SECONDS`)
- Long-polling mode for local development with zero setup

---

## Directory Structure

```text
.
├── .env
├── .env.example
├── .gitignore
├── LICENSE
├── README.md
├── go.mod
├── go.sum
├── main.go               # Local runner
├── vercel.json           # Vercel serverless routing
├── api/
│   └── webhook.go        # Vercel HTTP serverless handler + /f/:token redirect
└── pkg/
    ├── bot/              # Telegram API client, bot types & config
    ├── database/         # MongoDB connection & queries
    ├── handlers/         # Message, file, command & callback handlers
    ├── models/           # MongoDB document models
    └── utils/            # Hashing, token generator & formatters
```

---

## Environment Variables (`.env`)

```env
BOT_TOKEN=1521299940:AAEWLOAE1iyiEKBQG41mOMWB-hXEE3XNS6I
MONGODB_URI=mongodb://localhost:27017/
MONGODB_DATABASE=file_store
LOG_CHANNEL_ID=-1001692309008
OWNER_ID=1287407305

# true = anyone can store files; false = owner only
PUBLIC_USE=true

# Force Subscription (set to false or leave channel ID empty to disable)
FORCE_SUB_ENABLED=false
FORCE_SUB_CHANNEL_ID=

# Auto-delete files delivered to users (seconds; 0 to disable)
AUTO_DELETE_SECONDS=0

# Optional: Auto-detected by Vercel if empty
WEBHOOK_URL=
WEBHOOK_SECRET=
```

---

## Local Run

```bash
go run main.go
```

---

## Permanent Webhook File Links

The webhook endpoint protects the bot link:

```text
https://your-bot.vercel.app/f/<TOKEN>
```

When opened in any browser, it redirects to the Telegram bot:

```text
https://t.me/<BOT_USERNAME>?start=<TOKEN>
```

---

## Commands

### User Commands:
- `/start` — Start bot or retrieve file
- `/batch` — Enter batch mode to group multiple files into one link
- `/help` — Help menu
- `/about` — Bot specifications
- `/id` — User & Chat ID
- `/ping` — Latency check

### Owner Commands:
- `/stats` — Bot & file stats
- `/users` — Registered user count
- `/files` — Recent files
- `/dbstats` — Database ping
- `/delete <hash>` — Delete by hash
- `/delete_token <token>` — Delete by token
- `/broadcast` — Broadcast replied message to all users
- `/broadcast_forward` — Forward replied message to all users
- `/setforcesub <on/off>` — Toggle force subscription

---

## License
MIT License. Developed by [MrAbhi2k3](https://github.com/MrAbhi2k3).
