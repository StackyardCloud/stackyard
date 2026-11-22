package server

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

const pipesTestRole = "arn:aws:iam::123456789012:role/pipe"

func pipesCreateQueue(t *testing.T, srv *Server, name string) string {
	t.Helper()
	if _, err := srv.sqs.CreateQueue(name, nil); err != nil {
		t.Fatalf("create queue %s: %v", name, err)
	}
	return "arn:aws:sqs:us-east-1:123456789012:" + name
}

func TestPipesStage1Lifecycle(t *testing.T) {
	srv, ts := newPipesTestServer(t)
	src, dst := pipesCreateQueue(t, srv, "pipe-src"), pipesCreateQueue(t, srv, "pipe-dst")
	body := fmt.Sprintf(`{"Source":%q,"Target":%q,"RoleArn":%q,"Description":"d","Tags":{"team":"a"},
		"SourceParameters":{"SqsQueueParameters":{"BatchSize":5},"FilterCriteria":{"Filters":[{"Pattern":"{\"body\":{\"k\":[\"v\"]}}"}]}},
		"TargetParameters":{"SqsQueueParameters":{"MessageGroupId":"g"},"InputTemplate":"{}"}}`, src, dst, pipesTestRole)

	created := pipesCall(t, ts, http.StatusOK, http.MethodPost, "/v1/pipes/p1", body)
	if created["CurrentState"] != "CREATING" || created["DesiredState"] != "RUNNING" || created["Arn"] != pipesARN("p1") {
		t.Fatalf("create: %v", created)
	}
	pipesCall(t, ts, http.StatusConflict, http.MethodPost, "/v1/pipes/p1", body)

	got := pipesCall(t, ts, http.StatusOK, http.MethodGet, "/v1/pipes/p1", "")
	if got["CurrentState"] != "RUNNING" || got["Source"] != src || got["Description"] != "d" || got["Tags"].(map[string]any)["team"] != "a" {
		t.Fatalf("describe: %v", got)
	}
	if got["SourceParameters"].(map[string]any)["SqsQueueParameters"].(map[string]any)["BatchSize"] != float64(5) {
		t.Fatalf("source parameters not kept: %v", got["SourceParameters"])
	}

	stopped := pipesCall(t, ts, http.StatusOK, http.MethodPost, "/v1/pipes/p1/stop", "")
	if stopped["CurrentState"] != "STOPPING" || stopped["DesiredState"] != "STOPPED" {
		t.Fatalf("stop: %v", stopped)
	}
	if got := pipesCall(t, ts, http.StatusOK, http.MethodGet, "/v1/pipes/p1", ""); got["CurrentState"] != "STOPPED" {
		t.Fatalf("after stop: %v", got["CurrentState"])
	}
	if started := pipesCall(t, ts, http.StatusOK, http.MethodPost, "/v1/pipes/p1/start", ""); started["CurrentState"] != "STARTING" {
		t.Fatalf("start: %v", started)
	}
	// A change can't start while the pipe is still STARTING.
	pipesCall(t, ts, http.StatusConflict, http.MethodPost, "/v1/pipes/p1/stop", "")

	pipesCall(t, ts, http.StatusOK, http.MethodGet, "/v1/pipes/p1", "")
	updated := pipesCall(t, ts, http.StatusOK, http.MethodPut, "/v1/pipes/p1", fmt.Sprintf(`{"RoleArn":%q,"Description":"d2","SourceParameters":{"SqsQueueParameters":{"MaximumBatchingWindowInSeconds":3}}}`, pipesTestRole))
	if updated["CurrentState"] != "UPDATING" {
		t.Fatalf("update: %v", updated)
	}
	pipesCall(t, ts, http.StatusConflict, http.MethodDelete, "/v1/pipes/p1", "")
	got = pipesCall(t, ts, http.StatusOK, http.MethodGet, "/v1/pipes/p1", "")
	sqsParams := got["SourceParameters"].(map[string]any)["SqsQueueParameters"].(map[string]any)
	if got["Description"] != "d2" || sqsParams["BatchSize"] != float64(5) || sqsParams["MaximumBatchingWindowInSeconds"] != float64(3) {
		t.Fatalf("update not merged: %v", got)
	}

	// Tagging.
	tagPath := "/tags/" + url.PathEscape(pipesARN("p1"))
	pipesCall(t, ts, http.StatusOK, http.MethodPost, tagPath, `{"tags":{"env":"dev"}}`)
	tags := pipesCall(t, ts, http.StatusOK, http.MethodGet, tagPath, "")["tags"].(map[string]any)
	if tags["env"] != "dev" || tags["team"] != "a" {
		t.Fatalf("tags: %v", tags)
	}
	pipesCall(t, ts, http.StatusOK, http.MethodDelete, tagPath+"?tagKeys=team", "")
	if tags := pipesCall(t, ts, http.StatusOK, http.MethodGet, tagPath, "")["tags"].(map[string]any); tags["team"] != nil {
		t.Fatalf("untag: %v", tags)
	}
	pipesCall(t, ts, http.StatusNotFound, http.MethodGet, "/tags/"+url.PathEscape(pipesARN("nope")), "")

	pipesCall(t, ts, http.StatusOK, http.MethodGet, "/v1/pipes/p1", "")
	deleted := pipesCall(t, ts, http.StatusOK, http.MethodDelete, "/v1/pipes/p1", "")
	if deleted["CurrentState"] != "DELETING" || deleted["DesiredState"] != "DELETED" {
		t.Fatalf("delete: %v", deleted)
	}
	pipesCall(t, ts, http.StatusNotFound, http.MethodGet, "/v1/pipes/p1", "")
}

