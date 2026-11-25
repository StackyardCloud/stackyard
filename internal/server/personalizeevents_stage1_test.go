package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// Personalize Events writes into the tracker's or dataset's dataset group and
// refuses records that don't fit the dataset.
func TestPersonalizeEventsStage1PutsLandInDatasets(t *testing.T) {
	srv, ts := newPersonalizeTestServer(t)
	fx := pszFixture(t, srv, ts)
	cases := []struct {
		name, path, body string
		status           int
	}{
		{"items", "/items", fmt.Sprintf(`{"datasetArn":%q,"items":[{"itemId":"i9","properties":"{\"genre\":\"drama\"}"}]}`, fx["items"]), http.StatusOK},
		{"unknown property", "/items", fmt.Sprintf(`{"datasetArn":%q,"items":[{"itemId":"i9","properties":"{\"color\":\"red\"}"}]}`, fx["items"]), http.StatusBadRequest},
		{"wrong dataset type", "/users", fmt.Sprintf(`{"datasetArn":%q,"users":[{"userId":"u9"}]}`, fx["items"]), http.StatusBadRequest},
		{"missing dataset", "/actions", `{"datasetArn":"arn:aws:personalize:us-east-1:123456789012:dataset/none/ACTIONS","actions":[{"actionId":"a"}]}`, http.StatusNotFound},
		{"event", "/events", fmt.Sprintf(`{"trackingId":%q,"userId":"u9","sessionId":"s","eventList":[{"eventType":"click","itemId":"i9","sentAt":1700000500}]}`, fx["trackingId"]), http.StatusOK},
		{"event without item", "/events", fmt.Sprintf(`{"trackingId":%q,"sessionId":"s","eventList":[{"eventType":"click","sentAt":1}]}`, fx["trackingId"]), http.StatusBadRequest},
		{"unknown tracker", "/events", `{"trackingId":"nope","sessionId":"s","eventList":[{"eventType":"click","itemId":"i","sentAt":1}]}`, http.StatusBadRequest},
		{"no action interactions dataset", "/action-interactions", fmt.Sprintf(`{"trackingId":%q,"actionInteractions":[{"actionId":"a","sessionId":"s","timestamp":1,"eventType":"Taken"}]}`, fx["trackingId"]), http.StatusNotFound},
		{"too many items", "/items", fmt.Sprintf(`{"datasetArn":%q,"items":[%s]}`, fx["items"], pszManyItems(11)), http.StatusBadRequest},
	}
	for _, c := range cases {
		status, out := pszREST(t, ts, c.path, c.body)
		if status != c.status {
			t.Errorf("%s: want %d, got %d %v", c.name, c.status, status, out)
		}
	}
	// The streamed event and the put item both reach the dataset group.
	srv.personalize.mu.Lock()
	inter, items := srv.personalize.res[fx["interactions"]].data, srv.personalize.res[fx["items"]].data
	last := inter.events[len(inter.events)-1]
	props, _ := json.Marshal(items.entities["i9"].props)
	srv.personalize.mu.Unlock()
	if last.user != "u9" || last.target != "i9" || !strings.Contains(string(props), `"GENRE":"drama"`) {
		t.Fatalf("stored: %+v %s", last, props)
	}
}

func pszManyItems(n int) string {
	out := ""
	for i := 0; i < n; i++ {
		if i > 0 {
			out += ","
		}
		out += fmt.Sprintf(`{"itemId":"x%d"}`, i)
	}
	return out
}
