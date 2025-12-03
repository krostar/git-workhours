package gitconfig

import (
	"context"
	"errors"
	"strings"
	"testing"

	gocmpopts "github.com/google/go-cmp/cmp/cmpopts"
	"github.com/krostar/test"
	"github.com/krostar/test/check"
)

func Test_SourceWithGitCommandExecutor(t *testing.T) {
	var o sourceOptions

	var called bool

	SourceWithGitCommandExecutor(func(context.Context, ...string) (string, error) {
		called = true
		return "", nil
	})(&o)

	test.Assert(t, o.gitCommandExecutor != nil)
	_, _ = o.gitCommandExecutor(t.Context())
	test.Assert(t, called)
}

func Test_SourceWithGitConfigParameterEnvName(t *testing.T) {
	var o sourceOptions
	SourceWithGitConfigParameterEnvName("foo")(&o)
	test.Assert(t, o.gitConfigParamEnvName == "foo")
}

func Test_SourceWithIgnoreConfigError(t *testing.T) {
	var o sourceOptions
	SourceWithIgnoreConfigError()(&o)
	test.Assert(t, o.ignoreSetFieldError)
}

func Test_Source(t *testing.T) {
	type awesomeConfig struct {
		Name    string
		Age     int
		Details map[string]string
	}

	for name, tc := range map[string]struct {
		baseConfig          awesomeConfig
		envValue            string
		gitConfigOutput     string
		gitExecFailure      bool
		expectedConfig      awesomeConfig
		expectedGitArgs     []string
		expectErrorContains string
	}{
		"no git config set": {
			baseConfig:     awesomeConfig{Name: "bob"},
			expectedConfig: awesomeConfig{Name: "bob"},
		},
		"invalid git config env": {
			envValue:            "foo",
			expectErrorContains: "could not parse git configuration parameters from env",
		},
		"unable to exec git config command": {
			gitExecFailure:      true,
			expectErrorContains: "unable to execute git config command",
		},
		"unable to set config from git config": {
			gitConfigOutput:     "foo",
			expectErrorContains: "could not parse git config output: foo",
		},
		"unable to apply git config": {
			gitConfigOutput:     "foo bar",
			expectErrorContains: "unable to apply git config",
		},
		"ok": {
			baseConfig:      awesomeConfig{Name: "bob", Age: 42},
			envValue:        "'test.name'='not bob' 'test.details.hello'='world'",
			gitConfigOutput: "test.details.say hello",
			expectedConfig: awesomeConfig{
				Name:    "not bob",
				Age:     42,
				Details: map[string]string{"hello": "world", "say": "hello"},
			},
			expectedGitArgs: []string{
				"-c", "test.name=not bob",
				"-c", "test.details.hello=world",
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			if tc.envValue != "" {
				t.Setenv("FOO", tc.envValue)
			} else {
				t.Setenv("FOO", "")
			}

			src := Source[awesomeConfig]("test",
				SourceWithGitConfigParameterEnvName("FOO"),
				SourceWithGitCommandExecutor(func(_ context.Context, args ...string) (string, error) {
					if tc.gitExecFailure {
						return "", errors.New("boom")
					}

					var output strings.Builder
					output.WriteString(tc.gitConfigOutput)

					if len(args) > 3 {
						args = args[:len(args)-3]

						test.Assert(check.Compare(t, args, tc.expectedGitArgs, gocmpopts.SortSlices(func(a, b string) bool { return a < b })))

						for i := 1; i < len(args); i += 2 { // fake apply env to git output
							split := strings.SplitN(strings.ReplaceAll(args[i], "'", ""), "=", 2)
							output.WriteString("\n" + split[0] + " " + split[1])
						}
					}

					return output.String(), nil
				}),
			)

			cfg := tc.baseConfig
			err := src(t.Context(), &cfg)

			if tc.expectErrorContains != "" {
				test.Assert(t, err != nil && strings.Contains(err.Error(), tc.expectErrorContains), err)
			} else {
				test.Require(t, err == nil, err)
				test.Assert(check.Compare(t, cfg, tc.expectedConfig))
			}
		})
	}
}
