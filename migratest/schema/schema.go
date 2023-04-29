package schema

import (
	_ "embed"
	"fmt"
)

type Identifier struct {
	Schema string
	Name   string
}

// TODO нужно ли добавлять "%s"."%s"?
func (i Identifier) String() string { return i.Schema + "." + i.Name }

// Schema отражает схему, расположенную в базе данных
type Schema struct {
	// TODO UserTypes
	Types map[string]*DBType

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
	Type *DBType
	// Аттрибуты колонки
	Attributes ColumnAttributes
}

//go:generate enumer -type DataType -trimprefix DataType
type DataType int

const (
	DataTypeUndefined DataType = iota
	DataTypeBuiltIn            // встроенные базовые типы. INT, BOOL, DATE, TEXT, INET, CIDR
	DataTypeArray              // массивы. INT[]
	DataTypeEnum               // Enum тип.
	DataTypeRange              // Тип-диапазон. INT4RANGE
	DataTypeComposite          // Тип-структура.
	DataTypeDomain             // Домен. Основан на любом другом типе и включает в себя ограничения для него
)

// DBType описывает тип данных базы
type DBType struct {
	// Имя типа
	TypeName Identifier
	DataType DataType
	// для типов с ограничением длины. VARCHAR(100)
	CharMaxLength int

	ArrayType     *ArrayType
	EnumType      *EnumType
	CompositeType *CompositeType
	DomainType    *DomainType
	RangeType     *RangeType
}

type ArrayType struct {
	TypeName Identifier
	// Уровень вложенности массива, например INTEGER[][]
	NestedCount int
	// TODO INT[4] - что с этим делать?
	// Тип элемента массива. INTEGER[][] -> INTEGER
	ElemType *DBType
}

/*
	SELECT
	n.nspname AS schema_name,
	t.typname AS enum_name,
	array_agg(e.enumlabel) AS enum_values

FROM

	pg_type t
	JOIN pg_enum e ON t.oid = e.enumtypid
	JOIN pg_namespace n ON n.oid = t.typnamespace

WHERE

	n.nspname = 'your_schema_name' AND
	t.typname = 'your_enum_name'

GROUP BY

	schema_name, enum_name;
*/
type EnumType struct {
	TypeName Identifier
	Values   []string
}

// TODO а что я вообще могу сделать с composite типом?
type CompositeType struct {
	TypeName   Identifier
	Attributes map[string]*CompositeAttribute
}

type CompositeAttribute struct {
	// Имя аттрибута
	Name string
	// COMPOSITE TYPE к которому относится аттрибут
	CompositeType *CompositeType
	// Тип аттрибута
	Type *DBType
}

type DomainType struct {
	TypeName Identifier
}

type RangeType struct {
	TypeName Identifier
	ElemType *DBType
}

// ColumnAttributes описывает аттрибуты колонки
type ColumnAttributes struct {
	IsDefault bool
	// Дефолтное значение
	Default string
	// Допустимы ли NULL значения колонки
	Nullable bool
	// Генерируемое значение колонки (может быть задано явно)
	ISGenerated bool
	// Условие генерации
	Generated string
}

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
