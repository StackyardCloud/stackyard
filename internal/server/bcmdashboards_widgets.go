package server

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Widget and query checks for BCM Dashboards. The model's name and title
// patterns use look-aheads RE2 can't compile, so their "no double space"
// rule is checked here; everything else the model covers is already
// validated by budServe.

const (
	bcmdGridColumns   = 6  // widget width + horizontal offset fit in this
	bcmdMaxExprDepth  = 10 // nested and/or/not levels in a filter
	bcmdMaxGroupBy    = 2  // Cost Explorer groups by at most two keys
	bcmdDefaultWidth  = 4
	bcmdDefaultHeight = 4
)

var bcmdDimensions = map[string]bool{
	"AZ": true, "INSTANCE_TYPE": true, "LINKED_ACCOUNT": true, "OPERATION": true, "PURCHASE_TYPE": true, "REGION": true,
	"SERVICE": true, "USAGE_TYPE": true, "USAGE_TYPE_GROUP": true, "RECORD_TYPE": true, "RESOURCE_ID": true,
	"SUBSCRIPTION_ID": true, "TAG_KEY": true, "OPERATING_SYSTEM": true, "TENANCY": true, "BILLING_ENTITY": true,
	"RESERVATION_ID": true, "COST_CATEGORY_NAME": true, "DATABASE_ENGINE": true, "LEGAL_ENTITY_NAME": true,
	"SAVINGS_PLANS_TYPE": true, "INSTANCE_TYPE_FAMILY": true, "CACHE_ENGINE": true, "DEPLOYMENT_OPTION": true,
	"SCOPE": true, "PLATFORM": true,
}

