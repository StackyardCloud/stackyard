package server

import (
	"bytes"
	"math"
	"strconv"
	"strings"
)

var rdsDataTableConstraints = map[string]bool{
	"PRIMARY": true, "UNIQUE": true, "KEY": true, "INDEX": true, "CONSTRAINT": true, "FOREIGN": true, "CHECK": true,
}

func (db rdsDataDB) createTable(p *rdsDataParser) (rdsDataResult, error) {
	ifNotExists := p.acceptWords("IF", "NOT", "EXISTS")
	name, err := p.ident()
	if err != nil {
		return rdsDataResult{}, err
	}
	if err := p.expectPunct("("); err != nil {
		return rdsDataResult{}, err
	}
	table := &rdsDataTable{Name: name, NextID: 1}
	for {
		def, err := p.columnDef()
		if err != nil {
			return rdsDataResult{}, err
		}
		if def != nil {
			if _, err := table.column(def.Name); err == nil {
				return rdsDataResult{}, rdsDataDBErr("duplicate column %q", def.Name)
			}
			def.Table = name
			table.Columns = append(table.Columns, *def)
		}
		if p.acceptPunct(",") {
			continue
		}
		if err := p.expectPunct(")"); err != nil {
			return rdsDataResult{}, err
		}
		break
	}
	if err := p.end(); err != nil {
		return rdsDataResult{}, err
	}
	if len(table.Columns) == 0 {
		return rdsDataResult{}, rdsDataDBErr("table %q needs at least one column", name)
	}
	if _, exists := db[strings.ToLower(name)]; exists {
		if ifNotExists {
			return rdsDataResult{}, nil
		}
		return rdsDataResult{}, rdsDataDBErr("table %q already exists", name)
	}
	db[strings.ToLower(name)] = table
	return rdsDataResult{}, nil
}

// columnDef reads one entry of a CREATE TABLE list up to the next top-level
// comma or closing parenthesis. Table constraints return nil.
func (p *rdsDataParser) columnDef() (*rdsDataColumn, error) {
	first := p.peek()
	if first.kind != "word" {
		return nil, rdsDataUnsupported("expected a column definition near %q", first.text)
	}
	constraint := rdsDataTableConstraints[strings.ToUpper(first.text)]
	var col rdsDataColumn
	if !constraint {
		col.Name = p.next().text
		typ := p.next()
		if typ.kind != "word" {
			return nil, rdsDataUnsupported("expected a type for column %q", col.Name)
		}
		col.TypeName = strings.ToUpper(typ.text)
	}
	for depth := 0; ; {
		t := p.peek()
		if t.kind == "" {
			return nil, rdsDataUnsupported("unterminated column list")
		}
		if t.kind == "punct" && depth == 0 && (t.text == "," || t.text == ")") {
			break
		}
		p.pos++
		switch {
		case t.kind == "punct" && t.text == "(":
			depth++
		case t.kind == "punct" && t.text == ")":
			depth--
		case t.kind == "word" && (strings.EqualFold(t.text, "AUTO_INCREMENT") || strings.EqualFold(t.text, "AUTOINCREMENT")):
			col.AutoInc = true
		}
	}
	if constraint {
		return nil, nil
	}
	col.Kind = rdsDataKind(col.TypeName)
	if strings.HasSuffix(col.TypeName, "SERIAL") {
		col.AutoInc = true
	}
	return &col, nil
}

func rdsDataKind(typeName string) string {
	switch typeName {
	case "INT", "INTEGER", "BIGINT", "SMALLINT", "TINYINT", "MEDIUMINT", "SERIAL", "BIGSERIAL", "SMALLSERIAL", "INT2", "INT4", "INT8":
		return "long"
	case "FLOAT", "DOUBLE", "REAL", "DECIMAL", "NUMERIC", "FLOAT4", "FLOAT8":
		return "double"
	case "BOOL", "BOOLEAN", "BIT":
		return "bool"
	case "BLOB", "BYTEA", "BINARY", "VARBINARY", "LONGBLOB", "MEDIUMBLOB", "TINYBLOB":
		return "blob"
	}
	return "string"
}

