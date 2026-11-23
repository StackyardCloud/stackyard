package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

const sarTestTemplate = `{
  "Transform": "AWS::Serverless-2016-10-31",
  "Parameters": {
    "Stage": {"Type": "String", "Default": "dev", "AllowedValues": ["dev", "prod"], "Description": "Stage"},
    "TableName": {"Type": "String", "MinLength": 3}
  },
  "Resources": {
    "Fn": {"Type": "AWS::Serverless::Function", "Properties": {"Environment": {"Variables": {"STAGE": {"Ref": "Stage"}}}}},
    "Role": {"Type": "AWS::IAM::Role", "Properties": {"RoleName": {"Ref": "TableName"}}},
    "Nested": {"Type": "AWS::Serverless::Application", "Properties": {"Location": {"ApplicationId": "arn:aws:serverlessrepo:us-east-1:111111111111:applications/dep", "SemanticVersion": "1.0.0"}}}
  }
}`

func sarAppPath(name string) string {
	return "/applications/" + url.PathEscape(sarAppARN(name))
}

func TestServerlessRepoStage1Applications(t *testing.T) {
	srv, ts := newServerlessRepoTestServer(t)
	tpl, _ := json.Marshal(sarTestTemplate)
	create := fmt.Sprintf(`{"name":"hello","author":"stackyard-dev","description":"Hello app","semanticVersion":"1.0.0","templateBody":%s,"readmeBody":"# Hello","labels":["demo"]}`, tpl)
	app := sarCall(t, ts, http.StatusCreated, http.MethodPost, "/applications", create)
	ver := app["version"].(map[string]any)
	defs := ver["parameterDefinitions"].([]any)
	if app["applicationId"] != sarAppARN("hello") || len(defs) != 2 || !strings.HasPrefix(app["readmeUrl"].(string), "https://"+sarBucket) {
		t.Fatalf("create: %v", app)
	}
	stage := defs[0].(map[string]any)
	if stage["name"] != "Stage" || stage["defaultValue"] != "dev" || stage["referencedByResources"].([]any)[0] != "Fn" {
		t.Fatalf("parameter definition: %v", stage)
	}
	caps := fmt.Sprint(ver["requiredCapabilities"])
	if caps != "[CAPABILITY_IAM CAPABILITY_NAMED_IAM CAPABILITY_AUTO_EXPAND]" {
		t.Fatalf("capabilities: %s", caps)
	}
	if _, err := srv.s3.GetObject(sarBucket, "hello/README.md"); err != nil {
		t.Fatalf("readme body wasn't stored: %v", err)
	}

	// Validation and conflicts.
	for name, body := range map[string]string{
		"bad name":            `{"name":"bad name","author":"a","description":"d"}`,
		"bad author":          `{"name":"x","author":"-Bad","description":"d"}`,
		"template no version": fmt.Sprintf(`{"name":"x","author":"a","description":"d","templateBody":%s}`, tpl),
		"version no template": `{"name":"x","author":"a","description":"d","semanticVersion":"1.0.0"}`,
		"both readme":         `{"name":"x","author":"a","description":"d","readmeBody":"r","readmeUrl":"https://b.s3.amazonaws.com/k"}`,
		"missing template":    `{"name":"x","author":"a","description":"d","semanticVersion":"1.0.0","templateUrl":"https://nope.s3.amazonaws.com/t.json"}`,
		"unknown member":      `{"name":"x","author":"a","description":"d","bogus":1}`,
	} {
		if code := sarErr(t, ts, http.MethodPost, "/applications", body); code != "BadRequestException" {
			t.Fatalf("%s: %s", name, code)
		}
	}
	if code := sarErr(t, ts, http.MethodPost, "/applications", create); code != "ConflictException" {
		t.Fatalf("duplicate: %s", code)
	}

	// Versions: semver checked, duplicates conflict, missing apps 404.
	path := sarAppPath("hello")
	sarCall(t, ts, http.StatusCreated, http.MethodPut, path+"/versions/1.1.0", fmt.Sprintf(`{"templateBody":%s}`, tpl))
	if code := sarErr(t, ts, http.MethodPut, path+"/versions/1.1.0", fmt.Sprintf(`{"templateBody":%s}`, tpl)); code != "ConflictException" {
		t.Fatalf("duplicate version: %s", code)
	}
	if code := sarErr(t, ts, http.MethodPut, path+"/versions/v1", fmt.Sprintf(`{"templateBody":%s}`, tpl)); code != "BadRequestException" {
		t.Fatalf("bad semver: %s", code)
	}
	if code := sarErr(t, ts, http.MethodPut, sarAppPath("nope")+"/versions/1.0.0", fmt.Sprintf(`{"templateBody":%s}`, tpl)); code != "NotFoundException" {
		t.Fatalf("missing app: %s", code)
	}
	if got := sarCall(t, ts, http.StatusOK, http.MethodGet, path, "")["version"].(map[string]any)["semanticVersion"]; got != "1.1.0" {
		t.Fatalf("latest version: %v", got)
	}
	if got := sarCall(t, ts, http.StatusOK, http.MethodGet, path+"?semanticVersion=1.0.0", "")["version"].(map[string]any)["semanticVersion"]; got != "1.0.0" {
		t.Fatalf("pinned version: %v", got)
	}
	vs := sarCall(t, ts, http.StatusOK, http.MethodGet, path+"/versions?maxItems=1", "")
	if len(vs["versions"].([]any)) != 1 || vs["nextToken"] != "1" {
		t.Fatalf("version paging: %v", vs)
	}
	deps := sarCall(t, ts, http.StatusOK, http.MethodGet, path+"/dependencies", "")["dependencies"].([]any)
	if len(deps) != 1 || deps[0].(map[string]any)["semanticVersion"] != "1.0.0" {
		t.Fatalf("dependencies: %v", deps)
	}

	// Update and list.
	up := sarCall(t, ts, http.StatusOK, http.MethodPatch, path, `{"description":"Updated","labels":["a","b"]}`)
	if up["description"] != "Updated" || len(up["labels"].([]any)) != 2 {
		t.Fatalf("update: %v", up)
	}
	if apps := sarCall(t, ts, http.StatusOK, http.MethodGet, "/applications", "")["applications"].([]any); len(apps) != 1 {
		t.Fatalf("list: %v", apps)
	}
}

