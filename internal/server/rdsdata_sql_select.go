package server

import (
	"sort"
	"strconv"
	"strings"
)

type rdsDataSelectItem struct {
	kind  string // star, count, column, literal
	name  string
	alias string
	value any
}

// selectItems reads the SELECT list: *, COUNT(*), columns or literals, each
// with an optional AS alias.
func (p *rdsDataParser) selectItems() ([]rdsDataSelectItem, error) {
	var items []rdsDataSelectItem
	for {
		var item rdsDataSelectItem
		switch {
		case p.acceptPunct("*"):
			item.kind = "star"
		case p.isWord("COUNT") && p.pos+1 < len(p.toks) && p.toks[p.pos+1].text == "(":
			p.pos++
			if err := p.expectPunct("("); err != nil {
				return nil, err
			}
			if err := p.expectPunct("*"); err != nil {
				return nil, rdsDataUnsupported("only COUNT(*) is supported")
			}
			if err := p.expectPunct(")"); err != nil {
				return nil, err
			}
			item.kind, item.name = "count", "count"
		case p.isName() && !p.isWord("NULL", "TRUE", "FALSE"):
			name, err := p.ident()
			if err != nil {
				return nil, err
			}
			item.kind, item.name = "column", name
		default:
			item.name = p.peek().text
			v, err := p.value()
			if err != nil {
				return nil, err
			}
			item.kind, item.value = "literal", v
		}
		if p.acceptWords("AS") {
			alias, err := p.ident()
			if err != nil {
				return nil, err
			}
			item.alias = alias
		}
		items = append(items, item)
		if !p.acceptPunct(",") {
			return items, nil
		}
	}
}

func (db rdsDataDB) selectRows(p *rdsDataParser) (rdsDataResult, error) {
	items, err := p.selectItems()
	if err != nil {
		return rdsDataResult{}, err
	}
	if !p.acceptWords("FROM") {
		return rdsDataLiteralSelect(p, items)
	}
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
	order, err := p.orderBy(table)
	if err != nil {
		return rdsDataResult{}, err
	}
	limit, err := p.limit()
	if err != nil {
		return rdsDataResult{}, err
	}
	if err := p.end(); err != nil {
		return rdsDataResult{}, err
	}

	// Collect matches into a fresh slice so sorting never reorders the table.
	matched := make([][]any, 0, len(table.Rows))
	for _, row := range table.Rows {
		if rdsDataMatch(table, row, conds) {
			matched = append(matched, row)
		}
	}
	if len(items) == 1 && items[0].kind == "count" {
		col := rdsDataColumn{Name: firstNonEmpty(items[0].alias, "count"), TypeName: "BIGINT", Kind: "long"}
		rows := [][]any{{int64(len(matched))}}
		if limit == 0 {
			rows = [][]any{}
		}
		return rdsDataResult{Query: true, Columns: []rdsDataColumn{col}, Rows: rows}, nil
	}
	rdsDataSort(table, matched, order)
	if limit >= 0 && int64(len(matched)) > limit {
		matched = matched[:limit]
	}

	var indexes []int
	result := rdsDataResult{Query: true, Rows: [][]any{}}
	for _, item := range items {
		switch item.kind {
		case "star":
			for i, col := range table.Columns {
				indexes = append(indexes, i)
				result.Columns = append(result.Columns, col)
			}
		case "column":
			i, err := table.column(item.name)
			if err != nil {
				return rdsDataResult{}, err
			}
			col := table.Columns[i]
			if item.alias != "" {
				col.Name = item.alias
			}
			indexes = append(indexes, i)
			result.Columns = append(result.Columns, col)
		default:
			return rdsDataResult{}, rdsDataUnsupported("COUNT(*) and literals can't be mixed with columns in a SELECT FROM")
		}
	}
	for _, row := range matched {
		out := make([]any, len(indexes))
		for n, i := range indexes {
			out[n] = row[i]
		}
		result.Rows = append(result.Rows, out)
	}
	return result, nil
}

// rdsDataLiteralSelect answers SELECT 1, SELECT :p AS x and similar with one row.
func rdsDataLiteralSelect(p *rdsDataParser, items []rdsDataSelectItem) (rdsDataResult, error) {
	if err := p.end(); err != nil {
		return rdsDataResult{}, err
	}
	result := rdsDataResult{Query: true}
	row := make([]any, 0, len(items))
	for _, item := range items {
		if item.kind != "literal" {
			if item.kind == "column" {
				return rdsDataResult{}, rdsDataDBErr("column %q does not exist", item.name)
			}
			return rdsDataResult{}, rdsDataUnsupported("SELECT without FROM accepts only literals and parameters")
		}
		col := rdsDataLiteralColumn(item.value)
		col.Name = firstNonEmpty(item.alias, item.name)
		v, err := rdsDataCoerce(col, item.value)
		if err != nil {
			return rdsDataResult{}, err
		}
		result.Columns = append(result.Columns, col)
		row = append(row, v)
	}
	result.Rows = [][]any{row}
	return result, nil
}

