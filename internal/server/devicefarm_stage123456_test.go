package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDeviceFarmStage12ProjectPoolAndUploadLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := deviceFarmRequest(t, ts, "CreateProject", `{"name":"stage-devicefarm-project"}`)
	assertStatus(t, resp, http.StatusOK)
	createProjectPayload := decodeDeviceFarmPayload(t, resp)
	projectARN := deviceFarmStringField(deviceFarmMap(createProjectPayload, "project"), "arn")
	if projectARN == "" {
		t.Fatalf("expected CreateProject to include project arn")
	}

	resp = deviceFarmRequest(t, ts, "GetProject", `{"arn":"`+projectARN+`"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = deviceFarmRequest(t, ts, "UpdateProject", `{"arn":"`+projectARN+`","name":"stage-devicefarm-project-updated"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = deviceFarmRequest(t, ts, "ListProjects", `{}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "stage-devicefarm-project-updated") {
		t.Fatalf("expected ListProjects to include updated project name, got %q", body)
	}

	resp = deviceFarmRequest(t, ts, "CreateDevicePool", `{"projectArn":"`+projectARN+`","name":"stage-device-pool"}`)
	assertStatus(t, resp, http.StatusOK)
	createPoolPayload := decodeDeviceFarmPayload(t, resp)
	devicePoolARN := deviceFarmStringField(deviceFarmMap(createPoolPayload, "devicePool"), "arn")
	if devicePoolARN == "" {
		t.Fatalf("expected CreateDevicePool to include device pool arn")
	}

	resp = deviceFarmRequest(t, ts, "GetDevicePool", `{"projectArn":"`+projectARN+`","arn":"`+devicePoolARN+`"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = deviceFarmRequest(t, ts, "ListDevicePools", `{"arn":"`+projectARN+`"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = deviceFarmRequest(t, ts, "UpdateDevicePool", `{"projectArn":"`+projectARN+`","arn":"`+devicePoolARN+`","name":"stage-device-pool-updated"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = deviceFarmRequest(t, ts, "CreateUpload", `{"projectArn":"`+projectARN+`","name":"stage-upload.apk","type":"ANDROID_APP"}`)
	assertStatus(t, resp, http.StatusOK)
	createUploadPayload := decodeDeviceFarmPayload(t, resp)
	uploadARN := deviceFarmStringField(deviceFarmMap(createUploadPayload, "upload"), "arn")
	if uploadARN == "" {
		t.Fatalf("expected CreateUpload to include upload arn")
	}

	resp = deviceFarmRequest(t, ts, "GetUpload", `{"projectArn":"`+projectARN+`","arn":"`+uploadARN+`"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = deviceFarmRequest(t, ts, "ListUploads", `{"arn":"`+projectARN+`"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = deviceFarmRequest(t, ts, "UpdateUpload", `{"projectArn":"`+projectARN+`","arn":"`+uploadARN+`","status":"SUCCEEDED"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = deviceFarmRequest(t, ts, "DeleteUpload", `{"projectArn":"`+projectARN+`","arn":"`+uploadARN+`"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = deviceFarmRequest(t, ts, "DeleteDevicePool", `{"projectArn":"`+projectARN+`","arn":"`+devicePoolARN+`"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = deviceFarmRequest(t, ts, "DeleteProject", `{"arn":"`+projectARN+`"}`)
	assertStatus(t, resp, http.StatusOK)
}

func TestDeviceFarmStage34RunRemoteAccessAndTestGridLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	projectARN := "arn:aws:devicefarm:us-east-1:123456789012:project:project-000001"

	resp := deviceFarmRequest(t, ts, "ScheduleRun", `{"projectArn":"`+projectARN+`","name":"stage-devicefarm-run"}`)
	assertStatus(t, resp, http.StatusOK)
	startRunPayload := decodeDeviceFarmPayload(t, resp)
	runARN := deviceFarmStringField(deviceFarmMap(startRunPayload, "run"), "arn")
	if runARN == "" {
		t.Fatalf("expected ScheduleRun to include run arn")
	}

	resp = deviceFarmRequest(t, ts, "GetRun", `{"projectArn":"`+projectARN+`","arn":"`+runARN+`"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = deviceFarmRequest(t, ts, "ListRuns", `{"arn":"`+projectARN+`"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = deviceFarmRequest(t, ts, "ListJobs", `{"arn":"`+runARN+`"}`)
	assertStatus(t, resp, http.StatusOK)
	jobsPayload := decodeDeviceFarmPayload(t, resp)
	jobARN := firstDeviceFarmARN(jobsPayload, "jobs")
	if jobARN == "" {
		t.Fatalf("expected ListJobs to return at least one job arn")
	}

	resp = deviceFarmRequest(t, ts, "GetJob", `{"runArn":"`+runARN+`","arn":"`+jobARN+`"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = deviceFarmRequest(t, ts, "StopJob", `{"runArn":"`+runARN+`","arn":"`+jobARN+`"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = deviceFarmRequest(t, ts, "ListSuites", `{"arn":"`+runARN+`"}`)
	assertStatus(t, resp, http.StatusOK)
	suitesPayload := decodeDeviceFarmPayload(t, resp)
	suiteARN := firstDeviceFarmARN(suitesPayload, "suites")
	if suiteARN == "" {
		t.Fatalf("expected ListSuites to return at least one suite arn")
	}
	resp = deviceFarmRequest(t, ts, "GetSuite", `{"runArn":"`+runARN+`","arn":"`+suiteARN+`"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = deviceFarmRequest(t, ts, "ListTests", `{"arn":"`+runARN+`"}`)
	assertStatus(t, resp, http.StatusOK)
	testsPayload := decodeDeviceFarmPayload(t, resp)
	testARN := firstDeviceFarmARN(testsPayload, "tests")
	if testARN == "" {
		t.Fatalf("expected ListTests to return at least one test arn")
	}
	resp = deviceFarmRequest(t, ts, "GetTest", `{"runArn":"`+runARN+`","arn":"`+testARN+`"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = deviceFarmRequest(t, ts, "ListArtifacts", `{"arn":"`+runARN+`"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = deviceFarmRequest(t, ts, "ListSamples", `{"arn":"`+runARN+`"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = deviceFarmRequest(t, ts, "ListUniqueProblems", `{"arn":"`+runARN+`"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = deviceFarmRequest(t, ts, "CreateRemoteAccessSession", `{"projectArn":"`+projectARN+`","name":"stage-remote-access-session"}`)
	assertStatus(t, resp, http.StatusOK)
	remotePayload := decodeDeviceFarmPayload(t, resp)
	remoteSessionARN := deviceFarmStringField(deviceFarmMap(remotePayload, "remoteAccessSession"), "arn")
	if remoteSessionARN == "" {
		t.Fatalf("expected CreateRemoteAccessSession to include session arn")
	}

	resp = deviceFarmRequest(t, ts, "InstallToRemoteAccessSession", `{"arn":"`+remoteSessionARN+`","appArn":"arn:aws:devicefarm:us-east-1:123456789012:upload:project-000001/upload-000001"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = deviceFarmRequest(t, ts, "GetRemoteAccessSession", `{"projectArn":"`+projectARN+`","arn":"`+remoteSessionARN+`"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = deviceFarmRequest(t, ts, "ListRemoteAccessSessions", `{"arn":"`+projectARN+`"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = deviceFarmRequest(t, ts, "StopRemoteAccessSession", `{"projectArn":"`+projectARN+`","arn":"`+remoteSessionARN+`"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = deviceFarmRequest(t, ts, "DeleteRemoteAccessSession", `{"projectArn":"`+projectARN+`","arn":"`+remoteSessionARN+`"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = deviceFarmRequest(t, ts, "CreateTestGridProject", `{"name":"stage-testgrid-project"}`)
	assertStatus(t, resp, http.StatusOK)
	createTestGridProjectPayload := decodeDeviceFarmPayload(t, resp)
	testGridProjectARN := deviceFarmStringField(deviceFarmMap(createTestGridProjectPayload, "testGridProject"), "arn")
	if testGridProjectARN == "" {
		t.Fatalf("expected CreateTestGridProject to include project arn")
	}

	resp = deviceFarmRequest(t, ts, "GetTestGridProject", `{"arn":"`+testGridProjectARN+`"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = deviceFarmRequest(t, ts, "UpdateTestGridProject", `{"arn":"`+testGridProjectARN+`","name":"stage-testgrid-project-updated"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = deviceFarmRequest(t, ts, "ListTestGridProjects", `{}`)
	assertStatus(t, resp, http.StatusOK)

	resp = deviceFarmRequest(t, ts, "CreateTestGridUrl", `{"projectArn":"`+testGridProjectARN+`"}`)
	assertStatus(t, resp, http.StatusOK)
	testGridURLPayload := decodeDeviceFarmPayload(t, resp)
	testGridSessionARN := deviceFarmStringField(testGridURLPayload, "testGridSessionArn")

	resp = deviceFarmRequest(t, ts, "ListTestGridSessions", `{"projectArn":"`+testGridProjectARN+`"}`)
	assertStatus(t, resp, http.StatusOK)
	if testGridSessionARN == "" {
		testGridSessionsPayload := decodeDeviceFarmPayload(t, resp)
		testGridSessionARN = firstDeviceFarmARN(testGridSessionsPayload, "testGridSessions")
	}
	if testGridSessionARN == "" {
		t.Fatalf("expected test grid session arn from CreateTestGridUrl or ListTestGridSessions")
	}
	resp = deviceFarmRequest(t, ts, "GetTestGridSession", `{"arn":"`+testGridSessionARN+`"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = deviceFarmRequest(t, ts, "ListTestGridSessionActions", `{"arn":"`+testGridSessionARN+`"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = deviceFarmRequest(t, ts, "ListTestGridSessionArtifacts", `{"arn":"`+testGridSessionARN+`"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = deviceFarmRequest(t, ts, "DeleteTestGridProject", `{"arn":"`+testGridProjectARN+`"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = deviceFarmRequest(t, ts, "StopRun", `{"projectArn":"`+projectARN+`","arn":"`+runARN+`"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = deviceFarmRequest(t, ts, "DeleteRun", `{"projectArn":"`+projectARN+`","arn":"`+runARN+`"}`)
	assertStatus(t, resp, http.StatusOK)
}

func TestDeviceFarmStage56ProfilesOfferingsTaggingValidationAndIdempotency(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := deviceFarmRequest(t, ts, "GetAccountSettings", `{}`)
	assertStatus(t, resp, http.StatusOK)

	resp = deviceFarmRequest(t, ts, "CreateInstanceProfile", `{"name":"stage-instance-profile"}`)
	assertStatus(t, resp, http.StatusOK)
	createInstanceProfilePayload := decodeDeviceFarmPayload(t, resp)
	instanceProfileARN := deviceFarmStringField(deviceFarmMap(createInstanceProfilePayload, "instanceProfile"), "arn")
	if instanceProfileARN == "" {
		t.Fatalf("expected CreateInstanceProfile to include arn")
	}
	resp = deviceFarmRequest(t, ts, "GetInstanceProfile", `{"arn":"`+instanceProfileARN+`"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = deviceFarmRequest(t, ts, "ListInstanceProfiles", `{}`)
	assertStatus(t, resp, http.StatusOK)
	resp = deviceFarmRequest(t, ts, "UpdateInstanceProfile", `{"arn":"`+instanceProfileARN+`","name":"stage-instance-profile-updated"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = deviceFarmRequest(t, ts, "CreateNetworkProfile", `{"name":"stage-network-profile"}`)
	assertStatus(t, resp, http.StatusOK)
	createNetworkProfilePayload := decodeDeviceFarmPayload(t, resp)
	networkProfileARN := deviceFarmStringField(deviceFarmMap(createNetworkProfilePayload, "networkProfile"), "arn")
	if networkProfileARN == "" {
		t.Fatalf("expected CreateNetworkProfile to include arn")
	}
	resp = deviceFarmRequest(t, ts, "GetNetworkProfile", `{"arn":"`+networkProfileARN+`"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = deviceFarmRequest(t, ts, "ListNetworkProfiles", `{}`)
	assertStatus(t, resp, http.StatusOK)
	resp = deviceFarmRequest(t, ts, "UpdateNetworkProfile", `{"arn":"`+networkProfileARN+`","name":"stage-network-profile-updated"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = deviceFarmRequest(t, ts, "CreateVPCEConfiguration", `{"name":"stage-vpce-configuration"}`)
	assertStatus(t, resp, http.StatusOK)
	createVPCEPayload := decodeDeviceFarmPayload(t, resp)
	vpceConfigurationARN := deviceFarmStringField(deviceFarmMap(createVPCEPayload, "vpceConfiguration"), "arn")
	if vpceConfigurationARN == "" {
		t.Fatalf("expected CreateVPCEConfiguration to include arn")
	}
	resp = deviceFarmRequest(t, ts, "GetVPCEConfiguration", `{"arn":"`+vpceConfigurationARN+`"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = deviceFarmRequest(t, ts, "ListVPCEConfigurations", `{}`)
	assertStatus(t, resp, http.StatusOK)
	resp = deviceFarmRequest(t, ts, "UpdateVPCEConfiguration", `{"arn":"`+vpceConfigurationARN+`","name":"stage-vpce-configuration-updated"}`)
	assertStatus(t, resp, http.StatusOK)

	resourceARN := "arn:aws:devicefarm:us-east-1:123456789012:project:project-000001"
	resp = deviceFarmRequest(t, ts, "TagResource", `{"resourceARN":"`+resourceARN+`","tags":{"env":"stage","owner":"qa"}}`)
	assertStatus(t, resp, http.StatusOK)
	resp = deviceFarmRequest(t, ts, "TagResource", `{"resourceARN":"`+resourceARN+`","tags":{"owner":"qa"}}`)
	assertStatus(t, resp, http.StatusOK)
	resp = deviceFarmRequest(t, ts, "ListTagsForResource", `{"resourceARN":"`+resourceARN+`"}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "owner") {
		t.Fatalf("expected ListTagsForResource to include owner tag, got %q", body)
	}
	resp = deviceFarmRequest(t, ts, "UntagResource", `{"resourceARN":"`+resourceARN+`","tagKeys":["owner"]}`)
	assertStatus(t, resp, http.StatusOK)

	resp = deviceFarmRequest(t, ts, "ListOfferings", `{}`)
	assertStatus(t, resp, http.StatusOK)
	resp = deviceFarmRequest(t, ts, "PurchaseOffering", `{"offeringId":"offering-000001","quantity":1}`)
	assertStatus(t, resp, http.StatusOK)
	resp = deviceFarmRequest(t, ts, "RenewOffering", `{"offeringId":"offering-000001","quantity":1}`)
	assertStatus(t, resp, http.StatusOK)
	resp = deviceFarmRequest(t, ts, "GetOfferingStatus", `{"offeringId":"offering-000001"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = deviceFarmRequest(t, ts, "ListOfferingTransactions", `{}`)
	assertStatus(t, resp, http.StatusOK)
	resp = deviceFarmRequest(t, ts, "ListOfferingPromotions", `{}`)
	assertStatus(t, resp, http.StatusOK)

	resp = deviceFarmRequest(t, ts, "GetDevice", `{"arn":"arn:aws:devicefarm:us-east-1:123456789012:device:device-000001"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = deviceFarmRequest(t, ts, "ListDevices", `{}`)
	assertStatus(t, resp, http.StatusOK)
	resp = deviceFarmRequest(t, ts, "GetDeviceInstance", `{"arn":"arn:aws:devicefarm:us-east-1:123456789012:deviceinstance:di-000001"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = deviceFarmRequest(t, ts, "ListDeviceInstances", `{}`)
	assertStatus(t, resp, http.StatusOK)
	resp = deviceFarmRequest(t, ts, "UpdateDeviceInstance", `{"arn":"arn:aws:devicefarm:us-east-1:123456789012:deviceinstance:di-000001","status":"AVAILABLE"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = deviceFarmRequest(t, ts, "DeleteVPCEConfiguration", `{"arn":"`+vpceConfigurationARN+`"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = deviceFarmRequest(t, ts, "DeleteNetworkProfile", `{"arn":"`+networkProfileARN+`"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = deviceFarmRequest(t, ts, "DeleteInstanceProfile", `{"arn":"`+instanceProfileARN+`"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = deviceFarmRequest(t, ts, "TotallyUnknownAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException for unknown action, got %q", body)
	}

	resp = signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(`{"broken":`),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "DeviceFarm_20150623.ListProjects",
		},
		"devicefarm",
	)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "invalid JSON body") {
		t.Fatalf("expected invalid JSON body error, got %q", body)
	}
}

func decodeDeviceFarmPayload(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	out := map[string]any{}
	if err := json.Unmarshal(mustBody(t, resp), &out); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	return out
}

func deviceFarmMap(payload map[string]any, key string) map[string]any {
	if payload == nil {
		return map[string]any{}
	}
	raw, ok := payload[key]
	if !ok {
		return map[string]any{}
	}
	value, ok := raw.(map[string]any)
	if !ok {
		return map[string]any{}
	}
	return value
}

func deviceFarmStringField(payload map[string]any, key string) string {
	if payload == nil {
		return ""
	}
	raw, ok := payload[key]
	if !ok {
		return ""
	}
	value, ok := raw.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}

func firstDeviceFarmARN(payload map[string]any, key string) string {
	if payload == nil {
		return ""
	}
	raw, ok := payload[key]
	if !ok {
		return ""
	}
	items, ok := raw.([]any)
	if !ok || len(items) == 0 {
		return ""
	}
	item, ok := items[0].(map[string]any)
	if !ok {
		return ""
	}
	return deviceFarmStringField(item, "arn")
}
