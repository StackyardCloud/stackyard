package server

import (
	"fmt"
	"strconv"
	"strings"
)

// A deliberately small, deterministic SQL subset for the Redshift Data API
// emulator. It parses one statement at a time into a rsdStmt; execution lives
// in redshiftdata_sql_exec.go.
//
// ponytail: supported statements are CREATE TABLE/SCHEMA/DATABASE, DROP TABLE,
// INSERT ... VALUES, SELECT (columns, *, COUNT(*), literals) with WHERE
// `col = value` / `col IS [NOT] NULL` joined by AND, ORDER BY one column and
// LIMIT, UPDATE ... SET and DELETE. No joins, OR, GROUP BY, functions other
// than COUNT(*), or subqueries; anything else FAILS with an "unsupported"
// error. Swap in a real engine if clients need more.

const rsdMaxSQLLength = 100000 // ExecuteStatement accepts up to 100 KB of SQL.

type rsdTokKind int

const (
	rsdTokWord rsdTokKind = iota
	rsdTokQuoted
	rsdTokString
	rsdTokNumber
	rsdTokParam
	rsdTokPunct
)

type rsdToken struct {
	kind rsdTokKind
	text string
}

func rsdTokenize(sql string) ([]rsdToken, error) {
	var out []rsdToken
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
				return nil, fmt.Errorf("unterminated /* comment")
			}
			i += end + 4
		case c == '\'':
			var b strings.Builder
			j := i + 1
			for {
				if j >= len(sql) {
					return nil, fmt.Errorf("unterminated quoted string at or near \"%s\"", rsdSnippet(sql[i:]))
				}
				if sql[j] == '\\' && j+1 < len(sql) {
					esc := sql[j+1]
					if mapped, ok := rsdEscapes[esc]; ok {
						esc = mapped
					}
					b.WriteByte(esc)
					j += 2
					continue
				}
				if sql[j] == '\'' {
					if j+1 < len(sql) && sql[j+1] == '\'' {
						b.WriteByte('\'')
						j += 2
						continue
					}
					break
				}
				b.WriteByte(sql[j])
				j++
			}
			out = append(out, rsdToken{rsdTokString, b.String()})
			i = j + 1
		case c == '"':
			end := strings.IndexByte(sql[i+1:], '"')
			if end < 0 {
				return nil, fmt.Errorf("unterminated quoted identifier at or near \"%s\"", rsdSnippet(sql[i:]))
			}
			out = append(out, rsdToken{rsdTokQuoted, strings.ToLower(sql[i+1 : i+1+end])})
			i += end + 2
		case c == ':' && i+1 < len(sql) && rsdIdentChar(sql[i+1]):
			j := i + 1
			for j < len(sql) && rsdIdentChar(sql[j]) {
				j++
			}
			out = append(out, rsdToken{rsdTokParam, sql[i+1 : j]})
			i = j
		case c >= '0' && c <= '9' || c == '.' && i+1 < len(sql) && sql[i+1] >= '0' && sql[i+1] <= '9':
			j := i
			for j < len(sql) && (sql[j] >= '0' && sql[j] <= '9' || sql[j] == '.' || sql[j] == 'e' || sql[j] == 'E' ||
				(sql[j] == '-' || sql[j] == '+') && j > i && (sql[j-1] == 'e' || sql[j-1] == 'E')) {
				j++
			}
			out = append(out, rsdToken{rsdTokNumber, sql[i:j]})
			i = j
		case rsdIdentChar(c):
			j := i
			for j < len(sql) && (rsdIdentChar(sql[j]) || sql[j] == '$') {
				j++
			}
			out = append(out, rsdToken{rsdTokWord, strings.ToLower(sql[i:j])})
			i = j
		case strings.IndexByte("(),;*=.-<>!", c) >= 0:
			out = append(out, rsdToken{rsdTokPunct, string(c)})
			i++
		default:
			return nil, fmt.Errorf("syntax error at or near \"%c\"", c)
		}
	}
	return out, nil
}

// rsdEscapes maps backslash escapes in string literals; any other escaped
// character (\\, \', ...) stands for itself.
var rsdEscapes = map[byte]byte{'n': '\n', 't': '\t', 'r': '\r', 'b': '\b', 'f': '\f', '0': 0}

func rsdIdentChar(c byte) bool {
	return c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
}

