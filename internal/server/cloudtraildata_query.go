package server

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
)

// A small CloudTrail Lake SQL subset over events ingested with PutAuditEvents:
//
//	SELECT col[, col...] | * FROM <event data store id|arn> [WHERE col = 'v' [AND ...]] [LIMIT n]
//
// Columns are top-level Lake columns (eventID, eventTime, ...) or dotted paths into
// eventData (eventData.eventName, eventData.userIdentity.principalId).
// ponytail: equality-AND filters only; add operators when a caller needs them.
var (
	cloudTrailQuerySelectRe = regexp.MustCompile(`(?is)^\s*select\s+(.+?)\s+from\s+(\S+)(.*)$`)
	cloudTrailQueryRestRe   = regexp.MustCompile(`(?is)^\s*(?:where\s+(.+?))?\s*(?:limit\s+(\d+))?\s*;?\s*$`)
	cloudTrailQueryCondRe   = regexp.MustCompile(`^\s*([A-Za-z0-9_.]+)\s*=\s*'((?:[^']|'')*)'\s*$`)
	cloudTrailQueryAndRe    = regexp.MustCompile(`(?i)\s+and\s+`)
	cloudTrailQueryColumnRe = regexp.MustCompile(`^[A-Za-z0-9_.]+$`)
)

type cloudTrailQueryCond struct{ column, value string }

// runAuditQueryLocked evaluates a query against ingested audit events. matched is false
// when the FROM target is not a known event data store, so callers keep the stub result.
func (s *cloudTrailStore) runAuditQueryLocked(statement string) (rows []any, matched bool, errMsg string) {
	m := cloudTrailQuerySelectRe.FindStringSubmatch(statement)
	if m == nil {
		return nil, false, ""
	}
	store := cloudTrailNameFromARN(strings.Trim(m[2], "`\"';"))
	if _, ok := s.eventStores[store]; !ok {
		if _, ok := s.auditEvents[store]; !ok {
			return nil, false, ""
		}
	}

	rest := cloudTrailQueryRestRe.FindStringSubmatch(m[3])
	if rest == nil {
		return nil, true, "unsupported query clause: " + strings.TrimSpace(m[3])
	}
	var conds []cloudTrailQueryCond
	if where := strings.TrimSpace(rest[1]); where != "" {
		// ponytail: splitting on AND ignores quoting, so a literal containing " and " is unsupported.
		for _, part := range cloudTrailQueryAndRe.Split(where, -1) {
			cm := cloudTrailQueryCondRe.FindStringSubmatch(part)
			if cm == nil {
				return nil, true, "unsupported WHERE condition: " + strings.TrimSpace(part)
			}
			conds = append(conds, cloudTrailQueryCond{cm[1], strings.ReplaceAll(cm[2], "''", "'")})
		}
	}
	limit := -1
	if rest[2] != "" {
		limit, _ = strconv.Atoi(rest[2])
	}

	columns, errMsg := cloudTrailQueryColumns(m[1])
	if errMsg != "" {
		return nil, true, errMsg
	}

	rows = []any{}
	for _, record := range s.auditEvents[store] {
		if limit >= 0 && len(rows) >= limit {
			break
		}
		values := record.columns()
		if !cloudTrailQueryMatches(values, conds) {
			continue
		}
		cols := columns
		if cols == nil {
			cols = sortedKeysAny(values)
		}
		row := make([]any, 0, len(cols))
		for _, col := range cols {
			row = append(row, map[string]any{col: cloudTrailQueryValue(values, col)})
		}
		rows = append(rows, row)
	}
	return rows, true, ""
}

// cloudTrailQueryColumns returns nil for SELECT *.
func cloudTrailQueryColumns(list string) ([]string, string) {
	if strings.TrimSpace(list) == "*" {
		return nil, ""
	}
	parts := strings.Split(list, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		col := strings.TrimSpace(part)
		if !cloudTrailQueryColumnRe.MatchString(col) {
			return nil, "unsupported select expression: " + col
		}
		out = append(out, col)
	}
	return out, ""
}

func cloudTrailQueryMatches(values map[string]any, conds []cloudTrailQueryCond) bool {
	for _, cond := range conds {
		if cloudTrailQueryValue(values, cond.column) != cond.value {
			return false
		}
	}
	return true
}

func cloudTrailQueryValue(values map[string]any, path string) string {
	var current any = values
	for _, key := range strings.Split(path, ".") {
		obj, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		current = obj[key]
	}
	switch typed := current.(type) {
	case nil:
		return ""
	case string:
		return typed
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(typed)
	default:
		encoded, _ := json.Marshal(typed)
		return string(encoded)
	}
}
