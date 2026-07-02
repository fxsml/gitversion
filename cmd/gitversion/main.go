package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime/debug"

	"github.com/fxsml/gitversion"
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

	versionCmd = &cli.Command{
		Name:  "version",
		Usage: "print the version of gitversion itself",
		Action: func(_ context.Context, _ *cli.Command) error {
			info, ok := debug.ReadBuildInfo()
			if !ok {
				fmt.Println("version: unknown (build info not available)")
				return nil
			}

			v := info.Main.Version
			if v == "" || v == "(devel)" {
				v = "(devel)"
			}
			fmt.Printf("gitversion %s\n", v)

			var commit, vcsTime, modified string
			for _, s := range info.Settings {
				switch s.Key {
				case "vcs.revision":
					commit = s.Value
				case "vcs.time":
					vcsTime = s.Value
				case "vcs.modified":
					modified = s.Value
				}
			}

			if len(commit) >= 7 {
				commit = commit[:7]
			}
			if commit != "" {
				line := "  commit: " + commit
				if modified == "true" {
					line += " (dirty)"
				}
				fmt.Println(line)
			}
			if vcsTime != "" {
				fmt.Println("  time:   " + vcsTime)
			}
			fmt.Println("  go:     " + info.GoVersion)
			return nil
		},
	}

	rootCmd = &cli.Command{
		Name:  "gitversion",
		Usage: "determine semantic version based on git tags",
		Commands: []*cli.Command{
			versionCmd,
		},
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
