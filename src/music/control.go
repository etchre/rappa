package music

import (
	"context"
	"fmt"
	"os"

	"github.com/disgoorg/disgolink/v3/disgolink"
	"github.com/disgoorg/disgolink/v3/lavalink"
	"github.com/disgoorg/snowflake/v2"
)

func (p *Player) Skip(ctx context.Context, guildID snowflake.ID) (SkipResult, error) {
	p.mu.Lock()
	playback := p.playback(guildID)
	playback.looping = false
	if len(playback.queue) == 0 {
		wasPlaying := playback.playing
		playback.playing = false
		playback.current = nil
		p.mu.Unlock()

		if wasPlaying {
			if err := p.clearLavalinkTrack(ctx, guildID); err != nil {
				return SkipResult{}, err
			}
		}

		return SkipResult{Stopped: wasPlaying}, nil
	}

	next := playback.queue[0]
	playback.queue = playback.queue[1:]
	playback.current = &next
	playback.playing = true
	p.mu.Unlock()

	if err := p.playTrack(ctx, guildID, next); err != nil {
		return SkipResult{}, err
	}

	return SkipResult{Next: &next}, nil
}

func (p *Player) Stop(ctx context.Context, guildID snowflake.ID) error {
	p.mu.Lock()
	playback := p.playback(guildID)
	wasPlaying := playback.playing
	playback.playing = false
	playback.current = nil
	playback.queue = nil
	playback.looping = false
	p.mu.Unlock()

	if !wasPlaying {
		return nil
	}

	return p.clearLavalinkTrack(ctx, guildID)
}

func (p *Player) OnTrackEnd(player disgolink.Player, event lavalink.TrackEndEvent) {
	if event.Reason == lavalink.TrackEndReasonLoadFailed {
		if p.startPremiumFallback(context.Background(), player, event.Track) {
			return
		}
	}

	if !event.Reason.MayStartNext() {
		return
	}

	if err := p.playNext(context.Background(), player.GuildID()); err != nil {
		fmt.Fprintf(os.Stderr, "play next failed: %v\n", err)
	}
}

func (p *Player) OnTrackException(player disgolink.Player, event lavalink.TrackExceptionEvent) {
	if !looksAccountGated(event.Exception.Message, event.Exception.Cause, event.Exception.CauseStackTrace) {
		debugLavalinkException("playback", playerNodeName(player), trackTitle(event.Track), event.Exception)
		return
	}

	p.startPremiumFallback(context.Background(), player, event.Track)
}

func (p *Player) startPremiumFallback(ctx context.Context, player disgolink.Player, track lavalink.Track) bool {
	if playerNodeName(player) == p.premiumNodeName {
		p.logPremiumFailure(player.GuildID(), track)
		return false
	}

	p.mu.Lock()
	playback := p.playback(player.GuildID())
	if playback.premiumFallbackBusy {
		p.mu.Unlock()
		return true
	}
	if playback.current == nil || playback.current.Encoded != track.Encoded {
		p.mu.Unlock()
		return false
	}
	if playback.premiumFailureLog == nil {
		playback.premiumFailureLog = map[string]bool{}
	}
	if !playback.premiumFailureLog[track.Encoded] {
		fmt.Printf("Failed to play, likely a premium track: %s\n", trackTitle(track))
		playback.premiumFailureLog[track.Encoded] = true
	}

	requester := playback.requesterFor(track)
	if !playback.premiumFallbackAllowedFor(track) {
		fmt.Printf(
			"Refused to use premium fallback for %s requester_id=%s allowed_user_ids=%q\n",
			requester,
			playback.requesterIDFor(track),
			playback.allowedUserIDsFor(track),
		)
		p.mu.Unlock()
		return false
	}
	fmt.Printf("Trying premium fallback for %s\n", trackTitle(track))
	playback.premiumFallbackBusy = true
	p.mu.Unlock()

	go func() {
		if err := p.fallbackToPremium(ctx, player.GuildID(), track); err != nil {
			fmt.Fprintf(os.Stderr, "premium fallback failed: %v\n", err)
			p.mu.Lock()
			p.playback(player.GuildID()).premiumFallbackBusy = false
			p.mu.Unlock()
		}
	}()

	return true
}

func (p *Player) logPremiumFailure(guildID snowflake.ID, track lavalink.Track) {
	p.mu.Lock()
	defer p.mu.Unlock()

	playback := p.playback(guildID)
	if playback.premiumFailureLog == nil {
		playback.premiumFailureLog = map[string]bool{}
	}
	if playback.premiumFailureLog[track.Encoded] {
		return
	}

	fmt.Printf("Failed to play, likely a premium track: %s\n", trackTitle(track))
	playback.premiumFailureLog[track.Encoded] = true
}

func (p *Player) playNext(ctx context.Context, guildID snowflake.ID) error {
	p.mu.Lock()
	playback := p.playback(guildID)
	if playback.premiumFallbackBusy {
		p.mu.Unlock()
		return nil
	}
	if playback.looping && playback.current != nil {
		current := *playback.current
		p.mu.Unlock()
		return p.playTrack(ctx, guildID, current)
	}

	if len(playback.queue) == 0 {
		playback.playing = false
		playback.current = nil
		playback.looping = false
		p.mu.Unlock()
		return nil
	}

	next := playback.queue[0]
	playback.queue = playback.queue[1:]
	playback.current = &next
	p.mu.Unlock()

	return p.playTrack(ctx, guildID, next)
}

