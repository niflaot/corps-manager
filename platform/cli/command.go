// Package cli contains the discord-bot command adapter.
package cli

import (
	"context"
	"fmt"
	"io"

	appconfig "github.com/pixelados-net/discord-bot/platform/app"
	"github.com/pixelados-net/discord-bot/platform/bootstrap"
)

// Execute runs the discord-bot command with explicit process dependencies.
func Execute(ctx context.Context, args []string, stdout io.Writer, version string) error {
	if len(args) == 0 || args[0] == "serve" {
		return serve(ctx, version)
	}
	switch args[0] {
	case "version", "--version", "-v":
		_, err := fmt.Fprintf(stdout, "discord-bot v%s\n", version)
		return err
	case "help", "--help", "-h":
		_, err := fmt.Fprintln(stdout, "usage: discord-bot [serve|version]")
		return err
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func serve(ctx context.Context, version string) error {
	if err := appconfig.LoadDotenv(); err != nil {
		return fmt.Errorf("load dotenv: %w", err)
	}
	application, err := bootstrap.New(ctx, version)
	if err != nil {
		return err
	}
	defer application.Close()
	return application.Run(ctx)
}
