package schema

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/volatiletech/null/v8"
)

type Identifier struct {
	Schema string
	Name   string
}

func (i Identifier) String() string { return fmt.Sprintf(`%s.%s`, i.Schema, i.Name) }

// Schema отражает схему, расположенную в базе данных
type Schema struct {
	// TODO UserTypes

	// Таблицы
	Tables map[string]*Table
	// имена таблиц в том же порядке что и в базе
	TableNames []string
	// Все индексы
	Constraints map[string]*Constraint
	// имена индексов в том же порядке что и в базе
	ConstraintNames []string
}

// Table описывает таблицу базы данных
type Table struct {
	// имя таблицы
	Name Identifier
	// TODO имена в порядке базы
	// мапа колонок, где ключ - имя колонки хранить имена как
	Columns map[string]*Column
	// имена колонок в том же порядке что и в базе
	ColumnNames []string

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
	UDTSchema null.String
	UDT       null.String

	IsArray bool

	Enum  string
	Range string

	DomainSchema null.String
	Domain       null.String

	CharMaxLength null.Int
}

func (t DBType) String() string {
	if t.IsUserType {
		return fmt.Sprintf("%s.%s", t.UDTSchema.String, t.UDT.String)
	}
	if t.IsArray {
		var arrayNestedCount int
		arrayElemType := strings.TrimLeftFunc(t.UDT.String, func(r rune) bool {
			if r == '_' {
				arrayNestedCount++
				return true
			}
			return false
		})
		// TODO get nested type info
		return fmt.Sprintf("%s.%s%s",
			t.UDTSchema.String,
			arrayElemType,
			strings.Repeat("[]", arrayNestedCount),
		)
	}
	if t.Domain.Valid {
		return fmt.Sprintf("%s.%s", t.DomainSchema.String, t.Domain.String)
	}
	return t.Type
}

// ColumnAttributes описывает аттрибуты колонки
type ColumnAttributes struct {
	// Дефолтное значение (если указано), так же может быть SQL выражением
	Default null.String
	// Допустимы ли NULL значения колонки
	Nullable bool
	// Генерируемое значение колонки (может быть задано явно)
	ISGenerated bool
	// Условие генерации
	Generated null.String
}

func (ca ColumnAttributes) String() string {
	var sb strings.Builder
	var needSpace bool
	space := func() {
		if needSpace {
			sb.WriteByte(' ')
			needSpace = false
		} else {
			needSpace = true
		}
	}

	if !ca.Nullable {
		space()
		sb.WriteString("NOT NULL")
	}

	if ca.Default.Valid {
		space()
		sb.WriteString("DEFAULT ")
		sb.WriteString(ca.Default.String)
	}

	if ca.ISGenerated {
		space()
		sb.WriteString("GENERATED ALWAYS AS ")
		sb.WriteString(ca.Generated.String)
		sb.WriteString(" STORED")
	}

	return sb.String()
}

// TODO
//
//go:generate enumer -type ConstraintType -trimprefix ConstraintType
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
	// Только для UNIQUE индекса
	NullsNotDistinct bool
	// Результат функции pg_getconstraintdef. Я не уверен что это вообще нужно, но пусть будет.
	Definition string
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
