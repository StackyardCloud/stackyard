package server

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// A parser for the subset of Cedar that Verified Permissions evaluates here:
// permit/forbid policies (and templates with ?principal / ?resource slots)
// with ==, in and is scope constraints, and when/unless conditions over
// literals, entities, sets, records, attribute access, has, like, is,
// if-then-else, comparison, arithmetic, boolean logic and the set methods
// contains, containsAll, containsAny and isEmpty.
// ponytail: extension functions (ip, decimal, datetime...) and entity tags are
// rejected at parse time; add them when a workload needs them.

const (
	vpMaxPolicyBytes = 10000 // AWS's per-policy size quota
	vpMaxParseDepth  = 64    // nesting of expressions
)

type cedarEntity struct{ Type, ID string }

func (e cedarEntity) String() string { return e.Type + "::" + strconv.Quote(e.ID) }

type cedarPolicy struct {
	Effect     string // "permit" or "forbid"
	Principal  cedarScope
	Action     cedarActionScope
	Resource   cedarScope
	Conditions []cedarCondition
}

// cedarScope is a principal or resource constraint. Op is "" (any), "==",
// "in" or "is"; Entity or Slot gives the right-hand side (InEntity/InSlot for
// "is T in E").
type cedarScope struct {
	Op     string
	Entity *cedarEntity
	Slot   bool
	IsType string
}

type cedarActionScope struct {
	Op       string // "", "==", "in"
	Entities []cedarEntity
}

type cedarCondition struct {
	When bool // false: unless
	Expr cedarExpr
}

type cedarExpr interface{}

type (
	cedarLit   struct{ V any }
	cedarVar   struct{ Name string }
	cedarUnary struct {
		Op string
		X  cedarExpr
	}
	cedarBinary struct {
		Op   string
		L, R cedarExpr
	}
	cedarAttr struct {
		X    cedarExpr
		Name string
	}
	cedarHas struct {
		X    cedarExpr
		Name string
	}
	cedarLike struct {
		X       cedarExpr
		Pattern []cedarPatternPart
	}
	cedarIs struct {
		X    cedarExpr
		Type string
		In   cedarExpr // optional
	}
	cedarIf     struct{ C, T, E cedarExpr }
	cedarSetLit struct{ Elems []cedarExpr }
	cedarRecLit struct {
		Keys []string
		Vals []cedarExpr
	}
	cedarCall struct {
		Recv   cedarExpr
		Method string
		Args   []cedarExpr
	}
)

// cedarPatternPart is a literal run or a wildcard in a like pattern.
type cedarPatternPart struct {
	Wildcard bool
	Lit      string
}

type cedarToken struct {
	kind      string // "ident", "string", "int", "op", "eof"
	text      string // identifier, operator, or decoded string
	raw       string // string contents before unescaping (for like patterns)
	star      bool   // contains \*, which is only valid in a like pattern
	line, col int
}

type cedarParseError struct {
	line, col int
	msg       string
}

func (e *cedarParseError) Error() string {
	return fmt.Sprintf("line %d, column %d: %s", e.line, e.col, e.msg)
}