func TestServerlessRepoStage1PolicyTemplatesAndChangeSets(t *testing.T) {
	srv, ts := newServerlessRepoTestServer(t)
	tpl, _ := json.Marshal(sarTestTemplate)
	sarCall(t, ts, http.StatusCreated, http.MethodPost, "/applications", fmt.Sprintf(`{"name":"app","author":"a","description":"d","semanticVersion":"1.0.0","templateBody":%s}`, tpl))
	path := sarAppPath("app")

	// Policy: allowed actions and principals, org sharing, unshare.
	if code := sarErr(t, ts, http.MethodPut, path+"/policy", `{"statements":[{"actions":["DeleteApplication"],"principals":["*"]}]}`); code != "BadRequestException" {
		t.Fatalf("bad action: %s", code)
	}
	if code := sarErr(t, ts, http.MethodPut, path+"/policy", `{"statements":[{"actions":["Deploy"],"principals":["123"]}]}`); code != "BadRequestException" {
		t.Fatalf("bad principal: %s", code)
	}
	sarCall(t, ts, http.StatusOK, http.MethodPut, path+"/policy", `{"statements":[{"actions":["Deploy"],"principals":["*"],"principalOrgIDs":["o-abcdefghij"]}]}`)
	sarCall(t, ts, http.StatusNoContent, http.MethodPost, path+"/unshare", `{"organizationId":"o-abcdefghij"}`)
	if st := sarCall(t, ts, http.StatusOK, http.MethodGet, path+"/policy", "")["statements"].([]any); len(st) != 0 {
		t.Fatalf("after unshare: %v", st)
	}
	if code := sarErr(t, ts, http.MethodPost, path+"/unshare", `{"organizationId":"o-abcdefghij"}`); code != "NotFoundException" {
		t.Fatalf("unshare again: %s", code)
	}

	// Generated templates are PREPARING, then ACTIVE, and stored in S3.
	g := sarCall(t, ts, http.StatusCreated, http.MethodPost, path+"/templates", `{"semanticVersion":"1.0.0"}`)
	id := g["templateId"].(string)
	if g["status"] != "PREPARING" {
		t.Fatalf("template: %v", g)
	}
	sarCall(t, ts, http.StatusOK, http.MethodGet, path+"/templates/"+id, "")
	if got := sarCall(t, ts, http.StatusOK, http.MethodGet, path+"/templates/"+id, "")["status"]; got != "ACTIVE" {
		t.Fatalf("template status: %v", got)
	}
	if _, err := srv.s3.GetObject(sarBucket, "app/templates/1.0.0.yaml"); err != nil {
		t.Fatalf("generated template wasn't stored: %v", err)
	}

	// Change sets: required parameters, allowed values, capabilities.
	cs := func(body string) string { return sarErr(t, ts, http.MethodPost, path+"/changesets", body) }
	allCaps := `"capabilities":["CAPABILITY_IAM","CAPABILITY_NAMED_IAM","CAPABILITY_AUTO_EXPAND"]`
	if code := cs(`{"stackName":"s1",` + allCaps + `}`); code != "BadRequestException" {
		t.Fatalf("missing required parameter: %s", code)
	}
	if code := cs(`{"stackName":"s1",` + allCaps + `,"parameterOverrides":[{"name":"TableName","value":"t"},{"name":"Stage","value":"qa"}]}`); code != "BadRequestException" {
		t.Fatalf("disallowed value: %s", code)
	}
	if code := cs(`{"stackName":"s1","capabilities":["CAPABILITY_IAM"],"parameterOverrides":[{"name":"TableName","value":"t"}]}`); code != "BadRequestException" {
		t.Fatalf("missing capability: %s", code)
	}
	ok := `{"stackName":"s1","clientToken":"tok",` + allCaps + `,"parameterOverrides":[{"name":"TableName","value":"t"}],"templateId":"` + id + `"}`
	out := sarCall(t, ts, http.StatusCreated, http.MethodPost, path+"/changesets", ok)
	if !strings.Contains(out["stackId"].(string), ":stack/serverlessrepo-s1/") || !strings.Contains(out["changeSetId"].(string), ":changeSet/") {
		t.Fatalf("change set: %v", out)
	}
	if again := sarCall(t, ts, http.StatusCreated, http.MethodPost, path+"/changesets", ok); again["changeSetId"] != out["changeSetId"] {
		t.Fatalf("client token retry: %v vs %v", again, out)
	}
	srv.cloudformation.mu.Lock()
	_, registered := srv.cloudformation.stacks["serverlessrepo-s1"]
	srv.cloudformation.mu.Unlock()
	if !registered {
		t.Fatal("stack not registered in CloudFormation")
	}

	sarCall(t, ts, http.StatusNoContent, http.MethodDelete, path, "")
	if code := sarErr(t, ts, http.MethodGet, path, ""); code != "NotFoundException" {
		t.Fatalf("after delete: %s", code)
	}
}

