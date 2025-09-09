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

	if !cmd.cfg.AllowOvertime || !cmd.cfg.FakeValidTime {
		cmd.logger.WarnContext(ctx, "commit created outside of schedule, author time is over time", "previous_shift", schedule.PreviousShift(authorDate).String(), "next_shift", schedule.NextShift(authorDate).String())
		return nil
	}

	previousShift := schedule.PreviousShift(authorDate)

	probableTime, err := cmd.fakeAuthorDate(ctx, previousShift)
	if err != nil {
		return fmt.Errorf("unable to calculate probable commit time: %w", err)
	}

	cmd.logger.InfoContext(ctx, "changing last commit date to avoid overtime",
		"shift", previousShift.String(),
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

// fakeAuthorDate generates a realistic commit time within the previous work shift boundaries
// while preserving chronological order with the previous commit.
//
// Rules:
// - starts with previous work shift time boundaries as the base window
// - adjusts lower bound to previous commit time if it's after shift start (maintains chronological order)
// - adjusts upper bound to previous commit time + 10min if previous commit is after shift end
// - caps upper bound to current time if upper bound is in the future
// - for tight windows (<45min): distributes randomly across available time
// - for normal windows: adds realistic delay (5-15min base + 0-30min random) to simulate human timing
func (*cmdPostCommit) fakeAuthorDate(ctx context.Context, authorLastShift *workhours.WorkingShift) (time.Time, error) {
	lastCommitTime, err := git.GetCommitTime(ctx, "HEAD", 1)
	if err != nil {
		return time.Time{}, fmt.Errorf("could not get last commit time: %w", err)
	}

	lowerBound := authorLastShift[0]
	upperBound := authorLastShift[1]

	if lastCommitTime.After(lowerBound) {
		lowerBound = lastCommitTime
	}

	if lastCommitTime.After(upperBound) {
		upperBound = lastCommitTime.Add(time.Minute * 10)
	}

	if now := time.Now(); upperBound.After(now) {
		upperBound = now
	}

	if timeAvailable := upperBound.Sub(lowerBound); timeAvailable < time.Minute*45 {
		return lowerBound.Add(time.Duration(rand.Int64N(int64(timeAvailable)))), nil
	}

	return lowerBound.Add(
		(time.Minute * time.Duration(5+rand.Int64N(10))) + // 5-15mn
			time.Duration(rand.Int64N(int64(30*time.Minute))), // 0-30mn
	), nil
}
