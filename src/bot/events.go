package bot

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"

	"ytdlpPlayer/commandrouter"
	"ytdlpPlayer/commands"
)

func (app *app) onVoiceStateUpdate(ctx context.Context) func(event *events.GuildVoiceStateUpdate) {
	return func(event *events.GuildVoiceStateUpdate) {
		app.recordVoiceState(event.VoiceState.GuildID, event.VoiceState.UserID, event.VoiceState.ChannelID)

		if event.VoiceState.UserID != app.lavalink.UserID() {
			app.checkLeftAloneSoon(ctx, event.VoiceState.GuildID)
			return
		}

		app.setBotVoiceChannel(event.VoiceState.GuildID, event.VoiceState.ChannelID)
		app.player.OnVoiceStateUpdate(ctx, event.VoiceState.GuildID, event.VoiceState.ChannelID, event.VoiceState.SessionID)
		if event.VoiceState.ChannelID == nil {
			app.idle.cancel(event.VoiceState.GuildID)
			return
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
		app.clearGuildCommands()
	}
}

func (app *app) onGuildReady(_ context.Context) func(event *events.GuildReady) {
	return func(event *events.GuildReady) {
		if !app.config.clearGuildCommands {
			return
		}

		if _, err := app.discord.Rest.SetGuildCommands(app.discord.ApplicationID, event.GuildID, nil); err != nil {
			fmt.Fprintf(os.Stderr, "clear guild commands failed guild_id=%s: %v\n", event.GuildID, err)
			return
		}
		fmt.Printf("Cleared guild commands for guild_id=%s.\n", event.GuildID)
	}
}

func (app *app) onApplicationCommand(ctx context.Context) func(event *events.ApplicationCommandInteractionCreate) {
	return func(event *events.ApplicationCommandInteractionCreate) {
		go app.router.Handle(ctx, event)
	}
}

func (app *app) onAutocomplete(ctx context.Context) func(event *events.AutocompleteInteractionCreate) {
	return func(event *events.AutocompleteInteractionCreate) {
		go commands.HandleAutocomplete(commandrouter.Context{
			Context: ctx,
			Player:  app.player,
		}, event)
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
	botChannelID, ok := app.botVoiceChannel(guildID)
	if !ok {
		return
	}

	if app.hasTrackedUserInChannel(guildID, botChannelID) {
		return
	}

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

func (app *app) checkLeftAloneSoon(ctx context.Context, guildID snowflake.ID) {
	go func() {
		time.Sleep(750 * time.Millisecond)
		app.leaveIfAlone(ctx, guildID)
	}()
}

func (app *app) setBotVoiceChannel(guildID snowflake.ID, channelID *snowflake.ID) {
	app.voiceMu.Lock()
	defer app.voiceMu.Unlock()

	if channelID == nil {
		delete(app.voice, guildID)
		return
	}

	app.voice[guildID] = *channelID
}

func (app *app) setUserVoiceChannel(guildID snowflake.ID, userID snowflake.ID, channelID snowflake.ID) {
	app.recordVoiceState(guildID, userID, &channelID)
}

func (app *app) recordVoiceState(guildID snowflake.ID, userID snowflake.ID, channelID *snowflake.ID) {
	app.voiceMu.Lock()
	defer app.voiceMu.Unlock()

	if channelID == nil {
		if users := app.users[guildID]; users != nil {
			delete(users, userID)
			if len(users) == 0 {
				delete(app.users, guildID)
			}
		}
		return
	}

	users := app.users[guildID]
	if users == nil {
		users = map[snowflake.ID]snowflake.ID{}
		app.users[guildID] = users
	}
	users[userID] = *channelID
}

func (app *app) hasTrackedUserInChannel(guildID snowflake.ID, channelID snowflake.ID) bool {
	app.voiceMu.Lock()
	defer app.voiceMu.Unlock()

	for userID, userChannelID := range app.users[guildID] {
		if userID != app.lavalink.UserID() && userChannelID == channelID {
			return true
		}
	}

	return false
}

func (app *app) botVoiceChannel(guildID snowflake.ID) (snowflake.ID, bool) {
	if botVoiceState, ok := app.discord.Caches.VoiceState(guildID, app.lavalink.UserID()); ok && botVoiceState.ChannelID != nil {
		return *botVoiceState.ChannelID, true
	}

	app.voiceMu.Lock()
	defer app.voiceMu.Unlock()

	channelID, ok := app.voice[guildID]
	return channelID, ok
}
