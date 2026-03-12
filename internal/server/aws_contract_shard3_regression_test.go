package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBatchShard3DescribeJobsIncludesStartedAt(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := batchRequest(t, ts, http.MethodPost, "/v1/createcomputeenvironment", []byte(`{"computeEnvironmentName":"shard3-ce","type":"UNMANAGED","state":"ENABLED","unmanagedvCpus":16}`))
	assertStatus(t, resp, http.StatusOK)

	resp = batchRequest(t, ts, http.MethodPost, "/v1/createjobqueue", []byte(`{"jobQueueName":"shard3-queue","state":"ENABLED","priority":1,"computeEnvironmentOrder":[{"order":1,"computeEnvironment":"shard3-ce"}]}`))
	assertStatus(t, resp, http.StatusOK)

	resp = batchRequest(t, ts, http.MethodPost, "/v1/registerjobdefinition", []byte(`{"jobDefinitionName":"shard3-jobdef","type":"container"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = batchRequest(t, ts, http.MethodPost, "/v1/submitjob", []byte(`{"jobName":"shard3-job","jobQueue":"shard3-queue","jobDefinition":"shard3-jobdef"}`))
	assertStatus(t, resp, http.StatusOK)
	var submitOut struct {
		JobID string `json:"jobId"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &submitOut); err != nil {
		t.Fatalf("unmarshal submit job response: %v", err)
	}
	if submitOut.JobID == "" {
		t.Fatalf("expected submitted job id")
	}

	resp = batchRequest(t, ts, http.MethodPost, "/v1/describejobs", []byte(`{"jobs":["`+submitOut.JobID+`"]}`))
	assertStatus(t, resp, http.StatusOK)
	var describeOut struct {
		Jobs []struct {
			JobID     string `json:"jobId"`
			StartedAt int64  `json:"startedAt"`
		} `json:"jobs"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &describeOut); err != nil {
		t.Fatalf("unmarshal describe jobs response: %v", err)
	}
	if len(describeOut.Jobs) != 1 || describeOut.Jobs[0].StartedAt == 0 {
		t.Fatalf("expected describe jobs to include startedAt, got %+v", describeOut.Jobs)
	}
}

func TestDynamoDBShard3ShapesIncludeConsumedCapacityAndGlobalTableSettings(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := dynamodbRequest(t, ts, "CreateTable", []byte(`{
		"TableName":"shard3-table",
		"AttributeDefinitions":[{"AttributeName":"pk","AttributeType":"S"}],
		"KeySchema":[{"AttributeName":"pk","KeyType":"HASH"}],
		"BillingMode":"PAY_PER_REQUEST"
	}`))
	assertStatus(t, resp, http.StatusOK)

	type consumedCapacity struct {
		TableName string `json:"TableName"`
	}

	resp = dynamodbRequest(t, ts, "PutItem", []byte(`{
		"TableName":"shard3-table",
		"Item":{"pk":{"S":"k1"},"status":{"S":"NEW"}},
		"ReturnConsumedCapacity":"TOTAL"
	}`))
	assertStatus(t, resp, http.StatusOK)
	var putOut struct {
		ConsumedCapacity consumedCapacity `json:"ConsumedCapacity"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &putOut); err != nil {
		t.Fatalf("unmarshal put item response: %v", err)
	}
	if putOut.ConsumedCapacity.TableName != "shard3-table" {
		t.Fatalf("expected put item consumed capacity table name, got %+v", putOut.ConsumedCapacity)
	}

	resp = dynamodbRequest(t, ts, "Query", []byte(`{
		"TableName":"shard3-table",
		"KeyConditionExpression":"pk = :pk",
		"ExpressionAttributeValues":{":pk":{"S":"k1"}},
		"ReturnConsumedCapacity":"TOTAL"
	}`))
	assertStatus(t, resp, http.StatusOK)
	var queryOut struct {
		ConsumedCapacity consumedCapacity `json:"ConsumedCapacity"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &queryOut); err != nil {
		t.Fatalf("unmarshal query response: %v", err)
	}
	if queryOut.ConsumedCapacity.TableName != "shard3-table" {
		t.Fatalf("expected query consumed capacity table name, got %+v", queryOut.ConsumedCapacity)
	}

	resp = dynamodbRequest(t, ts, "TransactWriteItems", []byte(`{
		"TransactItems":[{"Put":{"TableName":"shard3-table","Item":{"pk":{"S":"k2"}}}}],
		"ReturnConsumedCapacity":"TOTAL"
	}`))
	assertStatus(t, resp, http.StatusOK)
	var txOut struct {
		ConsumedCapacity []consumedCapacity `json:"ConsumedCapacity"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &txOut); err != nil {
		t.Fatalf("unmarshal transact write items response: %v", err)
	}
	if len(txOut.ConsumedCapacity) == 0 || txOut.ConsumedCapacity[0].TableName != "shard3-table" {
		t.Fatalf("expected transact write consumed capacity, got %+v", txOut.ConsumedCapacity)
	}

	resp = dynamodbRequest(t, ts, "CreateGlobalTable", []byte(`{
		"GlobalTableName":"shard3-global",
		"ReplicationGroup":[{"RegionName":"us-east-1"}]
	}`))
	assertStatus(t, resp, http.StatusOK)

	resp = dynamodbRequest(t, ts, "DescribeGlobalTableSettings", []byte(`{"GlobalTableName":"shard3-global"}`))
	assertStatus(t, resp, http.StatusOK)
	var describeSettingsOut struct {
		GlobalTableName string `json:"GlobalTableName"`
		ReplicaSettings []struct {
			RegionName string `json:"RegionName"`
		} `json:"ReplicaSettings"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &describeSettingsOut); err != nil {
		t.Fatalf("unmarshal describe global table settings response: %v", err)
	}
	if describeSettingsOut.GlobalTableName != "shard3-global" || len(describeSettingsOut.ReplicaSettings) == 0 {
		t.Fatalf("expected top-level global table settings fields, got %+v", describeSettingsOut)
	}

	resp = dynamodbRequest(t, ts, "UpdateGlobalTableSettings", []byte(`{
		"GlobalTableName":"shard3-global",
		"ReplicaSettingsUpdate":[{"RegionName":"us-east-1"},{"RegionName":"us-west-2"}]
	}`))
	assertStatus(t, resp, http.StatusOK)
	var updateSettingsOut struct {
		GlobalTableName string `json:"GlobalTableName"`
		ReplicaSettings []struct {
			RegionName string `json:"RegionName"`
		} `json:"ReplicaSettings"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &updateSettingsOut); err != nil {
		t.Fatalf("unmarshal update global table settings response: %v", err)
	}
	if updateSettingsOut.GlobalTableName != "shard3-global" || len(updateSettingsOut.ReplicaSettings) == 0 {
		t.Fatalf("expected update global table settings top-level fields, got %+v", updateSettingsOut)
	}
}

func TestElasticLoadBalancingShard3TrustStoreLocationMembers(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := elasticLoadBalancingRequest(t, ts, "GetTrustStoreCaCertificatesBundle", nil)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "<Location>https://stackyard.local/truststores/stackyard-ca-bundle.pem</Location>") {
		t.Fatalf("expected bundle location member, got %q", body)
	}

	resp = elasticLoadBalancingRequest(t, ts, "GetTrustStoreRevocationContent", nil)
	assertStatus(t, resp, http.StatusOK)
	body = string(mustBody(t, resp))
	if !strings.Contains(body, "<Location>https://stackyard.local/truststores/stackyard-revocations.crl</Location>") {
		t.Fatalf("expected revocation location member, got %q", body)
	}
}

