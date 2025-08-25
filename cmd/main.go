package main

import (
	"os"
	"syscall"

	"github.com/krostar/cli"
	spf13cobra "github.com/krostar/cli/mapper/spf13/cobra"

	"github.com/krostar/git-workhours/cmd/handler"
	handlerhooks "github.com/krostar/git-workhours/cmd/handler/hooks"
)

func main() {
	ctx, cancel := cli.NewContextCancelableBySignal(syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	cli.Exit(ctx, spf13cobra.Execute(ctx, os.Args, buildCLI()))
}

func buildCLI() *cli.CLI {
	return cli.New(handler.Root()).
		Mount("hooks", cli.New(handlerhooks.Root()).
			AddCommand("print-config", handlerhooks.PrintConfig()).
			AddCommand("pre-commit", handlerhooks.PreCommit()).
			AddCommand("post-commit", handlerhooks.PostCommit()).
			AddCommand("pre-push", handlerhooks.PrePush()),
		)
}
