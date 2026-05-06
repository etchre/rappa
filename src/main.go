package main

import (
	"errors"
	"log/slog"
	"os"

	"github.com/joho/godotenv"

	"rappa/bot"
)

func main() {
	if err := loadEnv(); err != nil {
		slog.Warn(".env file not loaded")
	}

	if err := bot.Run(); err != nil {
		slog.Error("bot failed", "err", err)
		os.Exit(1)
	}
}

func loadEnv() error {
	for _, filename := range []string{".env", "../.env"} {
		err := godotenv.Load(filename)
		if err == nil {
			return nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return os.ErrNotExist
}
