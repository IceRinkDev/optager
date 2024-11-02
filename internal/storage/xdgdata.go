/*
Copyright © 2024 IceRinkDev
*/
package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type xdgDataStorage struct {
	filePath string
	packages []Pkg
}

type Pkg struct {
	FolderName string   `json:"folderName"`
	Name       string   `json:"name,omitempty"`
	Binaries   []string `json:"binaries,omitempty"`
	Global     bool     `json:"global,omitempty"`
}

func (p Pkg) String() string {
	var str string
	if p.Name != "" {
		str = fmt.Sprintf("%s (%s)", p.Name, p.FolderName)
	} else {
		str = p.FolderName
	}
	return str
}

func New() *xdgDataStorage {
	dataBaseDir := os.Getenv("XDG_DATA_HOME")
	if dataBaseDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error: no home folder found")
			os.Exit(1)
		}
		dataBaseDir = filepath.Join(homeDir, ".local", "share")
	}
	dataFilePath := filepath.Join(dataBaseDir, "optager", "pkgs.json")

	ds := xdgDataStorage{
		filePath: dataFilePath,
		packages: make([]Pkg, 0),
	}

	fileContent, err := os.ReadFile(dataFilePath)
	if err == nil {
		json.Unmarshal(fileContent, &ds.packages)
	}

	return &ds
}

type GetPkgResult struct {
	Pkg
	Index int
}

func (ds xdgDataStorage) GetPkg(name string) (GetPkgResult, error) {
	name = strings.ToLower(name)
	for i, pkg := range ds.packages {
		if name == strings.ToLower(pkg.Name) || name == strings.ToLower(pkg.FolderName) {
			return GetPkgResult{pkg, i}, nil
		}
	}
	return GetPkgResult{}, fmt.Errorf("package not found")
}

func (ds *xdgDataStorage) AddPkg(newPkg Pkg) {
	if !slices.ContainsFunc(ds.packages, func(p Pkg) bool {
		return p.FolderName == newPkg.FolderName
	}) {
		ds.packages = append(ds.packages, newPkg)
		ds.saveToFS()
	}
}

func (ds *xdgDataStorage) RemovePkgAt(index int) error {
	if index < 0 || index >= len(ds.packages) {
		return fmt.Errorf("index out of bounds")
	}
	ds.packages = append(ds.packages[:index], ds.packages[index+1:]...)
	ds.saveToFS()
	return nil
}

func (ds xdgDataStorage) String() string {
	return ds.printPackages(func(builder *strings.Builder, pkg Pkg, indent string) {
		builder.WriteString(indent + pkg.String() + "\n")
	})
}

func (ds xdgDataStorage) DetailedString() string {
	return ds.printPackages(func(builder *strings.Builder, pkg Pkg, indent string) {
		doubleIndent := indent + indent
		builder.WriteString(indent + pkg.String() + ":\n")
		builder.WriteString(doubleIndent + "Binaries: [")
		for i, binary := range pkg.Binaries {
			if i != len(pkg.Binaries)-1 {
				builder.WriteString(binary + ", ")
			} else {
				builder.WriteString(binary)
			}
		}
		builder.WriteString("]\n")
	})
}

func (ds xdgDataStorage) printPackages(f func(builder *strings.Builder, pkg Pkg, indent string)) string {
	indent := "   "
	sbGlobal := &strings.Builder{}
	sbLocal := &strings.Builder{}
	for _, pkg := range ds.packages {
		sb := sbLocal
		if pkg.Global {
			sb = sbGlobal
		}
		f(sb, pkg, indent)
	}
	strGlobal := sbGlobal.String()
	if strGlobal == "" {
		strGlobal = "Global: none"
	} else {
		strGlobal = "Global:\n" + strGlobal
	}
	strLocal := sbLocal.String()
	if strLocal == "" {
		strLocal = "Local: none\n"
	} else {
		strLocal = "Local:\n" + strLocal
	}
	return strings.TrimSpace(strGlobal + "\n" + strLocal)
}

func (ds xdgDataStorage) saveToFS() {
	newContent, err := json.Marshal(ds.packages)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: could not marshal new package list")
		os.Exit(1)
	}
	err = os.MkdirAll(filepath.Dir(ds.filePath), 0700)
	if err != nil && !errors.Is(err, fs.ErrExist) {
		fmt.Fprintln(os.Stderr, "Error: could not create folder", filepath.Dir(ds.filePath), "for saving")
		os.Exit(1)
	}
	err = os.WriteFile(ds.filePath, newContent, 0600)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: could not store changes\n%s\n", err)
	}
}
