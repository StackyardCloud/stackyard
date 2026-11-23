package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestNetworkManagerPolicyListsAreCapped(t *testing.T) {
	segs := make([]map[string]any, 20000)
	for i := range segs {
		segs[i] = map[string]any{"name": "seg" + strings.Repeat("x", i%7)}
	}
	edges := make([]map[string]any, 20000)
	for i := range edges {
		edges[i] = map[string]any{"location": "us-east-1"}
	}
	raw, _ := json.Marshal(map[string]any{
		"version":                    "2021.12",
		"core-network-configuration": map[string]any{"asn-ranges": []string{"64512-65534"}, "edge-locations": edges},
		"segments":                   segs,
	})
	start := time.Now()
	if _, err := nmParsePolicy(string(raw)); err == nil {
		t.Fatal("an oversized policy was accepted")
	}
	if d := time.Since(start); d > 2*time.Second {
		t.Fatalf("parsing an oversized policy took %s", d)
	}
}

func TestNetworkManagerPolicyBytesAreBoundedAcrossCoreNetworks(t *testing.T) {
	st := newNetworkManagerStore()
	doc := strings.Repeat("x", 1<<20)
	for core := 0; core < 5; core++ {
		for v := 0; v < 40; v++ {
			st.addPolicy("core-"+string(rune('a'+core)), doc, "", nil)
		}
	}
	total := 0
	for _, vs := range st.policies {
		for _, p := range vs {
			total += len(p.document)
		}
	}
	if total > nmMaxPolicyBytes {
		t.Fatalf("stored %d policy bytes, cap %d", total, nmMaxPolicyBytes)
	}
}

// An empty LinkId must be refused, not dereference a missing link.
func TestNetworkManagerAssociateLinkWithoutLink(t *testing.T) {
	srv, ts := newNetworkManagerTestServer(t)
	gnID := nmCall(t, ts, http.StatusOK, http.MethodPost, "/global-networks", `{}`)["GlobalNetwork"].(map[string]any)["GlobalNetworkId"].(string)
	dev := nmCall(t, ts, http.StatusOK, http.MethodPost, "/global-networks/"+gnID+"/devices", `{}`)["Device"].(map[string]any)["DeviceId"].(string)
	srv.networkManager.mu.Lock()
	_, err := nmAssociateLink(srv.networkManager, map[string]any{"GlobalNetworkId": gnID, "DeviceId": dev})
	srv.networkManager.mu.Unlock()
	if err == nil {
		t.Fatal("AssociateLink without a LinkId succeeded")
	}
}