func (p *Player) playTrack(ctx context.Context, guildID snowflake.ID, track lavalink.Track) error {
	return p.playTrackOnNode(ctx, guildID, p.node(), track)
}

func (p *Player) playTrackOnNode(ctx context.Context, guildID snowflake.ID, node disgolink.Node, track lavalink.Track) error {
	if err := p.ensurePlayerOnNode(ctx, guildID, node); err != nil {
		return err
	}

	player := p.lavalinkPlayerOnNode(guildID, node)
	if err := player.Update(ctx, lavalink.WithTrack(track)); err != nil {
		debugTrackPlayError(player, track, err)
		return fmt.Errorf("play track: %w", err)
	}

	fmt.Printf("Now playing on Lavalink node %q: %s\n", playerNodeName(player), trackTitle(track))
	return nil
}

func (p *Player) fallbackToPremium(ctx context.Context, guildID snowflake.ID, failedTrack lavalink.Track) error {
	premiumNode := p.premiumNode()
	if premiumNode == nil {
		return fmt.Errorf("premium lavalink node %q is not connected", p.premiumNodeName)
	}

	identifier := originalTrackIdentifier(failedTrack)
	if identifier == "" {
		return fmt.Errorf("cannot determine original track URL for premium fallback")
	}

	defer func() {
		p.mu.Lock()
		p.playback(guildID).premiumFallbackBusy = false
		p.mu.Unlock()
	}()

	loaded, err := loadPlayableTracks(ctx, premiumNode, identifier)
	if err != nil {
		return fmt.Errorf("load premium fallback track: %w", err)
	}
	if len(loaded.Tracks) == 0 {
		return fmt.Errorf("premium fallback returned no tracks")
	}

	premiumTrack := loaded.Tracks[0]
	premiumTrack.Info = failedTrack.Info

	p.mu.Lock()
	playback := p.playback(guildID)
	if playback.current == nil || playback.current.Encoded != failedTrack.Encoded {
		p.mu.Unlock()
		return nil
	}
	p.mu.Unlock()

	if err := p.movePlayerToNode(ctx, guildID, premiumNode); err != nil {
		return err
	}

	p.mu.Lock()
	playback = p.playback(guildID)
	shouldPlay := false
	if playback.current != nil && playback.current.Encoded == failedTrack.Encoded {
		playback.current = &premiumTrack
		playback.playing = true
		shouldPlay = true
	}
	p.mu.Unlock()
	if !shouldPlay {
		return nil
	}

	return p.playTrackOnNode(ctx, guildID, premiumNode, premiumTrack)
}

func (p *Player) ensurePlayerOnNode(ctx context.Context, guildID snowflake.ID, node disgolink.Node) error {
	existing := p.lavalink.ExistingPlayer(guildID)
	if existing == nil || playerNodeName(existing) == nodeName(node) {
		return nil
	}

	return p.movePlayerToNode(ctx, guildID, node)
}

func (p *Player) movePlayerToNode(ctx context.Context, guildID snowflake.ID, node disgolink.Node) error {
	existing := p.lavalink.ExistingPlayer(guildID)
	if existing != nil && playerNodeName(existing) == nodeName(node) {
		return nil
	}

	p.mu.Lock()
	playback := p.playback(guildID)
	channelID := playback.voiceChannelID
	sessionID := playback.voiceSessionID
	token := playback.voiceServerToken
	endpoint := playback.voiceEndpoint
	p.mu.Unlock()

	if existing != nil {
		if err := existing.Destroy(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "destroy old lavalink player failed during node switch: %v\n", err)
		}
		p.lavalink.RemovePlayer(guildID)
	}

	player := p.lavalinkPlayerOnNode(guildID, node)
	if channelID != nil {
		player.OnVoiceStateUpdate(ctx, channelID, sessionID)
	}
	if token != "" && endpoint != "" {
		player.OnVoiceServerUpdate(ctx, token, endpoint)
	}

	return nil
}

func (p *Player) clearLavalinkTrack(ctx context.Context, guildID snowflake.ID) error {
	player := p.lavalinkPlayer(guildID)
	if err := player.Update(ctx, lavalink.WithNullTrack()); err != nil {
		return fmt.Errorf("stop track: %w", err)
	}

	return nil
}

func (p *Player) ToggleLoop(guildID snowflake.ID) (LoopResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	playback := p.playback(guildID)
	if !playback.playing || playback.current == nil {
		return LoopResult{}, fmt.Errorf("nothing is playing")
	}

	playback.looping = !playback.looping
	return LoopResult{
		Track:   *playback.current,
		Looping: playback.looping,
	}, nil
}

func (p *Player) Restart(ctx context.Context, guildID snowflake.ID) (lavalink.Track, error) {
	p.mu.Lock()
	playback := p.playback(guildID)
	if !playback.playing || playback.current == nil {
		p.mu.Unlock()
		return lavalink.Track{}, fmt.Errorf("nothing is playing")
	}
	current := *playback.current
	p.mu.Unlock()

	player := p.lavalinkPlayer(guildID)
	if err := player.Update(ctx, lavalink.WithPosition(0)); err != nil {
		return lavalink.Track{}, fmt.Errorf("restart track: %w", err)
	}

	return current, nil
}
