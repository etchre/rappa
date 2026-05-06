package bot

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func Run() error {
	ctx := context.Background()

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	app, err := newApp(ctx, cfg)
	if err != nil {
		return err
	}
	defer app.close(ctx)

	if err := app.connectLavalink(ctx); err != nil {
		return fmt.Errorf("connect to lavalink node: %w", err)
	}

	if err := app.registerCommands(); err != nil {
		return fmt.Errorf("sync global commands: %w", err)
	}

	app.warmDiscordRest()

	if err := app.discord.OpenGateway(ctx); err != nil {
		return fmt.Errorf("open discord gateway: %w", err)
	}

	slog.Info("Bot is ready. Use /play with a link. Press Ctrl+C to disconnect.")
	waitForShutdown()

	return nil
}

func waitForShutdown() {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
}
