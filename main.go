package main

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
)

var version = "1.0.0"

func main() {
	var rootCmd = &cobra.Command{
		Use:   "ddnswitch",
		Short: "Switch between different versions of DDN CLI",
		Long: `DDN CLI Switcher allows you to easily switch between different versions of the DDN CLI.
Similar to tfswitch for Terraform, this tool helps manage multiple DDN CLI versions.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				// Interactive mode - show available versions
				if err := listAndSelectVersion(); err != nil {
					log.Fatalf("Error: %v", err)
				}
			} else {
				// Direct version specification
				targetVersion := args[0]
				if err := switchToVersion(targetVersion); err != nil {
					log.Fatalf("Error switching to version %s: %v", targetVersion, err)
				}
			}
		},
	}

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

	rootCmd.AddCommand(listCmd, installCmd, currentCmd, versionCmd, uninstallCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}