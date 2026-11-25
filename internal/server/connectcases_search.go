package server

import (
	"sort"
	"strings"
)

// Case search. Filters are checked before the store lock is taken: at most
// cncMaxFilterNodes nodes nested at most cncMaxFilterDepth deep, so the work
// under the lock is bounded by (cases x filter nodes) plus one pass of the
// search term over the domain's string values (bounded by cncMaxStoreBytes).

const (
	cncMaxFilterNodes = 20
	cncMaxFilterDepth = 5
)

var cncComparisons = map[string]bool{"equalTo": true, "contains": true, "greaterThan": true, "greaterThanOrEqualTo": true, "lessThan": true, "lessThanOrEqualTo": true}

// cncCheckFilter validates a CaseFilter (withTags) or CustomFieldsFilter tree.
func cncCheckFilter(f map[string]any, depth int, budget *int, withTags bool) *cncErr {
	*budget--
	if *budget < 0 {
		return cncValidation("a filter has at most %d conditions", cncMaxFilterNodes)
	}
	if depth > cncMaxFilterDepth {
		return cncValidation("filters nest at most %d deep", cncMaxFilterDepth)
	}
	kind, body, err := cncUnion(f, "filter")
	if err != nil {
		return err
	}
	switch kind {
	case "field":
		op, leaf, err := cncUnion(body.(map[string]any), "field filter")
		if err != nil {
			return err
		}
		if !cncComparisons[op] {
			return cncValidation("unknown comparison %s", op)
		}
		_, _, err = cncUnion(cncMap(leaf.(map[string]any), "value"), "filter value")
		return err
	case "tag":
		if !withTags {
			return cncValidation("tag filters apply to cases only")
		}
		_, _, err := cncUnion(body.(map[string]any), "tag filter")
		return err
	case "not":
		return cncCheckFilter(body.(map[string]any), depth+1, budget, withTags)
	case "andAll", "orAll":
		for _, sub := range body.([]any) {
			if err := cncCheckFilter(sub.(map[string]any), depth+1, budget, withTags); err != nil {
				return err
			}
		}
		return nil
	}
	return cncValidation("unknown filter %s", kind)
}

func cncCheckFieldsFilter(f map[string]any, depth int, budget *int) *cncErr {
	return cncCheckFilter(f, depth, budget, false)
}

func cncEvalFieldsFilter(f map[string]any, values map[string]map[string]any) bool {
	return cncEvalFilter(f, values, nil)
}

// cncEvalFilter evaluates a checked filter against field values and tags.
func cncEvalFilter(f map[string]any, values map[string]map[string]any, tags map[string]string) bool {
	for kind, body := range f {
		switch kind {
		case "field":
			for op, leaf := range body.(map[string]any) {
				m := leaf.(map[string]any)
				return cncCompare(values[cncStr(m, "id")], op, cncMap(m, "value"))
			}
		case "tag":
			eq := cncMap(body.(map[string]any), "equalTo")
			v, ok := tags[cncStr(eq, "key")]
			return ok && (eq["value"] == nil || eq["value"] == v)
		case "not":
			return !cncEvalFilter(body.(map[string]any), values, tags)
		case "andAll":
			for _, sub := range body.([]any) {
				if !cncEvalFilter(sub.(map[string]any), values, tags) {
					return false
				}
			}
			return true
		case "orAll":
			for _, sub := range body.([]any) {
				if cncEvalFilter(sub.(map[string]any), values, tags) {
					return true
				}
			}
			return false
		}
	}
	return false
}

func cncSameValue(cur, want map[string]any) bool {
	if want["emptyValue"] != nil {
		return cncEmpty(cur)
	}
	for k, v := range want {
		return cur != nil && cur[k] == v
	}
	return false
}

// cncCompare applies one comparison: numbers by value, strings by
// (case-insensitive) order or substring.
func cncCompare(cur map[string]any, op string, want map[string]any) bool {
	if op == "equalTo" {
		return cncSameValue(cur, want)
	}
	if cur == nil {
		return false
	}
	var c int
	switch {
	case want["doubleValue"] != nil:
		a, ok1 := cur["doubleValue"].(float64)
		b, ok2 := want["doubleValue"].(float64)
		if !ok1 || !ok2 {
			return false
		}
		if a < b {
			c = -1
		} else if a > b {
			c = 1
		}
	case want["stringValue"] != nil:
		a, ok1 := cur["stringValue"].(string)
		b, _ := want["stringValue"].(string)
		if !ok1 {
			return false
		}
		if op == "contains" {
			return strings.Contains(strings.ToLower(a), strings.ToLower(b))
		}
		c = strings.Compare(a, b)
	default:
		return false
	}
	switch op {
	case "greaterThan":
		return c > 0
	case "greaterThanOrEqualTo":
		return c >= 0
	case "lessThan":
		return c < 0
	case "lessThanOrEqualTo":
		return c <= 0
	}
	return false
}

// cncTermMatch reports whether any string value contains the lower-cased term.
func (c *cncCase) termMatch(term string) bool {
	for _, v := range c.fields {
		if s, ok := v["stringValue"].(string); ok && len(s) >= len(term) && strings.Contains(strings.ToLower(s), term) {
			return true
		}
	}
	return false
}

func (st *cncStore) searchCases(in map[string]any) (map[string]any, *cncErr) {
	filter := cncMap(in, "filter")
	if filter != nil {
		budget := cncMaxFilterNodes
		if err := cncCheckFilter(filter, 0, &budget, true); err != nil {
			return nil, err
		}
	}
	term := strings.ToLower(strings.TrimSpace(cncStr(in, "searchTerm")))
	st.mu.Lock()
	defer st.mu.Unlock()
	d, err := st.domainLocked(cncStr(in, "domainId"))
	if err != nil {
		return nil, err
	}
	hits := []*cncCase{}
	for _, c := range d.cases {
		if term != "" && !c.termMatch(term) {
			continue
		}
		if filter != nil && !cncEvalFilter(filter, c.fields, st.tags[c.arn]) {
			continue
		}
		hits = append(hits, c)
	}
	sort.Slice(hits, func(i, j int) bool { return hits[i].seq > hits[j].seq })
	sorts := cncList(in, "sorts")
	for k := len(sorts) - 1; k >= 0; k-- {
		s := sorts[k].(map[string]any)
		id, desc := cncStr(s, "fieldId"), s["sortOrder"] == "Desc"
		sort.SliceStable(hits, func(i, j int) bool {
			a, b := hits[i].fields[id], hits[j].fields[id]
			if a == nil || b == nil {
				return a != nil && b == nil // unset values sort last
			}
			if cncCompare(a, "lessThan", b) {
				return !desc
			}
			if cncCompare(a, "greaterThan", b) {
				return desc
			}
			return false
		})
	}
	start, end, next, err := cncPage(in, len(hits), 10, 100)
	if err != nil {
		return nil, err
	}
	requested := cncList(in, "fields")
	out := []any{}
	for _, c := range hits[start:end] {
		out = append(out, map[string]any{"caseId": c.id, "templateId": c.templateID, "fields": c.fieldsOut(requested), "tags": st.tagsOut(c.arn)})
	}
	res := cncWithNext(map[string]any{"cases": out}, next)
	res["totalCount"] = len(hits)
	return res, nil
}
