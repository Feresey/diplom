package parse

import (
	"fmt"

	"github.com/Feresey/mtest/parse/query"
	"github.com/Feresey/mtest/schema"
)

// Перевод значений колонки pg_constraint.type.
var pgConstraintType = map[string]schema.ConstraintType{
	"p": schema.ConstraintTypePK,
	"f": schema.ConstraintTypeFK,
	"c": schema.ConstraintTypeCheck,
	"u": schema.ConstraintTypeUnique,
	"t": schema.ConstraintTypeTrigger,
	"x": schema.ConstraintTypeExclusion,
}

// Перевод значений колонки pg_type.typtype.
var pgTypType = map[string]schema.DataType{
	"b": schema.DataTypeBase,
	"c": schema.DataTypeComposite,
	"d": schema.DataTypeDomain,
	"e": schema.DataTypeEnum,
	"r": schema.DataTypeRange,
	"m": schema.DataTypeMultiRange,
	"p": schema.DataTypePseudo,
}

type parseSchema struct {
	typesLoadOrder []int
	typesByOID     map[int]schema.DBType
	types          map[int]query.Type

	enumList []int
	enums    map[int]query.Enum

	tables map[int]parseTable

	constraints      map[int]query.Constraint
	constraintsByOID map[int]schema.Constraint
	indexes          map[int]query.Index
}

type parseTable struct {
	table   query.Table
	columns map[int]query.Column
}

func (ps *parseSchema) convertToSchema() (*schema.Schema, error) {
	s := &schema.Schema{
		Types:  make(map[string]schema.DBType),
		Enums:  make(map[string]schema.EnumType),
		Tables: make(map[string]schema.Table),
	}

	reverse(ps.typesLoadOrder)

	if err := ps.convertTypes(s); err != nil {
		return nil, err
	}
	if err := ps.convertTables(s); err != nil {
		return nil, err
	}
	if err := ps.convertConstraints(s); err != nil {
		return nil, err
	}
	if err := ps.convertIndexes(s); err != nil {
		return nil, err
	}
	return s, nil
}

func (ps *parseSchema) convertTypes(s *schema.Schema) error {
	for _, typeOID := range ps.typesLoadOrder {
		typ, ok := ps.types[typeOID]
		if !ok {
			return fmt.Errorf("internal error: type with oid %d not found", typeOID)
		}
		t, err := ps.fillType(&typ)
		if err != nil {
			return err
		}
		ps.typesByOID[typeOID] = t
	}

	for _, typ := range ps.typesByOID {
		s.Types[typ.String()] = typ
		if typ.TypType() == schema.DataTypeEnum {
			// panic?
			s.Enums[typ.String()] = *typ.(*schema.EnumType)
		}
	}
	return nil
}

func (ps *parseSchema) getType(oid int) (schema.DBType, error) {
	elem, ok := ps.typesByOID[oid]
	if !ok {
		return nil, fmt.Errorf("type with oid %d not found", oid)
	}
	return elem, nil
}

func (ps *parseSchema) fillType(dbtype *query.Type) (schema.DBType, error) {
	typType, ok := pgTypType[dbtype.TypeType]
	if !ok {
		return nil, fmt.Errorf("typtype value is undefined: %q", dbtype.TypeType)
	}
	if dbtype.IsArray {
		typType = schema.DataTypeArray
	}

	baseType := schema.BaseType{
		TypeName: schema.Identifier{
			OID:    dbtype.TypeOID,
			Schema: dbtype.TypeSchema,
			Name:   dbtype.TypeName,
		},
		Type: typType,
	}

	switch typType {
	default:
		return nil, fmt.Errorf("data type is undefined: %s", typType)
	case schema.DataTypeBase:
	case schema.DataTypeArray:
		elem, err := ps.getType(int(dbtype.ElemTypeOID.Int32))
		if err != nil {
			return nil, err
		}
		return &schema.ElemType{
			BaseType: baseType,
			ElemType: elem,
		}, nil
	case schema.DataTypeEnum:
		values, ok := ps.enums[baseType.GetOID()]
		if !ok {
			return nil, fmt.Errorf("values for enum %q not found", baseType.String())
		}
		return &schema.EnumType{
			BaseType: baseType,
			Values:   values.Values,
		}, nil
	case schema.DataTypeRange:
		elem, err := ps.getType(int(dbtype.RangeElementTypeOID.Int32))
		if err != nil {
			return nil, err
		}
		return &schema.ElemType{
			BaseType: baseType,
			ElemType: elem,
		}, nil
	case schema.DataTypeMultiRange:
	// TODO add multirange type
	case schema.DataTypeComposite:
	// TODO add composite type
	case schema.DataTypeDomain:
		elem, err := ps.getType(int(dbtype.ElemTypeOID.Int32))
		if err != nil {
			return nil, err
		}
		return &schema.DomainType{
			Attributes: schema.DomainAttributes{
				NotNullable:      !dbtype.DomainIsNotNullable,
				HasCharMaxLength: dbtype.DomainCharacterMaxSize.Valid,
				CharMaxLength:    int(dbtype.DomainCharacterMaxSize.Int32),
				ArrayDims:        dbtype.DomainArrayDims,
				IsNumeric:        dbtype.DomainIsNumeric,
				NumericPrecision: int(dbtype.DomainNumericPrecision.Int32),
				NumericScale:     int(dbtype.DomainNumericScale.Int32),
			},
			ElemType: elem,
		}, nil
	case schema.DataTypePseudo:
	}
	return &baseType, nil
}

