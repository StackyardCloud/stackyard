package server

import (
	"cmp"
	"fmt"
	"math"
	"math/big"
	"sort"
	"strconv"
	"strings"
)

type rsdKind int

const (
	rsdKindString rsdKind = iota
	rsdKindLong
	rsdKindDouble
	rsdKindBool
	rsdKindNumeric
	rsdKindBlob
)

type rsdType struct {
	name      string // canonical Redshift type name reported in ColumnMetadata.typeName
	kind      rsdKind
	length    int
	precision int
	scale     int
	min, max  int64 // integer range
}

func rsdResolveType(words string, args []int) (rsdType, error) {
	arg := func(i, def int) int {
		if i < len(args) {
			return args[i]
		}
		return def
	}
	switch words {
	case "smallint", "int2":
		return rsdType{name: "int2", kind: rsdKindLong, precision: 5, min: math.MinInt16, max: math.MaxInt16}, nil
	case "int", "integer", "int4":
		return rsdType{name: "int4", kind: rsdKindLong, precision: 10, min: math.MinInt32, max: math.MaxInt32}, nil
	case "bigint", "int8":
		return rsdType{name: "int8", kind: rsdKindLong, precision: 19, min: math.MinInt64, max: math.MaxInt64}, nil
	case "real", "float4":
		return rsdType{name: "float4", kind: rsdKindDouble, precision: 8}, nil
	case "double precision", "float8", "float":
		return rsdType{name: "float8", kind: rsdKindDouble, precision: 17}, nil
	case "decimal", "numeric":
		p, sc := arg(0, 18), arg(1, 0)
		if p < 1 || p > 38 || sc < 0 || sc > p {
			return rsdType{}, fmt.Errorf("NUMERIC precision %d must be between 1 and 38 and scale %d between 0 and the precision", p, sc)
		}
		return rsdType{name: "numeric", kind: rsdKindNumeric, precision: p, scale: sc}, nil
	case "boolean", "bool":
		return rsdType{name: "bool", kind: rsdKindBool, precision: 1}, nil
	case "varchar", "character varying", "nvarchar", "text":
		n := arg(0, 256)
		if n < 1 || n > 65535 {
			return rsdType{}, fmt.Errorf("length for type varchar must be between 1 and 65535")
		}
		return rsdType{name: "varchar", kind: rsdKindString, length: n, precision: n}, nil
	case "char", "character", "nchar", "bpchar":
		n := arg(0, 1)
		if n < 1 || n > 4096 {
			return rsdType{}, fmt.Errorf("length for type character must be between 1 and 4096")
		}
		return rsdType{name: "bpchar", kind: rsdKindString, length: n, precision: n}, nil
	case "date", "time", "timetz", "timestamptz", "super":
		return rsdType{name: words, kind: rsdKindString}, nil
	case "timestamp", "timestamp without time zone":
		return rsdType{name: "timestamp", kind: rsdKindString}, nil
	case "timestamp with time zone":
		return rsdType{name: "timestamptz", kind: rsdKindString}, nil
	case "varbyte", "varbinary", "binary varying":
		n := arg(0, 64000)
		return rsdType{name: "varbyte", kind: rsdKindBlob, length: n, precision: n}, nil
	}
	return rsdType{}, fmt.Errorf("type \"%s\" does not exist", words)
}

type rsdTable struct {
	schema, name string
	cols         []rsdColumnDef
	rows         [][]any
	identity     map[int]int64 // next identity value per column index
}

func (t *rsdTable) colIndex(name string) int {
	for i, c := range t.cols {
		if c.name == name {
			return i
		}
	}
	return -1
}

func (t *rsdTable) clone() *rsdTable {
	cp := *t
	cp.rows = make([][]any, len(t.rows))
	for i, r := range t.rows {
		cp.rows[i] = append([]any(nil), r...) // cell values are immutable (strings, numbers, []byte never mutated)
	}
	cp.identity = make(map[int]int64, len(t.identity))
	for k, v := range t.identity {
		cp.identity[k] = v
	}
	return &cp
}

type rsdDatabase struct {
	schemas map[string]bool
	tables  map[string]*rsdTable // "schema.table"
}

func newRSDDatabase() *rsdDatabase {
	return &rsdDatabase{schemas: map[string]bool{"public": true}, tables: map[string]*rsdTable{}}
}

