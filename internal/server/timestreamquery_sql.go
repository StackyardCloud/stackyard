package server

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// A practical subset of Timestream SQL over the stored records:
//   SELECT items FROM "db"."table" [WHERE cond] [GROUP BY exprs]
//          [ORDER BY expr [ASC|DESC], ...] [LIMIT n]
// with columns (time, measure_name, dimensions, multi-measure attributes,
// measure_value::<type>), literals, durations (15m, 1h, 2d), now(), ago(),
// bin(), from_milliseconds(), + and - on timestamps, AND/OR/NOT, comparisons,
// BETWEEN, IN, IS [NOT] NULL, LIKE, and count/avg/sum/min/max.
// ponytail: no joins, subqueries, window functions or CASE; they get a
// ValidationException naming what isn't supported.

const tsqTimeLayout = "2006-01-02 15:04:05.000000000"

// --- values ---

type tsqVal struct {
	typ string // VARCHAR DOUBLE BIGINT BOOLEAN TIMESTAMP INTERVAL_DAY_TO_SECOND, "" = NULL
	s   string
	f   float64
	i   int64 // BIGINT, TIMESTAMP (ns) and INTERVAL (ns)
	b   bool
}

func (v tsqVal) null() bool { return v.typ == "" }

func (v tsqVal) num() (float64, bool) {
	switch v.typ {
	case "DOUBLE":
		return v.f, true
	case "BIGINT", "TIMESTAMP", "INTERVAL_DAY_TO_SECOND":
		return float64(v.i), true
	}
	return 0, false
}

func (v tsqVal) format() string {
	switch v.typ {
	case "VARCHAR":
		return v.s
	case "DOUBLE":
		s := strconv.FormatFloat(v.f, 'f', -1, 64)
		if !strings.ContainsAny(s, ".eEIN") {
			s += ".0"
		}
		return s
	case "BIGINT":
		return strconv.FormatInt(v.i, 10)
	case "BOOLEAN":
		return strconv.FormatBool(v.b)
	case "TIMESTAMP":
		return time.Unix(0, v.i).UTC().Format(tsqTimeLayout)
	case "INTERVAL_DAY_TO_SECOND":
		d := time.Duration(v.i)
		days := int64(d / (24 * time.Hour))
		d -= time.Duration(days) * 24 * time.Hour
		return fmt.Sprintf("%d %02d:%02d:%02d.%09d", days, int(d.Hours()), int(d.Minutes())%60, int(d.Seconds())%60, d.Nanoseconds()%1e9)
	}
	return ""
}

func tsqScalar(typ, raw string) tsqVal {
	switch typ {
	case "DOUBLE":
		f, _ := strconv.ParseFloat(raw, 64)
		return tsqVal{typ: "DOUBLE", f: f}
	case "BIGINT":
		n, _ := strconv.ParseInt(raw, 10, 64)
		return tsqVal{typ: "BIGINT", i: n}
	case "TIMESTAMP":
		n, _ := strconv.ParseInt(raw, 10, 64) // stored as epoch milliseconds
		return tsqVal{typ: "TIMESTAMP", i: n * 1_000_000}
	case "BOOLEAN":
		return tsqVal{typ: "BOOLEAN", b: raw == "true"}
	}
	return tsqVal{typ: "VARCHAR", s: raw}
}

func tsqParseTime(s string) (int64, bool) {
	for _, layout := range []string{"2006-01-02 15:04:05.999999999", "2006-01-02 15:04:05", "2006-01-02T15:04:05.999999999Z07:00", "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UnixNano(), true
		}
	}
	return 0, false
}

// tsqCompare orders two non-null values; ok is false when they can't be compared.
func tsqCompare(a, b tsqVal) (int, bool) {
	if a.typ == "TIMESTAMP" && b.typ == "VARCHAR" {
		n, ok := tsqParseTime(b.s)
		if !ok {
			return 0, false
		}
		b = tsqVal{typ: "TIMESTAMP", i: n}
	}
	if b.typ == "TIMESTAMP" && a.typ == "VARCHAR" {
		c, ok := tsqCompare(b, a)
		return -c, ok
	}
	if a.typ == "VARCHAR" || b.typ == "VARCHAR" {
		if a.typ != b.typ {
			return 0, false
		}
		return strings.Compare(a.s, b.s), true
	}
	if a.typ == "BOOLEAN" || b.typ == "BOOLEAN" {
		if a.typ != b.typ {
			return 0, false
		}
		switch {
		case a.b == b.b:
			return 0, true
		case !a.b:
			return -1, true
		}
		return 1, true
	}
	if (a.typ == "TIMESTAMP") != (b.typ == "TIMESTAMP") {
		return 0, false
	}
	if a.typ != "DOUBLE" && b.typ != "DOUBLE" {
		switch {
		case a.i < b.i:
			return -1, true
		case a.i > b.i:
			return 1, true
		}
		return 0, true
	}
	x, _ := a.num()
	y, _ := b.num()
	switch {
	case x < y:
		return -1, true
	case x > y:
		return 1, true
	}
	return 0, true
}

