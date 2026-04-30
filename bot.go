package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgo/gateway"
	"github.com/disgoorg/disgolink/v3/disgolink"
	"github.com/disgoorg/disgolink/v3/lavalink"

	"ytdlpPlayer/commandrouter"
	"ytdlpPlayer/commands"
	"ytdlpPlayer/music"
)

type botConfig struct {
	token               string
	clearGlobalCommands bool
	syncGlobalCommands  bool
	lavalink            disgolink.NodeConfig
}

func runBot() error {
	ctx := context.Background()

	cfg, err := loadBotConfig()
	if err != nil {
		return err
	}

	app, err := newBotApp(ctx, cfg)
	if err != nil {
		return err
	}
	defer app.close(ctx)

	if err := app.connectLavalink(ctx); err != nil {
		return fmt.Errorf("connect to lavalink node: %w", err)
	}

	if err := app.registerCommands(); err != nil {
		return fmt.Errorf("sync global commands: %w", err)
	}

	if err := app.discord.OpenGateway(ctx); err != nil {
		return fmt.Errorf("open discord gateway: %w", err)
	}

	fmt.Println("Bot is ready. Use /play with a link. Press Ctrl+C to disconnect.")
	waitForShutdown()

	return nil
}

type botApp struct {
	config   botConfig
	discord  *bot.Client
	lavalink disgolink.Client
	player   *music.Player
	router   commandrouter.Router
}

func newBotApp(ctx context.Context, cfg botConfig) (*botApp, error) {
	app := &botApp{config: cfg}

	client, err := disgo.New(
		cfg.token,
		bot.WithGatewayConfigOpts(
			gateway.WithIntents(
				gateway.IntentGuilds,
				gateway.IntentGuildVoiceStates,
			),
		),
		bot.WithEventListenerFunc(app.onVoiceStateUpdate(ctx)),
		bot.WithEventListenerFunc(app.onVoiceServerUpdate(ctx)),
		bot.WithEventListenerFunc(app.onApplicationCommand(ctx)),
	)
	if err != nil {
		return nil, fmt.Errorf("create discord client: %w", err)
	}

	app.discord = client
	app.lavalink = disgolink.New(
		client.ApplicationID,
		disgolink.WithListenerFunc(func(disgolinkPlayer disgolink.Player, event lavalink.TrackEndEvent) {
			app.player.OnTrackEnd(disgolinkPlayer, event)
		}),
	)
	app.player = music.NewPlayer(app.lavalink)
	app.router = commandrouter.New(commandrouter.Context{
		Player: app.player,
	}, commands.All)

	return app, nil
}

func (app *botApp) connectLavalink(ctx context.Context) error {
	_, err := app.lavalink.AddNode(ctx, app.config.lavalink)
	return err
}

func (app *botApp) registerCommands() error {
	if app.config.clearGlobalCommands {
		if _, err := app.discord.Rest.SetGlobalCommands(app.discord.ApplicationID, nil); err != nil {
			return fmt.Errorf("clear global commands: %w", err)
		}
	}
	if !app.config.syncGlobalCommands {
		return nil
	}

	_, err := app.discord.Rest.SetGlobalCommands(app.discord.ApplicationID, app.router.Definitions())
	return err
}

func (app *botApp) close(ctx context.Context) {
	if app.discord != nil {
		app.discord.Close(ctx)
	}

	if app.lavalink != nil {
		app.lavalink.Close()
	}
}

func (app *botApp) onVoiceStateUpdate(ctx context.Context) func(event *events.GuildVoiceStateUpdate) {
	return func(event *events.GuildVoiceStateUpdate) {
		if event.VoiceState.UserID != app.lavalink.UserID() {
			return
		}

		app.lavalink.OnVoiceStateUpdate(ctx, event.VoiceState.GuildID, event.VoiceState.ChannelID, event.VoiceState.SessionID)
	}
}

func (app *botApp) onVoiceServerUpdate(ctx context.Context) func(event *events.VoiceServerUpdate) {
	return func(event *events.VoiceServerUpdate) {
		if event.Endpoint == nil {
			return
		}

		app.lavalink.OnVoiceServerUpdate(ctx, event.GuildID, event.Token, *event.Endpoint)
	}
}

func (app *botApp) onApplicationCommand(ctx context.Context) func(event *events.ApplicationCommandInteractionCreate) {
	return func(event *events.ApplicationCommandInteractionCreate) {
		app.router.Handle(ctx, event)
	}
}

func waitForShutdown() {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
}

func loadBotConfig() (botConfig, error) {
	token := os.Getenv("DISCORD_BOT_TOKEN")
	if token == "" {
		return botConfig{}, fmt.Errorf("DISCORD_BOT_TOKEN is required")
	}

	lavalinkSecure, err := envBool("LAVALINK_SECURE", false)
	if err != nil {
		return botConfig{}, err
	}

	clearGlobalCommands, err := envBool("CLEAR_GLOBAL_COMMANDS", false)
	if err != nil {
		return botConfig{}, err
	}

	syncGlobalCommands, err := envBool("SYNC_GLOBAL_COMMANDS", false)
	if err != nil {
		return botConfig{}, err
	}

	return botConfig{
		token:               token,
		clearGlobalCommands: clearGlobalCommands,
		syncGlobalCommands:  syncGlobalCommands,
		lavalink: disgolink.NodeConfig{
			Name:     envDefault("LAVALINK_NODE_NAME", "local"),
			Address:  envDefault("LAVALINK_ADDRESS", "localhost:2333"),
			Password: envDefault("LAVALINK_PASSWORD", "youshallnotpass"),
			Secure:   lavalinkSecure,
		},
	}, nil
}

func envDefault(name string, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}

	return fallback
}

func envBool(name string, fallback bool) (bool, error) {
	value := os.Getenv(name)
	if value == "" {
		return fallback, nil
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("parse %s: %w", name, err)
	}

	return parsed, nil
}
