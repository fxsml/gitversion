package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/fxsml/gitversion/pkg/version"
	"github.com/urfave/cli/v3"
)

const gitversionEnvPrefix = "GITVERSION_"

var (
	detailedFlag = &cli.BoolFlag{
		Name:    "detailed",
		Usage:   "show detailed version information",
		Sources: cli.EnvVars(gitversionEnvPrefix + "DETAILED"),
	}
	repoDirFlag = &cli.StringFlag{
		Name:    "repo-path",
		Usage:   "path to git repository",
		Value:   ".",
		Sources: cli.EnvVars(gitversionEnvPrefix + "REPO_PATH"),
	}
	defaultBranchFlag = &cli.StringFlag{
		Name:    "default-branch",
		Usage:   "default branch name (auto-detected if not set)",
		Sources: cli.EnvVars(gitversionEnvPrefix + "DEFAULT_BRANCH"),
	}
	prefixFlag = &cli.StringFlag{
		Name:    "prefix",
		Usage:   "scope tag lookup to tags beginning with `prefix`",
		Sources: cli.EnvVars(gitversionEnvPrefix + "PREFIX"),
	}
	stripPrefixFlag = &cli.BoolFlag{
		Name:    "strip-prefix",
		Usage:   "strip the prefix from the version output",
		Sources: cli.EnvVars(gitversionEnvPrefix + "STRIP_PREFIX"),
	}
	pathFlag = &cli.StringFlag{
		Name:    "path",
		Usage:   "only count commits that touched files under `path`",
		Sources: cli.EnvVars(gitversionEnvPrefix + "PATH"),
	}
	latestFlag = &cli.BoolFlag{
		Name:    "latest",
		Usage:   "output the nearest tag reachable from HEAD without any generated suffix",
		Sources: cli.EnvVars(gitversionEnvPrefix + "LATEST"),
	}
	targetFlag = &cli.StringFlag{
		Name:    "target",
		Usage:   "output next semver bump from the latest tag: dev (default), patch, minor, or major",
		Sources: cli.EnvVars(gitversionEnvPrefix + "TARGET"),
	}

	rootCmd = &cli.Command{
		Name:  "gitversion",
		Usage: "determine semantic version based on git tags",
		Flags: []cli.Flag{
			detailedFlag,
			repoDirFlag,
			defaultBranchFlag,
			prefixFlag,
			stripPrefixFlag,
			pathFlag,
			latestFlag,
			targetFlag,
		},
		Action: run,
	}
)

func run(ctx context.Context, cmd *cli.Command) error {
	info, err := version.GetVersionInfo(cmd.String("C"), version.Options{
		DefaultBranch: cmd.String("default-branch"),
		TagPrefix:     cmd.String("prefix"),
		StripPrefix:   cmd.Bool("strip-prefix"),
		FilterPath:    cmd.String("path"),
		Latest:        cmd.Bool("latest"),
		Target:        cmd.String("target"),
	})
	if err != nil {
		return err
	}

	if cmd.Bool("detailed") {
		fmt.Println(info.DetailedString())
	} else {
		fmt.Println(info.Version)
	}
	return nil
}

func main() {
	if err := rootCmd.Run(context.Background(), os.Args); err != nil {
		slog.Error("command failed", "error", err)
		os.Exit(1)
	}
}
