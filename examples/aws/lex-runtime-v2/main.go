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
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

type client struct {
	ctx      context.Context
	endpoint string
	region   string
	creds    aws.CredentialsProvider
}

type reply struct {
	status int
	body   []byte
}

// main builds a hotel-booking bot with the Lex V2 models API, then books a
// room through Lex Runtime V2: the bot recognizes the intent, asks for each
// required slot, and closes once both are filled.
func main() {
	c := client{
		ctx:      context.Background(),
		endpoint: strings.TrimRight(getenv("STACKYARD_ENDPOINT", "http://localhost:4566"), "/"),
		region:   getenv("AWS_REGION", "us-east-1"),
		creds: credentials.NewStaticCredentialsProvider(
			getenv("AWS_ACCESS_KEY_ID", "stackyard"),
			getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
			"",
		),
	}
	fmt.Printf("Stackyard Lex Runtime V2 client using %s\n", c.endpoint)

	botID := c.field("CreateBot", http.MethodPut, "/bots/", map[string]any{
		"botName": "hotel-bot", "roleArn": "arn:aws:iam::123456789012:role/lex", "idleSessionTTLInSeconds": 300,
		"dataPrivacy": map[string]any{"childDirected": false},
	}, "botId")
	draft := "/bots/" + botID + "/botversions/DRAFT/botlocales/"
	c.field("CreateBotLocale", http.MethodPut, draft, map[string]any{"localeId": "en_US", "nluIntentConfidenceThreshold": 0.4}, "localeId")
	intentID := c.field("CreateIntent", http.MethodPut, draft+"en_US/intents/", map[string]any{
		"intentName":       "BookHotel",
		"sampleUtterances": []map[string]string{{"utterance": "book a hotel"}, {"utterance": "book a {RoomType} room"}},
	}, "intentId")
	for _, slot := range []struct{ name, prompt string }{{"RoomType", "Which room type?"}, {"Nights", "How many nights?"}} {
		c.field("CreateSlot", http.MethodPut, draft+"en_US/intents/"+intentID+"/slots/", map[string]any{
			"slotName": slot.name, "slotTypeId": "AMAZON.AlphaNumeric",
			"valueElicitationSetting": map[string]any{
				"slotConstraint": "Required",
				"promptSpecification": map[string]any{"maxRetries": 1, "messageGroups": []any{
					map[string]any{"message": map[string]any{"plainTextMessage": map[string]any{"value": slot.prompt}}},
				}},
			},
		}, "slotId")
	}

	// TSTALIASID is the test alias every bot has, pointing at its DRAFT version.
	session := "/bots/" + botID + "/botAliases/TSTALIASID/botLocales/en_US/sessions/guest-1"
	say := func(text, wantAction string) map[string]any {
		r := c.json(http.MethodPost, session+"/text", map[string]any{"text": text})
		expect2xx("RecognizeText", r)
		var out struct {
			Messages     []struct{ Content string }
			SessionState map[string]any
		}
		if err := json.Unmarshal(r.body, &out); err != nil {
			exitf("RecognizeText: %v", err)
		}
		action, _ := out.SessionState["dialogAction"].(map[string]any)
		if action["type"] != wantAction {
			exitf("after %q expected dialog action %s, got %v", text, wantAction, action)
		}
		prompt := ""
		if len(out.Messages) > 0 {
			prompt = out.Messages[0].Content
		}
		logf("user: %-14q -> %s %v %q", text, action["type"], action["slotToElicit"], prompt)
		return out.SessionState
	}
	say("book a hotel", "ElicitSlot")
	say("king", "ElicitSlot")
	state := say("2", "Close")
	intent, _ := state["intent"].(map[string]any)
	if intent["state"] != "ReadyForFulfillment" {
		exitf("expected ReadyForFulfillment, got %v", intent["state"])
	}

	expect2xx("GetSession", c.do(http.MethodGet, session, nil, nil))
	expect2xx("DeleteSession", c.do(http.MethodDelete, session, nil, nil))
	if r := c.do(http.MethodGet, session, nil, nil); r.status != http.StatusNotFound {
		exitf("expected the deleted session to be gone, got %d", r.status)
	}
	fmt.Println("Lex Runtime V2 example completed")
}

func (c client) field(action, method, path string, payload map[string]any, field string) string {
	r := c.json(method, path, payload)
	expect2xx(action, r)
	var out map[string]any
	if err := json.Unmarshal(r.body, &out); err != nil {
		exitf("%s: %v", action, err)
	}
	v, _ := out[field].(string)
	if v == "" {
		exitf("%s returned no %s: %s", action, field, r.body)
	}
	logf("%s %s=%s", action, field, v)
	return v
}

func (c client) json(method, path string, payload any) reply {
	body, err := json.Marshal(payload)
	if err != nil {
		exitf("marshal %s: %v", path, err)
	}
	return c.do(method, path, body, map[string]string{"Content-Type": "application/json"})
}

// do sends a request signed for "lex", the signing name of both the Lex V2
// models API and the runtime.
func (c client) do(method, path string, body []byte, headers map[string]string) reply {
	req, err := http.NewRequestWithContext(c.ctx, method, c.endpoint+path, bytes.NewReader(body))
	if err != nil {
		exitf("build %s %s: %v", method, path, err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	credValue, err := c.creds.Retrieve(c.ctx)
	if err != nil {
		exitf("credentials: %v", err)
	}
	sum := sha256.Sum256(body)
	if err := v4.NewSigner().SignHTTP(c.ctx, credValue, req, hex.EncodeToString(sum[:]), "lex", c.region, time.Now()); err != nil {
		exitf("sign %s %s: %v", method, path, err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		exitf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		exitf("read %s %s: %v", method, path, err)
	}
	return reply{status: resp.StatusCode, body: respBody}
}

func expect2xx(action string, r reply) {
	if r.status >= 200 && r.status < 300 {
		return
	}
	exitf("expected %s to return 2xx, got %d: %s", action, r.status, strings.TrimSpace(string(r.body)))
}

func getenv(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return def
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func logf(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}
