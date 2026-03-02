package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

const (
	defaultRegion       = "us-east-1"
	defaultAccountID    = "123456789012"
	defaultDomainPrefix = "ra-data-mesh"
)

type serviceError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message,omitempty"`
}

type producerInfo struct {
	Domain  string `json:"domain"`
	Service string `json:"service"`
}

type eventEnvelope struct {
	EventID       string                 `json:"event_id"`
	EventType     string                 `json:"event_type"`
	SchemaVersion string                 `json:"schema_version"`
	OccurredAt    string                 `json:"occurred_at"`
	Producer      producerInfo           `json:"producer"`
	TenantID      string                 `json:"tenant_id"`
	CorrelationID string                 `json:"correlation_id"`
	Payload       map[string]any         `json:"payload"`
	Poison        bool                   `json:"poison,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

type publishEventRequest struct {
	EventID       string         `json:"event_id"`
	EventType     string         `json:"event_type"`
	SchemaVersion string         `json:"schema_version"`
	OccurredAt    string         `json:"occurred_at"`
	CorrelationID string         `json:"correlation_id"`
	Producer      *producerInfo  `json:"producer"`
	Payload       map[string]any `json:"payload"`
	Poison        bool           `json:"poison"`
}

type domainMetrics struct {
	Published          int64 `json:"published"`
	Projected          int64 `json:"projected"`
	Duplicates         int64 `json:"duplicates"`
	ProjectionFailures int64 `json:"projection_failures"`
	DLQTotal           int64 `json:"dlq_total"`
	DLQDepth           int64 `json:"dlq_depth"`
	Replayed           int64 `json:"replayed"`
}

type tenantState struct {
	TenantID   string `json:"tenant_id"`
	KMSKeyID   string `json:"kms_key_id"`
	KMSKeyARN  string `json:"kms_key_arn"`
	UpdatedAt  string `json:"updated_at"`
	DomainName string `json:"domain"`
}

type domainState struct {
	Domain         string                  `json:"domain"`
	StreamName     string                  `json:"stream_name"`
	ReadModelTable string                  `json:"read_model_table"`
	ProcessedTable string                  `json:"processed_events_table"`
	DLQName        string                  `json:"dlq_name"`
	DLQURL         string                  `json:"dlq_url"`
	DLQARN         string                  `json:"dlq_arn"`
	Tenants        map[string]*tenantState `json:"tenants"`
	Metrics        domainMetrics           `json:"metrics"`
	UpdatedAt      string                  `json:"updated_at"`
}

type app struct {
	mu sync.Mutex

	endpoint     string
	region       string
	accountID    string
	domainPrefix string

	eventSeq uint64
	domains  map[string]*domainState
	pending  map[string][]eventEnvelope

	awsCreds aws.CredentialsProvider
	signer   *v4.Signer
	httpc    *http.Client

	kinesisClient *kinesis.Client
	dynamoClient  *dynamodb.Client
	sqsClient     *sqs.Client
	kmsClient     *kms.Client
}

func main() {
	ctx := context.Background()
	instance, err := newApp(ctx)
	if err != nil {
		log.Fatalf("initialize app: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", instance.handleHealth)
	mux.HandleFunc("/api/v1/summary", instance.handleSummary)
	mux.HandleFunc("/api/v1/domains/", instance.handleDomains)

	addr := getenv("APP_ADDR", ":8080")
	log.Printf("data-mesh backend listening on %s", addr)
	if err := http.ListenAndServe(addr, loggingMiddleware(mux)); err != nil {
		log.Fatalf("listen: %v", err)
	}
}

func newApp(ctx context.Context) (*app, error) {
	endpoint := strings.TrimRight(getenv("STACKYARD_ENDPOINT", "http://localhost:4566"), "/")
	region := getenv("AWS_REGION", defaultRegion)
	accountID := getenv("AWS_ACCOUNT_ID", defaultAccountID)
	accessKey := getenv("AWS_ACCESS_KEY_ID", "stackyard")
	secretKey := getenv("AWS_SECRET_ACCESS_KEY", "stackyard")
	domainPrefix := getenv("DATA_MESH_DOMAIN_PREFIX", defaultDomainPrefix)

	creds := credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")
	resolver := aws.EndpointResolverWithOptionsFunc(func(service, r string, _ ...any) (aws.Endpoint, error) {
		switch service {
		case dynamodb.ServiceID, kinesis.ServiceID, sqs.ServiceID, kms.ServiceID:
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

	return &app{
		endpoint:      endpoint,
		region:        region,
		accountID:     accountID,
		domainPrefix:  strings.ToLower(strings.TrimSpace(domainPrefix)),
		domains:       map[string]*domainState{},
		pending:       map[string][]eventEnvelope{},
		awsCreds:      creds,
		signer:        v4.NewSigner(),
		httpc:         &http.Client{Timeout: 15 * time.Second},
		kinesisClient: kinesis.NewFromConfig(cfg),
		dynamoClient:  dynamodb.NewFromConfig(cfg),
		sqsClient:     sqs.NewFromConfig(cfg),
		kmsClient:     kms.NewFromConfig(cfg),
	}, nil
}

func (a *app) handleHealth(w http.ResponseWriter, _ *http.Request) {
	respondJSON(w, http.StatusOK, map[string]any{
		"status":        "ok",
		"stackyard":     a.endpoint,
		"region":        a.region,
		"domain_prefix": a.domainPrefix,
	})
}

func (a *app) handleSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondErr(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	domains := make([]map[string]any, 0, len(a.domains))
	for _, domain := range a.domains {
		domains = append(domains, map[string]any{
			"domain":              domain.Domain,
			"stream_name":         domain.StreamName,
			"read_model_table":    domain.ReadModelTable,
			"processed_table":     domain.ProcessedTable,
			"dlq_name":            domain.DLQName,
			"tenant_count":        len(domain.Tenants),
			"pending_event_count": len(a.pending[domain.Domain]),
			"metrics":             domain.Metrics,
			"updated_at":          domain.UpdatedAt,
		})
	}
	sort.Slice(domains, func(i, j int) bool {
		return stringValue(domains[i]["domain"]) < stringValue(domains[j]["domain"])
	})

	respondJSON(w, http.StatusOK, map[string]any{
		"domains": domains,
	})
}

func (a *app) handleDomains(w http.ResponseWriter, r *http.Request) {
	parts := splitPath(strings.TrimPrefix(r.URL.Path, "/api/v1/domains/"))
	if len(parts) == 0 {
		respondErr(w, http.StatusBadRequest, "ValidationException", "missing domain")
		return
	}

	domain := normalizeDomain(parts[0])
	if !isValidName(domain, 3, 48) {
		respondErr(w, http.StatusBadRequest, "ValidationException", "domain must match [a-z0-9-]{3,48}")
		return
	}

	if len(parts) == 1 {
		if r.Method == http.MethodPost {
			a.bootstrapDomain(r.Context(), w, domain)
			return
		}
		respondErr(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		return
	}

	if r.Method == http.MethodPost && parts[1] == "bootstrap" {
		a.bootstrapDomain(r.Context(), w, domain)
		return
	}
	if r.Method == http.MethodPost && parts[1] == "project" {
		a.projectDomain(r.Context(), w, r, domain)
		return
	}
	if r.Method == http.MethodPost && parts[1] == "replay-dlq" {
		a.replayDLQ(r.Context(), w, r, domain)
		return
	}

	if len(parts) >= 4 && parts[1] == "tenants" && r.Method == http.MethodPost && parts[3] == "events" {
		tenantID := normalizeTenant(parts[2])
		if !isValidName(tenantID, 3, 48) {
			respondErr(w, http.StatusBadRequest, "ValidationException", "tenant id must match [a-z0-9-]{3,48}")
			return
		}
		a.publishEvent(r.Context(), w, r, domain, tenantID)
		return
	}

	if len(parts) >= 4 && parts[1] == "products" && parts[2] == "orders" && r.Method == http.MethodGet {
		tenantID := normalizeTenant(parts[3])
		if !isValidName(tenantID, 3, 48) {
			respondErr(w, http.StatusBadRequest, "ValidationException", "tenant id must match [a-z0-9-]{3,48}")
			return
		}
		a.getOrderProduct(w, r, domain, tenantID)
		return
	}

	respondErr(w, http.StatusNotFound, "ResourceNotFoundException", "unknown route")
}

func (a *app) bootstrapDomain(ctx context.Context, w http.ResponseWriter, domain string) {
	state, err := a.ensureDomainInfrastructure(ctx, domain)
	if err != nil {
		respondErr(w, http.StatusBadGateway, "ServiceError", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"message": "domain bootstrap complete",
		"domain":  cloneDomain(state),
	})
}

func (a *app) publishEvent(ctx context.Context, w http.ResponseWriter, r *http.Request, domain, tenantID string) {
	state, err := a.ensureDomainInfrastructure(ctx, domain)
	if err != nil {
		respondErr(w, http.StatusBadGateway, "ServiceError", err.Error())
		return
	}

	req := publishEventRequest{}
	if err := decodeJSON(r, &req); err != nil {
		respondErr(w, http.StatusBadRequest, "ValidationException", err.Error())
		return
	}
	if strings.TrimSpace(req.EventType) == "" {
		respondErr(w, http.StatusBadRequest, "ValidationException", "event_type is required")
		return
	}
	if req.Payload == nil {
		req.Payload = map[string]any{}
	}

	tenant, err := a.ensureTenantKey(ctx, state, tenantID)
	if err != nil {
		respondErr(w, http.StatusBadGateway, "ServiceError", err.Error())
		return
	}

	occurredAt := strings.TrimSpace(req.OccurredAt)
	if occurredAt == "" {
		occurredAt = time.Now().UTC().Format(time.RFC3339)
	}
	env := eventEnvelope{
		EventID:       firstNonEmpty(strings.TrimSpace(req.EventID), a.nextEventID()),
		EventType:     strings.TrimSpace(req.EventType),
		SchemaVersion: firstNonEmpty(strings.TrimSpace(req.SchemaVersion), "v1"),
		OccurredAt:    occurredAt,
		Producer: producerInfo{
			Domain:  domain,
			Service: "data-mesh-api",
		},
		TenantID:      tenantID,
		CorrelationID: firstNonEmpty(strings.TrimSpace(req.CorrelationID), a.nextEventID()),
		Payload:       req.Payload,
		Poison:        req.Poison,
		Metadata: map[string]interface{}{
			"kms_key_arn": tenant.KMSKeyARN,
		},
	}
	if req.Producer != nil {
		env.Producer.Domain = firstNonEmpty(strings.TrimSpace(req.Producer.Domain), env.Producer.Domain)
		env.Producer.Service = firstNonEmpty(strings.TrimSpace(req.Producer.Service), env.Producer.Service)
	}

	data, _ := json.Marshal(env)
	putOut, err := a.kinesisClient.PutRecord(ctx, &kinesis.PutRecordInput{
		StreamName:   aws.String(state.StreamName),
		PartitionKey: aws.String(tenantID),
		Data:         data,
	})
	if err != nil {
		respondErr(w, http.StatusBadGateway, "ServiceError", fmt.Sprintf("put kinesis record: %v", err))
		return
	}

	a.mu.Lock()
	a.pending[domain] = append(a.pending[domain], env)
	state.Metrics.Published++
	state.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	pendingCount := len(a.pending[domain])
	a.mu.Unlock()

	respondJSON(w, http.StatusAccepted, map[string]any{
		"message":             "event accepted",
		"domain":              domain,
		"tenant_id":           tenantID,
		"sequence_number":     aws.ToString(putOut.SequenceNumber),
		"stream_name":         state.StreamName,
		"pending_event_count": pendingCount,
		"event":               env,
	})
}

func (a *app) projectDomain(ctx context.Context, w http.ResponseWriter, r *http.Request, domain string) {
	state, err := a.ensureDomainInfrastructure(ctx, domain)
	if err != nil {
		respondErr(w, http.StatusBadGateway, "ServiceError", err.Error())
		return
	}

	request := map[string]any{}
	_ = decodeJSON(r, &request)
	limit := intValue(request["limit"], 25)
	if limit < 1 {
		limit = 1
	}
	if limit > 250 {
		limit = 250
	}

	batch := a.popPending(domain, limit)
	results := map[string]int{
		"received":   len(batch),
		"projected":  0,
		"duplicates": 0,
		"dlq":        0,
		"failed":     0,
	}
	failures := make([]map[string]string, 0)

	for _, evt := range batch {
		if evt.Poison {
			if dlqErr := a.sendEventToDLQ(ctx, state, evt, "poison_event_flag"); dlqErr != nil {
				results["failed"]++
				failures = append(failures, map[string]string{"event_id": evt.EventID, "reason": dlqErr.Error()})
			} else {
				results["dlq"]++
			}
			continue
		}

		processed, checkErr := a.isEventProcessed(ctx, state, evt.EventID)
		if checkErr != nil {
			results["failed"]++
			failures = append(failures, map[string]string{"event_id": evt.EventID, "reason": checkErr.Error()})
			continue
		}
		if processed {
			results["duplicates"]++
			continue
		}

		if err := a.projectToReadModel(ctx, state, evt); err != nil {
			if dlqErr := a.sendEventToDLQ(ctx, state, evt, "projection_failed: "+err.Error()); dlqErr != nil {
				results["failed"]++
				failures = append(failures, map[string]string{"event_id": evt.EventID, "reason": dlqErr.Error()})
			} else {
				results["dlq"]++
			}
			continue
		}

		if err := a.markEventProcessed(ctx, state, evt.EventID); err != nil {
			results["failed"]++
			failures = append(failures, map[string]string{"event_id": evt.EventID, "reason": err.Error()})
			continue
		}

		results["projected"]++
	}

	a.mu.Lock()
	state.Metrics.Projected += int64(results["projected"])
	state.Metrics.Duplicates += int64(results["duplicates"])
	state.Metrics.ProjectionFailures += int64(results["failed"])
	state.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	remaining := len(a.pending[domain])
	metrics := state.Metrics
	a.mu.Unlock()

	respondJSON(w, http.StatusOK, map[string]any{
		"message":                "projection cycle complete",
		"domain":                 domain,
		"results":                results,
		"failures":               failures,
		"pending_event_count":    remaining,
		"domain_runtime_metrics": metrics,
	})
}

func (a *app) replayDLQ(ctx context.Context, w http.ResponseWriter, r *http.Request, domain string) {
	state, err := a.ensureDomainInfrastructure(ctx, domain)
	if err != nil {
		respondErr(w, http.StatusBadGateway, "ServiceError", err.Error())
		return
	}

	request := map[string]any{}
	_ = decodeJSON(r, &request)
	maxMessages := int32(intValue(request["max_messages"], 10))
	if maxMessages < 1 {
		maxMessages = 1
	}
	if maxMessages > 10 {
		maxMessages = 10
	}

	resp, err := a.sqsClient.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(state.DLQURL),
		MaxNumberOfMessages: maxMessages,
		VisibilityTimeout:   30,
	})
	if err != nil {
		respondErr(w, http.StatusBadGateway, "ServiceError", fmt.Sprintf("receive from dlq: %v", err))
		return
	}

	replayed := 0
	failed := 0
	for _, msg := range resp.Messages {
		payload := map[string]any{}
		if unmarshalErr := json.Unmarshal([]byte(aws.ToString(msg.Body)), &payload); unmarshalErr != nil {
			failed++
			continue
		}
		rawEvent, ok := payload["event"]
		if !ok {
			failed++
			continue
		}
		evtBytes, _ := json.Marshal(rawEvent)
		evt := eventEnvelope{}
		if unmarshalErr := json.Unmarshal(evtBytes, &evt); unmarshalErr != nil {
			failed++
			continue
		}
		evt.Poison = false
		a.mu.Lock()
		a.pending[domain] = append(a.pending[domain], evt)
		state.Metrics.Replayed++
		if state.Metrics.DLQDepth > 0 {
			state.Metrics.DLQDepth--
		}
		state.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		a.mu.Unlock()
		replayed++

		_, delErr := a.sqsClient.DeleteMessage(ctx, &sqs.DeleteMessageInput{
			QueueUrl:      aws.String(state.DLQURL),
			ReceiptHandle: msg.ReceiptHandle,
		})
		if delErr != nil {
			failed++
		}
	}

	a.mu.Lock()
	pendingCount := len(a.pending[domain])
	metrics := state.Metrics
	a.mu.Unlock()

	respondJSON(w, http.StatusOK, map[string]any{
		"message":                "dlq replay attempted",
		"domain":                 domain,
		"replayed":               replayed,
		"failed":                 failed,
		"pending_event_count":    pendingCount,
		"domain_runtime_metrics": metrics,
	})
}

func (a *app) getOrderProduct(w http.ResponseWriter, r *http.Request, domain, tenantID string) {
	state := a.getDomain(domain)
	if state == nil {
		respondErr(w, http.StatusNotFound, "ResourceNotFoundException", "domain is not bootstrapped")
		return
	}

	claimTenant := normalizeTenant(r.Header.Get("X-Claim-Tenant-Id"))
	if claimTenant == "" || claimTenant != tenantID {
		respondErr(w, http.StatusForbidden, "AccessDeniedException", "tenant claim mismatch")
		return
	}

	scopes := parseScopes(r.Header.Get("X-Claim-Scopes"))
	accessMode := "none"
	if scopes["read:full"] {
		accessMode = "full"
	} else if scopes["read:masked"] {
		accessMode = "masked"
	}

	queryOut, err := a.dynamoClient.Query(r.Context(), &dynamodb.QueryInput{
		TableName:              aws.String(state.ReadModelTable),
		KeyConditionExpression: aws.String("tenant_id = :tenant"),
		ExpressionAttributeValues: map[string]dynamodbtypes.AttributeValue{
			":tenant": &dynamodbtypes.AttributeValueMemberS{Value: tenantID},
		},
	})
	if err != nil {
		respondErr(w, http.StatusBadGateway, "ServiceError", fmt.Sprintf("query read model: %v", err))
		return
	}

	records := make([]map[string]any, 0, len(queryOut.Items))
	for _, item := range queryOut.Items {
		rec := map[string]any{
			"entity_id":      avString(item, "entity_id"),
			"event_id":       avString(item, "event_id"),
			"event_type":     avString(item, "event_type"),
			"occurred_at":    avString(item, "occurred_at"),
			"schema_version": avString(item, "schema_version"),
		}
		if amount := avNumber(item, "amount"); amount != nil {
			rec["amount"] = *amount
		}
		switch accessMode {
		case "full":
			rec["customer_email"] = pseudoDecrypt(tenantID, avString(item, "customer_email_ciphertext"))
		case "masked":
			rec["customer_email"] = avString(item, "customer_email_masked")
		}
		records = append(records, rec)
	}
	sort.Slice(records, func(i, j int) bool {
		return stringValue(records[i]["entity_id"]) < stringValue(records[j]["entity_id"])
	})

	respondJSON(w, http.StatusOK, map[string]any{
		"domain":      domain,
		"tenant_id":   tenantID,
		"access_mode": accessMode,
		"count":       len(records),
		"records":     records,
	})
}

func (a *app) ensureDomainInfrastructure(ctx context.Context, domain string) (*domainState, error) {
	a.mu.Lock()
	state := a.domains[domain]
	if state != nil {
		a.mu.Unlock()
		return state, nil
	}

	state = &domainState{
		Domain:         domain,
		StreamName:     fmt.Sprintf("%s-%s-ingress", a.domainPrefix, domain),
		ReadModelTable: fmt.Sprintf("%s-%s-read-model", a.domainPrefix, domain),
		ProcessedTable: fmt.Sprintf("%s-%s-processed-events", a.domainPrefix, domain),
		DLQName:        fmt.Sprintf("%s-%s-dlq", a.domainPrefix, domain),
		Tenants:        map[string]*tenantState{},
		UpdatedAt:      time.Now().UTC().Format(time.RFC3339),
	}
	a.domains[domain] = state
	a.mu.Unlock()

	if err := a.createKinesisStreamIfNeeded(ctx, state.StreamName); err != nil {
		return nil, err
	}
	if err := a.createReadModelTableIfNeeded(ctx, state.ReadModelTable); err != nil {
		return nil, err
	}
	if err := a.createProcessedEventsTableIfNeeded(ctx, state.ProcessedTable); err != nil {
		return nil, err
	}
	dlqURL, err := a.createQueueIfNeeded(ctx, state.DLQName)
	if err != nil {
		return nil, err
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	state.DLQURL = dlqURL
	state.DLQARN = fmt.Sprintf("arn:aws:sqs:%s:%s:%s", a.region, a.accountID, state.DLQName)
	state.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return state, nil
}

func (a *app) ensureTenantKey(ctx context.Context, state *domainState, tenantID string) (*tenantState, error) {
	a.mu.Lock()
	if tenant := state.Tenants[tenantID]; tenant != nil {
		a.mu.Unlock()
		return tenant, nil
	}
	a.mu.Unlock()

	keyID, keyARN, err := a.createTenantKMSKeyRaw(ctx, state.Domain, tenantID)
	if err != nil {
		return nil, fmt.Errorf("create kms key: %w", err)
	}

	key := &tenantState{
		TenantID:   tenantID,
		DomainName: state.Domain,
		KMSKeyID:   keyID,
		KMSKeyARN:  keyARN,
		UpdatedAt:  time.Now().UTC().Format(time.RFC3339),
	}

	a.mu.Lock()
	state.Tenants[tenantID] = key
	state.UpdatedAt = key.UpdatedAt
	a.mu.Unlock()

	return key, nil
}

func (a *app) createTenantKMSKeyRaw(ctx context.Context, domain, tenantID string) (string, string, error) {
	payload := map[string]any{
		"Description": fmt.Sprintf("Reference architecture key for domain %s tenant %s", domain, tenantID),
		"KeyUsage":    "ENCRYPT_DECRYPT",
		"Tags": []map[string]string{
			{"TagKey": "domain", "TagValue": domain},
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
	if strings.TrimSpace(out.KeyMetadata.Arn) == "" || strings.TrimSpace(out.KeyMetadata.KeyID) == "" {
		return "", "", errors.New("kms create key returned incomplete metadata")
	}
	return out.KeyMetadata.KeyID, out.KeyMetadata.Arn, nil
}

func (a *app) popPending(domain string, limit int) []eventEnvelope {
	a.mu.Lock()
	defer a.mu.Unlock()
	queue := a.pending[domain]
	if len(queue) == 0 {
		return nil
	}
	if limit > len(queue) {
		limit = len(queue)
	}
	batch := append([]eventEnvelope(nil), queue[:limit]...)
	a.pending[domain] = append([]eventEnvelope(nil), queue[limit:]...)
	return batch
}

func (a *app) isEventProcessed(ctx context.Context, state *domainState, eventID string) (bool, error) {
	out, err := a.dynamoClient.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(state.ProcessedTable),
		Key: map[string]dynamodbtypes.AttributeValue{
			"event_id": &dynamodbtypes.AttributeValueMemberS{Value: eventID},
		},
	})
	if err != nil {
		return false, fmt.Errorf("check processed event: %w", err)
	}
	return len(out.Item) > 0, nil
}

func (a *app) markEventProcessed(ctx context.Context, state *domainState, eventID string) error {
	ttl := strconv.FormatInt(time.Now().UTC().Add(7*24*time.Hour).Unix(), 10)
	_, err := a.dynamoClient.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(state.ProcessedTable),
		Item: map[string]dynamodbtypes.AttributeValue{
			"event_id":     &dynamodbtypes.AttributeValueMemberS{Value: eventID},
			"processed_at": &dynamodbtypes.AttributeValueMemberS{Value: time.Now().UTC().Format(time.RFC3339)},
			"expires_at":   &dynamodbtypes.AttributeValueMemberN{Value: ttl},
		},
	})
	if err != nil {
		return fmt.Errorf("mark processed event: %w", err)
	}
	return nil
}

func (a *app) projectToReadModel(ctx context.Context, state *domainState, evt eventEnvelope) error {
	tenant, err := a.ensureTenantKey(ctx, state, evt.TenantID)
	if err != nil {
		return err
	}

	entityID := firstNonEmpty(
		strings.TrimSpace(stringValue(evt.Payload["entity_id"])),
		strings.TrimSpace(stringValue(evt.Payload["order_id"])),
		evt.EventID,
	)
	customerEmail := strings.TrimSpace(stringValue(evt.Payload["customer_email"]))
	payloadJSON, _ := json.Marshal(evt.Payload)

	item := map[string]dynamodbtypes.AttributeValue{
		"tenant_id":                 &dynamodbtypes.AttributeValueMemberS{Value: evt.TenantID},
		"entity_id":                 &dynamodbtypes.AttributeValueMemberS{Value: entityID},
		"event_id":                  &dynamodbtypes.AttributeValueMemberS{Value: evt.EventID},
		"event_type":                &dynamodbtypes.AttributeValueMemberS{Value: evt.EventType},
		"schema_version":            &dynamodbtypes.AttributeValueMemberS{Value: evt.SchemaVersion},
		"occurred_at":               &dynamodbtypes.AttributeValueMemberS{Value: evt.OccurredAt},
		"correlation_id":            &dynamodbtypes.AttributeValueMemberS{Value: evt.CorrelationID},
		"producer_domain":           &dynamodbtypes.AttributeValueMemberS{Value: evt.Producer.Domain},
		"producer_service":          &dynamodbtypes.AttributeValueMemberS{Value: evt.Producer.Service},
		"payload_json":              &dynamodbtypes.AttributeValueMemberS{Value: string(payloadJSON)},
		"customer_email_ciphertext": &dynamodbtypes.AttributeValueMemberS{Value: pseudoEncrypt(evt.TenantID, customerEmail)},
		"customer_email_masked":     &dynamodbtypes.AttributeValueMemberS{Value: maskEmail(customerEmail)},
		"kms_key_arn":               &dynamodbtypes.AttributeValueMemberS{Value: tenant.KMSKeyARN},
		"updated_at":                &dynamodbtypes.AttributeValueMemberS{Value: time.Now().UTC().Format(time.RFC3339)},
	}

	if amount := numericValue(evt.Payload["amount"]); amount != nil {
		item["amount"] = &dynamodbtypes.AttributeValueMemberN{Value: strconv.FormatFloat(*amount, 'f', -1, 64)}
	}

	_, err = a.dynamoClient.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(state.ReadModelTable),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("put read-model item: %w", err)
	}
	return nil
}

func (a *app) sendEventToDLQ(ctx context.Context, state *domainState, evt eventEnvelope, reason string) error {
	body, _ := json.Marshal(map[string]any{
		"reason":      reason,
		"failed_at":   time.Now().UTC().Format(time.RFC3339),
		"domain":      state.Domain,
		"stream_name": state.StreamName,
		"event":       evt,
	})
	_, err := a.sqsClient.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    aws.String(state.DLQURL),
		MessageBody: aws.String(string(body)),
	})
	if err != nil {
		return fmt.Errorf("send to dlq: %w", err)
	}

	a.mu.Lock()
	state.Metrics.DLQTotal++
	state.Metrics.DLQDepth++
	state.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	a.mu.Unlock()
	return nil
}

func (a *app) createKinesisStreamIfNeeded(ctx context.Context, streamName string) error {
	_, err := a.kinesisClient.CreateStream(ctx, &kinesis.CreateStreamInput{
		StreamName: aws.String(streamName),
		ShardCount: aws.Int32(1),
	})
	if err == nil {
		return nil
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "resourceinuseexception") || strings.Contains(msg, "already exists") {
		return nil
	}
	return fmt.Errorf("create kinesis stream %s: %w", streamName, err)
}

func (a *app) createReadModelTableIfNeeded(ctx context.Context, tableName string) error {
	_, err := a.dynamoClient.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName: aws.String(tableName),
		AttributeDefinitions: []dynamodbtypes.AttributeDefinition{
			{AttributeName: aws.String("tenant_id"), AttributeType: dynamodbtypes.ScalarAttributeTypeS},
			{AttributeName: aws.String("entity_id"), AttributeType: dynamodbtypes.ScalarAttributeTypeS},
		},
		KeySchema: []dynamodbtypes.KeySchemaElement{
			{AttributeName: aws.String("tenant_id"), KeyType: dynamodbtypes.KeyTypeHash},
			{AttributeName: aws.String("entity_id"), KeyType: dynamodbtypes.KeyTypeRange},
		},
		BillingMode: dynamodbtypes.BillingModePayPerRequest,
	})
	if err == nil {
		return nil
	}
	if isDynamoDBCreateTableDateDecodeError(err) {
		return nil
	}
	if isAlreadyExists(err) {
		return nil
	}
	return fmt.Errorf("create read model table %s: %w", tableName, err)
}

func (a *app) createProcessedEventsTableIfNeeded(ctx context.Context, tableName string) error {
	_, err := a.dynamoClient.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName: aws.String(tableName),
		AttributeDefinitions: []dynamodbtypes.AttributeDefinition{
			{AttributeName: aws.String("event_id"), AttributeType: dynamodbtypes.ScalarAttributeTypeS},
		},
		KeySchema: []dynamodbtypes.KeySchemaElement{
			{AttributeName: aws.String("event_id"), KeyType: dynamodbtypes.KeyTypeHash},
		},
		BillingMode: dynamodbtypes.BillingModePayPerRequest,
	})
	if err == nil {
		return nil
	}
	if isDynamoDBCreateTableDateDecodeError(err) {
		return nil
	}
	if isAlreadyExists(err) {
		return nil
	}
	return fmt.Errorf("create idempotency table %s: %w", tableName, err)
}

func (a *app) createQueueIfNeeded(ctx context.Context, queueName string) (string, error) {
	out, err := a.sqsClient.CreateQueue(ctx, &sqs.CreateQueueInput{
		QueueName: aws.String(queueName),
	})
	if err != nil {
		return "", fmt.Errorf("create queue %s: %w", queueName, err)
	}
	return aws.ToString(out.QueueUrl), nil
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

func (a *app) getDomain(domain string) *domainState {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.domains[domain]
}

func cloneDomain(in *domainState) *domainState {
	if in == nil {
		return nil
	}
	copy := *in
	copy.Tenants = map[string]*tenantState{}
	for k, v := range in.Tenants {
		t := *v
		copy.Tenants[k] = &t
	}
	return &copy
}

func parseScopes(raw string) map[string]bool {
	raw = strings.TrimSpace(raw)
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ' ' || r == ';'
	})
	out := map[string]bool{}
	for _, p := range parts {
		p = strings.ToLower(strings.TrimSpace(p))
		if p != "" {
			out[p] = true
		}
	}
	return out
}

func decodeJSON(r *http.Request, out any) error {
	if r.Body == nil {
		return nil
	}
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		if errors.Is(err, http.ErrBodyReadAfterClose) {
			return nil
		}
		if strings.Contains(strings.ToLower(err.Error()), "eof") {
			return nil
		}
		return fmt.Errorf("invalid json payload: %w", err)
	}
	return nil
}

func splitPath(path string) []string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func isValidName(v string, minLen, maxLen int) bool {
	if len(v) < minLen || len(v) > maxLen {
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

func normalizeDomain(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}

func normalizeTenant(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}

func isAlreadyExists(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "resourceinuseexception") ||
		strings.Contains(msg, "already exists") ||
		strings.Contains(msg, "table already exists")
}

func isDynamoDBCreateTableDateDecodeError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "dynamodb: createtable") &&
		strings.Contains(msg, "deserialization failed") &&
		strings.Contains(msg, "expected date to be a json number")
}

func (a *app) nextEventID() string {
	n := atomic.AddUint64(&a.eventSeq, 1)
	return fmt.Sprintf("evt-%d-%06d", time.Now().UTC().UnixMilli(), n)
}

func pseudoEncrypt(tenantID, plaintext string) string {
	if strings.TrimSpace(plaintext) == "" {
		return ""
	}
	value := tenantID + "|" + plaintext
	return base64.StdEncoding.EncodeToString([]byte(value))
}

func pseudoDecrypt(tenantID, ciphertext string) string {
	if strings.TrimSpace(ciphertext) == "" {
		return ""
	}
	decoded, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return ""
	}
	parts := strings.SplitN(string(decoded), "|", 2)
	if len(parts) != 2 {
		return ""
	}
	if parts[0] != tenantID {
		return ""
	}
	return parts[1]
}

func maskEmail(email string) string {
	email = strings.TrimSpace(email)
	if email == "" {
		return ""
	}
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		if len(email) <= 2 {
			return "**"
		}
		return email[:1] + strings.Repeat("*", len(email)-2) + email[len(email)-1:]
	}
	local := parts[0]
	domain := parts[1]
	if len(local) <= 2 {
		local = strings.Repeat("*", len(local))
	} else {
		local = local[:1] + strings.Repeat("*", len(local)-2) + local[len(local)-1:]
	}
	return local + "@" + domain
}

func avString(item map[string]dynamodbtypes.AttributeValue, key string) string {
	value, ok := item[key]
	if !ok {
		return ""
	}
	switch typed := value.(type) {
	case *dynamodbtypes.AttributeValueMemberS:
		return typed.Value
	case *dynamodbtypes.AttributeValueMemberN:
		return typed.Value
	default:
		return ""
	}
}

func avNumber(item map[string]dynamodbtypes.AttributeValue, key string) *float64 {
	raw, ok := item[key]
	if !ok {
		return nil
	}
	num, ok := raw.(*dynamodbtypes.AttributeValueMemberN)
	if !ok {
		return nil
	}
	parsed, err := strconv.ParseFloat(strings.TrimSpace(num.Value), 64)
	if err != nil {
		return nil
	}
	return &parsed
}

func numericValue(v any) *float64 {
	switch typed := v.(type) {
	case float64:
		value := typed
		return &value
	case float32:
		value := float64(typed)
		return &value
	case int:
		value := float64(typed)
		return &value
	case int64:
		value := float64(typed)
		return &value
	case json.Number:
		parsed, err := typed.Float64()
		if err != nil {
			return nil
		}
		return &parsed
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		if err != nil {
			return nil
		}
		return &parsed
	default:
		return nil
	}
}

func intValue(v any, fallback int) int {
	switch typed := v.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case json.Number:
		parsed, err := typed.Int64()
		if err != nil {
			return fallback
		}
		return int(parsed)
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		if err != nil {
			return fallback
		}
		return parsed
	default:
		return fallback
	}
}

func stringValue(v any) string {
	switch typed := v.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	default:
		return ""
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
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

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func hashSHA256(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}
