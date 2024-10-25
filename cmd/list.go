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
		fmt.Println(storage.New())
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
