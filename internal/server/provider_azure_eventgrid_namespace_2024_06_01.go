package server

import (
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Azure Event Grid namespace data plane (2024-06-01): publishing Cloud
// Events to a namespace topic and pull delivery through its queue event
// subscriptions (receive, acknowledge, reject, release, renewLock), served at
// /azure/eventgrid-namespace/2024-06-01/{namespaceHost}/topics/{topic}...
// The host's first label names an Event Grid namespace, and the topic and
// subscriptions are its namespaces/topics and their eventSubscriptions on the
// shared ARM handler. Publishing copies each event into the queue of every
// queue-mode subscription whose includedEventTypes filter matches it.
// Rejected events and events past maxDeliveryCount are dropped (no
// dead-lettering); maxWaitTime doesn't block.

const (
	azureEventGridNamespace20240601APIVersion = "2024-06-01"
	azureEventGridNamespace20240601Prefix     = "/azure/eventgrid-namespace/2024-06-01"
	egnsMaxBody                               = 1 << 20
	egnsMaxBatch                              = 1000
	// egnsMaxQueue caps the events one subscription holds.
	egnsMaxQueue   = 10000
	egnsMaxTokens  = 100
	egnsDefaultTTL = 60 * time.Second
)

type egnsEvent struct {
	event         map[string]any
	token         string
	deliveryCount int
	lockedUntil   time.Time
	availableAt   time.Time
}

// egnsBroker holds one server's subscription queues, keyed by subscription
// ID and creation time so a recreated subscription starts empty.
type egnsBroker struct {
	mu     sync.Mutex
	queues map[string][]*egnsEvent
}

// ponytail: brokers live beside the ARM store, keyed by server; queues of
// deleted subscriptions linger (each capped at egnsMaxQueue) until the server goes.
var egnsBrokers sync.Map

func (s *Server) egnsBroker() *egnsBroker {
	broker, _ := egnsBrokers.LoadOrStore(s, &egnsBroker{queues: map[string][]*egnsEvent{}})
	return broker.(*egnsBroker)
}

func egnsQueueKey(sub *azureARMResource) string {
	return azureARMKey(sub.ID) + "#" + strconv.FormatInt(sub.CreatedAt.UnixNano(), 10)
}

func (s *Server) handleAzureEventGridNamespace20240601Router(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, azureEventGridNamespace20240601Prefix+"/") {
		return false
	}
	seg := splitPathSegments(strings.TrimPrefix(path, azureEventGridNamespace20240601Prefix))
	if n := len(seg); n > 0 {
		if last, err := url.PathUnescape(seg[n-1]); err == nil {
			seg[n-1] = last
		}
	}
	switch {
	case r.Method != http.MethodPost:
		azureARMError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "Namespace operations use POST.")
	case r.URL.Query().Get("api-version") != azureEventGridNamespace20240601APIVersion:
		azureARMError(w, http.StatusBadRequest, "InvalidApiVersionParameter", "api-version must be "+azureEventGridNamespace20240601APIVersion+".")
	case len(seg) == 3 && strings.EqualFold(seg[1], "topics"):
		topic, op, _ := strings.Cut(seg[2], ":")
		if op != "publish" {
			azureARMError(w, http.StatusNotFound, "NotFound", "Topics support only :publish.")
			return true
		}
		s.egnsPublish(w, r, seg[0], topic)
	case len(seg) == 5 && strings.EqualFold(seg[1], "topics") && strings.EqualFold(seg[3], "eventsubscriptions"):
		name, op, _ := strings.Cut(seg[4], ":")
		s.egnsSubscriptionOp(w, r, seg[0], seg[2], name, op)
	default:
		azureARMError(w, http.StatusNotFound, "NotFound", "Unknown namespace route.")
	}
	return true
}

// egnsTopic resolves the namespace host and topic, checks credentials, and
// returns the topic, or writes the error and returns nil.
func (s *Server) egnsTopic(w http.ResponseWriter, r *http.Request, host, topicName string) *azureARMResource {
	name := strings.SplitN(host, ".", 2)[0]
	namespaces := s.azureARMResourcesWhere(func(rec *azureARMResource) bool {
		return strings.EqualFold(rec.Type, egcpNamespace+"/namespaces") && strings.EqualFold(rec.Name, name)
	})
	if len(namespaces) == 0 {
		azureARMError(w, http.StatusNotFound, "NotFound", "No Event Grid namespace has the host "+host+".")
		return nil
	}
	namespace := namespaces[0]
	topic := s.azureARMLookup(namespace.ID + "/topics/" + topicName)
	if topic == nil {
		azureARMError(w, http.StatusNotFound, "NotFound", "The namespace has no topic "+topicName+".")
		return nil
	}
	auth := r.Header.Get("Authorization")
	if key, ok := strings.CutPrefix(auth, "SharedAccessKey "); ok {
		valid := false
		for _, rec := range []*azureARMResource{namespace, topic} {
			valid = valid || key == s.azureResourceKey(rec, "key1") || key == s.azureResourceKey(rec, "key2")
		}
		if !valid {
			azureARMError(w, http.StatusUnauthorized, "Unauthorized", "The shared access key doesn't match the namespace or topic.")
			return nil
		}
	} else if auth == "" {
		azureARMError(w, http.StatusUnauthorized, "Unauthorized", "The request needs an Authorization header.")
		return nil
	}
	return topic
}