// --- lexer ---

type tsqTok struct {
	kind string // ident qident str num dur op param eof
	text string
}

func tsqLex(src string) ([]tsqTok, error) {
	var out []tsqTok
	r := []rune(src)
	for i := 0; i < len(r); {
		c := r[i]
		switch {
		case unicode.IsSpace(c):
			i++
		case c == '\'':
			var b strings.Builder
			i++
			for {
				if i >= len(r) {
					return nil, fmt.Errorf("unterminated string literal")
				}
				if r[i] == '\'' {
					if i+1 < len(r) && r[i+1] == '\'' {
						b.WriteRune('\'')
						i += 2
						continue
					}
					i++
					break
				}
				b.WriteRune(r[i])
				i++
			}
			out = append(out, tsqTok{"str", b.String()})
		case c == '"':
			j := i + 1
			for j < len(r) && r[j] != '"' {
				j++
			}
			if j >= len(r) {
				return nil, fmt.Errorf("unterminated quoted identifier")
			}
			out = append(out, tsqTok{"qident", string(r[i+1 : j])})
			i = j + 1
		case unicode.IsDigit(c) || (c == '.' && i+1 < len(r) && unicode.IsDigit(r[i+1])):
			j := i
			for j < len(r) && (unicode.IsDigit(r[j]) || r[j] == '.' || r[j] == 'e' || r[j] == 'E') {
				j++
			}
			k := j
			for k < len(r) && unicode.IsLetter(r[k]) {
				k++
			}
			if k > j {
				out = append(out, tsqTok{"dur", string(r[i:k])})
				i = k
				continue
			}
			out = append(out, tsqTok{"num", string(r[i:j])})
			i = j
		case c == '@':
			j := i + 1
			for j < len(r) && (unicode.IsLetter(r[j]) || unicode.IsDigit(r[j]) || r[j] == '_') {
				j++
			}
			out = append(out, tsqTok{"param", string(r[i:j])})
			i = j
		case unicode.IsLetter(c) || c == '_':
			j := i
			for j < len(r) && (unicode.IsLetter(r[j]) || unicode.IsDigit(r[j]) || r[j] == '_') {
				j++
			}
			out = append(out, tsqTok{"ident", string(r[i:j])})
			i = j
		default:
			two := ""
			if i+1 < len(r) {
				two = string(r[i : i+2])
			}
			switch two {
			case "::", "!=", "<>", "<=", ">=", "||":
				out = append(out, tsqTok{"op", two})
				i += 2
				continue
			}
			if !strings.ContainsRune("(),.*=<>+-/;%", c) {
				return nil, fmt.Errorf("unexpected character %q", c)
			}
			out = append(out, tsqTok{"op", string(c)})
			i++
		}
	}
	return append(out, tsqTok{kind: "eof"}), nil
}

// --- parser ---

type tsqNode struct {
	op   string // col lit param call agg cmp and or not between in isnull like neg arith star
	name string
	args []*tsqNode
	val  tsqVal
	text string
}

type tsqItem struct {
	expr  *tsqNode
	alias string
}

type tsqOrder struct {
	expr *tsqNode
	desc bool
}

type tsqQuery struct {
	star    bool
	items   []tsqItem
	db, tbl string
	where   *tsqNode
	groupBy []*tsqNode
	orderBy []tsqOrder
	limit   int
	params  []string
}

type tsqParser struct {
	toks   []tsqTok
	pos    int
	params map[string]bool
	depth  int // expression nesting, bounded so a hostile query can't exhaust the stack
}

const tsqMaxDepth = 64

func (p *tsqParser) peek() tsqTok { return p.toks[p.pos] }

func (p *tsqParser) next() tsqTok {
	t := p.toks[p.pos]
	if t.kind != "eof" {
		p.pos++
	}
	return t
}

func (p *tsqParser) kw(word string) bool {
	t := p.peek()
	if t.kind == "ident" && strings.EqualFold(t.text, word) {
		p.pos++
		return true
	}
	return false
}

func (p *tsqParser) isOp(op string) bool {
	if t := p.peek(); t.kind == "op" && t.text == op {
		p.pos++
		return true
	}
	return false
}

func (p *tsqParser) expectOp(op string) error {
	if !p.isOp(op) {
		return fmt.Errorf("expected %q near %q", op, p.peek().text)
	}
	return nil
}

var tsqReserved = map[string]bool{"from": true, "where": true, "group": true, "order": true, "limit": true, "by": true,
	"and": true, "or": true, "not": true, "as": true, "asc": true, "desc": true, "between": true, "in": true, "is": true, "like": true, "null": true}

