package dynamodb

import (
	"encoding/json"
	"fmt"
	"sort"
)

// pendingWrite is one validated item write; item nil means delete.
type pendingWrite struct {
	table *Table
	key   string
	item  map[string]any
}

// commitWritesLocked applies validated writes in order, recording stream
// records, and refreshes the metrics of every table it touched.
func (s *Service) commitWritesLocked(writes []pendingWrite) {
	touched := map[*Table]bool{}
	for _, w := range writes {
		if w.item == nil {
			s.removeItemLocked(w.table, w.key)
		} else {
			s.writeItemLocked(w.table, w.key, w.item)
		}
		touched[w.table] = true
	}
	for table := range touched {
		refreshTableMetrics(table)
	}
}

// claimItem rejects a second operation on the same item within one request.
func claimItem(seen map[string]bool, table *Table, key, msg string) error {
	id := table.description.TableName + "\x00" + key
	if seen[id] {
		return fmt.Errorf("%w: %s", ErrValidation, msg)
	}
	seen[id] = true
	return nil
}

// BatchWriteItem validates every request before writing any, so a bad entry
// leaves the tables (and their streams) untouched. Tables are processed in
// name order so stream sequence numbers are deterministic.
func (s *Service) BatchWriteItem(requestItems map[string]any) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	names := make([]string, 0, len(requestItems))
	for name := range requestItems {
		names = append(names, name)
	}
	sort.Strings(names)

	const dup = "Provided list of item keys contains duplicates"
	seen := map[string]bool{}
	var writes []pendingWrite
	for _, tableName := range names {
		table, err := s.requireTable(tableName)
		if err != nil {
			return nil, err
		}
		requests, ok := requestItems[tableName].([]any)
		if !ok {
			return nil, ErrValidation
		}
		for _, rawReq := range requests {
			req, ok := rawReq.(map[string]any)
			if !ok {
				return nil, ErrValidation
			}
			w := pendingWrite{table: table}
			if putReq, ok := req["PutRequest"].(map[string]any); ok {
				w.item, _ = putReq["Item"].(map[string]any)
				if len(w.item) == 0 {
					return nil, ErrValidation
				}
				if w.key, err = table.itemKey(w.item); err != nil {
					return nil, err
				}
			} else if delReq, ok := req["DeleteRequest"].(map[string]any); ok {
				keySpec, _ := delReq["Key"].(map[string]any)
				if w.key, err = table.lookupKey(keySpec); err != nil {
					return nil, err
				}
			} else {
				return nil, ErrValidation
			}
			if err := claimItem(seen, table, w.key, dup); err != nil {
				return nil, err
			}
			writes = append(writes, w)
		}
	}
	s.commitWritesLocked(writes)
	return map[string]any{}, nil
}

// TransactWriteItems is all-or-nothing: every entry is resolved against the
// state before the transaction, and nothing is written unless all succeed.
// ponytail: Put/Update/Delete ConditionExpression is not evaluated (pre-existing).
func (s *Service) TransactWriteItems(transactItems []map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	const dup = "Transaction request cannot include multiple operations on one item"
	seen := map[string]bool{}
	var writes []pendingWrite
	for _, entry := range transactItems {
		kind, req := transactEntry(entry)
		if req == nil {
			return ErrValidation
		}
		table, err := s.requireTable(stringValue(req["TableName"]))
		if err != nil {
			return err
		}
		w := pendingWrite{table: table}
		keySpec, _ := req["Key"].(map[string]any)
		switch kind {
		case "Put":
			w.item, _ = req["Item"].(map[string]any)
			if len(w.item) == 0 {
				return ErrValidation
			}
			w.key, err = table.itemKey(w.item)
		default:
			w.key, err = table.lookupKey(keySpec)
		}
		if err != nil {
			return err
		}
		if err := claimItem(seen, table, w.key, dup); err != nil {
			return err
		}
		switch kind {
		case "Update":
			w.item = cloneItem(table.items[w.key])
			if w.item == nil {
				w.item = cloneItem(keySpec)
			}
			err = applyUpdateExpression(
				w.item,
				stringValue(req["UpdateExpression"]),
				toStringMap(req["ExpressionAttributeNames"]),
				toAnyMap(req["ExpressionAttributeValues"]),
			)
			if err != nil {
				return err
			}
		case "ConditionCheck":
			if table.items[w.key] == nil {
				return ErrConditionalCheckFailed
			}
			continue // checks write nothing
		}
		writes = append(writes, w)
	}
	s.commitWritesLocked(writes)
	return nil
}

func transactEntry(entry map[string]any) (string, map[string]any) {
	for _, kind := range []string{"Put", "Delete", "Update", "ConditionCheck"} {
		if req, ok := entry[kind].(map[string]any); ok {
			return kind, req
		}
	}
	return "", nil
}

// sameItem compares two items, treating SS/NS/BS sets as unordered.
func sameItem(a, b map[string]any) bool {
	ea, errA := json.Marshal(normalizeSets(a))
	eb, errB := json.Marshal(normalizeSets(b))
	return errA == nil && errB == nil && string(ea) == string(eb)
}

func normalizeSets(v any) any {
	switch value := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(value))
		for k, inner := range value {
			if list, ok := inner.([]any); ok && len(value) == 1 && (k == "SS" || k == "NS" || k == "BS") {
				sorted := make([]string, 0, len(list))
				for _, el := range list {
					sorted = append(sorted, fmt.Sprint(el))
				}
				sort.Strings(sorted)
				out[k] = sorted
				continue
			}
			out[k] = normalizeSets(inner)
		}
		return out
	case []any:
		out := make([]any, len(value))
		for i, el := range value {
			out[i] = normalizeSets(el)
		}
		return out
	default:
		return v
	}
}
