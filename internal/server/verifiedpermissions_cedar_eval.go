package server

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

// Cedar values: bool, int64 (Long), string, cedarEntity, cedarSet, cedarRecord
// and cedarExt (an extension value such as an ipaddr or decimal, compared by
// its text only).
type (
	cedarSet    []any
	cedarRecord map[string]any
	cedarExt    struct{ Kind, Text string }
)

type cedarEntityData struct {
	Attrs   cedarRecord
	Parents []cedarEntity
}

type cedarRequest struct {
	Principal, Action, Resource cedarEntity
	Context                     cedarRecord
	Entities                    map[cedarEntity]*cedarEntityData
}

// cedarAuthorize evaluates policies (keyed by id) against req: any satisfied
// forbid denies; otherwise any satisfied permit allows; otherwise deny. A
// policy whose evaluation errors is skipped and its error reported, as Cedar
// does.
//
// All policies share one evaluation budget. If it runs out the request is
// denied outright: skipping the unevaluated policies could skip a forbid.
func cedarAuthorize(policies map[string]*cedarPolicy, req *cedarRequest) (allow bool, determining []string, errs []string) {
	budget := cedarEvalBudget
	ids := make([]string, 0, len(policies))
	for id := range policies {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	var permits, forbids []string
	for _, id := range ids {
		ok, err := cedarSatisfiedWithin(policies[id], req, &budget)
		if errors.Is(err, errCedarBudget) {
			return false, nil, []string{fmt.Sprintf("evaluation stopped: the request exceeded %d evaluation steps", cedarEvalBudget)}
		}
		if err != nil {
			errs = append(errs, fmt.Sprintf("while evaluating policy `%s`, encountered the following error: %s", id, err))
			continue
		}
		if !ok {
			continue
		}
		if policies[id].Effect == "forbid" {
			forbids = append(forbids, id)
		} else {
			permits = append(permits, id)
		}
	}
	if len(forbids) > 0 {
		return false, forbids, errs
	}
	return len(permits) > 0, permits, errs
}

func cedarSatisfied(p *cedarPolicy, req *cedarRequest) (bool, error) {
	budget := cedarEvalBudget
	return cedarSatisfiedWithin(p, req, &budget)
}

func cedarSatisfiedWithin(p *cedarPolicy, req *cedarRequest, budget *int) (bool, error) {
	ev := &cedarEval{req: req, budget: budget}
	if !ev.scope(p.Principal, req.Principal) || !ev.actionScope(p.Action) || !ev.scope(p.Resource, req.Resource) {
		return false, ev.err
	}
	for _, c := range p.Conditions {
		v, err := ev.eval(c.Expr)
		if err != nil {
			return false, err
		}
		b, ok := v.(bool)
		if !ok {
			return false, fmt.Errorf("type error: condition must be a boolean, got %s", cedarTypeName(v))
		}
		if b != c.When {
			return false, nil
		}
	}
	return true, nil
}

// cedarEvalBudget bounds the work one authorization request can do: each
// expression node, hierarchy step and set element costs one, and computing a
// value's canonical key costs its length in 64-byte units.
const cedarEvalBudget = 1_000_000

var errCedarBudget = errors.New("evaluation budget exceeded")

type cedarEval struct {
	req    *cedarRequest
	budget *int
	err    error // set when the budget runs out inside a bool-returning helper
}

func (ev *cedarEval) charge(n int) error {
	*ev.budget -= n
	if *ev.budget < 0 {
		ev.err = errCedarBudget
		return errCedarBudget
	}
	return nil
}

// key is cedarKey with its cost charged.
func (ev *cedarEval) key(v any) (string, error) {
	k := cedarKey(v)
	return k, ev.charge(1 + len(k)/64)
}

func (ev *cedarEval) scope(sc cedarScope, e cedarEntity) bool {
	switch sc.Op {
	case "==":
		return sc.Entity != nil && e == *sc.Entity
	case "in":
		return sc.Entity != nil && ev.inEntity(e, *sc.Entity)
	case "is":
		return e.Type == sc.IsType && (sc.Entity == nil || ev.inEntity(e, *sc.Entity))
	}
	return true
}

func (ev *cedarEval) actionScope(sc cedarActionScope) bool {
	switch sc.Op {
	case "==":
		return ev.req.Action == sc.Entities[0]
	case "in":
		for _, a := range sc.Entities {
			if ev.inEntity(ev.req.Action, a) {
				return true
			}
		}
		return false
	}
	return true
}

// inEntity reports whether e is target or a descendant of target. The walk is
// breadth-first with a visited set, so cyclic hierarchies terminate.
func (ev *cedarEval) inEntity(e, target cedarEntity) bool {
	if e == target {
		return true
	}
	seen := map[cedarEntity]bool{e: true}
	queue := []cedarEntity{e}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if ev.charge(1) != nil {
			return false
		}
		data := ev.req.Entities[cur]
		if data == nil {
			continue
		}
		for _, p := range data.Parents {
			if p == target {
				return true
			}
			if !seen[p] {
				seen[p] = true
				queue = append(queue, p)
			}
		}
	}
	return false
}

