package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/forgant-foundry/vergant/internal/config"
	"github.com/forgant-foundry/vergant/internal/git"
	"github.com/forgant-foundry/vergant/internal/tool"
)

const defaultConfigFile = ".vergant.yml"

const usageText = `vergant — branch-driven semantic versioning

Usage:
  vergant <command> [flags]

Commands:
  last-version   print the last release or candidate version reachable from HEAD
  last-release   print the last release version reachable from HEAD
  new            calculate and apply the next version tag
  promote        promote a candidate version to release
  list           list all reachable version tags

Global flags (all commands):
  -config string   config file path (default ".vergant.yml")
  -no-fetch        skip fetching tags from remote

Command flags:
  new:     -dry-run   print the version without creating or pushing the tag
  promote: -dry-run   print the version without creating or pushing the tag
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usageText)
		os.Exit(1)
	}

	switch os.Args[1] {
	case "last-version":
		runLastVersion(os.Args[2:])
	case "last-release":
		runLastRelease(os.Args[2:])
	case "new":
		runNew(os.Args[2:])
	case "promote":
		runPromote(os.Args[2:])
	case "list":
		runList(os.Args[2:])
	case "-h", "--help", "help":
		fmt.Print(usageText)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s", os.Args[1], usageText)
		os.Exit(1)
	}
}

// globalFlags registers -config and -no-fetch on fs, which are shared by every
// subcommand. Each subcommand creates its own FlagSet and calls this so that
// global flags may appear after the subcommand name: vergant new -config x.json
func globalFlags(fs *flag.FlagSet) (configFile *string, noFetch *bool) {
	configFile = fs.String("config", defaultConfigFile, "config file path")
	noFetch = fs.Bool("no-fetch", false, "skip fetching tags from remote")
	return
}

func newTool(configFile string) (*tool.VersioningTool, error) {
	cfg, err := config.Load(configFile)
	if err != nil {
		return nil, err
	}
	return tool.New(cfg, git.New(".")), nil
}

func mayFetch(t *tool.VersioningTool, noFetch bool) error {
	if noFetch {
		return nil
	}
	return t.FetchTags()
}

func die(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runLastVersion(args []string) {
	fs := flag.NewFlagSet("last-version", flag.ExitOnError)
	configFile, noFetch := globalFlags(fs)
	die(fs.Parse(args))

	t, err := newTool(*configFile)
	die(err)
	die(mayFetch(t, *noFetch))
	v, err := t.LastVersion()
	die(err)
	if v != nil {
		fmt.Println(v.RenderCategorized())
	}
}

func runLastRelease(args []string) {
	fs := flag.NewFlagSet("last-release", flag.ExitOnError)
	configFile, noFetch := globalFlags(fs)
	die(fs.Parse(args))

	t, err := newTool(*configFile)
	die(err)
	die(mayFetch(t, *noFetch))
	v, err := t.LastRelease()
	die(err)
	if v != nil {
		fmt.Println(v.RenderCategorized())
	}
}

func runNew(args []string) {
	fs := flag.NewFlagSet("new", flag.ExitOnError)
	configFile, noFetch := globalFlags(fs)
	dryRun := fs.Bool("dry-run", false, "calculate version without pushing the tag")
	die(fs.Parse(args))

	t, err := newTool(*configFile)
	die(err)
	die(mayFetch(t, *noFetch))
	v, err := t.NewVersion()
	die(err)
	if !*dryRun {
		die(t.Tag(v))
	}
	fmt.Println(v.RenderCategorized())
}

func runPromote(args []string) {
	fs := flag.NewFlagSet("promote", flag.ExitOnError)
	configFile, noFetch := globalFlags(fs)
	dryRun := fs.Bool("dry-run", false, "calculate version without pushing the tag")
	die(fs.Parse(args))

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "promote requires a version argument")
		os.Exit(1)
	}

	t, err := newTool(*configFile)
	die(err)
	die(mayFetch(t, *noFetch))
	v, err := t.Promote(fs.Arg(0))
	die(err)
	if !*dryRun {
		die(t.Tag(v))
	}
	fmt.Println(v.RenderCategorized())
}

func runList(args []string) {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	configFile, noFetch := globalFlags(fs)
	die(fs.Parse(args))

	t, err := newTool(*configFile)
	die(err)
	die(mayFetch(t, *noFetch))
	tags, err := t.ListTags()
	die(err)
	fmt.Print(tags)
}
