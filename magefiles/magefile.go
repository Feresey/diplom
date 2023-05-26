package main

import (
	"bufio"
	"bytes"
	"log"
	"os"
	"regexp"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

func Lint() error {
	return sh.RunV("golangci-lint", "run")
}

func Generate() error {
	return sh.RunV("go", "generate", "./...")
}

func Update() error {
	if err := sh.RunV("go", "get", "-u", "-v"); err != nil {
		return err
	}
	return sh.RunV("go", "mod", "tidy", "-v")
}

type Test mg.Namespace

func (Test) All() error {
	return sh.RunV("go", "test", "-v", "./...")
}

type Tools mg.Namespace

func (Tools) chdir() error {
	return os.Chdir("tools")
}

func (t Tools) Update() {
	mg.Deps(t.chdir)
	mg.Deps(Update)
}

func (t Tools) Install() error {
	mg.Deps(t.chdir)
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
