package commands

import (
	"fmt"
	"os"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"

	"ytdlpPlayer/commandrouter"
)

var SetChannel = commandrouter.Command{
	Definition: discord.SlashCommandCreate{
		Name:        "setchannel",
		Description: "Use this channel for automatic now playing updates",
	},
	Handle: handleSetChannel,
}

func handleSetChannel(ctx commandrouter.Context, event *events.ApplicationCommandInteractionCreate) {
	if ctx.StatusChannels == nil {
		commandrouter.RespondError(event, "Status channel storage is not ready yet.")
		return
	}

	channelID := event.Channel().ID()
	ctx.StatusChannels.Set(ctx.GuildID, channelID)

	if err := event.CreateMessage(discord.NewMessageCreate().WithContent(fmt.Sprintf("Automatic now playing updates will post in <#%s>.", channelID))); err != nil {
		fmt.Fprintf(os.Stderr, "set channel response failed: %v\n", err)
	}
}