func rdsDataLiteralColumn(v any) rdsDataColumn {
	switch x := v.(type) {
	case rdsDataNumber:
		if _, err := strconv.ParseInt(string(x), 10, 64); err == nil {
			return rdsDataColumn{TypeName: "BIGINT", Kind: "long"}
		}
		return rdsDataColumn{TypeName: "NUMERIC", Kind: "decimal"}
	case int64:
		return rdsDataColumn{TypeName: "BIGINT", Kind: "long"}
	case float64:
		return rdsDataColumn{TypeName: "DOUBLE", Kind: "double"}
	case bool:
		return rdsDataColumn{TypeName: "BOOL", Kind: "bool"}
	case []byte:
		return rdsDataColumn{TypeName: "BYTEA", Kind: "blob"}
	}
	return rdsDataColumn{TypeName: "VARCHAR", Kind: "string"}
}

type rdsDataOrder struct {
	index int
	desc  bool
}

func (p *rdsDataParser) orderBy(table *rdsDataTable) ([]rdsDataOrder, error) {
	if !p.acceptWords("ORDER", "BY") {
		return nil, nil
	}
	var order []rdsDataOrder
	for {
		colName, err := p.ident()
		if err != nil {
			return nil, err
		}
		i, err := table.column(colName)
		if err != nil {
			return nil, err
		}
		item := rdsDataOrder{index: i}
		if p.acceptWords("DESC") {
			item.desc = true
		} else {
			p.acceptWords("ASC")
		}
		order = append(order, item)
		if !p.acceptPunct(",") {
			return order, nil
		}
	}
}

// limit reads an optional LIMIT n; -1 means no limit.
func (p *rdsDataParser) limit() (int64, error) {
	if !p.acceptWords("LIMIT") {
		return -1, nil
	}
	v, err := p.value()
	if err != nil {
		return 0, err
	}
	n, err := rdsDataCoerce(rdsDataColumn{Name: "LIMIT", TypeName: "BIGINT", Kind: "long"}, v)
	if err != nil || n == nil || n.(int64) < 0 {
		return 0, rdsDataDBErr("LIMIT must be a non-negative integer")
	}
	return n.(int64), nil
}

// rdsDataSort orders rows by the ORDER BY columns. NULLs sort first ascending
// (MySQL's order); rows that tie keep insertion order.
func rdsDataSort(table *rdsDataTable, rows [][]any, order []rdsDataOrder) {
	if len(order) == 0 {
		return
	}
	sort.SliceStable(rows, func(a, b int) bool {
		for _, o := range order {
			x, y := rows[a][o.index], rows[b][o.index]
			c := 0
			switch {
			case x == nil && y == nil:
			case x == nil:
				c = -1
			case y == nil:
				c = 1
			default:
				c = rdsDataCompare(table.Columns[o.index].Kind, x, y)
			}
			if o.desc {
				c = -c
			}
			if c != 0 {
				return c < 0
			}
		}
		return false
	})
}

// rdsDataSplitStatements splits ExecuteSql input on semicolons outside quotes
// and comments, using the same escape rules as the tokenizer.
func rdsDataSplitStatements(sql string) []string {
	var out []string
	start := 0
	flush := func(end int) {
		if part := strings.TrimSpace(sql[start:end]); part != "" {
			out = append(out, part)
		}
	}
	for i := 0; i < len(sql); i++ {
		switch c := sql[i]; {
		case c == '\'':
			for i++; i < len(sql); i++ {
				if sql[i] == '\\' {
					i++
				} else if sql[i] == '\'' {
					if i+1 < len(sql) && sql[i+1] == '\'' {
						i++
						continue
					}
					break
				}
			}
		case c == '"' || c == '`':
			if end := strings.IndexByte(sql[i+1:], c); end >= 0 {
				i += end + 1
			} else {
				i = len(sql)
			}
		case c == '-' && i+1 < len(sql) && sql[i+1] == '-':
			for i < len(sql) && sql[i] != '\n' {
				i++
			}
		case c == '/' && i+1 < len(sql) && sql[i+1] == '*':
			if end := strings.Index(sql[i+2:], "*/"); end >= 0 {
				i += end + 3
			} else {
				i = len(sql)
			}
		case c == ';':
			flush(i)
			start = i + 1
		}
	}
	flush(len(sql))
	return out
}