func cedarTypeName(v any) string {
	switch v.(type) {
	case bool:
		return "Bool"
	case int64:
		return "Long"
	case string:
		return "String"
	case cedarEntity:
		return "Entity"
	case cedarSet:
		return "Set"
	case cedarRecord:
		return "Record"
	case cedarExt:
		return "Extension"
	}
	return "unknown"
}

var errCedarOverflow = errors.New("integer overflow")

func (ev *cedarEval) eval(x cedarExpr) (any, error) {
	if err := ev.charge(1); err != nil {
		return nil, err
	}
	switch n := x.(type) {
	case cedarLit:
		return n.V, nil
	case cedarVar:
		switch n.Name {
		case "principal":
			return ev.req.Principal, nil
		case "action":
			return ev.req.Action, nil
		case "resource":
			return ev.req.Resource, nil
		}
		return ev.req.Context, nil
	case cedarSetLit:
		out := make(cedarSet, 0, len(n.Elems))
		for _, e := range n.Elems {
			v, err := ev.eval(e)
			if err != nil {
				return nil, err
			}
			out = append(out, v)
		}
		return out, nil
	case cedarRecLit:
		out := make(cedarRecord, len(n.Keys))
		for i, k := range n.Keys {
			v, err := ev.eval(n.Vals[i])
			if err != nil {
				return nil, err
			}
			out[k] = v
		}
		return out, nil
	case cedarIf:
		c, err := ev.evalBool(n.C)
		if err != nil {
			return nil, err
		}
		if c {
			return ev.eval(n.T)
		}
		return ev.eval(n.E)
	case cedarUnary:
		v, err := ev.eval(n.X)
		if err != nil {
			return nil, err
		}
		if n.Op == "!" {
			b, ok := v.(bool)
			if !ok {
				return nil, fmt.Errorf("type error: ! expects a boolean, got %s", cedarTypeName(v))
			}
			return !b, nil
		}
		i, ok := v.(int64)
		if !ok {
			return nil, fmt.Errorf("type error: unary - expects a long, got %s", cedarTypeName(v))
		}
		if i == math.MinInt64 {
			return nil, errCedarOverflow
		}
		return -i, nil
	case cedarBinary:
		return ev.binary(n)
	case cedarAttr:
		v, err := ev.eval(n.X)
		if err != nil {
			return nil, err
		}
		rec, err := ev.attrs(v)
		if err != nil {
			return nil, err
		}
		got, ok := rec[n.Name]
		if !ok {
			return nil, fmt.Errorf("attribute `%s` does not exist on %s", n.Name, cedarDescribeValue(v))
		}
		return got, nil
	case cedarHas:
		v, err := ev.eval(n.X)
		if err != nil {
			return nil, err
		}
		if e, ok := v.(cedarEntity); ok {
			data := ev.req.Entities[e]
			if data == nil {
				return false, nil
			}
			_, has := data.Attrs[n.Name]
			return has, nil
		}
		rec, ok := v.(cedarRecord)
		if !ok {
			return nil, fmt.Errorf("type error: has expects an entity or record, got %s", cedarTypeName(v))
		}
		_, has := rec[n.Name]
		return has, nil
	case cedarLike:
		v, err := ev.eval(n.X)
		if err != nil {
			return nil, err
		}
		s, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("type error: like expects a string, got %s", cedarTypeName(v))
		}
		return cedarMatch(n.Pattern, s), nil
	case cedarIs:
		v, err := ev.eval(n.X)
		if err != nil {
			return nil, err
		}
		e, ok := v.(cedarEntity)
		if !ok {
			return nil, fmt.Errorf("type error: is expects an entity, got %s", cedarTypeName(v))
		}
		if e.Type != n.Type {
			return false, nil
		}
		if n.In == nil {
			return true, nil
		}
		return ev.in(e, n.In)
	case cedarCall:
		return ev.call(n)
	}
	return nil, fmt.Errorf("unsupported expression")
}

