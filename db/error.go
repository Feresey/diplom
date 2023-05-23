package db

import (
	"fmt"

	"github.com/davecgh/go-spew/spew"
)

type Error struct {
	Err     error
	Message string

	Query string
	Args  []any
}

func (e Error) Error() string {
	return fmt.Sprintf("%s: %v", e.Message, e.Err)
}

func (e Error) Pretty() string {
	return fmt.Sprintf("%s: %v:\nquery:\n%s\n\n===\nargs: %s", e.Message, e.Err, e.Query, spew.Sdump(e.Args...))
}
