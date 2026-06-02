package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
)

type Config struct {
	StoreDir    string `json:"store_dir"`
	OutputDir   string `json:"game_dir"`
	ModpacksDir string `json:"modpacks_dir"`
	APIURL      string `json:"modrinth_api_url"`
	APIVersion  string `json:"modrinth_api_version"`
}

var ErrLoadConfig = errors.New("Failed to load config")

func loadConfig(path string) (c Config, err error) {
	cfgFile, err := os.Open(path)
	if err != nil {
		return c, fmt.Errorf("%w: %w", ErrLoadConfig, err)
	}

	jsonData, err := io.ReadAll(cfgFile)
	if err != nil {
		return c, fmt.Errorf("%w: %w", ErrLoadConfig, err)
	}

	var cfg Config
	if err := json.Unmarshal(jsonData, &cfg); err != nil {
		return c, fmt.Errorf("%w: %w", ErrLoadConfig, err)
	}
	return cfg, nil
}

const MaxEnvSupportOptions = 4

func parseEnvSupportList(option string) ([]string, bool) {
	tags := strings.Split(option, ",")
	options := make([]string, 0, MaxEnvSupportOptions)

	for _, tag := range tags {
		tag := strings.TrimSpace(strings.ToLower(tag))
		if EnvSupport(tag).isValid() {
			options = append(options, tag)
		}
	}
	if len(options) < 1 {
		return options, false
	}
	return options, true
}

// returns length of the longest string in the array
func longestStringLength(s []string) int {
	l := 0
	for _, a := range s {
		if len(a) > l {
			l = len(a)
		}
	}
	return l
}

func padRight(s string, l int) string {
	if l == 0 {
		return s
	}
	padding := l - len(s)
	if padding < 0 {
		return s
	}
	return s + strings.Repeat(" ", padding)
}

type UseCmdFlags struct {
	crate  string
	client string
	server string
	dryRun bool
}

func (u *UseCmdFlags) Run() {

}

