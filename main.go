package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/fxsml/gitversion/pkg/version"
)

func printHelp() {
	fmt.Println("gitversion - Git-based version string generator")
	fmt.Println()
	fmt.Println("USAGE:")
	fmt.Println("  gitversion [options]")
	fmt.Println("  gitversion help")
	fmt.Println()
	fmt.Println("OPTIONS:")
	fmt.Println("  -detailed              Show detailed version information")
	fmt.Println("  -short                 Show only the version string (default)")
	fmt.Println("  -C <path>              Path to Git repository (default: .)")
	fmt.Println("  -default-branch <name> Default branch name (auto-detected if not set)")
	fmt.Println("  -prefix <prefix>       Scope tag lookup to tags beginning with prefix")
	fmt.Println("  -strip-prefix          Strip the prefix from the version output")
	fmt.Println("  -path <path>           Only count commits that touched files under this path")
	fmt.Println("  -latest                Output the nearest tag reachable from HEAD (no generation)")
	fmt.Println("  -target <bump>         Output the next semver bumped from the latest tag;")
	fmt.Println("                         valid values: dev (default, normal output), patch, minor, major")
	fmt.Println()
	fmt.Println("VERSION LOGIC:")
	fmt.Println("  - Default branch with tags:    Uses semver from nearest matching tag")
	fmt.Println("  - Default branch, N ahead:     <tag>+dev.<N>.g<hash>")
	fmt.Println("  - Default branch without tags: <branch-slug>-g<hash>")
	fmt.Println("  - Other branches:              <branch-slug>-g<hash>")
	fmt.Println("  - Dirty tree:                  Appends -YYYYMMDDHHMMSS timestamp")
	fmt.Println()
	fmt.Println("EXAMPLES:")
	fmt.Println("  gitversion                                     # Print version")
	fmt.Println("  gitversion -detailed                           # Print detailed info")
	fmt.Println("  gitversion -C /repo                            # Version for specific repo")
	fmt.Println("  gitversion -default-branch master              # Specify default branch")
	fmt.Println("  gitversion -prefix my-service/v                     # Monorepo: scoped to prefix")
	fmt.Println("  gitversion -prefix my-service/v -strip-prefix       # Strip prefix from output")
	fmt.Println("  gitversion -prefix my-service/v -path my-service/   # Monorepo: prefix + path filter")
	fmt.Println("  gitversion -latest                              # Nearest release tag")
	fmt.Println("  gitversion -latest -prefix my-service/v         # Nearest tag for a module")
	fmt.Println("  gitversion -target patch                        # Next patch release")
	fmt.Println("  gitversion -prefix my-service/v -target minor   # Next minor for a module")
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "help" {
		printHelp()
		os.Exit(0)
	}

	var (
		detailedFlag      = flag.Bool("detailed", false, "Show detailed version information")
		shortFlag         = flag.Bool("short", false, "Show only the version string")
		repoDirFlag       = flag.String("C", ".", "Path to Git repository")
		defaultBranchFlag = flag.String("default-branch", "", "Default branch name (auto-detected if not set)")
		prefixFlag        = flag.String("prefix", "", "Scope tag lookup to tags with this prefix")
		stripPrefixFlag   = flag.Bool("strip-prefix", false, "Strip the prefix from the version output")
		pathFlag          = flag.String("path", "", "Only count commits that touched files under this path")
		latestFlag        = flag.Bool("latest", false, "Output the nearest tag reachable from HEAD without any generated suffix")
		targetFlag        = flag.String("target", "", "Output next semver bump from latest tag: dev (default), patch, minor, or major")
	)

	flag.Usage = printHelp
	flag.Parse()

	info, err := version.GetVersionInfo(*repoDirFlag, version.Options{
		DefaultBranch: *defaultBranchFlag,
		TagPrefix:     *prefixFlag,
		StripPrefix:   *stripPrefixFlag,
		FilterPath:    *pathFlag,
		Latest:        *latestFlag,
		Target:        *targetFlag,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if *shortFlag {
		fmt.Println(info.Version)
	} else if *detailedFlag {
		fmt.Println(info.DetailedString())
	} else {
		fmt.Println(info.Version)
	}
}