func TestPipesStage1Validation(t *testing.T) {
	srv, ts := newPipesTestServer(t)
	src, dst := pipesCreateQueue(t, srv, "v-src"), pipesCreateQueue(t, srv, "v-dst")
	base := func(extra string) string {
		return fmt.Sprintf(`{"Source":%q,"Target":%q,"RoleArn":%q%s}`, src, dst, pipesTestRole, extra)
	}
	for name, body := range map[string]string{
		"missing target":          fmt.Sprintf(`{"Source":%q,"RoleArn":%q}`, src, pipesTestRole),
		"bad role":                fmt.Sprintf(`{"Source":%q,"Target":%q,"RoleArn":"role"}`, src, dst),
		"batch size":              base(`,"SourceParameters":{"SqsQueueParameters":{"BatchSize":0}}`),
		"wrong source params":     base(`,"SourceParameters":{"KinesisStreamParameters":{"StartingPosition":"LATEST"}}`),
		"wrong target params":     base(`,"TargetParameters":{"LambdaFunctionParameters":{}}`),
		"bad filter":              base(`,"SourceParameters":{"FilterCriteria":{"Filters":[{"Pattern":"not json"}]}}`),
		"unknown member":          base(`,"Bogus":1`),
		"desired state":           base(`,"DesiredState":"PAUSED"`),
		"enrichment type":         base(`,"Enrichment":"arn:aws:sqs:us-east-1:123456789012:x"`),
		"log destination":         base(`,"LogConfiguration":{"Level":"ERROR"}`),
		"unsupported source":      fmt.Sprintf(`{"Source":"arn:aws:s3:::bucket","Target":%q,"RoleArn":%q}`, dst, pipesTestRole),
		"aws tag":                 base(`,"Tags":{"aws:x":"y"}`),
		"kinesis needs position":  fmt.Sprintf(`{"Source":"arn:aws:kinesis:us-east-1:123456789012:stream/s","Target":%q,"RoleArn":%q,"SourceParameters":{"KinesisStreamParameters":{"BatchSize":1}}}`, dst, pipesTestRole),
		"kinesis target position": base(`,"TargetParameters":{"KinesisStreamParameters":{}}`),
	} {
		out := pipesCall(t, ts, http.StatusBadRequest, http.MethodPost, "/v1/pipes/bad", body)
		if msg, _ := out["message"].(string); msg == "" {
			t.Fatalf("%s: no message: %v", name, out)
		}
	}
	pipesCall(t, ts, http.StatusBadRequest, http.MethodPost, "/v1/pipes/bad%20name", base(""))
	pipesCall(t, ts, http.StatusBadRequest, http.MethodPost, "/v1/pipes/bad", `{not json`)
	pipesCall(t, ts, http.StatusNotFound, http.MethodGet, "/v1/pipes/missing", "")
	pipesCall(t, ts, http.StatusNotFound, http.MethodPost, "/v1/pipes/missing/start", "")
	pipesCall(t, ts, http.StatusBadRequest, http.MethodGet, "/v1/pipes?Limit=0", "")
	pipesCall(t, ts, http.StatusBadRequest, http.MethodGet, "/v1/pipes?NextToken=%25%25", "")
}

