package main

import (
	"context"
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

	kms "cloud.google.com/go/kms/apiv1"
	"cloud.google.com/go/kms/apiv1/kmspb"
	pubsub "cloud.google.com/go/pubsub/apiv1"
	"cloud.google.com/go/pubsub/apiv1/pubsubpb"
	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"cloud.google.com/go/storage"
	workflows "cloud.google.com/go/workflows/apiv1"
	"cloud.google.com/go/workflows/apiv1/workflowspb"
	executions "cloud.google.com/go/workflows/executions/apiv1"
	executionspb "cloud.google.com/go/workflows/executions/apiv1/executionspb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

const (
	defaultProjectID   = "stackyard"
	defaultLocation    = "us-central1"
	defaultLakeBucket  = "stackyard-ra-data-lake-gcp"
	defaultAuditBucket = "stackyard-ra-audit-gcp"
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
	mu      sync.Mutex
	tenants map[string]*tenantState

	endpoint    string
	apiEndpoint string
	grpcTarget  string
	projectID   string
	locationID  string

	lakeBucket  string
	auditBucket string

	storageClient    *storage.Client
	kmsClient        *kms.KeyManagementClient
	secretClient     *secretmanager.Client
	publisherClient  *pubsub.PublisherClient
	subscriberClient *pubsub.SubscriberClient
	workflowsClient  *workflows.Client
	executionsClient *executions.Client
}

func main() {
	ctx := context.Background()
	instance, err := newApp(ctx)
	if err != nil {
		log.Fatalf("init app: %v", err)
	}
	defer instance.close()

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", instance.handleHealth)
	mux.HandleFunc("/api/v1/summary", instance.handleSummary)
	mux.HandleFunc("/api/v1/tenants/", instance.handleTenants)

	addr := getenv("APP_ADDR", ":8080")
	log.Printf("data-pipeline backend (gcp) listening on %s", addr)
	if err := http.ListenAndServe(addr, loggingMiddleware(mux)); err != nil {
		log.Fatalf("listen: %v", err)
	}
}

func newApp(ctx context.Context) (*app, error) {
	endpoint := strings.TrimRight(getenv("STACKYARD_ENDPOINT", "http://localhost:4566"), "/")
	apiEndpoint := endpoint + "/gcp"
	grpcTarget := normalizeGRPCTarget(getenv("STACKYARD_GCP_GRPC_ENDPOINT", endpoint))
	projectID := getenv("STACKYARD_GCP_PROJECT_ID", defaultProjectID)
	locationID := getenv("STACKYARD_GCP_LOCATION", defaultLocation)

	if err := os.Setenv("STORAGE_EMULATOR_HOST", apiEndpoint); err != nil {
		return nil, fmt.Errorf("set STORAGE_EMULATOR_HOST: %w", err)
	}

	storageHTTP := &http.Client{Transport: stackyardHeaderTransport{base: http.DefaultTransport, serviceName: "storage-apiv1"}}
	storageClient, err := storage.NewClient(ctx,
		option.WithHTTPClient(storageHTTP),
		option.WithoutAuthentication(),
	)
	if err != nil {
		return nil, fmt.Errorf("create storage client: %w", err)
	}

	kmsHTTP := &http.Client{Transport: stackyardHeaderTransport{base: http.DefaultTransport, serviceName: "kms"}}
	kmsClient, err := kms.NewKeyManagementRESTClient(ctx,
		option.WithHTTPClient(kmsHTTP),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		return nil, fmt.Errorf("create kms client: %w", err)
	}

	secretHTTP := &http.Client{Transport: stackyardHeaderTransport{base: http.DefaultTransport, serviceName: "secretmanager"}}
	secretClient, err := secretmanager.NewRESTClient(ctx,
		option.WithHTTPClient(secretHTTP),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		return nil, fmt.Errorf("create secret manager client: %w", err)
	}

	pubsubHTTP := &http.Client{Transport: stackyardHeaderTransport{base: http.DefaultTransport, serviceName: "pubsub"}}
	publisherClient, err := pubsub.NewPublisherRESTClient(ctx,
		option.WithHTTPClient(pubsubHTTP),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		return nil, fmt.Errorf("create pubsub publisher client: %w", err)
	}
	subscriberClient, err := pubsub.NewSubscriberRESTClient(ctx,
		option.WithHTTPClient(pubsubHTTP),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		return nil, fmt.Errorf("create pubsub subscriber client: %w", err)
	}

	workflowsHTTP := &http.Client{Transport: stackyardHeaderTransport{base: http.DefaultTransport, serviceName: "workflows"}}
	workflowsClient, err := workflows.NewRESTClient(ctx,
		option.WithHTTPClient(workflowsHTTP),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		return nil, fmt.Errorf("create workflows client: %w", err)
	}

	executionsClient, err := executions.NewClient(ctx,
		option.WithEndpoint(grpcTarget),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	if err != nil {
		return nil, fmt.Errorf("create workflow executions client: %w", err)
	}

	return &app{
		tenants:          map[string]*tenantState{},
		endpoint:         endpoint,
		apiEndpoint:      apiEndpoint,
		grpcTarget:       grpcTarget,
		projectID:        projectID,
		locationID:       locationID,
		lakeBucket:       getenv("STACKYARD_DATA_LAKE_BUCKET", defaultLakeBucket),
		auditBucket:      getenv("STACKYARD_AUDIT_BUCKET", defaultAuditBucket),
		storageClient:    storageClient,
		kmsClient:        kmsClient,
		secretClient:     secretClient,
		publisherClient:  publisherClient,
		subscriberClient: subscriberClient,
		workflowsClient:  workflowsClient,
		executionsClient: executionsClient,
	}, nil
}

func (a *app) close() {
	if a.executionsClient != nil {
		if err := a.executionsClient.Close(); err != nil {
			log.Printf("close executions client: %v", err)
		}
	}
	if a.workflowsClient != nil {
		if err := a.workflowsClient.Close(); err != nil {
			log.Printf("close workflows client: %v", err)
		}
	}
	if a.publisherClient != nil {
		if err := a.publisherClient.Close(); err != nil {
			log.Printf("close pubsub publisher client: %v", err)
		}
	}
	if a.subscriberClient != nil {
		if err := a.subscriberClient.Close(); err != nil {
			log.Printf("close pubsub subscriber client: %v", err)
		}
	}
	if a.secretClient != nil {
		if err := a.secretClient.Close(); err != nil {
			log.Printf("close secret manager client: %v", err)
		}
	}
	if a.kmsClient != nil {
		if err := a.kmsClient.Close(); err != nil {
			log.Printf("close kms client: %v", err)
		}
	}
	if a.storageClient != nil {
		if err := a.storageClient.Close(); err != nil {
			log.Printf("close storage client: %v", err)
		}
	}
}

func (a *app) handleHealth(w http.ResponseWriter, _ *http.Request) {
	respondJSON(w, http.StatusOK, map[string]any{
		"status":      "ok",
		"endpoint":    a.apiEndpoint,
		"grpcTarget":  a.grpcTarget,
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

	respondJSON(w, http.StatusOK, map[string]any{"tenants": items})
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

	respondJSON(w, http.StatusOK, tenantDetails{tenantState: *st, ObjectsByZone: objects})
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
		subscriptionName, topicName, err := a.createTenantPubSubResources(ctx, tenantID)
		if err != nil {
			respondErr(w, http.StatusBadGateway, "ServiceError", err.Error())
			return
		}
		st.QueueURL = subscriptionName
		st.QueueArn = subscriptionName
		st.EventBusName = topicName
		st.EventBusArn = topicName
		st.RuleName = "pubsub-route"
		st.RuleArn = topicName
	}

	if st.StateMachineArn == "" {
		workflowName, err := a.ensureTenantWorkflow(ctx, tenantID)
		if err != nil {
			respondErr(w, http.StatusBadGateway, "ServiceError", err.Error())
			return
		}
		st.StateMachineArn = workflowName
	}

	st.UpdatedAtUTC = time.Now().UTC().Format(time.RFC3339)
	respondJSON(w, http.StatusOK, map[string]any{"message": "tenant bootstrap complete", "tenant": st})
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

	if err := a.writeObject(ctx, a.lakeBucket, key, body); err != nil {
		respondErr(w, http.StatusBadGateway, "ServiceError", fmt.Sprintf("put raw object: %v", err))
		return
	}

	if strings.TrimSpace(st.EventBusArn) != "" {
		_, publishErr := a.publisherClient.Publish(ctx, &pubsubpb.PublishRequest{
			Topic: st.EventBusArn,
			Messages: []*pubsubpb.PubsubMessage{{
				Data: []byte(fmt.Sprintf(`{"tenantId":"%s","rawObjectKey":"%s"}`, tenantID, key)),
			}},
		})
		if publishErr != nil {
			respondErr(w, http.StatusBadGateway, "ServiceError", fmt.Sprintf("publish pubsub message: %v", publishErr))
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

	if err := a.writeObject(ctx, a.lakeBucket, curatedKey, curatedBody); err != nil {
		respondErr(w, http.StatusBadGateway, "ServiceError", err.Error())
		return
	}
	if err := a.writeObject(ctx, a.lakeBucket, publishedKey, curatedBody); err != nil {
		respondErr(w, http.StatusBadGateway, "ServiceError", err.Error())
		return
	}

	execName, err := a.startWorkflowExecution(ctx, st.StateMachineArn, runID, map[string]any{
		"tenantId":           tenantID,
		"rawObjectKey":       st.LatestRawKey,
		"curatedObjectKey":   curatedKey,
		"publishedObjectKey": publishedKey,
	})
	if err != nil {
		respondErr(w, http.StatusBadGateway, "ServiceError", err.Error())
		return
	}

	if strings.TrimSpace(st.EventBusArn) != "" {
		_, _ = a.publisherClient.Publish(ctx, &pubsubpb.PublishRequest{
			Topic: st.EventBusArn,
			Messages: []*pubsubpb.PubsubMessage{{
				Data: []byte(fmt.Sprintf(`{"tenantId":"%s","runId":"%s","status":"SUCCEEDED"}`, tenantID, runID)),
			}},
		})
	}

	auditKey := fmt.Sprintf("runs/tenant=%s/date=%s/%s.json", tenantID, datePart, runID)
	auditPayload, _ := json.MarshalIndent(map[string]any{
		"tenantId":     tenantID,
		"runId":        runID,
		"executionArn": execName,
		"timestamp":    now.Format(time.RFC3339),
		"status":       "SUCCEEDED",
	}, "", "  ")
	if err := a.writeObject(ctx, a.auditBucket, auditKey, auditPayload); err != nil {
		respondErr(w, http.StatusBadGateway, "ServiceError", err.Error())
		return
	}

	run := runRecord{
		RunID:          runID,
		ExecutionARN:   execName,
		RawObjectKey:   st.LatestRawKey,
		CuratedKey:     curatedKey,
		PublishedKey:   publishedKey,
		StartedAtUTC:   now.Format(time.RFC3339),
		CompletedAtUTC: time.Now().UTC().Format(time.RFC3339),
		Status:         "SUCCEEDED",
	}
	st.Runs = append([]runRecord{run}, st.Runs...)
	st.UpdatedAtUTC = time.Now().UTC().Format(time.RFC3339)

	respondJSON(w, http.StatusOK, map[string]any{"message": "pipeline run completed", "run": run, "tenant": st})
}

func (a *app) ensureBucket(ctx context.Context, bucket string) error {
	err := a.storageClient.Bucket(bucket).Create(ctx, a.projectID, &storage.BucketAttrs{Location: "US", StorageClass: "STANDARD"})
	if err == nil {
		return nil
	}
	if isAlreadyExists(err) {
		return nil
	}
	return err
}

func (a *app) createTenantKMSKey(ctx context.Context, tenantID string) (string, string, error) {
	locationParent := fmt.Sprintf("projects/%s/locations/%s", a.projectID, a.locationID)
	keyRingID := sanitizeID("ra-dp-" + tenantID)
	keyID := sanitizeID("key-" + tenantID)
	keyRingName := fmt.Sprintf("%s/keyRings/%s", locationParent, keyRingID)
	cryptoKeyName := fmt.Sprintf("%s/cryptoKeys/%s", keyRingName, keyID)

	_, err := a.kmsClient.CreateKeyRing(ctx, &kmspb.CreateKeyRingRequest{
		Parent:    locationParent,
		KeyRingId: keyRingID,
		KeyRing:   &kmspb.KeyRing{},
	})
	if err != nil && !isAlreadyExists(err) {
		return "", "", err
	}

	_, err = a.kmsClient.CreateCryptoKey(ctx, &kmspb.CreateCryptoKeyRequest{
		Parent:      keyRingName,
		CryptoKeyId: keyID,
		CryptoKey:   &kmspb.CryptoKey{Purpose: kmspb.CryptoKey_ENCRYPT_DECRYPT},
	})
	if err != nil && !isAlreadyExists(err) {
		return "", "", err
	}

	return cryptoKeyName, cryptoKeyName, nil
}

func (a *app) createOrUpdateTenantSecret(ctx context.Context, tenantID string) (string, error) {
	secretID := sanitizeID("ra-data-pipeline-" + tenantID + "-db")
	parent := "projects/" + a.projectID
	secretName := parent + "/secrets/" + secretID

	_, err := a.secretClient.CreateSecret(ctx, &secretmanagerpb.CreateSecretRequest{
		Parent:   parent,
		SecretId: secretID,
		Secret: &secretmanagerpb.Secret{
			Replication: &secretmanagerpb.Replication{Replication: &secretmanagerpb.Replication_Automatic_{Automatic: &secretmanagerpb.Replication_Automatic{}}},
		},
	})
	if err != nil && !isAlreadyExists(err) {
		return "", err
	}

	secretJSON := fmt.Sprintf(`{"username":"tenant_%s","password":"stackyard-demo"}`, tenantID)
	_, err = a.secretClient.AddSecretVersion(ctx, &secretmanagerpb.AddSecretVersionRequest{
		Parent: secretName,
		Payload: &secretmanagerpb.SecretPayload{
			Data: []byte(secretJSON),
		},
	})
	if err != nil {
		return "", err
	}
	return secretName, nil
}

func (a *app) createTenantPubSubResources(ctx context.Context, tenantID string) (string, string, error) {
	projectName := "projects/" + a.projectID
	topicID := sanitizeID("ra-data-pipeline-" + tenantID)
	subID := sanitizeID("ra-data-pipeline-" + tenantID + "-sub")
	topicName := projectName + "/topics/" + topicID
	subscriptionName := projectName + "/subscriptions/" + subID

	_, err := a.publisherClient.CreateTopic(ctx, &pubsubpb.Topic{Name: topicName})
	if err != nil && !isAlreadyExists(err) {
		return "", "", err
	}

	_, err = a.subscriberClient.CreateSubscription(ctx, &pubsubpb.Subscription{Name: subscriptionName, Topic: topicName})
	if err != nil && !isAlreadyExists(err) {
		return "", "", err
	}

	return subscriptionName, topicName, nil
}

func (a *app) ensureTenantWorkflow(ctx context.Context, tenantID string) (string, error) {
	parent := fmt.Sprintf("projects/%s/locations/%s", a.projectID, a.locationID)
	workflowID := sanitizeID("ra-pipeline-" + tenantID)
	workflowName := parent + "/workflows/" + workflowID

	_, err := a.workflowsClient.GetWorkflow(ctx, &workflowspb.GetWorkflowRequest{Name: workflowName})
	if err == nil {
		return workflowName, nil
	}
	if !isNotFound(err) {
		return "", err
	}

	_, err = a.workflowsClient.CreateWorkflow(ctx, &workflowspb.CreateWorkflowRequest{
		Parent:     parent,
		WorkflowId: workflowID,
		Workflow: &workflowspb.Workflow{
			Name:        workflowName,
			Description: "Reference architecture pipeline",
			SourceCode:  &workflowspb.Workflow_SourceContents{SourceContents: "main:\n  params: [input]\n  steps:\n  - done:\n      return: ${input}"},
		},
	})
	if err != nil && !isAlreadyExists(err) {
		return "", err
	}
	return workflowName, nil
}

func (a *app) startWorkflowExecution(ctx context.Context, workflowName, runID string, input map[string]any) (string, error) {
	if strings.TrimSpace(workflowName) == "" {
		return "", errors.New("workflow name is empty")
	}
	inputJSON, _ := json.Marshal(input)
	resp, err := a.executionsClient.CreateExecution(ctx, &executionspb.CreateExecutionRequest{
		Parent: workflowName,
		Execution: &executionspb.Execution{
			Argument: string(inputJSON),
			Labels:   map[string]string{"run_id": sanitizeLabel(runID)},
		},
	})
	if err != nil {
		return "", err
	}
	return resp.GetName(), nil
}

func (a *app) writeObject(ctx context.Context, bucket, key string, payload []byte) error {
	writer := a.storageClient.Bucket(bucket).Object(key).NewWriter(ctx)
	writer.ContentType = "application/json"
	if _, err := writer.Write(payload); err != nil {
		_ = writer.Close()
		return fmt.Errorf("put object %s/%s: %w", bucket, key, err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("close object writer %s/%s: %w", bucket, key, err)
	}
	return nil
}

func (a *app) readObject(ctx context.Context, bucket, key string) ([]byte, error) {
	reader, err := a.storageClient.Bucket(bucket).Object(key).NewReader(ctx)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}

func (a *app) listObjectsByPrefix(ctx context.Context, bucket, prefix string) ([]string, error) {
	it := a.storageClient.Bucket(bucket).Objects(ctx, &storage.Query{Prefix: prefix})
	keys := make([]string, 0)
	for {
		attrs, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		keys = append(keys, attrs.Name)
	}
	sort.Strings(keys)
	return keys, nil
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

func sanitizeID(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	builder := strings.Builder{}
	for _, r := range raw {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			builder.WriteRune(r)
		} else {
			builder.WriteRune('-')
		}
	}
	out := strings.Trim(builder.String(), "-")
	if out == "" {
		return "stackyard"
	}
	if len(out) > 63 {
		out = out[:63]
	}
	return out
}

func sanitizeLabel(raw string) string {
	out := sanitizeID(raw)
	if len(out) > 63 {
		out = out[:63]
	}
	if out == "" {
		return "run"
	}
	return out
}

func normalizeGRPCTarget(raw string) string {
	value := strings.TrimSpace(raw)
	value = strings.TrimPrefix(value, "http://")
	value = strings.TrimPrefix(value, "https://")
	if idx := strings.Index(value, "/"); idx >= 0 {
		value = value[:idx]
	}
	return value
}

func isAlreadyExists(err error) bool {
	if err == nil {
		return false
	}
	if s, ok := status.FromError(err); ok && s.Code() == codes.AlreadyExists {
		return true
	}
	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) {
		return apiErr.Code == http.StatusConflict
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "alreadyexists") || strings.Contains(msg, "already exists") || strings.Contains(msg, "resourceexistexception")
}

func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	if s, ok := status.FromError(err); ok && s.Code() == codes.NotFound {
		return true
	}
	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) {
		return apiErr.Code == http.StatusNotFound
	}
	return strings.Contains(strings.ToLower(err.Error()), "not found")
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

func getenv(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

type stackyardHeaderTransport struct {
	base        http.RoundTripper
	serviceName string
}

func (t stackyardHeaderTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	clone := req.Clone(req.Context())
	clone.Header.Set("X-Stackyard-GCP-Service", t.serviceName)
	return base.RoundTrip(clone)
}