func (ev *cedarEval) evalBool(x cedarExpr) (bool, error) {
	v, err := ev.eval(x)
	if err != nil {
		return false, err
	}
	b, ok := v.(bool)
	if !ok {
		return false, fmt.Errorf("type error: expected a boolean, got %s", cedarTypeName(v))
	}
	return b, nil
}

func (ev *cedarEval) attrs(v any) (cedarRecord, error) {
	switch t := v.(type) {
	case cedarRecord:
		return t, nil
	case cedarEntity:
		data := ev.req.Entities[t]
		if data == nil {
			return nil, fmt.Errorf("entity %s does not exist", t)
		}
		return data.Attrs, nil
	}
	return nil, fmt.Errorf("type error: attribute access expects an entity or record, got %s", cedarTypeName(v))
}

func (ev *cedarEval) binary(n cedarBinary) (any, error) {
	if n.Op == "&&" || n.Op == "||" { // short-circuit
		l, err := ev.evalBool(n.L)
		if err != nil {
			return nil, err
		}
		if (n.Op == "&&" && !l) || (n.Op == "||" && l) {
			return l, nil
		}
		return ev.evalBool(n.R)
	}
	l, err := ev.eval(n.L)
	if err != nil {
		return nil, err
	}
	if n.Op == "in" {
		e, ok := l.(cedarEntity)
		if !ok {
			return nil, fmt.Errorf("type error: in expects an entity on the left, got %s", cedarTypeName(l))
		}
		return ev.in(e, n.R)
	}
	r, err := ev.eval(n.R)
	if err != nil {
		return nil, err
	}
	switch n.Op {
	case "==", "!=":
		lk, err := ev.key(l)
		if err != nil {
			return nil, err
		}
		rk, err := ev.key(r)
		if err != nil {
			return nil, err
		}
		return (lk == rk) == (n.Op == "=="), nil
	}
	a, aok := l.(int64)
	b, bok := r.(int64)
	if !aok || !bok {
		return nil, fmt.Errorf("type error: %s expects longs, got %s and %s", n.Op, cedarTypeName(l), cedarTypeName(r))
	}
	switch n.Op {
	case "<":
		return a < b, nil
	case "<=":
		return a <= b, nil
	case ">":
		return a > b, nil
	case ">=":
		return a >= b, nil
	case "+":
		if (b > 0 && a > math.MaxInt64-b) || (b < 0 && a < math.MinInt64-b) {
			return nil, errCedarOverflow
		}
		return a + b, nil
	case "-":
		if (b < 0 && a > math.MaxInt64+b) || (b > 0 && a < math.MinInt64+b) {
			return nil, errCedarOverflow
		}
		return a - b, nil
	case "*":
		if a != 0 && b != 0 {
			p := a * b
			if p/b != a || (a == -1 && b == math.MinInt64) || (b == -1 && a == math.MinInt64) {
				return nil, errCedarOverflow
			}
			return p, nil
		}
		return int64(0), nil
	}
	return nil, fmt.Errorf("unsupported operator %s", n.Op)
}

// in evaluates `e in rhs`, where rhs is an entity or a set of entities.
func (ev *cedarEval) in(e cedarEntity, rhs cedarExpr) (any, error) {
	r, err := ev.eval(rhs)
	if err != nil {
		return nil, err
	}
	switch t := r.(type) {
	case cedarEntity:
		return ev.inEntity(e, t), ev.err
	case cedarSet:
		for _, item := range t {
			target, ok := item.(cedarEntity)
			if !ok {
				return nil, fmt.Errorf("type error: in expects a set of entities, found %s", cedarTypeName(item))
			}
			if ev.inEntity(e, target) {
				return true, nil
			}
			if ev.err != nil {
				return nil, ev.err
			}
		}
		return false, nil
	}
	return nil, fmt.Errorf("type error: in expects an entity or set on the right, got %s", cedarTypeName(r))
}