func tsqParse(src string) (*tsqQuery, error) {
	toks, err := tsqLex(src)
	if err != nil {
		return nil, err
	}
	p := &tsqParser{toks: toks, params: map[string]bool{}}
	q := &tsqQuery{limit: -1}
	if p.kw("with") {
		return nil, fmt.Errorf("WITH clauses are not supported")
	}
	if !p.kw("select") {
		return nil, fmt.Errorf("only SELECT queries are supported")
	}
	if p.kw("distinct") {
		return nil, fmt.Errorf("SELECT DISTINCT is not supported")
	}
	for {
		if p.isOp("*") {
			q.star = true
		} else {
			e, err := p.expr()
			if err != nil {
				return nil, err
			}
			item := tsqItem{expr: e}
			if p.kw("as") {
				t := p.next()
				if t.kind != "ident" && t.kind != "qident" {
					return nil, fmt.Errorf("expected an alias after AS")
				}
				item.alias = t.text
			} else if t := p.peek(); (t.kind == "ident" && !tsqReserved[strings.ToLower(t.text)]) || t.kind == "qident" {
				item.alias = p.next().text
			}
			q.items = append(q.items, item)
		}
		if !p.isOp(",") {
			break
		}
	}
	if !p.kw("from") {
		return nil, fmt.Errorf("expected FROM")
	}
	if p.isOp("(") {
		return nil, fmt.Errorf("subqueries are not supported")
	}
	db := p.next()
	if (db.kind != "ident" && db.kind != "qident") || !p.isOp(".") {
		return nil, fmt.Errorf(`FROM must name a table as "database"."table"`)
	}
	tbl := p.next()
	if tbl.kind != "ident" && tbl.kind != "qident" {
		return nil, fmt.Errorf(`FROM must name a table as "database"."table"`)
	}
	q.db, q.tbl = db.text, tbl.text
	if t := p.peek(); t.kind == "ident" && (strings.EqualFold(t.text, "join") || strings.EqualFold(t.text, "inner") || strings.EqualFold(t.text, "left") || strings.EqualFold(t.text, "cross")) || (t.kind == "op" && t.text == ",") {
		return nil, fmt.Errorf("joins are not supported")
	}
	if p.kw("where") {
		if q.where, err = p.boolExpr(); err != nil {
			return nil, err
		}
	}
	if p.kw("group") {
		if !p.kw("by") {
			return nil, fmt.Errorf("expected BY after GROUP")
		}
		for {
			e, err := p.expr()
			if err != nil {
				return nil, err
			}
			q.groupBy = append(q.groupBy, e)
			if !p.isOp(",") {
				break
			}
		}
	}
	if p.kw("having") {
		return nil, fmt.Errorf("HAVING is not supported")
	}
	if p.kw("order") {
		if !p.kw("by") {
			return nil, fmt.Errorf("expected BY after ORDER")
		}
		for {
			e, err := p.expr()
			if err != nil {
				return nil, err
			}
			o := tsqOrder{expr: e}
			if p.kw("desc") {
				o.desc = true
			} else {
				p.kw("asc")
			}
			q.orderBy = append(q.orderBy, o)
			if !p.isOp(",") {
				break
			}
		}
	}
	if p.kw("limit") {
		t := p.next()
		n, err := strconv.ParseInt(t.text, 10, 32)
		if t.kind != "num" || err != nil || n < 0 {
			return nil, fmt.Errorf("LIMIT needs a non-negative integer")
		}
		q.limit = int(n)
	}
	p.isOp(";")
	if t := p.peek(); t.kind != "eof" {
		return nil, fmt.Errorf("unsupported syntax near %q", t.text)
	}
	for name := range p.params {
		q.params = append(q.params, name)
	}
	sort.Strings(q.params)
	return q, nil
}

func (p *tsqParser) boolExpr() (*tsqNode, error) {
	if p.depth++; p.depth > tsqMaxDepth {
		return nil, fmt.Errorf("the query nests expressions more than %d levels deep", tsqMaxDepth)
	}
	defer func() { p.depth-- }()
	left, err := p.andExpr()
	if err != nil {
		return nil, err
	}
	for p.kw("or") {
		right, err := p.andExpr()
		if err != nil {
			return nil, err
		}
		left = &tsqNode{op: "or", args: []*tsqNode{left, right}}
	}
	return left, nil
}

func (p *tsqParser) andExpr() (*tsqNode, error) {
	left, err := p.notExpr()
	if err != nil {
		return nil, err
	}
	for p.kw("and") {
		right, err := p.notExpr()
		if err != nil {
			return nil, err
		}
		left = &tsqNode{op: "and", args: []*tsqNode{left, right}}
	}
	return left, nil
}

func (p *tsqParser) notExpr() (*tsqNode, error) {
	if p.depth++; p.depth > tsqMaxDepth {
		return nil, fmt.Errorf("the query nests expressions more than %d levels deep", tsqMaxDepth)
	}
	defer func() { p.depth-- }()
	if p.kw("not") {
		e, err := p.notExpr()
		if err != nil {
			return nil, err
		}
		return &tsqNode{op: "not", args: []*tsqNode{e}}, nil
	}
	return p.predicate()
}

