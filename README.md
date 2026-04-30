# Rappa

A Discord music bot written in Go. It uses DisGo for Discord interactions and Lavalink for audio playback.

> [!NOTE]
> this project is vibed out the wazoo, use at your own risk

## Setup

This project is intended to run with Docker Compose using [compose.yml](compose.yml).

## Discord Bot Setup

Create a Discord application in the [Discord Developer Portal](https://discord.com/developers/applications), then create a bot user for that application.

Copy the bot token from the **Bot** page and put it in `.env` as `DISCORD_BOT_TOKEN`. Treat this token like a password:

- Do not commit `.env`.
- Do not paste the token in Discord, GitHub, screenshots, or logs.
- If the token leaks, reset it in the Developer Portal and update `.env`.

The bot uses slash commands and joins voice channels. When generating the invite URL, use these scopes:

```text
bot
applications.commands
```

Recommended bot permissions:

```text
View Channels
Send Messages
Embed Links
Use Slash Commands
Connect
Speak
```

If you want a quick development invite, you can use Discord's permissions calculator in the OAuth2 URL Generator. Avoid granting Administrator unless you are only testing in a private server.

Gateway intents used by the bot:

```text
Guilds
Guild Voice States
```

These are configured in code. You generally do not need privileged intents for this bot.

### Slash Commands

The bot can register global slash commands for you. Global commands may take a while to appear or update in Discord.

Use this in `.env` when you want to sync command definitions:

```env
SYNC_GLOBAL_COMMANDS=true
```

After commands are synced, set it back to:

```env
SYNC_GLOBAL_COMMANDS=false
```

If your Discord application has old global commands from a previous bot, you can clear them with:

```env
CLEAR_GLOBAL_COMMANDS=true
```

Only use that intentionally. After stale commands are cleared, set it back to `false`.

Create a `.env` file next to `compose.yml`:

```env
DISCORD_BOT_TOKEN=your_discord_bot_token
LAVALINK_PASSWORD=youshallnotpass
YOUTUBE_REFRESH_TOKEN=
SYNC_GLOBAL_COMMANDS=false
CLEAR_GLOBAL_COMMANDS=false
```

The default compose stack starts two Lavalink nodes:

- `lavalink`: normal logged-out playback.
- `lavalink-premium`: OAuth-enabled YouTube playback for future premium fallback support.

The bot does not use the premium node yet unless the Go code is updated to select it. Leave `YOUTUBE_REFRESH_TOKEN` empty until you intentionally test OAuth.

## First Run

Build and start the bot plus Lavalink:

```bash
docker compose up -d --build
```

View logs:

```bash
docker logs -f rappa-bot
docker logs -f rappa-lavalink
docker logs -f rappa-lavalink-premium
```

## Subsequent Runs

Start the existing containers:

```bash
docker compose up -d
```

Stop the containers:

```bash
docker compose down
```

Rebuild after code changes:

```bash
docker compose up -d --build
```

## License

MIT
