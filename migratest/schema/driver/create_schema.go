package driver

import (
	"context"
	"fmt"
	"sort"

	"github.com/Feresey/diplom/migratest/schema"
	"github.com/volatiletech/strmangle"
	"go.uber.org/zap"
)

// Tables returns the metadata for all tables, minus the tables
// specified in the blacklist.
func (p *PostgresDriver) Tables(ctx context.Context) ([]schema.Table, error) {
	var err error

	names, err := p.TableNames(ctx)
	if err != nil {
		return nil, fmt.Errorf("get table names: %w", err)
	}

	sort.Strings(names)
	p.logger.Info("", zap.Strings("tables", names))

	tables := make([]schema.Table, 0, len(names))
	for _, tableName := range names {
		t := schema.Table{
			SchemaName: p.sc.SchemaName,
			Name:       tableName,
		}

		if t.Columns, err = p.Columns(ctx, tableName); err != nil {
			return nil, err
		}

		if t.PKey, err = p.PrimaryKeyInfo(ctx, tableName); err != nil {
			return nil, fmt.Errorf("get PK info for %s: %w", tableName, err)
		}

		if t.FKeys, err = p.ForeignKeyInfo(ctx, tableName); err != nil {
			return nil, fmt.Errorf("get FK info for %s: %w", tableName, err)
		}

		p.filterForeignKeys(&t)

		setIsJoinTable(&t)

		tables = append(tables, t)
	}

	// Relationships have a dependency on foreign key nullability.
	for i := range tables {
		tbl := &tables[i]
		setForeignKeyConstraints(tbl, tables)
	}
	for i := range tables {
		tbl := &tables[i]
		setRelationships(tbl, tables)
	}

	return tables, nil
}

// filterForeignKeys filter FK whose ForeignTable is not in whitelist or in blacklist.
func (p *PostgresDriver) filterForeignKeys(t *schema.Table) {
	var fkeys []schema.ForeignKey
	for _, fkey := range t.FKeys {
		if (len(p.sc.Whitelist) == 0 || strmangle.SetInclude(fkey.ForeignTable, p.sc.Whitelist)) &&
			(len(p.sc.Blacklist) == 0 || !strmangle.SetInclude(fkey.ForeignTable, p.sc.Blacklist)) {
			fkeys = append(fkeys, fkey)
		}
	}
	t.FKeys = fkeys
}

// setIsJoinTable if there are:
// - a composite primary key involving two columns.
// - both primary key columns are also foreign keys.
func setIsJoinTable(t *schema.Table) {
	if t.PKey == nil || len(t.PKey.Columns) != 2 || len(t.FKeys) < 2 || len(t.Columns) > 2 {
		return
	}

	for _, c := range t.PKey.Columns {
		found := false
		for _, f := range t.FKeys {
			if c == f.Column {
				found = true
				break
			}
		}
		if !found {
			return
		}
	}

	t.IsJoinTable = true
}

func setForeignKeyConstraints(t *schema.Table, tables []schema.Table) {
	for i, fkey := range t.FKeys {
		localColumn := t.GetColumn(fkey.Column)
		foreignTable := schema.GetTable(tables, fkey.ForeignTable)
		foreignColumn := foreignTable.GetColumn(fkey.ForeignColumn)

		t.FKeys[i].Nullable = localColumn.Nullable
		t.FKeys[i].Unique = localColumn.Unique
		t.FKeys[i].ForeignColumnNullable = foreignColumn.Nullable
		t.FKeys[i].ForeignColumnUnique = foreignColumn.Unique
	}
}

func setRelationships(t *schema.Table, tables []schema.Table) {
	t.ToOneRelationships = toOneRelationships(*t, tables)
	t.ToManyRelationships = toManyRelationships(*t, tables)
}

// ToOneRelationships relationship lookups
// Input should be the sql name of a table like: videos
func ToOneRelationships(table string, tables []schema.Table) []schema.ToOneRelationship {
	localTable := schema.GetTable(tables, table)

	return toOneRelationships(localTable, tables)
}

