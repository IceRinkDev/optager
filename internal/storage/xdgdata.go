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
)

type xdgDataStorage struct {
	filePath string
	packages []Pkg
}

type Pkg struct {
	FolderName string   `json:"folderName"`
	Binaries   []string `json:"binaries,omitempty"`
}

func New() *xdgDataStorage {
	dataBaseDir := os.Getenv("XDG_DATA_HOME")
	if dataBaseDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			fmt.Println("Error: no home folder found")
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

func (ds *xdgDataStorage) AddPkg(newPkg Pkg) {
	if !slices.ContainsFunc(ds.packages, func(p Pkg) bool {
		return p.FolderName == newPkg.FolderName
	}) {
		ds.packages = append(ds.packages, newPkg)
		ds.saveToFS()
	}
}

func (ds xdgDataStorage) saveToFS() {
	newContent, err := json.Marshal(ds.packages)
	if err != nil {
		fmt.Println("Error: could not marshal new package list")
		os.Exit(1)
	}
	err = os.MkdirAll(filepath.Dir(ds.filePath), 0700)
	if err != nil && !errors.Is(err, fs.ErrExist) {
		fmt.Println("Error: could not create folder", filepath.Dir(ds.filePath), "for saving")
		os.Exit(1)
	}
	err = os.WriteFile(ds.filePath, newContent, 0600)
	if err != nil {
		fmt.Printf("Error: could not store changes\n%s\n", err)
	}
}