func (p *tsqParser) predicate() (*tsqNode, error) {
	// A parenthesised condition, e.g. (a = 1 OR b = 2).
	if t := p.peek(); t.kind == "op" && t.text == "(" {
		save := p.pos
		p.pos++
		if e, err := p.boolExpr(); err == nil && p.isOp(")") {
			return e, nil
		}
		p.pos = save
	}
	left, err := p.expr()
	if err != nil {
		return nil, err
	}
	negate := p.kw("not")
	switch {
	case p.kw("between"):
		lo, err := p.expr()
		if err != nil {
			return nil, err
		}
		if !p.kw("and") {
			return nil, fmt.Errorf("expected AND in BETWEEN")
		}
		hi, err := p.expr()
		if err != nil {
			return nil, err
		}
		return tsqMaybeNot(&tsqNode{op: "between", args: []*tsqNode{left, lo, hi}}, negate), nil
	case p.kw("in"):
		if err := p.expectOp("("); err != nil {
			return nil, err
		}
		args := []*tsqNode{left}
		for {
			e, err := p.expr()
			if err != nil {
				return nil, err
			}
			args = append(args, e)
			if !p.isOp(",") {
				break
			}
		}
		if err := p.expectOp(")"); err != nil {
			return nil, err
		}
		return tsqMaybeNot(&tsqNode{op: "in", args: args}, negate), nil
	case p.kw("like"):
		pat, err := p.expr()
		if err != nil {
			return nil, err
		}
		return tsqMaybeNot(&tsqNode{op: "like", args: []*tsqNode{left, pat}}, negate), nil
	case negate:
		return nil, fmt.Errorf("expected BETWEEN, IN or LIKE after NOT")
	case p.kw("is"):
		not := p.kw("not")
		if !p.kw("null") {
			return nil, fmt.Errorf("expected NULL after IS")
		}
		return tsqMaybeNot(&tsqNode{op: "isnull", args: []*tsqNode{left}}, not), nil
	}
	if t := p.peek(); t.kind == "op" {
		switch t.text {
		case "=", "!=", "<>", "<", "<=", ">", ">=":
			p.pos++
			right, err := p.expr()
			if err != nil {
				return nil, err
			}
			return &tsqNode{op: "cmp", name: t.text, args: []*tsqNode{left, right}}, nil
		}
	}
	return left, nil // a bare boolean expression
}

func tsqMaybeNot(n *tsqNode, not bool) *tsqNode {
	if not {
		return &tsqNode{op: "not", args: []*tsqNode{n}}
	}
	return n
}

func (p *tsqParser) expr() (*tsqNode, error) {
	if p.depth++; p.depth > tsqMaxDepth {
		return nil, fmt.Errorf("the query nests expressions more than %d levels deep", tsqMaxDepth)
	}
	defer func() { p.depth-- }()
	left, err := p.term()
	if err != nil {
		return nil, err
	}
	for {
		t := p.peek()
		if t.kind != "op" || (t.text != "+" && t.text != "-") {
			return left, nil
		}
		p.pos++
		right, err := p.term()
		if err != nil {
			return nil, err
		}
		left = &tsqNode{op: "arith", name: t.text, args: []*tsqNode{left, right}, text: left.text + " " + t.text + " " + right.text}
	}
}

func (p *tsqParser) term() (*tsqNode, error) {
	left, err := p.factor()
	if err != nil {
		return nil, err
	}
	for {
		t := p.peek()
		if t.kind != "op" || (t.text != "*" && t.text != "/" && t.text != "%") {
			return left, nil
		}
		p.pos++
		right, err := p.factor()
		if err != nil {
			return nil, err
		}
		left = &tsqNode{op: "arith", name: t.text, args: []*tsqNode{left, right}, text: left.text + " " + t.text + " " + right.text}
	}
}

func (p *tsqParser) factor() (*tsqNode, error) {
	if p.depth++; p.depth > tsqMaxDepth {
		return nil, fmt.Errorf("the query nests expressions more than %d levels deep", tsqMaxDepth)
	}
	defer func() { p.depth-- }()
	if p.isOp("-") {
		e, err := p.factor()
		if err != nil {
			return nil, err
		}
		return &tsqNode{op: "neg", args: []*tsqNode{e}, text: "-" + e.text}, nil
	}
	return p.primary()
}

var tsqAggs = map[string]bool{"count": true, "avg": true, "sum": true, "min": true, "max": true}

var tsqFuncs = map[string]int{"now": 0, "ago": 1, "bin": 2, "from_milliseconds": 1, "to_milliseconds": 1, "abs": 1, "round": -1}

