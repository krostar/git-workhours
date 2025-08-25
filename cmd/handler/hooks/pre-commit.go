package handlerhooks

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/krostar/cli"
	clidi "github.com/krostar/cli/di"

	"github.com/krostar/git-workhours/internal/git"
	"github.com/krostar/git-workhours/internal/workhours"
)

// PreCommit returns the pre-commit hook command.
func PreCommit() cli.Command { return new(cmdPreCommit) }

type cmdPreCommit struct {
	cfg    cmdPreCommitConfig
	logger *slog.Logger
}

type cmdPreCommitConfig struct {
	hookSharedConfig

	AuthorDate string `env:"GIT_AUTHOR_DATE"`
}

func (*cmdPreCommit) Description() string {
	return "Pre-commit hook that validates commits are made within work hours."
}

func (cmd *cmdPreCommit) Flags() []cli.Flag {
	return []cli.Flag{
		cli.NewBuiltinFlag("author-date", "", &cmd.cfg.AuthorDate, "Override author date for commit validation"),
	}
}

func (cmd *cmdPreCommit) Hook() *cli.Hook {
	return &cli.Hook{
		BeforeCommandExecution: func(ctx context.Context) error {
			return errors.Join(
				sourceConfigHook(&cmd.cfg)(ctx),
				clidi.Invoke(ctx, func(shared *hookSharedConfig, logger *slog.Logger) {
					cmd.logger = logger.With("hook", "pre-commit")
					cmd.cfg.hookSharedConfig = *shared
				}),
			)
		},
	}
}

func (cmd *cmdPreCommit) Execute(ctx context.Context, _, _ []string) error {
	authorDate, err := git.ResolveDate(ctx, cmd.cfg.AuthorDate)
	if err != nil {
		return fmt.Errorf("could not resolve commit date: %w", err)
	}

	schedule, err := workhours.ParseWeeklySchedule(cmd.cfg.Schedule)
	if err != nil {
		return fmt.Errorf("unable to parse schedule: %w", err)
	}

	if cmd.cfg.InvertSchedule {
		schedule = schedule.Inverted()
	}

	if current := schedule.CurrentShift(authorDate); current != nil {
		cmd.logger.DebugContext(ctx, "author time is within work schedule",
			"shift", current.String(),
			"author_date", authorDate.Format(time.DateTime),
		)

		return nil
	}

	cmd.logger.WarnContext(ctx, "author time is over time", "previous_shift", schedule.PreviousShift(authorDate).String(), "next_shift", schedule.NextShift(authorDate).String())

	if !cmd.cfg.AllowOvertime {
		return cli.NewErrorWithExitStatus(fmt.Errorf("can't commit now, previous shift ended %s ago", time.Since(schedule.PreviousShift(authorDate)[1]).Truncate(time.Minute).String()), 3)
	}

	return nil
}