// bcmAsMap reads a list member as an object; a null member reads as nil,
// which the callers' checks then refuse.
func bcmAsMap(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func bcmdCheckName(field, v string) *budErr {
	if strings.Contains(v, "  ") {
		return bcmdInvalid("%s must not contain consecutive spaces", field)
	}
	return nil
}

// bcmdCheckWidgets validates widgets and returns copies with ids, width and
// height filled in.
func bcmdCheckWidgets(widgets []any) ([]any, *budErr) {
	out := make([]any, 0, len(widgets))
	seen := map[string]bool{}
	for i, item := range widgets {
		if bcmAsMap(item) == nil {
			return nil, bcmdInvalid("widget %d must be an object", i)
		}
		w := budCloneMap(bcmAsMap(item))
		if err := bcmdCheckName("widgets.title", budStr(w, "title")); err != nil {
			return nil, err
		}
		id := budStr(w, "id")
		if id == "" {
			id = strings.ReplaceAll(budNewID(), "-", "")
			w["id"] = id
		}
		if seen[id] {
			return nil, bcmdInvalid("widget id %s is used twice", id)
		}
		seen[id] = true
		width := budInt(w, "width", bcmdDefaultWidth)
		w["width"], w["height"] = float64(width), float64(budInt(w, "height", bcmdDefaultHeight))
		offset, _ := w["horizontalOffset"].(float64)
		if offset < 0 || int(offset)+width > bcmdGridColumns {
			return nil, bcmdInvalid("widget %d: horizontalOffset plus width must fit in %d columns", i, bcmdGridColumns)
		}
		w["horizontalOffset"] = offset
		for _, c := range budList(w, "configs") {
			if err := bcmdCheckConfig(bcmAsMap(c)); err != nil {
				return nil, bcmdInvalid("widget %d: %s", i, err.msg)
			}
		}
		out = append(out, w)
	}
	return out, nil
}

// bcmdOneOf checks that a union carries exactly one member and returns it.
func bcmdOneOf(union map[string]any, what string) (string, map[string]any, *budErr) {
	if len(union) != 1 {
		return "", nil, bcmdInvalid("%s must set exactly one member", what)
	}
	for k, v := range union {
		m, _ := v.(map[string]any)
		return k, m, nil
	}
	return "", nil, nil
}

func bcmdCheckConfig(c map[string]any) *budErr {
	kind, q, err := bcmdOneOf(budMap(c, "queryParameters"), "queryParameters")
	if err != nil {
		return err
	}
	if _, _, err := bcmdOneOf(budMap(c, "displayConfig"), "displayConfig"); err != nil {
		return err
	}
	if kind == "costAndUsage" {
		metrics := budStrings(q, "metrics")
		if len(metrics) == 0 {
			return bcmdInvalid("costAndUsage needs at least one metric")
		}
		seen := map[string]bool{}
		for _, m := range metrics {
			if seen[m] {
				return bcmdInvalid("metric %s is listed twice", m)
			}
			seen[m] = true
		}
	}
	if err := bcmdCheckRange(budMap(q, "timeRange")); err != nil {
		return err
	}
	groups := budList(q, "groupBy")
	if len(groups) > bcmdMaxGroupBy {
		return bcmdInvalid("groupBy takes at most %d groups", bcmdMaxGroupBy)
	}
	for _, g := range groups {
		gm, _ := g.(map[string]any)
		if t := budStr(gm, "type"); (t == "" || t == "DIMENSION") && !bcmdDimensions[budStr(gm, "key")] {
			return bcmdInvalid("groupBy key %s is not a dimension", budStr(gm, "key"))
		}
	}
	if f, ok := q["filter"].(map[string]any); ok {
		return bcmdCheckExpression(f, 0)
	}
	return nil
}

// bcmdCheckExpression applies Cost Explorer's filter rules: each level sets
// exactly one operator, and/or combine at least two expressions.
func bcmdCheckExpression(e map[string]any, depth int) *budErr {
	if depth > bcmdMaxExprDepth {
		return bcmdInvalid("filter expressions nest at most %d levels", bcmdMaxExprDepth)
	}
	op, _, err := bcmdOneOf(e, "each filter expression")
	if err != nil {
		return err
	}
	switch op {
	case "and", "or":
		list := budList(e, op)
		if len(list) < 2 {
			return bcmdInvalid("%s needs at least two expressions", op)
		}
		for _, sub := range list {
			if err := bcmdCheckExpression(bcmAsMap(sub), depth+1); err != nil {
				return err
			}
		}
	case "not":
		return bcmdCheckExpression(budMap(e, "not"), depth+1)
	default:
		values := budMap(e, op)
		if budStr(values, "key") == "" {
			return bcmdInvalid("%s filter needs a key", op)
		}
		if len(budList(values, "values")) == 0 && !strings.Contains(strings.Join(budStrings(values, "matchOptions"), ","), "ABSENT") {
			return bcmdInvalid("%s filter needs values unless it matches ABSENT", op)
		}
	}
	return nil
}

// bcmdRelative matches ISO 8601 durations such as -P7D or -P3M (RELATIVE
// date-time values count back, or forward, from now).
var bcmdRelative = regexp.MustCompile(`^(-)?P(?:(\d{1,4})Y)?(?:(\d{1,4})M)?(?:(\d{1,4})W)?(?:(\d{1,4})D)?$`)

func bcmdResolve(v map[string]any, now time.Time) (time.Time, *budErr) {
	value := budStr(v, "value")
	if budStr(v, "type") == "ABSOLUTE" {
		t, ok := budTime(value)
		if !ok {
			return time.Time{}, bcmdInvalid("ABSOLUTE value %q is not an ISO 8601 date-time", value)
		}
		return t, nil
	}
	m := bcmdRelative.FindStringSubmatch(value)
	if m == nil || value == "P" || value == "-P" {
		return time.Time{}, bcmdInvalid("RELATIVE value %q is not an ISO 8601 duration such as -P7D", value)
	}
	n := func(s string) int {
		v, _ := strconv.ParseInt(s, 10, 32)
		return int(v)
	}
	sign := 1
	if m[1] == "-" {
		sign = -1
	}
	return now.AddDate(sign*n(m[2]), sign*n(m[3]), sign*(7*n(m[4])+n(m[5]))), nil
}

func bcmdCheckRange(r map[string]any) *budErr {
	if r == nil {
		return nil
	}
	now := time.Now().UTC()
	start, err := bcmdResolve(budMap(r, "startTime"), now)
	if err != nil {
		return err
	}
	end, err := bcmdResolve(budMap(r, "endTime"), now)
	if err != nil {
		return err
	}
	if end.Before(start) {
		return bcmdInvalid("timeRange endTime is before startTime")
	}
	return nil
}