func (p *tsqParser) primary() (*tsqNode, error) {
	t := p.next()
	switch t.kind {
	case "num":
		if strings.ContainsAny(t.text, ".eE") {
			f, err := strconv.ParseFloat(t.text, 64)
			if err != nil {
				return nil, fmt.Errorf("bad number %q", t.text)
			}
			return &tsqNode{op: "lit", val: tsqVal{typ: "DOUBLE", f: f}, text: t.text}, nil
		}
		n, err := strconv.ParseInt(t.text, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("bad number %q", t.text)
		}
		return &tsqNode{op: "lit", val: tsqVal{typ: "BIGINT", i: n}, text: t.text}, nil
	case "dur":
		d, err := tsqDuration(t.text)
		if err != nil {
			return nil, err
		}
		return &tsqNode{op: "lit", val: tsqVal{typ: "INTERVAL_DAY_TO_SECOND", i: d}, text: t.text}, nil
	case "str":
		return &tsqNode{op: "lit", val: tsqVal{typ: "VARCHAR", s: t.text}, text: "'" + t.text + "'"}, nil
	case "param":
		p.params[t.text] = true
		return &tsqNode{op: "param", name: t.text, text: t.text}, nil
	case "qident":
		return &tsqNode{op: "col", name: t.text, text: `"` + t.text + `"`}, nil
	case "ident":
		low := strings.ToLower(t.text)
		switch low {
		case "true", "false":
			return &tsqNode{op: "lit", val: tsqVal{typ: "BOOLEAN", b: low == "true"}, text: low}, nil
		case "null":
			return &tsqNode{op: "lit", text: "null"}, nil
		case "case", "cast", "select", "exists":
			return nil, fmt.Errorf("%s expressions are not supported", strings.ToUpper(low))
		}
		if p.isOp("::") {
			typ := p.next()
			if typ.kind != "ident" {
				return nil, fmt.Errorf("expected a type after ::")
			}
			if low != "measure_value" {
				return nil, fmt.Errorf("the :: cast is only supported on measure_value")
			}
			name := "measure_value::" + strings.ToLower(typ.text)
			return &tsqNode{op: "col", name: name, text: name}, nil
		}
		if !p.isOp("(") {
			return &tsqNode{op: "col", name: t.text, text: t.text}, nil
		}
		n := &tsqNode{name: low}
		switch {
		case tsqAggs[low]:
			n.op = "agg"
			if p.isOp("*") {
				if low != "count" {
					return nil, fmt.Errorf("only count accepts *")
				}
				n.args = []*tsqNode{{op: "star", text: "*"}}
			}
		default:
			if _, ok := tsqFuncs[low]; !ok {
				return nil, fmt.Errorf("function %s is not supported", low)
			}
			n.op = "call"
		}
		if len(n.args) == 0 && !(n.op == "call" && p.peek().kind == "op" && p.peek().text == ")") {
			for {
				e, err := p.expr()
				if err != nil {
					return nil, err
				}
				n.args = append(n.args, e)
				if !p.isOp(",") {
					break
				}
			}
		}
		if err := p.expectOp(")"); err != nil {
			return nil, err
		}
		if want := tsqFuncs[low]; n.op == "call" && want >= 0 && len(n.args) != want {
			return nil, fmt.Errorf("%s takes %d arguments", low, want)
		}
		if n.op == "agg" && len(n.args) != 1 {
			return nil, fmt.Errorf("%s takes one argument", low)
		}
		parts := make([]string, len(n.args))
		for i, a := range n.args {
			parts[i] = a.text
		}
		n.text = low + "(" + strings.Join(parts, ", ") + ")"
		return n, nil
	case "op":
		if t.text == "(" {
			e, err := p.expr()
			if err != nil {
				return nil, err
			}
			if err := p.expectOp(")"); err != nil {
				return nil, err
			}
			return e, nil
		}
	}
	return nil, fmt.Errorf("unexpected %q", t.text)
}

func tsqDuration(s string) (int64, error) {
	i := 0
	for i < len(s) && (s[i] >= '0' && s[i] <= '9') {
		i++
	}
	n, err := strconv.ParseInt(s[:i], 10, 32)
	if err != nil || i == 0 {
		return 0, fmt.Errorf("bad interval %q", s)
	}
	units := map[string]time.Duration{"ns": time.Nanosecond, "us": time.Microsecond, "ms": time.Millisecond, "s": time.Second, "m": time.Minute, "h": time.Hour, "d": 24 * time.Hour}
	u, ok := units[strings.ToLower(s[i:])]
	if !ok {
		return 0, fmt.Errorf("bad interval unit in %q", s)
	}
	if n > math.MaxInt64/int64(u) {
		return 0, fmt.Errorf("interval %q is too large", s)
	}
	return n * int64(u), nil
}

// --- evaluation ---

type tsqEnv struct {
	now    int64
	params map[string]tsqVal
	types  map[string]string // column -> type for this table
	work   int64             // LIKE comparison steps so far
	over   bool              // set once work passes tsqMaxWork
}

// tsqMaxWork bounds the LIKE work one query may do (len(value)*len(pattern) per row).
const tsqMaxWork = 50_000_000

