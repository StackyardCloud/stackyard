package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const (
	itgTestNS       = "urn:tdm:us-east-1/123456789012/default:"
	itgTestEntities = `type Camera @deviceModel(id: "` + itgTestNS + `deviceModel:Camera", capability: "` + itgTestNS + `capability:Capture") {ignore:void}
type Capture @capability(id: "` + itgTestNS + `capability:Capture") {ignore:void}
type MyCamera @device(id: "` + itgTestNS + `device:MyCamera", deviceModel: "` + itgTestNS + `deviceModel:Camera") {ignore:void}`
	itgTestFlow     = `{ query Snap @workflowType(id: "` + itgTestNS + `Workflow:Snap") { variables { cam @property(id: "` + itgTestNS + `deviceModel:Camera") } } }`
	itgTestSystem   = `{ type Home @systemType(id: "` + itgTestNS + `System:Home") { camera: Camera @thing(id: "` + itgTestNS + `deviceModel:Camera") snap: Workflow @workflow(id: "` + itgTestNS + `Workflow:Snap") } }`
	itgTestInstance = `query Home1 @deployment(id: "` + itgTestNS + `Deployment:Home1", systemId: "` + itgTestNS + `System:Home") { camera(deviceId: "pump-1") }`
)

func itgJSON(t *testing.T, v any) string {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func itgDef(text string) map[string]any { return map[string]any{"language": "GRAPHQL", "text": text} }

func itgOK(t *testing.T, ts *httptest.Server, op string, in map[string]any) map[string]any {
	t.Helper()
	return budOKAPI(t, ts, itgAPI, op, itgJSON(t, in))
}

func itgErrCode(t *testing.T, ts *httptest.Server, op string, in map[string]any, status int) string {
	t.Helper()
	return budErrAPI(t, ts, itgAPI, op, itgJSON(t, in), status)
}

func TestIoTThingsGraphStage1Namespace(t *testing.T) {
	_, ts := newBudTestServer(t)
	if code := itgErrCode(t, ts, "DescribeNamespace", map[string]any{}, http.StatusNotFound); code != "ResourceNotFoundException" {
		t.Fatalf("no namespace yet: %s", code)
	}
	up := itgOK(t, ts, "UploadEntityDefinitions", map[string]any{"document": itgDef(itgTestEntities)})["uploadId"].(string)
	for _, want := range []string{"IN_PROGRESS", "SUCCEEDED"} {
		if got := itgOK(t, ts, "GetUploadStatus", map[string]any{"uploadId": up})["uploadStatus"]; got != want {
			t.Fatalf("upload status: %v, want %s", got, want)
		}
	}
	bad := itgOK(t, ts, "UploadEntityDefinitions", map[string]any{"document": itgDef(`type X @device(id: "urn:tdm:aws/examples:device:X") {ignore:void}`)})["uploadId"].(string)
	st := itgOK(t, ts, "GetUploadStatus", map[string]any{"uploadId": bad})
	if st["uploadStatus"] != "FAILED" || len(st["failureReason"].([]any)) == 0 {
		t.Fatalf("upload outside the namespace: %v", st)
	}
	ns := itgOK(t, ts, "DescribeNamespace", map[string]any{})
	if ns["namespaceVersion"].(float64) != 1 {
		t.Fatalf("namespace: %v", ns)
	}
	got := itgOK(t, ts, "GetEntities", map[string]any{"ids": []string{itgTestNS + "device:MyCamera"}})["descriptions"].([]any)
	if len(got) != 1 || got[0].(map[string]any)["type"] != "DEVICE" {
		t.Fatalf("GetEntities: %v", got)
	}
	found := itgOK(t, ts, "SearchEntities", map[string]any{"entityTypes": []string{"DEVICE_MODEL", "CAPABILITY"},
		"filters": []any{map[string]any{"name": "NAME", "value": []string{"Camera"}}}})["descriptions"].([]any)
	if len(found) != 1 {
		t.Fatalf("SearchEntities: %v", found)
	}
	itgOK(t, ts, "AssociateEntityToThing", map[string]any{"thingName": "pump-1", "entityId": itgTestNS + "device:MyCamera"})
	if code := itgErrCode(t, ts, "AssociateEntityToThing", map[string]any{"thingName": "pump-1", "entityId": itgTestNS + "capability:Capture"}, http.StatusBadRequest); code != "InvalidRequestException" {
		t.Fatalf("associate a capability: %s", code)
	}
	things := itgOK(t, ts, "SearchThings", map[string]any{"entityId": itgTestNS + "deviceModel:Camera"})["things"].([]any)
	if len(things) != 1 || things[0].(map[string]any)["thingName"] != "pump-1" {
		t.Fatalf("SearchThings by device model: %v", things)
	}
	itgOK(t, ts, "DissociateEntityFromThing", map[string]any{"thingName": "pump-1", "entityType": "DEVICE"})
	if code := itgErrCode(t, ts, "DissociateEntityFromThing", map[string]any{"thingName": "pump-1", "entityType": "DEVICE"}, http.StatusNotFound); code != "ResourceNotFoundException" {
		t.Fatalf("dissociate twice: %s", code)
	}
	itgOK(t, ts, "DeleteNamespace", map[string]any{})
	for _, want := range []string{"IN_PROGRESS", "SUCCEEDED"} {
		if got := itgOK(t, ts, "GetNamespaceDeletionStatus", map[string]any{})["status"]; got != want {
			t.Fatalf("deletion status: %v, want %s", got, want)
		}
	}
	if code := itgErrCode(t, ts, "GetEntities", map[string]any{"ids": []string{itgTestNS + "device:MyCamera"}}, http.StatusNotFound); code != "ResourceNotFoundException" {
		t.Fatalf("entity after namespace deletion: %s", code)
	}
}

func TestIoTThingsGraphStage1TemplatesAndDeployments(t *testing.T) {
	_, ts := newBudTestServer(t)
	// A flow referencing entities that don't exist yet is refused.
	if code := itgErrCode(t, ts, "CreateFlowTemplate", map[string]any{"definition": itgDef(itgTestFlow)}, http.StatusBadRequest); code != "InvalidRequestException" {
		t.Fatalf("flow before entities: %s", code)
	}
	itgOK(t, ts, "UploadEntityDefinitions", map[string]any{"document": itgDef(itgTestEntities)})
	flow := itgOK(t, ts, "CreateFlowTemplate", map[string]any{"definition": itgDef(itgTestFlow)})["summary"].(map[string]any)
	flowID := flow["id"].(string)
	if code := itgErrCode(t, ts, "CreateFlowTemplate", map[string]any{"definition": itgDef(itgTestFlow)}, http.StatusBadRequest); code != "ResourceAlreadyExistsException" {
		t.Fatalf("duplicate flow: %s", code)
	}
	if rev := itgOK(t, ts, "UpdateFlowTemplate", map[string]any{"id": flowID, "definition": itgDef(itgTestFlow)})["summary"].(map[string]any)["revisionNumber"]; rev.(float64) != 2 {
		t.Fatalf("revision: %v", rev)
	}
	revs := itgOK(t, ts, "GetFlowTemplateRevisions", map[string]any{"id": flowID})["summaries"].([]any)
	if len(revs) != 2 || revs[0].(map[string]any)["revisionNumber"].(float64) != 2 {
		t.Fatalf("revisions: %v", revs)
	}
	sys := itgOK(t, ts, "CreateSystemTemplate", map[string]any{"definition": itgDef(itgTestSystem)})["summary"].(map[string]any)["id"].(string)
	if n := len(itgOK(t, ts, "SearchSystemTemplates", map[string]any{"filters": []any{map[string]any{"name": "FLOW_TEMPLATE_ID", "value": []string{flowID}}}})["summaries"].([]any)); n != 1 {
		t.Fatalf("SearchSystemTemplates: %d", n)
	}
	if code := itgErrCode(t, ts, "DeleteFlowTemplate", map[string]any{"id": flowID}, http.StatusBadRequest); code != "ResourceInUseException" {
		t.Fatalf("delete a flow in use: %s", code)
	}
	if code := itgErrCode(t, ts, "CreateSystemInstance", map[string]any{"definition": itgDef(itgTestInstance), "target": "CLOUD"}, http.StatusBadRequest); code != "InvalidRequestException" {
		t.Fatalf("cloud instance without a role: %s", code)
	}
	inst := itgOK(t, ts, "CreateSystemInstance", map[string]any{"definition": itgDef(itgTestInstance), "target": "CLOUD",
		"flowActionsRoleArn": "arn:aws:iam::123456789012:role/things-graph", "tags": []any{map[string]any{"key": "team", "value": "home"}}})["summary"].(map[string]any)
	instID := inst["id"].(string)
	if inst["status"] != "NOT_DEPLOYED" {
		t.Fatalf("new instance: %v", inst)
	}
	if s := itgOK(t, ts, "DeploySystemInstance", map[string]any{"id": instID})["summary"].(map[string]any)["status"]; s != "DEPLOY_IN_PROGRESS" {
		t.Fatalf("deploy: %v", s)
	}
	desc := itgOK(t, ts, "GetSystemInstance", map[string]any{"id": instID})["description"].(map[string]any)
	if desc["summary"].(map[string]any)["status"] != "DEPLOYED_IN_TARGET" || len(desc["validatedDependencyRevisions"].([]any)) != 2 {
		t.Fatalf("deployed instance: %v", desc)
	}
	if code := itgErrCode(t, ts, "DeleteSystemInstance", map[string]any{"id": instID}, http.StatusBadRequest); code != "ResourceInUseException" {
		t.Fatalf("delete a deployed instance: %s", code)
	}
	execs := itgOK(t, ts, "SearchFlowExecutions", map[string]any{"systemInstanceId": instID})["summaries"].([]any)
	if len(execs) != 1 || execs[0].(map[string]any)["flowTemplateId"] != flowID {
		t.Fatalf("executions: %v", execs)
	}
	msgs := itgOK(t, ts, "ListFlowExecutionMessages", map[string]any{"flowExecutionId": execs[0].(map[string]any)["flowExecutionId"]})["messages"].([]any)
	if len(msgs) != 4 || msgs[3].(map[string]any)["eventType"] != "EXECUTION_SUCCEEDED" {
		t.Fatalf("messages: %v", msgs)
	}
	arn := inst["arn"].(string)
	itgOK(t, ts, "TagResource", map[string]any{"resourceArn": arn, "tags": []any{map[string]any{"key": "env", "value": "prod"}}})
	if tags := itgOK(t, ts, "ListTagsForResource", map[string]any{"resourceArn": arn})["tags"].([]any); len(tags) != 2 {
		t.Fatalf("tags: %v", tags)
	}
	itgOK(t, ts, "UndeploySystemInstance", map[string]any{"id": instID})
	if n := len(itgOK(t, ts, "SearchSystemInstances", map[string]any{"filters": []any{map[string]any{"name": "STATUS", "value": []string{"NOT_DEPLOYED"}}}})["summaries"].([]any)); n != 1 {
		t.Fatalf("undeployed instances: %d", n)
	}
	itgOK(t, ts, "DeleteSystemInstance", map[string]any{"id": instID})
	itgOK(t, ts, "DeprecateSystemTemplate", map[string]any{"id": sys})
	itgOK(t, ts, "DeleteSystemTemplate", map[string]any{"id": sys})
	itgOK(t, ts, "DeprecateFlowTemplate", map[string]any{"id": flowID})
	itgOK(t, ts, "DeleteFlowTemplate", map[string]any{"id": flowID})
	if code := itgErrCode(t, ts, "GetFlowTemplate", map[string]any{"id": flowID}, http.StatusNotFound); code != "ResourceNotFoundException" {
		t.Fatalf("deleted flow: %s", code)
	}
}

// TestIoTThingsGraphStage0ErrorShape checks the awsJson error body.
func TestIoTThingsGraphStage0ErrorShape(t *testing.T) {
	_, ts := newBudTestServer(t)
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(`{"uploadId":"missing"}`), map[string]string{
		"Content-Type": "application/x-amz-json-1.1", "X-Amz-Target": "IotThingsGraphFrontEndService.GetUploadStatus",
	}, "iotthingsgraph")
	raw := mustBody(t, resp)
	var out map[string]string
	if err := json.Unmarshal(raw, &out); err != nil || resp.StatusCode != http.StatusNotFound ||
		out["__type"] != "ResourceNotFoundException" || !strings.Contains(out["Message"], "missing") {
		t.Fatalf("error shape: %d %s", resp.StatusCode, raw)
	}
}
