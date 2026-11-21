package server

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func cfKVSTestCreate(t *testing.T, ts *httptest.Server, name string) string {
	t.Helper()
	body := []byte(`<CreateKeyValueStoreRequest xmlns="http://cloudfront.amazonaws.com/doc/2020-05-31/"><Name>` + name + `</Name><Comment>c</Comment></CreateKeyValueStoreRequest>`)
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/2020-05-31/key-value-store", body, map[string]string{"Content-Type": "application/xml"}, "cloudfront")
	assertStatus(t, resp, http.StatusCreated)
	var out cloudFrontKVSXML
	if err := xml.Unmarshal(mustBody(t, resp), &out); err != nil || out.ARN == "" {
		t.Fatalf("create: %+v %v", out, err)
	}
	return out.ARN
}

func cfKVSURL(ts *httptest.Server, arn string, rest ...string) string {
	u := ts.URL + "/key-value-stores/" + url.PathEscape(arn)
	for _, part := range rest {
		u += "/" + url.PathEscape(part)
	}
	return u
}

func cfKVSDo(t *testing.T, method, u, ifMatch string, body any) (*http.Response, map[string]any) {
	t.Helper()
	var raw []byte
	if body != nil {
		raw, _ = json.Marshal(body)
	}
	headers := map[string]string{"Content-Type": "application/json"}
	if ifMatch != "" {
		headers["If-Match"] = ifMatch
	}
	resp := signedRequestWithService(t, method, u, raw, headers, cfKVSSigningName)
	out := map[string]any{}
	_ = json.Unmarshal(mustBody(t, resp), &out)
	return resp, out
}

func cfKVSETag(t *testing.T, ts *httptest.Server, arn string) string {
	t.Helper()
	resp, _ := cfKVSDo(t, http.MethodGet, cfKVSURL(ts, arn), "", nil)
	assertStatus(t, resp, http.StatusOK)
	return resp.Header.Get("ETag")
}

func TestCloudFrontKeyValueStoreStage1KeyLifecycle(t *testing.T) {
	ts := newCFKVSTestServer(t)
	arn := cfKVSTestCreate(t, ts, "life")

	etag := cfKVSETag(t, ts, arn)
	resp, out := cfKVSDo(t, http.MethodPut, cfKVSURL(ts, arn, "keys", "a/b c"), etag, map[string]string{"Value": "v1"})
	assertStatus(t, resp, http.StatusOK)
	if out["ItemCount"] != float64(1) || out["TotalSizeInBytes"] != float64(7) {
		t.Fatalf("put stats: %v", out)
	}
	newETag := resp.Header.Get("ETag")
	if newETag == "" || newETag == etag {
		t.Fatalf("ETag did not change: %q -> %q", etag, newETag)
	}

	// A stale If-Match is a conflict and changes nothing.
	resp, out = cfKVSDo(t, http.MethodPut, cfKVSURL(ts, arn, "keys", "x"), etag, map[string]string{"Value": "v"})
	assertStatus(t, resp, http.StatusConflict)
	if out["__type"] != "ConflictException" {
		t.Fatalf("stale etag: %v", out)
	}
	resp, _ = cfKVSDo(t, http.MethodPut, cfKVSURL(ts, arn, "keys", "x"), "", map[string]string{"Value": "v"})
	assertStatus(t, resp, http.StatusBadRequest)

	resp, out = cfKVSDo(t, http.MethodGet, cfKVSURL(ts, arn, "keys", "a/b c"), "", nil)
	assertStatus(t, resp, http.StatusOK)
	if out["Key"] != "a/b c" || out["Value"] != "v1" {
		t.Fatalf("get: %v", out)
	}

	resp, _ = cfKVSDo(t, http.MethodDelete, cfKVSURL(ts, arn, "keys", "a/b c"), newETag, nil)
	assertStatus(t, resp, http.StatusOK)
	resp, out = cfKVSDo(t, http.MethodGet, cfKVSURL(ts, arn, "keys", "a/b c"), "", nil)
	assertStatus(t, resp, http.StatusNotFound)
	if out["__type"] != "ResourceNotFoundException" {
		t.Fatalf("get missing: %v", out)
	}
	resp, _ = cfKVSDo(t, http.MethodDelete, cfKVSURL(ts, arn, "keys", "a/b c"), cfKVSETag(t, ts, arn), nil)
	assertStatus(t, resp, http.StatusNotFound)

	resp, _ = cfKVSDo(t, http.MethodGet, cfKVSURL(ts, "arn:aws:cloudfront::123456789012:key-value-store/missing"), "", nil)
	assertStatus(t, resp, http.StatusNotFound)
	resp, _ = signedRequestWithServiceBody(t, http.MethodPut, cfKVSURL(ts, arn, "keys", "k"), "{bad", cfKVSETag(t, ts, arn))
	assertStatus(t, resp, http.StatusBadRequest)
}

func signedRequestWithServiceBody(t *testing.T, method, u, body, ifMatch string) (*http.Response, string) {
	t.Helper()
	resp := signedRequestWithService(t, method, u, []byte(body), map[string]string{"If-Match": ifMatch}, cfKVSSigningName)
	return resp, string(mustBody(t, resp))
}

