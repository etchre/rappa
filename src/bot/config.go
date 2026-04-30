package bot

import (
	"fmt"
	"os"
	"strconv"

	"github.com/disgoorg/disgolink/v3/disgolink"
)

type config struct {
	token               string
	clearGlobalCommands bool
	syncGlobalCommands  bool
	lavalink            disgolink.NodeConfig
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
