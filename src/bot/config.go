package bot

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/disgoorg/disgolink/v3/disgolink"
	"github.com/disgoorg/snowflake/v2"
)

type config struct {
	token                 string
	clearGlobalCommands   bool
	syncGlobalCommands    bool
	lavalink              disgolink.NodeConfig
	premiumAllowedUserIDs string
	premiumAllowedUsers   map[snowflake.ID]bool
}

func loadConfig() (config, error) {
	token := os.Getenv("DISCORD_BOT_TOKEN")
	if token == "" {
		return config{}, fmt.Errorf("DISCORD_BOT_TOKEN is required")
	}

	lavalinkSecure, err := envBool("LAVALINK_SECURE", false)
	if err != nil {
		return config{}, err
	}

	clearGlobalCommands, err := envBool("CLEAR_GLOBAL_COMMANDS", false)
	if err != nil {
		return config{}, err
	}

	syncGlobalCommands, err := envBool("SYNC_GLOBAL_COMMANDS", false)
	if err != nil {
		return config{}, err
	}

	return config{
		token:                 token,
		clearGlobalCommands:   clearGlobalCommands,
		syncGlobalCommands:    syncGlobalCommands,
		premiumAllowedUserIDs: os.Getenv("PREMIUM_ALLOWED_USER_IDS"),
		premiumAllowedUsers:   parseSnowflakeSet(os.Getenv("PREMIUM_ALLOWED_USER_IDS")),
		lavalink: disgolink.NodeConfig{
			Name:     envDefault("LAVALINK_NODE_NAME", "local"),
			Address:  envDefault("LAVALINK_ADDRESS", "localhost:2333"),
			Password: envDefault("LAVALINK_PASSWORD", "youshallnotpass"),
			Secure:   lavalinkSecure,
		},
	}, nil
}

func parseSnowflakeSet(value string) map[snowflake.ID]bool {
	ids := map[snowflake.ID]bool{}
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		id, err := snowflake.Parse(part)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ignoring invalid premium user id %q: %v\n", part, err)
			continue
		}
		ids[id] = true
	}

	return ids
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
