package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// mstdDo sends a MediaStore Data request to a container endpoint.
func mstdDo(t *testing.T, method, url, body string, headers map[string]string) (*http.Response, []byte) {
	t.Helper()
	var raw []byte
	if body != "" {
		raw = []byte(body)
	}
	resp := signedRequestWithService(t, method, url, raw, headers, "mediastore")
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return resp, data
}

func mstdTestContainer(t *testing.T, ts *httptest.Server, name string) string {
	t.Helper()
	mstOK(t, ts, "CreateContainer", `{"ContainerName":"`+name+`"}`)
	return mstOK(t, ts, "DescribeContainer", `{"ContainerName":"`+name+`"}`)["Container"].(map[string]any)["Endpoint"].(string)
}

func TestMediaStoreStage1DataObjects(t *testing.T) {
	ts := newMediaStoreTestServer(t)
	ep := mstdTestContainer(t, ts, "media")

	resp, body := mstdDo(t, http.MethodPut, ep+"/shows/ep1/index.m3u8", "#EXTM3U\n", map[string]string{"Content-Type": "application/vnd.apple.mpegurl", "Cache-Control": "max-age=60"})
	var put map[string]any
	if resp.StatusCode != http.StatusOK || json.Unmarshal(body, &put) != nil || put["StorageClass"] != "TEMPORAL" || len(put["ETag"].(string)) != 64 {
		t.Fatalf("put: %d %s", resp.StatusCode, body)
	}
	mstdDo(t, http.MethodPut, ep+"/shows/ep1/seg1.ts", "0123456789", map[string]string{"Content-Type": "video/MP2T"})
	mstdDo(t, http.MethodPut, ep+"/page.html", "<script>alert(1)</script>", map[string]string{"Content-Type": "text/html"})

	resp, body = mstdDo(t, http.MethodGet, ep+"/shows/ep1/index.m3u8", "", nil)
	if resp.StatusCode != http.StatusOK || string(body) != "#EXTM3U\n" || resp.Header.Get("Content-Type") != "application/vnd.apple.mpegurl" ||
		resp.Header.Get("Cache-Control") != "max-age=60" || resp.Header.Get("ETag") != put["ETag"] {
		t.Fatalf("get: %d %v %s", resp.StatusCode, resp.Header, body)
	}
	// Caller-chosen HTML is never echoed as HTML.
	resp, _ = mstdDo(t, http.MethodGet, ep+"/page.html", "", nil)
	if resp.Header.Get("Content-Type") != "application/octet-stream" || resp.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("html object headers: %v", resp.Header)
	}

	for _, c := range []struct {
		rng, want string
		status    int
	}{
		{"bytes=2-4", "234", http.StatusPartialContent},
		{"bytes=7-", "789", http.StatusPartialContent},
		{"bytes=-2", "89", http.StatusPartialContent},
		{"bytes=20-30", "", http.StatusRequestedRangeNotSatisfiable},
		{"lines=1-2", "", http.StatusBadRequest},
	} {
		resp, body = mstdDo(t, http.MethodGet, ep+"/shows/ep1/seg1.ts", "", map[string]string{"Range": c.rng})
		if resp.StatusCode != c.status || (c.want != "" && string(body) != c.want) {
			t.Fatalf("range %s: %d %s", c.rng, resp.StatusCode, body)
		}
	}

	resp, _ = mstdDo(t, http.MethodHead, ep+"/shows/ep1/seg1.ts", "", nil)
	if resp.StatusCode != http.StatusOK || resp.Header.Get("Content-Length") != "10" {
		t.Fatalf("head: %d %v", resp.StatusCode, resp.Header)
	}

	list := func(query string) map[string]any {
		resp, body := mstdDo(t, http.MethodGet, ep+"/"+query, "", nil)
		out := map[string]any{}
		if resp.StatusCode != http.StatusOK || json.Unmarshal(body, &out) != nil {
			t.Fatalf("list %s: %d %s", query, resp.StatusCode, body)
		}
		return out
	}
	root := list("")["Items"].([]any)
	if len(root) != 2 || root[0].(map[string]any)["Name"] != "page.html" || root[1].(map[string]any)["Type"] != "FOLDER" {
		t.Fatalf("root: %v", root)
	}
	page := list("?Path=shows/ep1&MaxResults=1")
	if len(page["Items"].([]any)) != 1 || page["NextToken"] == nil {
		t.Fatalf("page: %v", page)
	}
	if rest := list("?Path=shows/ep1&NextToken=" + page["NextToken"].(string))["Items"].([]any); len(rest) != 1 || rest[0].(map[string]any)["Name"] != "seg1.ts" {
		t.Fatalf("page 2: %v", rest)
	}

	// An object path can't run through another object.
	if resp, _ := mstdDo(t, http.MethodPut, ep+"/page.html/x", "y", nil); resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("object under object: %d", resp.StatusCode)
	}
	// Non-empty containers can't be deleted.
	mstFail(t, ts, "DeleteContainer", `{"ContainerName":"media"}`, "ContainerInUseException")
	for _, p := range []string{"/shows/ep1/index.m3u8", "/shows/ep1/seg1.ts", "/page.html"} {
		if resp, body := mstdDo(t, http.MethodDelete, ep+p, "", nil); resp.StatusCode != http.StatusOK {
			t.Fatalf("delete %s: %d %s", p, resp.StatusCode, body)
		}
	}
	resp, body = mstdDo(t, http.MethodGet, ep+"/page.html", "", nil)
	if resp.StatusCode != http.StatusNotFound || resp.Header.Get("X-Amzn-ErrorType") != "ObjectNotFoundException" || !strings.Contains(string(body), "page.html") {
		t.Fatalf("deleted object: %d %s", resp.StatusCode, body)
	}
	mstOK(t, ts, "DeleteContainer", `{"ContainerName":"media"}`)
	if resp, _ := mstdDo(t, http.MethodGet, ep+"/", "", nil); resp.Header.Get("X-Amzn-ErrorType") != "ContainerNotFoundException" {
		t.Fatalf("deleted container: %d", resp.StatusCode)
	}
}

