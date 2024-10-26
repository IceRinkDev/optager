/*
Copyright © 2024 IceRinkDev
*/
package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/IceRinkDev/optager/internal/storage"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:     "remove PACKAGENAME",
	Aliases: []string{"uninstall"},
	Short:   "Remove a package from /opt/",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("please specify at least the name of one package you want to remove")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		dataStorage := storage.New()
		for _, arg := range args {
			pkg, err := dataStorage.GetPkg(arg)
			if err == nil {
				var baseBinDir string
				if pkg.Global {
					baseBinDir = "/usr/local/bin/"
				} else {
					homeDir, err := os.UserHomeDir()
					if err != nil {
						fmt.Println("Error: no home folder found")
						os.Exit(1)
					}
					baseBinDir = filepath.Join(homeDir, ".local", "bin")
				}
				for _, binary := range pkg.Binaries {
					binPath := filepath.Join(baseBinDir, binary)
					_, err := os.Lstat(binPath)
					if err == nil {
						err = exec.Command("sudo", "rm", binPath).Run()
						if err != nil {
							fmt.Println("Error: could not remove", binary, "from", baseBinDir)
						}
					}
				}

				pkgPath := filepath.Join("/opt/", pkg.FolderName)
				_, err := os.Lstat(pkgPath)
				if err == nil {
					err := exec.Command("sudo", "rm", "-rf", pkgPath).Run()
					if err != nil {
						fmt.Println("Error: could not remove package from /opt/")
					}
				}

				err = dataStorage.RemovePkgAt(pkg.Index)
				if err != nil {
					fmt.Println("Error: could not remove package from package-list")
				}
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)
}