func TestRoute53Shard3CreateAndListShapes(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	createHostedZone := []byte(`<CreateHostedZoneRequest xmlns="https://route53.amazonaws.com/doc/2013-04-01/"><Name>stackyard.example.com</Name><CallerReference>shard3</CallerReference></CreateHostedZoneRequest>`)
	resp := route53Request(t, ts, http.MethodPost, "/2013-04-01/hostedzone", "CreateHostedZone", createHostedZone, map[string]string{
		"Content-Type": "application/xml",
	})
	assertStatus(t, resp, http.StatusOK)
	if !strings.Contains(resp.Header.Get("Location"), "/2013-04-01/hostedzone/") {
		t.Fatalf("expected hosted zone Location header, got %q", resp.Header.Get("Location"))
	}
	body := string(mustBody(t, resp))
	for _, fragment := range []string{"<HostedZone>", "<ChangeInfo>", "<DelegationSet>"} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("expected create hosted zone body to include %s, got %q", fragment, body)
		}
	}

	resp = route53Request(t, ts, http.MethodPost, "/2013-04-01/hostedzone", "ListHostedZones", nil, nil)
	assertStatus(t, resp, http.StatusOK)
	body = string(mustBody(t, resp))
	for _, fragment := range []string{"<Marker>", "<IsTruncated>false</IsTruncated>", "<MaxItems>100</MaxItems>"} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("expected list hosted zones body to include %s, got %q", fragment, body)
		}
	}

	createHealthCheck := []byte(`<CreateHealthCheckRequest xmlns="https://route53.amazonaws.com/doc/2013-04-01/"><CallerReference>shard3-hc</CallerReference><HealthCheckConfig><IPAddress>127.0.0.1</IPAddress><Port>443</Port><Type>HTTPS</Type><ResourcePath>/health</ResourcePath><RequestInterval>30</RequestInterval><FailureThreshold>3</FailureThreshold></HealthCheckConfig></CreateHealthCheckRequest>`)
	resp = route53Request(t, ts, http.MethodPost, "/2013-04-01/healthcheck", "CreateHealthCheck", createHealthCheck, map[string]string{
		"Content-Type": "application/xml",
	})
	assertStatus(t, resp, http.StatusOK)
	if !strings.Contains(resp.Header.Get("Location"), "/2013-04-01/healthcheck/") {
		t.Fatalf("expected health check Location header, got %q", resp.Header.Get("Location"))
	}
}

