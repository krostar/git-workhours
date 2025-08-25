package handlerhooks

import (
	"context"

	clicfg "github.com/krostar/cli/cfg"
	sourceenv "github.com/krostar/cli/cfg/source/env"
	sourceflag "github.com/krostar/cli/cfg/source/flag"

	gitconfig "github.com/krostar/git-workhours/internal/git/config"
)

type hookSharedConfig struct {
	Schedule       string
	InvertSchedule bool
	AllowOvertime  bool
}

func sourceConfigHook[T any](dest *T) func(context.Context) error {
	return clicfg.BeforeCommandExecutionHook(dest,
		gitconfig.Source[T]("wh", gitconfig.SourceWithIgnoreConfigError()),
		sourceenv.Source[T]("WH"),
		sourceflag.Source[T](dest),
	)
}