// ToManyRelationships relationship lookups
// Input should be the sql name of a table like: videos
func ToManyRelationships(table string, tables []schema.Table) []schema.ToManyRelationship {
	localTable := schema.GetTable(tables, table)

	return toManyRelationships(localTable, tables)
}

func toOneRelationships(table schema.Table, tables []schema.Table) []schema.ToOneRelationship {
	var relationships []schema.ToOneRelationship

	for _, t := range tables {
		for _, f := range t.FKeys {
			if f.ForeignTable == table.Name && !t.IsJoinTable && f.Unique {
				relationships = append(relationships, buildToOneRelationship(table, f, t, tables))
			}
		}
	}

	return relationships
}

func toManyRelationships(table schema.Table, tables []schema.Table) []schema.ToManyRelationship {
	var relationships []schema.ToManyRelationship

	for _, t := range tables {
		for _, f := range t.FKeys {
			if f.ForeignTable == table.Name && (t.IsJoinTable || !f.Unique) {
				relationships = append(relationships, buildToManyRelationship(table, f, t, tables))
			}
		}
	}

	return relationships
}

func buildToOneRelationship(
	localTable schema.Table,
	foreignKey schema.ForeignKey,
	foreignTable schema.Table,
	tables []schema.Table,
) schema.ToOneRelationship {
	return schema.ToOneRelationship{
		Name:     foreignKey.Name,
		Table:    localTable.Name,
		Column:   foreignKey.ForeignColumn,
		Nullable: foreignKey.ForeignColumnNullable,
		Unique:   foreignKey.ForeignColumnUnique,

		ForeignTable:          foreignTable.Name,
		ForeignColumn:         foreignKey.Column,
		ForeignColumnNullable: foreignKey.Nullable,
		ForeignColumnUnique:   foreignKey.Unique,
	}
}

func buildToManyRelationship(
	localTable schema.Table,
	foreignKey schema.ForeignKey,
	foreignTable schema.Table,
	tables []schema.Table,
) schema.ToManyRelationship {
	if !foreignTable.IsJoinTable {
		return schema.ToManyRelationship{
			Name:                  foreignKey.Name,
			Table:                 localTable.Name,
			Column:                foreignKey.ForeignColumn,
			Nullable:              foreignKey.ForeignColumnNullable,
			Unique:                foreignKey.ForeignColumnUnique,
			ForeignTable:          foreignTable.Name,
			ForeignColumn:         foreignKey.Column,
			ForeignColumnNullable: foreignKey.Nullable,
			ForeignColumnUnique:   foreignKey.Unique,
			ToJoinTable:           false,
		}
	}

	relationship := schema.ToManyRelationship{
		Table:    localTable.Name,
		Column:   foreignKey.ForeignColumn,
		Nullable: foreignKey.ForeignColumnNullable,
		Unique:   foreignKey.ForeignColumnUnique,

		ToJoinTable: true,
		JoinTable:   foreignTable.Name,

		JoinLocalFKeyName:       foreignKey.Name,
		JoinLocalColumn:         foreignKey.Column,
		JoinLocalColumnNullable: foreignKey.Nullable,
		JoinLocalColumnUnique:   foreignKey.Unique,
	}

	for _, fk := range foreignTable.FKeys {
		if fk.Name == foreignKey.Name {
			continue
		}

		relationship.JoinForeignFKeyName = fk.Name
		relationship.JoinForeignColumn = fk.Column
		relationship.JoinForeignColumnNullable = fk.Nullable
		relationship.JoinForeignColumnUnique = fk.Unique

		relationship.ForeignTable = fk.ForeignTable
		relationship.ForeignColumn = fk.ForeignColumn
		relationship.ForeignColumnNullable = fk.ForeignColumnNullable
		relationship.ForeignColumnUnique = fk.ForeignColumnUnique
	}

	return relationship
}