func (ps *parseSchema) convertTables(s *schema.Schema) error {
	for _, table := range ps.tables {
		t := schema.Table{
			Name: schema.Identifier{
				OID:    table.table.OID,
				Schema: table.table.Schema,
				Name:   table.table.Table,
			},
			Columns:      make(map[string]schema.Column),
			PrimaryKey:   nil,
			ForeignKeys:  make(map[string]schema.ForeignKey),
			ReferencedBy: make(map[string]schema.Constraint),
			Constraints:  make(map[string]schema.Constraint),
			Indexes:      make(map[string]schema.Index),
		}

		for _, col := range table.columns {
			typ, err := ps.getType(col.TypeOID)
			if err != nil {
				return err
			}

			t.Columns[col.ColumnName] = schema.Column{
				ColNum: col.ColumnNum,
				Name:   col.ColumnName,
				Type:   typ,
				Attributes: schema.ColumnAttributes{
					HasDefault:  col.HasDefault,
					IsGenerated: col.IsGenerated,
					Default:     col.DefaultExpr.String,
					DomainAttributes: schema.DomainAttributes{
						NotNullable:      col.IsNullable,
						HasCharMaxLength: col.CharacterMaxLength.Valid,
						CharMaxLength:    int(col.CharacterMaxLength.Int32),
						ArrayDims:        col.ArrayDims,
						IsNumeric:        col.IsNumeric,
						NumericPrecision: int(col.NumericPriecision.Int32),
						NumericScale:     int(col.NumericScale.Int32),
					},
				},
			}
		}

		s.Tables[t.String()] = t
	}

	return nil
}

func (ps *parseSchema) getTable(s *schema.Schema, oid int) (dbtable parseTable, table schema.Table, err error) {
	dbtable, ok := ps.tables[oid]
	if !ok {
		return dbtable, table, fmt.Errorf("table with oid %d not found", oid)
	}
	tableName := schema.Identifier{Schema: dbtable.table.Schema, Name: dbtable.table.Table}
	table, ok = s.Tables[tableName.String()]
	if !ok {
		return dbtable, table, fmt.Errorf("table %q not found", tableName)
	}
	return dbtable, table, nil
}

func (ps *parseSchema) convertConstraints(s *schema.Schema) error {
	for _, dbconstraint := range ps.constraints {
		dbtable, table, err := ps.getTable(s, dbconstraint.TableOID)
		if err != nil {
			return err
		}

		typ, ok := pgConstraintType[dbconstraint.ConstraintType]
		if !ok {
			return fmt.Errorf("unsupported constraint type: %q", dbconstraint.ConstraintType)
		}

		cols, err := ps.checkTableColumns(dbconstraint.Colnums, &dbtable)
		if err != nil {
			return err
		}

		c := schema.Constraint{
			OID:        dbconstraint.ConstraintOID,
			Name:       dbconstraint.ConstraintName,
			Type:       typ,
			Index:      nil,
			Definition: dbconstraint.ConstraintDef,
			Columns:    cols,
		}

		ps.constraintsByOID[c.GetOID()] = c
		table.Constraints[c.String()] = c

		switch c.Type {
		case schema.ConstraintTypePK:
			// PRIMARY KEY либо один либо нет его
			table.PrimaryKey = &c
			return nil
		case schema.ConstraintTypeFK:
			dbreftable, reftable, err := ps.getTable(s, dbconstraint.TableOID)
			if err != nil {
				return err
			}

			refcols, err := ps.checkTableColumns(dbconstraint.ForeignColnums, &dbreftable)
			if err != nil {
				return err
			}
			table.ForeignKeys[reftable.String()] = schema.ForeignKey{
				Constraint:       c,
				ReferenceTable:   reftable.String(),
				ReferenceColumns: refcols,
			}
		}
	}

	return nil
}

func (ps *parseSchema) convertIndexes(s *schema.Schema) error {
	for _, dbindex := range ps.indexes {
		dbtable, table, err := ps.getTable(s, dbindex.TableOID)
		if err != nil {
			return err
		}
		cols, err := ps.checkTableColumns(dbindex.Columns, &dbtable)
		if err != nil {
			return err
		}

		index := schema.Index{
			OID:                dbindex.IndexOID,
			Name:               dbindex.IndexName,
			Columns:            cols,
			Definition:         dbindex.IndexDefinition,
			IsUnique:           dbindex.IsUnique,
			IsPrimary:          dbindex.IsPrimary,
			IsNullsNotDistinct: dbindex.IsNullsNotDistinct,
		}

		table.Indexes[index.String()] = index

		if dbindex.ConstraintOID.Valid {
			conOID := int(dbindex.ConstraintOID.Int32)
			c, ok := ps.constraintsByOID[conOID]
			if !ok {
				return fmt.Errorf("constraint with oid %d not found for index %q",
					conOID, index.String())
			}

			c.Index = &index
			table.Constraints[c.String()] = c
		}
	}
	return nil
}

func (ps *parseSchema) checkTableColumns(
	colnums []int,
	table *parseTable,
) ([]string, error) {
	cols := make([]string, 0, len(colnums))
	for _, colnum := range colnums {
		tcol, ok := table.columns[colnum]
		if !ok {
			return nil, fmt.Errorf("column %d not found in table %d", colnum, table.table.OID)
		}
		cols = append(cols, tcol.ColumnName)
	}
	return cols, nil
}

func reverse[S ~[]E, E any](s S) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}
