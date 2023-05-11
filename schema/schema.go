package schema

import (
	"encoding/json"
	"fmt"
)

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
	Types          map[int]*DBType        `json:"types,omitempty"`
	ArrayTypes     map[int]*ArrayType     `json:"array_types,omitempty"`
	CompositeTypes map[int]*CompositeType `json:"composite_types,omitempty"`
	EnumTypes      map[int]*EnumType      `json:"enum_types,omitempty"`
	RangeTypes     map[int]*RangeType     `json:"range_types,omitempty"`
	DomainTypes    map[int]*DomainType    `json:"domain_types,omitempty"`

	Tables      map[int]*Table      `json:"tables,omitempty"`
	Constraints map[int]*Constraint `json:"constraints,omitempty"`
	Indexes     map[int]*Index      `json:"indexes,omitempty"`
}

// Table описывает таблицу базы данных.
type Table struct {
	// имя таблицы
	Name Identifier `json:"name,omitempty"`
	// мапа колонок, где ключ - имя колонки хранить имена как
	Columns map[int]*Column `json:"columns,omitempty"`

	// Главный ключ таблицы (может быть nil)
	PrimaryKey *Constraint `json:"primary_key,omitempty"`
	// Внешние ключи таблицы, ключ мапы - FK Name
	ForeignKeys map[int]*ForeignKey `json:"foreign_keys,omitempty"`
	// Ключи, которые ссылаются на эту таблицу
	ReferencedBy KeysMarshalMap[int, Constraint] `json:"referenced_by,omitempty"`

	// Список всех CONSTRAINT-ов текущей таблицы
	Constraints KeysMarshalMap[int, Constraint] `json:"constraints,omitempty"`
	// Список всех INDEX-ов текущей таблицы
	Indexes KeysMarshalMap[int, Index] `json:"indexes,omitempty"`
}

type KeysMarshalMap[K comparable, T any] map[K]*T

func (k KeysMarshalMap[K, T]) MarshalJSON() ([]byte, error) {
	arr := make([]K, 0, len(k))
	for key := range k {
		arr = append(arr, key)
	}
	return json.Marshal(arr)
}

func (k *KeysMarshalMap[K, T]) UnmarshalJSON(data []byte) error {
	var arr []K
	if err := json.Unmarshal(data, &arr); err != nil {
		return err
	}
	res := make(KeysMarshalMap[K, T], len(arr))
	for _, key := range arr {
		res[key] = nil
	}
	*k = res
	return nil
}

func (t *Table) String() string { return t.Name.String() }
func (t *Table) OID() int       { return t.Name.OID }

// ForeignKey описывает внешнюю связь
// В PostgreSQL FK может ссылаться на PRIMARY KEY CONSTRAINT, UNIQUE CONSTRAINT, UNIQUE INDEX.
type ForeignKey struct {
	// CONSTRAINT в текущей таблице
	// Foreign.Type == FK
	Foreign *Constraint `json:"-"`
	// Таблица, на которую ссылаются
	Reference *Table `json:"-"`
	// Список колонок во внешней таблице, на которые ссылается FOREIGN KEY
	ReferenceColumns KeysMarshalMap[int, Column] `json:"reference_columns,omitempty"`
}

func (f *ForeignKey) String() string { return f.Foreign.String() }

type (
	fkNotJSON ForeignKey
	fkJSON    struct {
		fkNotJSON
		ReferenceOID int `json:"reference,omitempty"`
	}
)

func (f ForeignKey) MarshalJSON() ([]byte, error) {
	return json.Marshal(fkJSON{
		fkNotJSON:    fkNotJSON(f),
		ReferenceOID: f.Reference.OID(),
	})
}

func (f *ForeignKey) UnmarshalJSON(data []byte) error {
	var v fkJSON
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*f = ForeignKey(v.fkNotJSON)
	f.Reference = &Table{
		Name: Identifier{
			OID: v.ReferenceOID,
		},
	}
	return nil
}