func rsdSnippet(s string) string {
	if len(s) > 20 {
		return s[:20]
	}
	return s
}

// rsdSplitStatements splits on top-level semicolons, dropping empty pieces.
func rsdSplitStatements(tokens []rsdToken) [][]rsdToken {
	var out [][]rsdToken
	start := 0
	for i, t := range tokens {
		if t.kind == rsdTokPunct && t.text == ";" {
			if i > start {
				out = append(out, tokens[start:i])
			}
			start = i + 1
		}
	}
	if start < len(tokens) {
		out = append(out, tokens[start:])
	}
	return out
}

type rsdExpr struct {
	kind rsdTokKind // rsdTokString, rsdTokNumber, rsdTokParam, or rsdTokWord for NULL/TRUE/FALSE
	text string
}

type rsdCond struct {
	col  string
	op   string // "=", "isnull", "notnull"
	expr rsdExpr
}

type rsdSelectItem struct {
	star  bool
	count bool
	col   string
	lit   *rsdExpr
	alias string
}

type rsdColumnDef struct {
	name       string
	typ        rsdType
	notNull    bool
	hasDefault bool
	def        rsdExpr
	identity   bool
	seed, step int64
}

type rsdStmt struct {
	verb        string // create_table, create_schema, create_database, drop_table, insert, select, update, delete
	schema      string
	table       string
	ifExists    bool
	ifNotExists bool
	columns     []rsdColumnDef
	insertCols  []string
	rows        [][]rsdExpr
	items       []rsdSelectItem
	hasFrom     bool
	where       []rsdCond
	orderBy     string
	desc        bool
	limit       int
	sets        []rsdCond
}

type rsdParser struct {
	toks []rsdToken
	pos  int
}

func (p *rsdParser) peek() rsdToken {
	if p.pos < len(p.toks) {
		return p.toks[p.pos]
	}
	return rsdToken{kind: rsdTokPunct, text: ""}
}

func (p *rsdParser) next() rsdToken { t := p.peek(); p.pos++; return t }

func (p *rsdParser) done() bool { return p.pos >= len(p.toks) }

func (p *rsdParser) isWord(w string) bool {
	t := p.peek()
	return t.kind == rsdTokWord && t.text == w
}

func (p *rsdParser) acceptWords(ws ...string) bool {
	save := p.pos
	for _, w := range ws {
		if !p.isWord(w) {
			p.pos = save
			return false
		}
		p.pos++
	}
	return true
}

func (p *rsdParser) isPunct(c string) bool {
	t := p.peek()
	return t.kind == rsdTokPunct && t.text == c
}

func (p *rsdParser) expectPunct(c string) error {
	if !p.isPunct(c) {
		return p.syntaxErr()
	}
	p.pos++
	return nil
}

func (p *rsdParser) syntaxErr() error {
	if p.done() {
		return fmt.Errorf("syntax error at end of input")
	}
	return fmt.Errorf("syntax error at or near \"%s\"", p.peek().text)
}

var rsdReserved = map[string]bool{
	"select": true, "from": true, "where": true, "and": true, "or": true, "order": true, "by": true, "limit": true,
	"insert": true, "into": true, "values": true, "update": true, "set": true, "delete": true, "create": true,
	"table": true, "drop": true, "null": true, "not": true, "is": true, "as": true, "true": true, "false": true,
}

func (p *rsdParser) ident() (string, error) {
	t := p.peek()
	if t.kind == rsdTokQuoted || t.kind == rsdTokWord && !rsdReserved[t.text] {
		p.pos++
		return t.text, nil
	}
	return "", p.syntaxErr()
}

// tableName reads [schema.]table; the schema defaults to public.
func (p *rsdParser) tableName() (string, string, error) {
	first, err := p.ident()
	if err != nil {
		return "", "", err
	}
	if p.isPunct(".") {
		p.pos++
		second, err := p.ident()
		if err != nil {
			return "", "", err
		}
		return first, second, nil
	}
	return "public", first, nil
}

