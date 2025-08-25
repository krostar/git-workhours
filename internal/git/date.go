package git

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"time"

	"al.essio.dev/pkg/shellescape"
)

// ResolveDate resolves a date string using git's expiry-date configuration format.
func ResolveDate(ctx context.Context, value string) (time.Time, error) {
	cmd := exec.CommandContext(ctx, "git", "config", "--local", "--type=expiry-date", "--default", shellescape.Quote(value), "--get", "wh.nonexisting")

	output, err := cmd.Output()
	if err != nil {
		var stdErr string
		if exitErr := (*exec.ExitError)(nil); errors.As(err, &exitErr) {
			stdErr = "; stderr: " + string(exitErr.Stderr)
		}

		return time.Time{}, fmt.Errorf("unable to resolve provided date %q: %w%s", value, err, stdErr)
	}

	ts, err := strconv.ParseInt(string(output[:len(output)-1]), 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("unable to parse timestamp %q: %w", string(output), err)
	}

	return time.Unix(ts, 0), nil
}
