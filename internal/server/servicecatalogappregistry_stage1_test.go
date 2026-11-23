package server

import (
	"net/http"
	"net/url"
	"testing"
)

func TestAppRegistryStage1Applications(t *testing.T) {
	srv, ts := newAppRegistryTestServer(t)
	app := aprCall(t, ts, http.StatusCreated, "POST", "/applications", `{"name":"shop","description":"d","clientToken":"t1","tags":{"env":"dev"}}`)["application"].(map[string]any)
	id, arn := app["id"].(string), app["arn"].(string)
	if len(id) != 26 || arn != aprAppARN(id) || app["applicationTag"].(map[string]any)["awsApplication"] != arn {
		t.Fatalf("created application: %v", app)
	}
	// A retried clientToken returns the same application; a reused token with
	// another name, or a duplicate name, is a conflict.
	if again := aprCall(t, ts, http.StatusCreated, "POST", "/applications", `{"name":"shop","clientToken":"t1"}`)["application"].(map[string]any); again["id"] != id {
		t.Fatalf("idempotent create returned %v", again["id"])
	}
	if code := aprErrCode(t, ts, "POST", "/applications", `{"name":"other","clientToken":"t1"}`); code != "ConflictException" {
		t.Fatalf("reused token: %s", code)
	}
	if code := aprErrCode(t, ts, "POST", "/applications", `{"name":"shop","clientToken":"t2"}`); code != "ConflictException" {
		t.Fatalf("duplicate name: %s", code)
	}
	if code := aprErrCode(t, ts, "POST", "/applications", `{"name":"bad name","clientToken":"t3"}`); code != "ValidationException" {
		t.Fatalf("invalid name: %s", code)
	}
	if code := aprErrCode(t, ts, "POST", "/applications", `{"name":"x","clientToken":"t4","tags":{"aws:x":"y"}}`); code != "ValidationException" {
		t.Fatalf("reserved tag key: %s", code)
	}

	// Every operation takes a name, id or ARN.
	for _, spec := range []string{"shop", id, url.PathEscape(arn)} {
		got := aprCall(t, ts, http.StatusOK, "GET", "/applications/"+spec, "")
		if got["id"] != id || got["associatedResourceCount"] != float64(0) {
			t.Fatalf("get by %s: %v", spec, got)
		}
	}
	got := aprCall(t, ts, http.StatusOK, "GET", "/applications/shop", "")
	rg := got["integrations"].(map[string]any)["resourceGroup"].(map[string]any)
	if rg["state"] != "CREATE_COMPLETE" || rg["arn"] != aprGroupARNFor(aprAppGroupName("shop")) {
		t.Fatalf("integrations: %v", got["integrations"])
	}
	// The application's resource group really exists in Resource Groups.
	srv.resourcegroups.mu.Lock()
	_, inRG := srv.resourcegroups.groups[aprGroupARNFor(aprAppGroupName("shop"))]
	srv.resourcegroups.mu.Unlock()
	if !inRG {
		t.Fatal("CreateApplication didn't create the application's resource group")
	}
	aprCall(t, ts, http.StatusNotFound, "GET", "/applications/nope", "")

	upd := aprCall(t, ts, http.StatusOK, "PATCH", "/applications/shop", `{"description":"new","name":"store"}`)["application"].(map[string]any)
	if upd["description"] != "new" || upd["name"] != "store" {
		t.Fatalf("update: %v", upd)
	}
	// Renaming keeps the resource group the application was created with.
	renamed := aprCall(t, ts, http.StatusOK, "GET", "/applications/store", "")
	if renamed["integrations"].(map[string]any)["resourceGroup"].(map[string]any)["arn"] != aprGroupARNFor(aprAppGroupName("shop")) {
		t.Fatalf("resource group after rename: %v", renamed["integrations"])
	}
	aprCall(t, ts, http.StatusOK, "PATCH", "/applications/store", `{"name":"shop"}`)
	aprCall(t, ts, http.StatusCreated, "POST", "/applications", `{"name":"other","clientToken":"t5"}`)
	page := aprCall(t, ts, http.StatusOK, "GET", "/applications?maxResults=1", "")
	if len(page["applications"].([]any)) != 1 || page["nextToken"] == nil {
		t.Fatalf("paging: %v", page)
	}
	rest := aprCall(t, ts, http.StatusOK, "GET", "/applications?nextToken="+url.QueryEscape(page["nextToken"].(string)), "")
	if len(rest["applications"].([]any)) != 1 || rest["nextToken"] != nil {
		t.Fatalf("second page: %v", rest)
	}
	if code := aprErrCode(t, ts, "GET", "/applications?nextToken=bm9wZQ==", ""); code != "ValidationException" {
		t.Fatalf("bad token: %s", code)
	}

	// Tags work on application ARNs.
	tagPath := "/tags/" + url.PathEscape(arn)
	aprCall(t, ts, http.StatusOK, "POST", tagPath, `{"tags":{"team":"a"}}`)
	if tags := aprCall(t, ts, http.StatusOK, "GET", tagPath, "")["tags"].(map[string]any); tags["env"] != "dev" || tags["team"] != "a" {
		t.Fatalf("tags: %v", tags)
	}
	aprCall(t, ts, http.StatusOK, "DELETE", tagPath+"?tagKeys=team", "")
	aprCall(t, ts, http.StatusNotFound, "GET", "/tags/"+url.PathEscape(aprAppARN("zzzzzzzzzzzzzzzzzzzzzzzzzz")), "")

	del := aprCall(t, ts, http.StatusOK, "DELETE", "/applications/other", "")["application"].(map[string]any)
	if del["name"] != "other" {
		t.Fatalf("delete: %v", del)
	}
	aprCall(t, ts, http.StatusNotFound, "GET", "/applications/other", "")
}

