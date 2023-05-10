package schema

import (
	"encoding/json"
	"fmt"
)

// Identifier описывает имя элемента
type Identifier struct {
	// Row identifier
	OID int `json:"oid,omitempty"`
	// Schema name
	Schema string `json:"schema,omitempty"`
	// Имя элемента
	Name string `json:"name,omitempty"`
}

func (i Identifier) String() string { return i.Schema + "." + i.Name }

// Schema отражает схему, расположенную в базе данных
type Schema struct {
	Types          map[string]*DBType        `json:"types,omitempty"`
	ArrayTypes     map[string]*ArrayType     `json:"array_types,omitempty"`
	CompositeTypes map[string]*CompositeType `json:"composite_types,omitempty"`
	EnumTypes      map[string]*EnumType      `json:"enum_types,omitempty"`
	RangeTypes     map[string]*RangeType     `json:"range_types,omitempty"`
	DomainTypes    map[string]*DomainType    `json:"domain_types,omitempty"`

	Tables      map[string]*Table      `json:"tables,omitempty"`
	Constraints map[string]*Constraint `json:"constraints,omitempty"`
	Indexes     map[string]*Index      `json:"indexes,omitempty"`
	// имена том же порядке что и в базе
	TableNames []string `json:"table_names,omitempty"`
}

// Table описывает таблицу базы данных
type Table struct {
	// имя таблицы
	Name Identifier `json:"name,omitempty"`
	// мапа колонок, где ключ - имя колонки хранить имена как
	Columns map[string]*Column `json:"columns,omitempty"`
	// имена колонок в том же порядке что и в базе
	ColumnNames []string `json:"column_names,omitempty"`

	// Главный ключ таблицы (может быть nil)
	PrimaryKey *Constraint `json:"primary_key,omitempty"`
	// Внешние ключи таблицы, ключ мапы - FK Name
	ForeignKeys map[string]*ForeignKey `json:"foreign_keys,omitempty"`
	// Ключи, которые ссылаются на эту таблицу, ключ мапы - UNIQUE CONSTRAINT
	ReferencedBy map[string]*Constraint `json:"referenced_by,omitempty"`

	// Список всех CONSTRAINT-ов текущей таблицы
	Constraints map[string]*Constraint `json:"constraints,omitempty"`
	// Список всех INDEX-ов текущей таблицы
	Indexes map[string]*Index `json:"indexes,omitempty"`
}

func (t *Table) String() string { return t.Name.String() }

// ForeignKey описывает внешнюю связь
// В PostgreSQL FK может ссылаться на PRIMARY KEY CONSTRAINT, UNIQUE CONSTRAINT, UNIQUE INDEX.
type ForeignKey struct {
	// CONSTRAINT в текущей таблице
	// Foreign.Type == FK
	Foreign *Constraint `json:"foreign,omitempty"`
	// Таблица, на которую ссылаются
	Reference *Table `json:"reference,omitempty"`
	// Список колонок во внешней таблице, на которые ссылается FOREIGN KEY
	ReferenceColumns map[string]*Column `json:"reference_columns,omitempty"`
}

func (f *ForeignKey) String() string { return f.Foreign.String() }

