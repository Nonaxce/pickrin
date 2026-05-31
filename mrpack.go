package main

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
)

//var hasherSHA256 = crypto.Hash(crypto.SHA256).New()

type Env string

const (
	Client Env = "Client"
	Server Env = "Server"
)

type EnvSupport string

const (
	Required    EnvSupport = "required"
	Optional    EnvSupport = "optional"
	Unsupported EnvSupport = "unsupported"
	Unknown     EnvSupport = "unknown"
)

type ModpackFile struct {
	Path string `json:"path"`

	Hashes struct {
		SHA1   string `json:"sha1"`
		SHA512 string `json:"sha512"`
	} `json:"hashes"`

	Downloads []string `json:"downloads"`
	FileSize  float64  `json:"fileSize"`

	Env struct {
		Client string `json:"client"`
		Server string `json:"server"`
	} `json:"env"`
}

type ModrinthIndex struct {
	FormatVersion float64 `json:"formatVersion"`
	Game          string  `json:"game"`
	VersionId     string  `json:"versionId"`
	Name          string  `json:"name"`
	Summary       string  `json:"summary"`

	Dependencies struct {
		Minecraft string `json:"minecraft"`
		Neoforge  string `json:"neoforge"`
		Fabric    string `json:"fabric"`
		Forge     string `json:"forge"`
		Quilt     string `json:"quilt"`
	} `json:"dependencies"`

	Files []ModpackFile `json:"files"`
}

func (m *ModrinthIndex) getClientSideWithTag(tag string) []string {
	filteredMods := make([]string, 0, len(m.Files)/2)
	for _, file := range m.Files {
		if file.Env.Client == tag {
			filteredMods = append(filteredMods, filepath.Base(file.Path))
		}
	}
	return filteredMods
}

func (m *ModrinthIndex) getServerSideWithTag(tag string) []string {
	filteredMods := make([]string, 0, len(m.Files)/2)
	for _, file := range m.Files {
		if file.Env.Server == tag {
			filteredMods = append(filteredMods, filepath.Base(file.Path))
		}
	}
	return filteredMods
}

func (m *ModrinthIndex) getServerSides() []string {
	filteredMods := make([]string, 0, len(m.Files)/2)
	for _, file := range m.Files {
		if file.Env.Server == "required" || file.Env.Server == "optional" {
			filteredMods = append(filteredMods, filepath.Base(file.Path))
		}
	}
	return filteredMods
}

func (m *ModrinthIndex) getClientSides() []string {
	filteredMods := make([]string, 0, len(m.Files)/2)
	for _, file := range m.Files {
		if file.Env.Client == "required" || file.Env.Client == "optional" {
			filteredMods = append(filteredMods, filepath.Base(file.Path))
		}
	}
	return filteredMods
}

func (m *ModrinthIndex) getFiles() []string {
	filteredMods := make([]string, 0, len(m.Files))
	for _, file := range m.Files {
		filteredMods = append(filteredMods, filepath.Base(file.Path))
	}
	return filteredMods
}

