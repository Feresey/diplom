//go:build tools

package tools

import (
	_ "github.com/alvaroloes/enumer"
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
	_ "github.com/magefile/mage"
	_ "github.com/vektra/mockery/v2"
)
