package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestKinesisVideoArchivedMediaStage0Catalog(t *testing.T) {
	if len(kinesisVideoArchivedMediaOperations) != 6 || len(kinesisVideoArchivedMediaOperationByName) != 6 {
		t.Fatalf("expected 6 archived media operations, got %d", len(kinesisVideoArchivedMediaOperations))
	}
	for _, op := range kinesisVideoArchivedMediaOperations {
		if kvsDataPlaneRoutes[op.URI] != op.Name {
			t.Fatalf("%s is not routed at %s", op.Name, op.URI)
		}
	}
	ts := newKVSMediaTestServer(t)
	// Unknown streams and missing selectors fail before any media lookup.
	resp := kvsMediaJSON(t, ts, "/listFragments", `{"StreamName":"missing"}`)
	if body := mustBody(t, resp); resp.StatusCode != http.StatusNotFound || !strings.Contains(string(body), "ResourceNotFoundException") {
		t.Fatalf("ListFragments unknown stream: %d %s", resp.StatusCode, body)
	}
	resp = kvsMediaJSON(t, ts, "/getClip", `{"StreamName":"stackyard-kvs-stream"}`)
	assertStatus(t, resp, http.StatusBadRequest)
	// Playback paths without a session token are left to other routers.
	resp, err := http.Get(ts.URL + kvsHLSMasterPath)
	if err != nil {
		t.Fatal(err)
	}
	if body := mustBody(t, resp); strings.Contains(string(body), "session token") {
		t.Fatalf("tokenless playback path should not be claimed: %s", body)
	}
	resp, err = http.Get(ts.URL + kvsHLSMasterPath + "?SessionToken=bogus")
	if err != nil {
		t.Fatal(err)
	}
	assertStatus(t, resp, http.StatusForbidden)
}