func TestCloudFrontKeyValueStoreStage1UpdateKeysIsAtomic(t *testing.T) {
	ts := newCFKVSTestServer(t)
	arn := cfKVSTestCreate(t, ts, "atomic")
	keysURL := cfKVSURL(ts, arn, "keys")

	resp, out := cfKVSDo(t, http.MethodPost, keysURL, cfKVSETag(t, ts, arn), map[string]any{
		"Puts": []map[string]string{{"Key": "k1", "Value": "1"}, {"Key": "k2", "Value": "2"}},
	})
	assertStatus(t, resp, http.StatusOK)
	if out["ItemCount"] != float64(2) {
		t.Fatalf("update: %v", out)
	}

	cases := []struct {
		name   string
		body   map[string]any
		status int
	}{
		{"delete of a missing key", map[string]any{"Puts": []map[string]string{{"Key": "k3", "Value": "3"}}, "Deletes": []map[string]string{{"Key": "nope"}}}, http.StatusNotFound},
		{"key in puts and deletes", map[string]any{"Puts": []map[string]string{{"Key": "k1", "Value": "x"}}, "Deletes": []map[string]string{{"Key": "k1"}}}, http.StatusBadRequest},
		{"oversized value", map[string]any{"Puts": []map[string]string{{"Key": "k4", "Value": strings.Repeat("v", cfKVSMaxValueBytes+1)}}}, http.StatusBadRequest},
		{"oversized key", map[string]any{"Puts": []map[string]string{{"Key": strings.Repeat("k", cfKVSMaxKeyBytes+1), "Value": "v"}}}, http.StatusBadRequest},
		{"put without value", map[string]any{"Puts": []map[string]string{{"Key": "k5"}}}, http.StatusBadRequest},
		{"empty update", map[string]any{}, http.StatusBadRequest},
	}
	for _, tc := range cases {
		etag := cfKVSETag(t, ts, arn)
		resp, _ := cfKVSDo(t, http.MethodPost, keysURL, etag, tc.body)
		if resp.StatusCode != tc.status {
			t.Fatalf("%s: status %d, want %d", tc.name, resp.StatusCode, tc.status)
		}
		if after := cfKVSETag(t, ts, arn); after != etag {
			t.Fatalf("%s: rejected update changed the ETag", tc.name)
		}
	}
	resp, _ = cfKVSDo(t, http.MethodGet, cfKVSURL(ts, arn, "keys", "k3"), "", nil)
	assertStatus(t, resp, http.StatusNotFound)

	resp, out = cfKVSDo(t, http.MethodPost, keysURL, cfKVSETag(t, ts, arn), map[string]any{
		"Puts": []map[string]string{{"Key": "k1", "Value": "one"}}, "Deletes": []map[string]string{{"Key": "k2"}},
	})
	assertStatus(t, resp, http.StatusOK)
	if out["ItemCount"] != float64(1) || out["TotalSizeInBytes"] != float64(5) {
		t.Fatalf("mixed update: %v", out)
	}
}

func TestCloudFrontKeyValueStoreStage1StoreSizeQuota(t *testing.T) {
	ts := newCFKVSTestServer(t)
	arn := cfKVSTestCreate(t, ts, "quota")
	value := strings.Repeat("v", cfKVSMaxValueBytes)
	// Fill the store just under 5 MB with full-size values in large batches.
	perItem := len("key-0000000") + cfKVSMaxValueBytes
	total := 0
	for batch := 0; total+perItem <= cfKVSMaxStoreBytes; batch++ {
		puts := []map[string]string{}
		for len(puts) < 2000 && total+perItem <= cfKVSMaxStoreBytes {
			puts = append(puts, map[string]string{"Key": fmt.Sprintf("key-%07d", total/perItem), "Value": value})
			total += perItem
		}
		resp, _ := cfKVSDo(t, http.MethodPost, cfKVSURL(ts, arn, "keys"), cfKVSETag(t, ts, arn), map[string]any{"Puts": puts})
		assertStatus(t, resp, http.StatusOK)
	}
	resp, out := cfKVSDo(t, http.MethodPut, cfKVSURL(ts, arn, "keys", "one-more"), cfKVSETag(t, ts, arn), map[string]string{"Value": value})
	assertStatus(t, resp, http.StatusPaymentRequired)
	if out["__type"] != "ServiceQuotaExceededException" {
		t.Fatalf("quota: %v", out)
	}
}

