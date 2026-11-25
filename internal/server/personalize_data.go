package server

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Dataset contents: items, users and actions keyed by id, and interaction
// events. Records arrive by bulk import (CSV in S3) or the Personalize Events
// Put* APIs; exports pick them by that source.

const (
	pszInteractions       = "Interactions"
	pszItems              = "Items"
	pszUsers              = "Users"
	pszActions            = "Actions"
	pszActionInteractions = "Action_Interactions"
	pszMaxSchemaFields    = 1000
)

// pszDatasetTypes maps a lower-cased dataset type to its canonical form and
// the schema fields it requires.
var pszDatasetTypes = map[string]struct {
	name     string
	required []string
}{
	"interactions":        {pszInteractions, []string{"USER_ID", "ITEM_ID", "TIMESTAMP"}},
	"items":               {pszItems, []string{"ITEM_ID"}},
	"users":               {pszUsers, []string{"USER_ID"}},
	"actions":             {pszActions, []string{"ACTION_ID"}},
	"action_interactions": {pszActionInteractions, []string{"USER_ID", "ACTION_ID", "TIMESTAMP", "EVENT_TYPE"}},
}

// pszIDField is the key column of an entity dataset.
var pszIDField = map[string]string{pszItems: "ITEM_ID", pszUsers: "USER_ID", pszActions: "ACTION_ID"}

type pszEntity struct {
	props map[string]string // schema field -> value
	bulk  bool
}

type pszEvent struct {
	user, target, eventType string // target: ITEM_ID or ACTION_ID
	at                      time.Time
	bulk                    bool
}

type pszData struct {
	typ      string
	fields   []string
	entities map[string]*pszEntity
	events   []pszEvent
}

func newPSZData(typ string, fields []string) *pszData {
	return &pszData{typ: typ, fields: fields, entities: map[string]*pszEntity{}}
}

func (d *pszData) isEvents() bool { return d.typ == pszInteractions || d.typ == pszActionInteractions }

// addEvents appends es and then drops the oldest events past pszMaxEvents
// with a single copy, so a large import stays linear.
func (d *pszData) addEvents(es []pszEvent) {
	d.events = append(d.events, es...)
	if over := len(d.events) - pszMaxEvents; over > 0 {
		d.events = append([]pszEvent(nil), d.events[over:]...)
	}
}

func (d *pszData) putEntity(id string, props map[string]string, bulk bool) *pszErr {
	if d.entities[id] == nil && len(d.entities) >= pszMaxRecords {
		return pszInvalid("Stackyard keeps at most %d records per dataset.", pszMaxRecords)
	}
	d.entities[id] = &pszEntity{props: props, bulk: bulk}
	return nil
}

var pszFieldName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// pszParseSchema checks an Avro record schema and returns its field names.
func pszParseSchema(raw string) ([]string, *pszErr) {
	var doc struct {
		Type   any `json:"type"`
		Name   string
		Fields []struct {
			Name string `json:"name"`
			Type any    `json:"type"`
		} `json:"fields"`
	}
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		return nil, pszInvalid("The schema isn't valid JSON: %v", err)
	}
	if doc.Type != "record" || doc.Name == "" {
		return nil, pszInvalid(`The schema must be an Avro record with "type": "record" and a name.`)
	}
	if len(doc.Fields) == 0 || len(doc.Fields) > pszMaxSchemaFields {
		return nil, pszInvalid("The schema must have between 1 and %d fields.", pszMaxSchemaFields)
	}
	seen := map[string]bool{}
	fields := make([]string, 0, len(doc.Fields))
	for _, f := range doc.Fields {
		if !pszFieldName.MatchString(f.Name) || f.Type == nil {
			return nil, pszInvalid("Schema field %q needs a valid name and a type.", f.Name)
		}
		if seen[strings.ToUpper(f.Name)] {
			return nil, pszInvalid("Schema field %s is defined twice.", f.Name)
		}
		seen[strings.ToUpper(f.Name)] = true
		fields = append(fields, f.Name)
	}
	return fields, nil
}

// pszHasFields reports the first required field the schema lacks.
func pszLacks(fields, required []string) string {
	for _, want := range required {
		if pszField(fields, want) == "" {
			return want
		}
	}
	return ""
}

// pszField finds a schema field by name, ignoring case and underscores
// (Put* properties use camelCase: numberOfRatings is NUMBER_OF_RATINGS).
func pszField(fields []string, name string) string {
	norm := func(s string) string { return strings.ToUpper(strings.ReplaceAll(s, "_", "")) }
	want := norm(name)
	for _, f := range fields {
		if norm(f) == want {
			return f
		}
	}
	return ""
}

