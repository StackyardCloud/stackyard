package server

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

const osisTestConfig = `version: "2"
log-pipeline:
  source:
    http:
      path: "/logs"
  sink:
    - opensearch:
        hosts: ["https://search-test.us-east-1.es.amazonaws.com"]
        index: "logs"
`

func osisCreateBody(name string, extra string) string {
	body := fmt.Sprintf(`{"PipelineName":%q,"MinUnits":1,"MaxUnits":2,"PipelineConfigurationBody":%q`, name, osisTestConfig)
	if extra != "" {
		body += "," + extra
	}
	return body + "}"
}

func osisPipelineOf(t *testing.T, out map[string]any) map[string]any {
	t.Helper()
	p, ok := out["Pipeline"].(map[string]any)
	if !ok {
		t.Fatalf("no Pipeline in %v", out)
	}
	return p
}

func TestOSISStage1PipelineLifecycle(t *testing.T) {
	_, ts := newOSISTestServer(t)
	p := osisPipelineOf(t, osisCall(t, ts, http.StatusOK, http.MethodPost, "createPipeline", osisCreateBody("logs", `"Tags":[{"Key":"env","Value":"dev"}]`)))
	if p["Status"] != "CREATING" || p["PipelineArn"] != "arn:aws:osis:us-east-1:123456789012:pipeline/logs" {
		t.Fatalf("create: %v", p)
	}
	if urls, _ := p["IngestEndpointUrls"].([]any); len(urls) != 1 || !strings.Contains(urls[0].(string), "/_stackyard/osis/logs-") {
		t.Fatalf("ingest URL: %v", p["IngestEndpointUrls"])
	}
	if d, _ := p["Destinations"].([]any); len(d) != 1 {
		t.Fatalf("destinations: %v", p["Destinations"])
	}
	if code := osisErr(t, ts, http.MethodPost, "createPipeline", osisCreateBody("logs", "")); code != "ResourceAlreadyExistsException" {
		t.Fatalf("duplicate: %s", code)
	}
	// Changes while CREATING are conflicts; the next read settles it.
	if code := osisErr(t, ts, http.MethodPut, "updatePipeline/logs", `{"MaxUnits":4}`); code != "ConflictException" {
		t.Fatalf("update while creating: %s", code)
	}
	progress := osisCall(t, ts, http.StatusOK, http.MethodGet, "getPipelineChangeProgress/logs", "")["ChangeProgressStatuses"].([]any)
	if len(progress) != 1 || progress[0].(map[string]any)["Status"] != "IN_PROGRESS" {
		t.Fatalf("change progress while creating: %v", progress)
	}
	if got := osisPipelineOf(t, osisCall(t, ts, http.StatusOK, http.MethodGet, "getPipeline/logs", ""))["Status"]; got != "ACTIVE" {
		t.Fatalf("settled status: %v", got)
	}

	// Update: units must stay ordered; the merged result is checked.
	if code := osisErr(t, ts, http.MethodPut, "updatePipeline/logs", `{"MinUnits":5}`); code != "ValidationException" {
		t.Fatalf("min above existing max: %s", code)
	}
	up := osisPipelineOf(t, osisCall(t, ts, http.StatusOK, http.MethodPut, "updatePipeline/logs", `{"MaxUnits":4}`))
	if up["Status"] != "UPDATING" || up["MaxUnits"] != float64(4) || up["MinUnits"] != float64(1) {
		t.Fatalf("update: %v", up)
	}
	osisCall(t, ts, http.StatusOK, http.MethodGet, "getPipeline/logs", "")
	if n := len(osisCall(t, ts, http.StatusOK, http.MethodGet, "getPipelineChangeProgress/logs", "")["ChangeProgressStatuses"].([]any)); n != 2 {
		t.Fatalf("change progress records: %d", n)
	}

	// Start/stop only from the opposite resting state.
	if code := osisErr(t, ts, http.MethodPut, "startPipeline/logs", ""); code != "ConflictException" {
		t.Fatalf("start while active: %s", code)
	}
	if s := osisPipelineOf(t, osisCall(t, ts, http.StatusOK, http.MethodPut, "stopPipeline/logs", ""))["Status"]; s != "STOPPING" {
		t.Fatalf("stop: %v", s)
	}
	osisCall(t, ts, http.StatusOK, http.MethodGet, "getPipeline/logs", "")
	if s := osisPipelineOf(t, osisCall(t, ts, http.StatusOK, http.MethodPut, "startPipeline/logs", ""))["Status"]; s != "STARTING" {
		t.Fatalf("start: %v", s)
	}

	// Listing settles and pages.
	osisCall(t, ts, http.StatusOK, http.MethodPost, "createPipeline", osisCreateBody("other", ""))
	page := osisCall(t, ts, http.StatusOK, http.MethodGet, "listPipelines?maxResults=1", "")
	if len(page["Pipelines"].([]any)) != 1 || page["NextToken"] == nil {
		t.Fatalf("first page: %v", page)
	}
	last := osisCall(t, ts, http.StatusOK, http.MethodGet, "listPipelines?maxResults=1&nextToken="+page["NextToken"].(string), "")
	if s := last["Pipelines"].([]any)[0].(map[string]any)["Status"]; s != "ACTIVE" || last["NextToken"] != nil {
		t.Fatalf("second page: %v", last)
	}
	if code := osisErr(t, ts, http.MethodGet, "listPipelines?nextToken=bogus", ""); code != "InvalidPaginationTokenException" {
		t.Fatalf("bad token: %s", code)
	}

	osisCall(t, ts, http.StatusOK, http.MethodDelete, "deletePipeline/logs", "")
	if code := osisErr(t, ts, http.MethodGet, "getPipeline/logs", ""); code != "ResourceNotFoundException" {
		t.Fatalf("after delete: %s", code)
	}
}