func TestCloudFrontKeyValueStoreStage1ListKeysPagination(t *testing.T) {
	ts := newCFKVSTestServer(t)
	arn := cfKVSTestCreate(t, ts, "pages")
	puts := []map[string]string{}
	for i := 0; i < 25; i++ {
		puts = append(puts, map[string]string{"Key": fmt.Sprintf("k%02d", i), "Value": "v"})
	}
	resp, _ := cfKVSDo(t, http.MethodPost, cfKVSURL(ts, arn, "keys"), cfKVSETag(t, ts, arn), map[string]any{"Puts": puts})
	assertStatus(t, resp, http.StatusOK)

	var seen []string
	token := ""
	for pages := 0; ; pages++ {
		u := cfKVSURL(ts, arn, "keys") + "?MaxResults=10"
		if token != "" {
			u += "&NextToken=" + url.QueryEscape(token)
		}
		resp, _ := cfKVSDo(t, http.MethodGet, u, "", nil)
		assertStatus(t, resp, http.StatusOK)
		var page struct {
			Items     []struct{ Key, Value string }
			NextToken string
		}
		// cfKVSDo already drained the body; re-request for typed decoding.
		resp = signedRequestWithService(t, http.MethodGet, u, nil, nil, cfKVSSigningName)
		if err := json.Unmarshal(mustBody(t, resp), &page); err != nil {
			t.Fatal(err)
		}
		for _, item := range page.Items {
			seen = append(seen, item.Key)
		}
		if page.NextToken == "" {
			break
		}
		token = page.NextToken
		if pages > 5 {
			t.Fatal("pagination did not terminate")
		}
	}
	if len(seen) != 25 || seen[0] != "k00" || seen[24] != "k24" {
		t.Fatalf("pages returned %v", seen)
	}

	for _, q := range []string{"?MaxResults=0", "?MaxResults=51", "?NextToken=%25%25"} {
		resp, _ := cfKVSDo(t, http.MethodGet, cfKVSURL(ts, arn, "keys")+q, "", nil)
		assertStatus(t, resp, http.StatusBadRequest)
	}
}

func TestCloudFrontKeyValueStoreStage1ControlPlane(t *testing.T) {
	ts := newCFKVSTestServer(t)
	arn := cfKVSTestCreate(t, ts, "cp")
	base := ts.URL + "/2020-05-31/key-value-store"

	dup := signedRequestWithService(t, http.MethodPost, base, []byte(`<CreateKeyValueStoreRequest><Name>cp</Name></CreateKeyValueStoreRequest>`), nil, "cloudfront")
	assertStatus(t, dup, http.StatusConflict)
	if body := mustBody(t, dup); !strings.Contains(string(body), "EntityAlreadyExists") {
		t.Fatalf("duplicate: %s", body)
	}
	bad := signedRequestWithService(t, http.MethodPost, base, []byte(`<CreateKeyValueStoreRequest><Name>bad name!</Name></CreateKeyValueStoreRequest>`), nil, "cloudfront")
	assertStatus(t, bad, http.StatusBadRequest)
	bad.Body.Close()

	resp := signedRequestWithService(t, http.MethodGet, base+"/cp", nil, nil, "cloudfront")
	assertStatus(t, resp, http.StatusOK)
	etag := resp.Header.Get("ETag")
	var got cloudFrontKVSXML
	if err := xml.Unmarshal(mustBody(t, resp), &got); err != nil || got.ARN != arn || got.Status != "READY" {
		t.Fatalf("describe: %+v %v", got, err)
	}

	update := []byte(`<UpdateKeyValueStoreRequest><Comment>new</Comment></UpdateKeyValueStoreRequest>`)
	resp = signedRequestWithService(t, http.MethodPut, base+"/cp", update, map[string]string{"If-Match": "stale"}, "cloudfront")
	assertStatus(t, resp, http.StatusPreconditionFailed)
	resp.Body.Close()
	resp = signedRequestWithService(t, http.MethodPut, base+"/cp", update, map[string]string{"If-Match": etag}, "cloudfront")
	assertStatus(t, resp, http.StatusOK)
	etag = resp.Header.Get("ETag")
	if body := mustBody(t, resp); !strings.Contains(string(body), "<Comment>new</Comment>") {
		t.Fatalf("update: %s", body)
	}

	resp = signedRequestWithService(t, http.MethodGet, base+"?MaxItems=1", nil, nil, "cloudfront")
	assertStatus(t, resp, http.StatusOK)
	var list cloudFrontKVSListXML
	if err := xml.Unmarshal(mustBody(t, resp), &list); err != nil || list.Quantity != 1 || list.Items[0].Name != "cp" {
		t.Fatalf("list: %+v %v", list, err)
	}

	resp = signedRequestWithService(t, http.MethodDelete, base+"/cp", nil, nil, "cloudfront")
	assertStatus(t, resp, http.StatusBadRequest)
	resp.Body.Close()
	resp = signedRequestWithService(t, http.MethodDelete, base+"/cp", nil, map[string]string{"If-Match": etag}, "cloudfront")
	assertStatus(t, resp, http.StatusNoContent)
	resp.Body.Close()

	// The data plane no longer finds the deleted store.
	resp, _ = cfKVSDo(t, http.MethodGet, cfKVSURL(ts, arn), "", nil)
	assertStatus(t, resp, http.StatusNotFound)
	resp = signedRequestWithService(t, http.MethodGet, base+"/cp", nil, nil, "cloudfront")
	assertStatus(t, resp, http.StatusNotFound)
	resp.Body.Close()
}
