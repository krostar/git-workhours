package handlerhooks

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/krostar/cli"
	clidi "github.com/krostar/cli/di"

	"github.com/krostar/git-workhours/internal/workhours"
)

// PrePush returns the pre-push hook command.
func PrePush() cli.Command { return new(cmdPrePush) }

type cmdPrePush struct {
	cfg    hookSharedConfig
	logger *slog.Logger
}

func (*cmdPrePush) Description() string {
	return "Pre-push hook that validates pushes are made within work hours."
}

func (cmd *cmdPrePush) Hook() *cli.Hook {
	return &cli.Hook{
		BeforeCommandExecution: func(ctx context.Context) error {
			return clidi.Invoke(ctx, func(shared *hookSharedConfig, logger *slog.Logger) {
				cmd.logger = logger.With("cmd", "pre-push")
				cmd.cfg = *shared
			})
		},
	}
}

func (cmd *cmdPrePush) Execute(ctx context.Context, _, _ []string) error {
	schedule, err := workhours.ParseWeeklySchedule(cmd.cfg.Schedule)
	if err != nil {
		return fmt.Errorf("unable to parse schedule: %w", err)
	}

	if cmd.cfg.InvertSchedule {
		schedule = schedule.Inverted()
	}

	pushTime := time.Now()

	if schedule.CurrentShift(pushTime) == nil {
		cmd.logger.WarnContext(ctx, "push time is over time", "previous_shift", schedule.PreviousShift(pushTime).String(), "next_shift", schedule.NextShift(pushTime).String())

		if !cmd.cfg.AllowOvertime {
			return cli.NewErrorWithExitStatus(fmt.Errorf("can't push now, previous shift ended %s ago", time.Since(schedule.PreviousShift(pushTime)[1]).Truncate(time.Minute).String()), 3)
		}
	}

	return nil
}
