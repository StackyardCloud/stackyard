package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const dbeTestCluster = `{"clusterName":"%s","adminUserName":"admin","adminUserPassword":"Passw0rd1","authType":"PLAIN_TEXT","shardCapacity":2,"shardCount":2}`

func dbeCreate(t *testing.T, ts *httptest.Server, name string) string {
	t.Helper()
	c := dbeCall(t, ts, http.StatusOK, http.MethodPost, "/cluster", fmt.Sprintf(dbeTestCluster, name))["cluster"].(map[string]any)
	return c["clusterArn"].(string)
}

func dbeErrCode(t *testing.T, ts *httptest.Server, want int, method, path, body string) string {
	t.Helper()
	resp := signedRequestWithService(t, method, ts.URL+path, []byte(body), map[string]string{"Content-Type": "application/json"}, "docdb-elastic")
	defer resp.Body.Close()
	if resp.StatusCode != want {
		t.Fatalf("%s %s: %d (want %d)", method, path, resp.StatusCode, want)
	}
	return resp.Header.Get("X-Amzn-ErrorType")
}

func TestDocDBElasticStage1ClusterLifecycle(t *testing.T) {
	_, ts := newDocDBElasticTestServer(t)
	out := dbeCall(t, ts, http.StatusOK, http.MethodPost, "/cluster", fmt.Sprintf(dbeTestCluster, "orders"))["cluster"].(map[string]any)
	arn := out["clusterArn"].(string)
	if out["status"] != "CREATING" || out["kmsKeyId"] != "AWS_OWNED_KMS_KEY" || len(out["shards"].([]any)) != 2 || out["shardInstanceCount"] != float64(2) {
		t.Fatalf("create: %v", out)
	}
	if _, leaked := out["adminUserPassword"]; leaked || strings.Contains(fmt.Sprint(out), "Passw0rd1") {
		t.Fatalf("password returned: %v", out)
	}
	if subnets := out["subnetIds"].([]any); len(subnets) == 0 || len(out["vpcSecurityGroupIds"].([]any)) != 1 {
		t.Fatalf("default network: %v", out)
	}
	// Changes before the cluster settles are conflicts; the next read settles it.
	if code := dbeErrCode(t, ts, http.StatusConflict, http.MethodPut, "/cluster/"+dbeEsc(arn), `{"shardCount":3}`); code != "ConflictException" {
		t.Fatalf("update while creating: %s", code)
	}
	if got := dbeCall(t, ts, http.StatusOK, http.MethodGet, "/cluster/"+dbeEsc(arn), "")["cluster"].(map[string]any); got["status"] != "ACTIVE" {
		t.Fatalf("settle: %v", got)
	}
	up := dbeCall(t, ts, http.StatusOK, http.MethodPut, "/cluster/"+dbeEsc(arn), `{"shardCount":3,"preferredMaintenanceWindow":"Mon:01:00-Mon:02:00"}`)["cluster"].(map[string]any)
	if up["status"] != "UPDATING" || up["shardCount"] != float64(3) || len(up["shards"].([]any)) != 3 || up["preferredMaintenanceWindow"] != "mon:01:00-mon:02:00" {
		t.Fatalf("update: %v", up)
	}
	dbeCall(t, ts, http.StatusOK, http.MethodGet, "/clusters", "")
	// Stop, settle, start: raw (unescaped) ARNs in the path work too.
	if got := dbeCall(t, ts, http.StatusOK, http.MethodPost, "/cluster/"+arn+"/stop", "")["cluster"].(map[string]any); got["status"] != "STOPPING" {
		t.Fatalf("stop: %v", got)
	}
	dbeCall(t, ts, http.StatusOK, http.MethodGet, "/cluster/"+dbeEsc(arn), "")
	if code := dbeErrCode(t, ts, http.StatusBadRequest, http.MethodPost, "/cluster/"+dbeEsc(arn)+"/stop", ""); code != "ValidationException" {
		t.Fatalf("stop while stopped: %s", code)
	}
	dbeCall(t, ts, http.StatusOK, http.MethodPost, "/cluster/"+dbeEsc(arn)+"/start", "")
	dbeCall(t, ts, http.StatusOK, http.MethodGet, "/cluster/"+dbeEsc(arn), "")
	if got := dbeCall(t, ts, http.StatusOK, http.MethodDelete, "/cluster/"+dbeEsc(arn), "")["cluster"].(map[string]any); got["status"] != "DELETING" {
		t.Fatalf("delete: %v", got)
	}
	if code := dbeErrCode(t, ts, http.StatusNotFound, http.MethodGet, "/cluster/"+dbeEsc(arn), ""); code != "ResourceNotFoundException" {
		t.Fatalf("get deleted: %s", code)
	}
}

