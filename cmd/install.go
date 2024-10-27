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
	"path/filepath"
	"strings"

	"github.com/IceRinkDev/optager/internal/storage"
	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install PATH-TO-ARCHIVE",
	Short: "Install an archive into /opt/",
	Long: `optager install lets you install an archive into /opt/.
By default it will also symlink the binaries to ~/.local/bin/.`,
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
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error: problem inspecting the archive")
			os.Exit(1)
		}

		var newPkg storage.Pkg

		outstr := string(output)
		folderNames := strings.Split(strings.TrimSpace(outstr), "\n")
		if len(folderNames) > 1 {
			fmt.Println("This would extract the following folders and files into /opt/:")
			for _, folderName := range folderNames {
				fmt.Println("\t", folderName)
			}
			fmt.Println("This archive is most likely not supposed to be installed into /opt/")
			os.Exit(1)
		} else if len(folderNames) == 1 {
			newPkg = storage.Pkg{FolderName: strings.Trim(folderNames[0], "/")}
		} else {
			fmt.Fprintln(os.Stderr, "Error: archive is empty")
			os.Exit(1)
		}

		if name, _ := cmd.Flags().GetString("name"); name != "" {
			newPkg.Name = name
		}

		err = exec.Command("sudo", "tar", "-xf", args[0], "-C", "/opt/").Run()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error: could not extract the archive")
			os.Exit(1)
		}

		var linkedBinaries []string
		if global, _ := cmd.Flags().GetBool("global"); global {
			newPkg.Global = true
			linkedBinaries = symlinkToRoot(newPkg.FolderName)
		} else {
			linkedBinaries = symlinkToUser(newPkg.FolderName)
		}
		if len(linkedBinaries) > 0 {
			newPkg.Binaries = linkedBinaries
			fmt.Println("Successfully installed the package")
			fmt.Print("You can now use ")
			for i, binary := range linkedBinaries {
				switch i {
				case 0:
					fmt.Printf("\"%s\"", binary)
				case len(linkedBinaries) - 1:
					fmt.Printf(" and \"%s\"", binary)
				default:
					fmt.Printf(", \"%s\"", binary)
				}
			}
			fmt.Println(" in the command line")
		} else {
			fmt.Println("No binaries usable")
		}
		xdgStorage := storage.New()
		xdgStorage.AddPkg(newPkg)
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
	installCmd.Flags().BoolP("global", "g", false, "symlink binaries to /usr/local/bin/")
	installCmd.Flags().StringP("name", "n", "", "set name for the package")
}

func symlinkToUser(folder string) []string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: no home folder found")
		return nil
	}
	localBin := filepath.Join(homeDir, ".local", "bin")
	err = os.MkdirAll(localBin, 0775)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: could not create folder", localBin)
		return nil
	}

	pkgBinPath := filepath.Join("/opt", folder, "bin")
	return symlink(pkgBinPath, localBin, false)
}

func symlinkToRoot(folder string) []string {
	pkgBinPath := filepath.Join("/opt", folder, "bin")
	return symlink(pkgBinPath, "/usr/local/bin/", true)
}

func symlink(srcPath, destPath string, sudo bool) (linkedBinaries []string) {
	pkgBinDir, err := os.Open(srcPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			fmt.Fprintln(os.Stderr, "Error: installed package has no bin/ folder")
		} else {
			fmt.Fprintln(os.Stderr, "Error: could not access", srcPath)
		}
		return
	}
	binaries, err := pkgBinDir.Readdirnames(0)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: could not read the contents of the", srcPath, "folder")
		return
	}
	for _, binary := range binaries {
		var symlinkCmd *exec.Cmd
		if sudo {
			symlinkCmd = exec.Command("sudo", "ln", "-s", filepath.Join(srcPath, binary), filepath.Join(destPath, binary))
		} else {
			symlinkCmd = exec.Command("ln", "-s", filepath.Join(srcPath, binary), filepath.Join(destPath, binary))
		}
		err := symlinkCmd.Run()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error: could not link binary", binary, "into", destPath)
		} else {
			linkedBinaries = append(linkedBinaries, binary)
		}
	}
	return
}
