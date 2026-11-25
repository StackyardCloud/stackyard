package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func csdCall(t *testing.T, ts *httptest.Server, method, path, contentType, body string) (int, map[string]any) {
	t.Helper()
	headers := map[string]string{}
	if contentType != "" {
		headers["Content-Type"] = contentType
	}
	var raw []byte
	if body != "" {
		raw = []byte(body)
	}
	resp := signedRequestWithService(t, method, ts.URL+path, raw, headers, "cloudsearch")
	b := mustBody(t, resp)
	out := map[string]any{}
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("%s %s: decode %q: %v", method, path, b, err)
	}
	return resp.StatusCode, out
}

func TestCloudSearchDomainStage1DocumentsAndSearch(t *testing.T) {
	ts := newCSTestServer(t)
	csSetupDomain(t, ts, "films")
	batch := `[{"type":"add","id":"1","fields":{"title":"Star Wars","genre":"scifi","year":1977,"tag_x":["a","b"]}},
		{"type":"add","id":"2","fields":{"title":"Star Trek","genre":"scifi","year":1979}},
		{"type":"add","id":"3","fields":{"title":"Casablanca","genre":"drama","year":1942}}]`
	status, out := csdCall(t, ts, http.MethodPost, "/films/2013-01-01/documents/batch?format=sdk", "application/json", batch)
	if status != http.StatusOK || out["adds"] != float64(3) {
		t.Fatalf("upload: %d %v", status, out)
	}
	// Undefined fields, bad values and unknown domains are refused.
	status, out = csdCall(t, ts, http.MethodPost, "/films/2013-01-01/documents/batch", "application/json", `[{"type":"add","id":"9","fields":{"nope":"x"}}]`)
	if status != http.StatusBadRequest || out["status"] != "error" {
		t.Fatalf("undefined field: %d %v", status, out)
	}
	status, _ = csdCall(t, ts, http.MethodPost, "/films/2013-01-01/documents/batch", "application/json", `[{"type":"add","id":"9","fields":{"year":"soon"}}]`)
	if status != http.StatusBadRequest {
		t.Fatalf("bad int: %d", status)
	}
	status, _ = csdCall(t, ts, http.MethodPost, "/nowhere/2013-01-01/documents/batch", "application/json", batch)
	if status != http.StatusNotFound {
		t.Fatalf("unknown domain: %d", status)
	}

	// XML batches delete too.
	status, out = csdCall(t, ts, http.MethodPost, "/films/2013-01-01/documents/batch", "application/xml",
		`<batch><add id="4"><field name="title">Star Dust</field><field name="year">2007</field></add><delete id="3"/></batch>`)
	if status != http.StatusOK || out["adds"] != float64(1) || out["deletes"] != float64(1) {
		t.Fatalf("xml upload: %d %v", status, out)
	}

	// Simple search, sorted by year, with facets and stats (POST form, like the CLI).
	form := url.Values{"q": {"star"}, "sort": {"year asc"}, "return": {"title,year,_score"},
		"facet": {`{"genre":{}}`}, "stats": {`{"year":{}}`}, "highlight": {`{"title":{}}`}}
	status, out = csdCall(t, ts, http.MethodPost, "/films/2013-01-01/search", "application/x-www-form-urlencoded", form.Encode())
	if status != http.StatusOK {
		t.Fatalf("search: %d %v", status, out)
	}
	hits := out["hits"].(map[string]any)
	list := hits["hit"].([]any)
	if hits["found"] != float64(3) || list[0].(map[string]any)["id"] != "1" || list[2].(map[string]any)["id"] != "4" {
		t.Fatalf("search hits: %v", hits)
	}
	first := list[0].(map[string]any)
	if hl := first["highlights"].(map[string]any)["title"]; hl != "<em>Star</em> Wars" {
		t.Fatalf("highlight: %v", hl)
	}
	buckets := out["facets"].(map[string]any)["genre"].(map[string]any)["buckets"].([]any)
	if len(buckets) != 1 || buckets[0].(map[string]any)["count"] != float64(2) {
		t.Fatalf("facets: %v", buckets)
	}
	if st := out["stats"].(map[string]any)["year"].(map[string]any); st["min"] != "1977" || st["max"] != "2007" {
		t.Fatalf("stats: %v", st)
	}

	// Structured queries, filters, ranges, exprs and cursors (GET).
	q := url.Values{"q": {"(and genre:'scifi' (range field=year [1978,}))"}, "q.parser": {"structured"}}
	status, out = csdCall(t, ts, http.MethodGet, "/films/2013-01-01/search?"+q.Encode(), "", "")
	if status != http.StatusOK || out["hits"].(map[string]any)["found"] != float64(1) {
		t.Fatalf("structured: %d %v", status, out)
	}
	q = url.Values{"q": {"matchall"}, "q.parser": {"structured"}, "fq": {"year:[1970,1980]"}, "size": {"1"}, "cursor": {"initial"},
		"expr": {`{"age":"2026-year"}`}, "sort": {"age desc"}, "return": {"age"}}
	status, out = csdCall(t, ts, http.MethodGet, "/films/2013-01-01/search?"+q.Encode(), "", "")
	hits = out["hits"].(map[string]any)
	if status != http.StatusOK || hits["found"] != float64(2) || hits["cursor"] == nil {
		t.Fatalf("cursor page: %d %v", status, out)
	}
	if e := hits["hit"].([]any)[0].(map[string]any)["exprs"].(map[string]any)["age"]; e != "49" {
		t.Fatalf("expr: %v", e)
	}
	q.Set("cursor", hits["cursor"].(string))
	_, out = csdCall(t, ts, http.MethodGet, "/films/2013-01-01/search?"+q.Encode(), "", "")
	if id := out["hits"].(map[string]any)["hit"].([]any)[0].(map[string]any)["id"]; id != "2" {
		t.Fatalf("second page: %v", out)
	}
	status, _ = csdCall(t, ts, http.MethodGet, "/films/2013-01-01/search?q=(and&q.parser=structured", "", "")
	if status != http.StatusBadRequest {
		t.Fatalf("bad structured query: %d", status)
	}

	// Suggest with fuzzy matching.
	status, out = csdCall(t, ts, http.MethodGet, "/films/2013-01-01/suggest?q=stax&suggester=titles", "", "")
	sug := out["suggest"].(map[string]any)
	if status != http.StatusOK || sug["found"] != float64(3) {
		t.Fatalf("suggest: %d %v", status, out)
	}
	status, _ = csdCall(t, ts, http.MethodGet, "/films/2013-01-01/suggest?q=st&suggester=none", "", "")
	if status != http.StatusBadRequest {
		t.Fatalf("unknown suggester: %d", status)
	}
}

