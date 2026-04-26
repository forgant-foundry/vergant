package main

import (
	"fmt"
	"os"

	"github.com/forgant-foundry/vergant/internal/config"
	"github.com/forgant-foundry/vergant/internal/git"
	"github.com/forgant-foundry/vergant/internal/tool"
	"github.com/spf13/cobra"
)

const defaultConfigFile = "vergant.config.json"

func main() {
	var configFile string
	var noFetch bool
	var dryRun bool

	newTool := func() (*tool.VersioningTool, error) {
		cfg, err := config.Load(configFile)
		if err != nil {
			return nil, err
		}
		return tool.New(cfg, git.New(".")), nil
	}

	mayFetch := func(t *tool.VersioningTool) error {
		if noFetch {
			return nil
		}
		return t.FetchTags()
	}

	root := &cobra.Command{
		Use:   "vergant",
		Short: "Git-based semantic versioning tool",
	}
	root.PersistentFlags().StringVarP(&configFile, "config", "c", defaultConfigFile, "config file path")
	root.PersistentFlags().BoolVar(&noFetch, "no-fetch", false, "skip fetching tags from remote")

	lastVersionCmd := &cobra.Command{
		Use:   "last-version",
		Short: "Print the last release or candidate version",
		RunE: func(cmd *cobra.Command, args []string) error {
			t, err := newTool()
			if err != nil {
				return err
			}
			if err := mayFetch(t); err != nil {
				return err
			}
			v, err := t.LastVersion()
			if err != nil {
				return err
			}
			if v != nil {
				fmt.Println(v.RenderCategorized())
			}
			return nil
		},
	}

	lastReleaseCmd := &cobra.Command{
		Use:   "last-release",
		Short: "Print the last release version",
		RunE: func(cmd *cobra.Command, args []string) error {
			t, err := newTool()
			if err != nil {
				return err
			}
			if err := mayFetch(t); err != nil {
				return err
			}
			v, err := t.LastRelease()
			if err != nil {
				return err
			}
			if v != nil {
				fmt.Println(v.RenderCategorized())
			}
			return nil
		},
	}

	newCmd := &cobra.Command{
		Use:   "new",
		Short: "Calculate and push a new version tag",
		RunE: func(cmd *cobra.Command, args []string) error {
			t, err := newTool()
			if err != nil {
				return err
			}
			if err := mayFetch(t); err != nil {
				return err
			}
			v, err := t.NewVersion()
			if err != nil {
				return err
			}
			if !dryRun {
				if err := t.Tag(v); err != nil {
					return err
				}
			}
			fmt.Println(v.RenderCategorized())
			return nil
		},
	}
	newCmd.Flags().BoolVar(&dryRun, "dry-run", false, "calculate version without pushing the tag")

	promoteCmd := &cobra.Command{
		Use:   "promote <version>",
		Short: "Promote a candidate version to release",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			t, err := newTool()
			if err != nil {
				return err
			}
			if err := mayFetch(t); err != nil {
				return err
			}
			v, err := t.Promote(args[0])
			if err != nil {
				return err
			}
			if !dryRun {
				if err := t.Tag(v); err != nil {
					return err
				}
			}
			fmt.Println(v.RenderCategorized())
			return nil
		},
	}
	promoteCmd.Flags().BoolVar(&dryRun, "dry-run", false, "calculate version without pushing the tag")

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all reachable version tags",
		RunE: func(cmd *cobra.Command, args []string) error {
			t, err := newTool()
			if err != nil {
				return err
			}
			if err := mayFetch(t); err != nil {
				return err
			}
			tags, err := t.ListTags()
			if err != nil {
				return err
			}
			fmt.Print(tags)
			return nil
		},
	}

	root.AddCommand(lastVersionCmd, lastReleaseCmd, newCmd, promoteCmd, listCmd)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
