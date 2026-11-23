package server

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode"
)

const (
	schOpenAPIVersion = "3.0.0"
	schDraft4         = "http://json-schema.org/draft-04/schema#"
	schMaxDepth       = 64 // nesting levels walked when inferring or converting
	schRootName       = "AWSEvent"
)

var schBuiltinSchemas = map[string]string{
	"aws.ec2@EC2InstanceStateChangeNotification": `{"openapi":"3.0.0","info":{"version":"1.0.0","title":"EC2InstanceStateChangeNotification"},"paths":{},"components":{"schemas":{"AWSEvent":{"type":"object","required":["detail-type","source","detail"],"properties":{"detail":{"$ref":"#/components/schemas/EC2InstanceStateChangeNotification"},"detail-type":{"type":"string"},"source":{"type":"string"},"time":{"type":"string","format":"date-time"}}},"EC2InstanceStateChangeNotification":{"type":"object","properties":{"instance-id":{"type":"string"},"state":{"type":"string"}}}}}}`,
	"aws.s3@ObjectCreated":                       `{"openapi":"3.0.0","info":{"version":"1.0.0","title":"ObjectCreated"},"paths":{},"components":{"schemas":{"AWSEvent":{"type":"object","required":["detail-type","source","detail"],"properties":{"detail":{"$ref":"#/components/schemas/ObjectCreated"},"detail-type":{"type":"string"},"source":{"type":"string"},"time":{"type":"string","format":"date-time"}}},"ObjectCreated":{"type":"object","properties":{"bucket":{"type":"object","properties":{"name":{"type":"string"}}},"object":{"type":"object","properties":{"key":{"type":"string"},"size":{"type":"number"}}}}}}}}`,
}

// schCheckContent checks that content is an OpenAPI 3 document with component
// schemas, or a JSON Schema Draft 4 document.
func schCheckContent(typ, content string) *schError {
	var doc map[string]any
	if err := json.Unmarshal([]byte(content), &doc); err != nil {
		return schBadRequest("Content must be a JSON object: %v.", err)
	}
	switch typ {
	case "OpenApi3":
		v, _ := doc["openapi"].(string)
		if !strings.HasPrefix(v, "3.") {
			return schBadRequest("OpenApi3 content needs an openapi version of 3.x.")
		}
		comps, _ := doc["components"].(map[string]any)
		if _, ok := comps["schemas"].(map[string]any); !ok {
			return schBadRequest("OpenApi3 content needs components.schemas.")
		}
	case "JSONSchemaDraft4":
		if v, ok := doc["$schema"]; ok {
			if s, _ := v.(string); !strings.Contains(s, "draft-04") {
				return schBadRequest("JSONSchemaDraft4 content must use the draft-04 $schema.")
			}
		}
		if doc["type"] == nil && doc["properties"] == nil && doc["$ref"] == nil && doc["definitions"] == nil {
			return schBadRequest("JSONSchemaDraft4 content needs a type, properties, $ref or definitions.")
		}
	default:
		return schBadRequest("Type must be OpenApi3 or JSONSchemaDraft4.")
	}
	return nil
}

// schOpenAPIToJSONSchema turns an OpenAPI 3 document into a Draft 4 schema:
// the root component (AWSEvent, else the first by name) becomes the document
// and the other components become definitions.
func schOpenAPIToJSONSchema(title, content string) (string, *schError) {
	var doc map[string]any
	if err := json.Unmarshal([]byte(content), &doc); err != nil {
		return "", schBadRequest("Stored content isn't JSON.")
	}
	comps, _ := doc["components"].(map[string]any)
	schemas, _ := comps["schemas"].(map[string]any)
	if len(schemas) == 0 {
		return "", schBadRequest("The schema has no component schemas to export.")
	}
	root := schRootName
	if _, ok := schemas[root]; !ok {
		names := make([]string, 0, len(schemas))
		for n := range schemas {
			names = append(names, n)
		}
		sort.Strings(names)
		root = names[0]
	}
	out, _ := schRewriteRefs(schemas[root], 0).(map[string]any)
	if out == nil {
		out = map[string]any{}
	}
	defs := map[string]any{}
	for n, v := range schemas {
		if n != root {
			defs[n] = schRewriteRefs(v, 0)
		}
	}
	out["$schema"] = schDraft4
	out["title"] = title
	if len(defs) > 0 {
		out["definitions"] = defs
	}
	raw, _ := json.Marshal(out)
	return string(raw), nil
}

