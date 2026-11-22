package server

import (
	"bytes"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// ponytail: the RDS Data API emulation runs a small SQL subset, not a database
// engine. Supported: CREATE TABLE (PRIMARY KEY, UNIQUE, NOT NULL, literal
// DEFAULT, AUTO_INCREMENT/SERIAL/IDENTITY), DROP TABLE, INSERT ... VALUES,
// UPDATE ... SET, DELETE FROM, SELECT with *, columns, COUNT(*) or literals
// (SELECT 1 with no FROM), WHERE `col = value` / `col IS [NOT] NULL` terms
// joined by AND, ORDER BY columns and LIMIT n, plus -- and /* */ comments.
// Still unsupported, answered with BadRequestException: joins, OR, subqueries,
// GROUP BY, OFFSET, comparison operators other than =, functions other than
// COUNT(*), and DEFAULT expressions. Schema-qualified names ignore the schema.
// Swap in an embedded SQL engine if clients need more.

// rdsDataNumber is a numeric literal kept exactly as written, so DECIMAL
// columns keep their representation (12.50 stays "12.50").
type rdsDataNumber string

// rdsDataColumn is one column of an emulated table. Kind is the storage kind
// derived from the declared type: long, double, decimal, bool, blob or string.
type rdsDataColumn struct {
	Name       string
	TypeName   string
	Kind       string
	AutoInc    bool
	NotNull    bool
	HasDefault bool
	Default    any
	Table      string
}

type rdsDataUnique struct {
	Name string
	Cols []int
}

// rdsDataTable is one table. Version changes on every committed write, which
// lets transactions detect conflicting writes.
type rdsDataTable struct {
	Name    string
	Columns []rdsDataColumn
	Uniques []rdsDataUnique
	Rows    [][]any
	NextID  int64
	Version int64
}

// rdsDataDB holds one database's tables, keyed by lower-case table name.
// Values are nil, int64, float64, bool, string (also DECIMAL text) or []byte.
type rdsDataDB map[string]*rdsDataTable

type rdsDataResult struct {
	Query     bool
	Columns   []rdsDataColumn
	Rows      [][]any
	Updated   int64
	Generated []any
}

// rdsDataSQLError is a statement the engine rejects. Unsupported statements map
// to BadRequestException; the rest are database errors.
type rdsDataSQLError struct {
	Unsupported bool
	Msg         string
}

func (e *rdsDataSQLError) Error() string { return e.Msg }

func rdsDataDBErr(format string, args ...any) error {
	return &rdsDataSQLError{Msg: fmt.Sprintf(format, args...)}
}

func rdsDataUnsupported(format string, args ...any) error {
	return &rdsDataSQLError{Unsupported: true, Msg: fmt.Sprintf(format, args...)}
}

func (t *rdsDataTable) clone() *rdsDataTable {
	if t == nil {
		return nil
	}
	copied := *t
	copied.Columns = append([]rdsDataColumn(nil), t.Columns...)
	copied.Uniques = append([]rdsDataUnique(nil), t.Uniques...)
	copied.Rows = make([][]any, len(t.Rows))
	for i, row := range t.Rows {
		copied.Rows[i] = append([]any(nil), row...)
	}
	return &copied
}

type rdsDataToken struct {
	kind string // word, ident (quoted), string, number, param, punct
	text string
}

func rdsDataTokenize(sql string) ([]rdsDataToken, error) {
	var out []rdsDataToken
	for i := 0; i < len(sql); {
		c := sql[i]
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			i++
		case c == '-' && i+1 < len(sql) && sql[i+1] == '-':
			for i < len(sql) && sql[i] != '\n' {
				i++
			}
		case c == '/' && i+1 < len(sql) && sql[i+1] == '*':
			end := strings.Index(sql[i+2:], "*/")
			if end < 0 {
				return nil, rdsDataDBErr("unterminated comment")
			}
			i += end + 4
		case c == '\'':
			text, next, err := rdsDataStringLiteral(sql, i)
			if err != nil {
				return nil, err
			}
			out = append(out, rdsDataToken{"string", text})
			i = next
		case c == '`' || c == '"':
			end := strings.IndexByte(sql[i+1:], c)
			if end < 0 {
				return nil, rdsDataDBErr("unterminated quoted identifier")
			}
			out = append(out, rdsDataToken{"ident", sql[i+1 : i+1+end]})
			i += end + 2
		case c == ':' && i+1 < len(sql) && rdsDataIdentByte(sql[i+1]):
			j := i + 1
			for j < len(sql) && rdsDataIdentByte(sql[j]) {
				j++
			}
			out = append(out, rdsDataToken{"param", sql[i+1 : j]})
			i = j
		case c >= '0' && c <= '9' || c == '-' && i+1 < len(sql) && sql[i+1] >= '0' && sql[i+1] <= '9':
			j := i + 1
			for j < len(sql) && (sql[j] >= '0' && sql[j] <= '9' || sql[j] == '.') {
				j++
			}
			out = append(out, rdsDataToken{"number", sql[i:j]})
			i = j
		case rdsDataIdentByte(c):
			j := i
			for j < len(sql) && rdsDataIdentByte(sql[j]) {
				j++
			}
			out = append(out, rdsDataToken{"word", sql[i:j]})
			i = j
		case strings.IndexByte("(),=*;.", c) >= 0:
			out = append(out, rdsDataToken{"punct", string(c)})
			i++
		default:
			return nil, rdsDataUnsupported("unsupported character %q in SQL", c)
		}
	}
	return out, nil
}

