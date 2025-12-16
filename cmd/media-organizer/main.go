package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/quidome/media-organizer-go/pkg/copy"
	"github.com/quidome/media-organizer-go/pkg/createdat"
	"github.com/quidome/media-organizer-go/pkg/plan"
	"github.com/quidome/media-organizer-go/pkg/reconcile"
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
	var jsonOutput bool

	organizeCmd := &cobra.Command{
		Use:   "organize [source] [destination]",
		Short: "Organize media files from source to destination",
		Long:  "Organize media files from a source directory to a destination directory based on their metadata.",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			source := args[0]
			destination := args[1]

			fsys := os.DirFS(source)
			scanOpts := scan.DefaultOptions()

			records, err := scan.ScanRecords(fsys, ".", scanOpts)
			if err != nil {
				return err
			}

			// Stage 2: Determine created_at for each file
			orderedSources := make([]string, 0, len(records))
			sources := make([]string, 0, len(records))
			sourceSizes := make(map[string]int64, len(records))
			sourceModTimes := make(map[string]time.Time, len(records))
			bestCreatedAt := make(map[string]time.Time)
			detailedBySource := make(map[string]createdat.DetailedResult)
			decisionsBySource := make(map[string]reconcile.Decision)

			for _, record := range records {
				sourceAbs := filepath.Join(source, filepath.FromSlash(record.Path))
				orderedSources = append(orderedSources, sourceAbs)
				sources = append(sources, sourceAbs)
				sourceSizes[sourceAbs] = record.FileSizeBytes
				sourceModTimes[sourceAbs] = record.ModTime

				detailed, err := createdat.DetermineDetailed(fsys, record.Path, createdat.Options{Location: time.Local})
				if err != nil {
					return err
				}
				detailedBySource[sourceAbs] = detailed

				if !detailed.Best.CreatedAt.IsZero() {
					bestCreatedAt[sourceAbs] = detailed.Best.CreatedAt
				}
			}

			// Stage 4b: Deduplicate sources (choose oldest per exact-content group)
			kept, dedupeDecisions, err := reconcile.DedupeSources(sources, detailedBySource, sourceSizes)
			if err != nil {
				return err
			}
			for _, d := range dedupeDecisions {
				decisionsBySource[d.SourcePath] = d
			}

			// Stage 3 & 4: Plan destinations for kept sources
			plannedOps, err := reconcile.PlanDestinations(destination, kept, bestCreatedAt)
			if err != nil {
				return err
			}

			// Stage 4c: Reconcile against destination filesystem
			destDecisions, err := reconcile.ResolveAgainstDestination(plannedOps)
			if err != nil {
				return err
			}
			for _, d := range destDecisions {
				// Do not override source-duplicate decisions.
				if existing, ok := decisionsBySource[d.SourcePath]; ok && existing.Action == reconcile.ActionSkippedDuplicateSrc {
					continue
				}
				decisionsBySource[d.SourcePath] = d
			}

			decisions := make([]reconcile.Decision, 0, len(orderedSources))
			for _, src := range orderedSources {
				if d, ok := decisionsBySource[src]; ok {
					decisions = append(decisions, d)
				}
			}

			if execute {
				// Copy only actions that require copying.
				opsToCopy := make([]plan.Operation, 0)
				for _, d := range decisions {
					if d.Action == reconcile.ActionCopy || d.Action == reconcile.ActionCopyRenamed {
						final := d.FinalDestinationPath
						if final == "" {
							final = d.DestinationPath
						}
						opsToCopy = append(opsToCopy, plan.Operation{SourcePath: d.SourcePath, DestinationPath: final})
					}
				}

				results, err := copy.Execute(opsToCopy, copy.Options{Overwrite: false})
				if err != nil {
					return err
				}
				resultBySource := make(map[string]copy.Result, len(results))
				for _, r := range results {
					resultBySource[r.Operation.SourcePath] = r
				}

				for i := range decisions {
					d := decisions[i]
					if d.Action != reconcile.ActionCopy && d.Action != reconcile.ActionCopyRenamed {
						continue
					}
					r, ok := resultBySource[d.SourcePath]
					if !ok {
						decisions[i].Action = reconcile.ActionFailed
						decisions[i].Error = fmt.Errorf("missing copy result")
						continue
					}
					if r.Success {
						if d.Action == reconcile.ActionCopyRenamed {
							decisions[i].Action = reconcile.ActionCopiedRenamed
						} else {
							decisions[i].Action = reconcile.ActionCopied
						}
					} else {
						decisions[i].Action = reconcile.ActionFailed
						decisions[i].Error = r.Error
					}
				}
			}

			if jsonOutput {
				return printJSONDecisions(cmd, decisions, detailedBySource, sourceSizes, sourceModTimes)
			}

			// Text output
			successCount := 0
			for _, d := range decisions {
				switch d.Action {
				case reconcile.ActionCopied, reconcile.ActionCopiedRenamed:
					successCount++
					fmt.Fprintf(cmd.OutOrStdout(), "copied %s -> %s\n", d.SourcePath, d.FinalDestinationPath)
				case reconcile.ActionCopy, reconcile.ActionCopyRenamed:
					fmt.Fprintf(cmd.OutOrStdout(), "%s -> %s\n", d.SourcePath, d.FinalDestinationPath)
				case reconcile.ActionSkippedIdentical:
					successCount++
					fmt.Fprintf(cmd.OutOrStdout(), "skipped %s -> %s (identical)\n", d.SourcePath, d.FinalDestinationPath)
				case reconcile.ActionSkippedDuplicateSrc:
					successCount++
					fmt.Fprintf(cmd.OutOrStdout(), "skipped %s (duplicate of %s)\n", d.SourcePath, d.DuplicateOf)
				case reconcile.ActionFailed:
					fmt.Fprintf(cmd.OutOrStderr(), "failed %s: %v\n", d.SourcePath, d.Error)
				default:
					fmt.Fprintf(cmd.OutOrStderr(), "failed %s: unknown action\n", d.SourcePath)
				}
			}

			if opts.verbose {
				cmd.PrintErrf("processed %d of %d files\n", successCount, len(decisions))
			}

			return nil
		},
	}

	organizeCmd.Flags().BoolVarP(&execute, "execute", "x", false, "execute copy operations (default: dry-run)")
	organizeCmd.Flags().BoolVar(&jsonOutput, "json", false, "output operations as JSON")

	return organizeCmd
}