func cedarLex(src string) ([]cedarToken, error) {
	var toks []cedarToken
	line, col := 1, 1
	i := 0
	advance := func(n int) {
		for _, r := range src[i : i+n] {
			if r == '\n' {
				line, col = line+1, 1
			} else {
				col++
			}
		}
		i += n
	}
	for i < len(src) {
		r, size := utf8.DecodeRuneInString(src[i:])
		switch {
		case unicode.IsSpace(r):
			advance(size)
		case strings.HasPrefix(src[i:], "//"):
			end := strings.IndexByte(src[i:], '\n')
			if end < 0 {
				end = len(src) - i
			}
			advance(end)
		case r == '_' || unicode.IsLetter(r):
			j := i
			for j < len(src) {
				c, n := utf8.DecodeRuneInString(src[j:])
				if c != '_' && !unicode.IsLetter(c) && !unicode.IsDigit(c) {
					break
				}
				j += n
			}
			toks = append(toks, cedarToken{kind: "ident", text: src[i:j], line: line, col: col})
			advance(j - i)
		case r >= '0' && r <= '9':
			j := i
			for j < len(src) && src[j] >= '0' && src[j] <= '9' {
				j++
			}
			toks = append(toks, cedarToken{kind: "int", text: src[i:j], line: line, col: col})
			advance(j - i)
		case r == '"':
			j := i + 1
			for j < len(src) && src[j] != '"' {
				if src[j] == '\\' {
					j++
				}
				j++
			}
			if j >= len(src) {
				return nil, &cedarParseError{line, col, "unterminated string literal"}
			}
			raw := src[i+1 : j]
			text, err := cedarUnescape(raw, true)
			if err != nil {
				return nil, &cedarParseError{line, col, err.Error()}
			}
			toks = append(toks, cedarToken{kind: "string", text: text, raw: raw, star: strings.Contains(text, `\*`), line: line, col: col})
			advance(j + 1 - i)
		default:
			op := ""
			for _, cand := range []string{"==", "!=", "<=", ">=", "&&", "||", "::", "<", ">", "!", "(", ")", "[", "]", "{", "}", ",", ";", ".", "+", "-", "*", "@", "?", ":"} {
				if strings.HasPrefix(src[i:], cand) {
					op = cand
					break
				}
			}
			if op == "" {
				return nil, &cedarParseError{line, col, fmt.Sprintf("unexpected character %q", r)}
			}
			toks = append(toks, cedarToken{kind: "op", text: op, line: line, col: col})
			advance(len(op))
		}
	}
	return append(toks, cedarToken{kind: "eof", line: line, col: col}), nil
}

// cedarUnescape decodes Cedar string escapes. In a like pattern, \* is a
// literal star (kept escaped so the pattern splitter can tell it apart).
func cedarUnescape(raw string, pattern bool) (string, error) {
	var b strings.Builder
	for i := 0; i < len(raw); i++ {
		c := raw[i]
		if c != '\\' {
			b.WriteByte(c)
			continue
		}
		i++
		if i >= len(raw) {
			return "", fmt.Errorf("bad escape at end of string")
		}
		switch raw[i] {
		case 'n':
			b.WriteByte('\n')
		case 'r':
			b.WriteByte('\r')
		case 't':
			b.WriteByte('\t')
		case '0':
			b.WriteByte(0)
		case '\\', '"', '\'':
			b.WriteByte(raw[i])
		case '*':
			if !pattern {
				return "", fmt.Errorf(`\* is only allowed in a like pattern`)
			}
			b.WriteString(`\*`)
		case 'u':
			end := strings.IndexByte(raw[i:], '}')
			if i+1 >= len(raw) || raw[i+1] != '{' || end < 0 {
				return "", fmt.Errorf(`bad \u escape`)
			}
			n, err := strconv.ParseUint(raw[i+2:i+end], 16, 32)
			if err != nil || n > unicode.MaxRune {
				return "", fmt.Errorf(`bad \u escape`)
			}
			b.WriteRune(rune(n))
			i += end
		default:
			return "", fmt.Errorf("unknown escape \\%c", raw[i])
		}
	}
	return b.String(), nil
}

type cedarParser struct {
	toks     []cedarToken
	pos      int
	depth    int
	template bool // slots allowed
	slots    map[string]bool
}

// cedarParsePolicy parses exactly one policy. template allows ?principal and
// ?resource slots in the scope.
func cedarParsePolicy(src string, template bool) (*cedarPolicy, map[string]bool, error) {
	if len(src) > vpMaxPolicyBytes {
		return nil, nil, fmt.Errorf("policy exceeds the maximum size of %d bytes", vpMaxPolicyBytes)
	}
	toks, err := cedarLex(src)
	if err != nil {
		return nil, nil, err
	}
	p := &cedarParser{toks: toks, template: template, slots: map[string]bool{}}
	pol, err := p.policy()
	if err != nil {
		return nil, nil, err
	}
	if t := p.peek(); t.kind != "eof" {
		return nil, nil, p.errAt(t, "expected exactly one policy, found more input")
	}
	return pol, p.slots, nil
}

func (p *cedarParser) peek() cedarToken { return p.toks[p.pos] }

