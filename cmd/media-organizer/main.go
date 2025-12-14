package main

import (
	"os"

	"github.com/quidome/media-organizer-go/pkg/scan"
	"github.com/spf13/cobra"
)

const version = "0.1.0"

type options struct {
	verbose bool
	dryRun  bool
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
			if opts.dryRun {
				cmd.Println("Dry run mode: enabled")
			}
			cmd.Println("")
			cmd.Println("Use --help to see available commands and options")
		},
	}

	rootCmd.SetOut(os.Stdout)
	rootCmd.SetErr(os.Stderr)

	rootCmd.PersistentFlags().BoolVarP(&opts.verbose, "verbose", "v", false, "enable verbose output")
	rootCmd.PersistentFlags().BoolVarP(&opts.dryRun, "dry-run", "n", false, "perform a dry run without making changes")

	rootCmd.AddCommand(newOrganizeCmd(opts))
	rootCmd.AddCommand(newScanCmd(opts))

	return rootCmd
}

func newOrganizeCmd(opts *options) *cobra.Command {
	return &cobra.Command{
		Use:   "organize [source] [destination]",
		Short: "Organize media files from source to destination",
		Long:  "Organize media files from a source directory to a destination directory based on their metadata (date taken, camera model, etc.)",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			source := args[0]
			dest := args[1]

			cmd.Println("Organizing media files...")
			cmd.Printf("Source: %s\n", source)
			cmd.Printf("Destination: %s\n", dest)

			if opts.verbose {
				cmd.Println("Verbose mode: enabled")
			}
			if opts.dryRun {
				cmd.Println("Dry run mode: No files will be moved")
			}

			cmd.Println("")
			cmd.Println("Organization logic not yet implemented")
		},
	}
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
