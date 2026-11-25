package server

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

func TestConnectContactLensStage1Segments(t *testing.T) {
	if len(connectContactLensOperations) != len(awsmodels.ConnectContactLensModel().Operations) {
		t.Fatalf("catalog doesn't match the model")
	}
	_, ts := newConnectTestServer(t)
	contact, conn := cnpStartChat(t, ts, "my order is broken and I am frustrated")
	cnpOK(t, ts, "/participant/message", conn, `{"ContentType":"text/plain","Content":"thanks, that was great"}`)
	cnpOK(t, ts, "/participant/message", conn, `{"ContentType":"application/json","Content":"{}"}`)

	call := func(body string) (int, map[string]any) {
		return cnxDo(t, ts, "connect", http.MethodPost, cnlPath, body, nil)
	}
	status, out := call(fmt.Sprintf(`{"InstanceId":"instance-000001","ContactId":%q,"MaxResults":1}`, contact))
	if status != http.StatusOK {
		t.Fatalf("%d %v", status, out)
	}
	seg := out["Segments"].([]any)[0].(map[string]any)["Transcript"].(map[string]any)
	if seg["Sentiment"] != "NEGATIVE" || seg["ParticipantRole"] != "CUSTOMER" || len(seg["IssuesDetected"].([]any)) != 1 || out["NextToken"] != "1" {
		t.Fatalf("first segment: %v", out)
	}
	status, out = call(fmt.Sprintf(`{"InstanceId":"instance-000001","ContactId":%q,"NextToken":"1"}`, contact))
	segs := out["Segments"].([]any)
	if status != http.StatusOK || len(segs) != 1 || segs[0].(map[string]any)["Transcript"].(map[string]any)["Sentiment"] != "POSITIVE" {
		t.Fatalf("second page (JSON messages aren't transcript): %d %v", status, out)
	}

	for _, c := range []struct {
		body   string
		status int
	}{
		{`{"InstanceId":"instance-000001","ContactId":"nope"}`, http.StatusNotFound},
		{fmt.Sprintf(`{"InstanceId":"nope","ContactId":%q}`, contact), http.StatusNotFound},
		{`{"InstanceId":"instance-000001"}`, http.StatusBadRequest},
		{fmt.Sprintf(`{"InstanceId":"instance-000001","ContactId":%q,"ListFilter":{"CategoriesAttributeFilter":{}}}`, contact), http.StatusBadRequest},
		{fmt.Sprintf(`{"InstanceId":"instance-000001","ContactId":%q,"NextToken":"x"}`, contact), http.StatusBadRequest},
	} {
		if status, out := call(c.body); status != c.status {
			t.Errorf("%s: want %d, got %d %v", c.body, c.status, status, out)
		}
	}

	// Connect (same signing name) keeps its own operations, including the V2 segments path.
	status, out = cnxDo(t, ts, "connect", http.MethodPost, "/contact/list-real-time-analysis-segments-v2/instance-000001/"+contact, `{"OutputType":"Raw","SegmentTypes":["Transcript"]}`, nil)
	if status != http.StatusOK {
		t.Fatalf("connect V2 segments: %d %v", status, out)
	}
	// Signed for another service, the path isn't Contact Lens's.
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+cnlPath, []byte(`{}`), nil, "s3")
	if body := mustBody(t, resp); strings.Contains(string(body), "Segments") || strings.Contains(string(body), "InvalidRequestException") {
		t.Fatalf("s3-signed request answered by Contact Lens: %s", body)
	}
}
