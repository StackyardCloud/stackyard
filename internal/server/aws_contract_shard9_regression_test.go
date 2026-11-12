package server

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func decodeShard9JSONBody(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	var out map[string]any
	dec := json.NewDecoder(bytes.NewReader(mustBody(t, resp)))
	dec.UseNumber()
	if err := dec.Decode(&out); err != nil {
		t.Fatalf("decode JSON body: %v", err)
	}
	return out
}

func TestCloudControlAPIShard9ProgressEventOmitsHooksProgressEvent(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := cloudControlAPIRequest(t, ts, "CreateResource", map[string]any{
		"TypeName":     "AWS::S3::Bucket",
		"DesiredState": `{"BucketName":"stackyard-shard9"}`,
	})
	assertStatus(t, resp, http.StatusOK)
	createBody := cloudControlAPIDecodeBody(t, resp)
	progressEvent, _ := createBody["ProgressEvent"].(map[string]any)
	requestToken, _ := progressEvent["RequestToken"].(string)
	if strings.TrimSpace(requestToken) == "" {
		t.Fatalf("expected RequestToken in CreateResource response")
	}

	resp = cloudControlAPIRequest(t, ts, "GetResourceRequestStatus", map[string]any{"RequestToken": requestToken})
	assertStatus(t, resp, http.StatusOK)
	statusBody := cloudControlAPIDecodeBody(t, resp)
	progressEvent, _ = statusBody["ProgressEvent"].(map[string]any)
	if _, ok := progressEvent["HooksProgressEvent"]; ok {
		t.Fatalf("expected HooksProgressEvent to be omitted, got %#v", progressEvent["HooksProgressEvent"])
	}
}

func TestCodeGuruShard9RecommendationAndEmptyMutationShapes(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	reviewArn := codeGuruCodeReviewARN("code-review-000001")
	associationArn := codeGuruAssociationARN("association-000001")

	resp := codeGuruRequest(t, ts, http.MethodGet, "/codereviews/"+url.PathEscape(reviewArn)+"/Recommendations", "")
	assertStatus(t, resp, http.StatusOK)
	recommendationsBody := decodeShard9JSONBody(t, resp)
	recommendations, ok := recommendationsBody["RecommendationSummaries"].([]any)
	if !ok || len(recommendations) == 0 {
		t.Fatalf("expected RecommendationSummaries, got %#v", recommendationsBody["RecommendationSummaries"])
	}
	recommendation, _ := recommendations[0].(map[string]any)
	ruleMetadata, _ := recommendation["RuleMetadata"].(map[string]any)
	if _, ok := ruleMetadata["ShortDescription"].(string); !ok {
		t.Fatalf("expected RuleMetadata.ShortDescription string, got %#v", ruleMetadata["ShortDescription"])
	}

	resp = codeGuruRequest(t, ts, http.MethodPut, "/feedback", `{"CodeReviewArn":"`+reviewArn+`","RecommendationId":"rec-000001","Reactions":["ThumbsDown"]}`)
	assertStatus(t, resp, http.StatusOK)
	if body := decodeShard9JSONBody(t, resp); len(body) != 0 {
		t.Fatalf("expected PutRecommendationFeedback empty JSON body, got %#v", body)
	}

	resp = codeGuruRequest(t, ts, http.MethodPost, "/tags/"+url.PathEscape(associationArn), `{"Tags":{"env":"shard9"}}`)
	assertStatus(t, resp, http.StatusOK)
	if body := decodeShard9JSONBody(t, resp); len(body) != 0 {
		t.Fatalf("expected TagResource empty JSON body, got %#v", body)
	}

	resp = codeGuruRequest(t, ts, http.MethodDelete, "/tags/"+url.PathEscape(associationArn), `{"TagKeys":["env"]}`)
	assertStatus(t, resp, http.StatusOK)
	if body := decodeShard9JSONBody(t, resp); len(body) != 0 {
		t.Fatalf("expected UntagResource empty JSON body, got %#v", body)
	}
}

