package commandrouter

import (
	"context"
	"sort"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"

	"ytdlpPlayer/music"
)

type Command struct {
	Definition discord.ApplicationCommandCreate
	Handle     func(ctx Context, event *events.ApplicationCommandInteractionCreate)
}

type Context struct {
	Context               context.Context
	GuildID               snowflake.ID
	Player                *music.Player
	PremiumAllowedUsers   map[snowflake.ID]bool
	PremiumAllowedUserIDs string
}

func (ctx Context) PremiumFallbackAllowed(userID snowflake.ID) bool {
	return ctx.PremiumAllowedUsers[userID]
}

type Router struct {
	commands map[string]Command
	context  Context
}

func New(context Context, commands map[string]Command) Router {
	return Router{
		commands: commands,
		context:  context,
	}
}

func (r Router) Definitions() []discord.ApplicationCommandCreate {
	definitions := make([]discord.ApplicationCommandCreate, 0, len(r.commands))
	names := make([]string, 0, len(r.commands))
	for name := range r.commands {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		definitions = append(definitions, r.commands[name].Definition)
	}

	return definitions
}

func (r Router) Handle(ctx context.Context, event *events.ApplicationCommandInteractionCreate) {
	name := event.SlashCommandInteractionData().CommandName()
	command, ok := r.commands[name]
	if !ok {
		return
	}

	commandContext := r.context
	commandContext.Context = ctx
	guildID := event.GuildID()
	if guildID == nil {
		RespondError(event, "This command can only be used in a server.")
		return
	}
	commandContext.GuildID = *guildID
	command.Handle(commandContext, event)
}
