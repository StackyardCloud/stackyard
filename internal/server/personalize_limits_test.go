package server

import (
	"strconv"
	"strings"
	"testing"
)

func TestPSZAddEventsKeepsNewestWithinCap(t *testing.T) {
	d := newPSZData(pszInteractions, nil)
	es := make([]pszEvent, pszMaxEvents+5)
	for i := range es {
		es[i].user = strconv.Itoa(i)
	}
	d.addEvents(es)
	if len(d.events) != pszMaxEvents || d.events[0].user != "5" {
		t.Fatalf("got %d events, first user=%q; want %d starting at 5", len(d.events), d.events[0].user, pszMaxEvents)
	}
}

func TestPSZLinesCapsTotalOutput(t *testing.T) {
	big := strings.Repeat("x", 1<<20)
	in := pszS3File{key: "in.json", body: []byte(strings.Repeat("{}\n", 16))}
	_, err := pszLines([]pszS3File{in}, func(map[string]any) map[string]any {
		return map[string]any{"output": big}
	})
	if err == nil || !strings.Contains(err.Error(), "bytes of batch output") {
		t.Fatalf("want output-cap error, got %v", err)
	}
}
