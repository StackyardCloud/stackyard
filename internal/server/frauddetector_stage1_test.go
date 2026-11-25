package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func frdCall(t *testing.T, ts *httptest.Server, op, body string) (int, map[string]any) {
	t.Helper()
	return jsonCall(t, ts, frdTargetPrefix+op, "frauddetector", body)
}

func frdOK(t *testing.T, ts *httptest.Server, op, body string) map[string]any {
	t.Helper()
	status, out := frdCall(t, ts, op, body)
	if status != http.StatusOK {
		t.Fatalf("%s: %d %v", op, status, out)
	}
	return out
}

func frdErrCode(t *testing.T, ts *httptest.Server, op, body string, want int) string {
	t.Helper()
	status, out := frdCall(t, ts, op, body)
	if status != want {
		t.Fatalf("%s: want %d, got %d %v", op, want, status, out)
	}
	code, _ := out["__type"].(string)
	return code
}

// frdSetup creates variables, an entity type, labels, outcomes, an event
// type and a detector named "payments".
func frdSetup(t *testing.T, ts *httptest.Server) {
	t.Helper()
	frdOK(t, ts, "BatchCreateVariable", `{"variableEntries":[
		{"name":"amount","dataType":"FLOAT","dataSource":"EVENT","defaultValue":"0.0"},
		{"name":"email","dataType":"STRING","dataSource":"EVENT","defaultValue":"unknown"}]}`)
	frdOK(t, ts, "PutEntityType", `{"name":"customer"}`)
	frdOK(t, ts, "PutLabel", `{"name":"fraud"}`)
	frdOK(t, ts, "PutLabel", `{"name":"legit"}`)
	frdOK(t, ts, "PutOutcome", `{"name":"block"}`)
	frdOK(t, ts, "PutOutcome", `{"name":"approve"}`)
	frdOK(t, ts, "PutEventType", `{"name":"payment","eventVariables":["amount","email"],"labels":["fraud","legit"],"entityTypes":["customer"]}`)
	frdOK(t, ts, "PutDetector", `{"detectorId":"payments","eventTypeName":"payment"}`)
}