func TestCloudSearchDomainStage1Limits(t *testing.T) {
	st := newCloudSearchStore()
	if _, err := st.createDomain(map[string]any{"DomainName": "big"}); err != nil {
		t.Fatal(err)
	}
	st.domains["big"].fields["body"] = &csOption{deployed: map[string]any{"IndexFieldName": "body", "IndexFieldType": "text"}}
	chunk := strings.Repeat("x", 900<<10)
	var batch []string
	for i := 0; i < 5; i++ {
		batch = append(batch, `{"type":"add","id":"d`+string(rune('a'+i))+`","fields":{"body":"`+chunk+`"}}`)
	}
	raw := []byte("[" + strings.Join(batch, ",") + "]")
	// Fill the domain close to its byte limit, then one more batch must fail
	// without changing anything.
	for i := 0; i < csdMaxDomainBytes/len(raw); i++ {
		d := st.domains["big"]
		for j := 0; j < 5; j++ {
			d.docs["filler"+string(rune('a'+i))+string(rune('a'+j))] = &csDoc{size: len(raw) / 5}
		}
		d.docBytes += len(raw)
	}
	before := st.domains["big"].docBytes
	if _, err := st.uploadDocuments("big", "application/json", raw); err == nil {
		t.Fatal("upload past the domain byte limit was accepted")
	}
	if st.domains["big"].docBytes != before {
		t.Fatal("a refused batch changed stored bytes")
	}
	// A document over 1 MB is refused on its own.
	huge := []byte(`[{"type":"add","id":"h","fields":{"body":"` + strings.Repeat("y", csdMaxDocBytes) + `"}}]`)
	if _, err := st.uploadDocuments("big", "application/json", huge); err == nil {
		t.Fatal("oversized document was accepted")
	}
}
