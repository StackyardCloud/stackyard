package server

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

const schTestOpenAPI = `{"openapi":"3.0.0","info":{"version":"1.0.0","title":"Order"},"paths":{},"components":{"schemas":{"AWSEvent":{"type":"object","properties":{"detail":{"$ref":"#/components/schemas/Order"},"source":{"type":"string"}}},"Order":{"type":"object","properties":{"orderId":{"type":"string"},"total":{"type":"number"}}}}}}`

func schJSONBody(v any) string {
	raw, _ := json.Marshal(v)
	return string(raw)
}

func TestSchemasStage1RegistriesAndSchemas(t *testing.T) {
	_, ts := newSchemasTestServer(t)
	reg := "/v1/registries/name/orders"
	out := schCall(t, ts, http.StatusCreated, http.MethodPost, reg, `{"Description":"Order events","tags":{"team":"core"}}`)
	if out["RegistryArn"] != "arn:aws:schemas:us-east-1:123456789012:registry/orders" || out["tags"].(map[string]any)["team"] != "core" {
		t.Fatalf("create registry: %v", out)
	}
	if code := schErrCode(t, ts, http.MethodPost, reg, `{}`); code != "ConflictException" {
		t.Fatalf("duplicate registry: %s", code)
	}
	if code := schErrCode(t, ts, http.MethodPost, "/v1/registries/name/aws.mine", `{}`); code != "BadRequestException" {
		t.Fatalf("aws. registry name: %s", code)
	}
	if code := schErrCode(t, ts, http.MethodDelete, "/v1/registries/name/aws.events", ""); code != "ForbiddenException" {
		t.Fatalf("delete aws.events: %s", code)
	}
	scope := schCall(t, ts, http.StatusOK, http.MethodGet, "/v1/registries?scope=AWS", "")["Registries"].([]any)
	if len(scope) != 1 || scope[0].(map[string]any)["RegistryName"] != "aws.events" {
		t.Fatalf("AWS scope: %v", scope)
	}

	sch := reg + "/schemas/name/OrderPlaced"
	if code := schErrCode(t, ts, http.MethodPost, sch, `{"Type":"OpenApi3","Content":"{\"openapi\":\"2.0\"}"}`); code != "BadRequestException" {
		t.Fatalf("bad OpenAPI content: %s", code)
	}
	out = schCall(t, ts, http.StatusCreated, http.MethodPost, sch, schJSONBody(map[string]any{"Type": "OpenApi3", "Content": schTestOpenAPI, "Description": "v1"}))
	if out["SchemaVersion"] != "1" || out["Type"] != "OpenApi3" {
		t.Fatalf("create schema: %v", out)
	}
	// Same content: no new version. New content: version 2. A retried
	// ClientTokenId doesn't add a third.
	schCall(t, ts, http.StatusOK, http.MethodPut, sch, schJSONBody(map[string]any{"Content": schTestOpenAPI}))
	v2 := strings.Replace(schTestOpenAPI, `"total":{"type":"number"}`, `"total":{"type":"number"},"currency":{"type":"string"}`, 1)
	body := schJSONBody(map[string]any{"Content": v2, "ClientTokenId": "tok-1"})
	if out = schCall(t, ts, http.StatusOK, http.MethodPut, sch, body); out["SchemaVersion"] != "2" {
		t.Fatalf("update schema: %v", out)
	}
	if out = schCall(t, ts, http.StatusOK, http.MethodPut, sch, body); out["SchemaVersion"] != "2" {
		t.Fatalf("retried update: %v", out)
	}
	versions := schCall(t, ts, http.StatusOK, http.MethodGet, sch+"/versions?limit=1", "")
	if len(versions["SchemaVersions"].([]any)) != 1 || versions["NextToken"] == nil {
		t.Fatalf("paged versions: %v", versions)
	}
	if d := schCall(t, ts, http.StatusOK, http.MethodGet, sch+"?schemaVersion=1", ""); d["Content"] != schTestOpenAPI {
		t.Fatalf("describe v1: %v", d["Content"])
	}

	// Export turns components into definitions and rewrites refs.
	exp := schCall(t, ts, http.StatusOK, http.MethodGet, sch+"/export?type=JSONSchemaDraft4", "")
	var doc map[string]any
	_ = json.Unmarshal([]byte(exp["Content"].(string)), &doc)
	detail := doc["properties"].(map[string]any)["detail"].(map[string]any)
	if doc["$schema"] != schDraft4 || detail["$ref"] != "#/definitions/Order" || doc["definitions"].(map[string]any)["Order"] == nil {
		t.Fatalf("export: %s", exp["Content"])
	}

	search := schCall(t, ts, http.StatusOK, http.MethodGet, reg+"/schemas/search?keywords=currency", "")["Schemas"].([]any)
	if len(search) != 1 || len(search[0].(map[string]any)["SchemaVersions"].([]any)) != 2 {
		t.Fatalf("search: %v", search)
	}
	if list := schCall(t, ts, http.StatusOK, http.MethodGet, reg+"/schemas", "")["Schemas"].([]any); list[0].(map[string]any)["VersionCount"] != float64(2) {
		t.Fatalf("list schemas: %v", list)
	}

	if code := schErrCode(t, ts, http.MethodDelete, sch+"/version/9", ""); code != "NotFoundException" {
		t.Fatalf("delete missing version: %s", code)
	}
	schCall(t, ts, http.StatusNoContent, http.MethodDelete, sch+"/version/1", "")
	if code := schErrCode(t, ts, http.MethodDelete, sch+"/version/2", ""); code != "BadRequestException" {
		t.Fatalf("delete only version: %s", code)
	}

	arn := url.PathEscape("arn:aws:schemas:us-east-1:123456789012:schema/orders/OrderPlaced")
	schCall(t, ts, http.StatusNoContent, http.MethodPost, "/tags/"+arn, `{"tags":{"env":"dev"}}`)
	if tags := schCall(t, ts, http.StatusOK, http.MethodGet, "/tags/"+arn, "")["tags"].(map[string]any); tags["env"] != "dev" {
		t.Fatalf("tags: %v", tags)
	}
	schCall(t, ts, http.StatusNoContent, http.MethodDelete, "/tags/"+arn+"?tagKeys=env", "")
	schCall(t, ts, http.StatusNoContent, http.MethodDelete, reg, "")
	if code := schErrCode(t, ts, http.MethodGet, "/tags/"+arn, ""); code != "NotFoundException" {
		t.Fatalf("tags after registry delete: %s", code)
	}
}