func (s *Server) egnsPublish(w http.ResponseWriter, r *http.Request, host, topicName string) {
	topic := s.egnsTopic(w, r, host, topicName)
	if topic == nil {
		return
	}
	if r.ContentLength < 0 || r.ContentLength > egnsMaxBody {
		azureARMError(w, http.StatusRequestEntityTooLarge, "RequestEntityTooLarge", "The request must carry a Content-Length of at most 1 MiB.")
		return
	}
	mediaType, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	batch := r.URL.Query().Get("_overload") == "publishCloudEvents" || mediaType == "application/cloudevents-batch+json"
	body, err := io.ReadAll(io.LimitReader(r.Body, egnsMaxBody))
	if err != nil {
		azureARMError(w, http.StatusBadRequest, "BadRequest", "The request body couldn't be read.")
		return
	}
	var events []map[string]any
	if batch {
		err = json.Unmarshal(body, &events)
	} else {
		var event map[string]any
		err = json.Unmarshal(body, &event)
		events = []map[string]any{event}
	}
	if err != nil || len(events) == 0 || len(events) > egnsMaxBatch {
		azureARMError(w, http.StatusBadRequest, "BadRequest", "The body must be a Cloud Event, or for a batch an array of 1 to 1000 Cloud Events.")
		return
	}
	for i, event := range events {
		if msg := egdpValidateEvent("cloudEvent", event); msg != "" {
			azureARMError(w, http.StatusBadRequest, "BadRequest", "Event "+strconv.Itoa(i)+": "+msg)
			return
		}
	}
	subs := s.azureARMResourcesWhere(func(rec *azureARMResource) bool {
		return strings.EqualFold(rec.Type, egcpNamespace+"/namespaces/topics/eventSubscriptions") &&
			strings.HasPrefix(azureARMKey(rec.ID), azureARMKey(topic.ID)+"/") &&
			azureARMObject(rec.Properties["deliveryConfiguration"])["deliveryMode"] == "Queue"
	})
	broker := s.egnsBroker()
	broker.mu.Lock()
	defer broker.mu.Unlock()
	// Check every queue first, so a full one rejects the request without
	// delivering it to the others.
	routed := make(map[string][]map[string]any, len(subs))
	for _, sub := range subs {
		var matched []map[string]any
		for _, event := range events {
			if egnsMatches(sub, event) {
				matched = append(matched, event)
			}
		}
		key := egnsQueueKey(sub)
		if len(broker.queues[key])+len(matched) > egnsMaxQueue {
			azureARMError(w, http.StatusForbidden, "QuotaExceeded", "Event subscription "+sub.Name+" already holds the maximum number of events.")
			return
		}
		routed[key] = matched
	}
	now := time.Now()
	for key, matched := range routed {
		for _, event := range matched {
			broker.queues[key] = append(broker.queues[key], &egnsEvent{event: event, availableAt: now})
		}
	}
	respondJSON(w, http.StatusOK, map[string]any{})
}

// egnsMatches applies the subscription's includedEventTypes filter.
func egnsMatches(sub *azureARMResource, event map[string]any) bool {
	included := azureARMSlice(azureARMObject(sub.Properties["filtersConfiguration"])["includedEventTypes"])
	if len(included) == 0 {
		return true
	}
	for _, typ := range included {
		if strings.EqualFold(azureARMString(typ), azureARMString(event["type"])) {
			return true
		}
	}
	return false
}