func tsqColumn(r *tsmRecord, name string) tsqVal {
	switch name {
	case "time":
		return tsqVal{typ: "TIMESTAMP", i: r.t}
	case "measure_name":
		return tsqVal{typ: "VARCHAR", s: r.measure}
	}
	if typ, ok := strings.CutPrefix(name, "measure_value::"); ok {
		if r.typ == strings.ToUpper(typ) {
			return tsqScalar(r.typ, r.value)
		}
		return tsqVal{}
	}
	if v, ok := r.dims[name]; ok {
		return tsqVal{typ: "VARCHAR", s: v}
	}
	for _, m := range r.multi {
		if m.name == name {
			return tsqScalar(m.typ, m.value)
		}
	}
	return tsqVal{}
}

func tsqTruth(v tsqVal) bool { return v.typ == "BOOLEAN" && v.b }

func tsqBool(b bool) tsqVal { return tsqVal{typ: "BOOLEAN", b: b} }

// tsqLike matches SQL LIKE (% any run, _ one character) in O(len(s)*len(pat))
// time, so a hostile pattern can't blow up.
func tsqLike(s, pat string) bool {
	str, p := []rune(s), []rune(pat)
	prev := make([]bool, len(str)+1)
	prev[0] = true
	for _, c := range p {
		cur := make([]bool, len(str)+1)
		if c == '%' {
			cur[0] = prev[0]
		}
		for i := 1; i <= len(str); i++ {
			switch c {
			case '%':
				cur[i] = prev[i] || cur[i-1]
			case '_':
				cur[i] = prev[i-1]
			default:
				cur[i] = prev[i-1] && str[i-1] == c
			}
		}
		prev = cur
	}
	return prev[len(str)]
}

// eval computes an expression from this package's own SQL AST over one
// record; it never executes code.
func (env *tsqEnv) eval(n *tsqNode, r *tsmRecord) tsqVal {
	switch n.op {
	case "lit":
		return n.val
	case "col":
		return tsqColumn(r, env.resolve(n.name))
	case "param":
		return env.params[n.name]
	case "neg":
		v := env.eval(n.args[0], r)
		switch v.typ {
		case "DOUBLE":
			v.f = -v.f
		case "BIGINT", "INTERVAL_DAY_TO_SECOND":
			v.i = -v.i
		default:
			return tsqVal{}
		}
		return v
	case "arith":
		return tsqArith(n.name, env.eval(n.args[0], r), env.eval(n.args[1], r))
	case "call":
		return env.call(n, func(a *tsqNode) tsqVal { return env.eval(a, r) })
	case "cmp", "between", "in", "like", "isnull", "and", "or", "not":
		return env.logic(n, func(a *tsqNode) tsqVal { return env.eval(a, r) })
	}
	return tsqVal{}
}

func (env *tsqEnv) logic(n *tsqNode, ev func(*tsqNode) tsqVal) tsqVal {
	switch n.op {
	case "and":
		return tsqBool(tsqTruth(ev(n.args[0])) && tsqTruth(ev(n.args[1])))
	case "or":
		return tsqBool(tsqTruth(ev(n.args[0])) || tsqTruth(ev(n.args[1])))
	case "not":
		v := ev(n.args[0])
		if v.null() {
			return v
		}
		return tsqBool(!tsqTruth(v))
	case "isnull":
		return tsqBool(ev(n.args[0]).null())
	case "like":
		a, b := ev(n.args[0]), ev(n.args[1])
		if a.typ != "VARCHAR" || b.typ != "VARCHAR" {
			return tsqVal{}
		}
		if env.work += int64(len(a.s)+1) * int64(len(b.s)+1); env.work > tsqMaxWork {
			env.over = true
			return tsqVal{}
		}
		return tsqBool(tsqLike(a.s, b.s))
	case "in":
		a := ev(n.args[0])
		if a.null() {
			return tsqVal{}
		}
		for _, x := range n.args[1:] {
			if c, ok := tsqCompare(a, ev(x)); ok && c == 0 {
				return tsqBool(true)
			}
		}
		return tsqBool(false)
	case "between":
		a, lo, hi := ev(n.args[0]), ev(n.args[1]), ev(n.args[2])
		if a.null() || lo.null() || hi.null() {
			return tsqVal{}
		}
		c1, ok1 := tsqCompare(a, lo)
		c2, ok2 := tsqCompare(a, hi)
		return tsqBool(ok1 && ok2 && c1 >= 0 && c2 <= 0)
	case "cmp":
		a, b := ev(n.args[0]), ev(n.args[1])
		if a.null() || b.null() {
			return tsqVal{}
		}
		c, ok := tsqCompare(a, b)
		if !ok {
			return tsqVal{}
		}
		switch n.name {
		case "=":
			return tsqBool(c == 0)
		case "!=", "<>":
			return tsqBool(c != 0)
		case "<":
			return tsqBool(c < 0)
		case "<=":
			return tsqBool(c <= 0)
		case ">":
			return tsqBool(c > 0)
		}
		return tsqBool(c >= 0)
	}
	return tsqVal{}
}

