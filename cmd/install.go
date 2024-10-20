/*
Copyright © 2024 IceRinkDev
*/
package cmd

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install [path-to-archive]",
	Short: "Install an archive into /opt/",
	Args: func(cmd *cobra.Command, args []string) error {
		err := cobra.ExactArgs(1)(cmd, args)
		if err != nil {
			return fmt.Errorf("please specify the path to the archive you want to install")
		}
		fileInfo, err := os.Stat(args[0])
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return fmt.Errorf("specified file does not exist")
			} else {
				return err
			}
		} else if fileInfo.IsDir() {
			return fmt.Errorf("specified path is a directory")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		err := exec.Command("sudo", "tar", "-xf", args[0], "-C", "/opt/").Run()
		if err != nil {
			log.Fatalln("could not extract the archive")
		}
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
}
