package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	files, err := filepath.Glob("internal/core/chapter/*.go")
	if err != nil {
		panic(err)
	}

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			panic(err)
		}

		newContent := bytes.ReplaceAll(content, []byte("package comic"), []byte("package chapter"))
		err = os.WriteFile(file, newContent, 0644)
		if err != nil {
			panic(err)
		}
		fmt.Println("Updated package in", file)
	}
}