func (p *cedarParser) next() cedarToken {
	t := p.toks[p.pos]
	if t.kind != "eof" {
		p.pos++
	}
	return t
}

// str consumes a plain string token: \* is only valid in a like pattern.
func (p *cedarParser) str(t cedarToken) error {
	if t.star {
		return p.errAt(t, `\* is only allowed in a like pattern`)
	}
	return nil
}

func (p *cedarParser) errAt(t cedarToken, format string, args ...any) error {
	return &cedarParseError{t.line, t.col, fmt.Sprintf(format, args...)}
}

func (p *cedarParser) isOp(op string) bool {
	t := p.peek()
	return t.kind == "op" && t.text == op
}

func (p *cedarParser) isIdent(word string) bool {
	t := p.peek()
	return t.kind == "ident" && t.text == word
}

func (p *cedarParser) expectOp(op string) error {
	if t := p.next(); t.kind != "op" || t.text != op {
		return p.errAt(t, "expected %q, found %s", op, cedarDescribe(t))
	}
	return nil
}

func (p *cedarParser) expectIdent(word string) error {
	if t := p.next(); t.kind != "ident" || t.text != word {
		return p.errAt(t, "expected %q, found %s", word, cedarDescribe(t))
	}
	return nil
}

func cedarDescribe(t cedarToken) string {
	switch t.kind {
	case "eof":
		return "end of input"
	case "string":
		return strconv.Quote(t.text)
	}
	return strconv.Quote(t.text)
}

func (p *cedarParser) policy() (*cedarPolicy, error) {
	for p.isOp("@") { // annotations are accepted and ignored
		p.next()
		if t := p.next(); t.kind != "ident" {
			return nil, p.errAt(t, "expected an annotation name")
		}
		if p.isOp("(") {
			p.next()
			if t := p.next(); t.kind != "string" {
				return nil, p.errAt(t, "expected an annotation string")
			}
			if err := p.expectOp(")"); err != nil {
				return nil, err
			}
		}
	}
	t := p.next()
	if t.kind != "ident" || (t.text != "permit" && t.text != "forbid") {
		return nil, p.errAt(t, "expected permit or forbid, found %s", cedarDescribe(t))
	}
	pol := &cedarPolicy{Effect: t.text}
	if err := p.expectOp("("); err != nil {
		return nil, err
	}
	var err error
	if pol.Principal, err = p.scope("principal"); err != nil {
		return nil, err
	}
	if err := p.expectOp(","); err != nil {
		return nil, err
	}
	if pol.Action, err = p.actionScope(); err != nil {
		return nil, err
	}
	if err := p.expectOp(","); err != nil {
		return nil, err
	}
	if pol.Resource, err = p.scope("resource"); err != nil {
		return nil, err
	}
	if err := p.expectOp(")"); err != nil {
		return nil, err
	}
	for p.isIdent("when") || p.isIdent("unless") {
		when := p.next().text == "when"
		if err := p.expectOp("{"); err != nil {
			return nil, err
		}
		e, err := p.expr()
		if err != nil {
			return nil, err
		}
		if err := p.expectOp("}"); err != nil {
			return nil, err
		}
		pol.Conditions = append(pol.Conditions, cedarCondition{When: when, Expr: e})
	}
	if err := p.expectOp(";"); err != nil {
		return nil, err
	}
	return pol, nil
}

func (p *cedarParser) scope(variable string) (cedarScope, error) {
	if err := p.expectIdent(variable); err != nil {
		return cedarScope{}, err
	}
	switch {
	case p.isOp("=="), p.isIdent("in"):
		op := p.next().text
		ent, slot, err := p.entityOrSlot(variable)
		return cedarScope{Op: op, Entity: ent, Slot: slot}, err
	case p.isIdent("is"):
		p.next()
		typ, err := p.path()
		if err != nil {
			return cedarScope{}, err
		}
		sc := cedarScope{Op: "is", IsType: typ}
		if p.isIdent("in") {
			p.next()
			sc.Entity, sc.Slot, err = p.entityOrSlot(variable)
		}
		return sc, err
	}
	return cedarScope{}, nil
}