func (p *rsdParser) expr() (rsdExpr, error) {
	t := p.peek()
	switch {
	case t.kind == rsdTokString || t.kind == rsdTokNumber || t.kind == rsdTokParam:
		p.pos++
		return rsdExpr{t.kind, t.text}, nil
	case t.kind == rsdTokPunct && t.text == "-":
		p.pos++
		n := p.next()
		if n.kind != rsdTokNumber {
			return rsdExpr{}, fmt.Errorf("unsupported expression: only literals and :parameters are supported")
		}
		return rsdExpr{rsdTokNumber, "-" + n.text}, nil
	case t.kind == rsdTokWord && (t.text == "null" || t.text == "true" || t.text == "false"):
		p.pos++
		return rsdExpr{rsdTokWord, t.text}, nil
	}
	return rsdExpr{}, fmt.Errorf("unsupported expression at or near \"%s\": only literals and :parameters are supported", t.text)
}

func rsdParse(tokens []rsdToken) (*rsdStmt, error) {
	p := &rsdParser{toks: tokens}
	var st *rsdStmt
	var err error
	switch {
	case p.acceptWords("create", "table"):
		st, err = p.createTable()
	case p.acceptWords("create", "schema"):
		st = &rsdStmt{verb: "create_schema"}
		st.ifNotExists = p.acceptWords("if", "not", "exists")
		st.schema, err = p.ident()
	case p.acceptWords("create", "database"):
		st = &rsdStmt{verb: "create_database"}
		st.table, err = p.ident()
	case p.acceptWords("drop", "table"):
		st = &rsdStmt{verb: "drop_table"}
		st.ifExists = p.acceptWords("if", "exists")
		st.schema, st.table, err = p.tableName()
	case p.acceptWords("insert", "into"):
		st, err = p.insert()
	case p.acceptWords("select"):
		st, err = p.selectStmt()
	case p.acceptWords("update"):
		st, err = p.update()
	case p.acceptWords("delete", "from"):
		st = &rsdStmt{verb: "delete"}
		if st.schema, st.table, err = p.tableName(); err == nil {
			st.where, err = p.whereClause()
		}
	default:
		return nil, fmt.Errorf("unsupported statement %q: the emulator supports CREATE TABLE/SCHEMA/DATABASE, DROP TABLE, INSERT, SELECT, UPDATE and DELETE", rsdLead(tokens))
	}
	if err != nil {
		return nil, err
	}
	if !p.done() {
		return nil, p.syntaxErr()
	}
	return st, nil
}

func rsdLead(tokens []rsdToken) string {
	var parts []string
	for _, t := range tokens {
		if t.kind != rsdTokWord || len(parts) == 2 {
			break
		}
		parts = append(parts, strings.ToUpper(t.text))
	}
	if len(parts) == 0 && len(tokens) > 0 {
		return tokens[0].text
	}
	return strings.Join(parts, " ")
}

var rsdColumnStop = map[string]bool{
	"not": true, "null": true, "default": true, "primary": true, "unique": true, "identity": true,
	"encode": true, "distkey": true, "sortkey": true, "references": true, "collate": true, "generated": true,
}

func (p *rsdParser) createTable() (*rsdStmt, error) {
	st := &rsdStmt{verb: "create_table"}
	st.ifNotExists = p.acceptWords("if", "not", "exists")
	var err error
	if st.schema, st.table, err = p.tableName(); err != nil {
		return nil, err
	}
	if err := p.expectPunct("("); err != nil {
		return nil, err
	}
	for {
		if p.isWord("primary") || p.isWord("unique") || p.isWord("foreign") || p.isWord("constraint") {
			p.skipToColumnEnd() // ponytail: table constraints are informational in Redshift too
		} else {
			col, err := p.columnDef()
			if err != nil {
				return nil, err
			}
			st.columns = append(st.columns, col)
		}
		if p.isPunct(",") {
			p.pos++
			continue
		}
		if err := p.expectPunct(")"); err != nil {
			return nil, err
		}
		break
	}
	for !p.done() { // table attributes (DISTSTYLE, DISTKEY(c), SORTKEY(...), ENCODE AUTO, ...) are ignored
		t := p.next()
		if t.kind != rsdTokWord && t.kind != rsdTokQuoted && !(t.kind == rsdTokPunct && strings.Contains("(),", t.text)) {
			return nil, fmt.Errorf("syntax error at or near \"%s\"", t.text)
		}
	}
	if len(st.columns) == 0 {
		return nil, fmt.Errorf("tables must have at least one column")
	}
	return st, nil
}

func (p *rsdParser) skipToColumnEnd() {
	depth := 0
	for !p.done() {
		t := p.peek()
		if t.kind == rsdTokPunct {
			switch {
			case t.text == "(":
				depth++
			case t.text == ")" && depth == 0, t.text == "," && depth == 0:
				return
			case t.text == ")":
				depth--
			}
		}
		p.pos++
	}
}

