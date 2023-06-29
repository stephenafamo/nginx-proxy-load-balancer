package main

//go:generate go install github.com/volatiletech/sqlboiler/v4
//go:generate go install github.com/volatiletech/sqlboiler-sqlite3
//go:generate go run github.com/volatiletech/sqlboiler/v4 --wipe --no-tests sqlite3

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/sethvargo/go-envconfig"
	"github.com/stephenafamo/warden/cmd"
	"github.com/stephenafamo/warden/internal"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Load env variables from a .env file if present
	err := godotenv.Overload(".env")
	if err != nil {
		// Ignore error if file is not present
		if !errors.Is(err, os.ErrNotExist) {
			panic(err)
		}
	}

	settings := internal.Settings{}

	if err := envconfig.Process(ctx, &settings); err != nil {
		panic(fmt.Errorf("error parsing config: %w", err))
	}

	if err := cmd.Execute(ctx, settings); err != nil {
		panic(err)
	}
}
