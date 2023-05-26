package parse

import (
	"errors"
	"fmt"

	"github.com/Feresey/mtest/parse/query"
	"github.com/Feresey/mtest/schema"
	mapset "github.com/deckarep/golang-set/v2"
	"golang.org/x/xerrors"
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
	typesByOID map[int]schema.DBType
	types      map[int]query.Type

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

	if err := ps.convertTypes(s); err != nil {
		return nil, xerrors.Errorf("convert types: %w", err)
	}
	if err := ps.convertTables(s); err != nil {
		return nil, xerrors.Errorf("convert tables: %w", err)
	}
	if err := ps.convertConstraints(s); err != nil {
		return nil, xerrors.Errorf("convert constraints: %w", err)
	}
	if err := ps.convertIndexes(s); err != nil {
		return nil, xerrors.Errorf("convert indexes: %w", err)
	}
	return s, nil
}

func (ps *parseSchema) convertTypes(s *schema.Schema) error {
	types := mapset.NewThreadUnsafeSetFromMapKeys(ps.types)

	for types.Cardinality() != 0 {
		var rerr error
		types.Each(func(typeOID int) bool {
			typ, ok := ps.types[typeOID]
			if !ok {
				rerr = xerrors.Errorf("internal error: type with oid %d not found", typeOID)
				return false
			}
			t, err := ps.fillType(&typ)
			if err != nil {
				var terr typeNotFoundError
				if errors.As(err, &terr) {
					return true
				}
				rerr = xerrors.Errorf("fill type %q(%d): %w", typ.TypeName, typ.TypeOID, err)
				return false
			}
			ps.typesByOID[typeOID] = t
			types.Remove(typeOID)
			return true
		})
		if rerr != nil {
			return rerr
		}
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

type typeNotFoundError struct {
	OID int
	xerrors.Frame
}

func (t typeNotFoundError) Error() string {
	return fmt.Sprintf("type with oid %d not found", t.OID)
}

func getTypeError(oid int) error {
	return typeNotFoundError{OID: oid}
}

func (ps *parseSchema) fillType(dbtype *query.Type) (schema.DBType, error) {
	typType, ok := pgTypType[dbtype.TypeType]
	if !ok {
		return nil, xerrors.Errorf("typtype value is undefined: %q", dbtype.TypeType)
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
		return nil, xerrors.Errorf("data type is undefined: %s", typType)
	case schema.DataTypeBase:
	case schema.DataTypeArray:
		oid := int(dbtype.ElemTypeOID.Int32)
		elem, ok := ps.typesByOID[oid]
		if !ok {
			return nil, xerrors.Errorf("get array elem type: %w", getTypeError(oid))
		}
		return &schema.ElemType{
			BaseType: baseType,
			ElemType: elem,
		}, nil
	case schema.DataTypeEnum:
		values, ok := ps.enums[baseType.GetOID()]
		if !ok {
			return nil, xerrors.Errorf("values for enum %q not found", baseType.String())
		}
		return &schema.EnumType{
			BaseType: baseType,
			Values:   values.Values,
		}, nil
	case schema.DataTypeRange:
		elem, ok := ps.typesByOID[int(dbtype.RangeElementTypeOID.Int32)]
		if !ok {
			return nil, xerrors.Errorf("get range elem type: %w", getTypeError(int(dbtype.RangeElementTypeOID.Int32)))
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
		elem, ok := ps.typesByOID[int(dbtype.DomainTypeOID.Int32)]
		if !ok {
			return nil, xerrors.Errorf("get domain elem type: %w", getTypeError(int(dbtype.DomainTypeOID.Int32)))
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
			typ, ok := ps.typesByOID[col.TypeOID]
			if !ok {
				return xerrors.Errorf("get column type for table %q: %w", t, getTypeError(col.TypeOID))
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
		return dbtable, table, xerrors.Errorf("table with oid %d not found", oid)
	}
	tableName := schema.Identifier{Schema: dbtable.table.Schema, Name: dbtable.table.Table}
	table, ok = s.Tables[tableName.String()]
	if !ok {
		return dbtable, table, xerrors.Errorf("table %q not found", tableName)
	}
	return dbtable, table, nil
}

func (ps *parseSchema) convertConstraints(s *schema.Schema) error {
	for _, dbconstraint := range ps.constraints {
		dbtable, table, err := ps.getTable(s, dbconstraint.TableOID)
		if err != nil {
			return xerrors.Errorf("get table for constraint %q: %w", dbconstraint.ConstraintName, err)
		}

		typ, ok := pgConstraintType[dbconstraint.ConstraintType]
		if !ok {
			return xerrors.Errorf("unsupported constraint type: %q", dbconstraint.ConstraintType)
		}

		cols, err := ps.checkTableColumns(dbconstraint.Colnums, &dbtable)
		if err != nil {
			return xerrors.Errorf("check table %q columns for constraint %q: %w",
				table, dbconstraint.ConstraintName, err)
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
		case schema.ConstraintTypeFK:
			dbreftable, reftable, err := ps.getTable(s, dbconstraint.TableOID)
			if err != nil {
				return xerrors.Errorf("get ref table for table %q fk constraint %q: %w", table, c, err)
			}

			refcols, err := ps.checkTableColumns(dbconstraint.ForeignColnums, &dbreftable)
			if err != nil {
				return xerrors.Errorf("check ref table %q columns for fk constraint %q of table %q: %w",
					reftable, dbconstraint.ConstraintName, table, err)
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
			return xerrors.Errorf("get table for index %q: %w", dbindex.IndexName, err)
		}
		cols, err := ps.checkTableColumns(dbindex.Columns, &dbtable)
		if err != nil {
			return xerrors.Errorf("check table %q columns for index %q: %w",
				table, dbindex.IndexName, err)
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
				return xerrors.Errorf("constraint with oid %d not found for index %q on table %q",
					conOID, index.String(), table)
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
			return nil, xerrors.Errorf("column %d not found in table %d", colnum, table.table.OID)
		}
		cols = append(cols, tcol.ColumnName)
	}
	return cols, nil
}
