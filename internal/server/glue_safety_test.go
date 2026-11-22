package server

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
)

// Responses are shaped under the store lock: concurrent reads of a table
// while it is being updated must not race (run with -race).
func TestGlueConcurrentReadsAndUpdatesDoNotRace(t *testing.T) {
	srv, _ := newGlueTestServer(t)
	st := srv.glue
	mustOK := func(op, out string, in glueRecord) {
		if _, err := st.handle(op, out, in); err != nil {
			t.Errorf("%s: %v", op, err.msg)
		}
	}
	mustOK("CreateDatabase", "CreateDatabaseResponse", glueRecord{"DatabaseInput": map[string]any{"Name": "racedb"}})
	mustOK("CreateTable", "CreateTableResponse", glueRecord{"DatabaseName": "racedb", "TableInput": map[string]any{"Name": "t"}})
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			mustOK("GetTable", "GetTableResponse", glueRecord{"DatabaseName": "racedb", "Name": "t"})
		}()
		go func(i int) {
			defer wg.Done()
			mustOK("UpdateTable", "UpdateTableResponse", glueRecord{"DatabaseName": "racedb", "TableInput": map[string]any{"Name": "t", "Description": fmt.Sprint(i)}})
		}(i)
	}
	wg.Wait()
}

// Store bookkeeping fields start with "_"; a request can't set them (a bad
// _next used to reach an unchecked type assertion and panic).
func TestGlueRequestCannotSetInternalFields(t *testing.T) {
	_, ts := newGlueTestServer(t)
	// _crawls/_started drive crawler metrics; set from a request they used
	// to make GetCrawlerMetrics panic on the _started type assertion.
	glueCall(t, ts, http.StatusOK, "CreateCrawler", `{"Name":"c1","Role":"arn:aws:iam::123456789012:role/glue","Targets":{"S3Targets":[{"Path":"s3://b/p"}]},"_crawls":[{"State":"RUNNING"}],"_started":"boom"}`)
	glueCall(t, ts, http.StatusOK, "GetCrawlerMetrics", `{"CrawlerNameList":["c1"]}`)
	if got := glueCall(t, ts, http.StatusOK, "GetCrawler", `{"Name":"c1"}`)["Crawler"].(map[string]any); got["_started"] != nil || got["_crawls"] != nil {
		t.Fatalf("internal fields leaked into the crawler: %v", got)
	}
}

func TestGlueHandleDropsUnderscoreRequestKeys(t *testing.T) {
	srv, _ := newGlueTestServer(t)
	in := glueRecord{"DatabaseInput": map[string]any{"Name": "uscore"}, "_indexes": "boom"}
	if _, err := srv.glue.handle("CreateDatabase", "CreateDatabaseResponse", in); err != nil {
		t.Fatal(err.msg)
	}
	if _, ok := in["_indexes"]; ok {
		t.Fatal("handle kept a caller-supplied _ key")
	}
}

func TestGlueRefusesBodiesWithoutDeclaredLength(t *testing.T) {
	_, ts := newGlueTestServer(t)
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/", io.MultiReader(strings.NewReader(`{}`)))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Amz-Target", "AWSGlue.GetDatabases")
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("chunked body: %d", resp.StatusCode)
	}
}
