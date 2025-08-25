package gitconfig

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	clicfg "github.com/krostar/cli/cfg"

	"github.com/krostar/git-workhours/internal/git/config/reflectx"
)

// SourceOption configures git config source behavior.
type SourceOption func(o *sourceOptions)

type sourceOptions struct {
	gitCommandExecutor    func(context.Context, ...string) (string, error)
	gitConfigParamEnvName string
	ignoreSetFieldError   bool
}

// SourceWithGitCommandExecutor sets a custom git command executor.
func SourceWithGitCommandExecutor(execFunc func(context.Context, ...string) (string, error)) SourceOption {
	return func(o *sourceOptions) { o.gitCommandExecutor = execFunc }
}

// SourceWithGitConfigParameterEnvName sets the environment variable name for git config parameters.
func SourceWithGitConfigParameterEnvName(name string) SourceOption {
	return func(o *sourceOptions) { o.gitConfigParamEnvName = name }
}

// SourceWithIgnoreConfigError ignores whenever a git config is not applicable in the destination.
func SourceWithIgnoreConfigError() SourceOption {
	return func(o *sourceOptions) { o.ignoreSetFieldError = true }
}

// Source creates a configuration source that loads git config values into a struct.
func Source[T any](sectionName string, opts ...SourceOption) clicfg.SourceFunc[T] {
	o := sourceOptions{
		gitCommandExecutor: func(ctx context.Context, args ...string) (string, error) {
			cmd := exec.CommandContext(ctx, "git", args...)

			output, err := cmd.Output()
			if err != nil {
				var stdErr string

				if exitErr := (*exec.ExitError)(nil); errors.As(err, &exitErr) {
					stdErr = "; stderr: " + string(exitErr.Stderr)
					if exitErr.ExitCode() == 1 { // config not found
						return "", nil
					}
				}

				return "", fmt.Errorf("%w%s", err, stdErr)
			}

			return string(output), nil
		},
		gitConfigParamEnvName: "GIT_CONFIG_PARAMETERS",
		ignoreSetFieldError:   false,
	}
	for _, opt := range opts {
		opt(&o)
	}

	return func(ctx context.Context, cfg *T) error {
		var configs []string

		{ // get git config for section
			var gitArgs []string

			if rawConfigParameters, found := os.LookupEnv(o.gitConfigParamEnvName); found {
				configParameters, err := ParseEnvParameters(rawConfigParameters)
				if err != nil {
					return fmt.Errorf("could not parse git configuration parameters from env: %w", err)
				}

				for name, value := range configParameters {
					gitArgs = append(gitArgs, "-c", name+"="+value)
				}
			}

			gitArgs = append(gitArgs, "config", "--get-regexp", fmt.Sprintf("^%s\\..*", sectionName))

			output, err := o.gitCommandExecutor(ctx, gitArgs...)
			if err != nil {
				return fmt.Errorf("unable to execute git config command: %w", err)
			}

			configs = strings.Split(strings.TrimSpace(output), "\n")
		}

		for _, config := range configs {
			config = strings.TrimSpace(config)
			if config == "" {
				continue
			}

			parts := strings.SplitN(strings.TrimSpace(config), " ", 2)
			if len(parts) < 2 {
				return fmt.Errorf("could not parse git config output: %s", config)
			}

			if err := reflectx.SetToPath[T](cfg, strings.TrimPrefix(parts[0], sectionName+"."), parts[1]); err != nil && !o.ignoreSetFieldError {
				return fmt.Errorf("unable to apply git config %q: %w", config, err)
			}
		}

		return nil
	}
}
