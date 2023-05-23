package schema

import lua "github.com/yuin/gopher-lua"

// Identifier описывает имя элемента.
type Identifier struct {
	// Row identifier
	OID int `json:"oid,omitempty"`
	// Schema name
	Schema string `json:"schema,omitempty"`
	// Имя элемента
	Name string `json:"name,omitempty"`
}

func (i Identifier) String() string { return i.Schema + "." + i.Name }

// Schema отражает схему, расположенную в базе данных.
type Schema struct {
	Types  map[string]DBType   `json:"types,omitempty"`
	Enums  map[string]EnumType `json:"enums,omitempty"`
	Tables map[string]Table    `json:"tables,omitempty"`
}

// Table описывает таблицу базы данных.
type Table struct {
	// имя таблицы
	Name Identifier `json:"name,omitempty"`
	// мапа колонок, где ключ - имя колонки
	Columns map[string]Column `json:"columns,omitempty"`

	// Главный ключ таблицы (может быть nil)
	PrimaryKey *Constraint `json:"primary_key,omitempty"`
	// Внешние ключи таблицы
	ForeignKeys map[string]ForeignKey `json:"foreign_keys,omitempty"`
	// Таблицы, которые ссылаются на эту таблицу
	ReferencedBy map[string]Constraint `json:"referenced_by,omitempty"`

	// Список всех CONSTRAINT-ов текущей таблицы
	Constraints map[string]Constraint `json:"constraints,omitempty"`
	// Список всех INDEX-ов текущей таблицы
	Indexes map[string]Index `json:"indexes,omitempty"`
}

func (t Table) String() string { return t.Name.String() }
func (t Table) GetOID() int    { return t.Name.OID }

// ForeignKey описывает внешнюю связь
// В PostgreSQL FK может ссылаться на PRIMARY KEY CONSTRAINT, UNIQUE CONSTRAINT, UNIQUE INDEX.
type ForeignKey struct {
	// CONSTRAINT в текущей таблице
	// Constraint.Type == FK
	Constraint Constraint `json:"constraint"`
	// Таблица, на которую ссылаются
	ReferenceTable string `json:"reference"`
	// Список колонок во внешней таблице, на которые ссылается FOREIGN KEY
	ReferenceColumns []string `json:"reference_columns,omitempty"`
}

func (fk ForeignKey) String() string { return fk.Constraint.String() }

// Column описывает колонку таблицы.
type Column struct {
	// Порядковый номер колонки
	ColNum int `json:"col_num,omitempty"`
	// Имя колонки
	Name string `json:"name,omitempty"`
	// Тип колонки
	Type DBType `json:"type,omitempty"`
	// Аттрибуты колонки
	Attributes ColumnAttributes `json:"attributes,omitempty"`
}

func (c *Column) String() string { return c.Name }

//go:generate enumer -type DataType -trimprefix DataType -json
type DataType int

const (
	DataTypeUndefined  DataType = iota
	DataTypeBase                // встроенные базовые типы. INT, BOOL, DATE, TEXT, INET, CIDR
	DataTypeArray               // массивы. INT[]
	DataTypeEnum                // Enum тип.
	DataTypeRange               // Тип-диапазон. INT4RANGE
	DataTypeMultiRange          // Множество диапазонов. Создается автоматически при создании RANGE типа. INT4MULTIRANGE
	DataTypeComposite           // Тип-структура.
	DataTypeDomain              // Домен. Основан на любом другом типе и включает в себя ограничения для него
	DataTypePseudo              // Домен. Основан на любом другом типе и включает в себя ограничения для него
)

type DBType interface {
	GetOID() int
	TypType() DataType
	String() string
	ToLua(l *lua.LState) *lua.LTable
}

type BaseType struct {
	TypeName Identifier `json:"type_name"`
	Type     DataType   `json:"typtype"`
}

func (t *BaseType) String() string                { return t.TypeName.String() }
func (t *BaseType) GetOID() int                   { return t.TypeName.OID }
func (t *BaseType) TypType() DataType             { return t.Type }
func (t *BaseType) ToLua(*lua.LState) *lua.LTable { return nil }

type ElemType struct {
	BaseType `json:",omitempty"`
	ElemType DBType `json:"elem_type"`
}

func (d *ElemType) ToLua(*lua.LState) *lua.LTable { return nil }

type EnumType struct {
	BaseType `json:",omitempty"`
	Values   []string `json:"values"`
}

func (d *EnumType) ToLua(*lua.LState) *lua.LTable { return nil }

type DomainType struct {
	BaseType   `json:",omitempty"`
	ElemType   DBType           `json:"elem_type"`
	Attributes DomainAttributes `json:"attributes"`
}

func (d *DomainType) ToLua(*lua.LState) *lua.LTable { return nil }

type DomainAttributes struct {
	// Допустимы ли NULL значения колонки
	NotNullable bool `json:"not_nullable,omitempty"`
	// для типов с ограничением длины. VARCHAR(100)
	HasCharMaxLength bool `json:"has_char_max_length,omitempty"`
	CharMaxLength    int  `json:"char_max_length,omitempty"`
	// Уровень вложенности массива, например INTEGER[][]
	ArrayDims        int  `json:"array_dims,omitempty"`
	IsNumeric        bool `json:"is_numeric,omitempty"`
	NumericPrecision int  `json:"numeric_precision,omitempty"`
	NumericScale     int  `json:"numeric_scale,omitempty"`
}

// ColumnAttributes описывает аттрибуты колонки.
type ColumnAttributes struct {
	DomainAttributes `json:",omitempty"`

	HasDefault  bool `json:"has_default,omitempty"`
	IsGenerated bool `json:"is_generated,omitempty"`
	// Дефолтное значение
	Default string `json:"default,omitempty"`
}

//go:generate enumer -type ConstraintType -trimprefix ConstraintType -json
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

// Constraint описывает условие статического арбитража таблицы.
type Constraint struct {
	OID int `json:"oid"`
	// Имя ограничения
	Name string `json:"name"`
	// Тип ограничения
	Type ConstraintType `json:"type"`
	// Индекс, на котором основано органичение (может быть пустым)
	Index *Index `json:"index,omitempty"`

	// Результат функции pg_getconstraintdef. Я не уверен что это вообще нужно, но пусть будет.
	Definition string `json:"definition,omitempty"`

	// Колонки, на которые действует ограничение
	// Колонки всегда принадлежат той же таблице, которой принадлежит ограничение
	// Количество колонок всегда >= 1
	Columns []string `json:"columns"`
}

func (c Constraint) String() string { return c.Name }
func (c Constraint) GetOID() int    { return c.OID }

type Index struct {
	OID int `json:"oid"`
	// Имя индекса
	Name string `json:"name"`
	// Колонки, которые затрагивает индекс
	Columns []string `json:"columns"`
	// Определение индекса
	Definition string `json:"definition,omitempty"`

	IsUnique  bool `json:"is_unique,omitempty"`
	IsPrimary bool `json:"is_primary,omitempty"`
	// Только для UNIQUE индекса
	IsNullsNotDistinct bool `json:"is_nulls_not_distinct,omitempty"`
}

func (i Index) String() string { return i.Name }
func (i Index) GetOID() int    { return i.OID }
