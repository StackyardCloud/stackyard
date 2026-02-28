package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRedshiftServerlessStage12NamespaceAndWorkgroupLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := redshiftServerlessRequest(t, ts, "CreateNamespace", `{"namespaceName":"stage-ns","adminUsername":"admin","dbName":"dev"}`)
	assertStatus(t, resp, http.StatusOK)
	createNamespacePayload := decodeRedshiftServerlessPayload(t, resp)
	namespace := redshiftServerlessMap(createNamespacePayload, "namespace")
	if got := redshiftServerlessString(namespace, "namespaceName"); got != "stage-ns" {
		t.Fatalf("expected namespaceName stage-ns, got %q", got)
	}

	resp = redshiftServerlessRequest(t, ts, "GetNamespace", `{"namespaceName":"stage-ns"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = redshiftServerlessRequest(t, ts, "ListNamespaces", `{}`)
	assertStatus(t, resp, http.StatusOK)

	resp = redshiftServerlessRequest(t, ts, "UpdateNamespace", `{"namespaceName":"stage-ns","adminUsername":"stage_admin"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = redshiftServerlessRequest(t, ts, "CreateWorkgroup", `{"workgroupName":"stage-wg","namespaceName":"stage-ns","baseCapacity":64}`)
	assertStatus(t, resp, http.StatusOK)
	createWorkgroupPayload := decodeRedshiftServerlessPayload(t, resp)
	workgroup := redshiftServerlessMap(createWorkgroupPayload, "workgroup")
	if got := redshiftServerlessString(workgroup, "workgroupName"); got != "stage-wg" {
		t.Fatalf("expected workgroupName stage-wg, got %q", got)
	}

	resp = redshiftServerlessRequest(t, ts, "GetWorkgroup", `{"workgroupName":"stage-wg"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = redshiftServerlessRequest(t, ts, "ListWorkgroups", `{}`)
	assertStatus(t, resp, http.StatusOK)

	resp = redshiftServerlessRequest(t, ts, "UpdateWorkgroup", `{"workgroupName":"stage-wg","baseCapacity":128}`)
	assertStatus(t, resp, http.StatusOK)

	resp = redshiftServerlessRequest(t, ts, "DeleteWorkgroup", `{"workgroupName":"stage-wg"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = redshiftServerlessRequest(t, ts, "DeleteNamespace", `{"namespaceName":"stage-ns"}`)
	assertStatus(t, resp, http.StatusOK)
}

func TestRedshiftServerlessStage34SnapshotRecoveryAndReadSurface(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := redshiftServerlessRequest(t, ts, "CreateSnapshot", `{"namespaceName":"stackyard-namespace","snapshotName":"stage-snapshot"}`)
	assertStatus(t, resp, http.StatusOK)
	createSnapshotPayload := decodeRedshiftServerlessPayload(t, resp)
	snapshot := redshiftServerlessMap(createSnapshotPayload, "snapshot")
	if got := redshiftServerlessString(snapshot, "snapshotName"); got != "stage-snapshot" {
		t.Fatalf("expected snapshotName stage-snapshot, got %q", got)
	}

	resp = redshiftServerlessRequest(t, ts, "GetSnapshot", `{"snapshotName":"stage-snapshot"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = redshiftServerlessRequest(t, ts, "ListSnapshots", `{}`)
	assertStatus(t, resp, http.StatusOK)

	resp = redshiftServerlessRequest(t, ts, "ListRecoveryPoints", `{}`)
	assertStatus(t, resp, http.StatusOK)

	resp = redshiftServerlessRequest(t, ts, "GetRecoveryPoint", `{"recoveryPointId":"rp-000001"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = redshiftServerlessRequest(t, ts, "RestoreFromSnapshot", `{"snapshotName":"stage-snapshot","namespaceName":"stackyard-namespace"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = redshiftServerlessRequest(t, ts, "RestoreFromRecoveryPoint", `{"recoveryPointId":"rp-000001","namespaceName":"stackyard-namespace"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = redshiftServerlessRequest(t, ts, "RestoreTableFromSnapshot", `{"snapshotName":"stage-snapshot","namespaceName":"stackyard-namespace","sourceTableName":"src","targetTableName":"dst"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = redshiftServerlessRequest(t, ts, "RestoreTableFromRecoveryPoint", `{"recoveryPointId":"rp-000001","namespaceName":"stackyard-namespace","sourceTableName":"src","targetTableName":"dst"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = redshiftServerlessRequest(t, ts, "GetTableRestoreStatus", `{"tableRestoreRequestId":"trs-000001"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = redshiftServerlessRequest(t, ts, "ListTableRestoreStatus", `{}`)
	assertStatus(t, resp, http.StatusOK)

	resp = redshiftServerlessRequest(t, ts, "ConvertRecoveryPointToSnapshot", `{"recoveryPointId":"rp-000001","snapshotName":"stage-converted"}`)
	assertStatus(t, resp, http.StatusOK)
}

func TestRedshiftServerlessStage45AdminTaggingAndPolicies(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resourceARN := "arn:aws:redshift-serverless:us-east-1:123456789012:workgroup/stackyard-workgroup"

	resp := redshiftServerlessRequest(t, ts, "CreateEndpointAccess", `{"endpointName":"stage-endpoint","workgroupName":"stackyard-workgroup"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = redshiftServerlessRequest(t, ts, "GetEndpointAccess", `{"endpointName":"stage-endpoint"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = redshiftServerlessRequest(t, ts, "ListEndpointAccess", `{}`)
	assertStatus(t, resp, http.StatusOK)
	resp = redshiftServerlessRequest(t, ts, "UpdateEndpointAccess", `{"endpointName":"stage-endpoint"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = redshiftServerlessRequest(t, ts, "DeleteEndpointAccess", `{"endpointName":"stage-endpoint"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = redshiftServerlessRequest(t, ts, "CreateCustomDomainAssociation", `{"customDomainName":"stage.example.com","workgroupName":"stackyard-workgroup"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = redshiftServerlessRequest(t, ts, "GetCustomDomainAssociation", `{"customDomainName":"stage.example.com"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = redshiftServerlessRequest(t, ts, "ListCustomDomainAssociations", `{}`)
	assertStatus(t, resp, http.StatusOK)
	resp = redshiftServerlessRequest(t, ts, "UpdateCustomDomainAssociation", `{"customDomainName":"stage.example.com"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = redshiftServerlessRequest(t, ts, "DeleteCustomDomainAssociation", `{"customDomainName":"stage.example.com"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = redshiftServerlessRequest(t, ts, "CreateScheduledAction", `{"scheduledActionName":"stage-schedule","namespaceName":"stackyard-namespace","workgroupName":"stackyard-workgroup","schedule":"rate(1 day)"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = redshiftServerlessRequest(t, ts, "GetScheduledAction", `{"scheduledActionName":"stage-schedule"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = redshiftServerlessRequest(t, ts, "ListScheduledActions", `{}`)
	assertStatus(t, resp, http.StatusOK)
	resp = redshiftServerlessRequest(t, ts, "UpdateScheduledAction", `{"scheduledActionName":"stage-schedule","schedule":"rate(2 days)"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = redshiftServerlessRequest(t, ts, "DeleteScheduledAction", `{"scheduledActionName":"stage-schedule"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = redshiftServerlessRequest(t, ts, "CreateUsageLimit", `{"usageLimitId":"stage-ul","namespaceName":"stackyard-namespace","workgroupName":"stackyard-workgroup","amount":200,"usageType":"serverless-compute","period":"daily"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = redshiftServerlessRequest(t, ts, "GetUsageLimit", `{"usageLimitId":"stage-ul"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = redshiftServerlessRequest(t, ts, "ListUsageLimits", `{}`)
	assertStatus(t, resp, http.StatusOK)
	resp = redshiftServerlessRequest(t, ts, "UpdateUsageLimit", `{"usageLimitId":"stage-ul","amount":250}`)
	assertStatus(t, resp, http.StatusOK)
	resp = redshiftServerlessRequest(t, ts, "DeleteUsageLimit", `{"usageLimitId":"stage-ul"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = redshiftServerlessRequest(t, ts, "PutResourcePolicy", `{"resourceArn":"`+resourceARN+`","policy":"{\"Version\":\"2012-10-17\",\"Statement\":[]}"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = redshiftServerlessRequest(t, ts, "GetResourcePolicy", `{"resourceArn":"`+resourceARN+`"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = redshiftServerlessRequest(t, ts, "DeleteResourcePolicy", `{"resourceArn":"`+resourceARN+`"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = redshiftServerlessRequest(t, ts, "TagResource", `{"resourceArn":"`+resourceARN+`","tags":[{"key":"owner","value":"qa"},{"key":"env","value":"stage"}]}`)
	assertStatus(t, resp, http.StatusOK)
	resp = redshiftServerlessRequest(t, ts, "ListTagsForResource", `{"resourceArn":"`+resourceARN+`"}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "owner") {
		t.Fatalf("expected ListTagsForResource to include owner tag, got %q", body)
	}
	resp = redshiftServerlessRequest(t, ts, "UntagResource", `{"resourceArn":"`+resourceARN+`","tagKeys":["owner"]}`)
	assertStatus(t, resp, http.StatusOK)
}

func TestRedshiftServerlessStage6ValidationAndIdempotency(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := redshiftServerlessRequest(t, ts, "CreateNamespace", `{"namespaceName":"idempotent-ns"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = redshiftServerlessRequest(t, ts, "CreateNamespace", `{"namespaceName":"idempotent-ns"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = redshiftServerlessRequest(t, ts, "CreateWorkgroup", `{"workgroupName":"idempotent-wg","namespaceName":"idempotent-ns"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = redshiftServerlessRequest(t, ts, "CreateWorkgroup", `{"workgroupName":"idempotent-wg","namespaceName":"idempotent-ns"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = redshiftServerlessRequest(t, ts, "GetCredentials", `{"workgroupName":"idempotent-wg","dbUser":"admin"}`)
	assertStatus(t, resp, http.StatusOK)
	credentialsPayload := decodeRedshiftServerlessPayload(t, resp)
	if redshiftServerlessString(credentialsPayload, "dbPassword") == "" {
		t.Fatalf("expected GetCredentials dbPassword")
	}

	resp = redshiftServerlessRequest(t, ts, "GetIdentityCenterAuthToken", `{}`)
	assertStatus(t, resp, http.StatusOK)

	resp = redshiftServerlessRequest(t, ts, "ListManagedWorkgroups", `{}`)
	assertStatus(t, resp, http.StatusOK)
	resp = redshiftServerlessRequest(t, ts, "GetTrack", `{"trackName":"current"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = redshiftServerlessRequest(t, ts, "ListTracks", `{}`)
	assertStatus(t, resp, http.StatusOK)

	resp = redshiftServerlessRequest(t, ts, "CreateReservation", `{"reservationName":"stage-reservation"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = redshiftServerlessRequest(t, ts, "GetReservation", `{"reservationName":"stage-reservation"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = redshiftServerlessRequest(t, ts, "GetReservationOffering", `{"reservationOfferingId":"offering-000001"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = redshiftServerlessRequest(t, ts, "ListReservations", `{}`)
	assertStatus(t, resp, http.StatusOK)
	resp = redshiftServerlessRequest(t, ts, "ListReservationOfferings", `{}`)
	assertStatus(t, resp, http.StatusOK)

	resp = redshiftServerlessRequest(t, ts, "CreateSnapshotCopyConfiguration", `{"snapshotCopyConfigurationName":"stage-copy","namespaceName":"idempotent-ns","destinationRegion":"us-west-2","retentionPeriod":7}`)
	assertStatus(t, resp, http.StatusOK)
	resp = redshiftServerlessRequest(t, ts, "UpdateSnapshotCopyConfiguration", `{"snapshotCopyConfigurationName":"stage-copy","destinationRegion":"us-west-1","retentionPeriod":14}`)
	assertStatus(t, resp, http.StatusOK)
	resp = redshiftServerlessRequest(t, ts, "ListSnapshotCopyConfigurations", `{}`)
	assertStatus(t, resp, http.StatusOK)
	resp = redshiftServerlessRequest(t, ts, "DeleteSnapshotCopyConfiguration", `{"snapshotCopyConfigurationName":"stage-copy"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = redshiftServerlessRequest(t, ts, "TotallyUnknownAction", `{}`)
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
			"X-Amz-Target": "RedshiftServerless.ListWorkgroups",
		},
		"redshift-serverless",
	)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "invalid JSON body") {
		t.Fatalf("expected invalid JSON body error, got %q", body)
	}
}

func decodeRedshiftServerlessPayload(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	payload := map[string]any{}
	if err := json.Unmarshal(mustBody(t, resp), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	return payload
}

func redshiftServerlessMap(payload map[string]any, key string) map[string]any {
	raw, ok := payload[key]
	if !ok || raw == nil {
		return map[string]any{}
	}
	value, ok := raw.(map[string]any)
	if !ok {
		return map[string]any{}
	}
	return value
}

func redshiftServerlessString(payload map[string]any, key string) string {
	raw, ok := payload[key]
	if !ok || raw == nil {
		return ""
	}
	value, ok := raw.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}