// extracts a zip archive like .mrpack to a destination
func extract(path string, dest string) error {
	dest = filepath.Clean(dest)

	fmt.Print("Creating directories...\r")
	// attempt to create directory
	err := os.MkdirAll(dest, os.ModePerm)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return fmt.Errorf("Directory or file \"%s\" already exists: %w", dest, err)
		}
		return fmt.Errorf("Unable to create destination directory \"%s\": %w", dest, err)
	}
	fmt.Println("\u2713 Creating directories...")

	fmt.Print("Extracting files...\r")
	reader, err := zip.OpenReader(path)
	if err != nil {
		log.Fatal("Cannot open reader")
		return err
	}
	defer reader.Close()

	for _, file := range reader.File {
		// path is for the destination file
		filePath := filepath.Join(dest, file.Name)

		// check if filepath is unsafe or is malicious by trying to escape the directory
		cleanedPath := filepath.Clean(dest)
		if !strings.HasPrefix(filePath, cleanedPath) {
			return errors.New("Invalid file path")
		}

		// if the file is a directory, create the directory along with any parents
		if file.FileInfo().IsDir() {
			err = os.MkdirAll(filePath, os.ModePerm)
			if err != nil {
				return err
			}
			continue
		}

		// Open the file in the archive to copy later
		fileInArchive, err := file.Open()
		if err != nil {
			fmt.Println("Failed to open file: ", file.FileInfo().Name())
			return err
		}
		defer fileInArchive.Close()

		// 1. Create all the parent folders of the file
		err = os.MkdirAll(filepath.Dir(filePath), os.ModePerm)
		if err != nil {
			return err
		}

		// 2. Open the file, and set its permission to be writable
		destFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode().Perm())
		if err != nil {
			return fmt.Errorf("Failed to copy over file %w", err)
		}
		defer destFile.Close()

		// copy the contents of the archive file to the destination
		_, err = io.Copy(destFile, fileInArchive)
		if err != nil {
			return fmt.Errorf("Failed to copy over file %w", err)
		}
	}
	fmt.Println("\u2713 Extracting files...")
	fmt.Println("Sucessfully extracted files from .mrpack")
	return nil
}

func (m *ModrinthIndex) createModrinthIndexJSON(mi ModrinthIndex) error {
	// check if file exists
	if _, err := os.Stat("modrinth.index.json"); errors.Is(err, os.ErrNotExist) {
		return err
	}
	jsonData, err := json.MarshalIndent(mi, "", "	")
	if err != nil {
		return err
	}
	err = os.WriteFile("modrinth.index.json", jsonData, 0644)
	if err != nil {
		return err
	}
	return nil
}

func readModrinthManifest(path string) (m ModrinthIndex, err error) {
	manifestPath := filepath.Join(path, "modrinth.index.json")
	manifest, err := os.Open(manifestPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return m, fmt.Errorf("Cannot find modrinth manifest within the modpack directory: %w", err)
		}
		return m, err
	}

	manifestData, err := io.ReadAll(manifest)
	if err != nil {
		return m, err
	}

	if err := json.Unmarshal(manifestData, &m); err != nil {
		return m, err
	}

	return m, nil
}

func copyModpackFiles(srcPath string, destDir string, verbose bool) (bytesCopied int64, err error) {
	destDir = filepath.Clean(destDir)
	srcPath, err = filepath.Abs(srcPath)
	if err != nil {
		return bytesCopied, err
	}

	wd, err := os.Getwd() // wd, err := os.Executable()
	if err != nil {
		return bytesCopied, err
	}

	// V:/root/src/path -> src/path
	relativeSrcDir, err := filepath.Rel(wd, srcPath)
	if err != nil {
		return bytesCopied, err
	}

	err = filepath.WalkDir(relativeSrcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			srcFile, err := os.Open(path)
			if err != nil {
				return err
			}
			defer srcFile.Close()

			// parent/source/dir/mods/hello.jar  -->  mods/hello.jar
			relDestFilePath := strings.TrimPrefix(path, relativeSrcDir+string(filepath.Separator))

			// mods/hello.jar  -->  dest/dir/mods/hello.jar
			destFilePath := filepath.Join(destDir, filepath.Clean(relDestFilePath))

			if err = os.MkdirAll(filepath.Dir(destFilePath), os.ModePerm); err != nil {
				return err
			}

			destFile, err := os.Create(destFilePath)
			if err != nil {
				return err
			}
			defer destFile.Close()

			b, err := io.Copy(destFile, srcFile)
			if err != nil {
				return err
			}
			bytesCopied += b

			if verbose {
				fmt.Printf("* %s\n", relDestFilePath)
			}
		}
		return nil
	})
	if err != nil {
		return bytesCopied, fmt.Errorf("Failed to override files: %w", err)
	}
	return bytesCopied, nil
}
