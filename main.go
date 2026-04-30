package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		fmt.Fprintln(os.Stderr, "warning: .env file not loaded")
	}

	if err := runBot(); err != nil {
		fmt.Fprintf(os.Stderr, "bot failed: %v\n", err)
		os.Exit(1)
	}
}