func TestFraudDetectorStage1RulesAndPredictions(t *testing.T) {
	ts := newForecastTestServer(t)
	frdSetup(t, ts)
	if code := frdErrCode(t, ts, "CreateVariable", `{"name":"amount","dataType":"FLOAT","dataSource":"EVENT","defaultValue":"0"}`, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("duplicate variable: %s", code)
	}
	if status, out := frdCall(t, ts, "CreateVariable", `{"name":"n2","dataType":"BOOLEAN","dataSource":"EVENT","defaultValue":"maybe"}`); status != http.StatusBadRequest || !strings.Contains(out["message"].(string), "not a valid BOOLEAN") {
		t.Fatalf("bad boolean default: %d %v", status, out)
	}
	if code := frdErrCode(t, ts, "CreateVariable", `{"name":"n","dataType":"INTEGER","dataSource":"EVENT","defaultValue":"x"}`, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("bad default: %s", code)
	}
	if code := frdErrCode(t, ts, "CreateRule", `{"ruleId":"bad","detectorId":"payments","expression":"$amount >","language":"DETECTORPL","outcomes":["block"]}`, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("bad expression: %s", code)
	}
	if code := frdErrCode(t, ts, "CreateRule", `{"ruleId":"bad","detectorId":"payments","expression":"$nope > 1","language":"DETECTORPL","outcomes":["block"]}`, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("unknown variable: %s", code)
	}
	frdOK(t, ts, "CreateRule", `{"ruleId":"big","detectorId":"payments","expression":"$amount > 1000 and $email != \"vip@example.com\"","language":"DETECTORPL","outcomes":["block"]}`)
	frdOK(t, ts, "CreateRule", `{"ruleId":"any","detectorId":"payments","expression":"$amount >= 0","language":"DETECTORPL","outcomes":["approve"]}`)
	v := frdOK(t, ts, "CreateDetectorVersion", `{"detectorId":"payments","rules":[{"detectorId":"payments","ruleId":"big","ruleVersion":"1"},{"detectorId":"payments","ruleId":"any","ruleVersion":"1"}]}`)
	if v["detectorVersionId"] != "1" || v["status"] != "DRAFT" {
		t.Fatalf("detector version: %v", v)
	}
	if code := frdErrCode(t, ts, "GetEventPrediction", `{"detectorId":"payments","eventId":"e0","eventTypeName":"payment","eventTimestamp":"2026-11-24T10:00:00Z","entities":[{"entityType":"customer","entityId":"c1"}],"eventVariables":{"amount":"5"}}`, http.StatusNotFound); code != "ResourceNotFoundException" {
		t.Fatalf("no active version: %s", code)
	}
	if code := frdErrCode(t, ts, "UpdateDetectorVersionStatus", `{"detectorId":"payments","detectorVersionId":"1","status":"INACTIVE"}`, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("DRAFT -> INACTIVE: %s", code)
	}
	frdOK(t, ts, "UpdateDetectorVersionStatus", `{"detectorId":"payments","detectorVersionId":"1","status":"ACTIVE"}`)

	predict := func(id, amount, email string) []any {
		out := frdOK(t, ts, "GetEventPrediction", fmt.Sprintf(`{"detectorId":"payments","eventId":%q,"eventTypeName":"payment","eventTimestamp":"2026-11-24T10:00:00Z",
			"entities":[{"entityType":"customer","entityId":"c1"}],"eventVariables":{"amount":%q,"email":%q}}`, id, amount, email))
		return out["ruleResults"].([]any)
	}
	if r := predict("e1", "5000", "a@example.com"); len(r) != 1 || r[0].(map[string]any)["ruleId"] != "big" {
		t.Fatalf("big payment: %v", r)
	}
	if r := predict("e2", "5000", "vip@example.com"); len(r) != 1 || r[0].(map[string]any)["ruleId"] != "any" {
		t.Fatalf("vip payment (numeric and string comparisons): %v", r)
	}
	if code := frdErrCode(t, ts, "GetEventPrediction", `{"detectorId":"payments","eventId":"e3","eventTypeName":"payment","eventTimestamp":"2026-11-24T10:00:00Z","entities":[{"entityType":"customer","entityId":"c1"}],"eventVariables":{"colour":"red"}}`, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("unknown event variable: %s", code)
	}

	// Ingestion stored the scored events; predictions are listed and found.
	ev := frdOK(t, ts, "GetEvent", `{"eventId":"e1","eventTypeName":"payment"}`)["event"].(map[string]any)
	if ev["eventVariables"].(map[string]any)["amount"] != "5000" {
		t.Fatalf("stored event: %v", ev)
	}
	preds := frdOK(t, ts, "ListEventPredictions", `{"eventId":{"value":"e1"}}`)["eventPredictionSummaries"].([]any)
	if len(preds) != 1 {
		t.Fatalf("predictions: %v", preds)
	}
	p := preds[0].(map[string]any)
	meta := frdOK(t, ts, "GetEventPredictionMetadata", fmt.Sprintf(`{"eventId":"e1","eventTypeName":"payment","detectorId":"payments","detectorVersionId":"1","predictionTimestamp":%q}`, p["predictionTimestamp"]))
	if out := meta["outcomes"].([]any); len(out) != 1 || out[0] != "block" {
		t.Fatalf("prediction metadata: %v", meta)
	}

	// Rules in use, active versions and used outcomes can't be deleted.
	if code := frdErrCode(t, ts, "DeleteRule", `{"rule":{"detectorId":"payments","ruleId":"big","ruleVersion":"1"}}`, http.StatusConflict); code != "ConflictException" {
		t.Fatalf("delete a used rule: %s", code)
	}
	if code := frdErrCode(t, ts, "DeleteDetectorVersion", `{"detectorId":"payments","detectorVersionId":"1"}`, http.StatusConflict); code != "ConflictException" {
		t.Fatalf("delete the active version: %s", code)
	}
	if code := frdErrCode(t, ts, "DeleteOutcome", `{"name":"block"}`, http.StatusConflict); code != "ConflictException" {
		t.Fatalf("delete a used outcome: %s", code)
	}
	if code := frdErrCode(t, ts, "UpdateDetectorVersion", `{"detectorId":"payments","detectorVersionId":"1","externalModelEndpoints":[],"rules":[]}`, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("update an active version: %s", code)
	}
	up := frdOK(t, ts, "UpdateRuleVersion", `{"rule":{"detectorId":"payments","ruleId":"big","ruleVersion":"1"},"expression":"$amount > 2000","language":"DETECTORPL","outcomes":["block"]}`)
	if up["rule"].(map[string]any)["ruleVersion"] != "2" {
		t.Fatalf("rule version: %v", up)
	}
	if rules := frdOK(t, ts, "GetRules", `{"detectorId":"payments","ruleId":"big"}`)["ruleDetails"].([]any); len(rules) != 2 {
		t.Fatalf("rule versions: %v", rules)
	}
}

