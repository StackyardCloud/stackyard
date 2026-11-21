package server

import (
	"fmt"
	"strconv"
	"strings"
)

// ponytail: the RDS Data API emulation runs a tiny SQL subset, not a database
// engine: CREATE TABLE, DROP TABLE, INSERT ... VALUES, SELECT * | cols FROM t,
// UPDATE ... SET and DELETE FROM, with WHERE limited to `col = value` terms
// joined by AND. Joins, ORDER BY, functions and expressions are rejected as
// unsupported. Swap in an embedded SQL engine if clients need more.

// rdsDataColumn is one column of an emulated table. Kind is the storage kind
// derived from the declared type: long, double, bool, blob or string.
type rdsDataColumn struct {
	Name     string
	TypeName string
	Kind     string
	AutoInc  bool
	Table    string
}

type rdsDataTable struct {
	Name    string
	Columns []rdsDataColumn
	Rows    [][]any
	NextID  int64
}

// rdsDataDB holds one database's tables, keyed by lower-case table name.
// Values are nil, int64, float64, bool, string or []byte.
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

func (db rdsDataDB) clone() rdsDataDB {
	out := make(rdsDataDB, len(db))
	for key, table := range db {
		copied := *table
		copied.Columns = append([]rdsDataColumn(nil), table.Columns...)
		copied.Rows = make([][]any, len(table.Rows))
		for i, row := range table.Rows {
			copied.Rows[i] = append([]any(nil), row...)
		}
		out[key] = &copied
	}
	return out
}

type rdsDataToken struct {
	kind string // word, string, number, param, punct
	text string
}

func rdsDataTokenize(sql string) ([]rdsDataToken, error) {
	var out []rdsDataToken
	for i := 0; i < len(sql); {
		c := sql[i]
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			i++
		case c == '\'':
			var b strings.Builder
			j := i + 1
			for ; j < len(sql); j++ {
				if sql[j] == '\'' {
					if j+1 < len(sql) && sql[j+1] == '\'' {
						b.WriteByte('\'')
						j++
						continue
					}
					break
				}
				b.WriteByte(sql[j])
			}
			if j >= len(sql) {
				return nil, rdsDataDBErr("unterminated string literal")
			}
			out = append(out, rdsDataToken{"string", b.String()})
			i = j + 1
		case c == '`' || c == '"':
			end := strings.IndexByte(sql[i+1:], c)
			if end < 0 {
				return nil, rdsDataDBErr("unterminated quoted identifier")
			}
			out = append(out, rdsDataToken{"word", sql[i+1 : i+1+end]})
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

func (p *rdsDataParser) ident() (string, error) {
	t := p.next()
	if t.kind != "word" {
		return "", rdsDataUnsupported("expected a name near %q", t.text)
	}
	// Accept schema-qualified names such as public.items by keeping the last part.
	for p.acceptPunct(".") {
		if t = p.next(); t.kind != "word" {
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
		if strings.Contains(t.text, ".") {
			f, err := strconv.ParseFloat(t.text, 64)
			if err != nil {
				return nil, rdsDataDBErr("invalid number %q", t.text)
			}
			return f, nil
		}
		n, err := strconv.ParseInt(t.text, 10, 64)
		if err != nil {
			return nil, rdsDataDBErr("invalid number %q", t.text)
		}
		return n, nil
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

// exec runs one statement against db, mutating it in place. Callers pass a
// clone when the statement must be atomic.
func (db rdsDataDB) exec(sql string, params map[string]any) (rdsDataResult, error) {
	toks, err := rdsDataTokenize(sql)
	if err != nil {
		return rdsDataResult{}, err
	}
	p := &rdsDataParser{toks: toks, params: params}
	switch {
	case p.acceptWords("CREATE", "TABLE"):
		return db.createTable(p)
	case p.acceptWords("DROP", "TABLE"):
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
	case p.acceptWords("INSERT", "INTO"):
		return db.insert(p)
	case p.acceptWords("SELECT"):
		return db.selectRows(p)
	case p.acceptWords("UPDATE"):
		return db.update(p)
	case p.acceptWords("DELETE", "FROM"):
		return db.deleteRows(p)
	}
	first := "empty statement"
	if len(toks) > 0 {
		first = strings.ToUpper(toks[0].text)
	}
	return rdsDataResult{}, rdsDataUnsupported("unsupported SQL statement: %s", first)
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
	return -1, rdsDataDBErr("unknown column %q in table %q", name, t.Name)
}