func TestPipesStage1MissingResourcesFailThePipe(t *testing.T) {
	srv, ts := newPipesTestServer(t)
	dst := pipesCreateQueue(t, srv, "f-dst")
	pipesCall(t, ts, http.StatusOK, http.MethodPost, "/v1/pipes/f1", fmt.Sprintf(`{"Source":"arn:aws:sqs:us-east-1:123456789012:absent","Target":%q,"RoleArn":%q}`, dst, pipesTestRole))
	got := pipesCall(t, ts, http.StatusOK, http.MethodGet, "/v1/pipes/f1", "")
	if got["CurrentState"] != "CREATE_FAILED" || !strings.Contains(fmt.Sprint(got["StateReason"]), "absent") {
		t.Fatalf("missing source should fail the pipe: %v", got)
	}
	// Fixing the target via update settles to RUNNING once the source exists.
	pipesCreateQueue(t, srv, "absent")
	pipesCall(t, ts, http.StatusOK, http.MethodPut, "/v1/pipes/f1", fmt.Sprintf(`{"RoleArn":%q}`, pipesTestRole))
	if got := pipesCall(t, ts, http.StatusOK, http.MethodGet, "/v1/pipes/f1", ""); got["CurrentState"] != "RUNNING" {
		t.Fatalf("an update from CREATE_FAILED should settle: %v", got)
	}
	if got := pipesCall(t, ts, http.StatusOK, http.MethodGet, "/v1/pipes/f1", ""); got["StateReason"] != nil {
		t.Fatalf("StateReason should clear: %v", got)
	}
}

func TestPipesStage1ListFiltersAndPages(t *testing.T) {
	srv, ts := newPipesTestServer(t)
	src, dst := pipesCreateQueue(t, srv, "l-src"), pipesCreateQueue(t, srv, "l-dst")
	for i := 0; i < 5; i++ {
		desired := "RUNNING"
		if i%2 == 1 {
			desired = "STOPPED"
		}
		pipesCall(t, ts, http.StatusOK, http.MethodPost, fmt.Sprintf("/v1/pipes/list-%d", i), fmt.Sprintf(`{"Source":%q,"Target":%q,"RoleArn":%q,"DesiredState":%q}`, src, dst, pipesTestRole, desired))
	}
	pipesCall(t, ts, http.StatusOK, http.MethodPost, "/v1/pipes/other", fmt.Sprintf(`{"Source":%q,"Target":%q,"RoleArn":%q}`, src, dst, pipesTestRole))

	var names []string
	token := ""
	for page := 0; page < 5; page++ {
		path := "/v1/pipes?NamePrefix=list-&Limit=2"
		if token != "" {
			path += "&NextToken=" + url.QueryEscape(token)
		}
		out := pipesCall(t, ts, http.StatusOK, http.MethodGet, path, "")
		for _, p := range out["Pipes"].([]any) {
			names = append(names, p.(map[string]any)["Name"].(string))
		}
		token, _ = out["NextToken"].(string)
		if token == "" {
			break
		}
	}
	if strings.Join(names, ",") != "list-0,list-1,list-2,list-3,list-4" {
		t.Fatalf("paged names: %v", names)
	}
	stopped := pipesCall(t, ts, http.StatusOK, http.MethodGet, "/v1/pipes?CurrentState=STOPPED", "")["Pipes"].([]any)
	if len(stopped) != 2 {
		t.Fatalf("CurrentState filter: %v", stopped)
	}
}
