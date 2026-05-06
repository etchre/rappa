package bot

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/disgoorg/snowflake/v2"
)

type idleDisconnects struct {
	mu         sync.Mutex
	timeout    time.Duration
	timers     map[snowflake.ID]*time.Timer
	disconnect func(ctx context.Context, guildID snowflake.ID, reason string)
}

func newIdleDisconnects(timeout time.Duration, disconnect func(ctx context.Context, guildID snowflake.ID, reason string)) *idleDisconnects {
	return &idleDisconnects{
		timeout:    timeout,
		timers:     map[snowflake.ID]*time.Timer{},
		disconnect: disconnect,
	}
}

func (idle *idleDisconnects) schedule(ctx context.Context, guildID snowflake.ID, reason string) {
	if idle == nil || idle.timeout <= 0 {
		return
	}

	idle.mu.Lock()
	if timer := idle.timers[guildID]; timer != nil {
		timer.Stop()
	}
	idle.timers[guildID] = time.AfterFunc(idle.timeout, func() {
		idle.mu.Lock()
		delete(idle.timers, guildID)
		idle.mu.Unlock()

		idle.disconnect(ctx, guildID, reason)
	})
	idle.mu.Unlock()
}

func (idle *idleDisconnects) cancel(guildID snowflake.ID) {
	if idle == nil {
		return
	}

	idle.mu.Lock()
	defer idle.mu.Unlock()

	if timer := idle.timers[guildID]; timer != nil {
		timer.Stop()
		delete(idle.timers, guildID)
	}
}

func (idle *idleDisconnects) stopAll() {
	if idle == nil {
		return
	}

	idle.mu.Lock()
	defer idle.mu.Unlock()

	for guildID, timer := range idle.timers {
		timer.Stop()
		delete(idle.timers, guildID)
	}
}

func (app *app) scheduleIdleDisconnect(ctx context.Context, guildID snowflake.ID) {
	app.idle.schedule(ctx, guildID, "idle")
}

func (app *app) cancelIdleDisconnect(_ context.Context, guildID snowflake.ID) {
	app.idle.cancel(guildID)
}

func (app *app) disconnectFromVoice(ctx context.Context, guildID snowflake.ID, reason string) {
	slog.Info("disconnecting from voice", "reason", reason)
	if err := app.player.Stop(ctx, guildID); err != nil {
		slog.Error("stop before voice disconnect failed", "err", err)
	}
	if err := app.discord.UpdateVoiceState(ctx, guildID, nil, false, false); err != nil {
		slog.Error("voice disconnect failed", "err", err)
	}
	app.setBotVoiceChannel(guildID, nil)
	app.idle.cancel(guildID)
}
