package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	ebtypes "github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

const (
	defaultRegion      = "us-east-1"
	defaultAccountID   = "123456789012"
	defaultLakeBucket  = "stackyard-ra-data-lake"
	defaultAuditBucket = "stackyard-ra-audit"
)

type serviceError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message,omitempty"`
}

type runRecord struct {
	RunID          string `json:"runId"`
	ExecutionARN   string `json:"executionArn"`
	RawObjectKey   string `json:"rawObjectKey"`
	CuratedKey     string `json:"curatedObjectKey"`
	PublishedKey   string `json:"publishedObjectKey"`
	StartedAtUTC   string `json:"startedAtUtc"`
	CompletedAtUTC string `json:"completedAtUtc"`
	Status         string `json:"status"`
}

type tenantState struct {
	TenantID        string      `json:"tenantId"`
	KMSKeyArn       string      `json:"kmsKeyArn"`
	KMSKeyID        string      `json:"kmsKeyId"`
	SecretArn       string      `json:"secretArn"`
	QueueURL        string      `json:"queueUrl"`
	QueueArn        string      `json:"queueArn"`
	EventBusName    string      `json:"eventBusName"`
	EventBusArn     string      `json:"eventBusArn"`
	RuleName        string      `json:"ruleName"`
	RuleArn         string      `json:"ruleArn"`
	StateMachineArn string      `json:"stateMachineArn"`
	LatestRawKey    string      `json:"latestRawKey"`
	Runs            []runRecord `json:"runs"`
	UpdatedAtUTC    string      `json:"updatedAtUtc"`
}

type tenantDetails struct {
	tenantState
	ObjectsByZone map[string][]string `json:"objectsByZone"`
}

type app struct {
	mu       sync.Mutex
	tenants  map[string]*tenantState
	endpoint string
	region   string
	account  string

	lakeBucket  string
	auditBucket string

	awsCreds aws.CredentialsProvider
	signer   *v4.Signer
	httpc    *http.Client

	s3Client  *s3.Client
	sqsClient *sqs.Client
	ebClient  *eventbridge.Client
}

func main() {
	ctx := context.Background()
	instance, err := newApp(ctx)
	if err != nil {
		log.Fatalf("init app: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", instance.handleHealth)
	mux.HandleFunc("/api/v1/summary", instance.handleSummary)
	mux.HandleFunc("/api/v1/tenants/", instance.handleTenants)

	addr := getenv("APP_ADDR", ":8080")
	log.Printf("data-pipeline backend listening on %s", addr)
	if err := http.ListenAndServe(addr, loggingMiddleware(mux)); err != nil {
		log.Fatalf("listen: %v", err)
	}
}

func newApp(ctx context.Context) (*app, error) {
	endpoint := strings.TrimRight(getenv("STACKYARD_ENDPOINT", "http://localhost:4566"), "/")
	region := getenv("AWS_REGION", defaultRegion)
	accessKey := getenv("AWS_ACCESS_KEY_ID", "stackyard")
	secretKey := getenv("AWS_SECRET_ACCESS_KEY", "stackyard")
	account := getenv("AWS_ACCOUNT_ID", defaultAccountID)

	creds := credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")
	resolver := aws.EndpointResolverWithOptionsFunc(func(service, r string, _ ...any) (aws.Endpoint, error) {
		switch service {
		case s3.ServiceID, sqs.ServiceID, eventbridge.ServiceID:
			return aws.Endpoint{URL: endpoint, SigningRegion: region, HostnameImmutable: true}, nil
		default:
			return aws.Endpoint{}, &aws.EndpointNotFoundError{}
		}
	})

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
		config.WithCredentialsProvider(creds),
		config.WithEndpointResolverWithOptions(resolver),
	)
	if err != nil {
		return nil, err
	}

	instance := &app{
		tenants:     map[string]*tenantState{},
		endpoint:    endpoint,
		region:      region,
		account:     account,
		lakeBucket:  getenv("STACKYARD_DATA_LAKE_BUCKET", defaultLakeBucket),
		auditBucket: getenv("STACKYARD_AUDIT_BUCKET", defaultAuditBucket),
		awsCreds:    creds,
		signer:      v4.NewSigner(),
		httpc:       &http.Client{Timeout: 15 * time.Second},
		s3Client: s3.NewFromConfig(cfg, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(endpoint)
			o.UsePathStyle = true
		}),
		sqsClient: sqs.NewFromConfig(cfg),
		ebClient:  eventbridge.NewFromConfig(cfg),
	}

	return instance, nil
}