func (d *rsdDatabase) clone() *rsdDatabase {
	cp := &rsdDatabase{schemas: make(map[string]bool, len(d.schemas)), tables: make(map[string]*rsdTable, len(d.tables))}
	for k, v := range d.schemas {
		cp.schemas[k] = v
	}
	for k, v := range d.tables {
		cp.tables[k] = v.clone()
	}
	return cp
}

func (d *rsdDatabase) table(schema, name string) (*rsdTable, error) {
	t, ok := d.tables[schema+"."+name]
	if !ok {
		return nil, fmt.Errorf("relation \"%s.%s\" does not exist", schema, name)
	}
	return t, nil
}

type rsdOutCol struct {
	name, table, schema string
	typ                 rsdType
	nullable            bool
	def                 string
}

type rsdResult struct {
	hasResultSet bool
	cols         []rsdOutCol
	rows         [][]any
	affected     int64
}

// rsdCoerce converts a literal or bound parameter to a column's type.
func rsdCoerce(e rsdExpr, typ rsdType, params map[string]string) (any, error) {
	kind, text := e.kind, e.text
	if kind == rsdTokParam {
		v, ok := params[text]
		if !ok {
			return nil, fmt.Errorf("parameter \":%s\" is not provided", text)
		}
		kind, text = rsdTokString, v
	}
	if kind == rsdTokWord && text == "null" {
		return nil, nil
	}
	switch typ.kind {
	case rsdKindLong:
		if kind == rsdTokWord {
			return nil, fmt.Errorf("column is of type %s but expression is of type boolean", typ.name)
		}
		n, err := strconv.ParseInt(strings.TrimSpace(text), 10, 64)
		if err != nil {
			if f, ferr := strconv.ParseFloat(strings.TrimSpace(text), 64); ferr == nil && kind == rsdTokNumber && f == math.Trunc(f) && math.Abs(f) < 9e18 {
				n, err = int64(f), nil
			}
		}
		if err != nil {
			return nil, fmt.Errorf("invalid input syntax for integer: \"%s\"", text)
		}
		if n < typ.min || n > typ.max {
			return nil, fmt.Errorf("value %d is out of range for type %s", n, typ.name)
		}
		return n, nil
	case rsdKindDouble:
		f, err := strconv.ParseFloat(strings.TrimSpace(text), 64)
		if kind == rsdTokWord || err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
			return nil, fmt.Errorf("invalid input syntax for type double precision: \"%s\"", text)
		}
		return f, nil
	case rsdKindNumeric:
		t := strings.TrimSpace(text)
		if _, ok := new(big.Rat).SetString(t); kind == rsdTokWord || !ok {
			return nil, fmt.Errorf("invalid input syntax for type numeric: \"%s\"", text)
		}
		return t, nil // ponytail: kept as text so scale survives; no rounding to the declared scale
	case rsdKindBool:
		switch strings.ToLower(strings.TrimSpace(text)) {
		case "true", "t", "yes", "y", "on", "1":
			if kind != rsdTokNumber {
				return true, nil
			}
		case "false", "f", "no", "n", "off", "0":
			if kind != rsdTokNumber {
				return false, nil
			}
		}
		return nil, fmt.Errorf("invalid input syntax for type boolean: \"%s\"", text)
	case rsdKindBlob:
		if len(text) > typ.length {
			return nil, fmt.Errorf("value too long for type varbyte(%d)", typ.length)
		}
		return []byte(text), nil
	}
	if typ.length > 0 && len(text) > typ.length {
		return nil, fmt.Errorf("value too long for type character varying(%d)", typ.length)
	}
	return text, nil
}

// rsdLiteralType infers the column type of a bare literal in SELECT.
func rsdLiteralType(e rsdExpr) rsdType {
	switch {
	case e.kind == rsdTokNumber && !strings.ContainsAny(e.text, ".eE"):
		if n, err := strconv.ParseInt(e.text, 10, 64); err == nil && n >= math.MinInt32 && n <= math.MaxInt32 {
			return rsdType{name: "int4", kind: rsdKindLong, precision: 10, min: math.MinInt32, max: math.MaxInt32}
		}
		return rsdType{name: "int8", kind: rsdKindLong, precision: 19, min: math.MinInt64, max: math.MaxInt64}
	case e.kind == rsdTokNumber:
		return rsdType{name: "numeric", kind: rsdKindNumeric, precision: 18}
	case e.kind == rsdTokWord && e.text != "null":
		return rsdType{name: "bool", kind: rsdKindBool, precision: 1}
	}
	return rsdType{name: "varchar", kind: rsdKindString, length: 65535, precision: 65535}
}

