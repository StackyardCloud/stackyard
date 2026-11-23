package server

import (
	"math"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/google/uuid"
)

// Layout model: each page is a column of lines, top to bottom, inside a 5%
// margin. A word's horizontal position is its rune offset in the line, so the
// geometry is stable and consistent between LINE, WORD, CELL and KEY blocks.

const (
	txLineConfidence = 99.0
	txKVConfidence   = 95.0
	txMaxQueries     = 100
)

type txBox struct{ left, top, width, height float64 }

func (b txBox) union(o txBox) txBox {
	if b.width == 0 && b.height == 0 {
		return o
	}
	l, t := math.Min(b.left, o.left), math.Min(b.top, o.top)
	r, bt := math.Max(b.left+b.width, o.left+o.width), math.Max(b.top+b.height, o.top+o.height)
	return txBox{l, t, r - l, bt - t}
}

func txRound(f float64) float64 { return math.Round(f*1e6) / 1e6 }

func (b txBox) geometry() map[string]any {
	l, t, w, h := txRound(b.left), txRound(b.top), txRound(b.width), txRound(b.height)
	pt := func(x, y float64) map[string]any { return map[string]any{"X": txRound(x), "Y": txRound(y)} }
	return map[string]any{
		"BoundingBox":   map[string]any{"Left": l, "Top": t, "Width": w, "Height": h},
		"Polygon":       []any{pt(l, t), pt(l+w, t), pt(l+w, t+h), pt(l, t+h)},
		"RotationAngle": 0.0,
	}
}

type txKV struct {
	key, value         string
	keyEnd, valueStart int // rune offsets: key is [0,keyEnd), value is [valueStart, len)
	keyWords, valWords []int
}

type txCell struct {
	text       string
	start, end int
	words      []int
}

type txLine struct {
	page, index int // 1-based page, 0-based line index on the page
	text        string
	runes       int
	words       []txWord
	id          string
	wordIDs     []string
	kv          *txKV
	inTable     bool
	cells       []txCell // set for table rows (not separator rows)
}

type txTable struct{ lines []*txLine }

type txPageInfo struct {
	number   int
	lines    []*txLine
	tables   []*txTable
	maxRunes int
}

type txAnalysis struct {
	doc   *txDocument
	pages []*txPageInfo
}

func (a *txAnalysis) id(parts ...string) string {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(a.doc.seed+"/"+strings.Join(parts, "/"))).String()
}

func txItoa(n int) string { return strconv.Itoa(n) }

// txAnalyze lays out lines and words and finds tables and key/value lines.
func txAnalyze(doc *txDocument) *txAnalysis {
	a := &txAnalysis{doc: doc}
	for p, texts := range doc.pages {
		page := &txPageInfo{number: p + 1, maxRunes: 80}
		for i, t := range texts {
			l := &txLine{page: p + 1, index: i, text: t, runes: len([]rune(t)), words: txSplitWords(t)}
			l.id = a.id("line", txItoa(p+1), txItoa(i))
			for w := range l.words {
				l.wordIDs = append(l.wordIDs, a.id("word", txItoa(p+1), txItoa(i), txItoa(w)))
			}
			if l.runes > page.maxRunes {
				page.maxRunes = l.runes
			}
			page.lines = append(page.lines, l)
		}
		page.tables = txFindTables(page.lines)
		for _, l := range page.lines {
			if !l.inTable {
				l.kv = txParseKV(l)
			}
		}
		a.pages = append(a.pages, page)
	}
	return a
}

func (p *txPageInfo) lineBox(l *txLine) txBox {
	return p.spanBox(l, 0, l.runes)
}

func (p *txPageInfo) spanBox(l *txLine, start, end int) txBox {
	slot := 0.9 / math.Max(float64(len(p.lines)), 1)
	top := 0.05 + slot*float64(l.index)
	unit := 0.9 / float64(p.maxRunes)
	return txBox{left: 0.05 + unit*float64(start), top: top, width: unit * float64(end-start), height: math.Min(slot*0.8, 0.03)}
}

func (p *txPageInfo) wordsBox(l *txLine, idx []int) txBox {
	var b txBox
	for _, w := range idx {
		b = b.union(p.spanBox(l, l.words[w].start, l.words[w].end))
	}
	return b
}

var txSeparatorCell = regexp.MustCompile(`^:?-+:?$`)

