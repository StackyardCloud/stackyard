package server

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Schema Registry: registries, schemas, schema versions (with a compatibility
// check against the previous version), checkpoints and version metadata.

const (
	glueDefaultRegistry = "default-registry"
	// glueMaxVersionMetadata is the Schema Registry limit of metadata
	// key-value pairs per schema version.
	glueMaxVersionMetadata = 10
)

func init() {
	glueKinds["schemaversion"] = glueKind{fields: []string{"SchemaVersionId"}, parent: "schema", parentFields: []string{"RegistryName", "SchemaName"}}
	registerGlueHandlers(map[string]glueHandler{
		"CreateRegistry":              glueCreateRegistry,
		"GetRegistry":                 glueGetRegistry,
		"ListRegistries":              glueListRegistries,
		"UpdateRegistry":              glueUpdateRegistry,
		"DeleteRegistry":              glueDeleteRegistry,
		"CreateSchema":                glueCreateSchema,
		"GetSchema":                   glueGetSchema,
		"ListSchemas":                 glueListSchemas,
		"UpdateSchema":                glueUpdateSchema,
		"DeleteSchema":                glueDeleteSchema,
		"RegisterSchemaVersion":       glueRegisterSchemaVersion,
		"GetSchemaVersion":            glueGetSchemaVersion,
		"GetSchemaByDefinition":       glueGetSchemaByDefinition,
		"ListSchemaVersions":          glueListSchemaVersions,
		"DeleteSchemaVersions":        glueDeleteSchemaVersions,
		"CheckSchemaVersionValidity":  glueCheckSchemaVersionValidity,
		"GetSchemaVersionsDiff":       glueGetSchemaVersionsDiff,
		"PutSchemaVersionMetadata":    glueSchemaVersionMetadata(true),
		"RemoveSchemaVersionMetadata": glueSchemaVersionMetadata(false),
		"QuerySchemaVersionMetadata":  glueQuerySchemaVersionMetadata,
	})
}

// glueARNName returns the last path segment of a Glue ARN.
func glueARNName(arn string) string {
	if i := strings.LastIndex(arn, "/"); i >= 0 {
		return arn[i+1:]
	}
	return arn
}

func (st *glueStore) registry(id any) (glueRecord, *glueError) {
	m, _ := id.(map[string]any)
	name := glueFirst(glueStr(m, "RegistryName"), glueARNName(glueStr(m, "RegistryArn")))
	if name == "" {
		return nil, glueInvalid("RegistryId needs a RegistryName or RegistryArn")
	}
	return st.find("registry", []any{name})
}

func (st *glueStore) schema(id any) (glueRecord, *glueError) {
	m, _ := id.(map[string]any)
	if arn := glueStr(m, "SchemaArn"); arn != "" {
		parts := strings.Split(arn, "/")
		if len(parts) < 3 {
			return nil, glueInvalid("invalid SchemaArn %s", arn)
		}
		return st.find("schema", []any{parts[len(parts)-2], parts[len(parts)-1]})
	}
	if glueStr(m, "SchemaName") == "" {
		return nil, glueInvalid("SchemaId needs a SchemaArn or a SchemaName")
	}
	return st.find("schema", []any{glueFirst(glueStr(m, "RegistryName"), glueDefaultRegistry), m["SchemaName"]})
}

func (st *glueStore) newRegistry(name any, description any, tags map[string]string) (glueRecord, *glueError) {
	rec := glueRecord{"RegistryName": name, "Description": description, "Status": "AVAILABLE"}
	rec["RegistryArn"] = st.arn("registry", rec)
	return st.create("registry", rec, tags)
}

func glueCreateRegistry(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	rec, err := st.newRegistry(in["RegistryName"], in["Description"], glueTags(in["Tags"]))
	if err != nil {
		return nil, err
	}
	return glueMerge(rec, glueRecord{"Tags": in["Tags"]}), nil
}

func glueGetRegistry(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	return st.registry(in["RegistryId"])
}

func glueListRegistries(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	return gluePageRecords(st.all("registry"), in, "Registries")
}

func glueUpdateRegistry(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	rec, err := st.registry(in["RegistryId"])
	if err != nil {
		return nil, err
	}
	rec["Description"] = in["Description"]
	glueTouch(rec)
	return rec, nil
}

func glueDeleteRegistry(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	rec, err := st.registry(in["RegistryId"])
	if err != nil {
		return nil, err
	}
	st.remove("registry", rec)
	return glueRecord{"RegistryName": rec["RegistryName"], "RegistryArn": rec["RegistryArn"], "Status": "DELETING"}, nil
}

func glueCreateSchema(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	var reg glueRecord
	var err *glueError
	if in["RegistryId"] != nil {
		if reg, err = st.registry(in["RegistryId"]); err != nil {
			return nil, err
		}
	} else if reg, err = st.find("registry", []any{glueDefaultRegistry}); err != nil {
		if reg, err = st.newRegistry(glueDefaultRegistry, "Default registry", nil); err != nil {
			return nil, err
		}
	}
	compat := glueFirst(glueStr(in, "Compatibility"), "BACKWARD")
	rec := glueRecord{
		"RegistryName": reg["RegistryName"], "RegistryArn": reg["RegistryArn"], "SchemaName": in["SchemaName"],
		"DataFormat": in["DataFormat"], "Compatibility": compat, "Description": in["Description"],
		"SchemaStatus": "AVAILABLE", "LatestSchemaVersion": float64(0), "NextSchemaVersion": float64(1), "SchemaCheckpoint": float64(0),
	}
	rec["SchemaArn"] = st.arn("schema", rec)
	if def := glueStr(in, "SchemaDefinition"); def != "" {
		if msg := glueSchemaSyntaxError(glueStr(in, "DataFormat"), def); msg != "" {
			return nil, glueInvalid("invalid schema definition: %s", msg)
		}
	}
	if _, err := st.create("schema", rec, glueTags(in["Tags"])); err != nil {
		return nil, err
	}
	out := glueMerge(rec, glueRecord{"Tags": in["Tags"]})
	if def := glueStr(in, "SchemaDefinition"); def != "" {
		v, err := st.registerVersion(rec, def)
		if err != nil {
			return nil, err
		}
		rec["SchemaCheckpoint"] = v["VersionNumber"]
		out["SchemaVersionId"], out["SchemaVersionStatus"] = v["SchemaVersionId"], v["Status"]
		out["LatestSchemaVersion"], out["NextSchemaVersion"], out["SchemaCheckpoint"] = rec["LatestSchemaVersion"], rec["NextSchemaVersion"], rec["SchemaCheckpoint"]
	}
	return out, nil
}

func glueGetSchema(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	return st.schema(in["SchemaId"])
}

func glueListSchemas(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	var reg glueRecord
	if in["RegistryId"] != nil {
		var err *glueError
		if reg, err = st.registry(in["RegistryId"]); err != nil {
			return nil, err
		}
	}
	var out []glueRecord
	for _, s := range st.all("schema") {
		if reg == nil || s["RegistryName"] == reg["RegistryName"] {
			out = append(out, s)
		}
	}
	return gluePageRecords(out, in, "Schemas")
}

func glueUpdateSchema(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	rec, err := st.schema(in["SchemaId"])
	if err != nil {
		return nil, err
	}
	if in["Compatibility"] == nil && in["Description"] == nil && in["SchemaVersionNumber"] == nil {
		return nil, glueInvalid("specify Compatibility, Description or SchemaVersionNumber")
	}
	if c := in["Compatibility"]; c != nil {
		rec["Compatibility"] = c
	}
	if d := in["Description"]; d != nil {
		rec["Description"] = d
	}
	if in["SchemaVersionNumber"] != nil {
		v, err := st.schemaVersionByNumber(rec, in["SchemaVersionNumber"])
		if err != nil {
			return nil, err
		}
		rec["SchemaCheckpoint"] = v["VersionNumber"]
	}
	glueTouch(rec)
	return rec, nil
}

func glueDeleteSchema(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	rec, err := st.schema(in["SchemaId"])
	if err != nil {
		return nil, err
	}
	st.remove("schema", rec)
	return glueRecord{"SchemaArn": rec["SchemaArn"], "SchemaName": rec["SchemaName"], "Status": "DELETING"}, nil
}

func (st *glueStore) schemaVersions(schema glueRecord) []glueRecord {
	var out []glueRecord
	for _, v := range st.all("schemaversion") {
		if v["RegistryName"] == schema["RegistryName"] && v["SchemaName"] == schema["SchemaName"] {
			out = append(out, v)
		}
	}
	// Order by version number.
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j]["VersionNumber"].(float64) < out[j-1]["VersionNumber"].(float64); j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

// registerVersion adds def as the schema's next version, or returns the
// existing version with the same definition.
func (st *glueStore) registerVersion(schema glueRecord, def string) (glueRecord, *glueError) {
	if schema["Compatibility"] == "DISABLED" {
		return nil, glueInvalid("compatibility DISABLED does not allow new versions of schema %v", schema["SchemaName"])
	}
	if msg := glueSchemaSyntaxError(glueStr(schema, "DataFormat"), def); msg != "" {
		return nil, glueInvalid("invalid schema definition: %s", msg)
	}
	versions := st.schemaVersions(schema)
	for _, v := range versions {
		if v["SchemaDefinition"] == def {
			return v, nil
		}
	}
	status := "AVAILABLE"
	if len(versions) > 0 && !glueSchemaCompatible(glueStr(schema, "Compatibility"), glueStr(schema, "DataFormat"), versions, def) {
		status = "FAILURE"
	}
	next := schema["NextSchemaVersion"].(float64)
	v, err := st.create("schemaversion", glueRecord{
		"SchemaVersionId": glueUUID(st.newID("")), "RegistryName": schema["RegistryName"], "SchemaName": schema["SchemaName"],
		"SchemaArn": schema["SchemaArn"], "DataFormat": schema["DataFormat"], "SchemaDefinition": def,
		"VersionNumber": next, "Status": status, "_metadata": map[string]any{},
	}, nil)
	if err != nil {
		return nil, err
	}
	if status == "AVAILABLE" {
		schema["LatestSchemaVersion"] = next
		if schema["SchemaCheckpoint"].(float64) == 0 {
			schema["SchemaCheckpoint"] = next
		}
	}
	schema["NextSchemaVersion"] = next + 1
	glueTouch(schema)
	return v, nil
}

// glueUUID formats 32+ hex characters as a UUID (SchemaVersionId is one).
func glueUUID(hex string) string {
	for len(hex) < 32 {
		hex += "0"
	}
	return fmt.Sprintf("%s-%s-%s-%s-%s", hex[0:8], hex[8:12], hex[12:16], hex[16:20], hex[20:32])
}

func glueSchemaSyntaxError(format, def string) string {
	switch format {
	case "AVRO", "JSON":
		var v any
		if err := json.Unmarshal([]byte(def), &v); err != nil {
			return err.Error()
		}
		if format == "AVRO" {
			if m, ok := v.(map[string]any); ok && m["type"] == nil {
				return "an Avro schema needs a type"
			}
		}
	case "PROTOBUF":
		if !strings.Contains(def, "message") {
			return "a Protobuf schema needs at least one message"
		}
	}
	return ""
}

// glueSchemaCompatible checks Avro and JSON record schemas field by field:
// BACKWARD (readers of the new schema read old data) forbids adding a field
// without a default; FORWARD forbids removing a field without a default;
// FULL forbids both. The _ALL variants check every earlier version instead
// of only the latest. Protobuf and NONE are always compatible.
func glueSchemaCompatible(mode, format string, versions []glueRecord, def string) bool {
	if mode == "NONE" || format == "PROTOBUF" {
		return true
	}
	compare := versions
	if !strings.HasSuffix(mode, "_ALL") {
		var latest []glueRecord
		for i := len(versions) - 1; i >= 0; i-- {
			if versions[i]["Status"] == "AVAILABLE" {
				latest = []glueRecord{versions[i]}
				break
			}
		}
		compare = latest
	}
	base := strings.TrimSuffix(mode, "_ALL")
	newFields := glueSchemaFields(format, def)
	for _, old := range compare {
		if old["Status"] != "AVAILABLE" {
			continue
		}
		oldFields := glueSchemaFields(format, glueStr(old, "SchemaDefinition"))
		if base == "BACKWARD" || base == "FULL" {
			for name, hasDefault := range newFields {
				if _, ok := oldFields[name]; !ok && !hasDefault {
					return false
				}
			}
		}
		if base == "FORWARD" || base == "FULL" {
			for name, hasDefault := range oldFields {
				if _, ok := newFields[name]; !ok && !hasDefault {
					return false
				}
			}
		}
	}
	return true
}

// glueSchemaFields maps top-level field names to whether they have a default
// (Avro "default", or JSON Schema properties not listed as required).
func glueSchemaFields(format, def string) map[string]bool {
	var m map[string]any
	_ = json.Unmarshal([]byte(def), &m)
	out := map[string]bool{}
	if format == "AVRO" {
		for _, f := range glueList(m["fields"]) {
			if fm, ok := f.(map[string]any); ok {
				_, hasDefault := fm["default"]
				out[glueStr(fm, "name")] = hasDefault
			}
		}
		return out
	}
	required := map[string]bool{}
	for _, r := range glueList(m["required"]) {
		required[fmt.Sprint(r)] = true
	}
	props, _ := m["properties"].(map[string]any)
	for name := range props {
		out[name] = !required[name]
	}
	return out
}

func glueRegisterSchemaVersion(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	schema, err := st.schema(in["SchemaId"])
	if err != nil {
		return nil, err
	}
	v, err := st.registerVersion(schema, glueStr(in, "SchemaDefinition"))
	if err != nil {
		return nil, err
	}
	return glueRecord{"SchemaVersionId": v["SchemaVersionId"], "VersionNumber": v["VersionNumber"], "Status": v["Status"]}, nil
}

func (st *glueStore) schemaVersionByNumber(schema glueRecord, number any) (glueRecord, *glueError) {
	m, _ := number.(map[string]any)
	versions := st.schemaVersions(schema)
	if m["LatestVersion"] == true {
		for i := len(versions) - 1; i >= 0; i-- {
			if versions[i]["Status"] == "AVAILABLE" {
				return versions[i], nil
			}
		}
		return nil, glueNotFound("schema %v has no versions", schema["SchemaName"])
	}
	n, ok := m["VersionNumber"].(float64)
	if !ok {
		return nil, glueInvalid("SchemaVersionNumber needs LatestVersion or VersionNumber")
	}
	for _, v := range versions {
		if v["VersionNumber"] == n {
			return v, nil
		}
	}
	return nil, glueNotFound("schema %v has no version %v", schema["SchemaName"], n)
}

// schemaVersion resolves SchemaVersionId, or SchemaId plus SchemaVersionNumber.
func (st *glueStore) schemaVersion(in glueRecord) (glueRecord, *glueError) {
	if id := in["SchemaVersionId"]; id != nil {
		return st.find("schemaversion", []any{id})
	}
	if in["SchemaId"] == nil || in["SchemaVersionNumber"] == nil {
		return nil, glueInvalid("specify SchemaVersionId, or SchemaId with SchemaVersionNumber")
	}
	schema, err := st.schema(in["SchemaId"])
	if err != nil {
		return nil, err
	}
	return st.schemaVersionByNumber(schema, in["SchemaVersionNumber"])
}

func glueGetSchemaVersion(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	return st.schemaVersion(in)
}

func glueGetSchemaByDefinition(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	schema, err := st.schema(in["SchemaId"])
	if err != nil {
		return nil, err
	}
	for _, v := range st.schemaVersions(schema) {
		if v["SchemaDefinition"] == in["SchemaDefinition"] {
			return v, nil
		}
	}
	return nil, glueNotFound("schema %v has no version with that definition", schema["SchemaName"])
}

func glueListSchemaVersions(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	schema, err := st.schema(in["SchemaId"])
	if err != nil {
		return nil, err
	}
	return gluePageRecords(st.schemaVersions(schema), in, "Schemas")
}

// glueVersionRange parses "3" or "2-5".
func glueVersionRange(s string) (float64, float64, bool) {
	lo, hi, found := strings.Cut(strings.TrimSpace(s), "-")
	a, err1 := strconv.ParseInt(strings.TrimSpace(lo), 10, 64)
	if err1 != nil || a < 1 {
		return 0, 0, false
	}
	b := a
	if found {
		var err2 error
		if b, err2 = strconv.ParseInt(strings.TrimSpace(hi), 10, 64); err2 != nil || b < a {
			return 0, 0, false
		}
	}
	return float64(a), float64(b), true
}

func glueDeleteSchemaVersions(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	schema, err := st.schema(in["SchemaId"])
	if err != nil {
		return nil, err
	}
	lo, hi, ok := glueVersionRange(glueStr(in, "Versions"))
	if !ok {
		return nil, glueInvalid("Versions must be a version number or a range such as 2-5")
	}
	var errs []any
	for _, v := range st.schemaVersions(schema) {
		n := v["VersionNumber"].(float64)
		if n < lo || n > hi {
			continue
		}
		if n == schema["SchemaCheckpoint"] {
			errs = append(errs, glueRecord{"VersionNumber": n, "ErrorDetails": glueRecord{"ErrorCode": "InvalidInputException", "ErrorMessage": "the checkpoint version can't be deleted"}})
			continue
		}
		st.remove("schemaversion", v)
	}
	latest := float64(0)
	for _, v := range st.schemaVersions(schema) {
		if v["Status"] == "AVAILABLE" {
			latest = v["VersionNumber"].(float64)
		}
	}
	schema["LatestSchemaVersion"] = latest
	return glueRecord{"SchemaVersionErrors": glueNonNil(errs)}, nil
}

func glueCheckSchemaVersionValidity(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	if msg := glueSchemaSyntaxError(glueStr(in, "DataFormat"), glueStr(in, "SchemaDefinition")); msg != "" {
		return glueRecord{"Valid": false, "Error": msg}, nil
	}
	return glueRecord{"Valid": true}, nil
}

func glueGetSchemaVersionsDiff(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	schema, err := st.schema(in["SchemaId"])
	if err != nil {
		return nil, err
	}
	a, err := st.schemaVersionByNumber(schema, in["FirstSchemaVersionNumber"])
	if err != nil {
		return nil, err
	}
	b, err := st.schemaVersionByNumber(schema, in["SecondSchemaVersionNumber"])
	if err != nil {
		return nil, err
	}
	return glueRecord{"Diff": glueLineDiff(glueStr(a, "SchemaDefinition"), glueStr(b, "SchemaDefinition"))}, nil
}

// glueLineDiff is a minimal unified-style diff: lines only in the first
// version are prefixed "-", lines only in the second "+".
func glueLineDiff(a, b string) string {
	inA, inB := map[string]bool{}, map[string]bool{}
	la, lb := strings.Split(a, "\n"), strings.Split(b, "\n")
	for _, l := range la {
		inA[l] = true
	}
	for _, l := range lb {
		inB[l] = true
	}
	var out []string
	for _, l := range la {
		if !inB[l] {
			out = append(out, "-"+l)
		}
	}
	for _, l := range lb {
		if !inA[l] {
			out = append(out, "+"+l)
		}
	}
	return strings.Join(out, "\n")
}

func glueSchemaVersionMetadata(put bool) glueHandler {
	return func(st *glueStore, in glueRecord) (glueRecord, *glueError) {
		v, err := st.schemaVersion(in)
		if err != nil {
			return nil, err
		}
		kv, _ := in["MetadataKeyValue"].(map[string]any)
		key, value := glueStr(kv, "MetadataKey"), glueStr(kv, "MetadataValue")
		meta := v["_metadata"].(map[string]any)
		values, _ := meta[key].([]any)
		if put {
			for _, existing := range values {
				if existing.(glueRecord)["MetadataValue"] == value {
					return nil, glueErr("AlreadyExistsException", "metadata %s=%s already exists", key, value)
				}
			}
			if len(values) == 0 && len(meta) >= glueMaxVersionMetadata {
				return nil, glueErr("ResourceNumberLimitExceededException", "a schema version can have at most %d metadata keys", glueMaxVersionMetadata)
			}
			if len(values) >= glueMaxVersionMetadata {
				return nil, glueErr("ResourceNumberLimitExceededException", "metadata key %s can have at most %d values", key, glueMaxVersionMetadata)
			}
			meta[key] = append(values, glueRecord{"MetadataValue": value, "CreatedTime": glueEpoch(time.Now())})
		} else {
			var kept []any
			found := false
			for _, existing := range values {
				if existing.(glueRecord)["MetadataValue"] == value {
					found = true
					continue
				}
				kept = append(kept, existing)
			}
			if !found {
				return nil, glueNotFound("metadata %s=%s not found", key, value)
			}
			if len(kept) == 0 {
				delete(meta, key)
			} else {
				meta[key] = kept
			}
		}
		schema, _ := st.find("schema", []any{v["RegistryName"], v["SchemaName"]})
		return glueRecord{
			"SchemaArn": v["SchemaArn"], "SchemaName": v["SchemaName"], "RegistryName": v["RegistryName"], "SchemaVersionId": v["SchemaVersionId"],
			"VersionNumber": v["VersionNumber"], "LatestVersion": schema != nil && schema["LatestSchemaVersion"] == v["VersionNumber"],
			"MetadataKey": key, "MetadataValue": value,
		}, nil
	}
}

func glueQuerySchemaVersionMetadata(st *glueStore, in glueRecord) (glueRecord, *glueError) {
	v, err := st.schemaVersion(in)
	if err != nil {
		return nil, err
	}
	meta := v["_metadata"].(map[string]any)
	want := map[string]string{}
	for _, raw := range glueList(in["MetadataList"]) {
		m, _ := raw.(map[string]any)
		want[glueStr(m, "MetadataKey")] = glueStr(m, "MetadataValue")
	}
	out := map[string]any{}
	for key, raw := range meta {
		values := raw.([]any)
		if wv, ok := want[key]; len(want) > 0 && (!ok || (wv != "" && !glueMetadataHas(values, wv))) {
			continue
		}
		first := values[0].(glueRecord)
		info := glueRecord{"MetadataValue": first["MetadataValue"], "CreatedTime": first["CreatedTime"]}
		var others []any
		for _, o := range values[1:] {
			others = append(others, o)
		}
		info["OtherMetadataValueList"] = glueNonNil(others)
		out[key] = info
	}
	return glueRecord{"MetadataInfoMap": out, "SchemaVersionId": v["SchemaVersionId"]}, nil
}

func glueMetadataHas(values []any, value string) bool {
	for _, v := range values {
		if v.(glueRecord)["MetadataValue"] == value {
			return true
		}
	}
	return false
}
