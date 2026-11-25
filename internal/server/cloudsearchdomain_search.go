package server

import (
	"encoding/base64"
	"encoding/json"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/google/uuid"
)

// Search and suggest over a domain's documents: in-memory term matching with
// the simple and structured query parsers (lucene and dismax queries are read
// as simple ones), filter queries, sorting, paging by start or cursor, return
// fields, facets, stats, highlights and arithmetic expressions.

func csdSearchErr(format string, args ...any) *csErr { return csValidation(format, args...) }

// csdTokens splits text into lower-cased words.
func csdTokens(s string) []string {
	return strings.FieldsFunc(strings.ToLower(s), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
}

// ---- query tree ----

type csdNode interface {
	// eval returns whether doc matches and a relevance score.
	eval(c *csdCtx, doc *csDoc) (bool, float64)
}

type csdCtx struct {
	sc       *csdSchema
	defaults []string // fields searched when a term names none
}

type csdMatchAll struct{}

func (csdMatchAll) eval(*csdCtx, *csDoc) (bool, float64) { return true, 1 }

type csdBool struct {
	op   string // and, or, not
	kids []csdNode
}

func (n *csdBool) eval(c *csdCtx, doc *csDoc) (bool, float64) {
	score := 0.0
	switch n.op {
	case "not":
		for _, k := range n.kids {
			if ok, _ := k.eval(c, doc); ok {
				return false, 0
			}
		}
		return true, 0
	case "or":
		any := false
		for _, k := range n.kids {
			if ok, s := k.eval(c, doc); ok {
				any, score = true, score+s
			}
		}
		return any, score
	}
	for _, k := range n.kids {
		ok, s := k.eval(c, doc)
		if !ok {
			return false, 0
		}
		score += s
	}
	return true, score
}

// csdTerm matches value in field (or the default fields): word or phrase
// matching for text, prefix matching when prefix is set, exact or numeric
// comparison otherwise, and ranges when lo/hi are set.
type csdTerm struct {
	field, value    string
	prefix, isRange bool
	lo, hi          string
	loIncl, hiIncl  bool
	boost           float64
}

func (t *csdTerm) eval(c *csdCtx, doc *csDoc) (bool, float64) {
	fields := c.defaults
	if t.field != "" {
		fields = []string{t.field}
	}
	best := 0.0
	for _, name := range fields {
		f := c.sc.field(name)
		if f == nil {
			continue
		}
		for _, v := range doc.fields[name] {
			if s := t.matchValue(f, v); s > best {
				best = s
			}
		}
	}
	boost := t.boost
	if boost == 0 {
		boost = 1
	}
	return best > 0, best * boost
}

func (t *csdTerm) matchValue(f *csdField, v string) float64 {
	if t.isRange {
		if csdInRange(v, t.lo, t.hi, t.loIncl, t.hiIncl) {
			return 1
		}
		return 0
	}
	if f.base() != "text" {
		if t.prefix {
			if strings.HasPrefix(v, t.value) {
				return 1
			}
			return 0
		}
		if csdEqual(v, t.value) {
			return 1
		}
		return 0
	}
	words, want := csdTokens(v), csdTokens(t.value)
	if len(want) == 0 {
		return 0
	}
	if t.prefix {
		count := 0.0
		for _, w := range words {
			if strings.HasPrefix(w, want[len(want)-1]) {
				count++
			}
		}
		return count
	}
	// Count occurrences of the word sequence (a phrase when it has several).
	count := 0.0
	for i := 0; i+len(want) <= len(words); i++ {
		match := true
		for j, w := range want {
			if words[i+j] != w {
				match = false
				break
			}
		}
		if match {
			count++
		}
	}
	return count
}

func csdEqual(a, b string) bool {
	if a == b {
		return true
	}
	x, err1 := strconv.ParseFloat(a, 64)
	y, err2 := strconv.ParseFloat(b, 64)
	return err1 == nil && err2 == nil && x == y
}

// csdCompare orders numerically when both parse, else as strings (dates are
// stored as RFC 3339 UTC, which sorts as text).
func csdCompare(a, b string) int {
	x, err1 := strconv.ParseFloat(a, 64)
	y, err2 := strconv.ParseFloat(b, 64)
	if err1 == nil && err2 == nil {
		switch {
		case x < y:
			return -1
		case x > y:
			return 1
		}
		return 0
	}
	return strings.Compare(a, b)
}

func csdInRange(v, lo, hi string, loIncl, hiIncl bool) bool {
	if lo != "" {
		c := csdCompare(v, lo)
		if c < 0 || (c == 0 && !loIncl) {
			return false
		}
	}
	if hi != "" {
		c := csdCompare(v, hi)
		if c > 0 || (c == 0 && !hiIncl) {
			return false
		}
	}
	return true
}

// ---- simple query parser ----

// csdParseSimple reads the simple syntax: words, "phrases", -exclusions,
// +required terms, trailing * prefixes and | between terms for OR.
func csdParseSimple(q, defaultOp string) csdNode {
	var must, should, not []csdNode
	or := defaultOp == "or"
	for _, tok := range csdSimpleTokens(q) {
		if tok == "|" {
			or = true
			continue
		}
		neg, req := strings.HasPrefix(tok, "-"), strings.HasPrefix(tok, "+")
		tok = strings.TrimLeft(tok, "+-")
		tok = strings.Trim(tok, `"`)
		if tok == "" {
			continue
		}
		term := &csdTerm{value: tok}
		if strings.HasSuffix(tok, "*") {
			term.value, term.prefix = strings.TrimSuffix(tok, "*"), true
		}
		switch {
		case neg:
			not = append(not, term)
		case req:
			must = append(must, term)
		default:
			should = append(should, term)
		}
	}
	root := &csdBool{op: "and"}
	if or && len(should) > 0 {
		root.kids = append(root.kids, &csdBool{op: "or", kids: should})
	} else {
		root.kids = append(root.kids, should...)
	}
	root.kids = append(root.kids, must...)
	if len(not) > 0 {
		root.kids = append(root.kids, &csdBool{op: "not", kids: not})
	}
	if len(root.kids) == 0 {
		return &csdBool{op: "or"} // matches nothing
	}
	return root
}

func csdSimpleTokens(q string) []string {
	var out []string
	var cur strings.Builder
	quoted := false
	for _, r := range q {
		switch {
		case r == '"':
			quoted = !quoted
			cur.WriteRune(r)
		case unicode.IsSpace(r) && !quoted:
			if cur.Len() > 0 {
				out = append(out, cur.String())
				cur.Reset()
			}
		default:
			cur.WriteRune(r)
		}
	}
	if cur.Len() > 0 {
		out = append(out, cur.String())
	}
	return out
}

// ---- structured query parser ----

type csdLexer struct {
	toks []string
	pos  int
}

const (
	csdMaxDepth        = 32
	csdMaxQueryBytes   = 8 << 10
	csdMaxSuggestQuery = 1 << 10
)

func csdLex(q string) ([]string, *csErr) {
	var toks []string
	rs := []rune(q)
	for i := 0; i < len(rs); {
		r := rs[i]
		switch {
		case unicode.IsSpace(r):
			i++
		case r == '(' || r == ')' || r == ':' || r == '=':
			toks = append(toks, string(r))
			i++
		case r == '\'':
			var b strings.Builder
			b.WriteRune('\'')
			i++
			for i < len(rs) && rs[i] != '\'' {
				if rs[i] == '\\' && i+1 < len(rs) {
					i++
				}
				b.WriteRune(rs[i])
				i++
			}
			if i >= len(rs) {
				return nil, csdSearchErr("Syntax error in query: unterminated string")
			}
			i++
			toks = append(toks, b.String())
		case r == '[' || r == '{':
			j := i
			for j < len(rs) && rs[j] != ']' && rs[j] != '}' {
				j++
			}
			if j >= len(rs) {
				return nil, csdSearchErr("Syntax error in query: unterminated range")
			}
			toks = append(toks, string(rs[i:j+1]))
			i = j + 1
		default:
			j := i
			for j < len(rs) && !unicode.IsSpace(rs[j]) && !strings.ContainsRune("():='[{", rs[j]) {
				j++
			}
			toks = append(toks, string(rs[i:j]))
			i = j
		}
	}
	return toks, nil
}

func (l *csdLexer) peek() string {
	if l.pos < len(l.toks) {
		return l.toks[l.pos]
	}
	return ""
}

func (l *csdLexer) next() string {
	t := l.peek()
	l.pos++
	return t
}

func csdParseStructured(q string) (csdNode, *csErr) {
	toks, err := csdLex(q)
	if err != nil {
		return nil, err
	}
	l := &csdLexer{toks: toks}
	n, err := l.expr(0)
	if err != nil {
		return nil, err
	}
	if l.pos != len(l.toks) {
		return nil, csdSearchErr("Syntax error in query: unexpected %q", l.peek())
	}
	return n, nil
}

func (l *csdLexer) expr(depth int) (csdNode, *csErr) {
	if depth > csdMaxDepth {
		return nil, csdSearchErr("Syntax error in query: nested too deeply")
	}
	tok := l.next()
	switch {
	case tok == "":
		return nil, csdSearchErr("Syntax error in query: unexpected end")
	case tok == "(":
		return l.compound(depth)
	case tok == "matchall":
		return csdMatchAll{}, nil
	}
	if l.peek() == ":" { // field:value
		l.next()
		return csdValueTerm(tok, l.next())
	}
	return csdValueTerm("", tok)
}

func (l *csdLexer) compound(depth int) (csdNode, *csErr) {
	op := l.next()
	opts := map[string]string{}
	for l.pos+1 < len(l.toks) && l.toks[l.pos+1] == "=" {
		k := l.next()
		l.next()
		opts[k] = strings.TrimPrefix(l.next(), "'")
	}
	var kids []csdNode
	for l.peek() != ")" {
		if l.peek() == "" {
			return nil, csdSearchErr("Syntax error in query: missing )")
		}
		if op == "and" || op == "or" || op == "not" {
			k, err := l.expr(depth + 1)
			if err != nil {
				return nil, err
			}
			kids = append(kids, k)
			continue
		}
		k, err := csdValueTerm(opts["field"], l.next())
		if err != nil {
			return nil, err
		}
		if t, ok := k.(*csdTerm); ok {
			t.prefix = t.prefix || op == "prefix"
			t.boost, _ = strconv.ParseFloat(opts["boost"], 64)
		}
		kids = append(kids, k)
	}
	l.next()
	switch op {
	case "and", "or", "not":
		return &csdBool{op: op, kids: kids}, nil
	case "term", "phrase", "prefix", "range", "near":
		return &csdBool{op: "and", kids: kids}, nil
	}
	return nil, csdSearchErr("Syntax error in query: unknown operator %q", op)
}

// csdValueTerm builds a term from a 'string', a number or a [lo,hi] range.
func csdValueTerm(field, tok string) (csdNode, *csErr) {
	if tok == "" || tok == ")" || tok == "(" {
		return nil, csdSearchErr("Syntax error in query: missing value")
	}
	t := &csdTerm{field: field}
	switch tok[0] {
	case '\'':
		t.value = tok[1:]
	case '[', '{':
		parts := strings.SplitN(tok[1:len(tok)-1], ",", 2)
		if len(parts) != 2 {
			return nil, csdSearchErr("Syntax error in query: bad range %s", tok)
		}
		t.isRange, t.loIncl, t.hiIncl = true, tok[0] == '[', tok[len(tok)-1] == ']'
		t.lo, t.hi = strings.Trim(strings.TrimSpace(parts[0]), "'"), strings.Trim(strings.TrimSpace(parts[1]), "'")
	default:
		t.value = tok
	}
	return t, nil
}

// ---- search ----

type csdHit struct {
	id    string
	doc   *csDoc
	score float64
	exprs map[string]float64
}

func (st *csStore) search(domain string, p url.Values) (map[string]any, *csErr) {
	q := p.Get("q")
	if q == "" || len(q) > csdMaxQueryBytes {
		return nil, csdSearchErr("q is required and can be at most %d bytes", csdMaxQueryBytes)
	}
	size, err := csdIntParam(p, "size", csdDefaultSize, csdMaxSearchSize)
	if err != nil {
		return nil, err
	}
	start, err := csdIntParam(p, "start", 0, math.MaxInt32)
	if err != nil {
		return nil, err
	}
	cursor := p.Get("cursor")
	if cursor != "" {
		if p.Get("start") != "" {
			return nil, csdSearchErr("cursor and start can't be used together")
		}
		if start, err = csdDecodeCursor(cursor); err != nil {
			return nil, err
		}
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	d, err := st.csdDomain(domain)
	if err != nil {
		return nil, err
	}
	ctx, root, err := csdBuildQuery(d.schema(), q, p.Get("q.parser"), p.Get("q.options"))
	if err != nil {
		return nil, err
	}
	var filter csdNode
	if fq := p.Get("fq"); fq != "" {
		if filter, err = csdParseStructured(fq); err != nil {
			return nil, err
		}
	}
	exprs, err := csdExprParam(d, p.Get("expr"))
	if err != nil {
		return nil, err
	}
	var hits []*csdHit
	for id, doc := range d.docs {
		ok, score := root.eval(ctx, doc)
		if !ok {
			continue
		}
		if filter != nil {
			if fok, _ := filter.eval(ctx, doc); !fok {
				continue
			}
		}
		h := &csdHit{id: id, doc: doc, score: score, exprs: map[string]float64{}}
		for name, e := range exprs {
			h.exprs[name] = e.eval(doc, score)
		}
		hits = append(hits, h)
	}
	if err := csdSort(ctx.sc, hits, p.Get("sort")); err != nil {
		return nil, err
	}
	out := map[string]any{"status": csdStatus()}
	facets, err := csdFacets(ctx.sc, hits, p.Get("facet"))
	if err != nil {
		return nil, err
	}
	if facets != nil {
		out["facets"] = facets
	}
	stats, err := csdStats(ctx.sc, hits, p.Get("stats"))
	if err != nil {
		return nil, err
	}
	if stats != nil {
		out["stats"] = stats
	}
	highlights, err := csdHighlightParam(ctx.sc, p.Get("highlight"))
	if err != nil {
		return nil, err
	}
	page := []any{}
	end := min(len(hits), start+size)
	for i := start; i < end; i++ {
		page = append(page, csdHitOut(ctx, hits[i], p.Get("return"), highlights, q))
	}
	hitsOut := map[string]any{"found": len(hits), "start": start, "hit": page}
	if cursor != "" && end < len(hits) {
		hitsOut["cursor"] = csdEncodeCursor(end)
	}
	out["hits"] = hitsOut
	return out, nil
}

func csdStatus() map[string]any {
	return map[string]any{"timems": 1, "rid": uuid.NewString()}
}

func csdIntParam(p url.Values, key string, def, max int) (int, *csErr) {
	raw := p.Get(key)
	if raw == "" {
		return def, nil
	}
	n, err := strconv.ParseInt(raw, 10, 32)
	if err != nil || n < 0 || int(n) > max {
		return 0, csdSearchErr("%s must be an integer from 0 to %d", key, max)
	}
	return int(n), nil
}

func csdEncodeCursor(offset int) string {
	return base64.RawURLEncoding.EncodeToString([]byte("o:" + strconv.Itoa(offset)))
}

func csdDecodeCursor(c string) (int, *csErr) {
	if c == "initial" {
		return 0, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(c)
	if err == nil && strings.HasPrefix(string(raw), "o:") {
		if n, perr := strconv.ParseInt(string(raw[2:]), 10, 32); perr == nil && n >= 0 {
			return int(n), nil
		}
	}
	return 0, csdSearchErr("invalid cursor")
}

// csdBuildQuery parses q with the chosen parser and works out the default
// fields: q.options fields, else every text field, else every field.
func csdBuildQuery(sc *csdSchema, q, parser, rawOpts string) (*csdCtx, csdNode, *csErr) {
	var opts struct {
		DefaultOperator string   `json:"defaultOperator"`
		Fields          []string `json:"fields"`
	}
	if rawOpts != "" && json.Unmarshal([]byte(rawOpts), &opts) != nil {
		return nil, nil, csdSearchErr("q.options must be a JSON object")
	}
	ctx := &csdCtx{sc: sc}
	for _, f := range opts.Fields {
		ctx.defaults = append(ctx.defaults, strings.SplitN(f, "^", 2)[0])
	}
	if len(ctx.defaults) == 0 {
		var all []string
		for name, f := range sc.exact {
			all = append(all, name)
			if f.base() == "text" {
				ctx.defaults = append(ctx.defaults, name)
			}
		}
		if len(ctx.defaults) == 0 {
			ctx.defaults = all
		}
		sort.Strings(ctx.defaults)
	}
	switch parser {
	case "", "simple", "lucene", "dismax":
		return ctx, csdParseSimple(q, strings.ToLower(opts.DefaultOperator)), nil
	case "structured":
		n, err := csdParseStructured(q)
		return ctx, n, err
	}
	return nil, nil, csdSearchErr("unknown q.parser %s", parser)
}

// csdSortValue is the value a hit sorts on for key: _score, an expression or
// a field's first value.
func csdSortValue(h *csdHit, key string) (string, bool) {
	if key == "_score" {
		return strconv.FormatFloat(h.score, 'f', -1, 64), true
	}
	if v, ok := h.exprs[key]; ok {
		return strconv.FormatFloat(v, 'f', -1, 64), true
	}
	if vals := h.doc.fields[key]; len(vals) > 0 {
		return vals[0], true
	}
	return "", false
}

func csdSort(sc *csdSchema, hits []*csdHit, spec string) *csErr {
	type key struct {
		name string
		desc bool
	}
	keys := []key{{"_score", true}}
	if spec != "" {
		keys = nil
		for _, part := range strings.Split(spec, ",") {
			f := strings.Fields(part)
			if len(f) != 2 || (f[1] != "asc" && f[1] != "desc") {
				return csdSearchErr("sort must be a list of '<field> asc|desc'")
			}
			if f[0] != "_score" && sc.field(f[0]) == nil && (len(hits) == 0 || !csdHasExpr(hits[0], f[0])) {
				return csdSearchErr("unknown sort field %s", f[0])
			}
			keys = append(keys, key{f[0], f[1] == "desc"})
		}
	}
	sort.SliceStable(hits, func(i, j int) bool {
		for _, k := range keys {
			a, aok := csdSortValue(hits[i], k.name)
			b, bok := csdSortValue(hits[j], k.name)
			if aok != bok { // missing values sort last
				return aok
			}
			if c := csdCompare(a, b); c != 0 {
				return (c < 0) != k.desc
			}
		}
		return hits[i].id < hits[j].id
	})
	return nil
}

func csdHasExpr(h *csdHit, name string) bool {
	_, ok := h.exprs[name]
	return ok
}

func csdHitOut(ctx *csdCtx, h *csdHit, ret string, highlights map[string]csdHighlight, q string) map[string]any {
	fields := map[string]any{}
	want := map[string]bool{}
	all := ret == "" || ret == "_all_fields"
	for _, name := range strings.Split(ret, ",") {
		want[strings.TrimSpace(name)] = true
	}
	if ret != "_no_fields" {
		for name, vals := range h.doc.fields {
			f := ctx.sc.field(name)
			if f != nil && ((all && f.enabled("ReturnEnabled")) || want[name]) {
				fields[name] = append([]string(nil), vals...)
			}
		}
	}
	if want["_score"] {
		fields["_score"] = []string{strconv.FormatFloat(h.score, 'f', -1, 64)}
	}
	out := map[string]any{"id": h.id, "fields": fields}
	exprs := map[string]any{}
	for name, v := range h.exprs {
		if want[name] {
			exprs[name] = strconv.FormatFloat(v, 'f', -1, 64)
		}
	}
	if len(exprs) > 0 {
		out["exprs"] = exprs
	}
	if len(highlights) > 0 {
		hl := map[string]any{}
		words := map[string]bool{}
		for _, w := range csdTokens(q) {
			words[w] = true
		}
		for name, opt := range highlights {
			if vals := h.doc.fields[name]; len(vals) > 0 {
				hl[name] = csdHighlightText(vals[0], words, opt)
			}
		}
		out["highlights"] = hl
	}
	return out
}

// ---- facets ----

func csdFacets(sc *csdSchema, hits []*csdHit, raw string) (map[string]any, *csErr) {
	if raw == "" {
		return nil, nil
	}
	var spec map[string]struct {
		Sort    string   `json:"sort"`
		Size    *int     `json:"size"`
		Buckets []string `json:"buckets"`
	}
	if json.Unmarshal([]byte(raw), &spec) != nil {
		return nil, csdSearchErr("facet must be a JSON object of field options")
	}
	out := map[string]any{}
	for name, opt := range spec {
		f := sc.field(name)
		if f == nil || f.base() == "text" || !f.enabled("FacetEnabled") {
			return nil, csdSearchErr("%s is not a facet-enabled field", name)
		}
		counts := map[string]int{}
		for _, h := range hits {
			for _, v := range h.doc.fields[name] {
				counts[v]++
			}
		}
		var values []string
		if len(opt.Buckets) > 0 {
			values = opt.Buckets
		} else {
			for v := range counts {
				values = append(values, v)
			}
		}
		sort.Slice(values, func(i, j int) bool {
			if opt.Sort != "bucket" && counts[values[i]] != counts[values[j]] {
				return counts[values[i]] > counts[values[j]]
			}
			return csdCompare(values[i], values[j]) < 0
		})
		limit := csdDefaultSize
		if opt.Size != nil && *opt.Size >= 0 {
			limit = *opt.Size
		}
		buckets := []any{}
		for _, v := range values {
			if len(buckets) >= limit {
				break
			}
			buckets = append(buckets, map[string]any{"value": v, "count": counts[v]})
		}
		out[name] = map[string]any{"buckets": buckets}
	}
	return out, nil
}

// ---- stats ----

func csdStats(sc *csdSchema, hits []*csdHit, raw string) (map[string]any, *csErr) {
	if raw == "" {
		return nil, nil
	}
	var spec map[string]json.RawMessage
	if json.Unmarshal([]byte(raw), &spec) != nil {
		return nil, csdSearchErr("stats must be a JSON object of field options")
	}
	out := map[string]any{}
	for name := range spec {
		f := sc.field(name)
		if f == nil || (f.base() != "int" && f.base() != "double") {
			return nil, csdSearchErr("stats need an int or double field, not %s", name)
		}
		count, missing := 0, 0
		sum, sumSq, lo, hi := 0.0, 0.0, math.Inf(1), math.Inf(-1)
		for _, h := range hits {
			vals := h.doc.fields[name]
			if len(vals) == 0 {
				missing++
			}
			for _, v := range vals {
				x, _ := strconv.ParseFloat(v, 64)
				count++
				sum, sumSq = sum+x, sumSq+x*x
				lo, hi = math.Min(lo, x), math.Max(hi, x)
			}
		}
		st := map[string]any{"count": count, "missing": missing, "sum": sum, "sumOfSquares": sumSq}
		if count > 0 {
			mean := sum / float64(count)
			st["min"], st["max"] = strconv.FormatFloat(lo, 'f', -1, 64), strconv.FormatFloat(hi, 'f', -1, 64)
			st["mean"] = strconv.FormatFloat(mean, 'f', -1, 64)
			st["stddev"] = math.Sqrt(math.Max(0, sumSq/float64(count)-mean*mean))
		}
		out[name] = st
	}
	return out, nil
}

// ---- highlights ----

type csdHighlight struct {
	Pre  string `json:"pre_tag"`
	Post string `json:"post_tag"`
}

func csdHighlightParam(sc *csdSchema, raw string) (map[string]csdHighlight, *csErr) {
	if raw == "" {
		return nil, nil
	}
	var spec map[string]csdHighlight
	if json.Unmarshal([]byte(raw), &spec) != nil {
		return nil, csdSearchErr("highlight must be a JSON object of field options")
	}
	for name, opt := range spec {
		if f := sc.field(name); f == nil || f.base() != "text" {
			return nil, csdSearchErr("%s is not a text field", name)
		}
		if opt.Pre == "" && opt.Post == "" {
			opt.Pre, opt.Post = "<em>", "</em>"
		}
		spec[name] = opt
	}
	return spec, nil
}

// csdHighlightText wraps the words of text that appear in the query.
func csdHighlightText(text string, words map[string]bool, opt csdHighlight) string {
	var b strings.Builder
	rs := []rune(text)
	for i := 0; i < len(rs); {
		if !unicode.IsLetter(rs[i]) && !unicode.IsDigit(rs[i]) {
			b.WriteRune(rs[i])
			i++
			continue
		}
		j := i
		for j < len(rs) && (unicode.IsLetter(rs[j]) || unicode.IsDigit(rs[j])) {
			j++
		}
		w := string(rs[i:j])
		if words[strings.ToLower(w)] {
			b.WriteString(opt.Pre + w + opt.Post)
		} else {
			b.WriteString(w)
		}
		i = j
	}
	return b.String()
}

// ---- suggest ----

func (st *csStore) suggest(domain string, p url.Values) (map[string]any, *csErr) {
	q, name := p.Get("q"), p.Get("suggester")
	if q == "" || name == "" || len(q) > csdMaxSuggestQuery {
		return nil, csdSearchErr("q (at most %d bytes) and suggester are required", csdMaxSuggestQuery)
	}
	size, err := csdIntParam(p, "size", csdDefaultSize, csdMaxSuggestSize)
	if err != nil {
		return nil, err
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	d, err := st.csdDomain(domain)
	if err != nil {
		return nil, err
	}
	o := d.suggesters[name]
	if o == nil || o.deployed == nil {
		return nil, &csErr{http.StatusBadRequest, "", "No suggester named " + name + " is deployed; define it and run IndexDocuments."}
	}
	sug, _ := o.deployed.(map[string]any)
	opts := csMap(sug, "DocumentSuggesterOptions")
	source, sortField := csStr(opts, "SourceField"), strings.TrimSpace(csStr(opts, "SortExpression"))
	maxDist := map[string]int{"low": 1, "high": 2}[csStr(opts, "FuzzyMatching")]
	want := []rune(strings.ToLower(q))
	type match struct {
		text, id string
		score    int64
	}
	var matches []match
	for id, doc := range d.docs {
		for _, v := range doc.fields[source] {
			lv := []rune(strings.ToLower(v))
			if len(lv) < len(want) || csdLevenshtein(lv[:len(want)], want) > maxDist {
				continue
			}
			var score int64
			if vals := doc.fields[sortField]; sortField != "" && len(vals) > 0 {
				f, _ := strconv.ParseFloat(vals[0], 64)
				score = int64(f)
			}
			matches = append(matches, match{v, id, score})
			break
		}
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].score != matches[j].score {
			return matches[i].score > matches[j].score
		}
		if matches[i].text != matches[j].text {
			return matches[i].text < matches[j].text
		}
		return matches[i].id < matches[j].id
	})
	list := []any{}
	for i := 0; i < len(matches) && i < size; i++ {
		list = append(list, map[string]any{"suggestion": matches[i].text, "score": matches[i].score, "id": matches[i].id})
	}
	return map[string]any{
		"status":  csdStatus(),
		"suggest": map[string]any{"query": q, "found": len(list), "suggestions": list},
	}, nil
}

func csdLevenshtein(a, b []rune) int {
	prev := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur := make([]int, len(b)+1)
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(b)]
}

// ---- expressions ----

// csdExpr is a parsed arithmetic expression over numeric fields and _score:
// numbers, field names, + - * /, unary minus and parentheses.
type csdExpr struct {
	op          byte // 0 for a leaf
	num         float64
	ident       string
	left, right *csdExpr
}

func (e *csdExpr) eval(doc *csDoc, score float64) float64 {
	switch e.op {
	case 0:
		if e.ident == "_score" {
			return score
		}
		if e.ident != "" {
			if vals := doc.fields[e.ident]; len(vals) > 0 {
				f, _ := strconv.ParseFloat(vals[0], 64)
				return f
			}
			return 0
		}
		return e.num
	case 'n':
		return -e.left.eval(doc, score)
	}
	a, b := e.left.eval(doc, score), e.right.eval(doc, score)
	switch e.op {
	case '+':
		return a + b
	case '-':
		return a - b
	case '*':
		return a * b
	}
	if b == 0 {
		return 0
	}
	return a / b
}

// csdExprParam parses the expr parameter and adds the domain's deployed
// expressions, so either can be returned or sorted on.
func csdExprParam(d *csDomain, raw string) (map[string]*csdExpr, *csErr) {
	src := map[string]string{}
	for name, o := range d.expressions {
		if e, _ := o.deployed.(map[string]any); e != nil {
			src[name] = csStr(e, "ExpressionValue")
		}
	}
	if raw != "" {
		var spec map[string]string
		if json.Unmarshal([]byte(raw), &spec) != nil {
			return nil, csdSearchErr("expr must be a JSON object of expression names and values")
		}
		for k, v := range spec {
			src[k] = v
		}
	}
	out := map[string]*csdExpr{}
	for name, v := range src {
		p := &csdExprParser{s: v}
		e, err := p.sum(0)
		if err == nil && p.skip() < len(p.s) {
			err = csdSearchErr("unexpected %q", p.s[p.i:])
		}
		if err != nil {
			return nil, csdSearchErr("expression %s: %s", name, err.msg)
		}
		out[name] = e
	}
	return out, nil
}

type csdExprParser struct {
	s string
	i int
}

func (p *csdExprParser) skip() int {
	for p.i < len(p.s) && p.s[p.i] == ' ' {
		p.i++
	}
	return p.i
}

func (p *csdExprParser) sum(depth int) (*csdExpr, *csErr) {
	left, err := p.product(depth)
	for err == nil && p.skip() < len(p.s) && (p.s[p.i] == '+' || p.s[p.i] == '-') {
		op := p.s[p.i]
		p.i++
		var right *csdExpr
		if right, err = p.product(depth); err == nil {
			left = &csdExpr{op: op, left: left, right: right}
		}
	}
	return left, err
}

func (p *csdExprParser) product(depth int) (*csdExpr, *csErr) {
	left, err := p.unary(depth)
	for err == nil && p.skip() < len(p.s) && (p.s[p.i] == '*' || p.s[p.i] == '/') {
		op := p.s[p.i]
		p.i++
		var right *csdExpr
		if right, err = p.unary(depth); err == nil {
			left = &csdExpr{op: op, left: left, right: right}
		}
	}
	return left, err
}

func (p *csdExprParser) unary(depth int) (*csdExpr, *csErr) {
	if depth > csdMaxDepth {
		return nil, csdSearchErr("nested too deeply")
	}
	if p.skip() >= len(p.s) {
		return nil, csdSearchErr("unexpected end")
	}
	switch c := p.s[p.i]; {
	case c == '-':
		p.i++
		e, err := p.unary(depth + 1)
		return &csdExpr{op: 'n', left: e}, err
	case c == '(':
		p.i++
		e, err := p.sum(depth + 1)
		if err != nil {
			return nil, err
		}
		if p.skip() >= len(p.s) || p.s[p.i] != ')' {
			return nil, csdSearchErr("missing )")
		}
		p.i++
		return e, nil
	}
	j := p.i
	for j < len(p.s) && (p.s[j] == '_' || p.s[j] == '.' || unicode.IsLetter(rune(p.s[j])) || unicode.IsDigit(rune(p.s[j]))) {
		j++
	}
	tok := p.s[p.i:j]
	p.i = j
	if tok == "" {
		return nil, csdSearchErr("unexpected %q", p.s[p.i:])
	}
	if f, err := strconv.ParseFloat(tok, 64); err == nil {
		return &csdExpr{num: f}, nil
	}
	return &csdExpr{ident: tok}, nil
}