func TestCognitoUserPoolsShard9DomainTermsRiskAndLogShapes(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	userPoolID, clientID := cognitoUserPoolsMustCreatePoolAndClient(t, ts, "stackyard-shard9-pool", "stackyard-shard9-client")

	resp := cognitoUserPoolsRequestPayload(t, ts, "CreateUserPoolDomain", map[string]any{
		"UserPoolId": userPoolID,
		"Domain":     "stackyard-shard9-domain",
	})
	assertStatus(t, resp, http.StatusOK)
	createDomainBody := decodeCognitoUserPoolsBody(t, resp)
	if got, _ := createDomainBody["CloudFrontDomain"].(string); strings.TrimSpace(got) == "" {
		t.Fatalf("expected CloudFrontDomain in CreateUserPoolDomain response, got %#v", createDomainBody["CloudFrontDomain"])
	}
	if got, _ := createDomainBody["ManagedLoginVersion"].(json.Number); got.String() != "1" {
		t.Fatalf("expected ManagedLoginVersion 1 in CreateUserPoolDomain response, got %#v", createDomainBody["ManagedLoginVersion"])
	}

	resp = cognitoUserPoolsRequestPayload(t, ts, "DescribeUserPoolDomain", map[string]any{"Domain": "stackyard-shard9-domain"})
	assertStatus(t, resp, http.StatusOK)
	describeDomainBody := decodeCognitoUserPoolsBody(t, resp)
	domainDescription, _ := describeDomainBody["DomainDescription"].(map[string]any)
	if got, _ := domainDescription["Version"].(string); got != "1" {
		t.Fatalf("expected DescribeUserPoolDomain.Version string, got %#v", domainDescription["Version"])
	}

	resp = cognitoUserPoolsRequestPayload(t, ts, "SetRiskConfiguration", map[string]any{
		"UserPoolId": userPoolID,
		"ClientId":   clientID,
		"CompromisedCredentialsRiskConfiguration": map[string]any{
			"Actions": map[string]any{"EventAction": "BLOCK"},
		},
		"AccountTakeoverRiskConfiguration": map[string]any{
			"NotifyConfiguration": map[string]any{
				"SourceArn":     "arn:aws:ses:us-east-1:123456789012:identity/no-reply@example.com",
				"From":          "no-reply@example.com",
				"ReplyTo":       "security@example.com",
				"BlockEmail":    map[string]any{"Subject": "blocked", "HtmlBody": "<p>blocked</p>", "TextBody": "blocked"},
				"NoActionEmail": map[string]any{"Subject": "allowed", "HtmlBody": "<p>allowed</p>", "TextBody": "allowed"},
				"MfaEmail":      map[string]any{"Subject": "mfa", "HtmlBody": "<p>mfa</p>", "TextBody": "mfa"},
			},
			"Actions": map[string]any{
				"LowAction":    map[string]any{"EventAction": "NO_ACTION"},
				"MediumAction": map[string]any{"EventAction": "MFA_IF_CONFIGURED"},
				"HighAction":   map[string]any{"EventAction": "BLOCK"},
			},
		},
	})
	assertStatus(t, resp, http.StatusOK)

	resp = cognitoUserPoolsRequestPayload(t, ts, "DescribeRiskConfiguration", map[string]any{
		"UserPoolId": userPoolID,
		"ClientId":   clientID,
	})
	assertStatus(t, resp, http.StatusOK)
	riskBody := decodeCognitoUserPoolsBody(t, resp)
	riskConfig, _ := riskBody["RiskConfiguration"].(map[string]any)
	accountTakeover, _ := riskConfig["AccountTakeoverRiskConfiguration"].(map[string]any)
	actions, _ := accountTakeover["Actions"].(map[string]any)
	highAction, _ := actions["HighAction"].(map[string]any)
	if got, _ := highAction["EventAction"].(string); got != "BLOCK" {
		t.Fatalf("expected DescribeRiskConfiguration high action BLOCK, got %#v", highAction["EventAction"])
	}

	resp = cognitoUserPoolsRequestPayload(t, ts, "SetLogDeliveryConfiguration", map[string]any{
		"UserPoolId": userPoolID,
		"LogConfigurations": []map[string]any{{
			"LogLevel":    "ERROR",
			"EventSource": "userAuthEvents",
		}},
	})
	assertStatus(t, resp, http.StatusOK)
	setLogBody := decodeCognitoUserPoolsBody(t, resp)
	setLogConfig, _ := setLogBody["LogDeliveryConfiguration"].(map[string]any)
	if _, ok := setLogConfig["LogConfigurations"].([]any); !ok {
		t.Fatalf("expected wrapped LogConfigurations in SetLogDeliveryConfiguration response, got %#v", setLogConfig["LogConfigurations"])
	}

	resp = cognitoUserPoolsRequestPayload(t, ts, "CreateTerms", map[string]any{
		"UserPoolId":  userPoolID,
		"ClientId":    clientID,
		"TermsName":   "privacy-policy",
		"Enforcement": "NONE",
		"TermsSource": "LINK",
		"Links": []map[string]any{{
			"Text": "privacy",
			"Type": "privacy-policy",
			"Url":  "https://example.com/privacy",
		}},
	})
	assertStatus(t, resp, http.StatusOK)
	termsBody := decodeCognitoUserPoolsBody(t, resp)
	terms, _ := termsBody["Terms"].(map[string]any)
	if got, _ := terms["TermsSource"].(string); got != "LINK" {
		t.Fatalf("expected top-level TermsSource LINK, got %#v", terms["TermsSource"])
	}
	links, ok := terms["Links"].([]any)
	if !ok || len(links) != 1 {
		t.Fatalf("expected top-level Links in Terms response, got %#v", terms["Links"])
	}
}