func (a *app) handleHealth(w http.ResponseWriter, _ *http.Request) {
	respondJSON(w, http.StatusOK, map[string]any{
		"status":      "ok",
		"endpoint":    a.endpoint,
		"dataLake":    a.lakeBucket,
		"auditBucket": a.auditBucket,
	})
}

func (a *app) handleSummary(w http.ResponseWriter, _ *http.Request) {
	a.mu.Lock()
	defer a.mu.Unlock()

	items := make([]tenantState, 0, len(a.tenants))
	for _, t := range a.tenants {
		items = append(items, *cloneTenant(t))
	}
	sort.Slice(items, func(i, j int) bool { return items[i].TenantID < items[j].TenantID })

	respondJSON(w, http.StatusOK, map[string]any{
		"tenants": items,
	})
}

func (a *app) handleTenants(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		respondErr(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		return
	}

	trimmed := strings.TrimPrefix(r.URL.Path, "/api/v1/tenants/")
	parts := splitPath(trimmed)
	if len(parts) == 0 {
		respondErr(w, http.StatusBadRequest, "ValidationException", "missing tenant id")
		return
	}
	tenantID := strings.ToLower(strings.TrimSpace(parts[0]))
	if !isValidTenantID(tenantID) {
		respondErr(w, http.StatusBadRequest, "ValidationException", "tenant id must match [a-z0-9-]{3,32}")
		return
	}

	action := ""
	if len(parts) > 1 {
		action = strings.ToLower(strings.TrimSpace(parts[1]))
	}

	switch {
	case r.Method == http.MethodGet && action == "":
		a.getTenantDetails(r.Context(), w, tenantID)
	case r.Method == http.MethodPost && action == "bootstrap":
		a.bootstrapTenant(r.Context(), w, tenantID)
	case r.Method == http.MethodPost && action == "ingest":
		a.ingestTenant(r.Context(), w, tenantID)
	case r.Method == http.MethodPost && action == "run":
		a.runTenant(r.Context(), w, tenantID)
	default:
		respondErr(w, http.StatusNotFound, "ResourceNotFoundException", "unknown endpoint")
	}
}

func (a *app) getTenantDetails(ctx context.Context, w http.ResponseWriter, tenantID string) {
	st, err := a.getTenantState(tenantID)
	if err != nil {
		respondErr(w, http.StatusNotFound, "ResourceNotFoundException", err.Error())
		return
	}

	zones := map[string]string{
		"raw":        fmt.Sprintf("raw/tenant=%s/", tenantID),
		"curated":    fmt.Sprintf("curated/tenant=%s/", tenantID),
		"published":  fmt.Sprintf("published/tenant=%s/", tenantID),
		"quarantine": fmt.Sprintf("quarantine/tenant=%s/", tenantID),
	}
	objects := map[string][]string{}
	for zone, prefix := range zones {
		keys, listErr := a.listObjectsByPrefix(ctx, a.lakeBucket, prefix)
		if listErr != nil {
			respondErr(w, http.StatusBadGateway, "ServiceError", listErr.Error())
			return
		}
		objects[zone] = keys
	}

	respondJSON(w, http.StatusOK, tenantDetails{
		tenantState:   *st,
		ObjectsByZone: objects,
	})
}

