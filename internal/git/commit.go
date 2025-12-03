package git

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// GetCommitTime retrieves the commit time for a specific git revision.
func GetCommitTime(ctx context.Context, revision string, skip int) (time.Time, error) {
	cmd := exec.CommandContext(ctx, "git", "log", "--max-count=1", "--skip="+strconv.FormatInt(int64(skip), 10), "--format=%at", revision)

	output, err := cmd.Output()
	if err != nil {
		var stdErr string
		if exitErr := (*exec.ExitError)(nil); errors.As(err, &exitErr) {
			stdErr = "; stderr: " + string(exitErr.Stderr)
		}

		return time.Time{}, fmt.Errorf("unable to get last commit timestamp: %w%s", err, stdErr)
	}

	rawTimestamp := strings.TrimSpace(string(output))
	if rawTimestamp == "" {
		return time.Time{}, nil
	}

	timestamp, err := strconv.ParseInt(rawTimestamp, 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("unable to parse timestamp %q: %w", rawTimestamp, err)
	}

	return time.Unix(timestamp, 0), nil
}

// AmendLastCommitDate modifies the date of the last commit to the specified time.
func AmendLastCommitDate(ctx context.Context, date time.Time) error {
	rawDate := date.Format("Mon, 02 Jan 2006 15:04:05 -0700")

	var cmdEnv []string
	{
		envs := make(map[string]string)

		for _, e := range os.Environ() {
			if strings.HasPrefix(e, "GIT_") {
				raw := strings.SplitN(e, "=", 2)
				if len(raw) != 2 {
					continue
				}

				envs[raw[0]] = raw[1]
			}
		}

		for _, key := range []string{"GNUPGHOME", "GPG_TTY", "HOME", "PATH", "PWD"} {
			if value, isset := os.LookupEnv(key); isset {
				envs[key] = value
			}
		}

		envs["GIT_COMMITTER_DATE"] = rawDate

		for k, v := range envs {
			cmdEnv = append(cmdEnv, fmt.Sprintf("%s=%s", k, v))
		}
	}

	cmdArgs := []string{"commit", "--amend", "--no-edit", "--date", rawDate}

	git := exec.CommandContext(ctx, "git", cmdArgs...)
	git.Env = cmdEnv

	if out, err := git.CombinedOutput(); err != nil {
		return fmt.Errorf("unable to execute git command %q: %w b64(out):%s", strings.Join(append([]string{"git"}, cmdArgs...), " "), err, base64.StdEncoding.EncodeToString(out))
	}

	return nil
}
