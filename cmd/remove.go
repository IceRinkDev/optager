/*
Copyright © 2024 IceRinkDev
*/
package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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

		removedPkgNames := make([]string, 0, len(args))

	argsLoop:
		for _, arg := range args {
			pkg, err := dataStorage.GetPkg(arg)
			if err == nil {
				var baseBinDir string
				if pkg.Global {
					baseBinDir = "/usr/local/bin/"
				} else {
					homeDir, err := os.UserHomeDir()
					if err != nil {
						fmt.Fprintln(os.Stderr, "Error: no home folder found")
						continue
					}
					baseBinDir = filepath.Join(homeDir, ".local", "bin")
				}
				for _, binary := range pkg.Binaries {
					binPath := filepath.Join(baseBinDir, binary)
					_, err := os.Lstat(binPath)
					if err == nil {
						err = exec.Command("sudo", "rm", binPath).Run()
						if err != nil {
							fmt.Fprintln(os.Stderr, "Error: could not remove", binary, "from", baseBinDir)
							continue argsLoop
						}
					}
				}

				pkgPath := filepath.Join("/opt/", pkg.FolderName)
				_, err := os.Lstat(pkgPath)
				if err == nil {
					err := exec.Command("sudo", "rm", "-rf", pkgPath).Run()
					if err != nil {
						fmt.Fprintln(os.Stderr, "Error: could not remove", pkgPath)
						continue
					}
				}

				err = dataStorage.RemovePkgAt(pkg.Index)
				if err != nil {
					fmt.Fprintln(os.Stderr, "Error: could not remove", arg, "from package-list")
					continue
				}
				removedPkgNames = append(removedPkgNames, pkg.String())
			}
		}
		if len(removedPkgNames) < 1 {
			fmt.Println("No packages removed")
		} else {
			sb := strings.Builder{}
			for i, pkgName := range removedPkgNames {
				switch i {
				case 0:
					sb.WriteString(`"` + pkgName + `"`)
				case len(removedPkgNames) - 1:
					sb.WriteString(` and "` + pkgName + `"`)
				default:
					sb.WriteString(`, "` + pkgName + `"`)
				}
			}
			fmt.Println("Successfully removed " + sb.String())
		}
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)
}