// pszProperties decodes a Put* properties JSON object into schema fields.
func pszProperties(raw string, fields []string, idField string) (map[string]string, *pszErr) {
	props := map[string]string{}
	if raw == "" {
		return props, nil
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(raw), &obj); err != nil {
		return nil, pszInvalid("properties must be a JSON object.")
	}
	for k, v := range obj {
		f := pszField(fields, k)
		if f == "" || strings.EqualFold(f, idField) {
			return nil, pszInvalid("Property %s isn't a field of the dataset schema.", k)
		}
		switch t := v.(type) {
		case string:
			props[strings.ToUpper(f)] = t
		case float64:
			props[strings.ToUpper(f)] = strconv.FormatFloat(t, 'f', -1, 64)
		case bool:
			props[strings.ToUpper(f)] = strconv.FormatBool(t)
		case nil:
		default:
			return nil, pszInvalid("Property %s must be a string, number or boolean.", k)
		}
	}
	return props, nil
}

// pszCSVRecords parses a bulk import file for dataset d. FULL imports
// replace the bulk records; INCREMENTAL ones add to them.
type pszImport struct {
	entities map[string]map[string]string
	events   []pszEvent
}

func pszParseCSV(body []byte, d *pszData) (*pszImport, error) {
	rd := csv.NewReader(bytes.NewReader(body))
	rd.FieldsPerRecord = -1
	header, err := rd.Read()
	if err != nil {
		return nil, fmt.Errorf("the file has no CSV header")
	}
	cols := make([]string, len(header))
	for i, h := range header {
		f := pszField(d.fields, strings.TrimSpace(h))
		if f == "" {
			return nil, fmt.Errorf("column %s isn't a field of the dataset schema", h)
		}
		cols[i] = strings.ToUpper(f)
	}
	required, _ := pszDatasetTypes[strings.ToLower(d.typ)]
	for _, want := range required.required {
		if pszField(cols, want) == "" {
			return nil, fmt.Errorf("the file has no %s column", want)
		}
	}
	out := &pszImport{entities: map[string]map[string]string{}}
	for rows := 0; ; rows++ {
		rec, err := rd.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("row %d: %v", rows+2, err)
		}
		if rows >= pszMaxRecords {
			return nil, fmt.Errorf("Stackyard imports at most %d rows per file", pszMaxRecords)
		}
		row := map[string]string{}
		for i, v := range rec {
			if i < len(cols) {
				row[cols[i]] = v
			}
		}
		if d.isEvents() {
			e, err := pszRowEvent(row, d.typ)
			if err != nil {
				return nil, fmt.Errorf("row %d: %v", rows+2, err)
			}
			out.events = append(out.events, e)
			continue
		}
		id := row[pszIDField[d.typ]]
		if id == "" {
			return nil, fmt.Errorf("row %d has no %s", rows+2, pszIDField[d.typ])
		}
		delete(row, pszIDField[d.typ])
		out.entities[id] = row
	}
	return out, nil
}

func pszRowEvent(row map[string]string, typ string) (pszEvent, error) {
	target := row["ITEM_ID"]
	if typ == pszActionInteractions {
		target = row["ACTION_ID"]
	}
	sec, err := strconv.ParseInt(row["TIMESTAMP"], 10, 64)
	if err != nil || row["USER_ID"] == "" || target == "" {
		return pszEvent{}, fmt.Errorf("needs a user, a target id and an epoch-seconds TIMESTAMP")
	}
	return pszEvent{user: row["USER_ID"], target: target, eventType: row["EVENT_TYPE"], at: time.Unix(sec, 0).UTC(), bulk: true}, nil
}

// apply merges an import into d.
func (d *pszData) apply(imp *pszImport, full bool) *pszErr {
	if full {
		for id, e := range d.entities {
			if e.bulk {
				delete(d.entities, id)
			}
		}
		kept := d.events[:0:0]
		for _, e := range d.events {
			if !e.bulk {
				kept = append(kept, e)
			}
		}
		d.events = kept
	}
	ids := make([]string, 0, len(imp.entities))
	for id := range imp.entities {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if err := d.putEntity(id, imp.entities[id], true); err != nil {
			return err
		}
	}
	d.addEvents(imp.events)
	return nil
}

// export renders d as CSV, keeping the records whose source mode selects.
func (d *pszData) export(mode string) []byte {
	keep := func(bulk bool) bool { return mode == "ALL" || (mode == "BULK") == bulk }
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	cols := make([]string, len(d.fields))
	for i, f := range d.fields {
		cols[i] = strings.ToUpper(f)
	}
	_ = w.Write(cols)
	if d.isEvents() {
		for _, e := range d.events {
			if !keep(e.bulk) {
				continue
			}
			vals := map[string]string{"USER_ID": e.user, "ITEM_ID": e.target, "ACTION_ID": e.target,
				"EVENT_TYPE": e.eventType, "TIMESTAMP": strconv.FormatInt(e.at.Unix(), 10)}
			row := make([]string, len(cols))
			for i, c := range cols {
				row[i] = vals[c]
			}
			_ = w.Write(row)
		}
	} else {
		ids := make([]string, 0, len(d.entities))
		for id := range d.entities {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			e := d.entities[id]
			if !keep(e.bulk) {
				continue
			}
			row := make([]string, len(cols))
			for i, c := range cols {
				if c == pszIDField[d.typ] {
					row[i] = id
				} else {
					row[i] = e.props[c]
				}
			}
			_ = w.Write(row)
		}
	}
	w.Flush()
	return buf.Bytes()
}