func schRewriteRefs(v any, depth int) any {
	if depth > schMaxDepth {
		return map[string]any{}
	}
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			if s, ok := val.(string); ok && k == "$ref" {
				out[k] = strings.Replace(s, "#/components/schemas/", "#/definitions/", 1)
				continue
			}
			out[k] = schRewriteRefs(val, depth+1)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, val := range t {
			out[i] = schRewriteRefs(val, depth+1)
		}
		return out
	}
	return v
}

// --- discovery ---

// schDiscover infers one schema covering every sample event.
// ponytail: nested objects are inlined, not split into separate components.
func schDiscover(events []string, typ string) (string, *schError) {
	var merged map[string]any
	for i, e := range events {
		dec := json.NewDecoder(strings.NewReader(e))
		dec.UseNumber()
		var v any
		if err := dec.Decode(&v); err != nil {
			return "", schBadRequest("Events[%d] isn't valid JSON.", i)
		}
		if _, ok := v.(map[string]any); !ok {
			return "", schBadRequest("Events[%d] must be a JSON object.", i)
		}
		s := schInfer(v, 0)
		if merged == nil {
			merged = s
		} else {
			merged = schMerge(merged, s, 0)
		}
	}
	var out map[string]any
	if typ == "OpenApi3" {
		out = map[string]any{"openapi": schOpenAPIVersion, "info": map[string]any{"version": "1.0.0", "title": "Event"}, "paths": map[string]any{},
			"components": map[string]any{"schemas": map[string]any{schRootName: merged}}}
	} else {
		merged["$schema"] = schDraft4
		merged["title"] = "Event"
		out = merged
	}
	raw, _ := json.Marshal(out)
	return string(raw), nil
}

func schInfer(v any, depth int) map[string]any {
	if depth > schMaxDepth {
		return map[string]any{}
	}
	switch t := v.(type) {
	case map[string]any:
		props := map[string]any{}
		required := make([]any, 0, len(t))
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			props[k] = schInfer(t[k], depth+1)
			required = append(required, k)
		}
		out := map[string]any{"type": "object", "properties": props}
		if len(required) > 0 {
			out["required"] = required
		}
		return out
	case []any:
		var items map[string]any
		for _, e := range t {
			s := schInfer(e, depth+1)
			if items == nil {
				items = s
			} else {
				items = schMerge(items, s, depth+1)
			}
		}
		out := map[string]any{"type": "array"}
		if items != nil {
			out["items"] = items
		}
		return out
	case json.Number:
		if _, err := t.Int64(); err == nil {
			return map[string]any{"type": "integer"}
		}
		return map[string]any{"type": "number"}
	case string:
		if _, err := time.Parse(time.RFC3339, t); err == nil {
			return map[string]any{"type": "string", "format": "date-time"}
		}
		return map[string]any{"type": "string"}
	case bool:
		return map[string]any{"type": "boolean"}
	}
	return map[string]any{"type": "null"}
}