// rdsDataCoerce converts v to the column's storage kind.
func rdsDataCoerce(col rdsDataColumn, v any) (any, error) {
	if v == nil {
		return nil, nil
	}
	bad := rdsDataDBErr("incorrect %s value for column %q", strings.ToLower(col.TypeName), col.Name)
	switch col.Kind {
	case "long":
		switch x := v.(type) {
		case int64:
			return x, nil
		case float64:
			if x != math.Trunc(x) {
				return nil, bad
			}
			return int64(x), nil
		case bool:
			if x {
				return int64(1), nil
			}
			return int64(0), nil
		case string:
			n, err := strconv.ParseInt(strings.TrimSpace(x), 10, 64)
			if err != nil {
				return nil, bad
			}
			return n, nil
		}
	case "double":
		switch x := v.(type) {
		case int64:
			return float64(x), nil
		case float64:
			return x, nil
		case string:
			f, err := strconv.ParseFloat(strings.TrimSpace(x), 64)
			if err != nil {
				return nil, bad
			}
			return f, nil
		}
	case "bool":
		switch x := v.(type) {
		case bool:
			return x, nil
		case int64:
			return x != 0, nil
		case string:
			b, err := strconv.ParseBool(strings.TrimSpace(x))
			if err != nil {
				return nil, bad
			}
			return b, nil
		}
	case "blob":
		switch x := v.(type) {
		case []byte:
			return append([]byte(nil), x...), nil
		case string:
			return []byte(x), nil
		}
	default:
		switch x := v.(type) {
		case string:
			return x, nil
		case int64:
			return strconv.FormatInt(x, 10), nil
		case float64:
			return strconv.FormatFloat(x, 'f', -1, 64), nil
		case bool:
			return strconv.FormatBool(x), nil
		case []byte:
			return string(x), nil
		}
	}
	return nil, bad
}

func (db rdsDataDB) insert(p *rdsDataParser) (rdsDataResult, error) {
	name, err := p.ident()
	if err != nil {
		return rdsDataResult{}, err
	}
	table, err := db.table(name)
	if err != nil {
		return rdsDataResult{}, err
	}
	indexes := make([]int, 0, len(table.Columns))
	if p.acceptPunct("(") {
		for {
			colName, err := p.ident()
			if err != nil {
				return rdsDataResult{}, err
			}
			i, err := table.column(colName)
			if err != nil {
				return rdsDataResult{}, err
			}
			indexes = append(indexes, i)
			if !p.acceptPunct(",") {
				break
			}
		}
		if err := p.expectPunct(")"); err != nil {
			return rdsDataResult{}, err
		}
	} else {
		for i := range table.Columns {
			indexes = append(indexes, i)
		}
	}
	if err := p.expectWord("VALUES"); err != nil {
		return rdsDataResult{}, err
	}
	var result rdsDataResult
	for {
		if err := p.expectPunct("("); err != nil {
			return rdsDataResult{}, err
		}
		row := make([]any, len(table.Columns))
		set := make([]bool, len(table.Columns))
		for n := 0; ; n++ {
			if n >= len(indexes) {
				return rdsDataResult{}, rdsDataDBErr("column count doesn't match value count")
			}
			v, err := p.value()
			if err != nil {
				return rdsDataResult{}, err
			}
			if row[indexes[n]], err = rdsDataCoerce(table.Columns[indexes[n]], v); err != nil {
				return rdsDataResult{}, err
			}
			set[indexes[n]] = true
			if !p.acceptPunct(",") {
				if n+1 != len(indexes) {
					return rdsDataResult{}, rdsDataDBErr("column count doesn't match value count")
				}
				break
			}
		}
		if err := p.expectPunct(")"); err != nil {
			return rdsDataResult{}, err
		}
		for i, col := range table.Columns {
			if !col.AutoInc {
				continue
			}
			if !set[i] || row[i] == nil {
				row[i] = table.NextID
			}
			if id, ok := row[i].(int64); ok && id >= table.NextID {
				table.NextID = id + 1
			}
			result.Generated = append(result.Generated, row[i])
		}
		table.Rows = append(table.Rows, row)
		result.Updated++
		if !p.acceptPunct(",") {
			break
		}
	}
	return result, p.end()
}

type rdsDataCond struct {
	index int
	value any
}

// where reads an optional WHERE clause of `col = value` terms joined by AND.
func (p *rdsDataParser) where(table *rdsDataTable) ([]rdsDataCond, error) {
	if !p.acceptWords("WHERE") {
		return nil, nil
	}
	var conds []rdsDataCond
	for {
		colName, err := p.ident()
		if err != nil {
			return nil, err
		}
		i, err := table.column(colName)
		if err != nil {
			return nil, err
		}
		if err := p.expectPunct("="); err != nil {
			return nil, err
		}
		v, err := p.value()
		if err != nil {
			return nil, err
		}
		if v, err = rdsDataCoerce(table.Columns[i], v); err != nil {
			return nil, err
		}
		conds = append(conds, rdsDataCond{index: i, value: v})
		if !p.acceptWords("AND") {
			return conds, nil
		}
	}
}