// Column описывает колонку таблицы.
type Column struct {
	// Порядковый номер колонки
	ColNum int `json:"col_num,omitempty"`
	// Имя колонки
	Name string `json:"name,omitempty"`
	// Таблица, которой принадлежит колонка
	Table *Table `json:"-"`
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

// DBType описывает тип данных базы.
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
func (d *DBType) OID() int       { return d.TypeName.OID }

type ArrayType struct {
	TypeName Identifier `json:"type_name,omitempty"`
	// Тип элемента массива. INTEGER[] -> INTEGER
	ElemType *DBType `json:"elem_type,omitempty"`
}

func (a *ArrayType) String() string { return a.TypeName.String() }
func (a *ArrayType) OID() int       { return a.TypeName.OID }

func (a *ArrayType) GetElemType() *DBType     { return a.ElemType }
func (a *ArrayType) SetElemType(elem *DBType) { a.ElemType = elem }

type EnumType struct {
	TypeName Identifier `json:"type_name,omitempty"`
	Values   []string   `json:"values,omitempty"`
}

func (e *EnumType) String() string { return e.TypeName.String() }
func (e *EnumType) OID() int       { return e.TypeName.OID }

type CompositeType struct {
	TypeName Identifier `json:"type_name,omitempty"`
	// Attributes map[string]*CompositeAttribute `json:"attributes,omitempty"`
}

func (c *CompositeType) String() string { return c.TypeName.String() }
func (c *CompositeType) OID() int       { return c.TypeName.OID }

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
func (d *DomainType) OID() int                 { return d.TypeName.OID }
func (d *DomainType) GetElemType() *DBType     { return d.ElemType }
func (d *DomainType) SetElemType(elem *DBType) { d.ElemType = elem }

type RangeType struct {
	TypeName Identifier `json:"type_name,omitempty"`
	ElemType *DBType    `json:"elem_type,omitempty"`
}

func (r *RangeType) String() string           { return r.TypeName.String() }
func (r *RangeType) OID() int                 { return r.TypeName.OID }
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

// ColumnAttributes описывает аттрибуты колонки.
type ColumnAttributes struct {
	DomainAttributes `json:",omitempty"` //nolint:tagliatelle // embed struct

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
	// Имя ограничения
	Name Identifier `json:"name,omitempty"`
	// Тип ограничения
	Type ConstraintType `json:"type,omitempty"`
	// Таблица, которой принадлежит ограничение
	Table *Table `json:"-"`
	// Индекс, на котором основано органичение (может быть пустым)
	Index *Index `json:"index,omitempty"`

	// Результат функции pg_getconstraintdef. Я не уверен что это вообще нужно, но пусть будет.
	Definition string `json:"definition,omitempty"`

	// Колонки, на которые действует ограничение
	// Колонки всегда принадлежат той же таблице, которой принадлежит ограничение
	// Количество колонок всегда >= 1
	// Ключ мапы - имя колонки
	Columns KeysMarshalMap[int, Column] `json:"columns,omitempty"`
}

func (c *Constraint) String() string {
	return fmt.Sprintf("%s.%s.%s", c.Name.Schema, c.Table, c.Name.Name)
}
func (c *Constraint) OID() int { return c.Name.OID }

type Index struct {
	// Имя индекса
	Name Identifier `json:"name,omitempty"`
	// Таблица, для которой создан индекс
	Table *Table `json:"-"`
	// Колонки, которые затрагивает индекс
	Columns KeysMarshalMap[int, Column] `json:"columns,omitempty"`
	// Определение индекса
	Definition string `json:"definition,omitempty"`

	IsUnique  bool `json:"is_unique,omitempty"`
	IsPrimary bool `json:"is_primary,omitempty"`
	// Только для UNIQUE индекса
	IsNullsNotDistinct bool `json:"is_nulls_not_distinct,omitempty"`
}

func (i *Index) String() string {
	return fmt.Sprintf("%s.%s.%s", i.Name.Schema, i.Table, i.Name.Name)
}
func (i *Index) OID() int { return i.Name.OID }

func getValueFormat[K comparable, T fmt.Stringer](
	m map[K]T, key K,
	msg string, args ...any,
) (T, error) {
	value, ok := m[key]
	if !ok {
		return value, fmt.Errorf(msg, args...)
	}
	return value, nil
}

func getElemType[K int, T fmt.Stringer](
	basename string,
	m map[K]T,
	typename K,
	basetype T,
) (T, error) {
	return getValueFormat(m, typename,
		"%s type with oid %d not found for type %s", basename, typename, basetype)
}

type Elementer interface {
	fmt.Stringer
	OID() int
	GetElemType() *DBType
	SetElemType(elem *DBType)
}

func fillElemType[T Elementer](m map[int]*DBType, base T) error {
	if elem := base.GetElemType(); elem != nil {
		typ, err := getValueFormat(m, elem.OID(),
			"elem type with oid %d not found for type %q", elem.OID(), base)
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
			typ.ArrayType, err = getElemType("array",
				s.ArrayTypes, typ.OID(),
				typ.ArrayType,
			)
		case typ.CompositeType != nil:
			typ.CompositeType, err = getElemType("composite",
				s.CompositeTypes,
				typ.OID(),
				typ.CompositeType,
			)
		case typ.DomainType != nil:
			typ.DomainType, err = getElemType("domain",
				s.DomainTypes,
				typ.OID(),
				typ.DomainType,
			)
		case typ.EnumType != nil:
			typ.EnumType, err = getElemType("enum",
				s.EnumTypes,
				typ.OID(),
				typ.EnumType,
			)
		case typ.RangeType != nil:
			typ.RangeType, err = getElemType("range",
				s.RangeTypes,
				typ.OID(),
				typ.RangeType,
			)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func fillColumns(dst, src map[int]*Column, msg string, args ...any) error {
	for colName := range dst {
		col, err := getValueFormat(src, colName,
			"column %q not found for"+msg, append([]any{colName}, args...)...)
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
		table, err := getValueFormat(s.Tables, index.Table.OID(),
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
		table, err := getValueFormat(s.Tables, constraint.Table.OID(),
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
			index, err := getValueFormat(s.Indexes, constraint.Index.OID(),
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
		typ, err := getValueFormat(s.Types, col.Type.OID(),
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
		constraint.Table = table
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
		index.Table = table
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
		ref, err := getValueFormat(s.Tables, fk.Reference.OID(),
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

	if table.PrimaryKey == nil {
		return nil
	}
	pk, err := getValueFormat(s.Constraints, table.PrimaryKey.OID(),
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
	type schema Schema
	var temp schema
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}
	*s = Schema(temp)

	if err := s.fillConcreteTypes(); err != nil {
		return err
	}
	if err := s.fillTypes(); err != nil {
		return err
	}
	if err := s.fillTables(); err != nil {
		return err
	}
	if err := s.fillIndexes(); err != nil {
		return err
	}
	if err := s.fillConstraints(); err != nil {
		return err
	}

	return nil
}
