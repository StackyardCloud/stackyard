package server

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

const azureCommunicationJobRouterAPIVersion = "2023-11-01"
const azureCommunicationJobRouterPrefix = "/azure/communication-jobrouter/2023-11-01"

type azureCommunicationJobRouterResource struct {
	ID           string
	Kind         string
	Properties   map[string]any
	ETag         string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	LastModified time.Time
}

type azureCommunicationJobRouterJob struct {
	ID                      string
	ChannelID               string
	ChannelReference        string
	ClassificationPolicyID  string
	QueueID                 string
	DispositionCode         string
	Labels                  map[string]any
	Tags                    map[string]any
	MatchingMode            map[string]any
	RequestedWorkerSelector []any
	Notes                   []any
	Priority                int
	Status                  string
	AssignedWorkerID        string
	AssignmentID            string
	OfferIDs                []string
	ETag                    string
	CreatedAt               time.Time
	UpdatedAt               time.Time
	LastModified            time.Time
}

type azureCommunicationJobRouterWorker struct {
	ID                    string
	QueueIDs              []string
	Labels                map[string]any
	Tags                  map[string]any
	Capacity              int
	ChannelConfigurations []any
	Offers                []string
	Assignments           []string
	ETag                  string
	CreatedAt             time.Time
	UpdatedAt             time.Time
	LastModified          time.Time
}

type azureCommunicationJobRouterOffer struct {
	ID           string
	JobID        string
	WorkerID     string
	Status       string
	RetryOfferAt *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type azureCommunicationJobRouterAssignment struct {
	ID        string
	JobID     string
	WorkerID  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (s *Server) handleAzureCommunicationJobRouterRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, azureCommunicationJobRouterPrefix) {
		return false
	}
	if strings.TrimSpace(r.URL.Query().Get("api-version")) != azureCommunicationJobRouterAPIVersion {
		azureCommunicationJobRouterError(w, http.StatusBadRequest, "InvalidApiVersion", "api-version must be 2023-11-01")
		return true
	}

	relative := strings.TrimPrefix(path, azureCommunicationJobRouterPrefix)
	segments := splitPathSegments(strings.Trim(relative, "/"))
	if len(segments) == 0 || !strings.EqualFold(segments[0], "routing") {
		azureCommunicationJobRouterError(w, http.StatusNotFound, "PathNotFound", "route not found")
		return true
	}
	if len(segments) < 2 {
		azureCommunicationJobRouterError(w, http.StatusNotFound, "PathNotFound", "route not found")
		return true
	}

	switch strings.ToLower(strings.TrimSpace(segments[1])) {
	case "classificationpolicies":
		s.handleAzureCommunicationJobRouterAdminFamily(w, r, "classificationPolicy", segments[2:], s.azureCommunicationJobRouterClassificationPolicies)
		return true
	case "distributionpolicies":
		s.handleAzureCommunicationJobRouterAdminFamily(w, r, "distributionPolicy", segments[2:], s.azureCommunicationJobRouterDistributionPolicies)
		return true
	case "exceptionpolicies":
		s.handleAzureCommunicationJobRouterAdminFamily(w, r, "exceptionPolicy", segments[2:], s.azureCommunicationJobRouterExceptionPolicies)
		return true
	case "queues":
		s.handleAzureCommunicationJobRouterQueues(w, r, segments[2:])
		return true
	case "jobs":
		s.handleAzureCommunicationJobRouterJobs(w, r, segments[2:])
		return true
	case "workers":
		s.handleAzureCommunicationJobRouterWorkers(w, r, segments[2:])
		return true
	default:
		azureCommunicationJobRouterError(w, http.StatusNotFound, "PathNotFound", "route not found")
		return true
	}
}

