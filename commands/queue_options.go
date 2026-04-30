package commands

import (
	"fmt"

	"github.com/disgoorg/disgo/discord"
)

var minQueueNumber = 1

func queueNumberOption(description string) discord.ApplicationCommandOptionInt {
	return discord.ApplicationCommandOptionInt{
		Name:        "queue_number",
		Description: description,
		Required:    true,
		MinValue:    &minQueueNumber,
	}
}

func parseQueueNumber(data discord.SlashCommandInteractionData) (int, error) {
	if number, ok := data.OptInt("queue_number"); ok {
		return number, nil
	}

	intOptions := data.GetByType(discord.ApplicationCommandOptionTypeInt)
	if len(intOptions) == 1 {
		return intOptions[0].Int(), nil
	}

	return 0, fmt.Errorf("choose a queue number from `/queue`")
}