func (a *app) bootstrapTenant(ctx context.Context, w http.ResponseWriter, tenantID string) {
	if err := a.ensureBucket(ctx, a.lakeBucket); err != nil {
		respondErr(w, http.StatusBadGateway, "ServiceError", err.Error())
		return
	}
	if err := a.ensureBucket(ctx, a.auditBucket); err != nil {
		respondErr(w, http.StatusBadGateway, "ServiceError", err.Error())
		return
	}

	st := a.ensureTenant(tenantID)

	if st.KMSKeyArn == "" {
		keyID, keyARN, err := a.createTenantKMSKey(ctx, tenantID)
		if err != nil {
			respondErr(w, http.StatusBadGateway, "ServiceError", err.Error())
			return
		}
		st.KMSKeyID = keyID
		st.KMSKeyArn = keyARN
	}

	if st.SecretArn == "" {
		secretARN, err := a.createOrUpdateTenantSecret(ctx, tenantID)
		if err != nil {
			respondErr(w, http.StatusBadGateway, "ServiceError", err.Error())
			return
		}
		st.SecretArn = secretARN
	}

	if st.QueueURL == "" {
		queueURL, queueARN, err := a.createTenantQueue(ctx, tenantID)
		if err != nil {
			respondErr(w, http.StatusBadGateway, "ServiceError", err.Error())
			return
		}
		st.QueueURL = queueURL
		st.QueueArn = queueARN
	}

	if st.EventBusArn == "" {
		busName, busARN, err := a.createTenantEventBus(ctx, tenantID)
		if err != nil {
			respondErr(w, http.StatusBadGateway, "ServiceError", err.Error())
			return
		}
		st.EventBusName = busName
		st.EventBusArn = busARN
	}

	if st.RuleArn == "" {
		ruleName, ruleArn, err := a.createTenantRuleAndTarget(ctx, tenantID, st.EventBusName, st.QueueArn)
		if err != nil {
			respondErr(w, http.StatusBadGateway, "ServiceError", err.Error())
			return
		}
		st.RuleName = ruleName
		st.RuleArn = ruleArn
	}

	if st.StateMachineArn == "" {
		stateMachineArn, err := a.ensureTenantStateMachine(ctx, tenantID)
		if err != nil {
			respondErr(w, http.StatusBadGateway, "ServiceError", err.Error())
			return
		}
		st.StateMachineArn = stateMachineArn
	}

	st.UpdatedAtUTC = time.Now().UTC().Format(time.RFC3339)
	respondJSON(w, http.StatusOK, map[string]any{
		"message": "tenant bootstrap complete",
		"tenant":  st,
	})
}

func (a *app) ingestTenant(ctx context.Context, w http.ResponseWriter, tenantID string) {
	st, err := a.getTenantStatePtr(tenantID)
	if err != nil {
		respondErr(w, http.StatusNotFound, "ResourceNotFoundException", "tenant is not bootstrapped")
		return
	}

	if st.KMSKeyID == "" {
		respondErr(w, http.StatusBadRequest, "ValidationException", "tenant does not have a KMS key")
		return
	}

	now := time.Now().UTC()
	datePart := now.Format("2006-01-02")
	key := fmt.Sprintf("raw/tenant=%s/source=demo/date=%s/batch-%d.json", tenantID, datePart, now.UnixNano())
	payload := map[string]any{
		"tenantId":   tenantID,
		"ingestedAt": now.Format(time.RFC3339),
		"records": []map[string]any{
			{"id": "evt-1", "amount": 52.4, "currency": "USD"},
			{"id": "evt-2", "amount": 17.9, "currency": "USD"},
		},
	}
	body, _ := json.MarshalIndent(payload, "", "  ")

	_, err = a.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:               aws.String(a.lakeBucket),
		Key:                  aws.String(key),
		Body:                 bytes.NewReader(body),
		ContentType:          aws.String("application/json"),
		ServerSideEncryption: s3types.ServerSideEncryptionAwsKms,
		SSEKMSKeyId:          aws.String(st.KMSKeyID),
	})
	if err != nil {
		respondErr(w, http.StatusBadGateway, "ServiceError", fmt.Sprintf("put raw object: %v", err))
		return
	}

	if st.QueueURL != "" {
		_, sendErr := a.sqsClient.SendMessage(ctx, &sqs.SendMessageInput{
			QueueUrl:    aws.String(st.QueueURL),
			MessageBody: aws.String(fmt.Sprintf(`{"tenantId":"%s","rawObjectKey":"%s"}`, tenantID, key)),
		})
		if sendErr != nil {
			respondErr(w, http.StatusBadGateway, "ServiceError", fmt.Sprintf("send sqs message: %v", sendErr))
			return
		}
	}

	st.LatestRawKey = key
	st.UpdatedAtUTC = now.Format(time.RFC3339)

	respondJSON(w, http.StatusOK, map[string]any{
		"message":      "raw ingest complete",
		"rawObjectKey": key,
		"tenant":       st,
	})
}

