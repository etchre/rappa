package music

import (
	"context"
	"fmt"
	"time"

	"github.com/disgoorg/snowflake/v2"
)

func (p *Player) OnVoiceStateUpdate(ctx context.Context, guildID snowflake.ID, channelID *snowflake.ID, sessionID string) {
	p.lavalinkPlayer(guildID).OnVoiceStateUpdate(ctx, channelID, sessionID)
}

func (p *Player) OnVoiceServerUpdate(ctx context.Context, guildID snowflake.ID, token string, endpoint string) {
	p.lavalinkPlayer(guildID).OnVoiceServerUpdate(ctx, token, endpoint)
}

func (p *Player) WaitUntilVoiceReady(ctx context.Context, guildID snowflake.ID, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		if p.lavalinkPlayer(guildID).State().Connected {
			return nil
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for Lavalink voice connection")
		case <-ticker.C:
		}
	}
}
