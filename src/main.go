package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/joho/godotenv"

	"ytdlpPlayer/bot"
)

func main() {
	if err := loadEnv(); err != nil {
		fmt.Fprintln(os.Stderr, "warning: .env file not loaded")
	}

	if err := bot.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "bot failed: %v\n", err)
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