func TestServerlessRepoStage1TemplateParsing(t *testing.T) {
	for name, body := range map[string]string{
		"empty":         ``,
		"no resources":  `{"Resources":{}}`,
		"untyped":       `{"Resources":{"A":{}}}`,
		"not template":  `hello world`,
		"bad parameter": `{"Parameters":{"P":"x"},"Resources":{"A":{"Type":"AWS::SNS::Topic"}}}`,
	} {
		if _, err := sarParseTemplate(body); err == nil {
			t.Fatalf("%s: accepted", name)
		}
	}
	if tpl, err := sarParseTemplate("AWSTemplateFormatVersion: '2010-09-09'\nResources:\n  T:\n    Type: AWS::SNS::Topic\n"); err != nil || len(tpl.params) != 0 {
		t.Fatalf("yaml: %v %v", tpl, err)
	}
	tpl, err := sarParseTemplate(`{"Resources":{"P":{"Type":"AWS::Lambda::Permission"}}}`)
	if err != nil || fmt.Sprint(tpl.capabilities) != "[CAPABILITY_RESOURCE_POLICY]" {
		t.Fatalf("resource policy capability: %v %v", tpl, err)
	}
}

// Stored template text is bounded across the whole repository.
func TestServerlessRepoStage1TemplateStorageIsBounded(t *testing.T) {
	srv, ts := newServerlessRepoTestServer(t)
	sarCall(t, ts, http.StatusCreated, http.MethodPost, "/applications", `{"name":"big","author":"a","description":"d"}`)
	srv.serverlessRepo.mu.Lock()
	srv.serverlessRepo.bytes = sarMaxStoredBytes - 10
	srv.serverlessRepo.mu.Unlock()
	tpl, _ := json.Marshal(sarTestTemplate)
	if code := sarErr(t, ts, http.MethodPut, sarAppPath("big")+"/versions/1.0.0", fmt.Sprintf(`{"templateBody":%s}`, tpl)); code != "TooManyRequestsException" {
		t.Fatalf("over budget: %s", code)
	}
	sarCall(t, ts, http.StatusNoContent, http.MethodDelete, sarAppPath("big"), "")
}