// schMerge widens two inferred schemas into one that accepts both: objects
// union their properties (required keeps only keys present in both), integer
// and number become number, null yields to the other type, and anything else
// that disagrees becomes unconstrained.
func schMerge(a, b map[string]any, depth int) map[string]any {
	if depth > schMaxDepth {
		return map[string]any{}
	}
	ta, _ := a["type"].(string)
	tb, _ := b["type"].(string)
	switch {
	case ta == "null":
		return b
	case tb == "null":
		return a
	case ta == "" || tb == "":
		return map[string]any{}
	case ta != tb:
		if (ta == "integer" && tb == "number") || (ta == "number" && tb == "integer") {
			return map[string]any{"type": "number"}
		}
		return map[string]any{}
	case ta == "object":
		pa, _ := a["properties"].(map[string]any)
		pb, _ := b["properties"].(map[string]any)
		props := map[string]any{}
		for k, v := range pa {
			props[k] = v
		}
		for k, v := range pb {
			if cur, ok := props[k].(map[string]any); ok {
				props[k] = schMerge(cur, v.(map[string]any), depth+1)
			} else {
				props[k] = v
			}
		}
		inB := map[string]bool{}
		for _, k := range schStrings(b["required"]) {
			inB[k] = true
		}
		required := []any{}
		for _, k := range schStrings(a["required"]) {
			if inB[k] {
				required = append(required, k)
			}
		}
		out := map[string]any{"type": "object", "properties": props}
		if len(required) > 0 {
			out["required"] = required
		}
		return out
	case ta == "array":
		ia, _ := a["items"].(map[string]any)
		ib, _ := b["items"].(map[string]any)
		out := map[string]any{"type": "array"}
		switch {
		case ia != nil && ib != nil:
			out["items"] = schMerge(ia, ib, depth+1)
		case ia != nil:
			out["items"] = ia
		case ib != nil:
			out["items"] = ib
		}
		return out
	case ta == "string":
		if a["format"] != b["format"] {
			return map[string]any{"type": "string"}
		}
	}
	return a
}

