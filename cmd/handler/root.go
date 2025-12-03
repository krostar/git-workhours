package handler

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/krostar/cli"
	clidi "github.com/krostar/cli/di"
	"github.com/mattn/go-isatty"
	"gitlab.com/greyxor/slogor"
)

// Root returns the root command handler for the git-workhours CLI.
func Root() cli.Command { return new(cmdRoot) }

type cmdRoot struct {
	cfg cmdRootConfig
}

func (*cmdRoot) Description() string {
	return "Git hooks for enforcing work hours and managing commit times"
}

type cmdRootConfig struct {
	LoggerVerbosity string
}

func (cmd *cmdRoot) PersistentFlags() []cli.Flag {
	return []cli.Flag{
		cli.NewBuiltinFlag("log-verbosity", "v", &cmd.cfg.LoggerVerbosity, "Set log verbosity level (debug, info, warn, error)"),
	}
}

func (cmd *cmdRoot) PersistentHook() *cli.PersistentHook {
	return &cli.PersistentHook{
		BeforeFlagsDefinition: func(ctx context.Context) error {
			clidi.InitializeContainer(ctx)

			cmd.cfg = cmdRootConfig{LoggerVerbosity: "info"}
			return nil
		},
		BeforeCommandExecution: func(ctx context.Context) error {
			{ // logger related
				logger, err := cmd.createLogger()
				if err != nil {
					return fmt.Errorf("unable to create logger: %w", err)
				}

				slog.SetDefault(logger)
				clidi.AddProvider(ctx, func() *slog.Logger { return logger })
			}

			return nil
		},
	}
}

func (*cmdRoot) Execute(_ context.Context, _, _ []string) error {
	return cli.NewErrorWithExitStatus(cli.NewErrorWithHelp(nil), 0)
}

func (cmd *cmdRoot) createLogger() (*slog.Logger, error) {
	var loggerLeveler slog.LevelVar

	switch cmd.cfg.LoggerVerbosity {
	case "debug":
		loggerLeveler.Set(slog.LevelDebug)
	case "info":
		loggerLeveler.Set(slog.LevelInfo)
	case "warn":
		loggerLeveler.Set(slog.LevelWarn)
	case "error":
		loggerLeveler.Set(slog.LevelError)
	default:
		return nil, fmt.Errorf("unknown logger level: %s", cmd.cfg.LoggerVerbosity)
	}

	if isatty.IsTerminal(os.Stdout.Fd()) {
		return slog.New(slogor.NewHandler(os.Stdout,
			slogor.SetLevel(loggerLeveler.Level()),
			slogor.SetTimeFormat(time.Kitchen),
		)), nil
	}

	return slog.New(slog.DiscardHandler), nil
}