// rdsDataStringLiteral reads a '...' literal starting at sql[start], accepting
// both doubled quotes ('it”s') and backslash escapes ('it\'s').
func rdsDataStringLiteral(sql string, start int) (string, int, error) {
	var b strings.Builder
	for j := start + 1; j < len(sql); j++ {
		switch c := sql[j]; {
		case c == '\\' && j+1 < len(sql):
			j++
			switch sql[j] {
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			case 'r':
				b.WriteByte('\r')
			case '0':
				b.WriteByte(0)
			default:
				b.WriteByte(sql[j])
			}
		case c == '\'' && j+1 < len(sql) && sql[j+1] == '\'':
			b.WriteByte('\'')
			j++
		case c == '\'':
			return b.String(), j + 1, nil
		default:
			b.WriteByte(c)
		}
	}
	return "", 0, rdsDataDBErr("unterminated string literal")
}

func rdsDataIdentByte(c byte) bool {
	return c == '_' || c == '$' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
}

type rdsDataParser struct {
	toks   []rdsDataToken
	pos    int
	params map[string]any
}

func (p *rdsDataParser) peek() rdsDataToken {
	if p.pos < len(p.toks) {
		return p.toks[p.pos]
	}
	return rdsDataToken{}
}

func (p *rdsDataParser) next() rdsDataToken {
	t := p.peek()
	p.pos++
	return t
}

// isWord matches unquoted keywords only, so a quoted "select" stays a name.
func (p *rdsDataParser) isWord(words ...string) bool {
	t := p.peek()
	if t.kind != "word" {
		return false
	}
	for _, w := range words {
		if strings.EqualFold(t.text, w) {
			return true
		}
	}
	return false
}

func (p *rdsDataParser) acceptWords(words ...string) bool {
	start := p.pos
	for _, w := range words {
		if !p.isWord(w) {
			p.pos = start
			return false
		}
		p.pos++
	}
	return true
}

func (p *rdsDataParser) expectWord(word string) error {
	if !p.acceptWords(word) {
		return rdsDataUnsupported("expected %s near %q", strings.ToUpper(word), p.peek().text)
	}
	return nil
}

func (p *rdsDataParser) expectPunct(c string) error {
	if t := p.next(); t.kind != "punct" || t.text != c {
		return rdsDataUnsupported("expected %q near %q", c, t.text)
	}
	return nil
}

func (p *rdsDataParser) acceptPunct(c string) bool {
	if t := p.peek(); t.kind == "punct" && t.text == c {
		p.pos++
		return true
	}
	return false
}

func (p *rdsDataParser) isName() bool {
	t := p.peek()
	return t.kind == "word" || t.kind == "ident"
}

// ident reads a name. Schema-qualified names such as public.items keep only
// the last part.
func (p *rdsDataParser) ident() (string, error) {
	t := p.next()
	if t.kind != "word" && t.kind != "ident" {
		return "", rdsDataUnsupported("expected a name near %q", t.text)
	}
	for p.acceptPunct(".") {
		if t = p.next(); t.kind != "word" && t.kind != "ident" {
			return "", rdsDataUnsupported("expected a name after '.'")
		}
	}
	return t.text, nil
}

