/*
Copyright © 2024 IceRinkDev
*/
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "optager",
	Short: "A manager for your /opt/ folder",
	Long: `optager is a CLI application that helps you manage your /opt/ folder.
It enables you to install binaries from archives (e.g. .tar.gz) and
make them available in the command line. It also helps you to completely
remove them.`,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {}