func TestMediaStoreStage1DataLimits(t *testing.T) {
	ts := newMediaStoreTestServer(t)
	ep := mstdTestContainer(t, ts, "limits")
	big := strings.Repeat("x", mstdMaxObject+1)
	if resp, _ := mstdDo(t, http.MethodPut, ep+"/big", big, nil); resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("oversized object: %d", resp.StatusCode)
	}
	if resp, _ := mstdDo(t, http.MethodPut, ep+"/a", "x", map[string]string{"Content-Type": "text/<html>"}); resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("bad content type: %d", resp.StatusCode)
	}
	// The store-wide byte cap holds across overwrites and containers (driven
	// through the store: one shared 20 MiB slice, no 256 MiB of HTTP).
	st := newMediaStoreStore()
	for _, name := range []string{"one", "two"} {
		if _, err := st.createContainer(name, nil); err != nil {
			t.Fatal(err)
		}
	}
	chunk := make([]byte, 20<<20)
	n := 0
	for i := 0; i < 20; i++ {
		id := st.containers[[]string{"one", "two"}[i%2]].endpointID
		err := st.putObject(id, fmt.Sprintf("obj%d", i), &mstdObject{data: chunk})
		if err != nil {
			if err.code != "LimitExceededException" {
				t.Fatalf("over the cap: %v", err)
			}
			break
		}
		n++
	}
	if n == 0 || n == 20 || st.storedBytes > mstdMaxStoredBytes {
		t.Fatalf("stored %d chunks (%d bytes) under a %d byte cap", n, st.storedBytes, mstdMaxStoredBytes)
	}
	// Overwriting an existing object reuses its share of the cap.
	if err := st.putObject(st.containers["one"].endpointID, "obj0", &mstdObject{data: chunk}); err != nil {
		t.Fatalf("overwrite at the cap: %v", err)
	}
	if err := st.deleteObject(st.containers["one"].endpointID, "obj0"); err != nil || st.storedObjects != n-1 {
		t.Fatalf("delete: %v, %d objects", err, st.storedObjects)
	}
}