func TestEventBridgeShard9MutationShapes(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := eventBridgeRequest(t, ts, "CreateEventBus", []byte(`{"Name":"stage9-bus-a","Description":"primary"}`))
	assertStatus(t, resp, http.StatusOK)
	createBusABody := decodeShard9JSONBody(t, resp)
	busAArn, _ := createBusABody["EventBusArn"].(string)

	resp = eventBridgeRequest(t, ts, "CreateEventBus", []byte(`{"Name":"stage9-bus-b","Description":"secondary"}`))
	assertStatus(t, resp, http.StatusOK)
	createBusBBody := decodeShard9JSONBody(t, resp)
	busBArn, _ := createBusBBody["EventBusArn"].(string)

	resp = eventBridgeRequest(t, ts, "ListEventBuses", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	listBusesBody := decodeShard9JSONBody(t, resp)
	eventBuses, _ := listBusesBody["EventBuses"].([]any)
	foundA := false
	for _, item := range eventBuses {
		entry, _ := item.(map[string]any)
		name, _ := entry["Name"].(string)
		if name == "stage9-bus-a" {
			foundA = true
			if _, ok := entry["Description"]; ok {
				t.Fatalf("expected ListEventBuses entry to omit Description, got %#v", entry["Description"])
			}
		}
	}
	if !foundA {
		t.Fatalf("expected stage9-bus-a in ListEventBuses response")
	}

	resp = eventBridgeRequest(t, ts, "CreateEndpoint", []byte(`{"Name":"stage9-endpoint","Description":"demo","EventBuses":[{"EventBusArn":"`+busAArn+`"},{"EventBusArn":"`+busBArn+`"}]}`))
	assertStatus(t, resp, http.StatusOK)
	createEndpointBody := decodeShard9JSONBody(t, resp)
	if got, _ := createEndpointBody["Arn"].(string); strings.TrimSpace(got) == "" {
		t.Fatalf("expected Arn in CreateEndpoint response, got %#v", createEndpointBody["Arn"])
	}
	if got, _ := createEndpointBody["RoleArn"].(string); strings.TrimSpace(got) == "" {
		t.Fatalf("expected RoleArn in CreateEndpoint response, got %#v", createEndpointBody["RoleArn"])
	}
	if buses, _ := createEndpointBody["EventBuses"].([]any); len(buses) != 2 {
		t.Fatalf("expected two EventBuses in CreateEndpoint response, got %#v", createEndpointBody["EventBuses"])
	}

	resp = eventBridgeRequest(t, ts, "UpdateEndpoint", []byte(`{"Name":"stage9-endpoint","Description":"updated"}`))
	assertStatus(t, resp, http.StatusOK)
	updateEndpointBody := decodeShard9JSONBody(t, resp)
	if got, _ := updateEndpointBody["EndpointId"].(string); got != "stage9-endpoint" {
		t.Fatalf("expected EndpointId in UpdateEndpoint response, got %#v", updateEndpointBody["EndpointId"])
	}
	if got, _ := updateEndpointBody["EndpointUrl"].(string); strings.TrimSpace(got) == "" {
		t.Fatalf("expected EndpointUrl in UpdateEndpoint response, got %#v", updateEndpointBody["EndpointUrl"])
	}

	resp = eventBridgeRequest(t, ts, "CreateConnection", []byte(`{"Name":"stage9-connection","AuthorizationType":"API_KEY","Description":"demo connection"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = eventBridgeRequest(t, ts, "DeauthorizeConnection", []byte(`{"Name":"stage9-connection"}`))
	assertStatus(t, resp, http.StatusOK)
	deauthorizeBody := decodeShard9JSONBody(t, resp)
	if got, _ := deauthorizeBody["ConnectionArn"].(string); strings.TrimSpace(got) == "" {
		t.Fatalf("expected ConnectionArn in DeauthorizeConnection response, got %#v", deauthorizeBody["ConnectionArn"])
	}
	if _, ok := deauthorizeBody["ConnectionState"].(string); !ok {
		t.Fatalf("expected ConnectionState in DeauthorizeConnection response, got %#v", deauthorizeBody["ConnectionState"])
	}

	resp = eventBridgeRequest(t, ts, "DeleteConnection", []byte(`{"Name":"stage9-connection"}`))
	assertStatus(t, resp, http.StatusOK)
	deleteConnectionBody := decodeShard9JSONBody(t, resp)
	if got, _ := deleteConnectionBody["ConnectionArn"].(string); strings.TrimSpace(got) == "" {
		t.Fatalf("expected ConnectionArn in DeleteConnection response, got %#v", deleteConnectionBody["ConnectionArn"])
	}
	if _, ok := deleteConnectionBody["LastModifiedTime"]; !ok {
		t.Fatalf("expected LastModifiedTime in DeleteConnection response, got %#v", deleteConnectionBody)
	}
}

func TestFlinkShard9MutationAndSnapshotShapes(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := flinkRequest(t, ts, "CreateApplication", `{"ApplicationName":"stage9-flink","RuntimeEnvironment":"FLINK-1_18"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = flinkRequest(t, ts, "StartApplication", `{"ApplicationName":"stage9-flink"}`)
	assertStatus(t, resp, http.StatusOK)
	startBody := decodeShard9JSONBody(t, resp)
	if _, ok := startBody["OperationId"]; ok {
		t.Fatalf("expected StartApplication to omit OperationId, got %#v", startBody["OperationId"])
	}

	resp = flinkRequest(t, ts, "UpdateApplication", `{"ApplicationName":"stage9-flink"}`)
	assertStatus(t, resp, http.StatusOK)
	updateBody := decodeShard9JSONBody(t, resp)
	updateDetail, _ := updateBody["ApplicationDetail"].(map[string]any)
	if got, _ := updateDetail["ApplicationARN"].(string); strings.TrimSpace(got) == "" {
		t.Fatalf("expected ApplicationARN in UpdateApplication response, got %#v", updateDetail["ApplicationARN"])
	}
	if _, ok := updateBody["OperationId"]; ok {
		t.Fatalf("expected UpdateApplication to omit OperationId, got %#v", updateBody["OperationId"])
	}

	resp = flinkRequest(t, ts, "UpdateApplicationMaintenanceConfiguration", `{"ApplicationName":"stage9-flink"}`)
	assertStatus(t, resp, http.StatusOK)
	maintenanceBody := decodeShard9JSONBody(t, resp)
	if got, _ := maintenanceBody["ApplicationARN"].(string); strings.TrimSpace(got) == "" {
		t.Fatalf("expected ApplicationARN in UpdateApplicationMaintenanceConfiguration response, got %#v", maintenanceBody["ApplicationARN"])
	}

	resp = flinkRequest(t, ts, "CreateApplicationSnapshot", `{"ApplicationName":"stage9-flink","SnapshotName":"stage9-snapshot"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = flinkRequest(t, ts, "DescribeApplicationSnapshot", `{"ApplicationName":"stage9-flink","SnapshotName":"stage9-snapshot"}`)
	assertStatus(t, resp, http.StatusOK)
	describeSnapshotBody := decodeShard9JSONBody(t, resp)
	snapshotDetails, _ := describeSnapshotBody["SnapshotDetails"].(map[string]any)
	if got, _ := snapshotDetails["ApplicationVersionId"].(json.Number); got.String() == "" {
		t.Fatalf("expected ApplicationVersionId in DescribeApplicationSnapshot response, got %#v", snapshotDetails["ApplicationVersionId"])
	}
	if _, ok := snapshotDetails["RuntimeEnvironment"]; ok {
		t.Fatalf("expected DescribeApplicationSnapshot to omit RuntimeEnvironment, got %#v", snapshotDetails["RuntimeEnvironment"])
	}

	resp = flinkRequest(t, ts, "ListApplicationSnapshots", `{"ApplicationName":"stage9-flink"}`)
	assertStatus(t, resp, http.StatusOK)
	listSnapshotsBody := decodeShard9JSONBody(t, resp)
	snapshots, _ := listSnapshotsBody["SnapshotSummaries"].([]any)
	if len(snapshots) == 0 {
		t.Fatalf("expected SnapshotSummaries, got %#v", listSnapshotsBody["SnapshotSummaries"])
	}
	listedSnapshot, _ := snapshots[0].(map[string]any)
	if got, _ := listedSnapshot["ApplicationVersionId"].(json.Number); got.String() == "" {
		t.Fatalf("expected ApplicationVersionId in ListApplicationSnapshots response, got %#v", listedSnapshot["ApplicationVersionId"])
	}
	if _, ok := listedSnapshot["RuntimeEnvironment"]; ok {
		t.Fatalf("expected ListApplicationSnapshots to omit RuntimeEnvironment, got %#v", listedSnapshot["RuntimeEnvironment"])
	}

	resp = flinkRequest(t, ts, "DescribeApplicationVersion", `{"ApplicationName":"stage9-flink"}`)
	assertStatus(t, resp, http.StatusOK)
	describeVersionBody := decodeShard9JSONBody(t, resp)
	versionDetail, _ := describeVersionBody["ApplicationVersionDetail"].(map[string]any)
	if got, _ := versionDetail["ApplicationARN"].(string); strings.TrimSpace(got) == "" {
		t.Fatalf("expected ApplicationARN in DescribeApplicationVersion response, got %#v", versionDetail["ApplicationARN"])
	}
	if got, _ := versionDetail["RuntimeEnvironment"].(string); strings.TrimSpace(got) == "" {
		t.Fatalf("expected RuntimeEnvironment in DescribeApplicationVersion response, got %#v", versionDetail["RuntimeEnvironment"])
	}
}

func TestRDSShard9DirectRootAndCompatShapes(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	clusterID := "rds-shard9-cluster"
	instanceID := "rds-shard9-db"
	clusterARN := "arn:aws:rds:us-east-1:123456789012:cluster:" + clusterID
	instanceARN := "arn:aws:rds:us-east-1:123456789012:db:" + instanceID

	status, body := rdsRequest(t, ts, url.Values{
		"Action":               []string{"CreateDBInstance"},
		"DBInstanceIdentifier": []string{instanceID},
		"Engine":               []string{"mysql"},
		"DBInstanceClass":      []string{"db.t3.micro"},
		"AllocatedStorage":     []string{"20"},
		"MasterUsername":       []string{"admin"},
		"MasterUserPassword":   []string{"Secret1234"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected CreateDBInstance 200, got %d: %s", status, string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":              []string{"CreateDBCluster"},
		"DBClusterIdentifier": []string{clusterID},
		"Engine":              []string{"aurora-mysql"},
		"MasterUsername":      []string{"admin"},
		"MasterUserPassword":  []string{"Secret1234"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected CreateDBCluster 200, got %d: %s", status, string(body))
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":               []string{"StartExportTask"},
		"ExportTaskIdentifier": []string{"rds-shard9-export"},
		"SourceArn":            []string{"arn:aws:rds:us-east-1:123456789012:snapshot:rds-shard9-snap"},
		"S3BucketName":         []string{"demo-export-bucket"},
		"S3Prefix":             []string{"exports/shard9"},
		"KmsKeyId":             []string{"alias/aws/rds"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected StartExportTask 200, got %d: %s", status, string(body))
	}
	var startExport struct {
		Result struct {
			ExportTaskIdentifier string `xml:"ExportTaskIdentifier"`
			SourceArn            string `xml:"SourceArn"`
		} `xml:"StartExportTaskResult"`
	}
	if err := xml.Unmarshal(body, &startExport); err != nil {
		t.Fatalf("unmarshal StartExportTask response: %v", err)
	}
	if startExport.Result.ExportTaskIdentifier != "rds-shard9-export" {
		t.Fatalf("expected direct-root ExportTaskIdentifier, got %#v", startExport.Result.ExportTaskIdentifier)
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":               []string{"CancelExportTask"},
		"ExportTaskIdentifier": []string{"rds-shard9-export"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected CancelExportTask 200, got %d: %s", status, string(body))
	}
	var cancelExport struct {
		Result struct {
			Status string `xml:"Status"`
		} `xml:"CancelExportTaskResult"`
	}
	if err := xml.Unmarshal(body, &cancelExport); err != nil {
		t.Fatalf("unmarshal CancelExportTask response: %v", err)
	}
	if cancelExport.Result.Status != "canceled" {
		t.Fatalf("expected direct-root canceled export status, got %#v", cancelExport.Result.Status)
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":                []string{"CreateIntegration"},
		"IntegrationIdentifier": []string{"rds-shard9-int"},
		"IntegrationName":       []string{"shard9"},
		"SourceArn":             []string{instanceARN},
		"TargetArn":             []string{clusterARN},
	})
	if status != http.StatusOK {
		t.Fatalf("expected CreateIntegration 200, got %d: %s", status, string(body))
	}
	var createIntegration struct {
		Result struct {
			IntegrationName string `xml:"IntegrationName"`
			IntegrationArn  string `xml:"IntegrationArn"`
		} `xml:"CreateIntegrationResult"`
	}
	if err := xml.Unmarshal(body, &createIntegration); err != nil {
		t.Fatalf("unmarshal CreateIntegration response: %v", err)
	}
	if createIntegration.Result.IntegrationName != "shard9" || strings.TrimSpace(createIntegration.Result.IntegrationArn) == "" {
		t.Fatalf("expected direct-root integration fields, got %+v", createIntegration.Result)
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":                []string{"DeleteIntegration"},
		"IntegrationIdentifier": []string{"rds-shard9-int"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected DeleteIntegration 200, got %d: %s", status, string(body))
	}
	var deleteIntegration struct {
		Result struct {
			IntegrationArn string `xml:"IntegrationArn"`
		} `xml:"DeleteIntegrationResult"`
	}
	if err := xml.Unmarshal(body, &deleteIntegration); err != nil {
		t.Fatalf("unmarshal DeleteIntegration response: %v", err)
	}
	if strings.TrimSpace(deleteIntegration.Result.IntegrationArn) == "" {
		t.Fatalf("expected direct-root IntegrationArn in DeleteIntegration response")
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":      []string{"StartActivityStream"},
		"ResourceArn": []string{clusterARN},
		"Mode":        []string{"async"},
		"KmsKeyId":    []string{"alias/aws/rds"},
	})
	if status != http.StatusOK {
		t.Fatalf("expected StartActivityStream 200, got %d: %s", status, string(body))
	}
	var startActivity struct {
		Result rdsStartActivityStreamResultXML `xml:"StartActivityStreamResult"`
	}
	if err := xml.Unmarshal(body, &startActivity); err != nil {
		t.Fatalf("unmarshal StartActivityStream response: %v", err)
	}
	if startActivity.Result.KinesisStreamName == "" || startActivity.Result.Status == "" {
		t.Fatalf("expected modeled StartActivityStream response, got %+v", startActivity.Result)
	}

	status, body = rdsRequest(t, ts, url.Values{"Action": []string{"DescribeAccountAttributes"}})
	if status != http.StatusOK {
		t.Fatalf("expected DescribeAccountAttributes 200, got %d: %s", status, string(body))
	}
	bodyString := string(body)
	if !strings.Contains(bodyString, "<AccountQuotas>") || !strings.Contains(bodyString, "<Used>") {
		t.Fatalf("expected AccountQuotas with Used member in DescribeAccountAttributes response, got %s", bodyString)
	}

	status, body = rdsRequest(t, ts, url.Values{"Action": []string{"DescribeCertificates"}})
	if status != http.StatusOK {
		t.Fatalf("expected DescribeCertificates 200, got %d: %s", status, string(body))
	}
	var describeCertificates struct {
		Result struct {
			DefaultCertificateForNewLaunches string `xml:"DefaultCertificateForNewLaunches"`
		} `xml:"DescribeCertificatesResult"`
	}
	if err := xml.Unmarshal(body, &describeCertificates); err != nil {
		t.Fatalf("unmarshal DescribeCertificates response: %v", err)
	}
	if strings.TrimSpace(describeCertificates.Result.DefaultCertificateForNewLaunches) == "" {
		t.Fatalf("expected DefaultCertificateForNewLaunches in DescribeCertificates response")
	}

	status, body = rdsRequest(t, ts, url.Values{
		"Action":      []string{"EnableHttpEndpoint"},
		"ResourceArn": []string{clusterARN},
	})
	if status != http.StatusOK {
		t.Fatalf("expected EnableHttpEndpoint 200, got %d: %s", status, string(body))
	}
	var enableHTTPEndpoint struct {
		Result rdsHTTPEndpointResultXML `xml:"EnableHttpEndpointResult"`
	}
	if err := xml.Unmarshal(body, &enableHTTPEndpoint); err != nil {
		t.Fatalf("unmarshal EnableHttpEndpoint response: %v", err)
	}
	if !enableHTTPEndpoint.Result.HttpEndpointEnabled {
		t.Fatalf("expected EnableHttpEndpoint to return HttpEndpointEnabled=true")
	}

	for _, params := range []url.Values{
		{"Action": []string{"DescribeEngineDefaultParameters"}, "DBParameterGroupFamily": []string{"mysql8.0"}},
		{"Action": []string{"DescribeEventCategories"}, "SourceType": []string{"db-instance"}},
		{"Action": []string{"DescribeOptionGroupOptions"}, "EngineName": []string{"mysql"}, "MajorEngineVersion": []string{"8.0"}},
		{"Action": []string{"RemoveFromGlobalCluster"}, "GlobalClusterIdentifier": []string{"shard9-global"}, "DbClusterIdentifier": []string{clusterID}},
	} {
		status, body = rdsRequest(t, ts, params)
		if status != http.StatusOK {
			t.Fatalf("expected %s 200, got %d: %s", params.Get("Action"), status, string(body))
		}
		bodyString = string(body)
		switch params.Get("Action") {
		case "DescribeEngineDefaultParameters":
			if !strings.Contains(bodyString, "<EngineDefaults>") {
				t.Fatalf("expected EngineDefaults in %s response, got %s", params.Get("Action"), bodyString)
			}
		case "DescribeEventCategories":
			if !strings.Contains(bodyString, "<EventCategoriesMapList>") {
				t.Fatalf("expected EventCategoriesMapList in %s response, got %s", params.Get("Action"), bodyString)
			}
		case "DescribeOptionGroupOptions":
			if !strings.Contains(bodyString, "<OptionGroupOptions>") {
				t.Fatalf("expected OptionGroupOptions in %s response, got %s", params.Get("Action"), bodyString)
			}
		case "RemoveFromGlobalCluster":
			if !strings.Contains(bodyString, "<GlobalCluster>") {
				t.Fatalf("expected GlobalCluster in RemoveFromGlobalCluster response, got %s", bodyString)
			}
		}
	}
}

func TestSESShard9DescribeConfigurationSetOmitsEmptyKinesisFirehoseDestination(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := sesRequest(t, ts, url.Values{
		"Action":                []string{"CreateConfigurationSet"},
		"ConfigurationSet.Name": []string{"shard9-config"},
	})
	assertStatus(t, resp, http.StatusOK)

	resp = sesRequest(t, ts, url.Values{
		"Action":                   []string{"CreateConfigurationSetEventDestination"},
		"ConfigurationSetName":     []string{"shard9-config"},
		"EventDestination.Name":    []string{"sns-only"},
		"EventDestination.Enabled": []string{"true"},
		"EventDestination.MatchingEventTypes.member.1": []string{"send"},
		"EventDestination.SNSDestination.TopicARN":     []string{"arn:aws:sns:us-east-1:123456789012:ses-events"},
	})
	assertStatus(t, resp, http.StatusOK)

	resp = sesRequest(t, ts, url.Values{
		"Action":               []string{"DescribeConfigurationSet"},
		"ConfigurationSetName": []string{"shard9-config"},
	})
	assertStatus(t, resp, http.StatusOK)
	var describeConfigSet struct {
		Result sesDescribeConfigurationSetResult `xml:"DescribeConfigurationSetResult"`
	}
	if err := xml.Unmarshal(mustBody(t, resp), &describeConfigSet); err != nil {
		t.Fatalf("unmarshal DescribeConfigurationSet response: %v", err)
	}
	if len(describeConfigSet.Result.EventDestinations) != 1 {
		t.Fatalf("expected one event destination, got %#v", describeConfigSet.Result.EventDestinations)
	}
	destination := describeConfigSet.Result.EventDestinations[0]
	if destination.SNSDestination == nil || strings.TrimSpace(destination.SNSDestination.TopicARN) == "" {
		t.Fatalf("expected SNSDestination TopicARN, got %#v", destination.SNSDestination)
	}
	if destination.KinesisFirehoseDestination != nil {
		t.Fatalf("expected KinesisFirehoseDestination to be omitted for SNS-only destination, got %#v", destination.KinesisFirehoseDestination)
	}
}
