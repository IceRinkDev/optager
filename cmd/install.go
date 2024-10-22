/*
Copyright © 2024 IceRinkDev
*/
package cmd

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"strings"

	"github.com/IceRinkDev/optager/internal/storage"
	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install PATH-TO-ARCHIVE",
	Short: "Install an archive into /opt/",
	Args: func(cmd *cobra.Command, args []string) error {
		err := cobra.ExactArgs(1)(cmd, args)
		if err != nil {
			return fmt.Errorf("please specify the path to the archive you want to install")
		}
		fileInfo, err := os.Stat(args[0])
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return fmt.Errorf("specified path does not exist")
			} else {
				return err
			}
		} else if fileInfo.IsDir() {
			return fmt.Errorf("specified path is a directory")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		output, err := exec.Command("bash", "-c", fmt.Sprintf("tar --exclude=\"*/*\" -tf %s", args[0])).Output()
		if err == nil {
			outstr := string(output)
			folderNames := strings.Split(strings.TrimSpace(outstr), "\n")
			if len(folderNames) > 1 {
				fmt.Println("This would extract the following folders and files into /opt/:")
				for _, folderName := range folderNames {
					fmt.Println("\t", folderName)
				}
				fmt.Println("This archive is most likely not supposed to be installed into /opt/")
				os.Exit(1)
			}
			if len(folderNames) == 1 {
				xdgStorage := storage.New()
				xdgStorage.AddPkg(storage.Pkg{FolderName: folderNames[0]})
			} else {
				fmt.Println("Error: archive is empty")
				os.Exit(1)
			}
		}

		err = exec.Command("sudo", "tar", "-xf", args[0], "-C", "/opt/").Run()
		if err != nil {
			fmt.Println("Error: could not extract the archive")
		}
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
}
