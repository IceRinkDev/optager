/*
Copyright © 2025 IceRinkDev
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var version string = "git"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Prints the installed version of optager",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