func (env *tsqEnv) call(n *tsqNode, ev func(*tsqNode) tsqVal) tsqVal {
	switch n.name {
	case "now":
		return tsqVal{typ: "TIMESTAMP", i: env.now}
	case "ago":
		d := ev(n.args[0])
		if d.typ != "INTERVAL_DAY_TO_SECOND" {
			return tsqVal{}
		}
		return tsqVal{typ: "TIMESTAMP", i: env.now - d.i}
	case "bin":
		t, d := ev(n.args[0]), ev(n.args[1])
		if t.typ != "TIMESTAMP" || d.typ != "INTERVAL_DAY_TO_SECOND" || d.i <= 0 {
			return tsqVal{}
		}
		return tsqVal{typ: "TIMESTAMP", i: t.i - ((t.i%d.i)+d.i)%d.i}
	case "from_milliseconds":
		v := ev(n.args[0])
		if v.typ != "BIGINT" {
			return tsqVal{}
		}
		return tsqVal{typ: "TIMESTAMP", i: v.i * 1_000_000}
	case "to_milliseconds":
		v := ev(n.args[0])
		if v.typ != "TIMESTAMP" {
			return tsqVal{}
		}
		return tsqVal{typ: "BIGINT", i: v.i / 1_000_000}
	case "abs":
		v := ev(n.args[0])
		switch v.typ {
		case "DOUBLE":
			v.f = math.Abs(v.f)
		case "BIGINT":
			if v.i < 0 {
				v.i = -v.i
			}
		default:
			return tsqVal{}
		}
		return v
	case "round":
		if len(n.args) < 1 || len(n.args) > 2 {
			return tsqVal{}
		}
		v := ev(n.args[0])
		x, ok := v.num()
		if !ok || v.typ == "TIMESTAMP" {
			return tsqVal{}
		}
		digits := int64(0)
		if len(n.args) == 2 {
			d := ev(n.args[1])
			if d.typ != "BIGINT" || d.i < 0 || d.i > 15 {
				return tsqVal{}
			}
			digits = d.i
		}
		scale := math.Pow(10, float64(digits))
		if v.typ == "BIGINT" {
			return v
		}
		return tsqVal{typ: "DOUBLE", f: math.Round(x*scale) / scale}
	}
	return tsqVal{}
}

func tsqArith(op string, a, b tsqVal) tsqVal {
	if a.null() || b.null() {
		return tsqVal{}
	}
	if a.typ == "TIMESTAMP" && b.typ == "INTERVAL_DAY_TO_SECOND" {
		switch op {
		case "+":
			return tsqVal{typ: "TIMESTAMP", i: a.i + b.i}
		case "-":
			return tsqVal{typ: "TIMESTAMP", i: a.i - b.i}
		}
		return tsqVal{}
	}
	if a.typ == "TIMESTAMP" && b.typ == "TIMESTAMP" && op == "-" {
		return tsqVal{typ: "INTERVAL_DAY_TO_SECOND", i: a.i - b.i}
	}
	if a.typ == "BIGINT" && b.typ == "BIGINT" {
		switch op {
		case "+":
			return tsqVal{typ: "BIGINT", i: a.i + b.i}
		case "-":
			return tsqVal{typ: "BIGINT", i: a.i - b.i}
		case "*":
			return tsqVal{typ: "BIGINT", i: a.i * b.i}
		case "/":
			if b.i == 0 {
				return tsqVal{}
			}
			return tsqVal{typ: "BIGINT", i: a.i / b.i}
		case "%":
			if b.i == 0 {
				return tsqVal{}
			}
			return tsqVal{typ: "BIGINT", i: a.i % b.i}
		}
	}
	x, okx := a.num()
	y, oky := b.num()
	if !okx || !oky || a.typ == "TIMESTAMP" || b.typ == "TIMESTAMP" {
		return tsqVal{}
	}
	switch op {
	case "+":
		return tsqVal{typ: "DOUBLE", f: x + y}
	case "-":
		return tsqVal{typ: "DOUBLE", f: x - y}
	case "*":
		return tsqVal{typ: "DOUBLE", f: x * y}
	case "/":
		if y == 0 {
			return tsqVal{}
		}
		return tsqVal{typ: "DOUBLE", f: x / y}
	case "%":
		if y == 0 {
			return tsqVal{}
		}
		return tsqVal{typ: "DOUBLE", f: math.Mod(x, y)}
	}
	return tsqVal{}
}

// resolve maps an identifier to a known column, case-insensitively for
// unquoted names that don't match exactly.
func (env *tsqEnv) resolve(name string) string {
	if _, ok := env.types[name]; ok {
		return name
	}
	for col := range env.types {
		if strings.EqualFold(col, name) {
			return col
		}
	}
	return name
}