func (p *cedarParser) entityOrSlot(variable string) (*cedarEntity, bool, error) {
	if p.isOp("?") {
		t := p.next()
		name := p.next()
		if name.kind != "ident" || name.text != variable {
			return nil, false, p.errAt(name, "expected ?%s", variable)
		}
		if !p.template {
			return nil, false, p.errAt(t, "a static policy can't contain the slot ?%s; use a policy template", variable)
		}
		p.slots[variable] = true
		return nil, true, nil
	}
	e, err := p.entity()
	return e, false, err
}

func (p *cedarParser) actionScope() (cedarActionScope, error) {
	if err := p.expectIdent("action"); err != nil {
		return cedarActionScope{}, err
	}
	switch {
	case p.isOp("=="):
		p.next()
		e, err := p.entity()
		if err != nil {
			return cedarActionScope{}, err
		}
		return cedarActionScope{Op: "==", Entities: []cedarEntity{*e}}, nil
	case p.isIdent("in"):
		p.next()
		if !p.isOp("[") {
			e, err := p.entity()
			if err != nil {
				return cedarActionScope{}, err
			}
			return cedarActionScope{Op: "in", Entities: []cedarEntity{*e}}, nil
		}
		p.next()
		var list []cedarEntity
		for !p.isOp("]") {
			e, err := p.entity()
			if err != nil {
				return cedarActionScope{}, err
			}
			list = append(list, *e)
			if !p.isOp(",") {
				break
			}
			p.next()
		}
		if err := p.expectOp("]"); err != nil {
			return cedarActionScope{}, err
		}
		return cedarActionScope{Op: "in", Entities: list}, nil
	}
	return cedarActionScope{}, nil
}

// path parses Name(::Name)* without a trailing string.
func (p *cedarParser) path() (string, error) {
	t := p.next()
	if t.kind != "ident" {
		return "", p.errAt(t, "expected a type name, found %s", cedarDescribe(t))
	}
	parts := []string{t.text}
	for p.isOp("::") && p.toks[p.pos+1].kind == "ident" {
		p.next()
		parts = append(parts, p.next().text)
	}
	return strings.Join(parts, "::"), nil
}

// entity parses Type(::Type)*::"id".
func (p *cedarParser) entity() (*cedarEntity, error) {
	start := p.peek()
	typ, err := p.path()
	if err != nil {
		return nil, err
	}
	if err := p.expectOp("::"); err != nil {
		return nil, err
	}
	id := p.next()
	if id.kind != "string" {
		return nil, p.errAt(id, "expected an entity id string after %s::", typ)
	}
	if err := p.str(id); err != nil {
		return nil, err
	}
	if cedarReserved[typ] {
		return nil, p.errAt(start, "%q is not a valid entity type", typ)
	}
	return &cedarEntity{Type: typ, ID: id.text}, nil
}

var cedarReserved = map[string]bool{"true": true, "false": true, "if": true, "then": true, "else": true, "in": true, "like": true, "has": true, "is": true}

func (p *cedarParser) enter() error {
	p.depth++
	if p.depth > vpMaxParseDepth {
		return p.errAt(p.peek(), "expression nesting exceeds %d levels", vpMaxParseDepth)
	}
	return nil
}

func (p *cedarParser) expr() (cedarExpr, error) {
	if err := p.enter(); err != nil {
		return nil, err
	}
	defer func() { p.depth-- }()
	if p.isIdent("if") {
		p.next()
		c, err := p.expr()
		if err != nil {
			return nil, err
		}
		if err := p.expectIdent("then"); err != nil {
			return nil, err
		}
		t, err := p.expr()
		if err != nil {
			return nil, err
		}
		if err := p.expectIdent("else"); err != nil {
			return nil, err
		}
		e, err := p.expr()
		if err != nil {
			return nil, err
		}
		return cedarIf{c, t, e}, nil
	}
	return p.or()
}

func (p *cedarParser) or() (cedarExpr, error) {
	l, err := p.and()
	for err == nil && p.isOp("||") {
		p.next()
		var r cedarExpr
		if r, err = p.and(); err == nil {
			l = cedarBinary{"||", l, r}
		}
	}
	return l, err
}