// txFindTables groups consecutive tab- or pipe-delimited lines into tables.
func txFindTables(lines []*txLine) []*txTable {
	var tables []*txTable
	var cur *txTable
	var curDelim rune
	for _, l := range lines {
		delim := rune(0)
		switch {
		case strings.Contains(l.text, "\t"):
			delim = '\t'
		case strings.Count(l.text, "|") >= 1:
			delim = '|'
		}
		cells := txCells(l, delim)
		separator := delim == '|' && len(cells) > 0 && txAllSeparators(cells)
		if delim == 0 || (len(cells) < 2 && !separator) {
			cur = nil
			continue
		}
		if cur == nil || delim != curDelim {
			cur = &txTable{}
			curDelim = delim
			tables = append(tables, cur)
		}
		l.inTable = true
		if !separator {
			l.cells = cells
		}
		cur.lines = append(cur.lines, l)
	}
	out := tables[:0]
	for _, t := range tables {
		rows := 0
		for _, l := range t.lines {
			if l.cells != nil {
				rows++
			}
		}
		if rows > 0 {
			out = append(out, t)
		} else {
			for _, l := range t.lines {
				l.inTable = false
			}
		}
	}
	return out
}

func txAllSeparators(cells []txCell) bool {
	for _, c := range cells {
		if !txSeparatorCell.MatchString(c.text) {
			return false
		}
	}
	return true
}

// txCells splits a line on delim, trimming each cell, and assigns words.
func txCells(l *txLine, delim rune) []txCell {
	if delim == 0 {
		return nil
	}
	runes := []rune(l.text)
	var cells []txCell
	start := 0
	for i := 0; i <= len(runes); i++ {
		if i < len(runes) && runes[i] != delim {
			continue
		}
		s, e := start, i
		for s < e && unicode.IsSpace(runes[s]) {
			s++
		}
		for e > s && unicode.IsSpace(runes[e-1]) {
			e--
		}
		cells = append(cells, txCell{text: string(runes[s:e]), start: s, end: e})
		start = i + 1
	}
	if delim == '|' {
		if len(cells) > 0 && cells[0].text == "" {
			cells = cells[1:]
		}
		if len(cells) > 0 && cells[len(cells)-1].text == "" {
			cells = cells[:len(cells)-1]
		}
	}
	for ci := range cells {
		for wi, w := range l.words {
			if w.start >= cells[ci].start && w.end <= cells[ci].end {
				cells[ci].words = append(cells[ci].words, wi)
			}
		}
	}
	return cells
}

// txParseKV reads a "Key: Value" line.
func txParseKV(l *txLine) *txKV {
	runes := []rune(l.text)
	colon := -1
	for i, r := range runes {
		if r == ':' {
			colon = i
			break
		}
	}
	if colon <= 0 {
		return nil
	}
	key := strings.TrimSpace(string(runes[:colon]))
	rest := string(runes[colon+1:])
	if key == "" || len([]rune(key)) > 64 || strings.HasPrefix(rest, "//") || !strings.ContainsFunc(key, unicode.IsLetter) {
		return nil
	}
	kv := &txKV{key: key, value: strings.TrimSpace(rest), keyEnd: colon + 1, valueStart: colon + 1}
	for kv.valueStart < len(runes) && unicode.IsSpace(runes[kv.valueStart]) {
		kv.valueStart++
	}
	for wi, w := range l.words {
		if w.start < kv.keyEnd {
			kv.keyWords = append(kv.keyWords, wi)
		} else {
			kv.valWords = append(kv.valWords, wi)
		}
	}
	return kv
}

func txBlock(blockType, id string, page int, box txBox, confidence float64) map[string]any {
	return map[string]any{"BlockType": blockType, "Id": id, "Page": page, "Geometry": box.geometry(), "Confidence": confidence}
}

// txRel builds a Relationship. Ids must be []any: the model's Conform only
// keeps lists of that type.
func txRel(typ string, ids []string) map[string]any {
	list := make([]any, len(ids))
	for i, id := range ids {
		list[i] = id
	}
	return map[string]any{"Type": typ, "Ids": list}
}

func txIDs(l *txLine, idx []int) []string {
	out := make([]string, 0, len(idx))
	for _, i := range idx {
		out = append(out, l.wordIDs[i])
	}
	return out
}

