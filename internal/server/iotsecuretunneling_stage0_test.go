package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

func TestIoTBatchStage0CatalogsMatchModels(t *testing.T) {
	idaOps := make([]string, 0, len(iotDeviceAdvisorOperations))
	for _, op := range iotDeviceAdvisorOperations {
		idaOps = append(idaOps, op.Name)
	}
	istOps, itgOps := []string{}, []string{}
	for _, op := range iotSecureTunnelingOperations {
		istOps = append(istOps, op.Name)
	}
	for _, op := range iotThingsGraphOperations {
		itgOps = append(itgOps, op.Name)
	}
	for _, c := range []struct {
		name     string
		ops      []string
		model    *awsmodels.Model
		handlers int
	}{
		{"iotsecuretunneling", istOps, awsmodels.IoTSecureTunnelingModel(), len(istAPI.handlers)},
		{"iotthingsgraph", itgOps, awsmodels.IoTThingsGraphModel(), len(itgAPI.handlers)},
		{"iotdeviceadvisor", idaOps, awsmodels.IoTDeviceAdvisorModel(), len(idaOps)},
	} {
		if len(c.ops) != len(c.model.Operations) || c.handlers != len(c.ops) {
			t.Fatalf("%s: catalog %d, model %d, handlers %d", c.name, len(c.ops), len(c.model.Operations), c.handlers)
		}
		for _, op := range c.ops {
			if _, ok := c.model.Operations[op]; !ok {
				t.Errorf("%s: %s is not in the model", c.name, op)
			}
		}
	}
}

// TestIoTBatchStage0Routing proves the three services claim only their own
// traffic and the existing IoT services keep theirs.
func TestIoTBatchStage0Routing(t *testing.T) {
	_, ts := newBudTestServer(t)
	budOKAPI(t, ts, istAPI, "ListTunnels", `{}`)
	budOKAPI(t, ts, itgAPI, "SearchFlowTemplates", `{}`)
	if status, out := idaCall(t, ts, http.MethodGet, "/suiteDefinitions", ""); status != http.StatusOK {
		t.Fatalf("ListSuiteDefinitions: %d %v", status, out)
	}

	// IoT Core keeps its tag and thing paths.
	resp := signedRequestWithService(t, http.MethodGet, ts.URL+"/tags?resourceArn=arn:aws:iot:us-east-1:123456789012:thing/stackyard-thing", nil, nil, "iot")
	if body := mustBody(t, resp); resp.StatusCode != http.StatusOK {
		t.Fatalf("iot ListTagsForResource: %d %s", resp.StatusCode, body)
	}
	// IoT FleetWise keeps its JSON target.
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(`{}`), map[string]string{
		"Content-Type": "application/x-amz-json-1.0", "X-Amz-Target": "IoTAutobahnControlPlane.ListFleets",
	}, "iotfleetwise")
	if body := mustBody(t, resp); resp.StatusCode != http.StatusOK {
		t.Fatalf("iotfleetwise ListFleets: %d %s", resp.StatusCode, body)
	}
	// Device Advisor's /tags/{arn} is not served for other signing names.
	resp = signedRequestWithService(t, http.MethodGet, ts.URL+"/suiteDefinitions", nil, nil, "iot")
	if body := mustBody(t, resp); resp.StatusCode == http.StatusOK && string(body) != "" && idaLooksLikeList(body) {
		t.Fatalf("iot-signed request was served by Device Advisor: %s", body)
	}
	// A tunneling target signed for another service is refused.
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(`{}`), map[string]string{
		"Content-Type": "application/x-amz-json-1.1", "X-Amz-Target": "IoTSecuredTunneling.ListTunnels",
	}, "iot")
	if body := mustBody(t, resp); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("iot-signed tunneling request: %d %s", resp.StatusCode, body)
	}
	// Unknown members are refused by the model.
	if code := budErrAPI(t, ts, istAPI, "ListTunnels", `{"Nope":1}`, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("unknown member: %s", code)
	}
	if code := budErrAPI(t, ts, itgAPI, "GetFlowTemplate", `{"id":"nope"}`, http.StatusBadRequest); code != "InvalidRequestException" {
		t.Fatalf("bad id: %s", code)
	}
}

