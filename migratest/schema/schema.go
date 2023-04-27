package schema

import (
	"fmt"
)

type Schema struct {
	Tables     map[string]*Table
	TableNames []string

	Constraints     map[string]*Constraint
	ConstraintNames []string
}

func (s *Schema) setConstraintsNames() {
	names := make([]string, 0, len(s.Constraints))
	for _, c := range s.Constraints {
		names = append(names, c.Name)
	}
	s.ConstraintNames = names
}

func (s *Schema) setTableNames() {
	names := make([]string, 0, len(s.Tables))
	for name := range s.Tables {
		names = append(names, name)
	}
	s.TableNames = names
}

type Table struct {
	Name    string
	Columns map[string]*Column

	PrimaryKey   *Constraint
	ForeignKeys  map[string]ForeignKey
	ReferencedBy map[string]*Constraint
	Constraints  map[string]*Constraint
}

type ForeignKey struct {
	Local *Constraint
	// В PostgreSQL FK может ссылаться на PK или UNIQUE индекс
	ForeignUnique *Constraint
}

type Column struct {
	Name string
	Type DBType
	ColumnOptions
}

type DBType struct {
	// Pretty type
	PrettyType string
	// Underlying type
	UDT string

	Enum   string
	Domain string
	Range  string

	CharMaxLength int
}

type ColumnOptions struct {
	Default     string
	Nullable    bool
	ISGenerated bool
	Generated   string
}

// TODO
//
//go:generate ${TOOLS}/enumer -type ConstraintType -trimprefix ConstraintType
type ConstraintType int

const (
	ConstraintTypeUndefined ConstraintType = iota
	ConstraintTypePK
	ConstraintTypeFK
	ConstraintTypeUnique
	ConstraintTypeCheck
	ConstraintTypeTrigger
	ConstraintTypeExclusion
)

type Constraint struct {
	Name    string
	Type    ConstraintType
	Columns map[string]*Column
}

func (c *Constraint) SetType(constraintType string) error {
	switch constraintType {
	case "p":
		c.Type = ConstraintTypePK
	case "f":
		c.Type = ConstraintTypeFK
	case "c":
		c.Type = ConstraintTypeCheck
	case "u":
		c.Type = ConstraintTypeUnique
	case "t":
		c.Type = ConstraintTypeTrigger
	case "x":
		c.Type = ConstraintTypeExclusion
	default:
		return fmt.Errorf("unsupported constraint type: %q", constraintType)
	}
	return nil
}