func (p *rsdParser) columnDef() (rsdColumnDef, error) {
	col := rsdColumnDef{}
	var err error
	if col.name, err = p.ident(); err != nil {
		return col, err
	}
	var words []string
	for p.peek().kind == rsdTokWord && !rsdColumnStop[p.peek().text] {
		words = append(words, p.next().text)
	}
	var args []int
	if p.isPunct("(") {
		p.pos++
		for !p.isPunct(")") {
			t := p.next()
			if t.kind == rsdTokNumber {
				n, err := strconv.Atoi(t.text)
				if err != nil {
					return col, fmt.Errorf("invalid type modifier \"%s\"", t.text)
				}
				args = append(args, n)
			} else if p.done() {
				return col, p.syntaxErr()
			} else if t.kind == rsdTokWord && t.text == "max" {
				args = append(args, 65535)
			} else if !(t.kind == rsdTokPunct && t.text == ",") {
				return col, fmt.Errorf("syntax error at or near \"%s\"", t.text)
			}
		}
		p.pos++
		for p.peek().kind == rsdTokWord && !rsdColumnStop[p.peek().text] { // e.g. "timestamp(6) with time zone"
			words = append(words, p.next().text)
		}
	}
	if col.typ, err = rsdResolveType(strings.Join(words, " "), args); err != nil {
		return col, err
	}
	for !p.done() && !p.isPunct(",") && !p.isPunct(")") {
		switch {
		case p.acceptWords("not", "null"):
			col.notNull = true
		case p.acceptWords("null"):
		case p.acceptWords("default"):
			if col.def, err = p.expr(); err != nil {
				return col, err
			}
			col.hasDefault = true
		case p.acceptWords("primary", "key"):
			col.notNull = true // ponytail: PK/UNIQUE are informational in Redshift, only NOT NULL is implied
		case p.acceptWords("unique"):
		case p.acceptWords("identity"), p.acceptWords("generated", "by", "default", "as", "identity"):
			col.identity, col.seed, col.step = true, 1, 1
			if p.isPunct("(") {
				p.pos++
				a, b := p.next(), rsdToken{}
				if p.isPunct(",") {
					p.pos++
					b = p.next()
				}
				if err := p.expectPunct(")"); err != nil {
					return col, err
				}
				var err1, err2 error
				col.seed, err1 = strconv.ParseInt(a.text, 10, 64)
				if b.text != "" {
					col.step, err2 = strconv.ParseInt(b.text, 10, 64)
				}
				if err1 != nil || err2 != nil {
					return col, fmt.Errorf("IDENTITY seed and step must be integers")
				}
				if col.step == 0 {
					return col, fmt.Errorf("IDENTITY step can't be 0")
				}
			}
			if col.typ.kind != rsdKindLong {
				return col, fmt.Errorf("IDENTITY columns must be SMALLINT, INTEGER or BIGINT")
			}
		default:
			p.pos++ // ENCODE x, DISTKEY, SORTKEY, REFERENCES t(c), COLLATE x: accepted and ignored
			if p.isPunct("(") {
				p.skipParens()
			}
		}
	}
	return col, nil
}

func (p *rsdParser) skipParens() {
	depth := 0
	for !p.done() {
		t := p.next()
		if t.kind == rsdTokPunct && t.text == "(" {
			depth++
		} else if t.kind == rsdTokPunct && t.text == ")" {
			depth--
			if depth == 0 {
				return
			}
		}
	}
}

func (p *rsdParser) insert() (*rsdStmt, error) {
	st := &rsdStmt{verb: "insert"}
	var err error
	if st.schema, st.table, err = p.tableName(); err != nil {
		return nil, err
	}
	if p.isPunct("(") {
		p.pos++
		for {
			c, err := p.ident()
			if err != nil {
				return nil, err
			}
			st.insertCols = append(st.insertCols, c)
			if p.isPunct(",") {
				p.pos++
				continue
			}
			if err := p.expectPunct(")"); err != nil {
				return nil, err
			}
			break
		}
	}
	if !p.acceptWords("values") {
		return nil, fmt.Errorf("unsupported INSERT form: only INSERT ... VALUES is supported")
	}
	for {
		if err := p.expectPunct("("); err != nil {
			return nil, err
		}
		var row []rsdExpr
		for {
			e, err := p.expr()
			if err != nil {
				return nil, err
			}
			row = append(row, e)
			if p.isPunct(",") {
				p.pos++
				continue
			}
			if err := p.expectPunct(")"); err != nil {
				return nil, err
			}
			break
		}
		st.rows = append(st.rows, row)
		if !p.isPunct(",") {
			return st, nil
		}
		p.pos++
	}
}

