package schema

import (
	"fmt"
)

type Identifier struct {
	Schema string
	Name   string
}

func (i Identifier) String() string { return fmt.Sprintf(`"%s"."%s"`, i.Schema, i.Name) }

// Schema отражает схему, расположенную в базе данных
type Schema struct {
	// Таблицы
	Tables map[string]*Table
	// Все индексы
	Constraints map[string]*Constraint

	// имена таблиц
	TableNames []string
	// имена индексов
	ConstraintNames []string
}

// setConstraintsNames устанавливает ConstraintNames на основе поля Constraints
func (s *Schema) setConstraintsNames() {
	names := make([]string, 0, len(s.Constraints))
	for key := range s.Constraints {
		names = append(names, key)
	}
	s.ConstraintNames = names
}

// setTableNames устанавливает TableNames на основе поля Tables
func (s *Schema) setTableNames() {
	names := make([]string, 0, len(s.Tables))
	for key := range s.Tables {
		names = append(names, key)
	}
	s.TableNames = names
}

// Table описывает таблицу базы данных
type Table struct {
	// имя таблицы
	Name Identifier
	// мапа колонок, где ключ - имя колонки
	Columns map[string]*Column

	// Главный ключ таблицы (может быть nil)
	PrimaryKey *Constraint
	// Внешние ключи таблицы, ключ мапы - FK
	ForeignKeys map[string]ForeignKey
	// Ключи, которые ссылаются на эту таблицу, ключ мапы - UNIQUE CONSTRAINT
	ReferencedBy map[string]*Constraint

	// Список всех CONSTRAINT-ов текущей таблицы
	Constraints map[string]*Constraint
}

// ForeignKey описывает внешнюю связь
type ForeignKey struct {
	// CONSTRAINT в текущей таблице
	// Uniq.Type IN (PK, UNIQUE)
	Uniq *Constraint

	// В PostgreSQL FK может ссылаться на PK или UNIQUE индекс
	// CONSTRAINT во внешней таблице
	// Foreign.Type == FK
	Foreign *Constraint
}

// Column описывает колонку таблицы
type Column struct {
	// Имя колонки
	Name string
	// Таблица, которой принадлежит колонка
	Table *Table
	// Тип колонки
	Type DBType
	// Аттрибуты колонки
	Attributes ColumnAttributes
}

// DBType описывает тип данных базы
type DBType struct {
	// Pretty type
	Type       string
	IsUserType bool

	// Underlying type
	UDTSchema string
	UDT       string

	Enum  string
	Range string

	DomainSchema string
	Domain       string

	CharMaxLength int
}

// ColumnAttributes описывает аттрибуты колонки
type ColumnAttributes struct {
	// Дефолтное значение (если указано), так же может быть SQL выражением
	Default string
	// Допустимы ли NULL значения колонки
	Nullable bool
	// Генерируемое значение колонки (может быть задано явно)
	ISGenerated bool
	// Условие генерации
	Generated string
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

// Constraint описывает условие статического арбитража таблицы
type Constraint struct {
	// Имя ограничения
	Name Identifier
	// Таблица, которой принадлежит ограничение
	Table *Table
	// Тип ограничения
	Type ConstraintType
	// Колонки, на которые действует ограничение
	// Колонки всегда принадлежат той же таблице, которой принадлежит ограничение
	// Количество колонок всегда >= 1
	// Для PRIMARY KEY колонка всегда одна
	// Ключ мапы - имя колонки
	Columns map[string]*Column
}

var pgConstraintType = map[string]ConstraintType{
	"p": ConstraintTypePK,
	"f": ConstraintTypeFK,
	"c": ConstraintTypeCheck,
	"u": ConstraintTypeUnique,
	"t": ConstraintTypeTrigger,
	"x": ConstraintTypeExclusion,
}

// SetType устанавливает тип ограничения исходя из значений таблицы pg_constraint
func (c *Constraint) SetType(constraintType string) error {
	typ, ok := pgConstraintType[constraintType]
	if !ok {
		return fmt.Errorf("unsupported constraint type: %q", constraintType)
	}
	c.Type = typ
	return nil
}
