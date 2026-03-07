package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	firestore "cloud.google.com/go/firestore/apiv1"
	"cloud.google.com/go/firestore/apiv1/firestorepb"
	kms "cloud.google.com/go/kms/apiv1"
	"cloud.google.com/go/kms/apiv1/kmspb"
	pubsub "cloud.google.com/go/pubsub/apiv1"
	"cloud.google.com/go/pubsub/apiv1/pubsubpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	defaultDomainPrefix = "ra-data-mesh"
	defaultProjectID    = "stackyard"
	defaultLocation     = "us-central1"
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

	ReadModelDocs map[string]map[string]any `json:"-"`
	Processed     map[string]bool           `json:"-"`
	DLQMessages   []eventEnvelope           `json:"-"`
}

type app struct {
	mu sync.Mutex

	endpoint     string
	apiEndpoint  string
	projectID    string
	locationID   string
	domainPrefix string

	eventSeq uint64
	domains  map[string]*domainState
	pending  map[string][]eventEnvelope

	publisherClient  *pubsub.PublisherClient
	subscriberClient *pubsub.SubscriberClient
	kmsClient        *kms.KeyManagementClient
	firestoreClient  *firestore.Client
}

func main() {
	ctx := context.Background()
	instance, err := newApp(ctx)
	if err != nil {
		log.Fatalf("initialize app: %v", err)
	}
	defer instance.close()

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", instance.handleHealth)
	mux.HandleFunc("/api/v1/summary", instance.handleSummary)
	mux.HandleFunc("/api/v1/domains/", instance.handleDomains)

	addr := getenv("APP_ADDR", ":8080")
	log.Printf("data-mesh backend (gcp) listening on %s", addr)
	if err := http.ListenAndServe(addr, loggingMiddleware(mux)); err != nil {
		log.Fatalf("listen: %v", err)
	}
}

func newApp(ctx context.Context) (*app, error) {
	endpoint := strings.TrimRight(getenv("STACKYARD_ENDPOINT", "http://localhost:4566"), "/")
	apiEndpoint := endpoint + "/gcp"
	projectID := getenv("STACKYARD_GCP_PROJECT_ID", defaultProjectID)
	locationID := getenv("STACKYARD_GCP_LOCATION", defaultLocation)
	domainPrefix := strings.ToLower(strings.TrimSpace(getenv("DATA_MESH_DOMAIN_PREFIX", defaultDomainPrefix)))

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

	kmsHTTP := &http.Client{Transport: stackyardHeaderTransport{base: http.DefaultTransport, serviceName: "kms"}}
	kmsClient, err := kms.NewKeyManagementRESTClient(ctx,
		option.WithHTTPClient(kmsHTTP),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		return nil, fmt.Errorf("create kms client: %w", err)
	}

	firestoreHTTP := &http.Client{Transport: stackyardHeaderTransport{base: http.DefaultTransport, serviceName: "firestore"}}
	firestoreClient, err := firestore.NewRESTClient(ctx,
		option.WithHTTPClient(firestoreHTTP),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		return nil, fmt.Errorf("create firestore client: %w", err)
	}

	return &app{
		endpoint:         endpoint,
		apiEndpoint:      apiEndpoint,
		projectID:        projectID,
		locationID:       locationID,
		domainPrefix:     domainPrefix,
		domains:          map[string]*domainState{},
		pending:          map[string][]eventEnvelope{},
		publisherClient:  publisherClient,
		subscriberClient: subscriberClient,
		kmsClient:        kmsClient,
		firestoreClient:  firestoreClient,
	}, nil
}

func (a *app) close() {
	if a.firestoreClient != nil {
		if err := a.firestoreClient.Close(); err != nil {
			log.Printf("close firestore client: %v", err)
		}
	}
	if a.kmsClient != nil {
		if err := a.kmsClient.Close(); err != nil {
			log.Printf("close kms client: %v", err)
		}
	}
	if a.subscriberClient != nil {
		if err := a.subscriberClient.Close(); err != nil {
			log.Printf("close pubsub subscriber client: %v", err)
		}
	}
	if a.publisherClient != nil {
		if err := a.publisherClient.Close(); err != nil {
			log.Printf("close pubsub publisher client: %v", err)
		}
	}
}

