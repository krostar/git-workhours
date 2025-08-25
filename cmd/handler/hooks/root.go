package handlerhooks

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/krostar/cli"
	clidi "github.com/krostar/cli/di"
)

// Root returns the root command for hook-related operations.
func Root() cli.Command { return new(cmdRoot) }

type cmdRoot struct {
	cfg hookSharedConfig
}

func (*cmdRoot) Description() string {
	return "Git hook commands to deal with commits made outside of working hours."
}

func (cmd *cmdRoot) PersistentFlags() []cli.Flag {
	return []cli.Flag{
		cli.NewBuiltinFlag("schedule", "", &cmd.cfg.Schedule, "Work schedule in format 'slice of time shift', eg: ',8h-12h+13h-18h,9h-18h,,,,'"),
		cli.NewBuiltinFlag("inverse-schedule", "", &cmd.cfg.InvertSchedule, "Invert the work schedule"),
		cli.NewBuiltinFlag("allow-overtime", "", &cmd.cfg.AllowOvertime, "Allow commits outside work hours with warning"),
	}
}

func (cmd *cmdRoot) PersistentHook() *cli.PersistentHook {
	return &cli.PersistentHook{
		BeforeCommandExecution: func(ctx context.Context) error {
			clidi.AddProvider(ctx, func() *hookSharedConfig { return &cmd.cfg })

			if os.Getenv("GIT_WH_ONGOING") != "" {
				return cli.NewErrorWithExitStatus(errors.New("skipping to avoid infinite hook recursion"), 0)
			}

			if err := os.Setenv("GIT_WH_ONGOING", "1"); err != nil {
				return fmt.Errorf("could not set GIT_WH_ONGOING=true: %w", err)
			}

			return sourceConfigHook(&cmd.cfg)(ctx)
		},
	}
}

func (*cmdRoot) Execute(_ context.Context, _, _ []string) error {
	return cli.NewErrorWithExitStatus(cli.NewErrorWithHelp(nil), 0)
}
