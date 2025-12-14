package main

import (
	"fmt"
	"os"
	"time"

	"github.com/quidome/media-organizer-go/pkg/createdat"
	"github.com/quidome/media-organizer-go/pkg/scan"
	"github.com/spf13/cobra"
)

const version = "0.1.0"

type options struct {
	verbose bool
}

func main() {
	cmd := newRootCmd()
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	opts := &options{}

	rootCmd := &cobra.Command{
		Use:     "media-organizer",
		Short:   "A CLI tool to organize media files",
		Long:    "Media Organizer is a command-line tool that helps you organize your media files (photos, videos) based on metadata like date, location, etc.",
		Version: version,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Println("Media Organizer CLI")
			cmd.Printf("Version: %s\n", version)
			if opts.verbose {
				cmd.Println("Verbose mode: enabled")
			}
			cmd.Println("")
			cmd.Println("Use --help to see available commands and options")
		},
	}

	rootCmd.SetOut(os.Stdout)
	rootCmd.SetErr(os.Stderr)

	rootCmd.PersistentFlags().BoolVarP(&opts.verbose, "verbose", "v", false, "enable verbose output")

	rootCmd.AddCommand(newOrganizeCmd(opts))
	rootCmd.AddCommand(newScanCmd(opts))

	return rootCmd
}

func newOrganizeCmd(opts *options) *cobra.Command {
	var execute bool

	organizeCmd := &cobra.Command{
		Use:   "organize [source] [destination]",
		Short: "Organize media files from source to destination",
		Long:  "Organize media files from a source directory to a destination directory based on their metadata.",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			source := args[0]
			_ = args[1] // destination wiring comes in later stages

			if execute {
				return fmt.Errorf("execute mode not implemented yet")
			}

			fsys := os.DirFS(source)
			scanOpts := scan.DefaultOptions()

			matches, err := scan.Scan(fsys, ".", scanOpts)
			if err != nil {
				return err
			}

			for _, match := range matches {
				res, err := createdat.Determine(fsys, match, createdat.Options{Location: time.Local})
				if err != nil {
					return err
				}

				createdAt := "unknown"
				if !res.CreatedAt.IsZero() {
					createdAt = res.CreatedAt.Format(time.RFC3339)
				}

				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\n", match, createdAt, res.Source)
			}

			if opts.verbose {
				cmd.PrintErrf("found %d media files\n", len(matches))
			}

			return nil
		},
	}

	organizeCmd.Flags().BoolVarP(&execute, "execute", "x", false, "execute copy operations (default: dry-run)")

	return organizeCmd
}

func newScanCmd(opts *options) *cobra.Command {
	var maxDepth int

	scanCmd := &cobra.Command{
		Use:   "scan [directory]",
		Short: "Scan a directory for media files",
		Long:  "Scan a directory and print all media files found (relative to the scan root).",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			directory := args[0]

			scanOpts := scan.DefaultOptions()
			scanOpts.MaxDepth = maxDepth

			matches, err := scan.Scan(os.DirFS(directory), ".", scanOpts)
			if err != nil {
				return err
			}

			for _, match := range matches {
				cmd.Println(match)
			}

			if opts.verbose {
				cmd.PrintErrf("found %d media files\n", len(matches))
			}

			return nil
		},
	}

	scanCmd.Flags().IntVar(&maxDepth, "max-depth", -1, "maximum recursion depth (0 = no recursion)")

	return scanCmd
}
