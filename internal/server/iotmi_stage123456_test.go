package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestIoTManagedIntegrationsStage12AssociationsConnectorsAndConfigLifecycles(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := iotMIRequest(t, ts, http.MethodPost, "/account-associations", `{"accountAssociationId":"assoc-stage-001"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = iotMIRequest(t, ts, http.MethodGet, "/account-associations/assoc-stage-001", "")
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "assoc-stage-001") {
		t.Fatalf("expected GetAccountAssociation to include assoc-stage-001, got %q", body)
	}
	resp = iotMIRequest(t, ts, http.MethodPut, "/account-associations/assoc-stage-001", `{}`)
	assertStatus(t, resp, http.StatusOK)
	resp = iotMIRequest(t, ts, http.MethodGet, "/account-associations?MaxResults=10", "")
	assertStatus(t, resp, http.StatusOK)

	resp = iotMIRequest(t, ts, http.MethodPost, "/cloud-connectors", `{"identifier":"connector-stage-001"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = iotMIRequest(t, ts, http.MethodGet, "/cloud-connectors/connector-stage-001", "")
	assertStatus(t, resp, http.StatusOK)
	resp = iotMIRequest(t, ts, http.MethodPut, "/cloud-connectors/connector-stage-001", `{"type":"LAMBDA"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = iotMIRequest(t, ts, http.MethodGet, "/cloud-connectors?Type=LAMBDA&MaxResults=10", "")
	assertStatus(t, resp, http.StatusOK)

	resp = iotMIRequest(t, ts, http.MethodPost, "/connector-destinations", `{"identifier":"connector-destination-stage-001","cloudConnectorId":"connector-stage-001"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = iotMIRequest(t, ts, http.MethodGet, "/connector-destinations/connector-destination-stage-001", "")
	assertStatus(t, resp, http.StatusOK)
	resp = iotMIRequest(t, ts, http.MethodPut, "/connector-destinations/connector-destination-stage-001", `{}`)
	assertStatus(t, resp, http.StatusOK)
	resp = iotMIRequest(t, ts, http.MethodGet, "/connector-destinations?CloudConnectorId=connector-stage-001", "")
	assertStatus(t, resp, http.StatusOK)

	resp = iotMIRequest(t, ts, http.MethodPost, "/credential-lockers", `{"identifier":"locker-stage-001"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = iotMIRequest(t, ts, http.MethodGet, "/credential-lockers/locker-stage-001", "")
	assertStatus(t, resp, http.StatusOK)
	resp = iotMIRequest(t, ts, http.MethodGet, "/credential-lockers?MaxResults=10", "")
	assertStatus(t, resp, http.StatusOK)

	resp = iotMIRequest(t, ts, http.MethodPost, "/destinations", `{"name":"stage-destination-001"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = iotMIRequest(t, ts, http.MethodGet, "/destinations/stage-destination-001", "")
	assertStatus(t, resp, http.StatusOK)
	resp = iotMIRequest(t, ts, http.MethodPut, "/destinations/stage-destination-001", `{}`)
	assertStatus(t, resp, http.StatusOK)
	resp = iotMIRequest(t, ts, http.MethodGet, "/destinations?MaxResults=10", "")
	assertStatus(t, resp, http.StatusOK)

	resp = iotMIRequest(t, ts, http.MethodPost, "/event-log-configurations", `{"id":"event-log-stage-001"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = iotMIRequest(t, ts, http.MethodGet, "/event-log-configurations/event-log-stage-001", "")
	assertStatus(t, resp, http.StatusOK)
	resp = iotMIRequest(t, ts, http.MethodPatch, "/event-log-configurations/event-log-stage-001", `{"logLevel":"DEBUG"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = iotMIRequest(t, ts, http.MethodGet, "/event-log-configurations?MaxResults=10", "")
	assertStatus(t, resp, http.StatusOK)

	resp = iotMIRequest(t, ts, http.MethodPost, "/notification-configurations", `{"eventType":"DEVICE_DISCONNECTED"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = iotMIRequest(t, ts, http.MethodGet, "/notification-configurations/DEVICE_DISCONNECTED", "")
	assertStatus(t, resp, http.StatusOK)
	resp = iotMIRequest(t, ts, http.MethodPut, "/notification-configurations/DEVICE_DISCONNECTED", `{"status":"ENABLED"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = iotMIRequest(t, ts, http.MethodGet, "/notification-configurations?MaxResults=10", "")
	assertStatus(t, resp, http.StatusOK)

	resp = iotMIRequest(t, ts, http.MethodDelete, "/notification-configurations/DEVICE_DISCONNECTED", "")
	assertStatus(t, resp, http.StatusOK)
	resp = iotMIRequest(t, ts, http.MethodDelete, "/event-log-configurations/event-log-stage-001", "")
	assertStatus(t, resp, http.StatusOK)
	resp = iotMIRequest(t, ts, http.MethodDelete, "/destinations/stage-destination-001", "")
	assertStatus(t, resp, http.StatusOK)
	resp = iotMIRequest(t, ts, http.MethodDelete, "/credential-lockers/locker-stage-001", "")
	assertStatus(t, resp, http.StatusOK)
	resp = iotMIRequest(t, ts, http.MethodDelete, "/connector-destinations/connector-destination-stage-001", "")
	assertStatus(t, resp, http.StatusOK)
	resp = iotMIRequest(t, ts, http.MethodDelete, "/cloud-connectors/connector-stage-001", "")
	assertStatus(t, resp, http.StatusOK)
	resp = iotMIRequest(t, ts, http.MethodDelete, "/account-associations/assoc-stage-001", "")
	assertStatus(t, resp, http.StatusOK)
}

func TestIoTManagedIntegrationsStage34ManagedThingsSchemasOTAAndProvisioning(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := iotMIRequest(t, ts, http.MethodPost, "/managed-things", `{"identifier":"thing-stage-001","name":"stage-thing"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = iotMIRequest(t, ts, http.MethodGet, "/managed-things/thing-stage-001", "")
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "thing-stage-001") {
		t.Fatalf("expected GetManagedThing to include thing-stage-001, got %q", body)
	}

	resp = iotMIRequest(t, ts, http.MethodPut, "/managed-things/thing-stage-001", `{"name":"stage-thing-updated"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = iotMIRequest(t, ts, http.MethodGet, "/managed-things?MaxResults=10", "")
	assertStatus(t, resp, http.StatusOK)

	resp = iotMIRequest(t, ts, http.MethodPut, "/managed-thing-associations/register", `{"accountAssociationId":"assoc-000001","managedThingId":"thing-stage-001"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = iotMIRequest(t, ts, http.MethodGet, "/managed-thing-associations?ManagedThingId=thing-stage-001", "")
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "thing-stage-001") {
		t.Fatalf("expected ListManagedThingAccountAssociations to include thing-stage-001, got %q", body)
	}

	resp = iotMIRequest(t, ts, http.MethodGet, "/managed-things-metadata/thing-stage-001", "")
	assertStatus(t, resp, http.StatusOK)
	resp = iotMIRequest(t, ts, http.MethodGet, "/managed-things-capabilities/thing-stage-001", "")
	assertStatus(t, resp, http.StatusOK)
	resp = iotMIRequest(t, ts, http.MethodGet, "/managed-things-certificate/thing-stage-001", "")
	assertStatus(t, resp, http.StatusOK)
	resp = iotMIRequest(t, ts, http.MethodPost, "/managed-things-connectivity-data/thing-stage-001", `{}`)
	assertStatus(t, resp, http.StatusOK)
	resp = iotMIRequest(t, ts, http.MethodGet, "/managed-thing-states/thing-stage-001", "")
	assertStatus(t, resp, http.StatusOK)

	resp = iotMIRequest(t, ts, http.MethodGet, "/managed-thing-schemas/thing-stage-001?MaxResults=10", "")
	assertStatus(t, resp, http.StatusOK)
	resp = iotMIRequest(t, ts, http.MethodGet, "/schema-versions/capability/schema-000001?Format=JSON", "")
	assertStatus(t, resp, http.StatusOK)
	resp = iotMIRequest(t, ts, http.MethodGet, "/schema-versions/capability?NamespaceFilter=stackyard&MaxResults=10", "")
	assertStatus(t, resp, http.StatusOK)

	resp = iotMIRequest(t, ts, http.MethodPost, "/connector-event/connector-000001", `{}`)
	assertStatus(t, resp, http.StatusOK)
	resp = iotMIRequest(t, ts, http.MethodPost, "/managed-things-command/thing-stage-001", `{}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "command-") {
		t.Fatalf("expected SendManagedThingCommand to include command id, got %q", body)
	}

	resp = iotMIRequest(t, ts, http.MethodPost, "/ota-task-configurations", `{"identifier":"ota-config-stage-001"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = iotMIRequest(t, ts, http.MethodGet, "/ota-task-configurations/ota-config-stage-001", "")
	assertStatus(t, resp, http.StatusOK)
	resp = iotMIRequest(t, ts, http.MethodGet, "/ota-task-configurations?MaxResults=10", "")
	assertStatus(t, resp, http.StatusOK)

	resp = iotMIRequest(t, ts, http.MethodPost, "/ota-tasks", `{"identifier":"ota-task-stage-001","otaTaskConfigurationId":"ota-config-stage-001"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = iotMIRequest(t, ts, http.MethodGet, "/ota-tasks/ota-task-stage-001", "")
	assertStatus(t, resp, http.StatusOK)
	resp = iotMIRequest(t, ts, http.MethodPut, "/ota-tasks/ota-task-stage-001", `{"status":"IN_PROGRESS"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = iotMIRequest(t, ts, http.MethodGet, "/ota-tasks?MaxResults=10", "")
	assertStatus(t, resp, http.StatusOK)
	resp = iotMIRequest(t, ts, http.MethodGet, "/ota-tasks/ota-task-stage-001/devices?MaxResults=10", "")
	assertStatus(t, resp, http.StatusOK)

	resp = iotMIRequest(t, ts, http.MethodPost, "/provisioning-profiles", `{"identifier":"profile-stage-001"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = iotMIRequest(t, ts, http.MethodGet, "/provisioning-profiles/profile-stage-001", "")
	assertStatus(t, resp, http.StatusOK)
	resp = iotMIRequest(t, ts, http.MethodGet, "/provisioning-profiles?MaxResults=10", "")
	assertStatus(t, resp, http.StatusOK)

	resp = iotMIRequest(t, ts, http.MethodPost, "/device-discoveries", `{"type":"MATTER"}`)
	assertStatus(t, resp, http.StatusOK)
	discovery := map[string]any{}
	if err := json.Unmarshal(mustBody(t, resp), &discovery); err != nil {
		t.Fatalf("failed to decode StartDeviceDiscovery response: %v", err)
	}
	discoveryID, _ := discovery["identifier"].(string)
	if discoveryID == "" {
		t.Fatalf("expected StartDeviceDiscovery to return identifier")
	}
	resp = iotMIRequest(t, ts, http.MethodGet, "/device-discoveries/"+discoveryID, "")
	assertStatus(t, resp, http.StatusOK)
	resp = iotMIRequest(t, ts, http.MethodGet, "/device-discoveries?MaxResults=10&StatusFilter=IN_PROGRESS", "")
	assertStatus(t, resp, http.StatusOK)
	resp = iotMIRequest(t, ts, http.MethodGet, "/device-discoveries/"+discoveryID+"/devices?MaxResults=10", "")
	assertStatus(t, resp, http.StatusOK)

	resp = iotMIRequest(t, ts, http.MethodPut, "/managed-thing-associations/deregister", `{"accountAssociationId":"assoc-000001","managedThingId":"thing-stage-001"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = iotMIRequest(t, ts, http.MethodDelete, "/provisioning-profiles/profile-stage-001", "")
	assertStatus(t, resp, http.StatusOK)
	resp = iotMIRequest(t, ts, http.MethodDelete, "/ota-tasks/ota-task-stage-001", "")
	assertStatus(t, resp, http.StatusOK)
	resp = iotMIRequest(t, ts, http.MethodDelete, "/ota-task-configurations/ota-config-stage-001", "")
	assertStatus(t, resp, http.StatusOK)
	resp = iotMIRequest(t, ts, http.MethodDelete, "/managed-things/thing-stage-001?Force=true", "")
	assertStatus(t, resp, http.StatusOK)
}

func TestIoTManagedIntegrationsStage5ConfigurationRuntimeAndRefreshSurface(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := iotMIRequest(t, ts, http.MethodGet, "/custom-endpoint", "")
	assertStatus(t, resp, http.StatusOK)
	resp = iotMIRequest(t, ts, http.MethodPost, "/custom-endpoint", `{}`)
	assertStatus(t, resp, http.StatusOK)

	resp = iotMIRequest(t, ts, http.MethodGet, "/hub-configuration", "")
	assertStatus(t, resp, http.StatusOK)
	resp = iotMIRequest(t, ts, http.MethodPut, "/hub-configuration", `{"hubTokenTimerExpirySettingInSeconds":1800}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "1800") {
		t.Fatalf("expected PutHubConfiguration to include 1800 value, got %q", body)
	}

	resp = iotMIRequest(t, ts, http.MethodGet, "/configuration/account/encryption", "")
	assertStatus(t, resp, http.StatusOK)
	resp = iotMIRequest(t, ts, http.MethodPost, "/configuration/account/encryption", `{"kmsKeyArn":"arn:aws:kms:us-east-1:123456789012:key/stage"}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "key/stage") {
		t.Fatalf("expected PutDefaultEncryptionConfiguration response to include key/stage, got %q", body)
	}

	resp = iotMIRequest(t, ts, http.MethodPut, "/runtime-log-configurations/thing-000001", `{"logLevel":"DEBUG"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = iotMIRequest(t, ts, http.MethodGet, "/runtime-log-configurations/thing-000001", "")
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "DEBUG") {
		t.Fatalf("expected GetRuntimeLogConfiguration to include DEBUG, got %q", body)
	}
	resp = iotMIRequest(t, ts, http.MethodDelete, "/runtime-log-configurations/thing-000001", "")
	assertStatus(t, resp, http.StatusOK)
	resp = iotMIRequest(t, ts, http.MethodGet, "/runtime-log-configurations/thing-000001", "")
	assertStatus(t, resp, http.StatusOK)

	resp = iotMIRequest(t, ts, http.MethodPost, "/account-associations/assoc-000001/refresh", `{}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "IN_PROGRESS") {
		t.Fatalf("expected StartAccountAssociationRefresh to include IN_PROGRESS, got %q", body)
	}
}

func TestIoTManagedIntegrationsStage6TaggingValidationAndIdempotency(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resourceARN := "arn:aws:iotmanagedintegrations:us-east-1:123456789012:managed-thing/thing-stage-tags"
	encodedARN := url.PathEscape(resourceARN)

	resp := iotMIRequest(t, ts, http.MethodPost, "/tags/"+encodedARN, `{"tags":{"env":"stage","owner":"qa"}}`)
	assertStatus(t, resp, http.StatusOK)
	resp = iotMIRequest(t, ts, http.MethodGet, "/tags/"+encodedARN, "")
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "owner") {
		t.Fatalf("expected ListTagsForResource to include owner tag, got %q", body)
	}
	resp = iotMIRequest(t, ts, http.MethodDelete, "/tags/"+encodedARN+"?tagKeys=owner", "")
	assertStatus(t, resp, http.StatusOK)
	resp = iotMIRequest(t, ts, http.MethodGet, "/tags/"+encodedARN, "")
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); strings.Contains(body, `"owner"`) {
		t.Fatalf("expected owner tag removed, got %q", body)
	}

	resp = iotMIRequest(t, ts, http.MethodPost, "/managed-things", `{"identifier":"thing-idempotent-001"}`)
	assertStatus(t, resp, http.StatusOK)
	first := string(mustBody(t, resp))
	resp = iotMIRequest(t, ts, http.MethodPost, "/managed-things", `{"identifier":"thing-idempotent-001"}`)
	assertStatus(t, resp, http.StatusOK)
	second := string(mustBody(t, resp))
	if !strings.Contains(first, "thing-idempotent-001") || !strings.Contains(second, "thing-idempotent-001") {
		t.Fatalf("expected idempotent CreateManagedThing responses to include stable identifier")
	}

	resp = iotMIRequest(t, ts, http.MethodPost, "/totally-unknown", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException for unknown route, got %q", body)
	}

	resp = iotMIRequest(t, ts, http.MethodPost, "/custom-endpoint", `{"broken":`)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "invalid JSON body") {
		t.Fatalf("expected invalid JSON body error, got %q", body)
	}
}