func (p *cedarParser) and() (cedarExpr, error) {
	l, err := p.relation()
	for err == nil && p.isOp("&&") {
		p.next()
		var r cedarExpr
		if r, err = p.relation(); err == nil {
			l = cedarBinary{"&&", l, r}
		}
	}
	return l, err
}

func (p *cedarParser) relation() (cedarExpr, error) {
	l, err := p.add()
	if err != nil {
		return nil, err
	}
	switch t := p.peek(); {
	case t.kind == "op" && (t.text == "==" || t.text == "!=" || t.text == "<" || t.text == "<=" || t.text == ">" || t.text == ">="):
		p.next()
		r, err := p.add()
		return cedarBinary{t.text, l, r}, err
	case t.kind == "ident" && t.text == "in":
		p.next()
		r, err := p.add()
		return cedarBinary{"in", l, r}, err
	case t.kind == "ident" && t.text == "has":
		p.next()
		name := p.next()
		if name.kind != "ident" && name.kind != "string" {
			return nil, p.errAt(name, "expected an attribute name after has")
		}
		if err := p.str(name); err != nil {
			return nil, err
		}
		return cedarHas{l, name.text}, nil
	case t.kind == "ident" && t.text == "like":
		p.next()
		pat := p.next()
		if pat.kind != "string" {
			return nil, p.errAt(pat, "expected a string pattern after like")
		}
		parts, err := cedarPattern(pat.raw)
		if err != nil {
			return nil, p.errAt(pat, "%s", err.Error())
		}
		return cedarLike{l, parts}, nil
	case t.kind == "ident" && t.text == "is":
		p.next()
		typ, err := p.path()
		if err != nil {
			return nil, err
		}
		node := cedarIs{X: l, Type: typ}
		if p.isIdent("in") {
			p.next()
			if node.In, err = p.add(); err != nil {
				return nil, err
			}
		}
		return node, nil
	}
	return l, nil
}

// cedarPattern splits a like pattern's raw (still escaped) text into literal
// runs and wildcards: * is a wildcard, \* a literal star.
func cedarPattern(raw string) ([]cedarPatternPart, error) {
	var parts []cedarPatternPart
	var lit strings.Builder
	for i := 0; i < len(raw); i++ {
		switch {
		case raw[i] == '\\':
			if i+1 < len(raw) && raw[i+1] == '*' {
				lit.WriteByte('*')
				i++
				continue
			}
			n := 2
			if i+1 < len(raw) && raw[i+1] == 'u' {
				end := strings.IndexByte(raw[i:], '}')
				if end < 0 {
					return nil, fmt.Errorf(`bad \u escape`)
				}
				n = end + 1
			}
			if i+n > len(raw) {
				return nil, fmt.Errorf("bad escape at end of pattern")
			}
			s, err := cedarUnescape(raw[i:i+n], false)
			if err != nil {
				return nil, err
			}
			lit.WriteString(s)
			i += n - 1
		case raw[i] == '*':
			if lit.Len() > 0 {
				parts = append(parts, cedarPatternPart{Lit: lit.String()})
				lit.Reset()
			}
			parts = append(parts, cedarPatternPart{Wildcard: true})
		default:
			lit.WriteByte(raw[i])
		}
	}
	if lit.Len() > 0 {
		parts = append(parts, cedarPatternPart{Lit: lit.String()})
	}
	return parts, nil
}

func (p *cedarParser) add() (cedarExpr, error) {
	l, err := p.mult()
	for err == nil && (p.isOp("+") || p.isOp("-")) {
		op := p.next().text
		var r cedarExpr
		if r, err = p.mult(); err == nil {
			l = cedarBinary{op, l, r}
		}
	}
	return l, err
}

func (p *cedarParser) mult() (cedarExpr, error) {
	l, err := p.unary()
	for err == nil && p.isOp("*") {
		p.next()
		var r cedarExpr
		if r, err = p.unary(); err == nil {
			l = cedarBinary{"*", l, r}
		}
	}
	return l, err
}

