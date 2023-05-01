package main

import (
	"bufio"
	"bytes"
	"log"
	"os"
	"regexp"

	"github.com/magefile/mage/sh"
)

func UpdateTools() error {
	return sh.Run("go", "get", "-u", "-v")
}

func InstallTools() error {
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
