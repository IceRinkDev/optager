/*
Copyright © 2024 IceRinkDev
*/
package cmd

import (
	"fmt"

	"github.com/IceRinkDev/optager/internal/storage"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all installed packages",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if verbose, _ := cmd.Flags().GetBool("verbose"); verbose {
			fmt.Println(storage.New().DetailedString())
		} else {
			fmt.Println(storage.New())
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().BoolP("verbose", "v", false, "make list verbose")
}
