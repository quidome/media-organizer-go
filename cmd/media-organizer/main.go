package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	version = "0.1.0"
	verbose bool
	dryRun  bool
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "media-organizer",
	Short: "A CLI tool to organize media files",
	Long: `Media Organizer is a command-line tool that helps you organize 
your media files (photos, videos) based on metadata like date, location, etc.`,
	Version: version,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Media Organizer CLI")
		fmt.Printf("Version: %s\n", version)
		if verbose {
			fmt.Println("Verbose mode: enabled")
		}
		if dryRun {
			fmt.Println("Dry run mode: enabled")
		}
		fmt.Println("\nUse --help to see available commands and options")
	},
}

var organizeCmd = &cobra.Command{
	Use:   "organize [source] [destination]",
	Short: "Organize media files from source to destination",
	Long: `Organize media files from a source directory to a destination directory
based on their metadata (date taken, camera model, etc.)`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		source := args[0]
		dest := args[1]
		
		fmt.Printf("Organizing media files...\n")
		fmt.Printf("Source: %s\n", source)
		fmt.Printf("Destination: %s\n", dest)
		
		if verbose {
			fmt.Println("Verbose mode: enabled")
		}
		if dryRun {
			fmt.Println("Dry run mode: No files will be moved")
		}
		
		// TODO: Implement actual organization logic
		fmt.Println("\nOrganization logic not yet implemented")
	},
}

var scanCmd = &cobra.Command{
	Use:   "scan [directory]",
	Short: "Scan a directory for media files",
	Long:  `Scan a directory and report statistics about media files found`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		directory := args[0]
		
		fmt.Printf("Scanning directory: %s\n", directory)
		
		if verbose {
			fmt.Println("Verbose mode: enabled")
		}
		
		// TODO: Implement scanning logic
		fmt.Println("\nScanning logic not yet implemented")
	},
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")
	rootCmd.PersistentFlags().BoolVarP(&dryRun, "dry-run", "n", false, "perform a dry run without making changes")
	
	// Add subcommands
	rootCmd.AddCommand(organizeCmd)
	rootCmd.AddCommand(scanCmd)
}
