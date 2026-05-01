package bot

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/disgoorg/disgo"
	disgobot "github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/gateway"
	"github.com/disgoorg/disgolink/v3/disgolink"
	"github.com/disgoorg/disgolink/v3/lavalink"
	"github.com/disgoorg/snowflake/v2"

	"ytdlpPlayer/commandrouter"
	"ytdlpPlayer/commands"
	"ytdlpPlayer/commands/helpers"
	"ytdlpPlayer/music"
)

type app struct {
	config   config
	discord  *disgobot.Client
	lavalink disgolink.Client
	player   *music.Player
	router   commandrouter.Router
	channels *commandrouter.StatusChannels
	idle     *idleDisconnects
	voiceMu  sync.Mutex
	voice    map[snowflake.ID]snowflake.ID
}

func newApp(ctx context.Context, cfg config) (*app, error) {
	app := &app{
		config:   cfg,
		channels: commandrouter.NewStatusChannels(),
		voice:    map[snowflake.ID]snowflake.ID{},
	}

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
		disgobot.WithEventListenerFunc(app.onAutocomplete(ctx)),
		disgobot.WithEventListenerFunc(app.onComponentInteraction(ctx)),
		disgobot.WithEventListenerFunc(app.onGuildsReady(ctx)),
		disgobot.WithEventListenerFunc(app.onGuildReady(ctx)),
	)
	if err != nil {
		return nil, fmt.Errorf("create discord client: %w", err)
	}

	app.discord = client
	app.idle = newIdleDisconnects(cfg.idleDisconnectTimeout, app.disconnectFromVoice)
	app.lavalink = disgolink.New(
		client.ApplicationID,
		disgolink.WithListenerFunc(func(disgolinkPlayer disgolink.Player, event lavalink.TrackEndEvent) {
			app.player.OnTrackEnd(disgolinkPlayer, event)
		}),
		disgolink.WithListenerFunc(func(disgolinkPlayer disgolink.Player, event lavalink.TrackExceptionEvent) {
			app.player.OnTrackException(disgolinkPlayer, event)
		}),
	)
	app.player = music.NewPlayer(app.lavalink)
	app.player.SetAutoTrackStartNotifier(app.sendNowPlayingUpdate)
	app.player.SetTrackFailureNotifier(app.sendUnableToPlayUpdate)
	app.player.SetPlaybackActiveNotifier(app.cancelIdleDisconnect)
	app.player.SetPlaybackIdleNotifier(app.scheduleIdleDisconnect)
	app.router = commandrouter.New(commandrouter.Context{
		Player:                app.player,
		StatusChannels:        app.channels,
		PremiumAllowedUsers:   cfg.premiumAllowedUsers,
		PremiumAllowedUserIDs: cfg.premiumAllowedUserIDs,
	}, commands.All(cfg.jokeCommands))

	return app, nil
}

func (app *app) connectLavalink(ctx context.Context) error {
	node, err := app.lavalink.AddNode(ctx, app.config.lavalink)
	if err != nil {
		return err
	}

	return app.warmLavalink(ctx, node)
}

func (app *app) warmLavalink(ctx context.Context, node disgolink.Node) error {
	warmupCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	info, err := node.Info(warmupCtx)
	if err != nil {
		return fmt.Errorf("warm lavalink node: %w", err)
	}

	fmt.Printf("Lavalink node %q is ready: version=%s plugins=%d\n", node.Config().Name, info.Version.Semver, len(info.Plugins))
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

func (app *app) clearGuildCommands() {
	if !app.config.clearGuildCommands {
		return
	}

	guildIDs := map[snowflake.ID]bool{}
	for _, guildID := range app.config.clearGuildCommandIDs {
		guildIDs[guildID] = true
	}
	for guild := range app.discord.Caches.Guilds() {
		guildIDs[guild.ID] = true
	}

	cleared := 0
	for guildID := range guildIDs {
		if _, err := app.discord.Rest.SetGuildCommands(app.discord.ApplicationID, guildID, nil); err != nil {
			fmt.Fprintf(os.Stderr, "clear guild commands failed guild_id=%s: %v\n", guildID, err)
			continue
		}
		cleared++
	}

	fmt.Printf("Cleared guild commands for %d guild(s).\n", cleared)
}

func (app *app) warmDiscordRest() {
	if _, err := app.discord.Rest.GetGatewayBot(); err != nil {
		fmt.Fprintf(os.Stderr, "discord rest warmup failed: %v\n", err)
	}
}

func (app *app) sendNowPlayingUpdate(ctx context.Context, guildID snowflake.ID) {
	channelID, ok := app.channels.Get(guildID)
	if !ok {
		return
	}

	snapshot := app.player.Queue(guildID)
	if snapshot.Current == nil {
		return
	}

	embed := helpers.NowPlayingEmbed(*snapshot.Current, snapshot.Queued, snapshot.Position, snapshot.Volume, "")
	if _, err := app.discord.Rest.CreateMessage(channelID, discord.NewMessageCreate().WithEmbeds(embed)); err != nil {
		fmt.Fprintf(os.Stderr, "now playing update failed: %v\n", err)
	}
}

func (app *app) sendUnableToPlayUpdate(ctx context.Context, guildID snowflake.ID, track lavalink.Track) {
	channelID, ok := app.channels.Get(guildID)
	if !ok {
		return
	}

	embed := helpers.UnableToPlayEmbed(track)
	if _, err := app.discord.Rest.CreateMessage(channelID, discord.NewMessageCreate().WithEmbeds(embed)); err != nil {
		fmt.Fprintf(os.Stderr, "unable to play update failed: %v\n", err)
	}
}

func (app *app) close(ctx context.Context) {
	if app.idle != nil {
		app.idle.stopAll()
	}

	if app.discord != nil {
		app.discord.Close(ctx)
	}

	if app.lavalink != nil {
		app.lavalink.Close()
	}
}
