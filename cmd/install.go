/*
Copyright © 2024 IceRinkDev
*/
package cmd

import (
	"archive/tar"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/IceRinkDev/optager/internal/storage"
	"github.com/mholt/archives"
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
		newPkg, err := gatherPackageInfo(args[0])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
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
		xdgStorage.AddPkg(*newPkg)
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

func gatherPackageInfo(path string) (*storage.Pkg, error) {
	archive, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("Error: could not access archive")
	}

	var format archives.Decompressor
	switch filepath.Ext(path) {
	case ".gz":
		format = archives.Gz{}
	case ".xz":
		format = archives.Xz{}
	default:
		return nil, fmt.Errorf("Error: archive not supported")
	}

	decompressedStream, err := format.OpenReader(archive)
	if err != nil {
		return nil, fmt.Errorf("Error: could not read from archive")
	}

	tarReader := tar.NewReader(decompressedStream)

	result := storage.Pkg{}
	foldername := ""
	binaries := make([]string, 0)

	potentialBinaries := make(map[string]string)
	rootFolderCount := 0
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalln(err)
		}

		levels := 0
		parentDir := filepath.Dir(filepath.Clean(header.Name))
		if parentDir != "." {
			levels = len(strings.Split(parentDir, "/"))
		}
		if levels <= 2 || mapContains(potentialBinaries, header.Name) {
			switch header.Typeflag {
			case tar.TypeDir:
				if levels == 0 {
					foldername = filepath.Clean(header.Name)
					rootFolderCount++
				}
			case tar.TypeReg:
				if isExecutable(header, tarReader) {
					if link, ok := potentialBinaries[header.Name]; ok {
						binaries = append(binaries, link)
					} else {
						binaries = append(binaries, header.Name)
					}
				}
			case tar.TypeSymlink:
				linkPath := filepath.Join(filepath.Dir(header.Name), header.Linkname)
				potentialBinaries[linkPath] = header.Name
			default:
				fmt.Fprintf(os.Stderr, "Error: unknown filetype: %b in %s\n",
					header.Typeflag, header.Name)
			}
		}
	}

	if rootFolderCount != 1 {
		return nil, fmt.Errorf("Error: archive contains %d folders and thus is not supposed to be installed into /opt/", rootFolderCount)
	}

	if len(binaries) >= 1 {
		result.Binaries = binaries
	} else {
		return nil, fmt.Errorf("Error: archive contains no binaries")
	}

	if foldername == "" && len(binaries) == 1 {
		foldername = binaries[0]
	}
	result.FolderName = foldername

	return &result, nil
}

func isExecutable(h *tar.Header, r *tar.Reader) bool {
	if h.FileInfo().Mode()&0111 == 0 {
		// file has no execute permission set
		return false
	}

	switch filepath.Ext(h.Name) {
	case ".sh":
		return true
	case "", ".py", ".js":
		magicNumbers := make([]byte, 52)
		count, err := r.Read(magicNumbers)
		if err != nil && err != io.EOF {
			fmt.Fprintln(os.Stderr, "Error: could not open file")
			return false
		}

		// Check if it's an elf file
		if count >= 52 &&
			magicNumbers[0] == 0x7F && magicNumbers[1] == 0x45 &&
			magicNumbers[2] == 0x4C && magicNumbers[3] == 0x46 {
			return true
		}

		// Check if the file has a shebang
		if count >= 2 &&
			magicNumbers[0] == 0x23 && magicNumbers[1] == 0x21 {
			return true
		}
	default:
		return false
	}
	return false
}

func mapContains[K comparable, V any](m map[K]V, k K) bool {
	_, ok := m[k]
	return ok
}
