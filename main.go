package main

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
)

var version = "1.0.0"

// Keep this declaration and make it accessible to other files in the package
var includePrerelease bool

var (
	debugMode bool
)

func init() {
	// Add debug flag
	flag.BoolVar(&debugMode, "debug", false, "Enable debug logging")
}

func debugLog(format string, args ...interface{}) {
	if debugMode {
		fmt.Printf("[DEBUG] "+format+"\n", args...)
	}
}

func main() {
	var rootCmd = &cobra.Command{
		Use:   "ddnswitch [version]",
		Short: "Switch between different versions of DDN CLI",
		Long: `DDN CLI Switcher allows you to easily switch between different versions of the DDN CLI.
Similar to tfswitch for Terraform, this tool helps manage multiple DDN CLI versions.`,
		Args: cobra.ArbitraryArgs,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				// Interactive mode - show available versions
				if err := listAndSelectVersion(); err != nil {
					log.Fatalf("Error: %v", err)
				}
			} else {
				// Direct version specification
				targetVersion := args[0]
				fmt.Printf("Switching to DDN CLI version %s...\n", targetVersion)
				if err := switchToVersion(targetVersion); err != nil {
					log.Fatalf("Error switching to version %s: %v", targetVersion, err)
				}
			}
		},
	}

	// Add the prerelease flag to the root command
	rootCmd.PersistentFlags().BoolVar(&includePrerelease, "pre", false, "Include pre-release versions")

	var listCmd = &cobra.Command{
		Use:   "list",
		Short: "List all available DDN CLI versions",
		Run: func(cmd *cobra.Command, args []string) {
			if err := listAvailableVersions(); err != nil {
				log.Fatalf("Error listing versions: %v", err)
			}
		},
	}

	var installCmd = &cobra.Command{
		Use:   "install [version]",
		Short: "Install a specific version of DDN CLI",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			version := args[0]
			if err := installVersion(version); err != nil {
				log.Fatalf("Error installing version %s: %v", version, err)
			}
		},
	}

	var currentCmd = &cobra.Command{
		Use:   "current",
		Short: "Show currently active DDN CLI version",
		Run: func(cmd *cobra.Command, args []string) {
			if err := showCurrentVersion(); err != nil {
				log.Fatalf("Error getting current version: %v", err)
			}
		},
	}

	var versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Show ddnswitch version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("ddnswitch version %s\n", version)
		},
	}

	var uninstallCmd = &cobra.Command{
		Use:   "uninstall [version]",
		Short: "Uninstall a specific version of DDN CLI",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			version := args[0]
			if err := uninstallVersion(version); err != nil {
				log.Fatalf("Error uninstalling version %s: %v", version, err)
			}
		},
	}

	// Add subcommands
	rootCmd.AddCommand(listCmd, installCmd, currentCmd, versionCmd, uninstallCmd)

	// Execute the command
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
