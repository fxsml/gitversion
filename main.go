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
	fmt.Println("  -path <path>           Path to Git repository (default: .)")
	fmt.Println("  -default-branch <name> Default branch name (auto-detected if not set)")
	fmt.Println()
	fmt.Println("VERSION LOGIC:")
	fmt.Println("  - Default branch with tags:    Uses 'git describe' format (tag or tag-N-ghash)")
	fmt.Println("  - Default branch without tags: Uses '<branch-slug>-ghash'")
	fmt.Println("  - Other branches:              Always uses '<branch-slug>-ghash'")
	fmt.Println("  - Dirty tree:                  Appends '-YYYYMMDDHHMMSS' timestamp")
	fmt.Println()
	fmt.Println("EXAMPLES:")
	fmt.Println("  gitversion                         # Print version")
	fmt.Println("  gitversion -detailed               # Print detailed info")
	fmt.Println("  gitversion -path /repo             # Version for specific repo")
	fmt.Println("  gitversion -default-branch master  # Specify default branch")
}

func main() {
	// Check for help command first
	if len(os.Args) > 1 && os.Args[1] == "help" {
		printHelp()
		os.Exit(0)
	}

	var (
		detailedFlag      = flag.Bool("detailed", false, "Show detailed version information")
		shortFlag         = flag.Bool("short", false, "Show only the version string")
		pathFlag          = flag.String("path", ".", "Path to Git repository")
		defaultBranchFlag = flag.String("default-branch", "", "Default branch name (auto-detected if not set)")
	)

	flag.Usage = printHelp

	flag.Parse()

	info, err := version.GetVersionInfo(*pathFlag, *defaultBranchFlag)
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