func TestSchemasStage1CodeBindings(t *testing.T) {
	_, ts := newSchemasTestServer(t)
	schCall(t, ts, http.StatusCreated, http.MethodPost, "/v1/registries/name/r", `{}`)
	sch := "/v1/registries/name/r/schemas/name/shop@OrderPlaced"
	schCall(t, ts, http.StatusCreated, http.MethodPost, sch, schJSONBody(map[string]any{"Type": "OpenApi3", "Content": schTestOpenAPI}))
	if code := schErrCode(t, ts, http.MethodPost, sch+"/language/Cobol", ""); code != "BadRequestException" {
		t.Fatalf("bad language: %s", code)
	}
	if code := schErrCode(t, ts, http.MethodGet, sch+"/language/Java8/source", ""); code != "NotFoundException" {
		t.Fatalf("source before binding: %s", code)
	}
	if out := schCall(t, ts, http.StatusAccepted, http.MethodPost, sch+"/language/Java8", ""); out["Status"] != "CREATE_IN_PROGRESS" {
		t.Fatalf("put binding: %v", out)
	}
	if code := schErrCode(t, ts, http.MethodGet, sch+"/language/Java8/source", ""); code != "NotFoundException" {
		t.Fatalf("source while generating: %s", code)
	}
	schCall(t, ts, http.StatusOK, http.MethodGet, sch+"/language/Java8", "")
	if out := schCall(t, ts, http.StatusOK, http.MethodGet, sch+"/language/Java8", ""); out["Status"] != "CREATE_COMPLETE" {
		t.Fatalf("binding status: %v", out)
	}
	status, hdr, data := schRaw(t, ts, http.MethodGet, sch+"/language/Java8/source", "")
	if status != http.StatusOK || hdr.Get("Content-Type") != "application/zip" {
		t.Fatalf("source: %d %s", status, hdr.Get("Content-Type"))
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil || len(zr.File) != 1 || zr.File[0].Name != "src/main/java/schema/OrderPlaced.java" {
		t.Fatalf("archive: %v %v", err, zr)
	}
	f, _ := zr.File[0].Open()
	src, _ := io.ReadAll(f)
	if !strings.Contains(string(src), "public class OrderPlaced") || !strings.Contains(string(src), "private Object detail;") || !strings.Contains(string(src), "private String source;") {
		t.Fatalf("generated source:\n%s", src)
	}
	// The archive is deterministic.
	if _, _, again := schRaw(t, ts, http.MethodGet, sch+"/language/Java8/source", ""); !bytes.Equal(again, data) {
		t.Fatal("archive changed between reads")
	}
}

func TestSchemasStage1DiscoveryAndDiscoverers(t *testing.T) {
	_, ts := newSchemasTestServer(t)
	events := []string{
		`{"source":"shop","detail":{"id":1,"price":2,"tags":["a"],"at":"2026-01-02T03:04:05Z","note":null}}`,
		`{"source":"shop","detail":{"id":2,"price":2.5,"tags":[],"at":"2026-01-02T03:04:05Z","note":"x"},"extra":true}`,
	}
	out := schCall(t, ts, http.StatusOK, http.MethodPost, "/v1/discover", schJSONBody(map[string]any{"Type": "JSONSchemaDraft4", "Events": events}))
	var doc map[string]any
	_ = json.Unmarshal([]byte(out["Content"].(string)), &doc)
	props := doc["properties"].(map[string]any)
	detail := props["detail"].(map[string]any)["properties"].(map[string]any)
	for field, want := range map[string]string{"id": "integer", "price": "number", "note": "string"} {
		if got := detail[field].(map[string]any)["type"]; got != want {
			t.Fatalf("%s inferred as %v, want %s", field, got, want)
		}
	}
	if detail["at"].(map[string]any)["format"] != "date-time" || detail["tags"].(map[string]any)["items"].(map[string]any)["type"] != "string" {
		t.Fatalf("detail: %v", detail)
	}
	if fmt.Sprint(doc["required"]) != "[detail source]" || props["extra"] == nil {
		t.Fatalf("required keys present in every event only: %v / %v", doc["required"], props)
	}
	if code := schErrCode(t, ts, http.MethodPost, "/v1/discover", `{"Type":"OpenApi3","Events":["[1]"]}`); code != "BadRequestException" {
		t.Fatalf("non-object event: %s", code)
	}

	if code := schErrCode(t, ts, http.MethodPost, "/v1/discoverers", `{"SourceArn":"arn:aws:events:us-east-1:123456789012:event-bus/nope"}`); code != "BadRequestException" {
		t.Fatalf("missing bus: %s", code)
	}
	bus := "arn:aws:events:us-east-1:123456789012:event-bus/default"
	d := schCall(t, ts, http.StatusCreated, http.MethodPost, "/v1/discoverers", schJSONBody(map[string]any{"SourceArn": bus}))
	id := d["DiscovererId"].(string)
	if d["State"] != "STARTED" || d["CrossAccount"] != true {
		t.Fatalf("create discoverer: %v", d)
	}
	if code := schErrCode(t, ts, http.MethodPost, "/v1/discoverers", schJSONBody(map[string]any{"SourceArn": bus})); code != "ConflictException" {
		t.Fatalf("second discoverer for a bus: %s", code)
	}
	if out := schCall(t, ts, http.StatusOK, http.MethodPost, "/v1/discoverers/id/"+id+"/stop", ""); out["State"] != "STOPPED" {
		t.Fatalf("stop: %v", out)
	}
	if out := schCall(t, ts, http.StatusOK, http.MethodPut, "/v1/discoverers/id/"+id, `{"CrossAccount":false}`); out["CrossAccount"] != false {
		t.Fatalf("update: %v", out)
	}
	schCall(t, ts, http.StatusNoContent, http.MethodDelete, "/v1/discoverers/id/"+id, "")
	if code := schErrCode(t, ts, http.MethodGet, "/v1/discoverers/id/"+id, ""); code != "NotFoundException" {
		t.Fatalf("deleted discoverer: %s", code)
	}
}

func TestSchemasStage1ResourcePolicy(t *testing.T) {
	_, ts := newSchemasTestServer(t)
	if code := schErrCode(t, ts, http.MethodGet, "/v1/policy", ""); code != "NotFoundException" {
		t.Fatalf("no policy: %s", code)
	}
	policy := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":"*","Action":"schemas:*","Resource":"*"}]}`
	if code := schErrCode(t, ts, http.MethodPut, "/v1/policy", `{"Policy":"not json"}`); code != "BadRequestException" {
		t.Fatalf("bad policy: %s", code)
	}
	rev := schCall(t, ts, http.StatusOK, http.MethodPut, "/v1/policy", schJSONBody(map[string]any{"Policy": policy}))["RevisionId"].(string)
	if code := schErrCode(t, ts, http.MethodPut, "/v1/policy", schJSONBody(map[string]any{"Policy": policy, "RevisionId": "stale"})); code != "PreconditionFailedException" {
		t.Fatalf("stale revision: %s", code)
	}
	schCall(t, ts, http.StatusOK, http.MethodPut, "/v1/policy", schJSONBody(map[string]any{"Policy": policy, "RevisionId": rev}))
	schCall(t, ts, http.StatusNoContent, http.MethodDelete, "/v1/policy", "")
	if code := schErrCode(t, ts, http.MethodPut, "/v1/policy?registryName=nope", schJSONBody(map[string]any{"Policy": policy})); code != "NotFoundException" {
		t.Fatalf("policy on missing registry: %s", code)
	}
}

// Deeply nested events must not recurse without bound.
func TestSchemasStage1DiscoveryDepthIsBounded(t *testing.T) {
	deep := strings.Repeat(`{"a":`, 5000) + "1" + strings.Repeat("}", 5000)
	if _, err := schDiscover([]string{deep}, "JSONSchemaDraft4"); err != nil {
		t.Fatalf("deep event: %v", err.msg)
	}
}
