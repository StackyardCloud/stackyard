// Command chime-sdk-meetings runs Amazon Chime SDK Meetings against
// Stackyard: create a meeting with attendees (each gets a join token once),
// add more, narrow their capabilities, run live transcription, tag the
// meeting and end it.
//
// The SDK has no Chime SDK Meetings module in this repository's go.mod, so
// requests are signed by hand.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

// The Chime SDK APIs all sign as chime.
const signingName = "chime"

func main() {
	c := newClient()
	fmt.Printf("Stackyard Chime SDK Meetings client using %s\n", c.endpoint)
	token := fmt.Sprintf("standup-%d", time.Now().UnixNano())
	out := c.rest("CreateMeetingWithAttendees", http.MethodPost, "/meetings?operation=create-attendees", "", map[string]any{
		"ClientRequestToken": token, "ExternalMeetingId": token, "MediaRegion": "us-east-1",
		"MeetingFeatures": map[string]any{"Video": map[string]any{"MaxResolution": "HD"}, "Attendee": map[string]any{"MaxCount": 10}},
		"Attendees":       []any{map[string]any{"ExternalUserId": "alice@example.com"}, map[string]any{"ExternalUserId": "bob@example.com"}},
		"Tags":            []any{map[string]any{"Key": "team", "Value": "platform"}},
	})
	meeting := out["Meeting"].(map[string]any)
	id := meeting["MeetingId"].(string)
	fmt.Printf("meeting %s in %v, signaling at %v\n", id, meeting["MediaRegion"], meeting["MediaPlacement"].(map[string]any)["SignalingUrl"])
	for _, a := range out["Attendees"].([]any) {
		a := a.(map[string]any)
		fmt.Printf("attendee %v joined with a %d-character join token\n", a["ExternalUserId"], len(a["JoinToken"].(string)))
	}

	base := "/meetings/" + id + "/attendees"
	viewer := c.rest("CreateAttendee", http.MethodPost, base, "", map[string]any{
		"ExternalUserId": "viewer@example.com",
		"Capabilities":   map[string]any{"Audio": "Receive", "Video": "Receive", "Content": "Receive"},
	})["Attendee"].(map[string]any)
	vid := viewer["AttendeeId"].(string)
	got := c.rest("GetAttendee", http.MethodGet, base+"/"+vid, "", nil)["Attendee"].(map[string]any)
	fmt.Printf("viewer %v capabilities %v (join token returned again: %v)\n", got["ExternalUserId"], got["Capabilities"], got["JoinToken"] != nil)
	c.rest("BatchUpdateAttendeeCapabilitiesExcept", http.MethodPut, base+"/capabilities?operation=batch-update-except", "", map[string]any{
		"ExcludedAttendeeIds": []any{map[string]any{"AttendeeId": vid}},
		"Capabilities":        map[string]any{"Audio": "SendReceive", "Video": "Send", "Content": "None"},
	})
	list := c.rest("ListAttendees", http.MethodGet, base+"?max-results=10", "", nil)["Attendees"].([]any)
	fmt.Printf("%d attendees\n", len(list))

	tr := "/meetings/" + id + "/transcription"
	c.rest("StartMeetingTranscription", http.MethodPost, tr+"?operation=start", "", map[string]any{
		"TranscriptionConfiguration": map[string]any{"EngineTranscribeSettings": map[string]any{"LanguageCode": "en-US", "EnablePartialResultsStabilization": true}},
	})
	c.rest("StopMeetingTranscription", http.MethodPost, tr+"?operation=stop", "", nil)
	fmt.Println("transcription started and stopped")

	arn := meeting["MeetingArn"].(string)
	c.rest("TagResource", http.MethodPost, "/tags?operation=tag-resource", "", map[string]any{"ResourceARN": arn, "Tags": []any{map[string]any{"Key": "cost-center", "Value": "42"}}})
	tags := c.rest("ListTagsForResource", http.MethodGet, "/tags?arn="+url.QueryEscape(arn), "", nil)["Tags"].([]any)
	fmt.Printf("meeting tags: %v\n", tags)

	c.rest("DeleteAttendee", http.MethodDelete, base+"/"+vid, "", nil)
	c.rest("DeleteMeeting", http.MethodDelete, "/meetings/"+id, "", nil)
	fmt.Println("meeting ended")
}

type client struct {
	endpoint, region string
	creds            aws.CredentialsProvider
}

func newClient() client {
	return client{
		endpoint: strings.TrimRight(getenv("STACKYARD_ENDPOINT", "http://localhost:4566"), "/"),
		region:   getenv("AWS_REGION", "us-east-1"),
		creds: credentials.NewStaticCredentialsProvider(
			getenv("AWS_ACCESS_KEY_ID", "stackyard"), getenv("AWS_SECRET_ACCESS_KEY", "stackyard"), ""),
	}
}

// rest sends one REST JSON operation signed for chime, acting as bearer
// when it is set, and exits unless it returns 2xx.
func (c client) rest(op, method, path, bearer string, payload any) map[string]any {
	var body []byte
	if payload != nil {
		var err error
		if body, err = json.Marshal(payload); err != nil {
			exitf("%s: %v", op, err)
		}
	}
	headers := map[string]string{"Content-Type": "application/json"}
	if bearer != "" {
		headers["X-Amz-Chime-Bearer"] = bearer
	}
	raw := c.send(op, method, c.endpoint+path, body, headers)
	out := map[string]any{}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			exitf("%s: decode: %v", op, err)
		}
	}
	return out
}

// send signs a request and exits unless it returns 2xx.
func (c client) send(op, method, target string, body []byte, headers map[string]string) []byte {
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, method, target, bytes.NewReader(body))
	if err != nil {
		exitf("%s: %v", op, err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	creds, err := c.creds.Retrieve(ctx)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	sum := sha256.Sum256(body)
	payloadHash := hex.EncodeToString(sum[:])
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)
	if err := v4.NewSigner().SignHTTP(ctx, creds, req, payloadHash, signingName, c.region, time.Now()); err != nil {
		exitf("%s: %v", op, err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		exitf("%s: HTTP %d: %s", op, resp.StatusCode, raw)
	}
	return raw
}

func getenv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
