package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestBlockchainShard4ListShapesOmitLegacyFields(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := blockchainRequest(t, ts, http.MethodPost, "/list-transactions", []byte(`{
		"address":"0x1111111111111111111111111111111111111111",
		"network":"ETHEREUM_MAINNET"
	}`))
	assertStatus(t, resp, http.StatusOK)
	var transactionsOut struct {
		Transactions []map[string]any `json:"transactions"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &transactionsOut); err != nil {
		t.Fatalf("unmarshal list transactions response: %v", err)
	}
	if len(transactionsOut.Transactions) == 0 {
		t.Fatalf("expected at least one transaction")
	}
	if _, ok := transactionsOut.Transactions[0]["transactionId"]; ok {
		t.Fatalf("expected list transaction items to omit legacy transactionId field")
	}

	resp = blockchainRequest(t, ts, http.MethodPost, "/list-transaction-events", []byte(`{
		"network":"ETHEREUM_MAINNET",
		"transactionHash":"0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	}`))
	assertStatus(t, resp, http.StatusOK)
	var eventsOut struct {
		Events []map[string]any `json:"events"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &eventsOut); err != nil {
		t.Fatalf("unmarshal list transaction events response: %v", err)
	}
	if len(eventsOut.Events) == 0 {
		t.Fatalf("expected at least one transaction event")
	}
	for _, key := range []string{
		"transactionId",
		"voutIndex",
		"voutSpent",
		"spentVoutTransactionId",
		"spentVoutTransactionHash",
		"spentVoutIndex",
		"blockchainInstant",
		"confirmationStatus",
	} {
		if _, ok := eventsOut.Events[0][key]; ok {
			t.Fatalf("expected transaction event to omit legacy field %q", key)
		}
	}
}

func TestKMSShard4GetKeyPolicyOmitsLegacyPolicyName(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	keyID, _ := kmsCreateKeyForTest(t, ts, map[string]any{
		"Description": "shard4-key",
		"KeyUsage":    "ENCRYPT_DECRYPT",
		"KeySpec":     "SYMMETRIC_DEFAULT",
	})

	resp := kmsRequest(t, ts, "PutKeyPolicy", mustJSON(t, map[string]any{
		"KeyId":      keyID,
		"PolicyName": "default",
		"Policy":     `{"Version":"2012-10-17","Statement":[]}`,
	}))
	assertStatus(t, resp, http.StatusOK)

	resp = kmsRequest(t, ts, "GetKeyPolicy", mustJSON(t, map[string]any{
		"KeyId":      keyID,
		"PolicyName": "default",
	}))
	assertStatus(t, resp, http.StatusOK)
	var out map[string]any
	if err := json.Unmarshal(mustBody(t, resp), &out); err != nil {
		t.Fatalf("unmarshal get key policy response: %v", err)
	}
	if _, ok := out["PolicyName"]; ok {
		t.Fatalf("expected GetKeyPolicy response to omit PolicyName")
	}
	if policy := out["Policy"]; policy == nil || policy == "" {
		t.Fatalf("expected GetKeyPolicy response to include Policy")
	}
}

func TestLambdaShard4CodeSigningAndScalingShapes(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := lambdaRequest(t, ts, http.MethodPost, "/2015-03-31/functions", []byte(`{
		"FunctionName":"shard4-fn",
		"Role":"arn:aws:iam::123456789012:role/lambda-role",
		"Runtime":"provided.al2",
		"Handler":"bootstrap",
		"Code":{"ZipFile":"c3RhY2t5YXJk"}
	}`))
	assertStatus(t, resp, http.StatusCreated)

	resp = lambdaRequest(t, ts, http.MethodPost, "/2020-04-22/code-signing-configs", []byte(`{
		"Description":"shard4",
		"AllowedPublishers":{"SigningProfileVersionArns":["arn:aws:signer:us-east-1:123456789012:/signing-profiles/stackyard"]},
		"CodeSigningPolicies":{"UntrustedArtifactOnDeployment":"Warn"}
	}`))
	assertStatus(t, resp, http.StatusCreated)
	var createCodeSigningOut struct {
		CodeSigningConfig struct {
			CodeSigningConfigArn string `json:"CodeSigningConfigArn"`
		} `json:"CodeSigningConfig"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &createCodeSigningOut); err != nil {
		t.Fatalf("unmarshal create code signing config response: %v", err)
	}
	if createCodeSigningOut.CodeSigningConfig.CodeSigningConfigArn == "" {
		t.Fatalf("expected code signing config arn")
	}

	resp = lambdaRequest(
		t,
		ts,
		http.MethodPut,
		"/2020-06-30/functions/shard4-fn/code-signing-config",
		mustJSON(t, map[string]any{"CodeSigningConfigArn": createCodeSigningOut.CodeSigningConfig.CodeSigningConfigArn}),
	)
	assertStatus(t, resp, http.StatusOK)
	var putCodeSigningOut struct {
		CodeSigningConfigArn string `json:"CodeSigningConfigArn"`
		FunctionName         string `json:"FunctionName"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &putCodeSigningOut); err != nil {
		t.Fatalf("unmarshal put function code signing config response: %v", err)
	}
	if putCodeSigningOut.FunctionName != "shard4-fn" {
		t.Fatalf("expected FunctionName in put function code signing config response, got %q", putCodeSigningOut.FunctionName)
	}

	resp = lambdaRequest(t, ts, http.MethodGet, "/2020-06-30/functions/shard4-fn/code-signing-config", nil)
	assertStatus(t, resp, http.StatusOK)
	var getCodeSigningOut struct {
		CodeSigningConfigArn string `json:"CodeSigningConfigArn"`
		FunctionName         string `json:"FunctionName"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &getCodeSigningOut); err != nil {
		t.Fatalf("unmarshal get function code signing config response: %v", err)
	}
	if getCodeSigningOut.FunctionName != "shard4-fn" {
		t.Fatalf("expected FunctionName in get function code signing config response, got %q", getCodeSigningOut.FunctionName)
	}

	resp = lambdaRequest(
		t,
		ts,
		http.MethodGet,
		"/2020-04-22/code-signing-configs/"+url.PathEscape(createCodeSigningOut.CodeSigningConfig.CodeSigningConfigArn)+"/functions",
		nil,
	)
	assertStatus(t, resp, http.StatusOK)
	var listFunctionsOut struct {
		FunctionArns []string `json:"FunctionArns"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &listFunctionsOut); err != nil {
		t.Fatalf("unmarshal list functions by code signing config response: %v", err)
	}
	if len(listFunctionsOut.FunctionArns) == 0 {
		t.Fatalf("expected at least one function arn")
	}

	resp = lambdaRequest(
		t,
		ts,
		http.MethodPut,
		"/2025-11-30/functions/shard4-fn/function-scaling-config?Qualifier="+url.QueryEscape("$LATEST"),
		[]byte(`{"FunctionScalingConfig":{"MaxExecutionEnvironments":2,"MinExecutionEnvironments":1}}`),
	)
	assertStatus(t, resp, http.StatusOK)
	var putScalingOut struct {
		FunctionState string `json:"FunctionState"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &putScalingOut); err != nil {
		t.Fatalf("unmarshal put function scaling config response: %v", err)
	}
	if putScalingOut.FunctionState != "Active" {
		t.Fatalf("expected FunctionState=Active, got %q", putScalingOut.FunctionState)
	}

	resp = lambdaRequest(
		t,
		ts,
		http.MethodGet,
		"/2025-11-30/functions/shard4-fn/function-scaling-config?Qualifier="+url.QueryEscape("$LATEST"),
		nil,
	)
	assertStatus(t, resp, http.StatusOK)
	var getScalingOut struct {
		FunctionArn string `json:"FunctionArn"`
		Applied     struct {
			MaxExecutionEnvironments float64 `json:"MaxExecutionEnvironments"`
			MinExecutionEnvironments float64 `json:"MinExecutionEnvironments"`
		} `json:"AppliedFunctionScalingConfig"`
		Requested struct {
			MaxExecutionEnvironments float64 `json:"MaxExecutionEnvironments"`
			MinExecutionEnvironments float64 `json:"MinExecutionEnvironments"`
		} `json:"RequestedFunctionScalingConfig"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &getScalingOut); err != nil {
		t.Fatalf("unmarshal get function scaling config response: %v", err)
	}
	if getScalingOut.FunctionArn == "" || getScalingOut.Applied.MaxExecutionEnvironments != 2 || getScalingOut.Requested.MinExecutionEnvironments != 1 {
		t.Fatalf("expected modeled scaling config response, got %+v", getScalingOut)
	}
}

func TestLambdaShard4DurableExecutionShapes(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	durableExecutionARN := "arn:aws:lambda:us-east-1:123456789012:durable-execution:shard4"

	resp := lambdaRequest(
		t,
		ts,
		http.MethodPost,
		"/2025-12-01/durable-executions/"+url.PathEscape(durableExecutionARN)+"/checkpoint",
		[]byte(`{"CheckpointToken":"checkpoint-1","Updates":[]}`),
	)
	assertStatus(t, resp, http.StatusOK)
	var checkpointOut struct {
		CheckpointToken   string `json:"CheckpointToken"`
		NewExecutionState struct {
			Operations []any `json:"Operations"`
		} `json:"NewExecutionState"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &checkpointOut); err != nil {
		t.Fatalf("unmarshal checkpoint durable execution response: %v", err)
	}
	if checkpointOut.CheckpointToken == "" {
		t.Fatalf("expected next checkpoint token")
	}

	resp = lambdaRequest(
		t,
		ts,
		http.MethodGet,
		"/2025-12-01/durable-executions/"+url.PathEscape(durableExecutionARN)+"/state?CheckpointToken=checkpoint-1&MaxItems=10",
		nil,
	)
	assertStatus(t, resp, http.StatusOK)
	var stateOut struct {
		Operations []any `json:"Operations"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &stateOut); err != nil {
		t.Fatalf("unmarshal get durable execution state response: %v", err)
	}
	if stateOut.Operations == nil {
		t.Fatalf("expected Operations field in durable execution state response")
	}

	resp = lambdaRequest(
		t,
		ts,
		http.MethodPost,
		"/2025-12-01/durable-executions/"+url.PathEscape(durableExecutionARN)+"/stop",
		[]byte(`{"Error":{"ErrorType":"RuntimeError","ErrorMessage":"stop requested"}}`),
	)
	assertStatus(t, resp, http.StatusOK)
	var stopOut struct {
		StopTimestamp string `json:"StopTimestamp"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &stopOut); err != nil {
		t.Fatalf("unmarshal stop durable execution response: %v", err)
	}
	if stopOut.StopTimestamp == "" {
		t.Fatalf("expected StopTimestamp in stop durable execution response")
	}

	resp = lambdaRequest(
		t,
		ts,
		http.MethodPost,
		"/2025-12-01/durable-execution-callbacks/cb-shard4/succeed",
		[]byte("c3RhY2t5YXJk"),
	)
	assertStatus(t, resp, http.StatusOK)
	var callbackOut map[string]any
	if err := json.Unmarshal(mustBody(t, resp), &callbackOut); err != nil {
		t.Fatalf("unmarshal durable execution callback success response: %v", err)
	}
	if len(callbackOut) != 0 {
		t.Fatalf("expected empty durable execution callback success response, got %+v", callbackOut)
	}
}