func (p *rdsDataParser) end() error {
	p.acceptPunct(";")
	if p.pos < len(p.toks) {
		return rdsDataUnsupported("unsupported SQL near %q", p.peek().text)
	}
	return nil
}

// value reads a literal or :parameter.
func (p *rdsDataParser) value() (any, error) {
	t := p.next()
	switch t.kind {
	case "string":
		return t.text, nil
	case "number":
		if _, err := strconv.ParseFloat(t.text, 64); err != nil {
			return nil, rdsDataDBErr("invalid number %q", t.text)
		}
		return rdsDataNumber(t.text), nil
	case "param":
		v, ok := p.params[t.text]
		if !ok {
			return nil, rdsDataDBErr("no value specified for parameter %q", t.text)
		}
		return v, nil
	case "word":
		switch strings.ToUpper(t.text) {
		case "NULL":
			return nil, nil
		case "TRUE":
			return true, nil
		case "FALSE":
			return false, nil
		}
	}
	return nil, rdsDataUnsupported("unsupported value expression near %q", t.text)
}

func rdsDataKind(typeName string) string {
	switch typeName {
	case "INT", "INTEGER", "BIGINT", "SMALLINT", "TINYINT", "MEDIUMINT", "SERIAL", "BIGSERIAL", "SMALLSERIAL", "INT2", "INT4", "INT8":
		return "long"
	case "FLOAT", "DOUBLE", "REAL", "FLOAT4", "FLOAT8":
		return "double"
	case "DECIMAL", "NUMERIC":
		return "decimal"
	case "BOOL", "BOOLEAN", "BIT":
		return "bool"
	case "BLOB", "BYTEA", "BINARY", "VARBINARY", "LONGBLOB", "MEDIUMBLOB", "TINYBLOB":
		return "blob"
	}
	return "string"
}

// rdsDataFloat parses a finite float; NaN and ±Inf are rejected.
func rdsDataFloat(s string) (float64, bool) {
	f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return f, err == nil && !math.IsNaN(f) && !math.IsInf(f, 0)
}

// rdsDataLong converts an integral float that fits int64.
func rdsDataLong(f float64) (int64, bool) {
	if f != math.Trunc(f) || f < -9.223372036854775808e18 || f >= 9.223372036854775808e18 {
		return 0, false
	}
	return int64(f), true
}