// pageBlocks returns the PAGE, LINE and WORD blocks for one page, plus the
// LINE ids (the PAGE block's default children).
func (a *txAnalysis) pageBlocks(p *txPageInfo) (page map[string]any, rest []any, lineIDs []string) {
	page = txBlock("PAGE", a.id("page", txItoa(p.number)), p.number, txBox{0, 0, 1, 1}, txLineConfidence)
	var words []any
	for _, l := range p.lines {
		lb := txBlock("LINE", l.id, p.number, p.lineBox(l), txLineConfidence)
		lb["Text"] = l.text
		if len(l.wordIDs) > 0 {
			lb["Relationships"] = []any{txRel("CHILD", l.wordIDs)}
		}
		rest = append(rest, lb)
		lineIDs = append(lineIDs, l.id)
		for wi, w := range l.words {
			wb := txBlock("WORD", l.wordIDs[wi], p.number, p.spanBox(l, w.start, w.end), txLineConfidence)
			wb["Text"], wb["TextType"] = w.text, "PRINTED"
			words = append(words, wb)
		}
	}
	return page, append(rest, words...), lineIDs
}

// textBlocks is DetectDocumentText's output: PAGE, LINE and WORD blocks.
func (a *txAnalysis) textBlocks() []any {
	var out []any
	for _, p := range a.pages {
		page, rest, lineIDs := a.pageBlocks(p)
		if len(lineIDs) > 0 {
			page["Relationships"] = []any{txRel("CHILD", lineIDs)}
		}
		out = append(append(out, page), rest...)
	}
	return out
}

// analyzeBlocks is AnalyzeDocument's output for the requested features.
func (a *txAnalysis) analyzeBlocks(features map[string]bool, queries []any) ([]any, *txErr) {
	var queryBlocks []any
	if features["QUERIES"] {
		qb, err := a.queryBlocks(queries)
		if err != nil {
			return nil, err
		}
		queryBlocks = qb
	}
	var out []any
	for _, p := range a.pages {
		page, rest, children := a.pageBlocks(p)
		if features["FORMS"] {
			for _, l := range p.lines {
				if l.kv == nil {
					continue
				}
				kb, vb := a.kvBlocks(p, l)
				rest = append(rest, kb, vb)
				children = append(children, kb["Id"].(string), vb["Id"].(string))
			}
		}
		if features["TABLES"] {
			for ti, t := range p.tables {
				blocks := a.tableBlocks(p, ti, t)
				rest = append(rest, blocks...)
				children = append(children, blocks[0].(map[string]any)["Id"].(string))
			}
		}
		if features["SIGNATURES"] {
			for _, l := range p.lines {
				if box, ok := txSignature(p, l); ok {
					sb := txBlock("SIGNATURE", a.id("signature", txItoa(p.number), txItoa(l.index)), p.number, box, txKVConfidence)
					rest = append(rest, sb)
					children = append(children, sb["Id"].(string))
				}
			}
		}
		if features["LAYOUT"] {
			blocks := a.layoutBlocks(p)
			rest = append(rest, blocks...)
			for _, b := range blocks {
				children = append(children, b.(map[string]any)["Id"].(string))
			}
		}
		if len(children) > 0 {
			page["Relationships"] = []any{txRel("CHILD", children)}
		}
		out = append(append(out, page), rest...)
	}
	return append(out, queryBlocks...), nil
}

func (a *txAnalysis) kvBlocks(p *txPageInfo, l *txLine) (key, value map[string]any) {
	keyID := a.id("key", txItoa(p.number), txItoa(l.index))
	valID := a.id("value", txItoa(p.number), txItoa(l.index))
	key = txBlock("KEY_VALUE_SET", keyID, p.number, p.spanBox(l, 0, l.kv.keyEnd), txKVConfidence)
	key["EntityTypes"] = []any{"KEY"}
	rels := []any{txRel("VALUE", []string{valID})}
	if len(l.kv.keyWords) > 0 {
		rels = append(rels, txRel("CHILD", txIDs(l, l.kv.keyWords)))
	}
	key["Relationships"] = rels
	value = txBlock("KEY_VALUE_SET", valID, p.number, p.spanBox(l, l.kv.valueStart, l.runes), txKVConfidence)
	value["EntityTypes"] = []any{"VALUE"}
	if len(l.kv.valWords) > 0 {
		value["Relationships"] = []any{txRel("CHILD", txIDs(l, l.kv.valWords))}
	}
	return key, value
}