// aggregate evaluates n over a group of records.
func (env *tsqEnv) aggregate(n *tsqNode, group []*tsmRecord) tsqVal {
	switch n.op {
	case "agg":
		arg := n.args[0]
		var count int64
		var acc tsqVal
		var sum float64
		allInt := true
		for _, r := range group {
			var v tsqVal
			if arg.op == "star" {
				v = tsqVal{typ: "BIGINT", i: 1}
			} else {
				v = env.eval(arg, r)
			}
			if v.null() {
				continue
			}
			count++
			switch n.name {
			case "sum", "avg":
				x, ok := v.num()
				if !ok || v.typ == "TIMESTAMP" {
					continue
				}
				if v.typ != "BIGINT" {
					allInt = false
				}
				sum += x
			case "min", "max":
				if acc.null() {
					acc = v
				} else if c, ok := tsqCompare(v, acc); ok && ((n.name == "min" && c < 0) || (n.name == "max" && c > 0)) {
					acc = v
				}
			}
		}
		switch n.name {
		case "count":
			return tsqVal{typ: "BIGINT", i: count}
		case "sum":
			if count == 0 {
				return tsqVal{}
			}
			if allInt {
				return tsqVal{typ: "BIGINT", i: int64(sum)}
			}
			return tsqVal{typ: "DOUBLE", f: sum}
		case "avg":
			if count == 0 {
				return tsqVal{}
			}
			return tsqVal{typ: "DOUBLE", f: sum / float64(count)}
		}
		return acc
	case "lit", "param":
		return env.eval(n, nil)
	}
	if !tsqHasAgg(n) {
		if len(group) == 0 {
			return tsqVal{}
		}
		return env.eval(n, group[0])
	}
	switch n.op {
	case "arith":
		return tsqArith(n.name, env.aggregate(n.args[0], group), env.aggregate(n.args[1], group))
	case "neg":
		return env.eval(&tsqNode{op: "neg", args: []*tsqNode{{op: "lit", val: env.aggregate(n.args[0], group)}}}, nil)
	case "call":
		return env.call(n, func(a *tsqNode) tsqVal { return env.aggregate(a, group) })
	}
	return env.logic(n, func(a *tsqNode) tsqVal { return env.aggregate(a, group) })
}

func tsqHasAgg(n *tsqNode) bool {
	if n.op == "agg" {
		return true
	}
	for _, a := range n.args {
		if tsqHasAgg(a) {
			return true
		}
	}
	return false
}

// typeOf infers an expression's result type for ColumnInfo.
func (env *tsqEnv) typeOf(n *tsqNode) string {
	switch n.op {
	case "lit":
		if n.val.null() {
			return "UNKNOWN"
		}
		return n.val.typ
	case "col":
		if t := env.types[env.resolve(n.name)]; t != "" {
			return t
		}
		if typ, ok := strings.CutPrefix(n.name, "measure_value::"); ok {
			return strings.ToUpper(typ)
		}
		return "VARCHAR"
	case "param":
		return "TIMESTAMP"
	case "agg":
		switch n.name {
		case "count":
			return "BIGINT"
		case "avg":
			return "DOUBLE"
		}
		if n.args[0].op == "star" {
			return "BIGINT"
		}
		return env.typeOf(n.args[0])
	case "call":
		switch n.name {
		case "now", "ago", "bin", "from_milliseconds":
			return "TIMESTAMP"
		case "to_milliseconds":
			return "BIGINT"
		}
		return env.typeOf(n.args[0])
	case "arith":
		a, b := env.typeOf(n.args[0]), env.typeOf(n.args[1])
		switch {
		case a == "TIMESTAMP" && b == "INTERVAL_DAY_TO_SECOND":
			return "TIMESTAMP"
		case a == "TIMESTAMP" && b == "TIMESTAMP":
			return "INTERVAL_DAY_TO_SECOND"
		case a == "BIGINT" && b == "BIGINT":
			return "BIGINT"
		}
		return "DOUBLE"
	case "neg":
		return env.typeOf(n.args[0])
	}
	return "BOOLEAN"
}

// checkColumns reports the first column reference the table doesn't have.
func (env *tsqEnv) checkColumns(n *tsqNode) error {
	if n == nil {
		return nil
	}
	if n.op == "col" {
		name := env.resolve(n.name)
		if _, ok := env.types[name]; !ok && !strings.HasPrefix(name, "measure_value::") {
			return fmt.Errorf("column '%s' cannot be resolved", n.name)
		}
		if typ, ok := strings.CutPrefix(name, "measure_value::"); ok {
			switch typ {
			case "double", "bigint", "varchar", "boolean", "timestamp":
			default:
				return fmt.Errorf("unknown measure value type %s", typ)
			}
		}
	}
	for _, a := range n.args {
		if err := env.checkColumns(a); err != nil {
			return err
		}
	}
	return nil
}
