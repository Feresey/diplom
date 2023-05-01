package schema

// Identifier описывает имя элемента, находящегося в указанной схеме
type Identifier struct {
	// TODO использовать OID
	Schema string
	Name   string
}

// TODO нужно ли добавлять "%s"."%s"?
func (i Identifier) String() string { return i.Schema + "." + i.Name }

// Schema отражает схему, расположенную в базе данных
type Schema struct {
	Types          map[string]*DBType
	ArrayTypes     map[string]*ArrayType
	CompositeTypes map[string]*CompositeType
	EnumTypes      map[string]*EnumType
	RangeTypes     map[string]*RangeType
	DomainTypes    map[string]*DomainType
	Tables         map[string]*Table
	Constraints    map[string]*Constraint
	// имена том же порядке что и в базе
	TableNames []string
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
	ForeignKeys map[string]*ForeignKey
	// Ключи, которые ссылаются на эту таблицу, ключ мапы - UNIQUE CONSTRAINT
	ReferencedBy map[string]*Constraint

	// Список всех CONSTRAINT-ов текущей таблицы
	Constraints map[string]*Constraint
}

// ForeignKey описывает внешнюю связь
// В PostgreSQL FK может ссылаться на PK или UNIQUE
type ForeignKey struct {
	// CONSTRAINT во внешней таблице
	// Uniq.Type IN (PK, UNIQUE)
	Uniq *Constraint

	// CONSTRAINT в текущей таблице
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

// DBType описывает тип данных базы
type DBType struct {
	// Имя типа
	TypeName Identifier
	Type     DataType

	ArrayType     *ArrayType
	EnumType      *EnumType
	CompositeType *CompositeType
	DomainType    *DomainType
	RangeType     *RangeType
}

type ArrayType struct {
	TypeName Identifier
	// Тип элемента массива. INTEGER[] -> INTEGER
	ElemType *DBType
}

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
	TypeName   Identifier
	Attributes DomainAttributes
	ElemType   *DBType
}

type RangeType struct {
	TypeName Identifier
	ElemType *DBType
}

type DomainAttributes struct {
	// Допустимы ли NULL значения колонки
	NotNullable bool
	// для типов с ограничением длины. VARCHAR(100)
	HasCharMaxLength bool
	CharMaxLength    int
	// Уровень вложенности массива, например INTEGER[][]
	ArrayDims int
}

// ColumnAttributes описывает аттрибуты колонки
type ColumnAttributes struct {
	DomainAttributes
	// Есть ли дефолтное значение (или GENERATED ALWAYS)
	HasDefault bool
	// Дефолтное значение
	Default string
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
	// Тип ограничения
	Type ConstraintType
	// Только для UNIQUE индекса
	NullsNotDistinct bool
	// Таблица, которой принадлежит ограничение
	Table *Table

	// Результат функции pg_getconstraintdef. Я не уверен что это вообще нужно, но пусть будет.
	Definition string

	// Колонки, на которые действует ограничение
	// Колонки всегда принадлежат той же таблице, которой принадлежит ограничение
	// Количество колонок всегда >= 1
	// Ключ мапы - имя колонки
	Columns map[string]*Column
}