func (ev *cedarEval) call(n cedarCall) (any, error) {
	recv, err := ev.eval(n.Recv)
	if err != nil {
		return nil, err
	}
	set, ok := recv.(cedarSet)
	if !ok {
		return nil, fmt.Errorf("type error: %s expects a set, got %s", n.Method, cedarTypeName(recv))
	}
	if n.Method == "isEmpty" {
		return len(set) == 0, nil
	}
	arg, err := ev.eval(n.Args[0])
	if err != nil {
		return nil, err
	}
	have := make(map[string]bool, len(set))
	for _, v := range set {
		k, err := ev.key(v)
		if err != nil {
			return nil, err
		}
		have[k] = true
	}
	if n.Method == "contains" {
		k, err := ev.key(arg)
		return have[k], err
	}
	other, ok := arg.(cedarSet)
	if !ok {
		return nil, fmt.Errorf("type error: %s expects a set argument, got %s", n.Method, cedarTypeName(arg))
	}
	for _, v := range other {
		k, err := ev.key(v)
		if err != nil {
			return nil, err
		}
		in := have[k]
		if n.Method == "containsAny" && in {
			return true, nil
		}
		if n.Method == "containsAll" && !in {
			return false, nil
		}
	}
	return n.Method == "containsAll", nil
}

// cedarMatch matches s against a pattern of literal runs and wildcards. With
// only *, matching each literal run at its leftmost position is correct, so
// this is one pass of strings.Index calls rather than backtracking.
func cedarMatch(parts []cedarPatternPart, s string) bool {
	if len(parts) == 0 {
		return s == ""
	}
	first, last := 0, len(parts)
	if !parts[0].Wildcard {
		if !strings.HasPrefix(s, parts[0].Lit) {
			return false
		}
		s = s[len(parts[0].Lit):]
		first = 1
	}
	if last > first && !parts[last-1].Wildcard {
		suffix := parts[last-1].Lit
		if !strings.HasSuffix(s, suffix) {
			return false
		}
		s = s[:len(s)-len(suffix)]
		last--
	}
	if first == last { // the pattern was a single literal run
		return s == ""
	}
	for _, p := range parts[first:last] {
		if p.Wildcard {
			continue
		}
		i := strings.Index(s, p.Lit)
		if i < 0 {
			return false
		}
		s = s[i+len(p.Lit):]
	}
	// The run between the first and last literals starts or ends with a
	// wildcard here unless the pattern had no wildcard at all.
	hasWild := false
	for _, p := range parts {
		hasWild = hasWild || p.Wildcard
	}
	return hasWild || s == ""
}

// cedarKey is a canonical encoding of a value for equality and set
// membership: sets compare as sets and records by their sorted keys.
func cedarKey(v any) string {
	var b strings.Builder
	cedarWriteKey(&b, v)
	return b.String()
}

func cedarWriteKey(b *strings.Builder, v any) {
	switch t := v.(type) {
	case bool:
		b.WriteString("b:" + strconv.FormatBool(t))
	case int64:
		b.WriteString("l:" + strconv.FormatInt(t, 10))
	case string:
		b.WriteString("s:" + strconv.Quote(t))
	case cedarEntity:
		b.WriteString("e:" + strconv.Quote(t.Type) + strconv.Quote(t.ID))
	case cedarExt:
		b.WriteString("x:" + strconv.Quote(t.Kind) + strconv.Quote(t.Text))
	case cedarSet:
		keys := make([]string, 0, len(t))
		seen := map[string]bool{}
		for _, item := range t {
			if k := cedarKey(item); !seen[k] {
				seen[k] = true
				keys = append(keys, k)
			}
		}
		sort.Strings(keys)
		b.WriteString("S[" + strings.Join(keys, ",") + "]")
	case cedarRecord:
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		b.WriteString("R{")
		for _, k := range keys {
			b.WriteString(strconv.Quote(k) + "=")
			cedarWriteKey(b, t[k])
			b.WriteString(";")
		}
		b.WriteString("}")
	default:
		b.WriteString("?")
	}
}

func cedarDescribeValue(v any) string {
	if e, ok := v.(cedarEntity); ok {
		return e.String()
	}
	return "a " + strings.ToLower(cedarTypeName(v))
}
