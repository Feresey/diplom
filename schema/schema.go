package schema

// Identifier описывает имя элемента
type Identifier struct {
	// Row identifier
	OID int
	// Schema name
	Schema string
	// Имя элемента
	Name string
}

func (i Identifier) String() string { return i.Schema + "." + i.Name }

// Schema отражает схему, расположенную в базе данных
type Schema struct {
	Types          map[string]*DBType
	ArrayTypes     map[string]*ArrayType
	CompositeTypes map[string]*CompositeType
	EnumTypes      map[string]*EnumType
	RangeTypes     map[string]*RangeType
	DomainTypes    map[string]*DomainType

	Tables      map[string]*Table
	Constraints map[string]*Constraint
	Indexes     map[string]*Index
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
	// Внешние ключи таблицы, ключ мапы - FK Name
	ForeignKeys map[string]*ForeignKey
	// Ключи, которые ссылаются на эту таблицу, ключ мапы - UNIQUE CONSTRAINT
	ReferencedBy map[string]*Constraint

	// Список всех CONSTRAINT-ов текущей таблицы
	Constraints map[string]*Constraint
	// Список всех INDEX-ов текущей таблицы
	Indexes map[string]*Index
}

// ForeignKey описывает внешнюю связь
// В PostgreSQL FK может ссылаться на PRIMARY KEY CONSTRAINT, UNIQUE CONSTRAINT, UNIQUE INDEX.
type ForeignKey struct {
	// CONSTRAINT в текущей таблице
	// Foreign.Type == FK
	Foreign *Constraint
	// Таблица, на которую ссылаются
	Reference *Table
	// Список колонок во внешней таблице, на которые ссылается FOREIGN KEY
	ReferenceColumns map[string]*Column
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
	ArrayDims        int
	IsNumeric        bool
	NumericPrecision int
	NumericScale     int
}

// ColumnAttributes описывает аттрибуты колонки
type ColumnAttributes struct {
	DomainAttributes

	HasDefault  bool
	IsGenerated bool
	// Дефолтное значение
	Default string
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

// Constraint описывает условие статического арбитража таблицы
type Constraint struct {
	// Имя ограничения
	Name Identifier
	// Тип ограничения
	Type ConstraintType
	// Таблица, которой принадлежит ограничение
	Table *Table
	// Индекс, на котором основано органичение (может быть пустым)
	Index *Index

	// Результат функции pg_getconstraintdef. Я не уверен что это вообще нужно, но пусть будет.
	Definition string

	// Колонки, на которые действует ограничение
	// Колонки всегда принадлежат той же таблице, которой принадлежит ограничение
	// Количество колонок всегда >= 1
	// Ключ мапы - имя колонки
	Columns map[string]*Column
}

type Index struct {
	// Имя индекса
	Name Identifier
	// Таблица, для которой создан индекс
	Table *Table
	// Колонки, которые затрагивает индекс
	Columns map[string]*Column
	// Определение индекса
	Definition string

	IsUnique  bool
	IsPrimary bool
	// Только для UNIQUE индекса
	IsNullsNotDistinct bool
}