const OverridesDirName = "overrides"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func(ctx context.Context) {
		<-ctx.Done()
		clearLine()
		showCursor()
		os.Exit(0)
	}(ctx)

	var f flags
	f.defineFlags()

	cfg, err := loadConfig("config/config.json")
	if err != nil {
		log.Fatal(err)
	}

	flag.Usage = func() {
		fmt.Println()
		flag.PrintDefaults()
	}

	var useCmdFlags UseCmdFlags

	useCmd := flag.NewFlagSet("use", flag.ExitOnError)
	useCmd.StringVar(&useCmdFlags.client, "client", "all", "usage")
	useCmd.StringVar(&useCmdFlags.server, "server", "all", "usage")
	useCmd.StringVar(&useCmdFlags.crate, "profile", "", "usage")
	useCmd.BoolVar(&useCmdFlags.dryRun, "dry-run", false, "usage")

	flag.Parse()

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "use":
			if len(os.Args) <= 2 {
				return
			}

			modpackName := os.Args[2]
			modpackPath := filepath.Join(cfg.ModpacksDir, modpackName)
			dir, err := os.ReadDir(modpackPath)
			if err != nil {
				log.Fatal(err)
			}
			hasOverrides := false
			for _, f := range dir {
				if !f.IsDir() {
					continue // files should be placed in overrides directory
				}
				subDir := f.Name()
				if subDir == OverridesDirName {
					hasOverrides = true
					continue // overrides are copied last
				}

				modpackSubDir := filepath.Join(modpackPath, subDir)
				outputDir := filepath.Join(cfg.OutputDir, subDir)
				_, err := copyDirInModpack(modpackSubDir, outputDir, true)
				if err != nil {
					log.Fatal(err)
				}
			}
			if hasOverrides {
				overridesDir := filepath.Join(modpackPath, OverridesDirName)
				_, err := copyDirInModpack(overridesDir, cfg.OutputDir, true)
				if err != nil {
					log.Fatal(err)
				}
			}
		case "modpack":
			name := os.Args[2]
			p := os.Args[3]
			mpath := filepath.Join(cfg.ModpacksDir, name)
			if err := extract(p, mpath); err != nil {
				log.Fatal(err)
			}
			manifest, err := readModrinthManifest(mpath)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Printf("Name: %s\n", manifest.Name)
			fmt.Printf("Version: %s\n", manifest.VersionId)
			fmt.Printf("Summary: %s\n", manifest.Dependencies.Minecraft)
		case "list":
			name := os.Args[2]
			mpath := filepath.Join(cfg.ModpacksDir, name)
			dir, err := os.ReadDir(mpath)
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					fmt.Printf("Directory or modpack %s does not exist\n", name)
					os.Exit(0)
				} else {
					log.Fatal(err)
				}
			}
			fmt.Println("\n", name)
			fmt.Println("------------------")
			if len(dir) < 1 {
				fmt.Println(" *empty* ")
			}
			for _, file := range dir {
				fmt.Printf("* %s\n", file.Name())
			}
		case "dl":
			if len(os.Args) <= 2 {
				return
			}
			modpackName := os.Args[2]

			modpackPath := filepath.Join(cfg.ModpacksDir, modpackName)
			manifest, err := readModrinthManifest(modpackPath)
			if err != nil {
				log.Fatal(err)
			}
			fileCount := len(manifest.Files)

			// tracking progress
			downloadProgress := newProgress(fileCount)

			// setup http httpClient
			httpClient := newAPIClient(cfg.APIURL, cfg.APIVersion)

			hideCursor()
			fmt.Printf("\n    %s\n", modpackName)

			filesDownloaded := 0
			var totalBytes int64 = 0
			maxLen := longestStringLength(manifest.getFiles()) // formatting

			for i, file := range manifest.Files {
				if len(file.Downloads) < 1 {
					continue
				}

				downloadURL := file.Downloads[0]
				destFilepath := filepath.Join(modpackPath, file.Path)

				// skip existing files
				_, err := os.Stat(destFilepath)
				if err == nil {
					fmt.Printf("[SKIPPED] %s already exists\n", whiteBoldColor(file.Path))
					continue
				} else if !errors.Is(err, os.ErrNotExist) {
					log.Fatal(err)
				}

				filename := filepath.Base(file.Path)
				fmt.Printf("Downloading %s... (%s)\r", padRight(filename, 10), downloadProgress.ratio())
				bytesDownloaded, err := httpClient.DownloadFile(ctx, downloadURL, destFilepath)
				if err != nil {
					log.Fatal(err)
				}

				// statistics
				totalBytes += bytesDownloaded
				filesDownloaded++
				downloadProgress.Update(i + 1)

				clearLine()
				checkmark := "\u2713"
				fmt.Printf("[DOWNLOADED] %s %s\n\n", success(checkmark), padRight(filename, maxLen))
				cursorUp(1)
			}
			fmt.Printf("\nDownload Complete! (%d/%d) files; Total %d bytes \n", filesDownloaded, fileCount, int(totalBytes))
			showCursor()
		}
	}

	if f.listmodpack != "" {
		modpackPath := filepath.Join(cfg.ModpacksDir, f.listmodpack)
		manifest, err := readModrinthManifest(modpackPath)
		if err != nil {
			log.Fatal(err)
		}
		if !f.filter {
			fmt.Printf("\nAll mods in %s\n\n", whiteIntenseColor(f.listmodpack))
			files := manifest.getFiles()

			builder := strings.Builder{}
			for _, f := range files {
				builder.WriteString("* " + f + "\n")
			}
			fmt.Printf("%s\n", builder.String())
			return
		}

		if f.server != "none" {
			if strings.ToLower(f.server) == "all" {
				fmt.Printf("(Required | Optional) server mods")
				files := manifest.getServerSides()
				for _, f := range files {
					fmt.Printf("* %s\n", f)
				}
			}
			tags, ok := parseEnvSupportList(f.server)
			if ok {
				if len(tags) > 4 {
					log.Fatal("ok thats too many tags! only 4 MAX, k?")
				}
				fmt.Printf("%s server mods\n", tags)
				for _, tag := range tags {
					if !EnvSupport(tag).isValid() {
						continue
					}
					files := manifest.getServerSideWithTag(EnvSupport(tag))

					l := longestStringLength(files)
					for _, f := range files {
						fmt.Printf("* %s - [%s]\n", padRight(f, l), tag)
					}
				}
			}
		}

		if f.client != "none" {
			if strings.ToLower(f.client) == "all" {
				fmt.Printf("(Required | Optional) client mods")
				files := manifest.getClientSides()
				for _, f := range files {
					fmt.Printf("* %s\n", f)
				}
			}
			tags, ok := parseEnvSupportList(f.client)
			if ok {
				if len(tags) > 4 {
					log.Fatal("ok thats too many tags! only 4 MAX, k?")
				}
				fmt.Printf("%s client mods\n", tags)
				for _, tag := range tags {
					if !EnvSupport(tag).isValid() {
						continue
					}
					files := manifest.getServerSideWithTag(EnvSupport(tag))

					l := longestStringLength(files)
					for _, f := range files {
						fmt.Printf("* %s - [%s]\n", padRight(f, l), tag)
					}
				}
			}
		}
	}
}
