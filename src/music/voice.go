package music

import (
	"context"

	"github.com/disgoorg/snowflake/v2"
)

func (p *Player) OnVoiceStateUpdate(ctx context.Context, guildID snowflake.ID, channelID *snowflake.ID, sessionID string) {
	p.mu.Lock()
	playback := p.playback(guildID)
	playback.voiceChannelID = channelID
	playback.voiceSessionID = sessionID
	p.mu.Unlock()

	p.lavalinkPlayer(guildID).OnVoiceStateUpdate(ctx, channelID, sessionID)
}

func (p *Player) OnVoiceServerUpdate(ctx context.Context, guildID snowflake.ID, token string, endpoint string) {
	p.mu.Lock()
	playback := p.playback(guildID)
	playback.voiceServerToken = token
	playback.voiceEndpoint = endpoint
	p.mu.Unlock()

	p.lavalinkPlayer(guildID).OnVoiceServerUpdate(ctx, token, endpoint)
}