func (a *app) handleHealth(w http.ResponseWriter, _ *http.Request) {
	respondJSON(w, http.StatusOK, map[string]any{
		"status":        "ok",
		"stackyard":     a.endpoint,
		"api_endpoint":  a.apiEndpoint,
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

	respondJSON(w, http.StatusOK, map[string]any{"domains": domains})
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
	respondJSON(w, http.StatusOK, map[string]any{"message": "domain bootstrap complete", "domain": cloneDomain(state)})
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
		Producer:      producerInfo{Domain: domain, Service: "data-mesh-api"},
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
	publishOut, err := a.publisherClient.Publish(ctx, &pubsubpb.PublishRequest{
		Topic: state.StreamName,
		Messages: []*pubsubpb.PubsubMessage{{
			Data:        data,
			OrderingKey: tenantID,
		}},
	})
	if err != nil {
		respondErr(w, http.StatusBadGateway, "ServiceError", fmt.Sprintf("publish pubsub message: %v", err))
		return
	}

	sequence := ""
	if len(publishOut.GetMessageIds()) > 0 {
		sequence = publishOut.GetMessageIds()[0]
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
		"sequence_number":     sequence,
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
	results := map[string]int{"received": len(batch), "projected": 0, "duplicates": 0, "dlq": 0, "failed": 0}
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

		processed, checkErr := a.isEventProcessed(state, evt.EventID)
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
	maxMessages := intValue(request["max_messages"], 10)
	if maxMessages < 1 {
		maxMessages = 1
	}
	if maxMessages > 100 {
		maxMessages = 100
	}

	replayed := 0
	failed := 0

	a.mu.Lock()
	count := maxMessages
	if count > len(state.DLQMessages) {
		count = len(state.DLQMessages)
	}
	batch := append([]eventEnvelope(nil), state.DLQMessages[:count]...)
	state.DLQMessages = append([]eventEnvelope(nil), state.DLQMessages[count:]...)
	a.mu.Unlock()

	for _, evt := range batch {
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

		_, pubErr := a.publisherClient.Publish(ctx, &pubsubpb.PublishRequest{
			Topic: state.StreamName,
			Messages: []*pubsubpb.PubsubMessage{{
				Data: []byte(fmt.Sprintf(`{"event_id":"%s","replayed":true}`, evt.EventID)),
			}},
		})
		if pubErr != nil {
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

	a.mu.Lock()
	records := make([]map[string]any, 0)
	for key, record := range state.ReadModelDocs {
		if !strings.HasPrefix(key, tenantID+":") {
			continue
		}
		rec := map[string]any{
			"entity_id":      stringValue(record["entity_id"]),
			"event_id":       stringValue(record["event_id"]),
			"event_type":     stringValue(record["event_type"]),
			"occurred_at":    stringValue(record["occurred_at"]),
			"schema_version": stringValue(record["schema_version"]),
		}
		if amount := numericValue(record["amount"]); amount != nil {
			rec["amount"] = *amount
		}
		switch accessMode {
		case "full":
			rec["customer_email"] = pseudoDecrypt(tenantID, stringValue(record["customer_email_ciphertext"]))
		case "masked":
			rec["customer_email"] = stringValue(record["customer_email_masked"])
		}
		records = append(records, rec)
	}
	a.mu.Unlock()

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

	prefix := sanitizeID(a.domainPrefix + "-" + domain)
	projectName := "projects/" + a.projectID
	state = &domainState{
		Domain:         domain,
		StreamName:     projectName + "/topics/" + prefix + "-ingress",
		ReadModelTable: prefix + "-read-model",
		ProcessedTable: prefix + "-processed-events",
		DLQName:        prefix + "-dlq-sub",
		DLQURL:         projectName + "/subscriptions/" + prefix + "-dlq-sub",
		DLQARN:         projectName + "/topics/" + prefix + "-dlq",
		Tenants:        map[string]*tenantState{},
		UpdatedAt:      time.Now().UTC().Format(time.RFC3339),
		ReadModelDocs:  map[string]map[string]any{},
		Processed:      map[string]bool{},
		DLQMessages:    []eventEnvelope{},
	}
	a.domains[domain] = state
	a.mu.Unlock()

	if err := a.ensureTopic(ctx, state.StreamName); err != nil {
		return nil, err
	}
	if err := a.ensureTopic(ctx, state.DLQARN); err != nil {
		return nil, err
	}
	if err := a.ensureSubscription(ctx, state.DLQURL, state.DLQARN); err != nil {
		return nil, err
	}

	a.mu.Lock()
	state.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	a.mu.Unlock()
	return state, nil
}

func (a *app) ensureTenantKey(ctx context.Context, state *domainState, tenantID string) (*tenantState, error) {
	a.mu.Lock()
	if tenant := state.Tenants[tenantID]; tenant != nil {
		a.mu.Unlock()
		return tenant, nil
	}
	a.mu.Unlock()

	keyID, keyARN, err := a.createTenantKMSKey(ctx, state.Domain, tenantID)
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

func (a *app) createTenantKMSKey(ctx context.Context, domain, tenantID string) (string, string, error) {
	locationParent := fmt.Sprintf("projects/%s/locations/%s", a.projectID, a.locationID)
	keyRingID := sanitizeID("ra-dm-" + domain)
	keyID := sanitizeID("tenant-" + tenantID)
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

func (a *app) isEventProcessed(state *domainState, eventID string) (bool, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return state.Processed[eventID], nil
}

func (a *app) markEventProcessed(ctx context.Context, state *domainState, eventID string) error {
	a.mu.Lock()
	state.Processed[eventID] = true
	a.mu.Unlock()

	parent := fmt.Sprintf("projects/%s/databases/(default)/documents", a.projectID)
	collection := state.ProcessedTable
	docID := sanitizeID(eventID)
	name := fmt.Sprintf("%s/%s/%s", parent, collection, docID)
	_, err := a.firestoreClient.CreateDocument(ctx, &firestorepb.CreateDocumentRequest{
		Parent:       parent,
		CollectionId: collection,
		DocumentId:   docID,
		Document: &firestorepb.Document{
			Name: name,
			Fields: map[string]*firestorepb.Value{
				"event_id":     firestoreStringValue(eventID),
				"processed_at": firestoreStringValue(time.Now().UTC().Format(time.RFC3339)),
			},
		},
	})
	if err != nil && !isAlreadyExists(err) {
		return fmt.Errorf("mark processed event document: %w", err)
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

	record := map[string]any{
		"tenant_id":                 evt.TenantID,
		"entity_id":                 entityID,
		"event_id":                  evt.EventID,
		"event_type":                evt.EventType,
		"schema_version":            evt.SchemaVersion,
		"occurred_at":               evt.OccurredAt,
		"correlation_id":            evt.CorrelationID,
		"producer_domain":           evt.Producer.Domain,
		"producer_service":          evt.Producer.Service,
		"payload_json":              mustJSON(evt.Payload),
		"customer_email_ciphertext": pseudoEncrypt(evt.TenantID, customerEmail),
		"customer_email_masked":     maskEmail(customerEmail),
		"kms_key_arn":               tenant.KMSKeyARN,
		"updated_at":                time.Now().UTC().Format(time.RFC3339),
	}
	if amount := numericValue(evt.Payload["amount"]); amount != nil {
		record["amount"] = *amount
	}

	key := evt.TenantID + ":" + entityID
	a.mu.Lock()
	state.ReadModelDocs[key] = record
	a.mu.Unlock()

	parent := fmt.Sprintf("projects/%s/databases/(default)/documents", a.projectID)
	collection := state.ReadModelTable
	docID := sanitizeID(evt.TenantID + "-" + entityID)
	documentName := fmt.Sprintf("%s/%s/%s", parent, collection, docID)
	doc := &firestorepb.Document{
		Name: documentName,
		Fields: map[string]*firestorepb.Value{
			"tenant_id":      firestoreStringValue(evt.TenantID),
			"entity_id":      firestoreStringValue(entityID),
			"event_id":       firestoreStringValue(evt.EventID),
			"event_type":     firestoreStringValue(evt.EventType),
			"schema_version": firestoreStringValue(evt.SchemaVersion),
			"occurred_at":    firestoreStringValue(evt.OccurredAt),
			"updated_at":     firestoreStringValue(time.Now().UTC().Format(time.RFC3339)),
		},
	}
	if amount := numericValue(evt.Payload["amount"]); amount != nil {
		doc.Fields["amount"] = firestoreNumberValue(*amount)
	}

	_, err = a.firestoreClient.CreateDocument(ctx, &firestorepb.CreateDocumentRequest{
		Parent:       parent,
		CollectionId: collection,
		DocumentId:   docID,
		Document:     doc,
	})
	if err != nil {
		if !isAlreadyExists(err) {
			return fmt.Errorf("create read model document: %w", err)
		}
		_, err = a.firestoreClient.UpdateDocument(ctx, &firestorepb.UpdateDocumentRequest{Document: doc})
		if err != nil {
			return fmt.Errorf("update read model document: %w", err)
		}
	}
	return nil
}

func (a *app) sendEventToDLQ(ctx context.Context, state *domainState, evt eventEnvelope, reason string) error {
	_ = ctx
	a.mu.Lock()
	state.DLQMessages = append(state.DLQMessages, evt)
	state.Metrics.DLQTotal++
	state.Metrics.DLQDepth++
	state.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	a.mu.Unlock()

	_, err := a.publisherClient.Publish(context.Background(), &pubsubpb.PublishRequest{
		Topic: state.DLQARN,
		Messages: []*pubsubpb.PubsubMessage{{
			Data: []byte(fmt.Sprintf(`{"reason":"%s","event_id":"%s"}`, strings.ReplaceAll(reason, `"`, `'`), evt.EventID)),
		}},
	})
	if err != nil {
		return fmt.Errorf("publish to dlq topic: %w", err)
	}
	return nil
}

func (a *app) ensureTopic(ctx context.Context, topicName string) error {
	_, err := a.publisherClient.CreateTopic(ctx, &pubsubpb.Topic{Name: topicName})
	if err == nil || isAlreadyExists(err) {
		return nil
	}
	return fmt.Errorf("create topic %s: %w", topicName, err)
}

func (a *app) ensureSubscription(ctx context.Context, subscriptionName, topicName string) error {
	_, err := a.subscriberClient.CreateSubscription(ctx, &pubsubpb.Subscription{Name: subscriptionName, Topic: topicName})
	if err == nil || isAlreadyExists(err) {
		return nil
	}
	return fmt.Errorf("create subscription %s: %w", subscriptionName, err)
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
	copy.ReadModelDocs = nil
	copy.Processed = nil
	copy.DLQMessages = nil
	return &copy
}

func parseScopes(raw string) map[string]bool {
	raw = strings.TrimSpace(raw)
	parts := strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == ' ' || r == ';' })
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

func normalizeDomain(v string) string { return strings.ToLower(strings.TrimSpace(v)) }
func normalizeTenant(v string) string { return strings.ToLower(strings.TrimSpace(v)) }

func isAlreadyExists(err error) bool {
	if err == nil {
		return false
	}
	if s, ok := status.FromError(err); ok && s.Code() == codes.AlreadyExists {
		return true
	}
	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) && apiErr.Code == http.StatusConflict {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "already exists") || strings.Contains(msg, "alreadyexists") || strings.Contains(msg, "resourceexistexception")
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
	if len(parts) != 2 || parts[0] != tenantID {
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

func mustJSON(v any) string {
	payload, _ := json.Marshal(v)
	return string(payload)
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

func firestoreStringValue(v string) *firestorepb.Value {
	return &firestorepb.Value{ValueType: &firestorepb.Value_StringValue{StringValue: v}}
}

func firestoreNumberValue(v float64) *firestorepb.Value {
	return &firestorepb.Value{ValueType: &firestorepb.Value_DoubleValue{DoubleValue: v}}
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
