package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func sdmOK(t *testing.T, ts *httptest.Server, method, path, body string) map[string]any {
	t.Helper()
	return edgeOK(t, ts, "snow-device-management", method, path, body)
}

func TestSnowDeviceManagementStage1DevicesFollowSnowballJobs(t *testing.T) {
	srv, ts := newBudTestServer(t)
	devices := sdmOK(t, ts, http.MethodGet, "/managed-devices", "")["devices"].([]any)
	if len(devices) != 1 || devices[0].(map[string]any)["associatedWithJob"] != snowballDefaultJobID {
		t.Fatalf("seed device: %v", devices)
	}
	job := srv.snowball.Handle("CreateJob", map[string]any{"SnowballType": "SNC1_SSD"})["JobId"].(string)
	if n := len(sdmOK(t, ts, http.MethodGet, "/managed-devices", "")["devices"].([]any)); n != 2 {
		t.Fatalf("devices after CreateJob: %d", n)
	}
	mine := sdmOK(t, ts, http.MethodGet, "/managed-devices?jobId="+job, "")["devices"].([]any)
	dev := mine[0].(map[string]any)["managedDeviceId"].(string)
	d := sdmOK(t, ts, http.MethodPost, "/managed-device/"+dev+"/describe", "")
	if d["deviceType"] != "SNOWCONE" || d["deviceState"] != "LOCKED" {
		t.Fatalf("describe: %v", d)
	}
	res := sdmOK(t, ts, http.MethodGet, "/managed-device/"+dev+"/resources", "")["resources"].([]any)
	inst := res[0].(map[string]any)["id"].(string)
	got := sdmOK(t, ts, http.MethodPost, "/managed-device/"+dev+"/resources/ec2/describe", `{"instanceIds":["`+inst+`"]}`)["instances"].([]any)
	if len(got) != 1 {
		t.Fatalf("instances: %v", got)
	}
	if code := edgeErr(t, ts, "snow-device-management", http.MethodPost, "/managed-device/"+dev+"/resources/ec2/describe", `{"instanceIds":["s.i-nope"]}`, http.StatusNotFound); code != "ResourceNotFoundException" {
		t.Fatalf("unknown instance: %s", code)
	}

	// Unlock: QUEUED -> IN_PROGRESS (UNLOCKING) -> SUCCEEDED (UNLOCKED).
	task := sdmOK(t, ts, http.MethodPost, "/task", `{"command":{"unlock":{}},"targets":["`+dev+`"],"clientToken":"t1","tags":{"team":"edge"}}`)
	taskID := task["taskId"].(string)
	if again := sdmOK(t, ts, http.MethodPost, "/task", `{"command":{"unlock":{}},"targets":["`+dev+`"],"clientToken":"t1"}`); again["taskId"] != taskID {
		t.Fatalf("idempotency: %v", again)
	}
	if e := sdmOK(t, ts, http.MethodPost, "/task/"+taskID+"/execution/"+dev, ""); e["state"] != "IN_PROGRESS" {
		t.Fatalf("execution: %v", e)
	}
	if s := sdmOK(t, ts, http.MethodPost, "/managed-device/"+dev+"/describe", "")["deviceState"]; s != "UNLOCKING" {
		t.Fatalf("device while unlocking: %v", s)
	}
	if tk := sdmOK(t, ts, http.MethodPost, "/task/"+taskID, ""); tk["state"] != "COMPLETED" || tk["completedAt"] == nil {
		t.Fatalf("task: %v", tk)
	}
	if s := sdmOK(t, ts, http.MethodPost, "/managed-device/"+dev+"/describe", "")["deviceState"]; s != "UNLOCKED" {
		t.Fatalf("device after unlock: %v", s)
	}
	if ex := sdmOK(t, ts, http.MethodGet, "/executions?taskId="+taskID+"&state=SUCCEEDED", "")["executions"].([]any); len(ex) != 1 {
		t.Fatalf("executions: %v", ex)
	}
	if code := edgeErr(t, ts, "snow-device-management", http.MethodPost, "/task/"+taskID+"/cancel", "", http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("cancel completed task: %s", code)
	}

	// A cancelled job's device leaves management; queued work on it fails.
	reboot := sdmOK(t, ts, http.MethodPost, "/task", `{"command":{"reboot":{}},"targets":["`+dev+`"]}`)["taskId"].(string)
	srv.snowball.Handle("CancelJob", map[string]any{"JobId": job})
	if e := sdmOK(t, ts, http.MethodPost, "/task/"+reboot+"/execution/"+dev, ""); e["state"] != "FAILED" {
		t.Fatalf("execution on a removed device: %v", e)
	}
	if code := edgeErr(t, ts, "snow-device-management", http.MethodPost, "/managed-device/"+dev+"/describe", "", http.StatusNotFound); code != "ResourceNotFoundException" {
		t.Fatalf("removed device: %s", code)
	}
	// Errors carry the code header and a lowercase message member.
	if status, out := edgeCall(t, ts, "snow-device-management", http.MethodPost, "/task", `{"command":{"reboot":{}},"targets":["smd-nope"]}`); status != http.StatusNotFound || out["__code"] != "ResourceNotFoundException" || out["message"] == nil {
		t.Fatalf("unknown target: %d %v", status, out)
	}
	if code := edgeErr(t, ts, "snow-device-management", http.MethodPost, "/task", `{"command":{"reboot":{},"unlock":{}},"targets":["`+dev+`"]}`, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("two commands: %s", code)
	}

	// Cancelling a running unlock relocks the device.
	seed := devices[0].(map[string]any)["managedDeviceId"].(string)
	unlock := sdmOK(t, ts, http.MethodPost, "/task", `{"command":{"unlock":{}},"targets":["`+seed+`"]}`)["taskId"].(string)
	sdmOK(t, ts, http.MethodPost, "/task/"+unlock+"/execution/"+seed, "")
	sdmOK(t, ts, http.MethodPost, "/task/"+unlock+"/cancel", "")
	if s := sdmOK(t, ts, http.MethodPost, "/managed-device/"+seed+"/describe", "")["deviceState"]; s != "LOCKED" {
		t.Fatalf("device after cancelled unlock: %v", s)
	}

	// Cancel an in-progress task.
	running := sdmOK(t, ts, http.MethodPost, "/task", `{"command":{"reboot":{}},"targets":["`+seed+`"]}`)["taskId"].(string)
	sdmOK(t, ts, http.MethodPost, "/task/"+running+"/cancel", "")
	if s := sdmOK(t, ts, http.MethodPost, "/task/"+running, "")["state"]; s != "CANCELED" {
		t.Fatalf("cancelled task: %v", s)
	}
	if n := len(sdmOK(t, ts, http.MethodGet, "/tasks?state=CANCELED", "")["tasks"].([]any)); n != 2 {
		t.Fatalf("cancelled tasks: %d", n)
	}

	// Tags on tasks and devices.
	arn := url.PathEscape(task["taskArn"].(string))
	sdmOK(t, ts, http.MethodPost, "/tags/"+arn, `{"tags":{"env":"dev"}}`)
	sdmOK(t, ts, http.MethodDelete, "/tags/"+arn+"?tagKeys=team", "")
	if tags := sdmOK(t, ts, http.MethodGet, "/tags/"+arn, "")["tags"].(map[string]any); len(tags) != 1 || tags["env"] != "dev" {
		t.Fatalf("task tags: %v", tags)
	}
	devARN := url.PathEscape(devices[0].(map[string]any)["managedDeviceArn"].(string))
	sdmOK(t, ts, http.MethodPost, "/tags/"+devARN, `{"tags":{"site":"field"}}`)
	if tags := sdmOK(t, ts, http.MethodGet, "/managed-devices", "")["devices"].([]any)[0].(map[string]any)["tags"].(map[string]any); tags["site"] != "field" {
		t.Fatalf("device tags: %v", tags)
	}
	if code := edgeErr(t, ts, "snow-device-management", http.MethodGet, "/tasks?nextToken=abc", "", http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("bad token: %s", code)
	}
	targets := make([]string, 0, sdmMaxTargets+1)
	for i := 0; i <= sdmMaxTargets; i++ {
		targets = append(targets, `"`+seed+`"`)
	}
	if code := edgeErr(t, ts, "snow-device-management", http.MethodPost, "/task", `{"command":{"reboot":{}},"targets":[`+strings.Join(targets, ",")+`]}`, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("too many targets: %s", code)
	}
}