func schStrings(v any) []string {
	list, _ := v.([]any)
	out := make([]string, 0, len(list))
	for _, e := range list {
		if s, ok := e.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// --- code bindings ---

type schField struct{ name, typ string }

// schGenerateBinding builds a zip holding one generated model type for the
// schema's root object.
// ponytail: only the root object's top-level properties are generated, not
// nested types.
func schGenerateBinding(lang, schemaName, typ, content string) ([]byte, *schError) {
	var doc map[string]any
	if err := json.Unmarshal([]byte(content), &doc); err != nil {
		return nil, schBadRequest("Stored content isn't JSON.")
	}
	root := doc
	if typ == "OpenApi3" {
		comps, _ := doc["components"].(map[string]any)
		schemas, _ := comps["schemas"].(map[string]any)
		if r, ok := schemas[schRootName].(map[string]any); ok {
			root = r
		} else {
			names := make([]string, 0, len(schemas))
			for n := range schemas {
				names = append(names, n)
			}
			sort.Strings(names)
			if len(names) > 0 {
				root, _ = schemas[names[0]].(map[string]any)
			}
		}
	}
	props, _ := root["properties"].(map[string]any)
	keys := make([]string, 0, len(props))
	for k := range props {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	fields := make([]schField, 0, len(keys))
	for _, k := range keys {
		p, _ := props[k].(map[string]any)
		t, _ := p["type"].(string)
		fields = append(fields, schField{name: k, typ: t})
	}
	class := schIdent(schemaName[strings.LastIndexAny(schemaName, "@.")+1:], true)

	var path, src string
	switch lang {
	case "Java8":
		path, src = "src/main/java/schema/"+class+".java", schJava(class, fields)
	case "Python36":
		path, src = "schema/"+schSnake(class)+".py", schPython(class, fields)
	case "TypeScript3":
		path, src = "lib/schema/"+class+".ts", schTypeScript(class, fields)
	case "Go1":
		path, src = "schema/"+schSnake(class)+".go", schGo(class, fields)
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	fixed := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC) // deterministic archive
	w, err := zw.CreateHeader(&zip.FileHeader{Name: path, Method: zip.Deflate, Modified: fixed})
	if err == nil {
		_, err = w.Write([]byte(src))
	}
	if err == nil {
		err = zw.Close()
	}
	if err != nil {
		return nil, schErr(500, "InternalServerErrorException", "Couldn't build the code binding archive.")
	}
	return buf.Bytes(), nil
}

// schIdent turns a JSON key into an identifier: letters and digits only,
// camel-cased at separators, prefixed with _ if it would start with a digit.
func schIdent(s string, upper bool) string {
	var b strings.Builder
	nextUpper := upper
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			nextUpper = true
			continue
		}
		if r > unicode.MaxASCII {
			continue
		}
		if nextUpper && b.Len() > 0 || (upper && b.Len() == 0) {
			r = unicode.ToUpper(r)
		}
		nextUpper = false
		b.WriteRune(r)
	}
	out := b.String()
	if out == "" {
		out = "Field"
	}
	if unicode.IsDigit(rune(out[0])) {
		out = "_" + out
	}
	return out
}

func schSnake(s string) string {
	var b strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) && i > 0 {
			b.WriteByte('_')
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}

func schTypeFor(lang, t string) string {
	table := map[string][4]string{ // Java, Python, TypeScript, Go
		"string":  {"String", "str", "string", "string"},
		"integer": {"Long", "int", "number", "int64"},
		"number":  {"Double", "float", "number", "float64"},
		"boolean": {"Boolean", "bool", "boolean", "bool"},
		"array":   {"java.util.List<Object>", "list", "any[]", "[]any"},
	}
	col := map[string]int{"Java8": 0, "Python36": 1, "TypeScript3": 2, "Go1": 3}[lang]
	if row, ok := table[t]; ok {
		return row[col]
	}
	return [4]string{"Object", "object", "any", "any"}[col]
}

func schJava(class string, fields []schField) string {
	var b strings.Builder
	fmt.Fprintf(&b, "package schema;\n\n// Generated by Stackyard from an EventBridge schema.\npublic class %s {\n", class)
	for _, f := range fields {
		fmt.Fprintf(&b, "    private %s %s;\n", schTypeFor("Java8", f.typ), schIdent(f.name, false))
	}
	for _, f := range fields {
		jt, id := schTypeFor("Java8", f.typ), schIdent(f.name, false)
		fmt.Fprintf(&b, "\n    public %s get%s() { return %s; }\n    public void set%s(%s value) { this.%s = value; }\n", jt, schIdent(f.name, true), id, schIdent(f.name, true), jt, id)
	}
	b.WriteString("}\n")
	return b.String()
}

func schPython(class string, fields []schField) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Generated by Stackyard from an EventBridge schema.\nfrom dataclasses import dataclass\nfrom typing import Optional\n\n\n@dataclass\nclass %s:\n", class)
	if len(fields) == 0 {
		b.WriteString("    pass\n")
	}
	for _, f := range fields {
		fmt.Fprintf(&b, "    %s: Optional[%s] = None\n", schSnake(schIdent(f.name, false)), schTypeFor("Python36", f.typ))
	}
	return b.String()
}

func schTypeScript(class string, fields []schField) string {
	var b strings.Builder
	fmt.Fprintf(&b, "// Generated by Stackyard from an EventBridge schema.\nexport interface %s {\n", class)
	for _, f := range fields {
		fmt.Fprintf(&b, "  %s?: %s;\n", schIdent(f.name, false), schTypeFor("TypeScript3", f.typ))
	}
	b.WriteString("}\n")
	return b.String()
}

func schGo(class string, fields []schField) string {
	var b strings.Builder
	fmt.Fprintf(&b, "// Generated by Stackyard from an EventBridge schema.\npackage schema\n\ntype %s struct {\n", class)
	for _, f := range fields {
		tag, _ := json.Marshal(strings.ReplaceAll(f.name, "`", ""))
		fmt.Fprintf(&b, "\t%s %s `json:%s`\n", schIdent(f.name, true), schTypeFor("Go1", f.typ), tag)
	}
	b.WriteString("}\n")
	return b.String()
}