func (p *rsdParser) selectStmt() (*rsdStmt, error) {
	st := &rsdStmt{verb: "select", limit: -1}
	for {
		var item rsdSelectItem
		switch t := p.peek(); {
		case t.kind == rsdTokPunct && t.text == "*":
			p.pos++
			item.star = true
		case p.isWord("count") && p.pos+1 < len(p.toks) && p.toks[p.pos+1].text == "(" && p.toks[p.pos+1].kind == rsdTokPunct:
			p.pos++
			if p.expectPunct("(") != nil || p.expectPunct("*") != nil || p.expectPunct(")") != nil {
				return nil, fmt.Errorf("unsupported expression: only COUNT(*) is supported")
			}
			item.count = true
		case t.kind == rsdTokWord && !rsdReserved[t.text] || t.kind == rsdTokQuoted:
			p.pos++
			if p.isPunct("(") {
				return nil, fmt.Errorf("unsupported function %s(): only COUNT(*) is supported", strings.ToUpper(t.text))
			}
			item.col = t.text
		default:
			e, err := p.expr()
			if err != nil {
				return nil, err
			}
			item.lit = &e
		}
		if p.acceptWords("as") {
			alias, err := p.ident()
			if err != nil {
				return nil, err
			}
			item.alias = alias
		}
		st.items = append(st.items, item)
		if !p.isPunct(",") {
			break
		}
		p.pos++
	}
	if p.acceptWords("from") {
		st.hasFrom = true
		var err error
		if st.schema, st.table, err = p.tableName(); err != nil {
			return nil, err
		}
		if st.where, err = p.whereClause(); err != nil {
			return nil, err
		}
		if p.acceptWords("order", "by") {
			if st.orderBy, err = p.ident(); err != nil {
				return nil, err
			}
			if p.acceptWords("desc") {
				st.desc = true
			} else {
				p.acceptWords("asc")
			}
		}
	}
	if p.acceptWords("limit") {
		t := p.next()
		n, err := strconv.Atoi(t.text)
		if t.kind != rsdTokNumber || err != nil || n < 0 {
			return nil, fmt.Errorf("LIMIT must be a non-negative integer")
		}
		st.limit = n
	}
	return st, nil
}

func (p *rsdParser) update() (*rsdStmt, error) {
	st := &rsdStmt{verb: "update"}
	var err error
	if st.schema, st.table, err = p.tableName(); err != nil {
		return nil, err
	}
	if !p.acceptWords("set") {
		return nil, p.syntaxErr()
	}
	for {
		col, err := p.ident()
		if err != nil {
			return nil, err
		}
		if err := p.expectPunct("="); err != nil {
			return nil, err
		}
		e, err := p.expr()
		if err != nil {
			return nil, err
		}
		st.sets = append(st.sets, rsdCond{col: col, op: "=", expr: e})
		if !p.isPunct(",") {
			break
		}
		p.pos++
	}
	st.where, err = p.whereClause()
	return st, err
}

func (p *rsdParser) whereClause() ([]rsdCond, error) {
	if !p.acceptWords("where") {
		return nil, nil
	}
	var conds []rsdCond
	for {
		col, err := p.ident()
		if err != nil {
			return nil, err
		}
		switch {
		case p.isPunct("="):
			p.pos++
			e, err := p.expr()
			if err != nil {
				return nil, err
			}
			conds = append(conds, rsdCond{col: col, op: "=", expr: e})
		case p.acceptWords("is", "not", "null"):
			conds = append(conds, rsdCond{col: col, op: "notnull"})
		case p.acceptWords("is", "null"):
			conds = append(conds, rsdCond{col: col, op: "isnull"})
		default:
			return nil, fmt.Errorf("unsupported WHERE condition at or near \"%s\": only `col = value` and `col IS [NOT] NULL` joined by AND are supported", p.peek().text)
		}
		if !p.acceptWords("and") {
			return conds, nil
		}
	}
}