func TestSWFShard3DescribeAndPollShapes(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := swfRequest(t, ts, "SimpleWorkflowService.RegisterDomain", []byte(`{"name":"shard3-domain","workflowExecutionRetentionPeriodInDays":"7"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = swfRequest(t, ts, "SimpleWorkflowService.RegisterActivityType", []byte(`{"domain":"shard3-domain","name":"shard3-activity","version":"1"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = swfRequest(t, ts, "SimpleWorkflowService.RegisterWorkflowType", []byte(`{"domain":"shard3-domain","name":"shard3-workflow","version":"1"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = swfRequest(t, ts, "SimpleWorkflowService.StartWorkflowExecution", []byte(`{"domain":"shard3-domain","workflowId":"wf-shard3","workflowType":{"name":"shard3-workflow","version":"1"},"taskList":{"name":"main"}}`))
	assertStatus(t, resp, http.StatusOK)
	var startOut struct {
		RunID string `json:"runId"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &startOut); err != nil {
		t.Fatalf("unmarshal start workflow response: %v", err)
	}
	if startOut.RunID == "" {
		t.Fatalf("expected run id")
	}

	resp = swfRequest(t, ts, "SimpleWorkflowService.DescribeActivityType", []byte(`{"domain":"shard3-domain","activityType":{"name":"shard3-activity","version":"1"}}`))
	assertStatus(t, resp, http.StatusOK)
	var describeActivityOut struct {
		TypeInfo struct {
			CreationDate float64 `json:"creationDate"`
		} `json:"typeInfo"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &describeActivityOut); err != nil {
		t.Fatalf("unmarshal describe activity type response: %v", err)
	}
	if describeActivityOut.TypeInfo.CreationDate == 0 {
		t.Fatalf("expected activity type creationDate, got %+v", describeActivityOut.TypeInfo)
	}

	resp = swfRequest(t, ts, "SimpleWorkflowService.DescribeWorkflowExecution", []byte(`{"domain":"shard3-domain","execution":{"workflowId":"wf-shard3","runId":"`+startOut.RunID+`"}}`))
	assertStatus(t, resp, http.StatusOK)
	var describeWorkflowExecutionOut struct {
		ExecutionConfiguration struct {
			ChildPolicy                  string `json:"childPolicy"`
			ExecutionStartToCloseTimeout string `json:"executionStartToCloseTimeout"`
			TaskStartToCloseTimeout      string `json:"taskStartToCloseTimeout"`
			TaskList                     struct {
				Name string `json:"name"`
			} `json:"taskList"`
		} `json:"executionConfiguration"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &describeWorkflowExecutionOut); err != nil {
		t.Fatalf("unmarshal describe workflow execution response: %v", err)
	}
	if describeWorkflowExecutionOut.ExecutionConfiguration.ChildPolicy == "" ||
		describeWorkflowExecutionOut.ExecutionConfiguration.ExecutionStartToCloseTimeout == "" ||
		describeWorkflowExecutionOut.ExecutionConfiguration.TaskStartToCloseTimeout == "" ||
		describeWorkflowExecutionOut.ExecutionConfiguration.TaskList.Name == "" {
		t.Fatalf("expected execution configuration fields, got %+v", describeWorkflowExecutionOut.ExecutionConfiguration)
	}

	resp = swfRequest(t, ts, "SimpleWorkflowService.PollForActivityTask", []byte(`{"domain":"shard3-domain","taskList":{"name":"main"}}`))
	assertStatus(t, resp, http.StatusOK)
	var pollActivityOut struct {
		TaskToken      string `json:"taskToken"`
		StartedEventID int64  `json:"startedEventId"`
		ActivityType   struct {
			Name string `json:"name"`
		} `json:"activityType"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &pollActivityOut); err != nil {
		t.Fatalf("unmarshal poll for activity task response: %v", err)
	}
	if pollActivityOut.TaskToken == "" || pollActivityOut.StartedEventID == 0 || pollActivityOut.ActivityType.Name == "" {
		t.Fatalf("expected poll for activity task fields, got %+v", pollActivityOut)
	}

	resp = swfRequest(t, ts, "SimpleWorkflowService.PollForDecisionTask", []byte(`{"domain":"shard3-domain","taskList":{"name":"main"}}`))
	assertStatus(t, resp, http.StatusOK)
	var pollDecisionOut struct {
		TaskToken      string           `json:"taskToken"`
		StartedEventID int64            `json:"startedEventId"`
		Events         []map[string]any `json:"events"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &pollDecisionOut); err != nil {
		t.Fatalf("unmarshal poll for decision task response: %v", err)
	}
	if pollDecisionOut.TaskToken == "" || pollDecisionOut.StartedEventID == 0 || len(pollDecisionOut.Events) == 0 {
		t.Fatalf("expected poll for decision task fields, got %+v", pollDecisionOut)
	}
}