// tableBlocks returns a TABLE block (first) followed by its CELL blocks.
func (a *txAnalysis) tableBlocks(p *txPageInfo, ti int, t *txTable) []any {
	var rows []*txLine
	cols := 0
	var box txBox
	for _, l := range t.lines {
		box = box.union(p.lineBox(l))
		if l.cells != nil {
			rows = append(rows, l)
			cols = max(cols, len(l.cells))
		}
	}
	table := txBlock("TABLE", a.id("table", txItoa(p.number), txItoa(ti)), p.number, box, txKVConfidence)
	table["EntityTypes"] = []any{"STRUCTURED_TABLE"}
	var cellIDs []string
	var cells []any
	for ri, l := range rows {
		for ci := 0; ci < cols; ci++ {
			id := a.id("cell", txItoa(p.number), txItoa(ti), txItoa(ri), txItoa(ci))
			var cb map[string]any
			if ci < len(l.cells) {
				c := l.cells[ci]
				cb = txBlock("CELL", id, p.number, p.spanBox(l, c.start, c.end), txKVConfidence)
				if len(c.words) > 0 {
					cb["Relationships"] = []any{txRel("CHILD", txIDs(l, c.words))}
				}
			} else { // missing trailing cell: an empty cell at the line's end
				cb = txBlock("CELL", id, p.number, p.spanBox(l, l.runes, l.runes), txKVConfidence)
			}
			cb["RowIndex"], cb["ColumnIndex"], cb["RowSpan"], cb["ColumnSpan"] = ri+1, ci+1, 1, 1
			if ri == 0 && len(rows) > 1 {
				cb["EntityTypes"] = []any{"COLUMN_HEADER"}
			}
			cellIDs = append(cellIDs, id)
			cells = append(cells, cb)
		}
	}
	if len(cellIDs) > 0 {
		table["Relationships"] = []any{txRel("CHILD", cellIDs)}
	}
	return append([]any{table}, cells...)
}

var (
	txSignatureLabel = regexp.MustCompile(`(?i)^\s*(signature|signed|signed by)\s*:\s*\S`)
	txPageNumber     = regexp.MustCompile(`^\s*(page\s+)?\d+(\s+of\s+\d+)?\s*$`)
)

// txSignature reports a signature on "/s/ Name" or "Signature: ..." lines.
// ponytail: a text convention for signatures; real detection is visual.
func txSignature(p *txPageInfo, l *txLine) (txBox, bool) {
	trimmed := strings.TrimSpace(l.text)
	switch {
	case strings.HasPrefix(trimmed, "/s/"):
		return p.lineBox(l), true
	case txSignatureLabel.MatchString(l.text) && l.kv != nil:
		return p.spanBox(l, l.kv.valueStart, l.runes), true
	}
	return txBox{}, false
}

// layoutBlocks classifies each page element: a table becomes LAYOUT_TABLE,
// key/value lines LAYOUT_KEY_VALUE, the first line LAYOUT_TITLE, a trailing
// page number LAYOUT_PAGE_NUMBER, and the rest LAYOUT_TEXT.
func (a *txAnalysis) layoutBlocks(p *txPageInfo) []any {
	var out []any
	add := func(typ string, idx int, lines []*txLine) {
		var box txBox
		var ids []string
		for _, l := range lines {
			box = box.union(p.lineBox(l))
			ids = append(ids, l.id)
		}
		b := txBlock(typ, a.id("layout", txItoa(p.number), txItoa(idx)), p.number, box, txKVConfidence)
		b["Relationships"] = []any{txRel("CHILD", ids)}
		out = append(out, b)
	}
	tableOf := map[*txLine]*txTable{}
	for _, t := range p.tables {
		for _, l := range t.lines {
			tableOf[l] = t
		}
	}
	seen := map[*txTable]bool{}
	for i, l := range p.lines {
		if t := tableOf[l]; t != nil {
			if !seen[t] {
				seen[t] = true
				add("LAYOUT_TABLE", i, t.lines)
			}
			continue
		}
		switch {
		case l.kv != nil:
			add("LAYOUT_KEY_VALUE", i, []*txLine{l})
		case i == 0 && len(p.lines) > 1:
			add("LAYOUT_TITLE", i, []*txLine{l})
		case i == len(p.lines)-1 && txPageNumber.MatchString(l.text):
			add("LAYOUT_PAGE_NUMBER", i, []*txLine{l})
		default:
			add("LAYOUT_TEXT", i, []*txLine{l})
		}
	}
	return out
}