func (a *app) runTenant(ctx context.Context, w http.ResponseWriter, tenantID string) {
	st, err := a.getTenantStatePtr(tenantID)
	if err != nil {
		respondErr(w, http.StatusNotFound, "ResourceNotFoundException", "tenant is not bootstrapped")
		return
	}
	if st.LatestRawKey == "" {
		respondErr(w, http.StatusBadRequest, "ValidationException", "no raw data available, run ingest first")
		return
	}

	rawPayload, err := a.readObject(ctx, a.lakeBucket, st.LatestRawKey)
	if err != nil {
		respondErr(w, http.StatusBadGateway, "ServiceError", fmt.Sprintf("read raw object: %v", err))
		return
	}

	now := time.Now().UTC()
	runID := fmt.Sprintf("run-%d", now.UnixNano())
	datePart := now.Format("2006-01-02")
	curatedKey := fmt.Sprintf("curated/tenant=%s/dataset=orders/date=%s/%s.json", tenantID, datePart, runID)
	publishedKey := fmt.Sprintf("published/tenant=%s/dataset=orders/date=%s/%s.json", tenantID, datePart, runID)

	curatedPayload := map[string]any{
		"tenantId":    tenantID,
		"runId":       runID,
		"transformed": true,
		"sourceKey":   st.LatestRawKey,
		"raw":         json.RawMessage(rawPayload),
	}
	curatedBody, _ := json.MarshalIndent(curatedPayload, "", "  ")

	if err := a.writeObject(ctx, a.lakeBucket, curatedKey, curatedBody, st.KMSKeyID); err != nil {
		respondErr(w, http.StatusBadGateway, "ServiceError", err.Error())
		return
	}
	if err := a.writeObject(ctx, a.lakeBucket, publishedKey, curatedBody, st.KMSKeyID); err != nil {
		respondErr(w, http.StatusBadGateway, "ServiceError", err.Error())
		return
	}

	execArn, err := a.startStateMachineExecution(ctx, st.StateMachineArn, runID, map[string]any{
		"tenantId":           tenantID,
		"rawObjectKey":       st.LatestRawKey,
		"curatedObjectKey":   curatedKey,
		"publishedObjectKey": publishedKey,
	})
	if err != nil {
		respondErr(w, http.StatusBadGateway, "ServiceError", err.Error())
		return
	}

	_, ebErr := a.ebClient.PutEvents(ctx, &eventbridge.PutEventsInput{
		Entries: []ebtypes.PutEventsRequestEntry{
			{
				EventBusName: aws.String(st.EventBusName),
				Source:       aws.String("reference-architecture.data-pipeline"),
				DetailType:   aws.String("PipelineRunCompleted"),
				Detail:       aws.String(fmt.Sprintf(`{"tenantId":"%s","runId":"%s","status":"SUCCEEDED"}`, tenantID, runID)),
				Time:         aws.Time(now),
			},
		},
	})
	if ebErr != nil {
		respondErr(w, http.StatusBadGateway, "ServiceError", fmt.Sprintf("put eventbridge event: %v", ebErr))
		return
	}

	auditKey := fmt.Sprintf("runs/tenant=%s/date=%s/%s.json", tenantID, datePart, runID)
	auditPayload, _ := json.MarshalIndent(map[string]any{
		"tenantId":     tenantID,
		"runId":        runID,
		"executionArn": execArn,
		"timestamp":    now.Format(time.RFC3339),
		"status":       "SUCCEEDED",
	}, "", "  ")
	if err := a.writeObject(ctx, a.auditBucket, auditKey, auditPayload, st.KMSKeyID); err != nil {
		respondErr(w, http.StatusBadGateway, "ServiceError", err.Error())
		return
	}

	run := runRecord{
		RunID:          runID,
		ExecutionARN:   execArn,
		RawObjectKey:   st.LatestRawKey,
		CuratedKey:     curatedKey,
		PublishedKey:   publishedKey,
		StartedAtUTC:   now.Format(time.RFC3339),
		CompletedAtUTC: time.Now().UTC().Format(time.RFC3339),
		Status:         "SUCCEEDED",
	}
	st.Runs = append([]runRecord{run}, st.Runs...)
	st.UpdatedAtUTC = time.Now().UTC().Format(time.RFC3339)

	respondJSON(w, http.StatusOK, map[string]any{
		"message": "pipeline run completed",
		"run":     run,
		"tenant":  st,
	})
}

