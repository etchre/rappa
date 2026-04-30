# Rappa

A Discord music bot written in Go. It uses DisGo for Discord interactions and Lavalink for audio playback.

## Setup

This project is intended to run with Docker Compose using [compose.example.yml](compose.example.yml).

Create a `.env` file next to `compose.example.yml`:

```env
DISCORD_BOT_TOKEN=your_discord_bot_token
LAVALINK_PASSWORD=youshallnotpass
SYNC_GLOBAL_COMMANDS=false
CLEAR_GLOBAL_COMMANDS=false
```

Set `SYNC_GLOBAL_COMMANDS=true` when you want the bot to register/update its global slash commands. After the commands are synced, you can set it back to `false` for normal runs.

## First Run

Build and start the bot plus Lavalink:

```bash
docker compose -f compose.example.yml up -d --build
```

View logs:

```bash
docker logs -f rappa-bot
docker logs -f rappa-lavalink
```

## Subsequent Runs

Start the existing containers:

```bash
docker compose -f compose.example.yml up -d
```

Stop the containers:

```bash
docker compose -f compose.example.yml down
```

Rebuild after code changes:

```bash
docker compose -f compose.example.yml up -d --build
```
