package schema

import (
	"github.com/Masterminds/squirrel"
)

type Parser struct {
	db DBConn

	sb squirrel.SelectBuilder
}

func NewParser(db DBConn) *Parser {
	return &Parser{
		db: db,
		sb: squirrel.SelectBuilder{}.PlaceholderFormat(squirrel.Dollar),
	}
}

type Type struct {
	Base string

	Enum   string
	Domain string
	Range  string
}

type Column struct {
	Name string
	Type Type
}

type TableColumn struct {
	Table  string
	Column Column
}

//go:generate ${TOOLS}/enumer -type ConstraintType -trimprefix ConstraintType
type ConstraintType int

const (
	ConstraintTypeUndefined ConstraintType = iota
	ConstraintTypePK
	ConstraintTypeFK
	ConstraintTypeNotNull
	ConstraintTypeUnique
	ConstraintTypeCheck
	ConstraintTypeExclusion
)

type Constraint struct {
	Type       ConstraintType
	Columns    []Column
	References []TableColumn
}

type Table struct {
	Name        string
	Columns     []Column
	Constraints []Constraint
}

func (p *Parser) GetTables(schema string) ([]Table, error) {
	q := p.sb.From("pg_tables").
		Columns("tablename").
		Where("schemaname = ?", schema)
}
