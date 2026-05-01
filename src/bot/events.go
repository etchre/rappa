package bot

import (
	"context"

	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"

	"ytdlpPlayer/commandrouter"
	"ytdlpPlayer/commands"
)

func (app *app) onVoiceStateUpdate(ctx context.Context) func(event *events.GuildVoiceStateUpdate) {
	return func(event *events.GuildVoiceStateUpdate) {
		if event.VoiceState.UserID != app.lavalink.UserID() {
			app.leaveIfAlone(ctx, event.VoiceState.GuildID)
			return
		}

		app.player.OnVoiceStateUpdate(ctx, event.VoiceState.GuildID, event.VoiceState.ChannelID, event.VoiceState.SessionID)
		if event.VoiceState.ChannelID == nil {
			app.idle.cancel(event.VoiceState.GuildID)
		}
	}
}

func (app *app) onVoiceServerUpdate(ctx context.Context) func(event *events.VoiceServerUpdate) {
	return func(event *events.VoiceServerUpdate) {
		if event.Endpoint == nil {
			return
		}

		app.player.OnVoiceServerUpdate(ctx, event.GuildID, event.Token, *event.Endpoint)
	}
}

func (app *app) onGuildsReady(_ context.Context) func(event *events.GuildsReady) {
	return func(event *events.GuildsReady) {
		app.cleanup.Do(app.clearGuildCommands)
	}
}

func (app *app) onApplicationCommand(ctx context.Context) func(event *events.ApplicationCommandInteractionCreate) {
	return func(event *events.ApplicationCommandInteractionCreate) {
		go app.router.Handle(ctx, event)
	}
}

func (app *app) onComponentInteraction(ctx context.Context) func(event *events.ComponentInteractionCreate) {
	return func(event *events.ComponentInteractionCreate) {
		go func() {
			guildID := event.GuildID()
			if guildID == nil {
				return
			}

			commands.HandleComponent(commandrouter.Context{
				Context: ctx,
				GuildID: *guildID,
				Player:  app.player,
			}, event)
		}()
	}
}

func (app *app) leaveIfAlone(ctx context.Context, guildID snowflake.ID) {
	botVoiceState, ok := app.discord.Caches.VoiceState(guildID, app.lavalink.UserID())
	if !ok || botVoiceState.ChannelID == nil {
		return
	}

	botChannelID := *botVoiceState.ChannelID
	for voiceState := range app.discord.Caches.VoiceStates(guildID) {
		if voiceState.UserID == app.lavalink.UserID() || voiceState.ChannelID == nil {
			continue
		}
		if *voiceState.ChannelID == botChannelID {
			return
		}
	}

	go app.disconnectFromVoice(ctx, guildID, "being left alone")
}