func TestDocDBElasticStage1Validation(t *testing.T) {
	_, ts := newDocDBElasticTestServer(t)
	for name, body := range map[string]string{
		"missing required":  `{"clusterName":"a"}`,
		"bad capacity":      `{"clusterName":"a","adminUserName":"admin","adminUserPassword":"Passw0rd1","authType":"PLAIN_TEXT","shardCapacity":3,"shardCount":2}`,
		"too many shards":   `{"clusterName":"a","adminUserName":"admin","adminUserPassword":"Passw0rd1","authType":"PLAIN_TEXT","shardCapacity":2,"shardCount":33}`,
		"bad name":          `{"clusterName":"9bad--name","adminUserName":"admin","adminUserPassword":"Passw0rd1","authType":"PLAIN_TEXT","shardCapacity":2,"shardCount":1}`,
		"short password":    `{"clusterName":"a","adminUserName":"admin","adminUserPassword":"short","authType":"PLAIN_TEXT","shardCapacity":2,"shardCount":1}`,
		"missing secret":    `{"clusterName":"a","adminUserName":"admin","adminUserPassword":"arn:aws:secretsmanager:us-east-1:123456789012:secret:nope","authType":"SECRET_ARN","shardCapacity":2,"shardCount":1}`,
		"unknown subnet":    `{"clusterName":"a","adminUserName":"admin","adminUserPassword":"Passw0rd1","authType":"PLAIN_TEXT","shardCapacity":2,"shardCount":1,"subnetIds":["subnet-0000000000000beef"]}`,
		"unknown sg":        `{"clusterName":"a","adminUserName":"admin","adminUserPassword":"Passw0rd1","authType":"PLAIN_TEXT","shardCapacity":2,"shardCount":1,"vpcSecurityGroupIds":["sg-0000000000000beef"]}`,
		"short maintenance": `{"clusterName":"a","adminUserName":"admin","adminUserPassword":"Passw0rd1","authType":"PLAIN_TEXT","shardCapacity":2,"shardCount":1,"preferredMaintenanceWindow":"mon:01:00-mon:01:10"}`,
		"aws tag":           `{"clusterName":"a","adminUserName":"admin","adminUserPassword":"Passw0rd1","authType":"PLAIN_TEXT","shardCapacity":2,"shardCount":1,"tags":{"aws:x":"y"}}`,
	} {
		if code := dbeErrCode(t, ts, http.StatusBadRequest, http.MethodPost, "/cluster", body); code != "ValidationException" {
			t.Fatalf("%s: %s", name, code)
		}
	}
	dbeCreate(t, ts, "dup")
	if code := dbeErrCode(t, ts, http.StatusConflict, http.MethodPost, "/cluster", fmt.Sprintf(dbeTestCluster, "dup")); code != "ConflictException" {
		t.Fatalf("duplicate name: %s", code)
	}
	// The same clientToken returns the cluster it created.
	body := `{"clusterName":"tok","adminUserName":"admin","adminUserPassword":"Passw0rd1","authType":"PLAIN_TEXT","shardCapacity":2,"shardCount":1,"clientToken":"t1"}`
	a := dbeCall(t, ts, http.StatusOK, http.MethodPost, "/cluster", body)["cluster"].(map[string]any)["clusterArn"]
	b := dbeCall(t, ts, http.StatusOK, http.MethodPost, "/cluster", body)["cluster"].(map[string]any)["clusterArn"]
	if a != b {
		t.Fatalf("clientToken: %v != %v", a, b)
	}
	if code := dbeErrCode(t, ts, http.StatusBadRequest, http.MethodGet, "/clusters?maxResults=0", ""); code != "ValidationException" {
		t.Fatalf("maxResults: %s", code)
	}
	if code := dbeErrCode(t, ts, http.StatusBadRequest, http.MethodGet, "/clusters?nextToken=bogus", ""); code != "ValidationException" {
		t.Fatalf("nextToken: %s", code)
	}
}