func TestOSISStage1Validation(t *testing.T) {
	srv, ts := newOSISTestServer(t)
	for name, body := range map[string]string{
		"min above max":    strings.Replace(osisCreateBody("bad", ""), `"MinUnits":1`, `"MinUnits":3`, 1),
		"too many units":   strings.Replace(osisCreateBody("bad", ""), `"MaxUnits":2`, `"MaxUnits":385`, 1),
		"bad name":         osisCreateBody("Bad_Name", ""),
		"unknown member":   osisCreateBody("bad", `"Status":"ACTIVE"`),
		"bad config":       `{"PipelineName":"bad","MinUnits":1,"MaxUnits":2,"PipelineConfigurationBody":"version: 2\n"}`,
		"logging no group": osisCreateBody("bad", `"LogPublishingOptions":{"IsLoggingEnabled":true}`),
		"buffer one unit":  osisCreateBody("bad", `"BufferOptions":{"PersistentBufferEnabled":true}`),
		"missing subnet":   osisCreateBody("bad", `"VpcOptions":{"SubnetIds":["subnet-00000000"]}`),
		"aws tag":          osisCreateBody("bad", `"Tags":[{"Key":"aws:x","Value":"y"}]`),
	} {
		if code := osisErr(t, ts, http.MethodPost, "createPipeline", body); code != "ValidationException" {
			t.Fatalf("%s: %s", name, code)
		}
	}
	// A VPC pipeline gets a service-managed endpoint in the subnet's VPC.
	var subnet, vpc string
	for _, v := range srv.ec2.DescribeVpcs(nil) {
		if v.IsDefault {
			vpc = v.ID
			subnet = srv.ec2.DescribeSubnets(nil, []string{v.ID})[0].ID
		}
	}
	p := osisPipelineOf(t, osisCall(t, ts, http.StatusOK, http.MethodPost, "createPipeline", osisCreateBody("vpcpipe", fmt.Sprintf(`"VpcOptions":{"SubnetIds":[%q]}`, subnet))))
	ep := p["VpcEndpoints"].([]any)[0].(map[string]any)
	if ep["VpcId"] != vpc || !strings.HasPrefix(fmt.Sprint(ep["VpcEndpointId"]), "vpce-") {
		t.Fatalf("vpc endpoint: %v", ep)
	}
	// ValidatePipeline reports problems instead of failing.
	v := osisCall(t, ts, http.StatusOK, http.MethodPost, "validatePipeline", `{"PipelineConfigurationBody":"version: \"2\"\np:\n  source:\n    http:\n"}`)
	if v["isValid"] != false || len(v["Errors"].([]any)) == 0 {
		t.Fatalf("validate invalid: %v", v)
	}
	v = osisCall(t, ts, http.StatusOK, http.MethodPost, "validatePipeline", fmt.Sprintf(`{"PipelineConfigurationBody":%q}`, osisTestConfig))
	if v["isValid"] != true {
		t.Fatalf("validate valid: %v", v)
	}
}

