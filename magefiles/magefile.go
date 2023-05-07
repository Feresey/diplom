package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"

	"github.com/magefile/mage/sh"
)

func Update() error {
	return sh.RunV("go", "get", "-u", "-v")
}

func InstallTools() error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	if filepath.Base(wd) != "tools" {
		return fmt.Errorf("mage -w tools")
	}
	file, err := os.ReadFile("tools.go")
	if err != nil {
		return err
	}
	scanner := bufio.NewScanner(bytes.NewReader(file))

	toolRe := regexp.MustCompile(`^\s+_\s+"(.*)"$`)

	var tools []string
	for scanner.Scan() {
		split := toolRe.FindStringSubmatch(scanner.Text())
		if len(split) != 2 {
			continue
		}
		tools = append(tools, split[1])
	}
	log.Println("tools: ", tools)

	for _, tool := range tools {
		if err := sh.RunV("go", "install", "-v", tool); err != nil {
			return err
		}
	}
	return nil
}

func Lint() error {
	return sh.Run("golangci-lint", "run")
}