func TestAppRegistryStage1AttributeGroupsAndResources(t *testing.T) {
	srv, ts := newAppRegistryTestServer(t)
	aprCall(t, ts, http.StatusCreated, "POST", "/applications", `{"name":"shop","clientToken":"a1"}`)
	if code := aprErrCode(t, ts, "POST", "/attribute-groups", `{"name":"g","attributes":"not json","clientToken":"g0"}`); code != "ValidationException" {
		t.Fatalf("non-JSON attributes: %s", code)
	}
	g := aprCall(t, ts, http.StatusCreated, "POST", "/attribute-groups", `{"name":"meta","attributes":"{\"owner\":\"team-a\"}","clientToken":"g1"}`)["attributeGroup"].(map[string]any)
	gid := g["id"].(string)
	if got := aprCall(t, ts, http.StatusOK, "GET", "/attribute-groups/meta", ""); got["attributes"] != `{"owner":"team-a"}` {
		t.Fatalf("get attribute group: %v", got)
	}

	aprCall(t, ts, http.StatusOK, "PUT", "/applications/shop/attribute-groups/meta", "")
	if code := aprErrCode(t, ts, "PUT", "/applications/shop/attribute-groups/"+gid, ""); code != "ConflictException" {
		t.Fatalf("double association: %s", code)
	}
	if ids := aprCall(t, ts, http.StatusOK, "GET", "/applications/shop/attribute-groups", "")["attributeGroups"].([]any); len(ids) != 1 || ids[0] != gid {
		t.Fatalf("associated groups: %v", ids)
	}
	details := aprCall(t, ts, http.StatusOK, "GET", "/applications/shop/attribute-group-details", "")["attributeGroupsDetails"].([]any)
	if details[0].(map[string]any)["name"] != "meta" {
		t.Fatalf("group details: %v", details)
	}
	// An associated group can't be deleted, and an application with
	// associations can't be either.
	if code := aprErrCode(t, ts, "DELETE", "/attribute-groups/meta", ""); code != "ValidationException" {
		t.Fatalf("delete associated group: %s", code)
	}

	// CFN_STACK resources must exist in CloudFormation and belong to one app.
	if code := aprErrCode(t, ts, "PUT", "/applications/shop/resources/CFN_STACK/missing", "{}"); code != "ResourceNotFoundException" {
		t.Fatalf("missing stack: %s", code)
	}
	stackARN := srv.cloudformation.stacks["stackyard-default"]
	res := aprCall(t, ts, http.StatusOK, "PUT", "/applications/shop/resources/CFN_STACK/stackyard-default", `{"options":["APPLY_APPLICATION_TAG"]}`)
	if res["resourceArn"] != stackARN {
		t.Fatalf("associate stack: %v", res)
	}
	aprCall(t, ts, http.StatusCreated, "POST", "/applications", `{"name":"other","clientToken":"a2"}`)
	if code := aprErrCode(t, ts, "PUT", "/applications/other/resources/CFN_STACK/"+url.PathEscape(stackARN), "{}"); code != "ConflictException" {
		t.Fatalf("stack in two applications: %s", code)
	}
	if code := aprErrCode(t, ts, "PUT", "/applications/other/resources/RESOURCE_TAG_VALUE/v1", `{"options":["APPLY_APPLICATION_TAG","SKIP_APPLICATION_TAG"]}`); code != "ValidationException" {
		t.Fatalf("contradictory options: %s", code)
	}
	aprCall(t, ts, http.StatusOK, "PUT", "/applications/shop/resources/RESOURCE_TAG_VALUE/v1", "{}")

	got := aprCall(t, ts, http.StatusOK, "GET", "/applications/shop/resources/CFN_STACK/stackyard-default?resourceTagStatus=SUCCESS", "")
	if got["resource"].(map[string]any)["arn"] != stackARN || got["applicationTagResult"].(map[string]any)["applicationTagStatus"] != "SUCCESS" {
		t.Fatalf("associated resource: %v", got)
	}
	if code := aprErrCode(t, ts, "GET", "/applications/shop/resources/CFN_STACK/stackyard-default?resourceTagStatus=BOGUS", ""); code != "ValidationException" {
		t.Fatalf("bad status filter: %s", code)
	}
	list := aprCall(t, ts, http.StatusOK, "GET", "/applications/shop/resources", "")["resources"].([]any)
	if len(list) != 2 || list[1].(map[string]any)["resourceDetails"].(map[string]any)["tagValue"] != "v1" {
		t.Fatalf("associated resources: %v", list)
	}
	if n := aprCall(t, ts, http.StatusOK, "GET", "/applications/shop", "")["associatedResourceCount"]; n != float64(2) {
		t.Fatalf("resource count: %v", n)
	}

	// SyncResource reports START_SYNC for an associated stack.
	sync := aprCall(t, ts, http.StatusOK, "POST", "/sync/CFN_STACK/stackyard-default", "")
	if sync["actionTaken"] != "START_SYNC" || sync["applicationArn"] == nil {
		t.Fatalf("sync: %v", sync)
	}
	if code := aprErrCode(t, ts, "POST", "/sync/CFN_STACK/missing", ""); code != "ResourceNotFoundException" {
		t.Fatalf("sync missing stack: %s", code)
	}

	if code := aprErrCode(t, ts, "DELETE", "/applications/shop", ""); code != "ValidationException" {
		t.Fatalf("delete application with associations: %s", code)
	}
	aprCall(t, ts, http.StatusOK, "DELETE", "/applications/shop/resources/CFN_STACK/"+url.PathEscape(stackARN), "")
	aprCall(t, ts, http.StatusOK, "DELETE", "/applications/shop/resources/RESOURCE_TAG_VALUE/v1", "")
	aprCall(t, ts, http.StatusOK, "DELETE", "/applications/shop/attribute-groups/meta", "")
	aprCall(t, ts, http.StatusNotFound, "DELETE", "/applications/shop/attribute-groups/meta", "")
	// Once released, the stack can join another application.
	aprCall(t, ts, http.StatusOK, "PUT", "/applications/other/resources/CFN_STACK/stackyard-default", "{}")
	aprCall(t, ts, http.StatusOK, "DELETE", "/applications/shop", "")
	aprCall(t, ts, http.StatusOK, "DELETE", "/attribute-groups/meta", "")
	// A deleted group's clientToken is released.
	again := aprCall(t, ts, http.StatusCreated, "POST", "/attribute-groups", `{"name":"meta2","attributes":"{}","clientToken":"g1"}`)["attributeGroup"].(map[string]any)
	if again["id"] == gid {
		t.Fatal("deleted group's token still resolves to it")
	}
}

func TestAppRegistryStage1Configuration(t *testing.T) {
	_, ts := newAppRegistryTestServer(t)
	aprCall(t, ts, http.StatusOK, "PUT", "/configuration", `{"configuration":{"tagQueryConfiguration":{"tagKey":"app"}}}`)
	cfg := aprCall(t, ts, http.StatusOK, "GET", "/configuration", "")["configuration"].(map[string]any)
	if cfg["tagQueryConfiguration"].(map[string]any)["tagKey"] != "app" {
		t.Fatalf("configuration: %v", cfg)
	}
	if code := aprErrCode(t, ts, "PUT", "/configuration", `{"configuration":{"tagQueryConfiguration":{"tagKey":"   "}}}`); code != "ValidationException" {
		t.Fatalf("whitespace tag key: %s", code)
	}
	if code := aprErrCode(t, ts, "PUT", "/configuration", `{}`); code != "ValidationException" {
		t.Fatalf("missing configuration: %s", code)
	}
}