type jsonCreatedAt struct {
	Metadata string `json:"metadata,omitempty"`
	Filename string `json:"filename,omitempty"`
	Filestat string `json:"filestat,omitempty"`
}

type jsonOperation struct {
	SourcePath      string        `json:"source_path"`
	CreatedAt       jsonCreatedAt `json:"created_at"`
	FileSizeBytes   int64         `json:"file_size_bytes"`
	ModTime         time.Time     `json:"mod_time"`
	DestinationPath string        `json:"destination_path,omitempty"`

	Action               string `json:"action,omitempty"`
	FinalDestinationPath string `json:"final_destination_path,omitempty"`
	DuplicateOf          string `json:"duplicate_of,omitempty"`
	Error                string `json:"error,omitempty"`
}

func printJSONDecisions(cmd *cobra.Command, decisions []reconcile.Decision, detailedResults map[string]createdat.DetailedResult, sizes map[string]int64, modTimes map[string]time.Time) error {
	jsonOps := make([]jsonOperation, 0, len(decisions))

	for _, d := range decisions {
		detailed := detailedResults[d.SourcePath]

		createdAt := jsonCreatedAt{}
		if !detailed.Metadata.IsZero() {
			createdAt.Metadata = detailed.Metadata.Format(time.RFC3339)
		}
		if !detailed.Filename.IsZero() {
			createdAt.Filename = detailed.Filename.Format(time.RFC3339)
		}
		if !detailed.Filestat.IsZero() {
			createdAt.Filestat = detailed.Filestat.Format(time.RFC3339)
		}

		jsonOp := jsonOperation{
			SourcePath:      d.SourcePath,
			CreatedAt:       createdAt,
			FileSizeBytes:   sizes[d.SourcePath],
			ModTime:         modTimes[d.SourcePath],
			DestinationPath: d.DestinationPath,
			Action:          string(d.Action),
			DuplicateOf:     d.DuplicateOf,
		}
		if d.FinalDestinationPath != "" && d.FinalDestinationPath != d.DestinationPath {
			jsonOp.FinalDestinationPath = d.FinalDestinationPath
		}
		if d.Error != nil {
			jsonOp.Error = d.Error.Error()
		}

		jsonOps = append(jsonOps, jsonOp)
	}

	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(jsonOps)
}

func newScanCmd(opts *options) *cobra.Command {
	var maxDepth int
	var jsonOutput bool

	scanCmd := &cobra.Command{
		Use:   "scan [directory]",
		Short: "Scan a directory for media files",
		Long:  "Scan a directory and print all media files found (relative to the scan root).",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			directory := args[0]

			scanOpts := scan.DefaultOptions()
			scanOpts.MaxDepth = maxDepth

			records, err := scan.ScanRecords(os.DirFS(directory), ".", scanOpts)
			if err != nil {
				return err
			}

			if jsonOutput {
				// Enrich scan records with created_at candidates.
				type scanJSONRecord struct {
					SourcePath    string        `json:"source_path"`
					CreatedAt     jsonCreatedAt `json:"created_at"`
					FileSizeBytes int64         `json:"file_size_bytes"`
					ModTime       time.Time     `json:"mod_time"`
				}

				out := make([]scanJSONRecord, 0, len(records))
				fsys := os.DirFS(directory)
				for _, record := range records {
					detailed, err := createdat.DetermineDetailed(fsys, record.Path, createdat.Options{Location: time.Local})
					if err != nil {
						return err
					}

					createdAt := jsonCreatedAt{}
					if !detailed.Metadata.IsZero() {
						createdAt.Metadata = detailed.Metadata.Format(time.RFC3339)
					}
					if !detailed.Filename.IsZero() {
						createdAt.Filename = detailed.Filename.Format(time.RFC3339)
					}
					if !detailed.Filestat.IsZero() {
						createdAt.Filestat = detailed.Filestat.Format(time.RFC3339)
					}

					out = append(out, scanJSONRecord{
						SourcePath:    filepath.Join(directory, filepath.FromSlash(record.Path)),
						CreatedAt:     createdAt,
						FileSizeBytes: record.FileSizeBytes,
						ModTime:       record.ModTime,
					})
				}

				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(out)
			}

			for _, record := range records {
				cmd.Println(record.Path)
			}

			if opts.verbose {
				cmd.PrintErrf("found %d media files\n", len(records))
			}

			return nil
		},
	}

	scanCmd.Flags().IntVar(&maxDepth, "max-depth", -1, "maximum recursion depth (0 = no recursion)")
	scanCmd.Flags().BoolVar(&jsonOutput, "json", false, "output records as JSON")

	return scanCmd
}