// rdsDataCoerce converts v to the column's storage kind.
func rdsDataCoerce(col rdsDataColumn, v any) (any, error) {
	if v == nil {
		return nil, nil
	}
	bad := rdsDataDBErr("incorrect %s value for column %q", strings.ToLower(col.TypeName), col.Name)
	if n, ok := v.(rdsDataNumber); ok {
		v = string(n)
		if col.Kind == "bool" {
			f, ok := rdsDataFloat(string(n))
			if !ok {
				return nil, bad
			}
			return f != 0, nil
		}
	}
	switch col.Kind {
	case "long":
		switch x := v.(type) {
		case int64:
			return x, nil
		case float64:
			if n, ok := rdsDataLong(x); ok {
				return n, nil
			}
		case bool:
			if x {
				return int64(1), nil
			}
			return int64(0), nil
		case string:
			if n, err := strconv.ParseInt(strings.TrimSpace(x), 10, 64); err == nil {
				return n, nil
			}
			if f, ok := rdsDataFloat(x); ok {
				if n, ok := rdsDataLong(f); ok {
					return n, nil
				}
			}
		}
	case "double":
		switch x := v.(type) {
		case int64:
			return float64(x), nil
		case float64:
			return x, nil
		case string:
			if f, ok := rdsDataFloat(x); ok {
				return f, nil
			}
		}
	case "decimal":
		switch x := v.(type) {
		case int64:
			return strconv.FormatInt(x, 10), nil
		case float64:
			return strconv.FormatFloat(x, 'f', -1, 64), nil
		case string:
			if _, ok := rdsDataFloat(x); ok {
				return strings.TrimSpace(x), nil
			}
		}
	case "bool":
		switch x := v.(type) {
		case bool:
			return x, nil
		case int64:
			return x != 0, nil
		case string:
			if b, err := strconv.ParseBool(strings.TrimSpace(x)); err == nil {
				return b, nil
			}
		}
	case "blob":
		switch x := v.(type) {
		case []byte:
			return append([]byte{}, x...), nil // never nil, so an empty blob isn't NULL
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

// rdsDataCompare orders two non-NULL values of the same column kind.
func rdsDataCompare(kind string, a, b any) int {
	switch kind {
	case "long":
		x, y := a.(int64), b.(int64)
		return cmpOrdered(x, y)
	case "double":
		return cmpOrdered(a.(float64), b.(float64))
	case "decimal":
		x, _ := rdsDataFloat(a.(string))
		y, _ := rdsDataFloat(b.(string))
		return cmpOrdered(x, y)
	case "bool":
		x, y := a.(bool), b.(bool)
		switch {
		case x == y:
			return 0
		case !x:
			return -1
		}
		return 1
	case "blob":
		return bytes.Compare(a.([]byte), b.([]byte))
	}
	return strings.Compare(a.(string), b.(string))
}

func cmpOrdered[T int64 | float64](x, y T) int {
	switch {
	case x < y:
		return -1
	case x > y:
		return 1
	}
	return 0
}

// rdsDataStmt is a tokenized statement plus the table it touches, so the store
// can copy only that table and skip copies for reads.
type rdsDataStmt struct {
	toks   []rdsDataToken
	params map[string]any
	table  string
	write  bool
}

// rdsDataPrepare tokenizes sql and finds the statement's table and whether it
// writes. A SELECT with no FROM touches no table.
func rdsDataPrepare(sql string, params map[string]any) (rdsDataStmt, error) {
	toks, err := rdsDataTokenize(sql)
	if err != nil {
		return rdsDataStmt{}, err
	}
	stmt := rdsDataStmt{toks: toks, params: params}
	p := &rdsDataParser{toks: toks}
	switch {
	case p.acceptWords("CREATE", "TABLE"):
		p.acceptWords("IF", "NOT", "EXISTS")
		stmt.write = true
	case p.acceptWords("DROP", "TABLE"):
		p.acceptWords("IF", "EXISTS")
		stmt.write = true
	case p.acceptWords("INSERT", "INTO"), p.acceptWords("UPDATE"), p.acceptWords("DELETE", "FROM"):
		stmt.write = true
	case p.acceptWords("SELECT"):
		for p.pos < len(toks) && !p.isWord("FROM") {
			p.pos++
		}
		if !p.acceptWords("FROM") {
			return stmt, nil
		}
	default:
		first := "empty statement"
		if len(toks) > 0 {
			first = strings.ToUpper(toks[0].text)
		}
		return rdsDataStmt{}, rdsDataUnsupported("unsupported SQL statement: %s", first)
	}
	name, err := p.ident()
	if err != nil {
		return rdsDataStmt{}, err
	}
	stmt.table = strings.ToLower(name)
	return stmt, nil
}

// exec runs one prepared statement against db. Writes mutate db in place, so
// callers pass copies of the tables a write touches.
func (db rdsDataDB) exec(stmt rdsDataStmt) (rdsDataResult, error) {
	p := &rdsDataParser{toks: stmt.toks, params: stmt.params}
	switch {
	case p.acceptWords("CREATE", "TABLE"):
		return db.createTable(p)
	case p.acceptWords("DROP", "TABLE"):
		return db.dropTable(p)
	case p.acceptWords("INSERT", "INTO"):
		return db.insert(p)
	case p.acceptWords("SELECT"):
		return db.selectRows(p)
	case p.acceptWords("UPDATE"):
		return db.update(p)
	case p.acceptWords("DELETE", "FROM"):
		return db.deleteRows(p)
	}
	return rdsDataResult{}, rdsDataUnsupported("unsupported SQL statement")
}

func (db rdsDataDB) table(name string) (*rdsDataTable, error) {
	table, ok := db[strings.ToLower(name)]
	if !ok {
		return nil, rdsDataDBErr("table %q does not exist", name)
	}
	return table, nil
}

func (t *rdsDataTable) column(name string) (int, error) {
	for i, col := range t.Columns {
		if strings.EqualFold(col.Name, name) {
			return i, nil
		}
	}
	return -1, rdsDataDBErr("column %q does not exist in table %q", name, t.Name)
}
