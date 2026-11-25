package server

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Recommendation engine and filters shared by Personalize Runtime and batch
// jobs. Scores are deterministic: a hash of the model, user and item blended
// with interaction popularity (or co-occurrence for related items), then
// normalized to sum to 1 over the candidates, as Personalize reports them.

const (
	pszMaxRecommendations = 500
	pszMaxActionResults   = 100
)

type pszScored struct {
	id    string
	score float64
}

// pszGroup is a dataset group's datasets (nil when absent).
type pszGroup struct {
	items, users, actions, interactions, actionInteractions *pszData
}

func (st *personalizeStore) group(arn string) *pszGroup {
	get := func(typ string) *pszData {
		if d := st.dataset(arn, typ); d != nil {
			return d.data
		}
		return nil
	}
	return &pszGroup{items: get(pszItems), users: get(pszUsers), actions: get(pszActions),
		interactions: get(pszInteractions), actionInteractions: get(pszActionInteractions)}
}

func (d *pszData) entityIDs() []string {
	if d == nil {
		return nil
	}
	ids := make([]string, 0, len(d.entities))
	for id := range d.entities {
		ids = append(ids, id)
	}
	return ids
}

func (d *pszData) eventList() []pszEvent {
	if d == nil {
		return nil
	}
	return d.events
}