// rsdCompare orders two non-NULL values of the same column kind: text and
// binary compare as bytes, NUMERIC exactly as rationals, numbers natively.
func rsdCompare(kind rsdKind, a, b any) int {
	switch kind {
	case rsdKindLong:
		return cmp.Compare(a.(int64), b.(int64))
	case rsdKindDouble:
		return cmp.Compare(a.(float64), b.(float64))
	case rsdKindBool:
		return cmp.Compare(rsdBoolInt(a.(bool)), rsdBoolInt(b.(bool)))
	case rsdKindBlob:
		return strings.Compare(string(a.([]byte)), string(b.([]byte)))
	case rsdKindNumeric:
		x, okx := new(big.Rat).SetString(a.(string))
		y, oky := new(big.Rat).SetString(b.(string))
		if okx && oky {
			return x.Cmp(y)
		}
	}
	return strings.Compare(a.(string), b.(string))
}

func rsdBoolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

type rsdBoundCond struct {
	col  int
	kind rsdKind
	op   string
	val  any
}

// bindWhere resolves columns and coerces values once, so an unknown column,
// missing parameter or bad literal fails even when no row is examined.
func (t *rsdTable) bindWhere(where []rsdCond, params map[string]string) ([]rsdBoundCond, error) {
	out := make([]rsdBoundCond, 0, len(where))
	for _, c := range where {
		i := t.colIndex(c.col)
		if i < 0 {
			return nil, fmt.Errorf("column \"%s\" does not exist", c.col)
		}
		b := rsdBoundCond{col: i, kind: t.cols[i].typ.kind, op: c.op}
		if c.op == "=" {
			v, err := rsdCoerce(c.expr, t.cols[i].typ, params)
			if err != nil {
				return nil, err
			}
			b.val = v
		}
		out = append(out, b)
	}
	return out, nil
}

func rsdMatch(row []any, conds []rsdBoundCond) bool {
	for _, c := range conds {
		v := row[c.col]
		switch {
		case c.op == "isnull" && v != nil, c.op == "notnull" && v == nil:
			return false
		case c.op == "=" && (v == nil || c.val == nil || rsdCompare(c.kind, v, c.val) != 0):
			return false
		}
	}
	return true
}

func (t *rsdTable) outCol(i int) rsdOutCol {
	c := t.cols[i]
	oc := rsdOutCol{name: c.name, table: t.name, schema: t.schema, typ: c.typ, nullable: !c.notNull}
	if c.hasDefault {
		oc.def = c.def.text
	}
	if c.identity {
		oc.def = fmt.Sprintf("\"identity\"(%d, %d)", c.seed, c.step)
	}
	return oc
}

// rsdExec runs one parsed statement against a database. Callers clone the
// database first for writes, so a failure part-way leaves nothing applied.
func rsdExec(db *rsdDatabase, st *rsdStmt, params map[string]string) (rsdResult, error) {
	switch st.verb {
	case "create_schema":
		if db.schemas[st.schema] {
			if st.ifNotExists {
				return rsdResult{}, nil
			}
			return rsdResult{}, fmt.Errorf("Schema \"%s\" already exists", st.schema)
		}
		db.schemas[st.schema] = true
		return rsdResult{}, nil
	case "create_table":
		if !db.schemas[st.schema] {
			return rsdResult{}, fmt.Errorf("schema \"%s\" does not exist", st.schema)
		}
		key := st.schema + "." + st.table
		if _, ok := db.tables[key]; ok {
			if st.ifNotExists {
				return rsdResult{}, nil
			}
			return rsdResult{}, fmt.Errorf("relation \"%s\" already exists", st.table)
		}
		t := &rsdTable{schema: st.schema, name: st.table, identity: map[int]int64{}}
		for i, c := range st.columns {
			if t.colIndex(c.name) >= 0 {
				return rsdResult{}, fmt.Errorf("column \"%s\" specified more than once", c.name)
			}
			if c.hasDefault {
				if _, err := rsdCoerce(c.def, c.typ, nil); err != nil {
					return rsdResult{}, err
				}
			}
			if c.identity {
				t.identity[i] = c.seed
			}
			t.cols = append(t.cols, c)
		}
		db.tables[key] = t
		return rsdResult{}, nil
	case "drop_table":
		key := st.schema + "." + st.table
		if _, ok := db.tables[key]; !ok {
			if st.ifExists {
				return rsdResult{}, nil
			}
			return rsdResult{}, fmt.Errorf("Table \"%s\" does not exist", st.table)
		}
		delete(db.tables, key)
		return rsdResult{}, nil
	case "select":
		return rsdSelect(db, st, params)
	}
	t, err := db.table(st.schema, st.table)
	if err != nil {
		return rsdResult{}, err
	}
	switch st.verb {
	case "insert":
		return rsdInsert(t, st, params)
	case "update":
		return rsdUpdate(t, st, params)
	}
	conds, err := t.bindWhere(st.where, params)
	if err != nil {
		return rsdResult{}, err
	}
	var kept [][]any
	var n int64
	for _, row := range t.rows {
		if rsdMatch(row, conds) {
			n++
		} else {
			kept = append(kept, row)
		}
	}
	t.rows = kept
	return rsdResult{affected: n}, nil
}

