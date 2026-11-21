package server

import (
	"fmt"
	"strings"
)

type rdsDataPendingKey struct {
	primary bool
	name    string
	cols    []string
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
	var keys []rdsDataPendingKey
	for {
		key, handled, err := p.tableConstraint()
		if err != nil {
			return rdsDataResult{}, err
		}
		if key != nil {
			keys = append(keys, *key)
		} else if !handled {
			col, colKey, err := p.columnDef(name)
			if err != nil {
				return rdsDataResult{}, err
			}
			if _, err := table.column(col.Name); err == nil {
				return rdsDataResult{}, rdsDataDBErr("column %q specified more than once", col.Name)
			}
			table.Columns = append(table.Columns, col)
			if colKey != nil {
				keys = append(keys, *colKey)
			}
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
	for _, key := range keys {
		if err := table.addKey(key); err != nil {
			return rdsDataResult{}, err
		}
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

func (t *rdsDataTable) addKey(key rdsDataPendingKey) error {
	unique := rdsDataUnique{Name: key.name}
	for _, colName := range key.cols {
		i, err := t.column(colName)
		if err != nil {
			return err
		}
		unique.Cols = append(unique.Cols, i)
		if key.primary {
			t.Columns[i].NotNull = true
		}
	}
	if unique.Name == "" {
		unique.Name = t.Name + "_" + strings.Join(key.cols, "_") + "_key"
		if key.primary {
			unique.Name = t.Name + "_pkey"
		}
	}
	t.Uniques = append(t.Uniques, unique)
	return nil
}

// tableConstraint reads a table-level PRIMARY KEY or UNIQUE constraint. Other
// table-level entries (KEY, INDEX, FOREIGN KEY, CHECK) are skipped and
// reported as handled; a column definition returns handled false.
func (p *rdsDataParser) tableConstraint() (*rdsDataPendingKey, bool, error) {
	start := p.pos
	name := ""
	if p.acceptWords("CONSTRAINT") {
		n, err := p.ident()
		if err != nil {
			return nil, false, err
		}
		name = n
	}
	var key rdsDataPendingKey
	switch {
	case p.acceptWords("PRIMARY", "KEY"):
		key.primary = true
	case p.acceptWords("UNIQUE"):
		if !p.acceptWords("KEY") {
			p.acceptWords("INDEX")
		}
		if p.isName() {
			n, _ := p.ident()
			name = n
		}
	case p.isWord("KEY", "INDEX", "FOREIGN", "CHECK") || name != "":
		p.skipToEntryEnd()
		return nil, true, nil
	default:
		p.pos = start
		return nil, false, nil
	}
	key.name = name
	if err := p.expectPunct("("); err != nil {
		return nil, false, err
	}
	for {
		col, err := p.ident()
		if err != nil {
			return nil, false, err
		}
		key.cols = append(key.cols, col)
		if !p.acceptPunct(",") {
			break
		}
	}
	if err := p.expectPunct(")"); err != nil {
		return nil, false, err
	}
	p.skipToEntryEnd()
	return &key, true, nil
}

// skipToEntryEnd skips tokens up to the next top-level comma or closing
// parenthesis of the CREATE TABLE list.
func (p *rdsDataParser) skipToEntryEnd() {
	for depth := 0; p.pos < len(p.toks); p.pos++ {
		t := p.peek()
		if t.kind == "punct" && depth == 0 && (t.text == "," || t.text == ")") {
			return
		}
		if t.kind == "punct" && t.text == "(" {
			depth++
		} else if t.kind == "punct" && t.text == ")" {
			depth--
		}
	}
}

// columnDef reads `name TYPE [(n[,m])] [modifiers]`. A column-level PRIMARY
// KEY or UNIQUE comes back as a key.
func (p *rdsDataParser) columnDef(table string) (rdsDataColumn, *rdsDataPendingKey, error) {
	colName, err := p.ident()
	if err != nil {
		return rdsDataColumn{}, nil, err
	}
	typ := p.next()
	if typ.kind != "word" {
		return rdsDataColumn{}, nil, rdsDataUnsupported("expected a type for column %q", colName)
	}
	col := rdsDataColumn{Name: colName, TypeName: strings.ToUpper(typ.text), Table: table}
	col.Kind = rdsDataKind(col.TypeName)
	col.AutoInc = strings.HasSuffix(col.TypeName, "SERIAL")
	var key *rdsDataPendingKey
	for depth := 0; p.pos < len(p.toks); {
		t := p.peek()
		if t.kind == "punct" && depth == 0 && (t.text == "," || t.text == ")") {
			break
		}
		switch {
		case p.acceptWords("PRIMARY", "KEY"):
			key = &rdsDataPendingKey{primary: true, cols: []string{colName}}
		case p.acceptWords("UNIQUE"):
			p.acceptWords("KEY")
			key = &rdsDataPendingKey{cols: []string{colName}}
		case p.acceptWords("NOT", "NULL"):
			col.NotNull = true
		case p.acceptWords("DEFAULT"):
			v, err := p.value() // literals only; DEFAULT now() and the like are unsupported
			if err != nil {
				return rdsDataColumn{}, nil, err
			}
			if col.Default, err = rdsDataCoerce(col, v); err != nil {
				return rdsDataColumn{}, nil, err
			}
			col.HasDefault = true
		case p.acceptWords("AUTO_INCREMENT"), p.acceptWords("AUTOINCREMENT"), p.acceptWords("IDENTITY"):
			col.AutoInc = true
		default:
			if t.kind == "punct" && t.text == "(" {
				depth++
			} else if t.kind == "punct" && t.text == ")" {
				depth--
			}
			p.pos++
		}
	}
	return col, key, nil
}

func (db rdsDataDB) dropTable(p *rdsDataParser) (rdsDataResult, error) {
	ifExists := p.acceptWords("IF", "EXISTS")
	name, err := p.ident()
	if err != nil {
		return rdsDataResult{}, err
	}
	if err := p.end(); err != nil {
		return rdsDataResult{}, err
	}
	if _, ok := db[strings.ToLower(name)]; !ok && !ifExists {
		return rdsDataResult{}, rdsDataDBErr("table %q does not exist", name)
	}
	delete(db, strings.ToLower(name))
	return rdsDataResult{}, nil
}

func rdsDataFormatValue(v any) string {
	if b, ok := v.([]byte); ok {
		return string(b)
	}
	return fmt.Sprint(v)
}

// checkRow enforces NOT NULL and the unique keys for row i against every other
// row. NULLs never collide. ponytail: O(rows) per check; index keys if tables grow.
func (t *rdsDataTable) checkRow(i int) error {
	row := t.Rows[i]
	for c, col := range t.Columns {
		if col.NotNull && row[c] == nil {
			return rdsDataDBErr("null value in column %q of relation %q violates not-null constraint", col.Name, t.Name)
		}
	}
	for _, key := range t.Uniques {
		for j, other := range t.Rows {
			if j != i && rdsDataSameKey(t, key, row, other) {
				names, values := make([]string, 0, len(key.Cols)), make([]string, 0, len(key.Cols))
				for _, c := range key.Cols {
					names = append(names, t.Columns[c].Name)
					values = append(values, rdsDataFormatValue(row[c]))
				}
				return rdsDataDBErr("duplicate key value violates unique constraint %q: Key (%s)=(%s) already exists",
					key.Name, strings.Join(names, ", "), strings.Join(values, ", "))
			}
		}
	}
	return nil
}

func rdsDataSameKey(t *rdsDataTable, key rdsDataUnique, a, b []any) bool {
	for _, c := range key.Cols {
		if a[c] == nil || b[c] == nil || rdsDataCompare(t.Columns[c].Kind, a[c], b[c]) != 0 {
			return false
		}
	}
	return true
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
		row, err := p.insertRow(table, indexes)
		if err != nil {
			return rdsDataResult{}, err
		}
		set := make([]bool, len(table.Columns))
		for _, i := range indexes {
			set[i] = true
		}
		for i, col := range table.Columns {
			switch {
			case col.AutoInc && row[i] == nil:
				row[i] = table.NextID
				result.Generated = append(result.Generated, row[i])
			case !set[i] && col.HasDefault:
				row[i] = col.Default
			}
			if id, ok := row[i].(int64); ok && col.AutoInc && id >= table.NextID {
				table.NextID = id + 1
			}
		}
		table.Rows = append(table.Rows, row)
		if err := table.checkRow(len(table.Rows) - 1); err != nil {
			return rdsDataResult{}, err
		}
		result.Updated++
		if !p.acceptPunct(",") {
			break
		}
	}
	return result, p.end()
}

func (p *rdsDataParser) insertRow(table *rdsDataTable, indexes []int) ([]any, error) {
	if err := p.expectPunct("("); err != nil {
		return nil, err
	}
	row := make([]any, len(table.Columns))
	for n := 0; ; n++ {
		if n >= len(indexes) {
			return nil, rdsDataDBErr("INSERT has more expressions than target columns")
		}
		v, err := p.value()
		if err != nil {
			return nil, err
		}
		if row[indexes[n]], err = rdsDataCoerce(table.Columns[indexes[n]], v); err != nil {
			return nil, err
		}
		if !p.acceptPunct(",") {
			if n+1 != len(indexes) {
				return nil, rdsDataDBErr("INSERT has more target columns than expressions")
			}
			break
		}
	}
	return row, p.expectPunct(")")
}

type rdsDataCond struct {
	index int
	op    string // "=", "null", "notnull"
	value any
}

// where reads an optional WHERE clause of `col = value` and `col IS [NOT] NULL`
// terms joined by AND.
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
		cond := rdsDataCond{index: i, op: "="}
		switch {
		case p.acceptWords("IS", "NOT", "NULL"):
			cond.op = "notnull"
		case p.acceptWords("IS", "NULL"):
			cond.op = "null"
		default:
			if err := p.expectPunct("="); err != nil {
				return nil, err
			}
			v, err := p.value()
			if err != nil {
				return nil, err
			}
			if cond.value, err = rdsDataCoerce(table.Columns[i], v); err != nil {
				return nil, err
			}
		}
		conds = append(conds, cond)
		if !p.acceptWords("AND") {
			return conds, nil
		}
	}
}

// rdsDataMatch applies the conditions with SQL NULL semantics: `= NULL` never matches.
func rdsDataMatch(table *rdsDataTable, row []any, conds []rdsDataCond) bool {
	for _, c := range conds {
		got := row[c.index]
		switch c.op {
		case "null":
			if got != nil {
				return false
			}
		case "notnull":
			if got == nil {
				return false
			}
		default:
			if got == nil || c.value == nil || rdsDataCompare(table.Columns[c.index].Kind, got, c.value) != 0 {
				return false
			}
		}
	}
	return true
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
	var changed []int
	for r, row := range table.Rows {
		if !rdsDataMatch(table, row, conds) {
			continue
		}
		for _, set := range sets {
			row[set.index] = set.value
		}
		changed = append(changed, r)
		result.Updated++
	}
	for _, r := range changed {
		if err := table.checkRow(r); err != nil {
			return rdsDataResult{}, err
		}
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
	kept := make([][]any, 0, len(table.Rows))
	var result rdsDataResult
	for _, row := range table.Rows {
		if rdsDataMatch(table, row, conds) {
			result.Updated++
			continue
		}
		kept = append(kept, row)
	}
	table.Rows = kept
	return result, nil
}
