package handlerhooks

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/krostar/cli"
	clidi "github.com/krostar/cli/di"

	"github.com/krostar/git-workhours/internal/workhours"
)

// PrintConfig returns the print-config hook command.
func PrintConfig() cli.Command { return new(cmdPrintConfig) }

type cmdPrintConfig struct {
	cfg hookSharedConfig
}

func (cmd *cmdPrintConfig) Hook() *cli.Hook {
	return &cli.Hook{
		BeforeCommandExecution: func(ctx context.Context) error {
			return clidi.Invoke(ctx, func(shared *hookSharedConfig) { cmd.cfg = *shared })
		},
	}
}

func (*cmdPrintConfig) Description() string {
	return "Print configuration values"
}

func (cmd *cmdPrintConfig) Execute(_ context.Context, _, _ []string) error {
	fmt.Println("Hook Configuration")
	fmt.Printf("  Schedule: %q\n", cmd.cfg.Schedule)
	fmt.Printf("  InvertSchedule: %t\n", cmd.cfg.InvertSchedule)
	fmt.Printf("  AllowOvertime: %t\n", cmd.cfg.AllowOvertime)

	schedule, err := workhours.ParseWeeklySchedule(cmd.cfg.Schedule)
	if err != nil {
		return fmt.Errorf("unable to parse schedule: %w", err)
	}

	if cmd.cfg.InvertSchedule {
		schedule = schedule.Inverted()
	}

	fmt.Print("\nParsed Schedule:")

	days := []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}

	for day, shifts := range schedule {
		fmt.Printf("  %s: ", days[day])

		if len(shifts) == 0 {
			fmt.Println("No working hours")
			continue
		}

		shiftStrs := make([]string, len(shifts))
		for i, shift := range shifts {
			shiftStrs[i] = fmt.Sprintf("%v-%v", shift[0].Truncate(time.Minute), shift[1].Truncate(time.Minute))
		}

		fmt.Printf("%s\n", strings.Join(shiftStrs, ", "))
	}

	return nil
}