// rdsDataMatch applies SQL equality: NULL never equals anything.
func rdsDataMatch(row []any, conds []rdsDataCond) bool {
	for _, c := range conds {
		got := row[c.index]
		if got == nil || c.value == nil {
			return false
		}
		if b, ok := got.([]byte); ok {
			if other, ok := c.value.([]byte); !ok || !bytes.Equal(b, other) {
				return false
			}
			continue
		}
		if got != c.value {
			return false
		}
	}
	return true
}

func (db rdsDataDB) selectRows(p *rdsDataParser) (rdsDataResult, error) {
	var names []string
	if !p.acceptPunct("*") {
		for {
			colName, err := p.ident()
			if err != nil {
				return rdsDataResult{}, err
			}
			names = append(names, colName)
			if !p.acceptPunct(",") {
				break
			}
		}
	}
	if err := p.expectWord("FROM"); err != nil {
		return rdsDataResult{}, err
	}
	name, err := p.ident()
	if err != nil {
		return rdsDataResult{}, err
	}
	table, err := db.table(name)
	if err != nil {
		return rdsDataResult{}, err
	}
	indexes := make([]int, 0, len(table.Columns))
	if names == nil {
		for i := range table.Columns {
			indexes = append(indexes, i)
		}
	}
	for _, colName := range names {
		i, err := table.column(colName)
		if err != nil {
			return rdsDataResult{}, err
		}
		indexes = append(indexes, i)
	}
	conds, err := p.where(table)
	if err != nil {
		return rdsDataResult{}, err
	}
	if err := p.end(); err != nil {
		return rdsDataResult{}, err
	}
	result := rdsDataResult{Query: true, Rows: [][]any{}}
	for _, i := range indexes {
		result.Columns = append(result.Columns, table.Columns[i])
	}
	for _, row := range table.Rows {
		if !rdsDataMatch(row, conds) {
			continue
		}
		out := make([]any, len(indexes))
		for n, i := range indexes {
			out[n] = row[i]
		}
		result.Rows = append(result.Rows, out)
	}
	return result, nil
}

func (db rdsDataDB) update(p *rdsDataParser) (rdsDataResult, error) {
	name, err := p.ident()
	if err != nil {
		return rdsDataResult{}, err
	}
	table, err := db.table(name)
	if err != nil {
		return rdsDataResult{}, err
	}
	if err := p.expectWord("SET"); err != nil {
		return rdsDataResult{}, err
	}
	var sets []rdsDataCond
	for {
		colName, err := p.ident()
		if err != nil {
			return rdsDataResult{}, err
		}
		i, err := table.column(colName)
		if err != nil {
			return rdsDataResult{}, err
		}
		if err := p.expectPunct("="); err != nil {
			return rdsDataResult{}, err
		}
		v, err := p.value()
		if err != nil {
			return rdsDataResult{}, err
		}
		if v, err = rdsDataCoerce(table.Columns[i], v); err != nil {
			return rdsDataResult{}, err
		}
		sets = append(sets, rdsDataCond{index: i, value: v})
		if !p.acceptPunct(",") {
			break
		}
	}
	conds, err := p.where(table)
	if err != nil {
		return rdsDataResult{}, err
	}
	if err := p.end(); err != nil {
		return rdsDataResult{}, err
	}
	var result rdsDataResult
	for _, row := range table.Rows {
		if !rdsDataMatch(row, conds) {
			continue
		}
		for _, set := range sets {
			row[set.index] = set.value
		}
		result.Updated++
	}
	return result, nil
}

func (db rdsDataDB) deleteRows(p *rdsDataParser) (rdsDataResult, error) {
	name, err := p.ident()
	if err != nil {
		return rdsDataResult{}, err
	}
	table, err := db.table(name)
	if err != nil {
		return rdsDataResult{}, err
	}
	conds, err := p.where(table)
	if err != nil {
		return rdsDataResult{}, err
	}
	if err := p.end(); err != nil {
		return rdsDataResult{}, err
	}
	kept := table.Rows[:0]
	var result rdsDataResult
	for _, row := range table.Rows {
		if rdsDataMatch(row, conds) {
			result.Updated++
			continue
		}
		kept = append(kept, row)
	}
	table.Rows = kept
	return result, nil
}
