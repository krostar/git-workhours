package handlerhooks

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"time"

	"github.com/krostar/cli"
	clidi "github.com/krostar/cli/di"

	"github.com/krostar/git-workhours/internal/git"
	"github.com/krostar/git-workhours/internal/workhours"
)

// PostCommit returns the post-commit hook command.
func PostCommit() cli.Command { return new(cmdPostCommit) }

type cmdPostCommit struct {
	cfg    cmdPostCommitConfig
	logger *slog.Logger
}

type cmdPostCommitConfig struct {
	hookSharedConfig `env:"-"`

	AuthorDate    string `env:"GIT_AUTHOR_DATE"`
	FakeValidTime bool
	Force         bool
	DryRun        bool
}

func (*cmdPostCommit) Description() string {
	return "Post-commit hook that adjusts commit times to fall within work hours."
}

func (cmd *cmdPostCommit) Flags() []cli.Flag {
	return []cli.Flag{
		cli.NewBuiltinFlag("force", "f", &cmd.cfg.Force, "Amend the commit regardless of schedule"),
		cli.NewBuiltinFlag("dry-run", "", &cmd.cfg.DryRun, "Don't perform any writing operations"),
		cli.NewBuiltinFlag("author-date", "", &cmd.cfg.AuthorDate, "Date of the commit"),
		cli.NewBuiltinFlag("fake-valid-time", "", &cmd.cfg.FakeValidTime, "Automatically adjust commit times to fall within work hours"),
	}
}

func (cmd *cmdPostCommit) Hook() *cli.Hook {
	return &cli.Hook{
		BeforeCommandExecution: func(ctx context.Context) error {
			return errors.Join(
				clidi.Invoke(ctx, func(shared *hookSharedConfig, logger *slog.Logger) {
					cmd.logger = logger.With("hook", "post-commit")
					cmd.cfg.hookSharedConfig = *shared
				}),
				sourceConfigHook(&cmd.cfg)(ctx),
			)
		},
	}
}

func (cmd *cmdPostCommit) Execute(ctx context.Context, _, _ []string) error {
	if cmd.cfg.AuthorDate == "" {
		cmd.logger.DebugContext(ctx, "no author date provided, nothing to rewrite, skipping")
	}

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

	if schedule.CurrentShift(authorDate) != nil {
		cmd.logger.DebugContext(ctx, "author date is within current shift",
			"schedule", cmd.cfg.Schedule,
			"schedule_inverted", cmd.cfg.InvertSchedule,
			"author_date", authorDate.Format(time.DateTime),
		)

		if !cmd.cfg.Force {
			return nil
		}
	}

	if !cmd.cfg.FakeValidTime {
		cmd.logger.WarnContext(ctx, "commit created outside of schedule, author time is over time", "previous_shift", schedule.PreviousShift(authorDate).String(), "next_shift", schedule.NextShift(authorDate).String())
		return nil
	}

	probableTime, err := cmd.computeAuthorDate(ctx, &schedule, authorDate)
	if err != nil {
		return fmt.Errorf("unable to calculate probable commit time: %w", err)
	}

	cmd.logger.InfoContext(ctx, "changing last commit date to avoid overtime",
		"shift", schedule.PreviousShift(authorDate).String(),
		"old", authorDate.Format(time.DateTime),
		"new", probableTime.Format(time.DateTime),
	)

	if cmd.cfg.DryRun {
		cmd.logger.WarnContext(ctx, "amending last commit date to avoid overtime skipped due to dry-run")
		return nil
	}

	if err := git.AmendLastCommitDate(ctx, probableTime); err != nil {
		return fmt.Errorf("unable to amend last commit date: %w", err)
	}

	return nil
}

func (*cmdPostCommit) computeAuthorDate(ctx context.Context, schedule *workhours.WeeklySchedule, authorDate time.Time) (time.Time, error) {
	authorLastShift := schedule.PreviousShift(authorDate)

	lastCommitTime, err := git.GetCommitTime(ctx, "HEAD", 1)
	if err != nil {
		return time.Time{}, fmt.Errorf("could not get last commit time: %w", err)
	}

	lowerBound := authorLastShift[0]
	upperBound := authorLastShift[1]

	if !lastCommitTime.IsZero() {
		lowerBound = lastCommitTime
	}

	if lastCommitTime.After(authorLastShift[1]) {
		upperBound = time.Now()
	}

	if timeAvailable := upperBound.Sub(lowerBound); timeAvailable < time.Minute*15 {
		return lowerBound.Add(time.Duration(rand.Int64N(int64(timeAvailable)))), nil
	}

	return lowerBound.Add(time.Minute*time.Duration(3+rand.Int64N(2)) + time.Duration(rand.Int64N(int64(10*time.Minute)))), nil
}