func TestFraudDetectorStage1ModelsListsAndTags(t *testing.T) {
	ts := newForecastTestServer(t)
	frdSetup(t, ts)
	frdOK(t, ts, "CreateModel", `{"modelId":"pay_model","modelType":"ONLINE_FRAUD_INSIGHTS","eventTypeName":"payment"}`)
	if vars := frdOK(t, ts, "GetVariables", `{"name":"pay_model_insightscore"}`)["variables"].([]any); len(vars) != 1 {
		t.Fatalf("score variable: %v", vars)
	}
	if code := frdErrCode(t, ts, "CreateModelVersion", `{"modelId":"pay_model","modelType":"ONLINE_FRAUD_INSIGHTS","trainingDataSource":"EXTERNAL_EVENTS","trainingDataSchema":{"modelVariables":["amount"]}}`, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("missing externalEventsDetail: %s", code)
	}
	body := `{"modelId":"pay_model","modelType":"ONLINE_FRAUD_INSIGHTS","trainingDataSource":"EXTERNAL_EVENTS","trainingDataSchema":{"modelVariables":["amount","email"]},
		"externalEventsDetail":{"dataLocation":"s3://bucket/train.csv","dataAccessRoleArn":"arn:aws:iam::123456789012:role/fd"}}`
	if mv := frdOK(t, ts, "CreateModelVersion", body); mv["modelVersionNumber"] != "1.0" || mv["status"] != "TRAINING_IN_PROGRESS" {
		t.Fatalf("model version: %v", mv)
	}
	ref := `{"modelId":"pay_model","modelType":"ONLINE_FRAUD_INSIGHTS","modelVersionNumber":"1.0"`
	if code := frdErrCode(t, ts, "UpdateModelVersionStatus", ref+`,"status":"ACTIVE"}`, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("activate while training: %s", code)
	}
	// That read saw it training; the next one sees it trained.
	if mv := frdOK(t, ts, "GetModelVersion", ref+`}`); mv["status"] != "TRAINING_COMPLETE" {
		t.Fatalf("after training: %v", mv)
	}
	details := frdOK(t, ts, "DescribeModelVersions", `{"modelId":"pay_model"}`)["modelVersionDetails"].([]any)
	if len(details) != 1 || details[0].(map[string]any)["trainingResult"] == nil {
		t.Fatalf("model version details: %v", details)
	}
	frdOK(t, ts, "UpdateModelVersionStatus", ref+`,"status":"ACTIVE"}`)
	if u := frdOK(t, ts, "UpdateModelVersion", `{"modelId":"pay_model","modelType":"ONLINE_FRAUD_INSIGHTS","majorVersionNumber":"1"}`); u["modelVersionNumber"] != "1.1" {
		t.Fatalf("minor version: %v", u)
	}

	// A detector version that uses the model scores events with it.
	frdOK(t, ts, "CreateRule", `{"ruleId":"risky","detectorId":"payments","expression":"$pay_model_insightscore >= 0","language":"DETECTORPL","outcomes":["block"]}`)
	frdOK(t, ts, "CreateDetectorVersion", `{"detectorId":"payments","rules":[{"detectorId":"payments","ruleId":"risky","ruleVersion":"1"}],
		"modelVersions":[{"modelId":"pay_model","modelType":"ONLINE_FRAUD_INSIGHTS","modelVersionNumber":"1.0"}]}`)
	out := frdOK(t, ts, "GetEventPrediction", `{"detectorId":"payments","detectorVersionId":"1","eventId":"m1","eventTypeName":"payment","eventTimestamp":"2026-11-24T10:00:00Z",
		"entities":[{"entityType":"customer","entityId":"c1"}],"eventVariables":{"amount":"10"}}`)
	if s := out["modelScores"].([]any); len(s) != 1 || len(out["ruleResults"].([]any)) != 1 {
		t.Fatalf("model prediction: %v", out)
	}
	if code := frdErrCode(t, ts, "DeleteModelVersion", ref+`}`, http.StatusConflict); code != "ConflictException" {
		t.Fatalf("delete an active model version: %s", code)
	}

	// Lists: create, append, remove, page.
	frdOK(t, ts, "CreateList", `{"name":"blocked","elements":["a@x.com","b@x.com"],"variableType":"EMAIL_ADDRESS"}`)
	frdOK(t, ts, "UpdateList", `{"name":"blocked","elements":["c@x.com","a@x.com"],"updateMode":"APPEND"}`)
	frdOK(t, ts, "UpdateList", `{"name":"blocked","elements":["b@x.com"],"updateMode":"REMOVE"}`)
	if els := frdOK(t, ts, "GetListElements", `{"name":"blocked"}`)["elements"].([]any); len(els) != 2 {
		t.Fatalf("elements: %v", els)
	}

	// Tags on ARNs, and unknown ARNs.
	arn := frdOK(t, ts, "GetDetectors", `{"detectorId":"payments"}`)["detectors"].([]any)[0].(map[string]any)["arn"].(string)
	frdOK(t, ts, "TagResource", fmt.Sprintf(`{"resourceARN":%q,"tags":[{"key":"team","value":"risk"}]}`, arn))
	if tags := frdOK(t, ts, "ListTagsForResource", fmt.Sprintf(`{"resourceARN":%q}`, arn))["tags"].([]any); len(tags) != 1 {
		t.Fatalf("tags: %v", tags)
	}
	if code := frdErrCode(t, ts, "ListTagsForResource", `{"resourceARN":"arn:aws:frauddetector:us-east-1:123456789012:detector/none"}`, http.StatusNotFound); code != "ResourceNotFoundException" {
		t.Fatalf("unknown arn: %s", code)
	}

	// Batch jobs and event deletion settle on read.
	frdOK(t, ts, "CreateBatchImportJob", `{"jobId":"imp1","inputPath":"s3://bucket/in.csv","outputPath":"s3://bucket/out/","eventTypeName":"payment","iamRoleArn":"arn:aws:iam::123456789012:role/fd"}`)
	frdOK(t, ts, "CancelBatchImportJob", `{"jobId":"imp1"}`)
	if code := frdErrCode(t, ts, "CancelBatchImportJob", `{"jobId":"imp1"}`, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("cancel twice: %s", code)
	}
	frdOK(t, ts, "DeleteEventsByEventType", `{"eventTypeName":"payment"}`)
	if s := frdOK(t, ts, "GetDeleteEventsByEventTypeStatus", `{"eventTypeName":"payment"}`)["eventsDeletionStatus"]; s != "COMPLETE" {
		t.Fatalf("deletion status: %v", s)
	}
	if code := frdErrCode(t, ts, "DeleteEventType", `{"name":"payment"}`, http.StatusConflict); code != "ConflictException" {
		t.Fatalf("delete an event type in use: %s", code)
	}
}