func rsdInsert(t *rsdTable, st *rsdStmt, params map[string]string) (rsdResult, error) {
	idx := make([]int, 0, len(t.cols))
	if len(st.insertCols) == 0 {
		for i := range t.cols {
			idx = append(idx, i)
		}
	}
	for _, name := range st.insertCols {
		i := t.colIndex(name)
		if i < 0 {
			return rsdResult{}, fmt.Errorf("column \"%s\" of relation \"%s\" does not exist", name, t.name)
		}
		idx = append(idx, i)
	}
	for _, exprs := range st.rows {
		if len(exprs) != len(idx) {
			return rsdResult{}, fmt.Errorf("INSERT has more expressions than target columns")
		}
		row := make([]any, len(t.cols))
		set := make([]bool, len(t.cols))
		for j, e := range exprs {
			if t.cols[idx[j]].identity {
				return rsdResult{}, fmt.Errorf("cannot set an identity column to a value")
			}
			v, err := rsdCoerce(e, t.cols[idx[j]].typ, params)
			if err != nil {
				return rsdResult{}, err
			}
			row[idx[j]], set[idx[j]] = v, true
		}
		for i, c := range t.cols {
			switch {
			case c.identity:
				row[i] = t.identity[i]
				t.identity[i] += c.step
			case !set[i] && c.hasDefault:
				row[i], _ = rsdCoerce(c.def, c.typ, nil)
			}
			if c.notNull && row[i] == nil {
				return rsdResult{}, fmt.Errorf("Cannot insert a NULL value into column %s", c.name)
			}
		}
		t.rows = append(t.rows, row)
	}
	return rsdResult{affected: int64(len(st.rows))}, nil
}

func rsdUpdate(t *rsdTable, st *rsdStmt, params map[string]string) (rsdResult, error) {
	type set struct {
		i int
		v any
	}
	var sets []set
	for _, s := range st.sets {
		i := t.colIndex(s.col)
		if i < 0 {
			return rsdResult{}, fmt.Errorf("column \"%s\" of relation \"%s\" does not exist", s.col, t.name)
		}
		v, err := rsdCoerce(s.expr, t.cols[i].typ, params)
		if err != nil {
			return rsdResult{}, err
		}
		if v == nil && t.cols[i].notNull {
			return rsdResult{}, fmt.Errorf("Cannot set a NULL value to column %s", t.cols[i].name)
		}
		sets = append(sets, set{i, v})
	}
	conds, err := t.bindWhere(st.where, params)
	if err != nil {
		return rsdResult{}, err
	}
	var n int64
	for ri, row := range t.rows {
		if !rsdMatch(row, conds) {
			continue
		}
		updated := append([]any(nil), row...)
		for _, s := range sets {
			updated[s.i] = s.v
		}
		t.rows[ri] = updated
		n++
	}
	return rsdResult{affected: n}, nil
}