func (p *cedarParser) unary() (cedarExpr, error) {
	if p.isOp("!") || p.isOp("-") {
		op := p.next().text
		if err := p.enter(); err != nil {
			return nil, err
		}
		defer func() { p.depth-- }()
		if op == "-" && p.peek().kind == "int" { // a negative literal
			x, err := p.member()
			if lit, ok := x.(cedarLit); ok && err == nil {
				if n, ok := lit.V.(int64); ok {
					return cedarLit{-n}, nil
				}
			}
			return cedarUnary{op, x}, err
		}
		x, err := p.unary()
		return cedarUnary{op, x}, err
	}
	return p.member()
}

func (p *cedarParser) member() (cedarExpr, error) {
	x, err := p.primary()
	for err == nil {
		switch {
		case p.isOp("."):
			p.next()
			name := p.next()
			if name.kind != "ident" {
				return nil, p.errAt(name, "expected an attribute or method name after '.'")
			}
			if !p.isOp("(") {
				x = cedarAttr{x, name.text}
				continue
			}
			switch name.text {
			case "contains", "containsAll", "containsAny", "isEmpty":
			default:
				return nil, p.errAt(name, "unsupported method %s", name.text)
			}
			p.next()
			var args []cedarExpr
			if args, err = p.args(")"); err != nil {
				return nil, err
			}
			want := 1
			if name.text == "isEmpty" {
				want = 0
			}
			if len(args) != want {
				return nil, p.errAt(name, "%s takes %d argument(s), got %d", name.text, want, len(args))
			}
			x = cedarCall{x, name.text, args}
		case p.isOp("["):
			p.next()
			key := p.next()
			if key.kind != "string" {
				return nil, p.errAt(key, "expected a string key in [ ]")
			}
			if err = p.str(key); err != nil {
				return nil, err
			}
			if err = p.expectOp("]"); err != nil {
				return nil, err
			}
			x = cedarAttr{x, key.text}
		default:
			return x, nil
		}
	}
	return x, err
}

func (p *cedarParser) args(closer string) ([]cedarExpr, error) {
	var out []cedarExpr
	for !p.isOp(closer) {
		e, err := p.expr()
		if err != nil {
			return nil, err
		}
		out = append(out, e)
		if !p.isOp(",") {
			break
		}
		p.next()
	}
	return out, p.expectOp(closer)
}

func (p *cedarParser) primary() (cedarExpr, error) {
	t := p.peek()
	switch t.kind {
	case "int":
		p.next()
		n, err := strconv.ParseInt(t.text, 10, 64)
		if err != nil {
			return nil, p.errAt(t, "integer literal %s is out of range", t.text)
		}
		return cedarLit{n}, nil
	case "string":
		p.next()
		if err := p.str(t); err != nil {
			return nil, err
		}
		return cedarLit{t.text}, nil
	case "op":
		switch t.text {
		case "(":
			p.next()
			e, err := p.expr()
			if err != nil {
				return nil, err
			}
			return e, p.expectOp(")")
		case "[":
			p.next()
			elems, err := p.args("]")
			return cedarSetLit{elems}, err
		case "{":
			p.next()
			rec := cedarRecLit{}
			for !p.isOp("}") {
				k := p.next()
				if k.kind != "ident" && k.kind != "string" {
					return nil, p.errAt(k, "expected a record key")
				}
				if err := p.str(k); err != nil {
					return nil, err
				}
				if err := p.expectOp(":"); err != nil {
					return nil, err
				}
				v, err := p.expr()
				if err != nil {
					return nil, err
				}
				rec.Keys, rec.Vals = append(rec.Keys, k.text), append(rec.Vals, v)
				if !p.isOp(",") {
					break
				}
				p.next()
			}
			return rec, p.expectOp("}")
		}
	case "ident":
		switch t.text {
		case "true", "false":
			p.next()
			return cedarLit{t.text == "true"}, nil
		case "principal", "action", "resource", "context":
			if !(p.toks[p.pos+1].kind == "op" && p.toks[p.pos+1].text == "::") {
				p.next()
				return cedarVar{t.text}, nil
			}
		}
		if p.toks[p.pos+1].kind == "op" && p.toks[p.pos+1].text == "(" {
			return nil, p.errAt(t, "unsupported function %s", t.text)
		}
		e, err := p.entity()
		if err != nil {
			return nil, err
		}
		return cedarLit{*e}, nil
	}
	return nil, p.errAt(t, "unexpected %s", cedarDescribe(t))
}