// pszUnion returns the sorted distinct ids of a and b.
func pszUnion(a []string, b []pszEvent, user bool) []string {
	seen := map[string]bool{}
	for _, id := range a {
		seen[id] = true
	}
	for _, e := range b {
		if user {
			seen[e.user] = true
		} else {
			seen[e.target] = true
		}
	}
	out := make([]string, 0, len(seen))
	for id := range seen {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func pszCounts(events []pszEvent, keep func(pszEvent) bool, key func(pszEvent) string) (map[string]float64, float64) {
	counts, top := map[string]float64{}, 0.0
	for _, e := range events {
		if keep(e) {
			k := key(e)
			counts[k]++
			top = max(top, counts[k])
		}
	}
	return counts, max(top, 1)
}

func pszAll(pszEvent) bool        { return true }
func pszTarget(e pszEvent) string { return e.target }

// pszTop filters, sorts and normalizes scored candidates and keeps n.
func pszTop(scored []pszScored, n int) []pszScored {
	sum := 0.0
	for _, s := range scored {
		sum += s.score
	}
	sort.Slice(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		return scored[i].id < scored[j].id
	})
	if n >= 0 && len(scored) > n {
		scored = scored[:n]
	}
	for i := range scored {
		if sum > 0 {
			scored[i].score = pszRound(scored[i].score / sum)
		}
	}
	return scored
}

// recommendItems ranks the group's items for a user (USER_PERSONALIZATION),
// overall (POPULAR_ITEMS) or by co-occurrence with item (RELATED_ITEMS).
func (g *pszGroup) recommendItems(seed, recipeType, user, item string, n int, f *pszFilter, values map[string]string) ([]pszScored, *pszErr) {
	events := g.interactions.eventList()
	pop, top := pszCounts(events, pszAll, pszTarget)
	var co map[string]float64
	coTop := 1.0
	if recipeType == pszRelatedItems {
		peers := map[string]bool{}
		for _, e := range events {
			if e.target == item {
				peers[e.user] = true
			}
		}
		co, coTop = pszCounts(events, func(e pszEvent) bool { return peers[e.user] && e.target != item }, pszTarget)
	}
	ev := newPSZEval(g, f, user, values)
	var scored []pszScored
	for _, id := range pszUnion(g.items.entityIDs(), events, false) {
		if recipeType == pszRelatedItems && id == item {
			continue
		}
		ok, err := ev.allows(id)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		var s float64
		switch recipeType {
		case pszPopularItems:
			s = pop[id]/top + 0.01*pszUnit(seed, id)
		case pszRelatedItems:
			s = 0.7*co[id]/coTop + 0.3*pszUnit(seed, item, id)
		default:
			s = 0.7*pszUnit(seed, user, id) + 0.3*pop[id]/top
		}
		scored = append(scored, pszScored{id, s + 1e-6})
	}
	return pszTop(scored, n), nil
}

// rank orders inputList for user; filtered-out items are dropped.
func (g *pszGroup) rank(seed, user string, inputList []string, f *pszFilter, values map[string]string) ([]pszScored, *pszErr) {
	pop, top := pszCounts(g.interactions.eventList(), pszAll, pszTarget)
	ev := newPSZEval(g, f, user, values)
	seen := map[string]bool{}
	var scored []pszScored
	for _, id := range inputList {
		if seen[id] {
			continue
		}
		seen[id] = true
		ok, err := ev.allows(id)
		if err != nil {
			return nil, err
		}
		if ok {
			scored = append(scored, pszScored{id, 0.7*pszUnit(seed, user, id) + 0.3*pop[id]/top + 1e-6})
		}
	}
	return pszTop(scored, -1), nil
}

func (g *pszGroup) recommendActions(seed, user string, n int, f *pszFilter, values map[string]string) ([]pszScored, *pszErr) {
	pop, top := pszCounts(g.actionInteractions.eventList(), func(e pszEvent) bool { return e.eventType == "Taken" }, pszTarget)
	ev := newPSZEval(g, f, user, values)
	ids := g.actions.entityIDs()
	sort.Strings(ids)
	var scored []pszScored
	for _, id := range ids {
		ok, err := ev.allows(id)
		if err != nil {
			return nil, err
		}
		if ok {
			scored = append(scored, pszScored{id, 0.7*pszUnit(seed, user, id) + 0.3*pop[id]/top + 1e-6})
		}
	}
	return pszTop(scored, n), nil
}

// segment returns the users most likely to interact with any of items.
func (g *pszGroup) segment(seed string, items map[string]bool, n int, f *pszFilter, values map[string]string) ([]pszScored, *pszErr) {
	events := g.interactions.eventList()
	affinity, top := pszCounts(events, func(e pszEvent) bool { return items[e.target] }, func(e pszEvent) string { return e.user })
	ev := newPSZEval(g, f, "", values)
	key := strings.Join(pszSortedKeys(items), ",")
	var scored []pszScored
	for _, id := range pszUnion(g.users.entityIDs(), events, true) {
		ok, err := ev.allows(id)
		if err != nil {
			return nil, err
		}
		if ok {
			scored = append(scored, pszScored{id, 0.7*affinity[id]/top + 0.3*pszUnit(seed, key, id) + 1e-6})
		}
	}
	return pszTop(scored, n), nil
}

func pszSortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// ---- filters ----

type pszCond struct {
	dataset, field, op string
	values             []string
	param              string // $PARAM placeholder, resolved from filterValues
}

type pszFilter struct {
	include bool
	target  string // ItemID, UserID or ActionID
	conds   []pszCond
}

var pszFilterToken = regexp.MustCompile(`\s*("(?:[^"\\]|\\.)*"|\$[A-Za-z0-9_]+|[A-Za-z_][A-Za-z0-9_]*\.[A-Za-z_][A-Za-z0-9_]*|NOT\s+IN|<=|>=|!=|[=<>(),]|-?[0-9]+(?:\.[0-9]+)?|[A-Za-z_]+)`)

var pszFilterDatasets = map[string][]string{
	"ItemID": {pszItems, pszInteractions}, "UserID": {pszUsers, pszInteractions}, "ActionID": {pszActions, pszActionInteractions},
}

// pszParseFilter accepts INCLUDE|EXCLUDE ItemID|UserID|ActionID WHERE
// <Dataset>.<FIELD> <op> <values> [AND ...].
// ponytail: CurrentItem/CurrentUser references, OR and piped expressions
// aren't supported; they are refused rather than mis-evaluated.
func pszParseFilter(expr string) (*pszFilter, *pszErr) {
	var toks []string
	rest := strings.TrimSpace(expr)
	for rest != "" {
		loc := pszFilterToken.FindStringIndex(rest)
		if loc == nil || loc[0] != 0 {
			return nil, pszInvalid("Can't parse the filter expression near %q.", rest)
		}
		toks = append(toks, strings.TrimSpace(rest[:loc[1]]))
		rest = strings.TrimSpace(rest[loc[1]:])
	}
	bad := func() (*pszFilter, *pszErr) {
		return nil, pszInvalid("Stackyard supports filter expressions of the form INCLUDE|EXCLUDE ItemID|UserID|ActionID WHERE Dataset.FIELD <op> <values> [AND ...].")
	}
	if len(toks) < 6 || !strings.EqualFold(toks[2], "WHERE") {
		return bad()
	}
	f := &pszFilter{include: strings.EqualFold(toks[0], "INCLUDE"), target: toks[1]}
	if !f.include && !strings.EqualFold(toks[0], "EXCLUDE") || pszFilterDatasets[f.target] == nil {
		return bad()
	}
	i := 3
	for {
		if i+2 >= len(toks) {
			return bad()
		}
		ds, field, ok := strings.Cut(toks[i], ".")
		t, known := pszDatasetTypes[strings.ToLower(ds)]
		if !ok || !known || !pszContains(pszFilterDatasets[f.target], t.name) {
			return nil, pszInvalid("A %s filter can't use dataset %s.", f.target, ds)
		}
		if strings.Contains(t.name, "Interactions") && !strings.EqualFold(field, "EVENT_TYPE") {
			return nil, pszInvalid("Interaction conditions must use EVENT_TYPE.")
		}
		c := pszCond{dataset: t.name, field: strings.ToUpper(field), op: strings.Join(strings.Fields(strings.ToUpper(toks[i+1])), " ")}
		if !pszContains([]string{"IN", "NOT IN", "=", "!=", "<", ">", "<=", ">="}, c.op) {
			return bad()
		}
		i += 2
		vals, next, ok := pszFilterValues(toks, i)
		if !ok {
			return bad()
		}
		if len(vals) == 1 && strings.HasPrefix(vals[0], "$") {
			c.param = vals[0][1:]
		} else {
			for _, v := range vals {
				if strings.HasPrefix(v, "$") {
					return bad()
				}
			}
			c.values = vals
		}
		f.conds = append(f.conds, c)
		i = next
		if i == len(toks) {
			return f, nil
		}
		if !strings.EqualFold(toks[i], "AND") {
			return bad()
		}
		i++
	}
}

// pszFilterValues reads a value or a parenthesized list starting at toks[i].
func pszFilterValues(toks []string, i int) ([]string, int, bool) {
	if toks[i] != "(" {
		return []string{pszUnquote(toks[i])}, i + 1, toks[i] != ")" && toks[i] != ","
	}
	var vals []string
	for i++; i < len(toks); i += 2 {
		vals = append(vals, pszUnquote(toks[i]))
		if i+1 < len(toks) && toks[i+1] == ")" {
			return vals, i + 2, true
		}
		if i+1 >= len(toks) || toks[i+1] != "," {
			return nil, 0, false
		}
	}
	return nil, 0, false
}

func pszUnquote(tok string) string {
	if strings.HasPrefix(tok, `"`) {
		if s, err := strconv.Unquote(tok); err == nil {
			return s
		}
		return strings.Trim(tok, `"`)
	}
	return tok
}

func pszContains(list []string, v string) bool {
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}

// pszEval evaluates a filter for one request.
type pszEval struct {
	g      *pszGroup
	f      *pszFilter
	user   string
	values map[string]string
	events map[string]map[string]bool // candidate -> event types (lazily built)
}

func newPSZEval(g *pszGroup, f *pszFilter, user string, values map[string]string) *pszEval {
	return &pszEval{g: g, f: f, user: user, values: values}
}

func (ev *pszEval) eventTypes(candidate string) map[string]bool {
	if ev.events == nil {
		ev.events = map[string]map[string]bool{}
		events := ev.g.interactions.eventList()
		if ev.f.target == "ActionID" {
			events = ev.g.actionInteractions.eventList()
		}
		for _, e := range events {
			key := e.target
			if ev.f.target == "UserID" {
				key = e.user
			} else if e.user != ev.user {
				continue
			}
			if ev.events[key] == nil {
				ev.events[key] = map[string]bool{}
			}
			ev.events[key][e.eventType] = true
		}
	}
	return ev.events[candidate]
}

func (ev *pszEval) allows(candidate string) (bool, *pszErr) {
	if ev.f == nil {
		return true, nil
	}
	all := true
	for _, c := range ev.f.conds {
		want := c.values
		if c.param != "" {
			raw, ok := ev.values[c.param]
			if !ok {
				return false, pszInvalid("filterValues must set %s.", c.param)
			}
			want = pszParamValues(raw)
		}
		var have []string
		if strings.Contains(c.dataset, "Interactions") {
			for t := range ev.eventTypes(candidate) {
				have = append(have, t)
			}
		} else {
			d := map[string]*pszData{pszItems: ev.g.items, pszUsers: ev.g.users, pszActions: ev.g.actions}[c.dataset]
			if d != nil && d.entities[candidate] != nil {
				if v, ok := d.entities[candidate].props[c.field]; ok {
					have = strings.Split(v, "|")
				}
			}
		}
		if !pszCompare(c.op, have, want) {
			all = false
			break
		}
	}
	return all == ev.f.include, nil
}

// pszParamValues splits a filterValues entry: "\"a\",\"b\"" or 10.
func pszParamValues(raw string) []string {
	var out []string
	for _, part := range strings.Split(raw, ",") {
		out = append(out, pszUnquote(strings.TrimSpace(part)))
	}
	return out
}

func pszCompare(op string, have, want []string) bool {
	in := func(v string) bool { return pszContains(want, v) }
	switch op {
	case "IN", "=":
		for _, v := range have {
			if in(v) {
				return true
			}
		}
		return false
	case "NOT IN", "!=":
		for _, v := range have {
			if in(v) {
				return false
			}
		}
		return true
	}
	if len(want) == 0 {
		return false
	}
	w, err := strconv.ParseFloat(want[0], 64)
	if err != nil {
		return false
	}
	for _, v := range have {
		h, err := strconv.ParseFloat(v, 64)
		if err != nil {
			continue
		}
		if (op == "<" && h < w) || (op == ">" && h > w) || (op == "<=" && h <= w) || (op == ">=" && h >= w) {
			return true
		}
	}
	return false
}