func (a *app) ensureBucket(ctx context.Context, bucket string) error {
	_, err := a.s3Client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(bucket)})
	if err == nil {
		return nil
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "bucketalreadyownedbyyou") || strings.Contains(msg, "bucketalreadyexists") {
		return nil
	}
	return err
}

func (a *app) createTenantKMSKey(ctx context.Context, tenantID string) (string, string, error) {
	payload := map[string]any{
		"Description": fmt.Sprintf("Reference architecture key for tenant %s", tenantID),
		"KeyUsage":    "ENCRYPT_DECRYPT",
		"Tags": []map[string]string{
			{"TagKey": "tenant_id", "TagValue": tenantID},
		},
	}
	var out struct {
		KeyMetadata struct {
			Arn   string `json:"Arn"`
			KeyID string `json:"KeyId"`
		} `json:"KeyMetadata"`
	}
	if err := a.invokeJSON(ctx, "kms", "TrentService.CreateKey", "application/x-amz-json-1.1", payload, &out); err != nil {
		return "", "", err
	}
	if out.KeyMetadata.Arn == "" || out.KeyMetadata.KeyID == "" {
		return "", "", errors.New("kms create key returned incomplete metadata")
	}
	return out.KeyMetadata.KeyID, out.KeyMetadata.Arn, nil
}

func (a *app) createOrUpdateTenantSecret(ctx context.Context, tenantID string) (string, error) {
	name := fmt.Sprintf("reference-architecture/data-pipeline/%s/db", tenantID)
	secretJSON := fmt.Sprintf(`{"username":"tenant_%s","password":"stackyard-demo"}`, tenantID)

	createPayload := map[string]any{
		"Name":         name,
		"SecretString": secretJSON,
	}
	var createOut struct {
		ARN string `json:"ARN"`
	}
	err := a.invokeJSON(ctx, "secretsmanager", "secretsmanager.CreateSecret", "application/x-amz-json-1.1", createPayload, &createOut)
	if err == nil {
		return createOut.ARN, nil
	}
	if !strings.Contains(strings.ToLower(err.Error()), "resourceexistsexception") {
		return "", err
	}

	updatePayload := map[string]any{
		"SecretId":     name,
		"SecretString": secretJSON,
	}
	var updateOut struct {
		ARN string `json:"ARN"`
	}
	if upErr := a.invokeJSON(ctx, "secretsmanager", "secretsmanager.UpdateSecret", "application/x-amz-json-1.1", updatePayload, &updateOut); upErr != nil {
		return "", upErr
	}
	if updateOut.ARN != "" {
		return updateOut.ARN, nil
	}

	var describeOut struct {
		ARN string `json:"ARN"`
	}
	describePayload := map[string]any{"SecretId": name}
	if descErr := a.invokeJSON(ctx, "secretsmanager", "secretsmanager.DescribeSecret", "application/x-amz-json-1.1", describePayload, &describeOut); descErr != nil {
		return "", descErr
	}
	return describeOut.ARN, nil
}

func (a *app) createTenantQueue(ctx context.Context, tenantID string) (string, string, error) {
	queueName := fmt.Sprintf("ra-data-pipeline-%s", tenantID)
	resp, err := a.sqsClient.CreateQueue(ctx, &sqs.CreateQueueInput{QueueName: aws.String(queueName)})
	if err != nil {
		return "", "", err
	}
	queueURL := aws.ToString(resp.QueueUrl)
	queueARN := fmt.Sprintf("arn:aws:sqs:%s:%s:%s", a.region, a.account, queueName)
	return queueURL, queueARN, nil
}