func TestOSISStage1TagsPoliciesEndpointsBlueprints(t *testing.T) {
	srv, ts := newOSISTestServer(t)
	osisCall(t, ts, http.StatusOK, http.MethodPost, "createPipeline", osisCreateBody("tagged", ""))
	arn := "arn:aws:osis:us-east-1:123456789012:pipeline/tagged"
	q := "?arn=" + url.QueryEscape(arn)
	osisCall(t, ts, http.StatusOK, http.MethodPost, "tagResource/"+q, `{"Tags":[{"Key":"team","Value":"a"},{"Key":"env","Value":"dev"}]}`)
	osisCall(t, ts, http.StatusOK, http.MethodPost, "untagResource/"+q, `{"TagKeys":["env"]}`)
	if tags := osisCall(t, ts, http.StatusOK, http.MethodGet, "listTagsForResource/"+q, "")["Tags"].([]any); len(tags) != 1 {
		t.Fatalf("tags: %v", tags)
	}
	missing := "?arn=" + url.QueryEscape("arn:aws:osis:us-east-1:123456789012:pipeline/nope")
	if code := osisErr(t, ts, http.MethodGet, "listTagsForResource/"+missing, ""); code != "ResourceNotFoundException" {
		t.Fatalf("tags on missing pipeline: %s", code)
	}

	esc := url.PathEscape(arn)
	if code := osisErr(t, ts, http.MethodGet, "resourcePolicy/"+esc, ""); code != "ResourceNotFoundException" {
		t.Fatalf("policy before put: %s", code)
	}
	if code := osisErr(t, ts, http.MethodPut, "resourcePolicy/"+esc, `{"Policy":"{}"}`); code != "ValidationException" {
		t.Fatalf("policy without statement: %s", code)
	}
	policy := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"AWS":"123456789012"},"Action":"osis:Ingest","Resource":"*"}]}`
	osisCall(t, ts, http.StatusOK, http.MethodPut, "resourcePolicy/"+esc, fmt.Sprintf(`{"Policy":%q}`, policy))
	if got := osisCall(t, ts, http.StatusOK, http.MethodGet, "resourcePolicy/"+esc, "")["Policy"]; got != policy {
		t.Fatalf("policy: %v", got)
	}
	osisCall(t, ts, http.StatusOK, http.MethodDelete, "resourcePolicy/"+esc, "")

	// Endpoints need a real pipeline and VPC; revoke marks them REVOKED.
	var subnet string
	for _, v := range srv.ec2.DescribeVpcs(nil) {
		if v.IsDefault {
			subnet = srv.ec2.DescribeSubnets(nil, []string{v.ID})[0].ID
		}
	}
	if code := osisErr(t, ts, http.MethodPost, "createPipelineEndpoint", fmt.Sprintf(`{"PipelineArn":"arn:aws:osis:us-east-1:123456789012:pipeline/nope","VpcOptions":{"SubnetIds":[%q]}}`, subnet)); code != "ResourceNotFoundException" {
		t.Fatalf("endpoint for missing pipeline: %s", code)
	}
	ep := osisCall(t, ts, http.StatusOK, http.MethodPost, "createPipelineEndpoint", fmt.Sprintf(`{"PipelineArn":%q,"VpcOptions":{"SubnetIds":[%q]}}`, arn, subnet))
	id := ep["EndpointId"].(string)
	if ep["Status"] != "CREATING" {
		t.Fatalf("endpoint: %v", ep)
	}
	if s := osisCall(t, ts, http.StatusOK, http.MethodGet, "listPipelineEndpoints", "")["PipelineEndpoints"].([]any)[0].(map[string]any)["Status"]; s != "ACTIVE" {
		t.Fatalf("endpoint settled: %v", s)
	}
	osisCall(t, ts, http.StatusOK, http.MethodPost, "revokePipelineEndpointConnections", fmt.Sprintf(`{"PipelineArn":%q,"EndpointIds":[%q]}`, arn, id))
	if s := osisCall(t, ts, http.StatusOK, http.MethodGet, "listPipelineEndpointConnections", "")["PipelineEndpointConnections"].([]any)[0].(map[string]any)["Status"]; s != "REVOKED" {
		t.Fatalf("revoked: %v", s)
	}
	osisCall(t, ts, http.StatusOK, http.MethodDelete, "deletePipelineEndpoint/"+id, "")
	if code := osisErr(t, ts, http.MethodDelete, "deletePipelineEndpoint/"+id, ""); code != "ResourceNotFoundException" {
		t.Fatalf("delete twice: %s", code)
	}

	list := osisCall(t, ts, http.StatusOK, http.MethodPost, "listPipelineBlueprints", "")["Blueprints"].([]any)
	if len(list) != len(osisBlueprints) {
		t.Fatalf("blueprints: %v", list)
	}
	name := list[0].(map[string]any)["BlueprintName"].(string)
	got := osisCall(t, ts, http.StatusOK, http.MethodGet, "getPipelineBlueprint/"+name+"?format=JSON", "")
	if got["Format"] != "JSON" || !strings.HasPrefix(got["Blueprint"].(map[string]any)["PipelineConfigurationBody"].(string), "{") {
		t.Fatalf("JSON blueprint: %v", got)
	}
	if code := osisErr(t, ts, http.MethodGet, "getPipelineBlueprint/nope", ""); code != "ResourceNotFoundException" {
		t.Fatalf("missing blueprint: %s", code)
	}
}
