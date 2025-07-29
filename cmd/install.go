/*
Copyright © 2024 IceRinkDev
*/
package cmd

import (
	"archive/tar"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

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
		wait := make(chan bool)
		ctx, cancelProcessingDots := context.WithCancel(context.Background())
		go func(ctx context.Context) {
			fmt.Print("Analyzing archive...")
			for {
				select {
				case <-ctx.Done():
					fmt.Println()
					wait <- false
					return
				case <-time.After(500 * time.Millisecond):
					fmt.Print(".")
				}
			}
		}(ctx)

		newPkg, shouldCreateFolder, err := gatherPackageInfo(args[0])

		// Cancel the goroutine that prints dots and wait for it to finish
		cancelProcessingDots()
		<-wait
		close(wait)

		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		if name, _ := cmd.Flags().GetString("name"); name != "" {
			newPkg.Name = name
		}

		extractLocation := "/opt/"
		if shouldCreateFolder {
			extractLocation = filepath.Join("/opt/", newPkg.FolderName)
			err = exec.Command("sudo", "mkdir", "-p", extractLocation).Run()
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error: could not create folder in /opt/")
				os.Exit(1)
			}
		} else {
			for i := range len(newPkg.Binaries) {
				newPkg.Binaries[i] = strings.TrimPrefix(newPkg.Binaries[i], newPkg.FolderName+"/")
			}
		}

		err = exec.Command("sudo", "tar", "-xf", args[0], "-C", extractLocation).Run()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error: could not extract the archive")
			os.Exit(1)
		}

		var linkedBinaries []string
		if global, _ := cmd.Flags().GetBool("global"); global {
			newPkg.Global = true
			linkedBinaries = symlinkToRoot(newPkg)
		} else {
			linkedBinaries = symlinkToUser(newPkg)
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
			pkgPath := filepath.Join("/opt/", newPkg.FolderName)
			_, err := os.Lstat(pkgPath)
			if err == nil {
				exec.Command("sudo", "rm", "-rf", pkgPath).Run()
			}
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

func symlinkToUser(pkg *storage.Pkg) []string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: no home folder found")
		return nil
	}
	localBin := filepath.Join(homeDir, ".local", "bin")
	err = os.MkdirAll(localBin, 0755)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: could not create folder", localBin)
		return nil
	}

	return symlink(pkg, localBin, false)
}

func symlinkToRoot(pkg *storage.Pkg) []string {
	return symlink(pkg, "/usr/local/bin/", true)
}

func symlink(pkg *storage.Pkg, destPath string, sudo bool) (linkedBinaries []string) {
	for _, binary := range pkg.Binaries {
		var symlinkCmd *exec.Cmd

		linkSrcPath := filepath.Join("/opt/", pkg.FolderName, binary)
		linkDestPath := filepath.Join(destPath, filepath.Base(binary))

		if sudo {
			symlinkCmd = exec.Command("sudo", "ln", "-s", linkSrcPath, linkDestPath)
		} else {
			symlinkCmd = exec.Command("ln", "-s", linkSrcPath, linkDestPath)
		}
		err := symlinkCmd.Run()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error: could not link binary", binary, "into", destPath)
		} else {
			linkedBinaries = append(linkedBinaries, filepath.Base(binary))
		}
	}
	return
}

func gatherPackageInfo(path string) (*storage.Pkg, bool, error) {
	archive, err := os.Open(path)
	if err != nil {
		return nil, false, fmt.Errorf("Error: could not access archive")
	}

	var format archives.Decompressor
	switch filepath.Ext(path) {
	case ".gz":
		format = archives.Gz{}
	case ".xz":
		format = archives.Xz{}
	default:
		return nil, false, fmt.Errorf("Error: archive not supported")
	}

	decompressedStream, err := format.OpenReader(archive)
	if err != nil {
		return nil, false, fmt.Errorf("Error: could not read from archive")
	}

	tarReader := tar.NewReader(decompressedStream)

	result := storage.Pkg{}
	foldername := ""
	binaries := make([]string, 0)

	potentialBinaries := make(map[string]string)
	rootFolderCount := 0
	binDirPath := ""
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
				} else if levels == 1 && filepath.Base(header.Name) == "bin" {
					binDirPath = filepath.Clean(header.Name)
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

	if rootFolderCount > 1 {
		return nil, false, fmt.Errorf("Error: archive contains %d folders and thus is not supposed to be installed into /opt/", rootFolderCount)
	}

	if binDirPath != "" {
		i := 0
		for _, binPath := range binaries {
			if strings.HasPrefix(binPath, binDirPath) {
				binaries[i] = binPath
				i++
			}
		}
		binaries = binaries[:i]
	}

	if len(binaries) >= 1 {
		result.Binaries = binaries
	} else {
		return nil, false, fmt.Errorf("Error: archive contains no binaries")
	}

	shouldCreatePkgFolder := false
	if foldername == "" {
		shouldCreatePkgFolder = true
		if len(binaries) == 1 {
			foldername = binaries[0]
		} else if len(binaries) > 1 {
			archiveName := filepath.Base(path)
			ext := filepath.Ext(archiveName)
			// Loop to trim multiple extensions like .tar.gz
			for ext != "" {
				archiveName = strings.TrimSuffix(archiveName, ext)
				ext = filepath.Ext(archiveName)
			}
			foldername = archiveName
		}
	}
	result.FolderName = foldername

	return &result, shouldCreatePkgFolder, nil
}

func isExecutable(header *tar.Header, reader *tar.Reader) bool {
	// Check if file has no execute permission set
	if header.FileInfo().Mode()&0111 == 0 {
		return false
	}

	switch filepath.Ext(header.Name) {
	case ".sh":
		return true
	default:
		magicNumbers := make([]byte, 52)
		count, err := reader.Read(magicNumbers)
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
	}
	return false
}

func mapContains[K comparable, V any](m map[K]V, k K) bool {
	_, ok := m[k]
	return ok
}