func TestIoTSecureTunnelingStage1Lifecycle(t *testing.T) {
	srv, ts := newBudTestServer(t)
	open := budOKAPI(t, ts, istAPI, "OpenTunnel", `{"description":"ssh to pump","destinationConfig":{"thingName":"pump-1","services":["SSH"]},"timeoutConfig":{"maxLifetimeTimeoutMinutes":30},"tags":[{"key":"team","value":"ops"}]}`)
	id, _ := open["tunnelId"].(string)
	if id == "" || open["sourceAccessToken"] == "" || open["destinationAccessToken"] == "" || open["sourceAccessToken"] == open["destinationAccessToken"] {
		t.Fatalf("OpenTunnel: %v", open)
	}
	desc := budOKAPI(t, ts, istAPI, "DescribeTunnel", `{"tunnelId":"`+id+`"}`)["tunnel"].(map[string]any)
	if desc["status"] != "OPEN" || desc["destinationConfig"].(map[string]any)["thingName"] != "pump-1" {
		t.Fatalf("DescribeTunnel: %v", desc)
	}
	for _, k := range []string{"sourceAccessToken", "destinationAccessToken"} {
		if _, leaked := desc[k]; leaked {
			t.Fatalf("DescribeTunnel returned %s", k)
		}
	}
	rot := budOKAPI(t, ts, istAPI, "RotateTunnelAccessToken", `{"tunnelId":"`+id+`","clientMode":"SOURCE"}`)
	if rot["sourceAccessToken"] == nil || rot["sourceAccessToken"] == open["sourceAccessToken"] || rot["destinationAccessToken"] != nil {
		t.Fatalf("Rotate SOURCE: %v", rot)
	}
	if code := budErrAPI(t, ts, istAPI, "RotateTunnelAccessToken", `{"tunnelId":"`+id+`","clientMode":"SOURCE","destinationConfig":{"services":["SSH"]}}`, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("Rotate SOURCE with destinationConfig: %s", code)
	}
	list := budOKAPI(t, ts, istAPI, "ListTunnels", `{"thingName":"pump-1"}`)["tunnelSummaries"].([]any)
	if len(list) != 1 {
		t.Fatalf("ListTunnels: %v", list)
	}
	arn := open["tunnelArn"].(string)
	budOKAPI(t, ts, istAPI, "TagResource", `{"resourceArn":"`+arn+`","tags":[{"key":"env","value":"prod"}]}`)
	budOKAPI(t, ts, istAPI, "UntagResource", `{"resourceArn":"`+arn+`","tagKeys":["team"]}`)
	tags := budOKAPI(t, ts, istAPI, "ListTagsForResource", `{"resourceArn":"`+arn+`"}`)["tags"].([]any)
	if len(tags) != 1 || tags[0].(map[string]any)["key"] != "env" {
		t.Fatalf("tags: %v", tags)
	}
	// The tunnel closes itself once its lifetime has run out.
	st := srv.iotSecureTunneling
	st.mu.Lock()
	st.tunnels[id].created = st.tunnels[id].created.Add(-31 * 60e9)
	st.mu.Unlock()
	desc = budOKAPI(t, ts, istAPI, "DescribeTunnel", `{"tunnelId":"`+id+`"}`)["tunnel"].(map[string]any)
	if desc["status"] != "CLOSED" {
		t.Fatalf("expired tunnel: %v", desc["status"])
	}
	if code := budErrAPI(t, ts, istAPI, "RotateTunnelAccessToken", `{"tunnelId":"`+id+`","clientMode":"ALL"}`, http.StatusNotFound); code != "ResourceNotFoundException" {
		t.Fatalf("rotate closed: %s", code)
	}
	budOKAPI(t, ts, istAPI, "CloseTunnel", `{"tunnelId":"`+id+`","delete":true}`)
	if code := budErrAPI(t, ts, istAPI, "DescribeTunnel", `{"tunnelId":"`+id+`"}`, http.StatusNotFound); code != "ResourceNotFoundException" {
		t.Fatalf("deleted tunnel: %s", code)
	}
}

func TestIoTSecureTunnelingStage1OpenLimit(t *testing.T) {
	srv, ts := newBudTestServer(t)
	for i := 0; i < istMaxOpen; i++ {
		if _, err := srv.istOpenTunnel(map[string]any{}); err != nil {
			t.Fatalf("open %d: %v", i, err.msg)
		}
	}
	if code := budErrAPI(t, ts, istAPI, "OpenTunnel", `{}`, http.StatusBadRequest); code != "LimitExceededException" {
		t.Fatalf("over the open limit: %s", code)
	}
}

// TestIoTSecureTunnelingStage0ErrorShape checks the awsJson error body.
func TestIoTSecureTunnelingStage0ErrorShape(t *testing.T) {
	_, ts := newBudTestServer(t)
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(`{"tunnelId":"missing"}`), map[string]string{
		"Content-Type": "application/x-amz-json-1.1", "X-Amz-Target": "IoTSecuredTunneling.DescribeTunnel",
	}, "IoTSecuredTunneling")
	raw := mustBody(t, resp)
	var out map[string]string
	if err := json.Unmarshal(raw, &out); err != nil || resp.StatusCode != http.StatusNotFound ||
		out["__type"] != "ResourceNotFoundException" || !strings.Contains(out["Message"], "missing") {
		t.Fatalf("error shape: %d %s", resp.StatusCode, raw)
	}
}