// Column описывает колонку таблицы
type Column struct {
	// Имя колонки
	Name string `json:"name,omitempty"`
	// Таблица, которой принадлежит колонка
	Table *Table `json:"-,omitempty"`
	// Тип колонки
	Type *DBType `json:"type,omitempty"`
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

// DBType описывает тип данных базы
type DBType struct {
	// Имя типа
	TypeName Identifier `json:"type_name,omitempty"`
	Type     DataType   `json:"type,omitempty"`

	ArrayType     *ArrayType     `json:"array_type,omitempty"`
	EnumType      *EnumType      `json:"enum_type,omitempty"`
	CompositeType *CompositeType `json:"composite_type,omitempty"`
	DomainType    *DomainType    `json:"domain_type,omitempty"`
	RangeType     *RangeType     `json:"range_type,omitempty"`
}

func (d *DBType) String() string { return d.TypeName.String() }

type ArrayType struct {
	TypeName Identifier `json:"type_name,omitempty"`
	// Тип элемента массива. INTEGER[] -> INTEGER
	ElemType *DBType `json:"elem_type,omitempty"`
}

func (a *ArrayType) String() string { return a.TypeName.String() }

func (a *ArrayType) GetElemType() *DBType     { return a.ElemType }
func (a *ArrayType) SetElemType(elem *DBType) { a.ElemType = elem }

type EnumType struct {
	TypeName Identifier `json:"type_name,omitempty"`
	Values   []string   `json:"values,omitempty"`
}

func (e *EnumType) String() string { return e.TypeName.String() }

type CompositeType struct {
	TypeName Identifier `json:"type_name,omitempty"`
	// Attributes map[string]*CompositeAttribute `json:"attributes,omitempty"`
}

func (c *CompositeType) String() string { return c.TypeName.String() }

// type CompositeAttribute struct {
// 	// Имя аттрибута
// 	Name string `json:"name,omitempty"`
// 	// COMPOSITE TYPE к которому относится аттрибут
// 	CompositeType *CompositeType `json:"composite_type,omitempty"`
// 	// Тип аттрибута
// 	Type *DBType `json:"type,omitempty"`
// }

type DomainType struct {
	TypeName   Identifier       `json:"type_name,omitempty"`
	Attributes DomainAttributes `json:"attributes,omitempty"`
	ElemType   *DBType          `json:"elem_type,omitempty"`
}

func (d *DomainType) String() string           { return d.TypeName.String() }
func (d *DomainType) GetElemType() *DBType     { return d.ElemType }
func (d *DomainType) SetElemType(elem *DBType) { d.ElemType = elem }

type RangeType struct {
	TypeName Identifier `json:"type_name,omitempty"`
	ElemType *DBType    `json:"elem_type,omitempty"`
}

func (r *RangeType) String() string           { return r.TypeName.String() }
func (r *RangeType) GetElemType() *DBType     { return r.ElemType }
func (r *RangeType) SetElemType(elem *DBType) { r.ElemType = elem }

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

// ColumnAttributes описывает аттрибуты колонки
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

// Constraint описывает условие статического арбитража таблицы
type Constraint struct {
	// Имя ограничения
	Name Identifier `json:"name,omitempty"`
	// Тип ограничения
	Type ConstraintType `json:"type,omitempty"`
	// Таблица, которой принадлежит ограничение
	Table *Table `-:"table,omitempty"`
	// Индекс, на котором основано органичение (может быть пустым)
	Index *Index `json:"index,omitempty"`

	// Результат функции pg_getconstraintdef. Я не уверен что это вообще нужно, но пусть будет.
	Definition string `json:"definition,omitempty"`

	// Колонки, на которые действует ограничение
	// Колонки всегда принадлежат той же таблице, которой принадлежит ограничение
	// Количество колонок всегда >= 1
	// Ключ мапы - имя колонки
	Columns map[string]*Column `json:"columns,omitempty"`
}

func (c *Constraint) String() string { return c.Name.String() }

type Index struct {
	// Имя индекса
	Name Identifier `json:"name,omitempty"`
	// Таблица, для которой создан индекс
	Table *Table `json:"-,omitempty"`
	// Колонки, которые затрагивает индекс
	Columns map[string]*Column `json:"columns,omitempty"`
	// Определение индекса
	Definition string `json:"definition,omitempty"`

	IsUnique  bool `json:"is_unique,omitempty"`
	IsPrimary bool `json:"is_primary,omitempty"`
	// Только для UNIQUE индекса
	IsNullsNotDistinct bool `json:"is_nulls_not_distinct,omitempty"`
}

func (i *Index) String() string { return i.Name.String() }

func getValueFormat[T fmt.Stringer](
	m map[string]T, key string,
	msg string, args ...any,
) (T, error) {
	value, ok := m[key]
	if !ok {
		return value, fmt.Errorf(msg, args...)
	}
	return value, nil
}

func getElemType[T fmt.Stringer](
	basename, basetype string,
	typename string,
	m map[string]T,
) (T, error) {
	return getValueFormat(m, typename,
		"%s type %q not found for type %s", basename, typename, basetype)
}

func fillElemType[T interface {
	fmt.Stringer
	GetElemType() *DBType
	SetElemType(elem *DBType)
}](
	m map[string]*DBType, base T,
) error {
	if elem := base.GetElemType(); elem != nil {
		typ, err := getValueFormat(m, elem.String(),
			"elem type %q not found for type %q", elem, base)
		if err != nil {
			return err
		}
		base.SetElemType(typ)
	}
	return nil
}

func (s *Schema) fillConcreteTypes() error {
	for _, typ := range s.ArrayTypes {
		if err := fillElemType(s.Types, typ); err != nil {
			return err
		}
	}
	for _, typ := range s.DomainTypes {
		if err := fillElemType(s.Types, typ); err != nil {
			return err
		}
	}
	for _, typ := range s.RangeTypes {
		if err := fillElemType(s.Types, typ); err != nil {
			return err
		}
	}
	return nil
}

func (s *Schema) fillTypes() error {
	for _, typ := range s.Types {
		var err error
		switch {
		case typ.ArrayType != nil:
			typ.ArrayType, err = getElemType(
				"array", typ.String(),
				typ.ArrayType.String(), s.ArrayTypes)
		case typ.CompositeType != nil:
			typ.CompositeType, err = getElemType(
				"composite", typ.String(),
				typ.CompositeType.String(), s.CompositeTypes)
		case typ.DomainType != nil:
			typ.DomainType, err = getElemType(
				"domain", typ.String(),
				typ.DomainType.String(), s.DomainTypes)
		case typ.EnumType != nil:
			typ.EnumType, err = getElemType(
				"enum", typ.String(),
				typ.EnumType.String(), s.EnumTypes)
		case typ.RangeType != nil:
			typ.RangeType, err = getElemType(
				"range", typ.String(),
				typ.RangeType.String(), s.RangeTypes)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func fillColumns(dst, src map[string]*Column, msg string, args ...any) error {
	for colName := range dst {
		col, err := getValueFormat(src, colName,
			"column %q not found for"+msg, append([]any{colName}, args...))
		if err != nil {
			return err
		}
		dst[colName] = col
	}
	return nil
}

func (s *Schema) fillIndexes() error {
	for _, index := range s.Indexes {
		if index.Table == nil {
			return fmt.Errorf("table not specified for index %q", index)
		}
		table, err := getValueFormat(s.Tables, index.Table.String(),
			"table %q not found for index %q", index.Table, index)
		if err != nil {
			return err
		}
		index.Table = table

		err = fillColumns(index.Columns, table.Columns,
			"table %q in index %q", index.Table, index)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Schema) fillConstraints() error {
	for _, constraint := range s.Constraints {
		if constraint.Table == nil {
			return fmt.Errorf("table not specified for constraint %q", constraint)
		}
		table, err := getValueFormat(s.Tables, constraint.Table.String(),
			"table %q not found for constraint %q", constraint.Table, constraint)
		if err != nil {
			return err
		}
		constraint.Table = table

		err = fillColumns(constraint.Columns, table.Columns,
			"table %q in constraint %q", constraint.Table, constraint)
		if err != nil {
			return err
		}

		if constraint.Index != nil {
			index, err := getValueFormat(s.Indexes, constraint.Index.String(),
				"index %q not found for constraint %q", constraint.Index, constraint)
			if err != nil {
				return err
			}
			constraint.Index = index
		}
	}
	return nil
}

func (s *Schema) fillTableColumns(table *Table) error {
	for colname, col := range table.Columns {
		col.Table = table
		typ, err := getValueFormat(s.Types, col.Type.String(),
			"type %q not found for column %q in table %q ",
			col.Type, colname, table)
		if err != nil {
			return err
		}
		col.Type = typ
	}
	return nil
}

func (s *Schema) fillTableConstraints(table *Table) error {
	for conname := range table.Constraints {
		constraint, err := getValueFormat(s.Constraints, conname,
			"constraint %q not found for table %q", conname, table)
		if err != nil {
			return err
		}
		table.Constraints[conname] = constraint
	}
	return nil
}

func (s *Schema) fillTableIndexes(table *Table) error {
	for indname := range table.Indexes {
		index, err := getValueFormat(s.Indexes, indname,
			"index %q not found for table %q", indname, table)
		if err != nil {
			return err
		}
		table.Indexes[indname] = index
	}
	return nil
}

func (s *Schema) fillTableFk(table *Table) error {
	for fkname, fk := range table.ForeignKeys {
		fkey, err := getValueFormat(s.Constraints, fkname,
			"fk %q not found for table %q", fkname, table)
		if err != nil {
			return err
		}
		fk.Foreign = fkey
		ref, err := getValueFormat(s.Tables, fk.Reference.String(),
			"table %q not found for fk %q for table %q",
			fk.Reference, fkname, table)
		if err != nil {
			return err
		}
		fk.Reference = ref

		err = fillColumns(fk.ReferenceColumns, ref.Columns,
			"for fk table %q for fk %q in table %q", ref, fkname, table)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Schema) fillTableRefers(table *Table) error {
	for conname := range table.ReferencedBy {
		constraint, err := getValueFormat(s.Constraints, conname,
			"referenced by constraint %q not found for table %q",
			conname, table.Name)
		if err != nil {
			return err
		}
		table.ReferencedBy[conname] = constraint
	}
	return nil
}

func (s *Schema) fillTable(table *Table) error {
	if err := s.fillTableColumns(table); err != nil {
		return err
	}
	if err := s.fillTableConstraints(table); err != nil {
		return err
	}
	if err := s.fillTableIndexes(table); err != nil {
		return err
	}
	if err := s.fillTableFk(table); err != nil {
		return err
	}
	if err := s.fillTableRefers(table); err != nil {
		return err
	}

	pk, err := getValueFormat(s.Constraints, table.PrimaryKey.String(),
		"primary key %q not found for table %q",
		table.PrimaryKey, table)
	if err != nil {
		return err
	}
	table.PrimaryKey = pk

	return nil
}

func (s *Schema) fillTables() error {
	for _, table := range s.Tables {
		if err := s.fillTable(table); err != nil {
			return err
		}
	}
	return nil
}

func (s *Schema) UnmarshalJSON(data []byte) error {
	type ss Schema
	var temp ss
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*s = Schema(temp)

	if err := s.fillConcreteTypes(); err != nil {
		return err
	}
	if err := s.fillTypes(); err != nil {
		return err
	}
	if err := s.fillIndexes(); err != nil {
		return err
	}
	if err := s.fillConstraints(); err != nil {
		return err
	}
	if err := s.fillTables(); err != nil {
		return err
	}
	return nil
}
