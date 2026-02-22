package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	oldModule := "github.com/taibuivan/yomira"
	newModule := "github.com/taibuivan/yomira"

	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}

		// Only process text files (Go, mod, md, yaml, toml, etc.)
		if strings.HasSuffix(path, ".go") || strings.HasSuffix(path, ".mod") || strings.HasSuffix(path, ".md") || strings.HasSuffix(path, ".yml") || strings.HasSuffix(path, ".toml") {
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			s := string(content)
			if strings.Contains(s, oldModule) {
				newS := strings.ReplaceAll(s, oldModule, newModule)
				os.WriteFile(path, []byte(newS), 0644)
				fmt.Printf("Updated %s\n", path)
			}
		}
		return nil
	})

	if err != nil {
		fmt.Printf("Error during walk: %v\n", err)
	} else {
		fmt.Println("Module path fully updated!")
	}
}