func (a *app) createTenantEventBus(ctx context.Context, tenantID string) (string, string, error) {
	name := fmt.Sprintf("ra-data-pipeline-%s", tenantID)
	resp, err := a.ebClient.CreateEventBus(ctx, &eventbridge.CreateEventBusInput{Name: aws.String(name)})
	if err == nil {
		return name, aws.ToString(resp.EventBusArn), nil
	}

	if !strings.Contains(strings.ToLower(err.Error()), "resourcealreadyexistsexception") {
		return "", "", err
	}

	describe, describeErr := a.ebClient.DescribeEventBus(ctx, &eventbridge.DescribeEventBusInput{Name: aws.String(name)})
	if describeErr != nil {
		return "", "", describeErr
	}
	return name, aws.ToString(describe.Arn), nil
}

func (a *app) createTenantRuleAndTarget(ctx context.Context, tenantID, busName, queueArn string) (string, string, error) {
	ruleName := fmt.Sprintf("ra-pipeline-trigger-%s", tenantID)
	putRuleOut, err := a.ebClient.PutRule(ctx, &eventbridge.PutRuleInput{
		Name:         aws.String(ruleName),
		EventBusName: aws.String(busName),
		EventPattern: aws.String(fmt.Sprintf(`{"detail":{"tenantId":["%s"]}}`, tenantID)),
		State:        ebtypes.RuleStateEnabled,
		Description:  aws.String("Reference architecture pipeline trigger"),
	})
	if err != nil {
		return "", "", err
	}

	_, err = a.ebClient.PutTargets(ctx, &eventbridge.PutTargetsInput{
		EventBusName: aws.String(busName),
		Rule:         aws.String(ruleName),
		Targets: []ebtypes.Target{
			{Id: aws.String("pipeline-queue"), Arn: aws.String(queueArn)},
		},
	})
	if err != nil {
		return "", "", err
	}

	return ruleName, aws.ToString(putRuleOut.RuleArn), nil
}

func (a *app) ensureTenantStateMachine(ctx context.Context, tenantID string) (string, error) {
	name := fmt.Sprintf("ra-pipeline-%s", tenantID)
	definition := `{"Comment":"Reference architecture pipeline","StartAt":"Validate","States":{"Validate":{"Type":"Pass","Next":"Transform"},"Transform":{"Type":"Pass","Next":"Publish"},"Publish":{"Type":"Succeed"}}}`
	payload := map[string]any{
		"name":       name,
		"definition": definition,
		"roleArn":    fmt.Sprintf("arn:aws:iam::%s:role/stackyard-demo-step-functions", a.account),
		"type":       "STANDARD",
	}
	var out struct {
		StateMachineArn string `json:"stateMachineArn"`
	}
	err := a.invokeJSON(ctx, "states", "AWSStepFunctions.CreateStateMachine", "application/x-amz-json-1.0", payload, &out)
	if err == nil {
		return out.StateMachineArn, nil
	}
	if !strings.Contains(strings.ToLower(err.Error()), "statemachinealreadyexists") {
		return "", err
	}

	var listOut struct {
		StateMachines []struct {
			Name            string `json:"name"`
			StateMachineArn string `json:"stateMachineArn"`
		} `json:"stateMachines"`
	}
	if listErr := a.invokeJSON(ctx, "states", "AWSStepFunctions.ListStateMachines", "application/x-amz-json-1.0", map[string]any{"maxResults": 100}, &listOut); listErr != nil {
		return "", listErr
	}
	for _, sm := range listOut.StateMachines {
		if sm.Name == name {
			return sm.StateMachineArn, nil
		}
	}
	return "", errors.New("state machine already exists but was not listed")
}

