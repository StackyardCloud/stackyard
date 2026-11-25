package server

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// A subset of Detector PL: $variables, "string" and numeric literals,
// true/false, == != > >= < <=, and/or/not and parentheses. Comparisons are
// numeric when both sides are numbers, otherwise textual.

type frdTok struct{ kind, text string } // kind: var, str, num, op, word, (, )

func frdLex(src string) ([]frdTok, error) {
	var toks []frdTok
	for i := 0; i < len(src); {
		c := src[i]
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			i++
		case c == '(' || c == ')':
			toks = append(toks, frdTok{kind: string(c)})
			i++
		case c == '"' || c == '\'':
			j := strings.IndexByte(src[i+1:], c)
			if j < 0 {
				return nil, fmt.Errorf("unterminated string at %d", i)
			}
			toks = append(toks, frdTok{kind: "str", text: src[i+1 : i+1+j]})
			i += j + 2
		case strings.ContainsRune("=!<>", rune(c)):
			op := string(c)
			if i+1 < len(src) && src[i+1] == '=' {
				op += "="
			}
			if op == "=" || op == "!" {
				return nil, fmt.Errorf("unknown operator %q at %d", op, i)
			}
			toks = append(toks, frdTok{kind: "op", text: op})
			i += len(op)
		default:
			j := i
			if c == '$' || c == '-' {
				j++
			}
			for j < len(src) && (unicode.IsLetter(rune(src[j])) || unicode.IsDigit(rune(src[j])) || src[j] == '_' || src[j] == '.') {
				j++
			}
			word := src[i:j]
			switch {
			case j == i || word == "$" || word == "-":
				return nil, fmt.Errorf("unexpected %q at %d", c, i)
			case c == '$':
				toks = append(toks, frdTok{kind: "var", text: word[1:]})
			case isFrdNumber(word):
				toks = append(toks, frdTok{kind: "num", text: word})
			default:
				toks = append(toks, frdTok{kind: "word", text: strings.ToLower(word)})
			}
			i = j
		}
	}
	return toks, nil
}

func isFrdNumber(s string) bool {
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

type frdEval struct {
	toks   []frdTok
	pos    int
	lookup func(string) string // nil when only checking syntax
	vars   []string
}

func (e *frdEval) peek() frdTok {
	if e.pos < len(e.toks) {
		return e.toks[e.pos]
	}
	return frdTok{kind: "eof"}
}

func (e *frdEval) word(w string) bool {
	if t := e.peek(); t.kind == "word" && t.text == w {
		e.pos++
		return true
	}
	return false
}

func (e *frdEval) or() (string, error) {
	l, err := e.and()
	for err == nil && e.word("or") {
		var r string
		if r, err = e.and(); err == nil {
			l = strconv.FormatBool(l == "true" || r == "true")
		}
	}
	return l, err
}

func (e *frdEval) and() (string, error) {
	l, err := e.not()
	for err == nil && e.word("and") {
		var r string
		if r, err = e.not(); err == nil {
			l = strconv.FormatBool(l == "true" && r == "true")
		}
	}
	return l, err
}

func (e *frdEval) not() (string, error) {
	if e.word("not") {
		v, err := e.not()
		return strconv.FormatBool(v != "true"), err
	}
	return e.cmp()
}

func (e *frdEval) cmp() (string, error) {
	l, err := e.primary()
	if err != nil || e.peek().kind != "op" {
		return l, err
	}
	op := e.toks[e.pos].text
	e.pos++
	r, err := e.primary()
	if err != nil {
		return "", err
	}
	c := strings.Compare(l, r)
	if lf, lerr := strconv.ParseFloat(l, 64); lerr == nil {
		if rf, rerr := strconv.ParseFloat(r, 64); rerr == nil {
			switch {
			case lf < rf:
				c = -1
			case lf > rf:
				c = 1
			default:
				c = 0
			}
		}
	}
	res := map[string]bool{"==": c == 0, "!=": c != 0, "<": c < 0, "<=": c <= 0, ">": c > 0, ">=": c >= 0}[op]
	return strconv.FormatBool(res), nil
}

func (e *frdEval) primary() (string, error) {
	t := e.peek()
	e.pos++
	switch t.kind {
	case "(":
		v, err := e.or()
		if err == nil && e.peek().kind != ")" {
			err = fmt.Errorf("missing )")
		}
		e.pos++
		return v, err
	case "var":
		e.vars = append(e.vars, t.text)
		if e.lookup == nil {
			return "", nil
		}
		return e.lookup(t.text), nil
	case "str", "num":
		return t.text, nil
	case "word":
		if t.text == "true" || t.text == "false" {
			return t.text, nil
		}
	}
	return "", fmt.Errorf("unexpected %s %q", t.kind, t.text)
}

// frdRunExpr parses (and, with a lookup, evaluates) expr. It returns whether
// the expression matched and the variables it references.
func frdRunExpr(expr string, lookup func(string) string) (bool, []string, error) {
	toks, err := frdLex(expr)
	if err != nil {
		return false, nil, err
	}
	e := &frdEval{toks: toks, lookup: lookup}
	v, err := e.or()
	if err == nil && e.pos < len(toks) {
		err = fmt.Errorf("unexpected %q after the expression", toks[e.pos].text)
	}
	return v == "true", e.vars, err
}