func TestDocDBElasticStage1SnapshotsMaintenanceTags(t *testing.T) {
	_, ts := newDocDBElasticTestServer(t)
	arn := dbeCreate(t, ts, "snaps")
	if code := dbeErrCode(t, ts, http.StatusConflict, http.MethodPost, "/cluster-snapshot", `{"clusterArn":"`+arn+`","snapshotName":"s1"}`); code != "ConflictException" {
		t.Fatalf("snapshot of creating cluster: %s", code)
	}
	dbeCall(t, ts, http.StatusOK, http.MethodGet, "/cluster/"+dbeEsc(arn), "")
	// Settling to ACTIVE takes one automated snapshot and adds a maintenance action.
	auto := dbeCall(t, ts, http.StatusOK, http.MethodGet, "/cluster-snapshots?snapshotType=automated", "")["snapshots"].([]any)
	if len(auto) != 1 {
		t.Fatalf("automated snapshots: %v", auto)
	}
	autoArn := auto[0].(map[string]any)["snapshotArn"].(string)
	if code := dbeErrCode(t, ts, http.StatusBadRequest, http.MethodDelete, "/cluster-snapshot/"+dbeEsc(autoArn), ""); code != "ValidationException" {
		t.Fatalf("delete automated: %s", code)
	}
	snap := dbeCall(t, ts, http.StatusOK, http.MethodPost, "/cluster-snapshot", `{"clusterArn":"`+arn+`","snapshotName":"s1","tags":{"team":"data"}}`)["snapshot"].(map[string]any)
	sarn := snap["snapshotArn"].(string)
	if snap["status"] != "CREATING" || snap["snapshotType"] != "MANUAL" {
		t.Fatalf("create snapshot: %v", snap)
	}
	if code := dbeErrCode(t, ts, http.StatusConflict, http.MethodPost, "/cluster-snapshot/"+dbeEsc(sarn)+"/copy", `{"targetSnapshotName":"s2"}`); code != "ConflictException" {
		t.Fatalf("copy while creating: %s", code)
	}
	dbeCall(t, ts, http.StatusOK, http.MethodGet, "/cluster-snapshot/"+dbeEsc(sarn), "")
	cp := dbeCall(t, ts, http.StatusOK, http.MethodPost, "/cluster-snapshot/"+dbeEsc(sarn)+"/copy", `{"targetSnapshotName":"s2","copyTags":true}`)["snapshot"].(map[string]any)
	if tags := dbeCall(t, ts, http.StatusOK, http.MethodGet, "/tags/"+dbeEsc(cp["snapshotArn"].(string)), "")["tags"].(map[string]any); tags["team"] != "data" {
		t.Fatalf("copyTags: %v", tags)
	}
	if code := dbeErrCode(t, ts, http.StatusConflict, http.MethodPost, "/cluster-snapshot/"+dbeEsc(sarn)+"/copy", `{"targetSnapshotName":"s2"}`); code != "ConflictException" {
		t.Fatalf("duplicate snapshot name: %s", code)
	}
	restored := dbeCall(t, ts, http.StatusOK, http.MethodPost, "/cluster-snapshot/"+dbeEsc(sarn)+"/restore", `{"clusterName":"restored","shardCapacity":4}`)["cluster"].(map[string]any)
	if restored["adminUserName"] != "admin" || restored["shardCount"] != float64(2) || restored["shardCapacity"] != float64(4) {
		t.Fatalf("restore: %v", restored)
	}
	// Maintenance: opt in, then apply now; the action is gone once the cluster settles.
	pending := dbeCall(t, ts, http.StatusOK, http.MethodGet, "/pending-action/"+dbeEsc(arn), "")["resourcePendingMaintenanceAction"].(map[string]any)
	if len(pending["pendingMaintenanceActionDetails"].([]any)) != 1 {
		t.Fatalf("pending: %v", pending)
	}
	dbeCall(t, ts, http.StatusOK, http.MethodPost, "/pending-action", `{"resourceArn":"`+arn+`","applyAction":"SYSTEM_UPDATE","optInType":"NEXT_MAINTENANCE"}`)
	if code := dbeErrCode(t, ts, http.StatusBadRequest, http.MethodPost, "/pending-action", `{"resourceArn":"`+arn+`","applyAction":"SYSTEM_UPDATE","optInType":"APPLY_ON","applyOn":"soon"}`); code != "ValidationException" {
		t.Fatalf("bad applyOn: %s", code)
	}
	if code := dbeErrCode(t, ts, http.StatusNotFound, http.MethodPost, "/pending-action", `{"resourceArn":"`+arn+`","applyAction":"OTHER","optInType":"IMMEDIATE"}`); code != "ResourceNotFoundException" {
		t.Fatalf("unknown action: %s", code)
	}
	dbeCall(t, ts, http.StatusOK, http.MethodPost, "/pending-action", `{"resourceArn":"`+arn+`","applyAction":"SYSTEM_UPDATE","optInType":"IMMEDIATE"}`)
	if got := dbeCall(t, ts, http.StatusOK, http.MethodGet, "/cluster/"+dbeEsc(arn), "")["cluster"].(map[string]any); got["status"] != "ACTIVE" {
		t.Fatalf("after maintenance: %v", got)
	}
	if left := dbeCall(t, ts, http.StatusOK, http.MethodGet, "/pending-action/"+dbeEsc(arn), "")["resourcePendingMaintenanceAction"].(map[string]any)["pendingMaintenanceActionDetails"].([]any); len(left) != 0 {
		t.Fatalf("maintenance not applied: %v", left)
	}
	// Tags.
	dbeCall(t, ts, http.StatusOK, http.MethodPost, "/tags/"+dbeEsc(arn), `{"tags":{"env":"dev"}}`)
	dbeCall(t, ts, http.StatusOK, http.MethodDelete, "/tags/"+dbeEsc(arn)+"?tagKeys=env", "")
	if tags := dbeCall(t, ts, http.StatusOK, http.MethodGet, "/tags/"+dbeEsc(arn), "")["tags"].(map[string]any); len(tags) != 0 {
		t.Fatalf("untag: %v", tags)
	}
	if code := dbeErrCode(t, ts, http.StatusNotFound, http.MethodGet, "/tags/"+dbeEsc("arn:aws:docdb-elastic:us-east-1:123456789012:cluster/nope"), ""); code != "ResourceNotFoundException" {
		t.Fatalf("tags of missing resource: %s", code)
	}
	// Deleting the cluster removes its automated snapshot but keeps manual ones.
	dbeCall(t, ts, http.StatusOK, http.MethodDelete, "/cluster/"+dbeEsc(arn), "")
	left := dbeCall(t, ts, http.StatusOK, http.MethodGet, "/cluster-snapshots?clusterArn="+dbeEsc(arn), "")["snapshots"].([]any)
	if len(left) != 2 {
		t.Fatalf("snapshots after delete: %v", left)
	}
}