var txQueryStopwords = map[string]bool{
	"what": true, "whats": true, "is": true, "are": true, "the": true, "a": true, "an": true, "of": true,
	"on": true, "in": true, "for": true, "this": true, "that": true, "document": true, "s": true,
	"your": true, "my": true, "to": true, "do": true, "does": true, "was": true, "which": true,
}

func txTokens(s string) []string {
	var out []string
	for _, f := range strings.FieldsFunc(strings.ToLower(s), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsNumber(r) }) {
		if !txQueryStopwords[f] {
			out = append(out, f)
		}
	}
	return out
}

// txQueryPages turns Query.Pages ("1", "2-4", "*") into a page set; the
// default is the first page.
func txQueryPages(spec []any, pageCount int) (map[int]bool, *txErr) {
	pages := map[int]bool{}
	if len(spec) == 0 {
		pages[1] = true
		return pages, nil
	}
	parse := func(s string) (int, bool) {
		if s == "*" {
			return pageCount, true
		}
		n, err := strconv.ParseInt(s, 10, 32)
		return int(n), err == nil && n >= 1
	}
	for _, v := range spec {
		s, _ := v.(string)
		lo, hi := s, s
		if a, b, ok := strings.Cut(s, "-"); ok {
			lo, hi = a, b
		}
		if s == "*" {
			lo = "1"
		}
		from, ok1 := parse(lo)
		to, ok2 := parse(hi)
		if !ok1 || !ok2 || from > to {
			return nil, txError("InvalidParameterException", "Query page specification "+s+" is invalid")
		}
		for n := from; n <= min(to, pageCount); n++ {
			pages[n] = true
		}
	}
	return pages, nil
}

// queryBlocks answers each query from the key/value line whose key best
// matches the query's words.
// ponytail: keyword overlap against "Key: Value" lines; real queries use a language model.
func (a *txAnalysis) queryBlocks(queries []any) ([]any, *txErr) {
	if len(queries) > txMaxQueries {
		return nil, txError("InvalidParameterException", "At most "+txItoa(txMaxQueries)+" queries are allowed")
	}
	var out []any
	for qi, raw := range queries {
		q, _ := raw.(map[string]any)
		text, _ := q["Text"].(string)
		pagesSpec, _ := q["Pages"].([]any)
		allowed, err := txQueryPages(pagesSpec, len(a.pages))
		if err != nil {
			return nil, err
		}
		qTokens := txTokens(text)
		var best *txLine
		bestScore := 0.0
		for _, p := range a.pages {
			if !allowed[p.number] {
				continue
			}
			for _, l := range p.lines {
				if l.kv == nil {
					continue
				}
				kTokens := txTokens(l.kv.key)
				overlap := 0
				for _, qt := range qTokens {
					for _, kt := range kTokens {
						if qt == kt {
							overlap++
							break
						}
					}
				}
				if overlap == 0 {
					continue
				}
				score := float64(overlap) / float64(len(kTokens)+len(qTokens)-overlap)
				if score > bestScore {
					best, bestScore = l, score
				}
			}
		}
		qid := a.id("query", txItoa(qi))
		query := map[string]any{"Text": text}
		if alias, ok := q["Alias"].(string); ok {
			query["Alias"] = alias
		}
		if len(pagesSpec) > 0 {
			query["Pages"] = pagesSpec
		}
		qb := map[string]any{"BlockType": "QUERY", "Id": qid, "Query": query, "Page": 1}
		out = append(out, qb)
		if best == nil || best.kv.value == "" {
			continue
		}
		p := a.pages[best.page-1]
		rid := a.id("query-result", txItoa(qi))
		rb := txBlock("QUERY_RESULT", rid, best.page, p.spanBox(best, best.kv.valueStart, best.runes), math.Round(bestScore*1000)/10)
		rb["Text"] = best.kv.value
		qb["Page"] = best.page
		qb["Relationships"] = []any{txRel("ANSWER", []string{rid})}
		out = append(out, rb)
	}
	return out, nil
}