func rsdSelect(db *rsdDatabase, st *rsdStmt, params map[string]string) (rsdResult, error) {
	res := rsdResult{hasResultSet: true}
	if !st.hasFrom {
		row := make([]any, 0, len(st.items))
		for _, it := range st.items {
			if it.count { // COUNT(*) without FROM counts the single implicit row
				name := it.alias
				if name == "" {
					name = "count"
				}
				res.cols = append(res.cols, rsdOutCol{name: name, typ: rsdType{name: "int8", kind: rsdKindLong, precision: 19, min: math.MinInt64, max: math.MaxInt64}})
				row = append(row, int64(1))
				continue
			}
			if it.lit == nil {
				return rsdResult{}, fmt.Errorf("column reference requires a FROM clause")
			}
			typ := rsdLiteralType(*it.lit)
			v, err := rsdCoerce(*it.lit, typ, params)
			if err != nil {
				return rsdResult{}, err
			}
			name := it.alias
			if name == "" {
				name = "?column?"
			}
			res.cols = append(res.cols, rsdOutCol{name: name, typ: typ, nullable: true})
			row = append(row, v)
		}
		if st.limit != 0 {
			res.rows = [][]any{row}
		}
		return res, nil
	}
	t, err := db.table(st.schema, st.table)
	if err != nil {
		return rsdResult{}, err
	}
	conds, err := t.bindWhere(st.where, params)
	if err != nil {
		return rsdResult{}, err
	}
	var rows [][]any
	for _, row := range t.rows {
		if rsdMatch(row, conds) {
			rows = append(rows, row)
		}
	}
	if st.orderBy != "" {
		oi := t.colIndex(st.orderBy)
		if oi < 0 {
			return rsdResult{}, fmt.Errorf("column \"%s\" does not exist", st.orderBy)
		}
		kind := t.cols[oi].typ.kind
		sort.SliceStable(rows, func(a, b int) bool {
			x, y := rows[a][oi], rows[b][oi]
			if x == nil || y == nil { // NULLs sort last ascending, first descending
				if st.desc {
					return x == nil && y != nil
				}
				return y == nil && x != nil
			}
			if st.desc {
				return rsdCompare(kind, y, x) < 0
			}
			return rsdCompare(kind, x, y) < 0
		})
	}
	counting, total := false, int64(len(rows))
	for _, it := range st.items {
		counting = counting || it.count
	}
	type pick struct {
		col int
		lit *rsdExpr
		typ rsdType
	}
	var picks []pick
	for _, it := range st.items {
		switch {
		case it.star:
			if counting {
				return rsdResult{}, fmt.Errorf("column \"%s.*\" must appear in the GROUP BY clause or be used in an aggregate function", t.name)
			}
			for i := range t.cols {
				res.cols = append(res.cols, t.outCol(i))
				picks = append(picks, pick{col: i})
			}
		case it.count:
			name := it.alias
			if name == "" {
				name = "count"
			}
			typ := rsdType{name: "int8", kind: rsdKindLong, precision: 19, min: math.MinInt64, max: math.MaxInt64}
			res.cols = append(res.cols, rsdOutCol{name: name, typ: typ})
			picks = append(picks, pick{col: -1})
		case it.lit != nil:
			typ := rsdLiteralType(*it.lit)
			name := it.alias
			if name == "" {
				name = "?column?"
			}
			res.cols = append(res.cols, rsdOutCol{name: name, typ: typ, nullable: true})
			picks = append(picks, pick{col: -2, lit: it.lit, typ: typ})
		default:
			i := t.colIndex(it.col)
			if i < 0 {
				return rsdResult{}, fmt.Errorf("column \"%s\" does not exist in %s", it.col, t.name)
			}
			if counting {
				return rsdResult{}, fmt.Errorf("column \"%s.%s\" must appear in the GROUP BY clause or be used in an aggregate function", t.name, it.col)
			}
			oc := t.outCol(i)
			if it.alias != "" {
				oc.name = it.alias
			}
			res.cols = append(res.cols, oc)
			picks = append(picks, pick{col: i})
		}
	}
	project := func(row []any) ([]any, error) {
		out := make([]any, 0, len(picks))
		for _, p := range picks {
			switch p.col {
			case -1:
				out = append(out, total)
			case -2:
				v, err := rsdCoerce(*p.lit, p.typ, params)
				if err != nil {
					return nil, err
				}
				out = append(out, v)
			default:
				out = append(out, row[p.col])
			}
		}
		return out, nil
	}
	if counting {
		rows = [][]any{nil}
	}
	if st.limit >= 0 && st.limit < len(rows) {
		rows = rows[:st.limit]
	}
	for _, row := range rows {
		out, err := project(row)
		if err != nil {
			return rsdResult{}, err
		}
		res.rows = append(res.rows, out)
	}
	return res, nil
}