func (s *Server) egnsSubscriptionOp(w http.ResponseWriter, r *http.Request, host, topicName, subName, op string) {
	topic := s.egnsTopic(w, r, host, topicName)
	if topic == nil {
		return
	}
	sub := s.azureARMLookup(topic.ID + "/eventSubscriptions/" + subName)
	if sub == nil {
		azureARMError(w, http.StatusNotFound, "NotFound", "The topic has no event subscription "+subName+".")
		return
	}
	queue := azureARMObject(azureARMObject(sub.Properties["deliveryConfiguration"])["queue"])
	if azureARMObject(sub.Properties["deliveryConfiguration"])["deliveryMode"] != "Queue" {
		azureARMError(w, http.StatusBadRequest, "BadRequest", "Event subscription "+subName+" uses push delivery.")
		return
	}
	lock := egnsDefaultTTL
	if n, ok := queue["receiveLockDurationInSeconds"].(float64); ok {
		lock = time.Duration(n) * time.Second
	}
	maxDeliveries := 10
	if n, ok := queue["maxDeliveryCount"].(float64); ok {
		maxDeliveries = int(n)
	}
	if op == "receive" {
		s.egnsReceive(w, r, egnsQueueKey(sub), lock, maxDeliveries)
		return
	}
	var tokens []string
	switch op {
	case "acknowledge", "reject", "release", "renewLock":
		var payload struct {
			LockTokens []string `json:"lockTokens"`
		}
		if r.ContentLength < 0 || r.ContentLength > egnsMaxBody {
			azureARMError(w, http.StatusRequestEntityTooLarge, "RequestEntityTooLarge", "The request must carry a Content-Length of at most 1 MiB.")
			return
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, egnsMaxBody)).Decode(&payload); err != nil || len(payload.LockTokens) == 0 || len(payload.LockTokens) > egnsMaxTokens {
			azureARMError(w, http.StatusBadRequest, "BadRequest", "lockTokens must hold 1 to 100 lock tokens.")
			return
		}
		tokens = payload.LockTokens
	default:
		azureARMError(w, http.StatusNotFound, "NotFound", "Unknown event subscription operation "+op+".")
		return
	}
	delay := time.Duration(0)
	if op == "release" {
		raw := r.URL.Query().Get("releaseDelayInSeconds")
		switch raw {
		case "", "0", "10", "60", "600", "3600":
			n, _ := strconv.ParseInt(raw, 10, 32)
			delay = time.Duration(n) * time.Second
		default:
			azureARMError(w, http.StatusBadRequest, "BadRequest", "releaseDelayInSeconds must be 0, 10, 60, 600 or 3600.")
			return
		}
	}
	respondJSON(w, http.StatusOK, s.egnsSettle(egnsQueueKey(sub), op, tokens, lock, delay))
}

func (s *Server) egnsReceive(w http.ResponseWriter, r *http.Request, key string, lock time.Duration, maxDeliveries int) {
	query := r.URL.Query()
	maxEvents := int64(1)
	if raw := query.Get("maxEvents"); raw != "" {
		n, err := strconv.ParseInt(raw, 10, 32)
		if err != nil || n < 1 || n > 100 {
			azureARMError(w, http.StatusBadRequest, "BadRequest", "maxEvents must be an integer from 1 to 100.")
			return
		}
		maxEvents = n
	}
	if raw := query.Get("maxWaitTime"); raw != "" {
		if n, err := strconv.ParseInt(raw, 10, 32); err != nil || n < 10 || n > 120 {
			azureARMError(w, http.StatusBadRequest, "BadRequest", "maxWaitTime must be an integer from 10 to 120 seconds.")
			return
		}
	}
	broker := s.egnsBroker()
	broker.mu.Lock()
	defer broker.mu.Unlock()
	now := time.Now()
	value := []any{}
	kept := broker.queues[key][:0]
	for _, e := range broker.queues[key] {
		ready := now.After(e.availableAt) || now.Equal(e.availableAt)
		if int64(len(value)) < maxEvents && ready && now.After(e.lockedUntil) {
			if e.deliveryCount >= maxDeliveries {
				continue // Delivered too often: dropped, as dead-lettering isn't modeled.
			}
			e.deliveryCount++
			e.token = uuid.NewString()
			e.lockedUntil = now.Add(lock)
			value = append(value, map[string]any{
				"brokerProperties": map[string]any{"lockToken": e.token, "deliveryCount": e.deliveryCount},
				"event":            e.event,
			})
		}
		kept = append(kept, e)
	}
	broker.queues[key] = kept
	respondJSON(w, http.StatusOK, map[string]any{"value": value})
}

// egnsSettle applies acknowledge, reject, release or renewLock to each lock
// token, failing tokens whose lock is gone.
func (s *Server) egnsSettle(key, op string, tokens []string, lock, delay time.Duration) map[string]any {
	broker := s.egnsBroker()
	broker.mu.Lock()
	defer broker.mu.Unlock()
	now := time.Now()
	failed, succeeded := []any{}, []any{}
	for _, token := range tokens {
		index := -1
		for i, e := range broker.queues[key] {
			if e.token != "" && e.token == token && !now.After(e.lockedUntil) {
				index = i
				break
			}
		}
		if index < 0 {
			failed = append(failed, map[string]any{"lockToken": token, "error": map[string]any{"code": "TokenLost", "message": "The lock token is not valid or its lock has expired."}})
			continue
		}
		queue := broker.queues[key]
		e := queue[index]
		switch op {
		case "acknowledge", "reject":
			broker.queues[key] = append(queue[:index:index], queue[index+1:]...)
		case "release":
			e.token, e.lockedUntil, e.availableAt = "", time.Time{}, now.Add(delay)
		case "renewLock":
			e.lockedUntil = now.Add(lock)
		}
		succeeded = append(succeeded, token)
	}
	return map[string]any{"failedLockTokens": failed, "succeededLockTokens": succeeded}
}
