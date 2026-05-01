package bot

import (
	"context"

	"github.com/disgoorg/disgo/events"

	"ytdlpPlayer/commandrouter"
	"ytdlpPlayer/commands"
)

func (app *app) onVoiceStateUpdate(ctx context.Context) func(event *events.GuildVoiceStateUpdate) {
	return func(event *events.GuildVoiceStateUpdate) {
		if event.VoiceState.UserID != app.lavalink.UserID() {
			return
		}

		app.player.OnVoiceStateUpdate(ctx, event.VoiceState.GuildID, event.VoiceState.ChannelID, event.VoiceState.SessionID)
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