func (a *app) startStateMachineExecution(ctx context.Context, stateMachineARN, runID string, input map[string]any) (string, error) {
	if stateMachineARN == "" {
		return "", errors.New("state machine arn is empty")
	}
	inputJSON, _ := json.Marshal(input)
	payload := map[string]any{
		"stateMachineArn": stateMachineARN,
		"name":            runID,
		"input":           string(inputJSON),
	}
	var out struct {
		ExecutionArn string `json:"executionArn"`
	}
	if err := a.invokeJSON(ctx, "states", "AWSStepFunctions.StartExecution", "application/x-amz-json-1.0", payload, &out); err != nil {
		return "", err
	}
	return out.ExecutionArn, nil
}

func (a *app) writeObject(ctx context.Context, bucket, key string, payload []byte, kmsKeyID string) error {
	input := &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(payload),
		ContentType: aws.String("application/json"),
	}
	if strings.TrimSpace(kmsKeyID) != "" {
		input.ServerSideEncryption = s3types.ServerSideEncryptionAwsKms
		input.SSEKMSKeyId = aws.String(kmsKeyID)
	}
	_, err := a.s3Client.PutObject(ctx, input)
	if err != nil {
		return fmt.Errorf("put object %s/%s: %w", bucket, key, err)
	}
	return nil
}

func (a *app) readObject(ctx context.Context, bucket, key string) ([]byte, error) {
	resp, err := a.s3Client.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(bucket), Key: aws.String(key)})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func (a *app) listObjectsByPrefix(ctx context.Context, bucket, prefix string) ([]string, error) {
	resp, err := a.s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	})
	if err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(resp.Contents))
	for _, obj := range resp.Contents {
		keys = append(keys, aws.ToString(obj.Key))
	}
	sort.Strings(keys)
	return keys, nil
}

func (a *app) invokeJSON(ctx context.Context, service, target, contentType string, payload any, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.endpoint+"/", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("X-Amz-Target", target)

	credValue, err := a.awsCreds.Retrieve(ctx)
	if err != nil {
		return err
	}
	if err := a.signer.SignHTTP(ctx, credValue, req, hashSHA256(body), service, a.region, time.Now()); err != nil {
		return err
	}

	resp, err := a.httpc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= http.StatusBadRequest {
		var apiErr serviceError
		if json.Unmarshal(respBody, &apiErr) == nil {
			t := strings.TrimSpace(apiErr.Type)
			m := strings.TrimSpace(apiErr.Message)
			if t == "" {
				t = "ServiceError"
			}
			if m == "" {
				m = strings.TrimSpace(string(respBody))
			}
			return fmt.Errorf("%s: %s", t, m)
		}
		return fmt.Errorf("service %s returned %d: %s", service, resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	if out != nil {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("decode %s response: %w", target, err)
		}
	}
	return nil
}

func (a *app) ensureTenant(tenantID string) *tenantState {
	a.mu.Lock()
	defer a.mu.Unlock()
	if existing, ok := a.tenants[tenantID]; ok {
		return existing
	}
	st := &tenantState{TenantID: tenantID, Runs: []runRecord{}, UpdatedAtUTC: time.Now().UTC().Format(time.RFC3339)}
	a.tenants[tenantID] = st
	return st
}

func (a *app) getTenantState(tenantID string) (*tenantState, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	st, ok := a.tenants[tenantID]
	if !ok {
		return nil, fmt.Errorf("tenant %s not found", tenantID)
	}
	return cloneTenant(st), nil
}

func (a *app) getTenantStatePtr(tenantID string) (*tenantState, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	st, ok := a.tenants[tenantID]
	if !ok {
		return nil, fmt.Errorf("tenant %s not found", tenantID)
	}
	return st, nil
}

func cloneTenant(in *tenantState) *tenantState {
	if in == nil {
		return nil
	}
	cp := *in
	cp.Runs = append([]runRecord(nil), in.Runs...)
	return &cp
}

func isValidTenantID(v string) bool {
	if len(v) < 3 || len(v) > 32 {
		return false
	}
	for _, r := range v {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			continue
		}
		return false
	}
	return true
}

func splitPath(path string) []string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func respondErr(w http.ResponseWriter, status int, typ, msg string) {
	respondJSON(w, status, serviceError{Type: typ, Message: msg})
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s (%s)", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
	})
}

func hashSHA256(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func getenv(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}
