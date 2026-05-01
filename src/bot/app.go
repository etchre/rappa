package bot

import (
	"context"
	"fmt"
	"os"

	"github.com/disgoorg/disgo"
	disgobot "github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/gateway"
	"github.com/disgoorg/disgolink/v3/disgolink"
	"github.com/disgoorg/disgolink/v3/lavalink"

	"ytdlpPlayer/commandrouter"
	"ytdlpPlayer/commands"
	"ytdlpPlayer/music"
)

type app struct {
	config   config
	discord  *disgobot.Client
	lavalink disgolink.Client
	player   *music.Player
	router   commandrouter.Router
}

func newApp(ctx context.Context, cfg config) (*app, error) {
	app := &app{config: cfg}

	client, err := disgo.New(
		cfg.token,
		disgobot.WithGatewayConfigOpts(
			gateway.WithIntents(
				gateway.IntentGuilds,
				gateway.IntentGuildVoiceStates,
			),
		),
		disgobot.WithEventListenerFunc(app.onVoiceStateUpdate(ctx)),
		disgobot.WithEventListenerFunc(app.onVoiceServerUpdate(ctx)),
		disgobot.WithEventListenerFunc(app.onApplicationCommand(ctx)),
		disgobot.WithEventListenerFunc(app.onComponentInteraction(ctx)),
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
		disgolink.WithListenerFunc(func(disgolinkPlayer disgolink.Player, event lavalink.TrackExceptionEvent) {
			app.player.OnTrackException(disgolinkPlayer, event)
		}),
	)
	app.player = music.NewPlayer(app.lavalink, cfg.preferredNodeName, cfg.premiumNodeName)
	app.router = commandrouter.New(commandrouter.Context{
		Player:              app.player,
		PremiumAllowedUsers: cfg.premiumAllowedUsers,
	}, commands.All)

	return app, nil
}

func (app *app) connectLavalink(ctx context.Context) error {
	for _, nodeConfig := range app.config.lavalinkNodes {
		if _, err := app.lavalink.AddNode(ctx, nodeConfig); err != nil {
			return fmt.Errorf("connect %s node: %w", nodeConfig.Name, err)
		}
	}

	if app.lavalink.Node(app.config.preferredNodeName) == nil {
		return fmt.Errorf("preferred lavalink node %q is not connected", app.config.preferredNodeName)
	}

	return nil
}

func (app *app) registerCommands() error {
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

func (app *app) warmDiscordRest() {
	if _, err := app.discord.Rest.GetGatewayBot(); err != nil {
		fmt.Fprintf(os.Stderr, "discord rest warmup failed: %v\n", err)
	}
}

func (app *app) close(ctx context.Context) {
	if app.discord != nil {
		app.discord.Close(ctx)
	}

	if app.lavalink != nil {
		app.lavalink.Close()
	}
}