func (s *Server) handleAzureCommunicationJobRouterAdminFamily(w http.ResponseWriter, r *http.Request, kind string, tail []string, store map[string]*azureCommunicationJobRouterResource) {
	if len(tail) == 0 {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureCommunicationJobRouterListResources(w, r, store)
		return
	}
	if len(tail) != 1 {
		azureCommunicationJobRouterError(w, http.StatusNotFound, "PathNotFound", "route not found")
		return
	}
	id := strings.TrimSpace(tail[0])
	if id == "" {
		azureCommunicationJobRouterError(w, http.StatusBadRequest, "InvalidRequest", kind+"Id is required")
		return
	}

	switch r.Method {
	case http.MethodPatch:
		s.handleAzureCommunicationJobRouterUpsertResource(w, r, kind, id, store)
	case http.MethodGet:
		s.handleAzureCommunicationJobRouterGetResource(w, id, store)
	case http.MethodDelete:
		s.handleAzureCommunicationJobRouterDeleteResource(w, id, store)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAzureCommunicationJobRouterQueues(w http.ResponseWriter, r *http.Request, tail []string) {
	if len(tail) == 2 && strings.EqualFold(tail[1], "statistics") {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureCommunicationJobRouterGetQueueStatistics(w, tail[0])
		return
	}
	s.handleAzureCommunicationJobRouterAdminFamily(w, r, "queue", tail, s.azureCommunicationJobRouterQueues)
}

func (s *Server) handleAzureCommunicationJobRouterJobs(w http.ResponseWriter, r *http.Request, tail []string) {
	if len(tail) == 0 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureCommunicationJobRouterListJobs(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	jobID, action, actionOK := azureCommunicationJobRouterSplitAction(tail[0])
	if actionOK {
		if len(tail) != 1 || r.Method != http.MethodPost {
			if len(tail) == 1 {
				w.WriteHeader(http.StatusMethodNotAllowed)
			} else {
				azureCommunicationJobRouterError(w, http.StatusNotFound, "PathNotFound", "route not found")
			}
			return
		}
		s.handleAzureCommunicationJobRouterJobAction(w, r, jobID, action)
		return
	}

	jobID = strings.TrimSpace(tail[0])
	if jobID == "" {
		azureCommunicationJobRouterError(w, http.StatusBadRequest, "InvalidRequest", "jobId is required")
		return
	}
	if len(tail) == 1 {
		switch r.Method {
		case http.MethodPatch:
			s.handleAzureCommunicationJobRouterUpsertJob(w, r, jobID)
		case http.MethodGet:
			s.handleAzureCommunicationJobRouterGetJob(w, jobID)
		case http.MethodDelete:
			s.handleAzureCommunicationJobRouterDeleteJob(w, jobID)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}
	if len(tail) == 2 && strings.EqualFold(tail[1], "position") {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureCommunicationJobRouterGetInQueuePosition(w, jobID)
		return
	}
	azureCommunicationJobRouterError(w, http.StatusNotFound, "PathNotFound", "route not found")
}

func (s *Server) handleAzureCommunicationJobRouterWorkers(w http.ResponseWriter, r *http.Request, tail []string) {
	if len(tail) == 0 {
		switch r.Method {
		case http.MethodGet:
			s.handleAzureCommunicationJobRouterListWorkers(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}
	workerID := strings.TrimSpace(tail[0])
	if workerID == "" {
		azureCommunicationJobRouterError(w, http.StatusBadRequest, "InvalidRequest", "workerId is required")
		return
	}
	if len(tail) == 1 {
		switch r.Method {
		case http.MethodPatch:
			s.handleAzureCommunicationJobRouterUpsertWorker(w, r, workerID)
		case http.MethodGet:
			s.handleAzureCommunicationJobRouterGetWorker(w, workerID)
		case http.MethodDelete:
			s.handleAzureCommunicationJobRouterDeleteWorker(w, workerID)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}
	if len(tail) == 3 && strings.EqualFold(tail[1], "offers") {
		offerID, action, ok := azureCommunicationJobRouterSplitAction(tail[2])
		if !ok {
			azureCommunicationJobRouterError(w, http.StatusNotFound, "PathNotFound", "route not found")
			return
		}
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.handleAzureCommunicationJobRouterOfferAction(w, r, workerID, offerID, action)
		return
	}
	azureCommunicationJobRouterError(w, http.StatusNotFound, "PathNotFound", "route not found")
}

func (s *Server) handleAzureCommunicationJobRouterUpsertResource(w http.ResponseWriter, r *http.Request, kind, id string, store map[string]*azureCommunicationJobRouterResource) {
	payload, _, ok := azureCommunicationJobRouterReadObject(w, r)
	if !ok {
		return
	}
	if !azureCommunicationJobRouterValidateResourcePayload(kind, payload) {
		azureCommunicationJobRouterError(w, http.StatusBadRequest, "InvalidRequest", kind+" payload is invalid")
		return
	}

	now := time.Now().UTC()
	s.providerStorageMu.Lock()
	resource := store[id]
	status := http.StatusCreated
	if resource == nil {
		resource = &azureCommunicationJobRouterResource{
			ID:         id,
			Kind:       kind,
			Properties: map[string]any{},
			CreatedAt:  now,
		}
		store[id] = resource
	} else {
		status = http.StatusOK
	}
	resource.Properties = azureCommunicationJobRouterMergeObject(resource.Properties, payload)
	resource.UpdatedAt = now
	resource.LastModified = now
	resource.ETag = azureCommunicationJobRouterDigest(kind + "|" + id + "|" + now.Format(time.RFC3339Nano))[:16]
	fixture := azureCommunicationJobRouterResourceFixture(resource)
	etag := resource.ETag
	lastModified := resource.LastModified
	s.providerStorageMu.Unlock()

	w.Header().Set("ETag", etag)
	w.Header().Set("Last-Modified", lastModified.Format(http.TimeFormat))
	respondJSON(w, status, fixture)
}

func (s *Server) handleAzureCommunicationJobRouterGetResource(w http.ResponseWriter, id string, store map[string]*azureCommunicationJobRouterResource) {
	s.providerStorageMu.Lock()
	resource := store[id]
	s.providerStorageMu.Unlock()
	if resource == nil {
		azureCommunicationJobRouterError(w, http.StatusNotFound, "ResourceNotFound", "resource not found")
		return
	}
	w.Header().Set("ETag", resource.ETag)
	w.Header().Set("Last-Modified", resource.LastModified.Format(http.TimeFormat))
	respondJSON(w, http.StatusOK, azureCommunicationJobRouterResourceFixture(resource))
}

func (s *Server) handleAzureCommunicationJobRouterDeleteResource(w http.ResponseWriter, id string, store map[string]*azureCommunicationJobRouterResource) {
	s.providerStorageMu.Lock()
	if store[id] == nil {
		s.providerStorageMu.Unlock()
		azureCommunicationJobRouterError(w, http.StatusNotFound, "ResourceNotFound", "resource not found")
		return
	}
	delete(store, id)
	s.providerStorageMu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureCommunicationJobRouterListResources(w http.ResponseWriter, r *http.Request, store map[string]*azureCommunicationJobRouterResource) {
	maxPageSize, ok := azureCommunicationJobRouterParseMaxPageSize(w, r)
	if !ok {
		return
	}

	s.providerStorageMu.Lock()
	items := make([]*azureCommunicationJobRouterResource, 0, len(store))
	for _, item := range store {
		items = append(items, item)
	}
	s.providerStorageMu.Unlock()

	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	pageSize := len(items)
	if maxPageSize > 0 && maxPageSize < pageSize {
		pageSize = maxPageSize
	}
	value := make([]any, 0, pageSize)
	for _, item := range items[:pageSize] {
		value = append(value, azureCommunicationJobRouterResourceFixture(item))
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"value": value,
	})
}

func (s *Server) handleAzureCommunicationJobRouterUpsertJob(w http.ResponseWriter, r *http.Request, jobID string) {
	payload, _, ok := azureCommunicationJobRouterReadObject(w, r)
	if !ok {
		return
	}
	if !azureCommunicationJobRouterValidateJobPayload(payload) {
		azureCommunicationJobRouterError(w, http.StatusBadRequest, "InvalidRequest", "job payload is invalid")
		return
	}
	now := time.Now().UTC()
	s.providerStorageMu.Lock()
	job := s.azureCommunicationJobRouterJobs[jobID]
	status := http.StatusCreated
	if job == nil {
		job = &azureCommunicationJobRouterJob{
			ID:        jobID,
			Labels:    map[string]any{},
			Tags:      map[string]any{},
			Status:    "queued",
			CreatedAt: now,
		}
		s.azureCommunicationJobRouterJobs[jobID] = job
	} else {
		status = http.StatusOK
	}
	job.ChannelID = azureCommunicationJobRouterFirstNonEmptyString(payload, "channelId", job.ChannelID)
	job.ChannelReference = azureCommunicationJobRouterFirstNonEmptyString(payload, "channelReference", job.ChannelReference)
	job.ClassificationPolicyID = azureCommunicationJobRouterFirstNonEmptyString(payload, "classificationPolicyId", job.ClassificationPolicyID)
	job.QueueID = azureCommunicationJobRouterFirstNonEmptyString(payload, "queueId", job.QueueID)
	job.DispositionCode = azureCommunicationJobRouterFirstNonEmptyString(payload, "dispositionCode", job.DispositionCode)
	if labels, ok := payload["labels"].(map[string]any); ok {
		job.Labels = azureCommunicationJobRouterCloneObject(labels)
	}
	if tags, ok := payload["tags"].(map[string]any); ok {
		job.Tags = azureCommunicationJobRouterCloneObject(tags)
	}
	if matchingMode, ok := payload["matchingMode"].(map[string]any); ok {
		job.MatchingMode = azureCommunicationJobRouterCloneObject(matchingMode)
	}
	if requestedSelectors, ok := payload["requestedWorkerSelectors"].([]any); ok {
		job.RequestedWorkerSelector = azureCommunicationJobRouterCloneSlice(requestedSelectors)
	}
	if notes, ok := payload["notes"].([]any); ok {
		job.Notes = azureCommunicationJobRouterCloneSlice(notes)
	}
	if priority, ok := azureCommunicationJobRouterInt(payload["priority"]); ok {
		job.Priority = priority
	}
	if job.Status == "" || job.Status == "deleted" {
		job.Status = "queued"
	}
	job.UpdatedAt = now
	job.LastModified = now
	job.ETag = azureCommunicationJobRouterDigest("job|" + jobID + "|" + now.Format(time.RFC3339Nano))[:16]
	s.azureCommunicationJobRouterReconcileOffersLocked()
	fixture := azureCommunicationJobRouterJobFixtureLocked(job, s)
	etag := job.ETag
	lastModified := job.LastModified
	s.providerStorageMu.Unlock()

	w.Header().Set("ETag", etag)
	w.Header().Set("Last-Modified", lastModified.Format(http.TimeFormat))
	respondJSON(w, status, fixture)
}

func (s *Server) handleAzureCommunicationJobRouterGetJob(w http.ResponseWriter, jobID string) {
	s.providerStorageMu.Lock()
	job := s.azureCommunicationJobRouterJobs[jobID]
	s.providerStorageMu.Unlock()
	if job == nil {
		azureCommunicationJobRouterError(w, http.StatusNotFound, "JobNotFound", "job not found")
		return
	}
	w.Header().Set("ETag", job.ETag)
	w.Header().Set("Last-Modified", job.LastModified.Format(http.TimeFormat))
	respondJSON(w, http.StatusOK, azureCommunicationJobRouterJobFixture(job, s))
}

func (s *Server) handleAzureCommunicationJobRouterListJobs(w http.ResponseWriter, r *http.Request) {
	maxPageSize, ok := azureCommunicationJobRouterParseMaxPageSize(w, r)
	if !ok {
		return
	}
	s.providerStorageMu.Lock()
	items := make([]*azureCommunicationJobRouterJob, 0, len(s.azureCommunicationJobRouterJobs))
	for _, item := range s.azureCommunicationJobRouterJobs {
		items = append(items, item)
	}
	s.providerStorageMu.Unlock()
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	pageSize := len(items)
	if maxPageSize > 0 && maxPageSize < pageSize {
		pageSize = maxPageSize
	}
	value := make([]any, 0, pageSize)
	for _, item := range items[:pageSize] {
		value = append(value, azureCommunicationJobRouterJobFixture(item, s))
	}
	respondJSON(w, http.StatusOK, map[string]any{"value": value})
}

func (s *Server) handleAzureCommunicationJobRouterDeleteJob(w http.ResponseWriter, jobID string) {
	s.providerStorageMu.Lock()
	job := s.azureCommunicationJobRouterJobs[jobID]
	if job == nil {
		s.providerStorageMu.Unlock()
		azureCommunicationJobRouterError(w, http.StatusNotFound, "JobNotFound", "job not found")
		return
	}
	for _, offerID := range job.OfferIDs {
		delete(s.azureCommunicationJobRouterOffers, offerID)
	}
	if job.AssignmentID != "" {
		delete(s.azureCommunicationJobRouterAssignments, job.AssignmentID)
	}
	delete(s.azureCommunicationJobRouterJobs, jobID)
	s.providerStorageMu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureCommunicationJobRouterJobAction(w http.ResponseWriter, r *http.Request, jobID, action string) {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "cancel":
		s.handleAzureCommunicationJobRouterCancelJob(w, r, jobID)
	case "complete":
		s.handleAzureCommunicationJobRouterCompleteJob(w, jobID)
	case "close":
		s.handleAzureCommunicationJobRouterCloseJob(w, r, jobID)
	case "reclassify":
		s.handleAzureCommunicationJobRouterReclassifyJob(w, r, jobID)
	case "unassign":
		s.handleAzureCommunicationJobRouterUnassignJob(w, r, jobID)
	default:
		azureCommunicationJobRouterError(w, http.StatusNotFound, "PathNotFound", "route not found")
	}
}

func (s *Server) handleAzureCommunicationJobRouterCancelJob(w http.ResponseWriter, r *http.Request, jobID string) {
	payload, _, ok := azureCommunicationJobRouterReadObject(w, r)
	if !ok {
		return
	}
	s.providerStorageMu.Lock()
	job := s.azureCommunicationJobRouterJobs[jobID]
	if job == nil {
		s.providerStorageMu.Unlock()
		azureCommunicationJobRouterError(w, http.StatusNotFound, "JobNotFound", "job not found")
		return
	}
	job.Status = "canceled"
	job.DispositionCode = azureCommunicationJobRouterFirstNonEmptyString(payload, "dispositionCode", job.DispositionCode)
	job.UpdatedAt = time.Now().UTC()
	job.LastModified = job.UpdatedAt
	for _, offerID := range job.OfferIDs {
		if offer := s.azureCommunicationJobRouterOffers[offerID]; offer != nil && offer.Status == "pending" {
			offer.Status = "canceled"
			offer.UpdatedAt = job.UpdatedAt
		}
	}
	fixture := azureCommunicationJobRouterJobFixtureLocked(job, s)
	s.providerStorageMu.Unlock()
	respondJSON(w, http.StatusOK, fixture)
}

func (s *Server) handleAzureCommunicationJobRouterCompleteJob(w http.ResponseWriter, jobID string) {
	s.providerStorageMu.Lock()
	job := s.azureCommunicationJobRouterJobs[jobID]
	if job == nil {
		s.providerStorageMu.Unlock()
		azureCommunicationJobRouterError(w, http.StatusNotFound, "JobNotFound", "job not found")
		return
	}
	job.Status = "completed"
	job.UpdatedAt = time.Now().UTC()
	job.LastModified = job.UpdatedAt
	fixture := azureCommunicationJobRouterJobFixtureLocked(job, s)
	s.providerStorageMu.Unlock()
	respondJSON(w, http.StatusOK, fixture)
}

func (s *Server) handleAzureCommunicationJobRouterCloseJob(w http.ResponseWriter, r *http.Request, jobID string) {
	payload, _, ok := azureCommunicationJobRouterReadObject(w, r)
	if !ok {
		return
	}
	s.providerStorageMu.Lock()
	job := s.azureCommunicationJobRouterJobs[jobID]
	if job == nil {
		s.providerStorageMu.Unlock()
		azureCommunicationJobRouterError(w, http.StatusNotFound, "JobNotFound", "job not found")
		return
	}
	job.Status = "closed"
	job.DispositionCode = azureCommunicationJobRouterFirstNonEmptyString(payload, "dispositionCode", job.DispositionCode)
	job.UpdatedAt = time.Now().UTC()
	job.LastModified = job.UpdatedAt
	fixture := azureCommunicationJobRouterJobFixtureLocked(job, s)
	s.providerStorageMu.Unlock()
	respondJSON(w, http.StatusOK, fixture)
}

func (s *Server) handleAzureCommunicationJobRouterReclassifyJob(w http.ResponseWriter, r *http.Request, jobID string) {
	payload, _, ok := azureCommunicationJobRouterReadObject(w, r)
	if !ok {
		return
	}
	s.providerStorageMu.Lock()
	job := s.azureCommunicationJobRouterJobs[jobID]
	if job == nil {
		s.providerStorageMu.Unlock()
		azureCommunicationJobRouterError(w, http.StatusNotFound, "JobNotFound", "job not found")
		return
	}
	if queueID := strings.TrimSpace(azureCommunicationJobRouterString(payload["queueId"])); queueID != "" {
		job.QueueID = queueID
	}
	if selectors, ok := payload["requestedWorkerSelectors"].([]any); ok {
		job.RequestedWorkerSelector = azureCommunicationJobRouterCloneSlice(selectors)
	}
	job.Status = "queued"
	job.AssignedWorkerID = ""
	job.AssignmentID = ""
	job.UpdatedAt = time.Now().UTC()
	job.LastModified = job.UpdatedAt
	s.azureCommunicationJobRouterReconcileOffersLocked()
	fixture := azureCommunicationJobRouterJobFixtureLocked(job, s)
	s.providerStorageMu.Unlock()
	respondJSON(w, http.StatusOK, fixture)
}

func (s *Server) handleAzureCommunicationJobRouterUnassignJob(w http.ResponseWriter, r *http.Request, jobID string) {
	_, _, ok := azureCommunicationJobRouterReadObject(w, r)
	if !ok {
		return
	}
	s.providerStorageMu.Lock()
	job := s.azureCommunicationJobRouterJobs[jobID]
	if job == nil {
		s.providerStorageMu.Unlock()
		azureCommunicationJobRouterError(w, http.StatusNotFound, "JobNotFound", "job not found")
		return
	}
	if job.AssignmentID != "" {
		delete(s.azureCommunicationJobRouterAssignments, job.AssignmentID)
	}
	if worker := s.azureCommunicationJobRouterWorkers[job.AssignedWorkerID]; worker != nil {
		filtered := make([]string, 0, len(worker.Assignments))
		for _, assignmentID := range worker.Assignments {
			if assignmentID != job.AssignmentID {
				filtered = append(filtered, assignmentID)
			}
		}
		worker.Assignments = filtered
	}
	job.AssignedWorkerID = ""
	job.AssignmentID = ""
	job.Status = "queued"
	job.UpdatedAt = time.Now().UTC()
	job.LastModified = job.UpdatedAt
	s.azureCommunicationJobRouterReconcileOffersLocked()
	fixture := azureCommunicationJobRouterJobFixtureLocked(job, s)
	s.providerStorageMu.Unlock()
	respondJSON(w, http.StatusOK, fixture)
}

func (s *Server) handleAzureCommunicationJobRouterGetInQueuePosition(w http.ResponseWriter, jobID string) {
	s.providerStorageMu.Lock()
	job := s.azureCommunicationJobRouterJobs[jobID]
	if job == nil {
		s.providerStorageMu.Unlock()
		azureCommunicationJobRouterError(w, http.StatusNotFound, "JobNotFound", "job not found")
		return
	}
	queueID := job.QueueID
	queuedJobs := make([]*azureCommunicationJobRouterJob, 0)
	for _, item := range s.azureCommunicationJobRouterJobs {
		if item.QueueID == queueID && item.Status == "queued" {
			queuedJobs = append(queuedJobs, item)
		}
	}
	s.providerStorageMu.Unlock()

	sort.Slice(queuedJobs, func(i, j int) bool {
		if queuedJobs[i].Priority == queuedJobs[j].Priority {
			if queuedJobs[i].CreatedAt.Equal(queuedJobs[j].CreatedAt) {
				return queuedJobs[i].ID < queuedJobs[j].ID
			}
			return queuedJobs[i].CreatedAt.Before(queuedJobs[j].CreatedAt)
		}
		return queuedJobs[i].Priority > queuedJobs[j].Priority
	})

	position := 0
	for i, item := range queuedJobs {
		if item.ID == jobID {
			position = i + 1
			break
		}
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"jobId":    jobID,
		"queueId":  queueID,
		"position": position,
	})
}

func (s *Server) handleAzureCommunicationJobRouterUpsertWorker(w http.ResponseWriter, r *http.Request, workerID string) {
	payload, _, ok := azureCommunicationJobRouterReadObject(w, r)
	if !ok {
		return
	}
	if !azureCommunicationJobRouterValidateWorkerPayload(payload) {
		azureCommunicationJobRouterError(w, http.StatusBadRequest, "InvalidRequest", "worker payload is invalid")
		return
	}
	now := time.Now().UTC()
	s.providerStorageMu.Lock()
	worker := s.azureCommunicationJobRouterWorkers[workerID]
	status := http.StatusCreated
	if worker == nil {
		worker = &azureCommunicationJobRouterWorker{
			ID:        workerID,
			Labels:    map[string]any{},
			Tags:      map[string]any{},
			CreatedAt: now,
		}
		s.azureCommunicationJobRouterWorkers[workerID] = worker
	} else {
		status = http.StatusOK
	}
	if queueIDs, ok := azureCommunicationJobRouterStringSlice(payload["queueIds"]); ok && len(queueIDs) > 0 {
		worker.QueueIDs = queueIDs
	}
	if labels, ok := payload["labels"].(map[string]any); ok {
		worker.Labels = azureCommunicationJobRouterCloneObject(labels)
	}
	if tags, ok := payload["tags"].(map[string]any); ok {
		worker.Tags = azureCommunicationJobRouterCloneObject(tags)
	}
	if capacity, ok := azureCommunicationJobRouterInt(payload["capacity"]); ok {
		worker.Capacity = capacity
	}
	if channelConfigurations, ok := payload["channelConfigurations"].([]any); ok {
		worker.ChannelConfigurations = azureCommunicationJobRouterCloneSlice(channelConfigurations)
	}
	worker.UpdatedAt = now
	worker.LastModified = now
	worker.ETag = azureCommunicationJobRouterDigest("worker|" + workerID + "|" + now.Format(time.RFC3339Nano))[:16]
	s.azureCommunicationJobRouterReconcileOffersLocked()
	fixture := azureCommunicationJobRouterWorkerFixtureLocked(worker, s)
	etag := worker.ETag
	lastModified := worker.LastModified
	s.providerStorageMu.Unlock()

	w.Header().Set("ETag", etag)
	w.Header().Set("Last-Modified", lastModified.Format(http.TimeFormat))
	respondJSON(w, status, fixture)
}

func (s *Server) handleAzureCommunicationJobRouterGetWorker(w http.ResponseWriter, workerID string) {
	s.providerStorageMu.Lock()
	worker := s.azureCommunicationJobRouterWorkers[workerID]
	s.providerStorageMu.Unlock()
	if worker == nil {
		azureCommunicationJobRouterError(w, http.StatusNotFound, "WorkerNotFound", "worker not found")
		return
	}
	w.Header().Set("ETag", worker.ETag)
	w.Header().Set("Last-Modified", worker.LastModified.Format(http.TimeFormat))
	respondJSON(w, http.StatusOK, azureCommunicationJobRouterWorkerFixture(worker, s))
}

func (s *Server) handleAzureCommunicationJobRouterListWorkers(w http.ResponseWriter, r *http.Request) {
	maxPageSize, ok := azureCommunicationJobRouterParseMaxPageSize(w, r)
	if !ok {
		return
	}
	s.providerStorageMu.Lock()
	items := make([]*azureCommunicationJobRouterWorker, 0, len(s.azureCommunicationJobRouterWorkers))
	for _, item := range s.azureCommunicationJobRouterWorkers {
		items = append(items, item)
	}
	s.providerStorageMu.Unlock()
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	pageSize := len(items)
	if maxPageSize > 0 && maxPageSize < pageSize {
		pageSize = maxPageSize
	}
	value := make([]any, 0, pageSize)
	for _, item := range items[:pageSize] {
		value = append(value, azureCommunicationJobRouterWorkerFixture(item, s))
	}
	respondJSON(w, http.StatusOK, map[string]any{"value": value})
}

func (s *Server) handleAzureCommunicationJobRouterDeleteWorker(w http.ResponseWriter, workerID string) {
	s.providerStorageMu.Lock()
	worker := s.azureCommunicationJobRouterWorkers[workerID]
	if worker == nil {
		s.providerStorageMu.Unlock()
		azureCommunicationJobRouterError(w, http.StatusNotFound, "WorkerNotFound", "worker not found")
		return
	}
	for _, offerID := range worker.Offers {
		delete(s.azureCommunicationJobRouterOffers, offerID)
	}
	for _, assignmentID := range worker.Assignments {
		delete(s.azureCommunicationJobRouterAssignments, assignmentID)
	}
	delete(s.azureCommunicationJobRouterWorkers, workerID)
	s.providerStorageMu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAzureCommunicationJobRouterOfferAction(w http.ResponseWriter, r *http.Request, workerID, offerID, action string) {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "accept":
		s.handleAzureCommunicationJobRouterAcceptOffer(w, workerID, offerID)
	case "decline":
		s.handleAzureCommunicationJobRouterDeclineOffer(w, r, workerID, offerID)
	default:
		azureCommunicationJobRouterError(w, http.StatusNotFound, "PathNotFound", "route not found")
	}
}

func (s *Server) handleAzureCommunicationJobRouterAcceptOffer(w http.ResponseWriter, workerID, offerID string) {
	now := time.Now().UTC()
	s.providerStorageMu.Lock()
	worker := s.azureCommunicationJobRouterWorkers[workerID]
	offer := s.azureCommunicationJobRouterOffers[offerID]
	if worker == nil {
		s.providerStorageMu.Unlock()
		azureCommunicationJobRouterError(w, http.StatusNotFound, "WorkerNotFound", "worker not found")
		return
	}
	if offer == nil || offer.WorkerID != workerID {
		s.providerStorageMu.Unlock()
		azureCommunicationJobRouterError(w, http.StatusNotFound, "OfferNotFound", "offer not found")
		return
	}
	job := s.azureCommunicationJobRouterJobs[offer.JobID]
	if job == nil {
		s.providerStorageMu.Unlock()
		azureCommunicationJobRouterError(w, http.StatusNotFound, "JobNotFound", "job not found")
		return
	}
	if offer.Status != "pending" || job.AssignmentID != "" || job.Status == "assigned" {
		s.providerStorageMu.Unlock()
		azureCommunicationJobRouterError(w, http.StatusConflict, "OfferAlreadyAccepted", "offer cannot be accepted")
		return
	}
	assignmentID := "assignment-" + azureCommunicationJobRouterDigest(job.ID + "|" + workerID)[:12]
	assignment := &azureCommunicationJobRouterAssignment{
		ID:        assignmentID,
		JobID:     job.ID,
		WorkerID:  workerID,
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.azureCommunicationJobRouterAssignments[assignmentID] = assignment
	offer.Status = "accepted"
	offer.UpdatedAt = now
	job.AssignedWorkerID = workerID
	job.AssignmentID = assignmentID
	job.Status = "assigned"
	job.UpdatedAt = now
	job.LastModified = now
	worker.Assignments = append(worker.Assignments, assignmentID)
	for _, otherOfferID := range job.OfferIDs {
		if otherOfferID == offerID {
			continue
		}
		if otherOffer := s.azureCommunicationJobRouterOffers[otherOfferID]; otherOffer != nil && otherOffer.Status == "pending" {
			otherOffer.Status = "superseded"
			otherOffer.UpdatedAt = now
		}
	}
	s.providerStorageMu.Unlock()

	respondJSON(w, http.StatusOK, map[string]any{
		"assignmentId": assignmentID,
		"jobId":        job.ID,
		"workerId":     workerID,
	})
}

func (s *Server) handleAzureCommunicationJobRouterDeclineOffer(w http.ResponseWriter, r *http.Request, workerID, offerID string) {
	payload, _, ok := azureCommunicationJobRouterReadObject(w, r)
	if !ok {
		return
	}
	var retryOfferAt *time.Time
	if raw := strings.TrimSpace(azureCommunicationJobRouterString(payload["retryOfferAt"])); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			parsed, err = time.Parse(time.RFC3339Nano, raw)
		}
		if err != nil {
			azureCommunicationJobRouterError(w, http.StatusBadRequest, "InvalidRequest", "retryOfferAt must be a valid RFC3339 timestamp")
			return
		}
		retryOfferAt = &parsed
	}

	s.providerStorageMu.Lock()
	offer := s.azureCommunicationJobRouterOffers[offerID]
	if offer == nil || offer.WorkerID != workerID {
		s.providerStorageMu.Unlock()
		azureCommunicationJobRouterError(w, http.StatusNotFound, "OfferNotFound", "offer not found")
		return
	}
	offer.Status = "declined"
	offer.RetryOfferAt = retryOfferAt
	offer.UpdatedAt = time.Now().UTC()
	s.providerStorageMu.Unlock()
	respondJSON(w, http.StatusOK, map[string]any{})
}

func (s *Server) handleAzureCommunicationJobRouterGetQueueStatistics(w http.ResponseWriter, queueID string) {
	s.providerStorageMu.Lock()
	if s.azureCommunicationJobRouterQueues[queueID] == nil {
		s.providerStorageMu.Unlock()
		azureCommunicationJobRouterError(w, http.StatusNotFound, "QueueNotFound", "queue not found")
		return
	}
	var queued, assigned int
	for _, job := range s.azureCommunicationJobRouterJobs {
		if job.QueueID != queueID {
			continue
		}
		switch job.Status {
		case "queued":
			queued++
		case "assigned":
			assigned++
		}
	}
	s.providerStorageMu.Unlock()
	respondJSON(w, http.StatusOK, map[string]any{
		"queueId":                  queueID,
		"length":                   queued + assigned,
		"activeJobs":               queued + assigned,
		"assignedJobs":             assigned,
		"queuedJobs":               queued,
		"estimatedWaitTimeMinutes": 0,
	})
}

func (s *Server) azureCommunicationJobRouterReconcileOffersLocked() {
	for _, worker := range s.azureCommunicationJobRouterWorkers {
		worker.Offers = worker.Offers[:0]
	}
	for _, job := range s.azureCommunicationJobRouterJobs {
		job.OfferIDs = job.OfferIDs[:0]
	}
	for offerID, offer := range s.azureCommunicationJobRouterOffers {
		job := s.azureCommunicationJobRouterJobs[offer.JobID]
		worker := s.azureCommunicationJobRouterWorkers[offer.WorkerID]
		if job == nil || worker == nil {
			delete(s.azureCommunicationJobRouterOffers, offerID)
			continue
		}
		if offer.Status == "pending" && job.Status == "queued" && azureCommunicationJobRouterWorkerEligible(worker, job) {
			job.OfferIDs = append(job.OfferIDs, offerID)
			worker.Offers = append(worker.Offers, offerID)
		}
	}
	for _, job := range s.azureCommunicationJobRouterJobs {
		if job.Status != "queued" {
			continue
		}
		for _, worker := range s.azureCommunicationJobRouterWorkers {
			if !azureCommunicationJobRouterWorkerEligible(worker, job) {
				continue
			}
			offerID := "offer-" + azureCommunicationJobRouterDigest(job.ID + "|" + worker.ID)[:12]
			offer := s.azureCommunicationJobRouterOffers[offerID]
			if offer == nil {
				now := time.Now().UTC()
				offer = &azureCommunicationJobRouterOffer{
					ID:        offerID,
					JobID:     job.ID,
					WorkerID:  worker.ID,
					Status:    "pending",
					CreatedAt: now,
					UpdatedAt: now,
				}
				s.azureCommunicationJobRouterOffers[offerID] = offer
			}
			if offer.Status == "accepted" || offer.Status == "superseded" {
				offer.Status = "pending"
				offer.RetryOfferAt = nil
				offer.UpdatedAt = time.Now().UTC()
			}
			if offer.Status == "pending" {
				job.OfferIDs = append(job.OfferIDs, offerID)
				worker.Offers = append(worker.Offers, offerID)
			}
		}
	}
	for _, worker := range s.azureCommunicationJobRouterWorkers {
		sort.Strings(worker.Offers)
		sort.Strings(worker.Assignments)
	}
	for _, job := range s.azureCommunicationJobRouterJobs {
		sort.Strings(job.OfferIDs)
	}
}

func azureCommunicationJobRouterWorkerEligible(worker *azureCommunicationJobRouterWorker, job *azureCommunicationJobRouterJob) bool {
	if len(worker.QueueIDs) == 0 || job.QueueID == "" {
		return true
	}
	for _, queueID := range worker.QueueIDs {
		if strings.EqualFold(queueID, job.QueueID) {
			return true
		}
	}
	return false
}

func azureCommunicationJobRouterResourceFixture(resource *azureCommunicationJobRouterResource) map[string]any {
	fixture := azureCommunicationJobRouterCloneObject(resource.Properties)
	fixture["id"] = resource.ID
	fixture["etag"] = resource.ETag
	if _, ok := fixture["name"]; !ok {
		fixture["name"] = resource.ID
	}
	return fixture
}

func azureCommunicationJobRouterJobFixture(job *azureCommunicationJobRouterJob, s *Server) map[string]any {
	if s == nil {
		return azureCommunicationJobRouterJobFixtureLocked(job, nil)
	}
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	return azureCommunicationJobRouterJobFixtureLocked(job, s)
}

func azureCommunicationJobRouterJobFixtureLocked(job *azureCommunicationJobRouterJob, s *Server) map[string]any {
	fixture := map[string]any{
		"id":                       job.ID,
		"channelId":                job.ChannelID,
		"channelReference":         job.ChannelReference,
		"classificationPolicyId":   job.ClassificationPolicyID,
		"queueId":                  job.QueueID,
		"priority":                 job.Priority,
		"status":                   job.Status,
		"dispositionCode":          job.DispositionCode,
		"labels":                   azureCommunicationJobRouterCloneObject(job.Labels),
		"tags":                     azureCommunicationJobRouterCloneObject(job.Tags),
		"etag":                     job.ETag,
		"notes":                    azureCommunicationJobRouterCloneSlice(job.Notes),
		"requestedWorkerSelectors": azureCommunicationJobRouterCloneSlice(job.RequestedWorkerSelector),
	}
	if len(job.MatchingMode) > 0 {
		fixture["matchingMode"] = azureCommunicationJobRouterCloneObject(job.MatchingMode)
	}
	if job.AssignedWorkerID != "" {
		fixture["assignedWorkerId"] = job.AssignedWorkerID
	}
	if job.AssignmentID != "" {
		fixture["assignmentId"] = job.AssignmentID
	}
	offers := make([]any, 0, len(job.OfferIDs))
	if s != nil {
		for _, offerID := range job.OfferIDs {
			if offer := s.azureCommunicationJobRouterOffers[offerID]; offer != nil {
				offers = append(offers, azureCommunicationJobRouterOfferFixture(offer))
			}
		}
	}
	fixture["offers"] = offers
	return fixture
}

func azureCommunicationJobRouterWorkerFixture(worker *azureCommunicationJobRouterWorker, s *Server) map[string]any {
	if s == nil {
		return azureCommunicationJobRouterWorkerFixtureLocked(worker, nil)
	}
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	return azureCommunicationJobRouterWorkerFixtureLocked(worker, s)
}

func azureCommunicationJobRouterWorkerFixtureLocked(worker *azureCommunicationJobRouterWorker, s *Server) map[string]any {
	queueIDs := make([]any, 0, len(worker.QueueIDs))
	for _, queueID := range worker.QueueIDs {
		queueIDs = append(queueIDs, queueID)
	}
	assignments := make([]any, 0, len(worker.Assignments))
	offers := make([]any, 0, len(worker.Offers))
	if s != nil {
		for _, assignmentID := range worker.Assignments {
			if assignment := s.azureCommunicationJobRouterAssignments[assignmentID]; assignment != nil {
				assignments = append(assignments, map[string]any{
					"assignmentId": assignment.ID,
					"jobId":        assignment.JobID,
					"workerId":     assignment.WorkerID,
				})
			}
		}
		for _, offerID := range worker.Offers {
			if offer := s.azureCommunicationJobRouterOffers[offerID]; offer != nil {
				offers = append(offers, azureCommunicationJobRouterOfferFixture(offer))
			}
		}
	}
	return map[string]any{
		"id":                    worker.ID,
		"queueIds":              queueIDs,
		"labels":                azureCommunicationJobRouterCloneObject(worker.Labels),
		"tags":                  azureCommunicationJobRouterCloneObject(worker.Tags),
		"capacity":              worker.Capacity,
		"channelConfigurations": azureCommunicationJobRouterCloneSlice(worker.ChannelConfigurations),
		"etag":                  worker.ETag,
		"offers":                offers,
		"assignments":           assignments,
	}
}

func azureCommunicationJobRouterOfferFixture(offer *azureCommunicationJobRouterOffer) map[string]any {
	fixture := map[string]any{
		"id":       offer.ID,
		"jobId":    offer.JobID,
		"workerId": offer.WorkerID,
		"status":   offer.Status,
	}
	if offer.RetryOfferAt != nil {
		fixture["retryOfferAt"] = offer.RetryOfferAt.Format(time.RFC3339Nano)
	}
	return fixture
}

func azureCommunicationJobRouterValidateResourcePayload(kind string, payload map[string]any) bool {
	switch kind {
	case "classificationPolicy":
		return len(payload) > 0
	case "distributionPolicy":
		mode, _ := payload["mode"].(map[string]any)
		return strings.TrimSpace(azureCommunicationJobRouterString(mode["kind"])) != ""
	case "exceptionPolicy":
		if rules, ok := payload["exceptionRules"].([]any); ok && len(rules) > 0 {
			return true
		}
		return false
	case "queue":
		return strings.TrimSpace(azureCommunicationJobRouterString(payload["name"])) != "" || strings.TrimSpace(azureCommunicationJobRouterString(payload["distributionPolicyId"])) != ""
	default:
		return len(payload) > 0
	}
}

func azureCommunicationJobRouterValidateJobPayload(payload map[string]any) bool {
	if strings.TrimSpace(azureCommunicationJobRouterString(payload["channelId"])) != "" {
		return true
	}
	if strings.TrimSpace(azureCommunicationJobRouterString(payload["queueId"])) != "" {
		return true
	}
	return false
}

func azureCommunicationJobRouterValidateWorkerPayload(payload map[string]any) bool {
	if _, ok := payload["queueIds"]; ok {
		return true
	}
	if _, ok := payload["capacity"]; ok {
		return true
	}
	if _, ok := payload["labels"]; ok {
		return true
	}
	return false
}

func azureCommunicationJobRouterSplitAction(segment string) (string, string, bool) {
	idx := strings.Index(segment, ":")
	if idx <= 0 || idx == len(segment)-1 {
		return "", "", false
	}
	return strings.TrimSpace(segment[:idx]), strings.TrimSpace(segment[idx+1:]), true
}

func azureCommunicationJobRouterParseMaxPageSize(w http.ResponseWriter, r *http.Request) (int, bool) {
	raw := strings.TrimSpace(r.URL.Query().Get("maxpagesize"))
	if raw == "" {
		return 0, true
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		azureCommunicationJobRouterError(w, http.StatusBadRequest, "InvalidRequest", "maxpagesize must be a positive integer")
		return 0, false
	}
	return value, true
}

func azureCommunicationJobRouterReadObject(w http.ResponseWriter, r *http.Request) (map[string]any, []byte, bool) {
	body, err := readBodyBytes(r)
	if err != nil {
		azureCommunicationJobRouterError(w, http.StatusBadRequest, "InvalidRequest", "unable to read request body")
		return nil, nil, false
	}
	if len(body) == 0 {
		body = []byte(`{}`)
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		azureCommunicationJobRouterError(w, http.StatusBadRequest, "InvalidRequest", "request body must be valid JSON")
		return nil, nil, false
	}
	if payload == nil {
		payload = map[string]any{}
	}
	return payload, body, true
}

func azureCommunicationJobRouterMergeObject(base, patch map[string]any) map[string]any {
	out := azureCommunicationJobRouterCloneObject(base)
	for key, value := range patch {
		switch typed := value.(type) {
		case map[string]any:
			if existing, ok := out[key].(map[string]any); ok {
				out[key] = azureCommunicationJobRouterMergeObject(existing, typed)
			} else {
				out[key] = azureCommunicationJobRouterCloneObject(typed)
			}
		case []any:
			out[key] = azureCommunicationJobRouterCloneSlice(typed)
		default:
			out[key] = value
		}
	}
	return out
}

func azureCommunicationJobRouterCloneObject(input map[string]any) map[string]any {
	if input == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		switch typed := value.(type) {
		case map[string]any:
			out[key] = azureCommunicationJobRouterCloneObject(typed)
		case []any:
			out[key] = azureCommunicationJobRouterCloneSlice(typed)
		default:
			out[key] = value
		}
	}
	return out
}

func azureCommunicationJobRouterCloneSlice(input []any) []any {
	if input == nil {
		return []any{}
	}
	out := make([]any, len(input))
	for i, value := range input {
		if object, ok := value.(map[string]any); ok {
			out[i] = azureCommunicationJobRouterCloneObject(object)
			continue
		}
		if slice, ok := value.([]any); ok {
			out[i] = azureCommunicationJobRouterCloneSlice(slice)
			continue
		}
		out[i] = value
	}
	return out
}

func azureCommunicationJobRouterString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return ""
	}
}

func azureCommunicationJobRouterStringSlice(value any) ([]string, bool) {
	if value == nil {
		return []string{}, true
	}
	items, ok := value.([]any)
	if !ok {
		return nil, false
	}
	out := make([]string, 0, len(items))
	for _, raw := range items {
		item := strings.TrimSpace(azureCommunicationJobRouterString(raw))
		if item == "" {
			return nil, false
		}
		out = append(out, item)
	}
	return out, true
}

func azureCommunicationJobRouterInt(value any) (int, bool) {
	switch typed := value.(type) {
	case float64:
		out := int(typed)
		if float64(out) != typed {
			return 0, false
		}
		return out, true
	default:
		return 0, false
	}
}

func azureCommunicationJobRouterFirstNonEmptyString(payload map[string]any, key, fallback string) string {
	if value := strings.TrimSpace(azureCommunicationJobRouterString(payload[key])); value != "" {
		return value
	}
	return fallback
}

func azureCommunicationJobRouterDigest(input string) string {
	sum := sha1.Sum([]byte(input))
	return hex.EncodeToString(sum[:])
}

func azureCommunicationJobRouterError(w http.ResponseWriter, status int, code, message string) {
	respondJSON(w, status, map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	})
}
