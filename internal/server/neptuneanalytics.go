package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type neptuneAnalyticsError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

type neptuneAnalyticsAPIError struct {
	Status  int
	Code    string
	Message string
}

func (e *neptuneAnalyticsAPIError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

type neptuneAnalyticsGraphCreateRequest struct {
	GraphName                 string            `json:"graphName"`
	Tags                      map[string]string `json:"tags"`
	PublicConnectivity        *bool             `json:"publicConnectivity"`
	KMSKeyIdentifier          string            `json:"kmsKeyIdentifier"`
	VectorSearchConfiguration *struct {
		Dimension *int `json:"dimension"`
	} `json:"vectorSearchConfiguration"`
	ReplicaCount             *int   `json:"replicaCount"`
	DeletionProtection       *bool  `json:"deletionProtection"`
	ProvisionedMemory        *int   `json:"provisionedMemory"`
	SourceSnapshotIdentifier string `json:"sourceSnapshotIdentifier"`
}

type neptuneAnalyticsGraphUpdateRequest struct {
	PublicConnectivity *bool `json:"publicConnectivity"`
	ProvisionedMemory  *int  `json:"provisionedMemory"`
	DeletionProtection *bool `json:"deletionProtection"`
}

type neptuneAnalyticsGraphDeleteRequest struct {
	SkipSnapshot *bool `json:"skipSnapshot"`
}

type neptuneAnalyticsSnapshotCreateRequest struct {
	GraphIdentifier string            `json:"graphIdentifier"`
	SnapshotName    string            `json:"snapshotName"`
	Tags            map[string]string `json:"tags"`
}

type neptuneAnalyticsRestoreGraphRequest struct {
	SnapshotIdentifier string            `json:"snapshotIdentifier"`
	GraphName          string            `json:"graphName"`
	ProvisionedMemory  *int              `json:"provisionedMemory"`
	DeletionProtection *bool             `json:"deletionProtection"`
	Tags               map[string]string `json:"tags"`
	ReplicaCount       *int              `json:"replicaCount"`
	PublicConnectivity *bool             `json:"publicConnectivity"`
}

type neptuneAnalyticsGraph struct {
	ID                 string
	Name               string
	Arn                string
	Status             string
	StatusReason       string
	CreateTime         time.Time
	ProvisionedMemory  int
	Endpoint           string
	PublicConnectivity bool
	VectorDimension    *int
	ReplicaCount       int
	KMSKeyIdentifier   string
	SourceSnapshotID   string
	DeletionProtection bool
	BuildNumber        string
	Tags               map[string]string
}

type neptuneAnalyticsSnapshot struct {
	ID                       string
	Name                     string
	Arn                      string
	SourceGraphID            string
	SnapshotCreateTime       time.Time
	Status                   string
	KMSKeyIdentifier         string
	Tags                     map[string]string
	SourceProvisionedMemory  int
	SourceReplicaCount       int
	SourcePublicConnectivity bool
	SourceDeletionProtection bool
}

type neptuneAnalyticsStore struct {
	mu                    sync.Mutex
	graphs                map[string]neptuneAnalyticsGraph
	graphNameToID         map[string]string
	snapshots             map[string]neptuneAnalyticsSnapshot
	snapshotNameToID      map[string]string
	importTasks           map[string]neptuneAnalyticsImportTask
	exportTasks           map[string]neptuneAnalyticsExportTask
	queries               map[string]neptuneAnalyticsQuery
	privateEndpoints      map[string]map[string]neptuneAnalyticsPrivateEndpoint
	resourceTags          map[string]map[string]string
	nextGraphID           int
	nextSnapshotID        int
	nextImportTaskID      int
	nextExportTaskID      int
	nextQueryID           int
	nextPrivateEndpointID int
}

func newNeptuneAnalyticsStore() *neptuneAnalyticsStore {
	return &neptuneAnalyticsStore{
		graphs:           map[string]neptuneAnalyticsGraph{},
		graphNameToID:    map[string]string{},
		snapshots:        map[string]neptuneAnalyticsSnapshot{},
		snapshotNameToID: map[string]string{},
		importTasks:      map[string]neptuneAnalyticsImportTask{},
		exportTasks:      map[string]neptuneAnalyticsExportTask{},
		queries:          map[string]neptuneAnalyticsQuery{},
		privateEndpoints: map[string]map[string]neptuneAnalyticsPrivateEndpoint{},
		resourceTags:     map[string]map[string]string{},
		nextGraphID:      0,
		nextSnapshotID:   0,
	}
}

func (s *Server) handleNeptuneAnalyticsRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isNeptuneAnalyticsRESTCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "neptune-graph")
	if !ok {
		respondNeptuneAnalyticsError(w, status, code, msg)
		return true
	}

	segments := splitPathSegments(rawRequestPath(r))
	if len(segments) == 0 {
		respondNeptuneAnalyticsError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
		return true
	}

	switch segments[0] {
	case "graphs":
		s.handleNeptuneAnalyticsGraphs(w, r, segments)
		return true
	case "snapshots":
		s.handleNeptuneAnalyticsSnapshots(w, r, segments)
		return true
	case "importtasks":
		s.handleNeptuneAnalyticsImportTasks(w, r, segments)
		return true
	case "exporttasks":
		s.handleNeptuneAnalyticsExportTasks(w, r, segments)
		return true
	case "queries":
		s.handleNeptuneAnalyticsQueries(w, r, segments)
		return true
	case "summary":
		s.handleNeptuneAnalyticsGetGraphSummary(w, r, segments)
		return true
	case "tags":
		s.handleNeptuneAnalyticsTags(w, r, segments)
		return true
	default:
		respondNeptuneAnalyticsError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
		return true
	}
}

func (s *Server) handleNeptuneAnalyticsGraphs(w http.ResponseWriter, r *http.Request, segments []string) {
	if len(segments) == 1 {
		switch r.Method {
		case http.MethodGet:
			s.handleNeptuneAnalyticsListGraphs(w, r)
			return
		case http.MethodPost:
			s.handleNeptuneAnalyticsCreateGraph(w, r)
			return
		default:
			respondNeptuneAnalyticsError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
			return
		}
	}

	graphIdentifier, err := decodeNeptuneAnalyticsPathValue(segments[1])
	if err != nil {
		respondNeptuneAnalyticsError(w, http.StatusBadRequest, "ValidationException", "graphIdentifier is required")
		return
	}

	if len(segments) == 2 {
		switch r.Method {
		case http.MethodGet:
			graph, err := s.neptuneanalytics.GetGraph(graphIdentifier)
			if err != nil {
				respondNeptuneAnalyticsErrorForErr(w, err)
				return
			}
			respondNeptuneAnalyticsJSON(w, http.StatusOK, neptuneAnalyticsGraphPayload(graph))
			return
		case http.MethodPatch:
			s.handleNeptuneAnalyticsUpdateGraph(w, r, graphIdentifier)
			return
		case http.MethodPut:
			s.handleNeptuneAnalyticsResetGraph(w, r, graphIdentifier)
			return
		case http.MethodDelete:
			s.handleNeptuneAnalyticsDeleteGraph(w, r, graphIdentifier)
			return
		default:
			respondNeptuneAnalyticsError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
			return
		}
	}

	if len(segments) == 3 && segments[2] == "importtasks" && r.Method == http.MethodPost {
		s.handleNeptuneAnalyticsStartImportTask(w, r, graphIdentifier)
		return
	}

	if len(segments) == 3 && segments[2] == "endpoints" {
		s.handleNeptuneAnalyticsPrivateEndpoints(w, r, graphIdentifier, "")
		return
	}

	if len(segments) == 4 && segments[2] == "endpoints" {
		vpcID, err := decodeNeptuneAnalyticsPathValue(segments[3])
		if err != nil {
			respondNeptuneAnalyticsError(w, http.StatusBadRequest, "ValidationException", "vpcId is required")
			return
		}
		s.handleNeptuneAnalyticsPrivateEndpoints(w, r, graphIdentifier, vpcID)
		return
	}

	if len(segments) == 3 && segments[2] == "start" && r.Method == http.MethodPost {
		graph, err := s.neptuneanalytics.StartGraph(graphIdentifier)
		if err != nil {
			respondNeptuneAnalyticsErrorForErr(w, err)
			return
		}
		respondNeptuneAnalyticsJSON(w, http.StatusOK, neptuneAnalyticsGraphPayload(graph))
		return
	}

	if len(segments) == 3 && segments[2] == "stop" && r.Method == http.MethodPost {
		graph, err := s.neptuneanalytics.StopGraph(graphIdentifier)
		if err != nil {
			respondNeptuneAnalyticsErrorForErr(w, err)
			return
		}
		respondNeptuneAnalyticsJSON(w, http.StatusOK, neptuneAnalyticsGraphPayload(graph))
		return
	}

	// Compatibility: support older/alternate restore route shape.
	if len(segments) == 3 && segments[2] == "restore" && r.Method == http.MethodPost {
		var req neptuneAnalyticsRestoreGraphRequest
		if err := decodeNeptuneAnalyticsJSONBody(r, &req); err != nil {
			respondNeptuneAnalyticsError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
			return
		}
		if strings.TrimSpace(req.GraphName) == "" {
			req.GraphName = graphIdentifier
		}
		if strings.TrimSpace(req.SnapshotIdentifier) == "" {
			respondNeptuneAnalyticsError(w, http.StatusBadRequest, "ValidationException", "snapshotIdentifier is required")
			return
		}
		graph, err := s.neptuneanalytics.RestoreGraphFromSnapshot(strings.TrimSpace(req.SnapshotIdentifier), req)
		if err != nil {
			respondNeptuneAnalyticsErrorForErr(w, err)
			return
		}
		respondNeptuneAnalyticsJSON(w, http.StatusOK, neptuneAnalyticsGraphPayload(graph))
		return
	}

	respondNeptuneAnalyticsError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
}

func (s *Server) handleNeptuneAnalyticsSnapshots(w http.ResponseWriter, r *http.Request, segments []string) {
	if len(segments) == 1 {
		switch r.Method {
		case http.MethodGet:
			s.handleNeptuneAnalyticsListSnapshots(w, r)
			return
		case http.MethodPost:
			s.handleNeptuneAnalyticsCreateSnapshot(w, r)
			return
		default:
			respondNeptuneAnalyticsError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
			return
		}
	}

	snapshotIdentifier, err := decodeNeptuneAnalyticsPathValue(segments[1])
	if err != nil {
		respondNeptuneAnalyticsError(w, http.StatusBadRequest, "ValidationException", "snapshotIdentifier is required")
		return
	}

	if len(segments) == 2 {
		switch r.Method {
		case http.MethodGet:
			snapshot, err := s.neptuneanalytics.GetSnapshot(snapshotIdentifier)
			if err != nil {
				respondNeptuneAnalyticsErrorForErr(w, err)
				return
			}
			respondNeptuneAnalyticsJSON(w, http.StatusOK, neptuneAnalyticsSnapshotPayload(snapshot))
			return
		case http.MethodDelete:
			snapshot, err := s.neptuneanalytics.DeleteSnapshot(snapshotIdentifier)
			if err != nil {
				respondNeptuneAnalyticsErrorForErr(w, err)
				return
			}
			respondNeptuneAnalyticsJSON(w, http.StatusOK, neptuneAnalyticsSnapshotPayload(snapshot))
			return
		default:
			respondNeptuneAnalyticsError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
			return
		}
	}

	if len(segments) == 3 && segments[2] == "restore" && r.Method == http.MethodPost {
		var req neptuneAnalyticsRestoreGraphRequest
		if err := decodeNeptuneAnalyticsJSONBody(r, &req); err != nil {
			respondNeptuneAnalyticsError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
			return
		}
		graph, err := s.neptuneanalytics.RestoreGraphFromSnapshot(snapshotIdentifier, req)
		if err != nil {
			respondNeptuneAnalyticsErrorForErr(w, err)
			return
		}
		respondNeptuneAnalyticsJSON(w, http.StatusOK, neptuneAnalyticsGraphPayload(graph))
		return
	}

	respondNeptuneAnalyticsError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
}

func (s *Server) handleNeptuneAnalyticsCreateGraph(w http.ResponseWriter, r *http.Request) {
	var req neptuneAnalyticsGraphCreateRequest
	if err := decodeNeptuneAnalyticsJSONBody(r, &req); err != nil {
		respondNeptuneAnalyticsError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return
	}
	graph, err := s.neptuneanalytics.CreateGraph(req)
	if err != nil {
		respondNeptuneAnalyticsErrorForErr(w, err)
		return
	}
	respondNeptuneAnalyticsJSON(w, http.StatusOK, neptuneAnalyticsGraphPayload(graph))
}

func (s *Server) handleNeptuneAnalyticsListGraphs(w http.ResponseWriter, r *http.Request) {
	maxResults, nextToken, err := parseNeptuneAnalyticsPagination(r)
	if err != nil {
		respondNeptuneAnalyticsErrorForErr(w, err)
		return
	}
	graphs, outNextToken, err := s.neptuneanalytics.ListGraphs(maxResults, nextToken)
	if err != nil {
		respondNeptuneAnalyticsErrorForErr(w, err)
		return
	}

	out := map[string]any{
		"graphs": neptuneAnalyticsGraphSummariesPayload(graphs),
	}
	if outNextToken != "" {
		out["nextToken"] = outNextToken
	}
	respondNeptuneAnalyticsJSON(w, http.StatusOK, out)
}

func (s *Server) handleNeptuneAnalyticsUpdateGraph(w http.ResponseWriter, r *http.Request, graphIdentifier string) {
	var req neptuneAnalyticsGraphUpdateRequest
	if err := decodeNeptuneAnalyticsJSONBody(r, &req); err != nil {
		respondNeptuneAnalyticsError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return
	}
	graph, err := s.neptuneanalytics.UpdateGraph(graphIdentifier, req)
	if err != nil {
		respondNeptuneAnalyticsErrorForErr(w, err)
		return
	}
	respondNeptuneAnalyticsJSON(w, http.StatusOK, neptuneAnalyticsGraphPayload(graph))
}

func (s *Server) handleNeptuneAnalyticsDeleteGraph(w http.ResponseWriter, r *http.Request, graphIdentifier string) {
	var req neptuneAnalyticsGraphDeleteRequest
	if err := decodeNeptuneAnalyticsJSONBody(r, &req); err != nil {
		respondNeptuneAnalyticsError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return
	}

	skipSnapshot, err := parseNeptuneAnalyticsSkipSnapshot(r, req.SkipSnapshot)
	if err != nil {
		respondNeptuneAnalyticsErrorForErr(w, err)
		return
	}

	graph, err := s.neptuneanalytics.DeleteGraph(graphIdentifier, skipSnapshot)
	if err != nil {
		respondNeptuneAnalyticsErrorForErr(w, err)
		return
	}
	respondNeptuneAnalyticsJSON(w, http.StatusOK, neptuneAnalyticsGraphPayload(graph))
}

func (s *Server) handleNeptuneAnalyticsCreateSnapshot(w http.ResponseWriter, r *http.Request) {
	var req neptuneAnalyticsSnapshotCreateRequest
	if err := decodeNeptuneAnalyticsJSONBody(r, &req); err != nil {
		respondNeptuneAnalyticsError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return
	}
	snapshot, err := s.neptuneanalytics.CreateSnapshot(req)
	if err != nil {
		respondNeptuneAnalyticsErrorForErr(w, err)
		return
	}
	respondNeptuneAnalyticsJSON(w, http.StatusOK, neptuneAnalyticsSnapshotPayload(snapshot))
}

func (s *Server) handleNeptuneAnalyticsListSnapshots(w http.ResponseWriter, r *http.Request) {
	maxResults, nextToken, err := parseNeptuneAnalyticsPagination(r)
	if err != nil {
		respondNeptuneAnalyticsErrorForErr(w, err)
		return
	}
	graphIdentifier := strings.TrimSpace(r.URL.Query().Get("graphIdentifier"))

	snapshots, outNextToken, err := s.neptuneanalytics.ListSnapshots(graphIdentifier, maxResults, nextToken)
	if err != nil {
		respondNeptuneAnalyticsErrorForErr(w, err)
		return
	}

	out := map[string]any{
		"graphSnapshots": neptuneAnalyticsSnapshotPayloadList(snapshots),
	}
	if outNextToken != "" {
		out["nextToken"] = outNextToken
	}
	respondNeptuneAnalyticsJSON(w, http.StatusOK, out)
}

func (s *neptuneAnalyticsStore) CreateGraph(req neptuneAnalyticsGraphCreateRequest) (neptuneAnalyticsGraph, error) {
	graphName := strings.TrimSpace(req.GraphName)
	if err := validateNeptuneAnalyticsName(graphName, "graphName"); err != nil {
		return neptuneAnalyticsGraph{}, err
	}
	if req.ProvisionedMemory == nil {
		return neptuneAnalyticsGraph{}, validationNeptuneAnalytics("provisionedMemory is required")
	}
	if *req.ProvisionedMemory < 128 {
		return neptuneAnalyticsGraph{}, validationNeptuneAnalytics("provisionedMemory must be >= 128")
	}
	if req.ReplicaCount != nil && (*req.ReplicaCount < 0 || *req.ReplicaCount > 2) {
		return neptuneAnalyticsGraph{}, validationNeptuneAnalytics("replicaCount must be between 0 and 2")
	}

	var dimension *int
	if req.VectorSearchConfiguration != nil && req.VectorSearchConfiguration.Dimension != nil {
		if *req.VectorSearchConfiguration.Dimension < 1 || *req.VectorSearchConfiguration.Dimension > 65535 {
			return neptuneAnalyticsGraph{}, validationNeptuneAnalytics("vectorSearchConfiguration.dimension must be between 1 and 65535")
		}
		v := *req.VectorSearchConfiguration.Dimension
		dimension = &v
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.graphNameToID[graphName]; exists {
		return neptuneAnalyticsGraph{}, conflictNeptuneAnalytics("graphName already exists")
	}

	sourceSnapshotID := strings.TrimSpace(req.SourceSnapshotIdentifier)
	if sourceSnapshotID != "" {
		if _, ok := s.resolveSnapshotIDLocked(sourceSnapshotID); !ok {
			return neptuneAnalyticsGraph{}, notFoundNeptuneAnalytics("snapshot was not found")
		}
	}

	s.nextGraphID++
	graphID := fmt.Sprintf("g-%012d", s.nextGraphID)
	now := time.Now().UTC()
	graph := neptuneAnalyticsGraph{
		ID:                 graphID,
		Name:               graphName,
		Arn:                neptuneAnalyticsGraphArn(graphID),
		Status:             "AVAILABLE",
		StatusReason:       "",
		CreateTime:         now,
		ProvisionedMemory:  *req.ProvisionedMemory,
		Endpoint:           neptuneAnalyticsGraphEndpoint(graphID),
		PublicConnectivity: req.PublicConnectivity != nil && *req.PublicConnectivity,
		VectorDimension:    dimension,
		ReplicaCount:       1,
		KMSKeyIdentifier:   strings.TrimSpace(req.KMSKeyIdentifier),
		SourceSnapshotID:   sourceSnapshotID,
		DeletionProtection: req.DeletionProtection != nil && *req.DeletionProtection,
		BuildNumber:        "stackyard-1",
		Tags:               cloneNeptuneAnalyticsTags(req.Tags),
	}
	if req.ReplicaCount != nil {
		graph.ReplicaCount = *req.ReplicaCount
	}
	if graph.KMSKeyIdentifier == "" {
		graph.KMSKeyIdentifier = "alias/aws/neptune-graph"
	}

	s.graphs[graphID] = graph
	s.graphNameToID[graph.Name] = graphID
	s.resourceTags[graph.Arn] = cloneNeptuneAnalyticsTags(graph.Tags)
	return graph, nil
}

func (s *neptuneAnalyticsStore) GetGraph(identifier string) (neptuneAnalyticsGraph, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	graphID, ok := s.resolveGraphIDLocked(identifier)
	if !ok {
		return neptuneAnalyticsGraph{}, notFoundNeptuneAnalytics("graph was not found")
	}
	return s.graphs[graphID], nil
}

func (s *neptuneAnalyticsStore) ListGraphs(maxResults int, nextToken string) ([]neptuneAnalyticsGraph, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ids := make([]string, 0, len(s.graphs))
	for id := range s.graphs {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	start, err := parseNeptuneAnalyticsNextToken(nextToken, len(ids))
	if err != nil {
		return nil, "", err
	}
	if maxResults <= 0 {
		maxResults = 100
	}

	end := start + maxResults
	if end > len(ids) {
		end = len(ids)
	}
	out := make([]neptuneAnalyticsGraph, 0, end-start)
	for _, id := range ids[start:end] {
		out = append(out, s.graphs[id])
	}

	if end < len(ids) {
		return out, strconv.Itoa(end), nil
	}
	return out, "", nil
}

func (s *neptuneAnalyticsStore) UpdateGraph(identifier string, req neptuneAnalyticsGraphUpdateRequest) (neptuneAnalyticsGraph, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	graphID, ok := s.resolveGraphIDLocked(identifier)
	if !ok {
		return neptuneAnalyticsGraph{}, notFoundNeptuneAnalytics("graph was not found")
	}
	if req.PublicConnectivity == nil && req.ProvisionedMemory == nil && req.DeletionProtection == nil {
		return neptuneAnalyticsGraph{}, validationNeptuneAnalytics("at least one update field is required")
	}

	graph := s.graphs[graphID]
	if req.ProvisionedMemory != nil {
		if *req.ProvisionedMemory < 128 {
			return neptuneAnalyticsGraph{}, validationNeptuneAnalytics("provisionedMemory must be >= 128")
		}
		graph.ProvisionedMemory = *req.ProvisionedMemory
	}
	if req.PublicConnectivity != nil {
		graph.PublicConnectivity = *req.PublicConnectivity
	}
	if req.DeletionProtection != nil {
		graph.DeletionProtection = *req.DeletionProtection
	}
	graph.Status = "AVAILABLE"
	graph.StatusReason = ""
	s.graphs[graphID] = graph
	return graph, nil
}

func (s *neptuneAnalyticsStore) DeleteGraph(identifier string, skipSnapshot bool) (neptuneAnalyticsGraph, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	graphID, ok := s.resolveGraphIDLocked(identifier)
	if !ok {
		return neptuneAnalyticsGraph{}, notFoundNeptuneAnalytics("graph was not found")
	}

	graph := s.graphs[graphID]
	if graph.DeletionProtection {
		return neptuneAnalyticsGraph{}, conflictNeptuneAnalytics("graph has deletionProtection enabled")
	}

	if !skipSnapshot {
		autoSnapshotName := neptuneAnalyticsFinalSnapshotName(graph.Name)
		for s.snapshotNameToID[autoSnapshotName] != "" {
			autoSnapshotName = neptuneAnalyticsFinalSnapshotName(graph.Name)
		}
		_, _ = s.createSnapshotLocked(neptuneAnalyticsSnapshotCreateRequest{
			GraphIdentifier: graph.ID,
			SnapshotName:    autoSnapshotName,
			Tags:            cloneNeptuneAnalyticsTags(graph.Tags),
		})
	}

	delete(s.graphNameToID, graph.Name)
	delete(s.graphs, graph.ID)
	delete(s.resourceTags, graph.Arn)

	graph.Status = "DELETING"
	graph.StatusReason = "Graph deletion requested."
	return graph, nil
}

func (s *neptuneAnalyticsStore) StartGraph(identifier string) (neptuneAnalyticsGraph, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	graphID, ok := s.resolveGraphIDLocked(identifier)
	if !ok {
		return neptuneAnalyticsGraph{}, notFoundNeptuneAnalytics("graph was not found")
	}
	graph := s.graphs[graphID]
	graph.Status = "AVAILABLE"
	graph.StatusReason = ""
	s.graphs[graphID] = graph
	return graph, nil
}

func (s *neptuneAnalyticsStore) StopGraph(identifier string) (neptuneAnalyticsGraph, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	graphID, ok := s.resolveGraphIDLocked(identifier)
	if !ok {
		return neptuneAnalyticsGraph{}, notFoundNeptuneAnalytics("graph was not found")
	}
	graph := s.graphs[graphID]
	graph.Status = "STOPPED"
	graph.StatusReason = "Graph was stopped."
	s.graphs[graphID] = graph
	return graph, nil
}

func (s *neptuneAnalyticsStore) CreateSnapshot(req neptuneAnalyticsSnapshotCreateRequest) (neptuneAnalyticsSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.createSnapshotLocked(req)
}

func (s *neptuneAnalyticsStore) createSnapshotLocked(req neptuneAnalyticsSnapshotCreateRequest) (neptuneAnalyticsSnapshot, error) {
	graphIdentifier := strings.TrimSpace(req.GraphIdentifier)
	if graphIdentifier == "" {
		return neptuneAnalyticsSnapshot{}, validationNeptuneAnalytics("graphIdentifier is required")
	}
	if err := validateNeptuneAnalyticsName(strings.TrimSpace(req.SnapshotName), "snapshotName"); err != nil {
		return neptuneAnalyticsSnapshot{}, err
	}

	graphID, ok := s.resolveGraphIDLocked(graphIdentifier)
	if !ok {
		return neptuneAnalyticsSnapshot{}, notFoundNeptuneAnalytics("graph was not found")
	}
	graph := s.graphs[graphID]

	snapshotName := strings.TrimSpace(req.SnapshotName)
	if _, exists := s.snapshotNameToID[snapshotName]; exists {
		return neptuneAnalyticsSnapshot{}, conflictNeptuneAnalytics("snapshotName already exists")
	}

	s.nextSnapshotID++
	snapshotID := fmt.Sprintf("gs-%012d", s.nextSnapshotID)
	snapshot := neptuneAnalyticsSnapshot{
		ID:                       snapshotID,
		Name:                     snapshotName,
		Arn:                      neptuneAnalyticsSnapshotArn(snapshotID),
		SourceGraphID:            graph.ID,
		SnapshotCreateTime:       time.Now().UTC(),
		Status:                   "AVAILABLE",
		KMSKeyIdentifier:         graph.KMSKeyIdentifier,
		Tags:                     cloneNeptuneAnalyticsTags(req.Tags),
		SourceProvisionedMemory:  graph.ProvisionedMemory,
		SourceReplicaCount:       graph.ReplicaCount,
		SourcePublicConnectivity: graph.PublicConnectivity,
		SourceDeletionProtection: graph.DeletionProtection,
	}
	s.snapshots[snapshotID] = snapshot
	s.snapshotNameToID[snapshotName] = snapshotID
	s.resourceTags[snapshot.Arn] = cloneNeptuneAnalyticsTags(snapshot.Tags)
	return snapshot, nil
}

func (s *neptuneAnalyticsStore) GetSnapshot(identifier string) (neptuneAnalyticsSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	snapshotID, ok := s.resolveSnapshotIDLocked(identifier)
	if !ok {
		return neptuneAnalyticsSnapshot{}, notFoundNeptuneAnalytics("snapshot was not found")
	}
	return s.snapshots[snapshotID], nil
}

func (s *neptuneAnalyticsStore) ListSnapshots(graphIdentifier string, maxResults int, nextToken string) ([]neptuneAnalyticsSnapshot, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	filterGraphID := ""
	if trimmed := strings.TrimSpace(graphIdentifier); trimmed != "" {
		graphID, ok := s.resolveGraphIDLocked(trimmed)
		if !ok {
			return nil, "", notFoundNeptuneAnalytics("graph was not found")
		}
		filterGraphID = graphID
	}

	ids := make([]string, 0, len(s.snapshots))
	for id, snapshot := range s.snapshots {
		if filterGraphID != "" && snapshot.SourceGraphID != filterGraphID {
			continue
		}
		ids = append(ids, id)
	}
	sort.Strings(ids)

	start, err := parseNeptuneAnalyticsNextToken(nextToken, len(ids))
	if err != nil {
		return nil, "", err
	}
	if maxResults <= 0 {
		maxResults = 100
	}

	end := start + maxResults
	if end > len(ids) {
		end = len(ids)
	}
	out := make([]neptuneAnalyticsSnapshot, 0, end-start)
	for _, id := range ids[start:end] {
		out = append(out, s.snapshots[id])
	}

	if end < len(ids) {
		return out, strconv.Itoa(end), nil
	}
	return out, "", nil
}

func (s *neptuneAnalyticsStore) DeleteSnapshot(identifier string) (neptuneAnalyticsSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	snapshotID, ok := s.resolveSnapshotIDLocked(identifier)
	if !ok {
		return neptuneAnalyticsSnapshot{}, notFoundNeptuneAnalytics("snapshot was not found")
	}

	snapshot := s.snapshots[snapshotID]
	delete(s.snapshotNameToID, snapshot.Name)
	delete(s.snapshots, snapshotID)
	delete(s.resourceTags, snapshot.Arn)
	snapshot.Status = "DELETING"
	return snapshot, nil
}

func (s *neptuneAnalyticsStore) RestoreGraphFromSnapshot(snapshotIdentifier string, req neptuneAnalyticsRestoreGraphRequest) (neptuneAnalyticsGraph, error) {
	graphName := strings.TrimSpace(req.GraphName)
	if err := validateNeptuneAnalyticsName(graphName, "graphName"); err != nil {
		return neptuneAnalyticsGraph{}, err
	}

	if req.ReplicaCount != nil && (*req.ReplicaCount < 0 || *req.ReplicaCount > 2) {
		return neptuneAnalyticsGraph{}, validationNeptuneAnalytics("replicaCount must be between 0 and 2")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.graphNameToID[graphName]; exists {
		return neptuneAnalyticsGraph{}, conflictNeptuneAnalytics("graphName already exists")
	}

	snapshotID, ok := s.resolveSnapshotIDLocked(snapshotIdentifier)
	if !ok {
		return neptuneAnalyticsGraph{}, notFoundNeptuneAnalytics("snapshot was not found")
	}
	snapshot := s.snapshots[snapshotID]

	provisionedMemory := snapshot.SourceProvisionedMemory
	if provisionedMemory == 0 {
		provisionedMemory = 128
	}
	if req.ProvisionedMemory != nil {
		provisionedMemory = *req.ProvisionedMemory
	}
	if provisionedMemory < 128 {
		return neptuneAnalyticsGraph{}, validationNeptuneAnalytics("provisionedMemory must be >= 128")
	}

	replicaCount := snapshot.SourceReplicaCount
	if req.ReplicaCount != nil {
		replicaCount = *req.ReplicaCount
	}
	publicConnectivity := snapshot.SourcePublicConnectivity
	if req.PublicConnectivity != nil {
		publicConnectivity = *req.PublicConnectivity
	}
	deletionProtection := snapshot.SourceDeletionProtection
	if req.DeletionProtection != nil {
		deletionProtection = *req.DeletionProtection
	}

	s.nextGraphID++
	graphID := fmt.Sprintf("g-%012d", s.nextGraphID)
	graph := neptuneAnalyticsGraph{
		ID:                 graphID,
		Name:               graphName,
		Arn:                neptuneAnalyticsGraphArn(graphID),
		Status:             "AVAILABLE",
		StatusReason:       "",
		CreateTime:         time.Now().UTC(),
		ProvisionedMemory:  provisionedMemory,
		Endpoint:           neptuneAnalyticsGraphEndpoint(graphID),
		PublicConnectivity: publicConnectivity,
		ReplicaCount:       replicaCount,
		KMSKeyIdentifier:   snapshot.KMSKeyIdentifier,
		SourceSnapshotID:   snapshot.ID,
		DeletionProtection: deletionProtection,
		BuildNumber:        "stackyard-1",
		Tags:               cloneNeptuneAnalyticsTags(req.Tags),
	}

	s.graphs[graphID] = graph
	s.graphNameToID[graphName] = graphID
	s.resourceTags[graph.Arn] = cloneNeptuneAnalyticsTags(graph.Tags)
	return graph, nil
}

func (s *neptuneAnalyticsStore) resolveGraphIDLocked(identifier string) (string, bool) {
	trimmed := strings.TrimSpace(identifier)
	if trimmed == "" {
		return "", false
	}
	if _, ok := s.graphs[trimmed]; ok {
		return trimmed, true
	}
	if graphID, ok := s.graphNameToID[trimmed]; ok {
		return graphID, true
	}
	for id, graph := range s.graphs {
		if graph.Arn == trimmed {
			return id, true
		}
	}
	return "", false
}

func (s *neptuneAnalyticsStore) resolveSnapshotIDLocked(identifier string) (string, bool) {
	trimmed := strings.TrimSpace(identifier)
	if trimmed == "" {
		return "", false
	}
	if _, ok := s.snapshots[trimmed]; ok {
		return trimmed, true
	}
	if snapshotID, ok := s.snapshotNameToID[trimmed]; ok {
		return snapshotID, true
	}
	for id, snapshot := range s.snapshots {
		if snapshot.Arn == trimmed {
			return id, true
		}
	}
	return "", false
}

func isNeptuneAnalyticsRESTCandidate(r *http.Request) bool {
	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "neptune-graph" {
		return false
	}
	if service == "neptune-graph" {
		return true
	}

	path := strings.TrimSpace(r.URL.Path)
	for _, prefix := range []string{
		"/graphs",
		"/snapshots",
		"/queries",
		"/importtasks",
		"/exporttasks",
		"/summary",
		"/tags",
	} {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".neptune-graph.") || strings.HasPrefix(host, "neptune-graph.") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#neptune-graph") || strings.Contains(userAgent, " neptune-graph/") {
		return true
	}
	amzUserAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Amz-User-Agent")))
	return strings.Contains(amzUserAgent, "command#neptune-graph") || strings.Contains(amzUserAgent, " neptune-graph/")
}

func respondNeptuneAnalyticsJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondNeptuneAnalyticsError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondNeptuneAnalyticsJSON(w, status, neptuneAnalyticsError{Type: code, Message: msg})
}

func respondNeptuneAnalyticsErrorForErr(w http.ResponseWriter, err error) {
	var apiErr *neptuneAnalyticsAPIError
	if errors.As(err, &apiErr) {
		respondNeptuneAnalyticsError(w, apiErr.Status, apiErr.Code, apiErr.Message)
		return
	}
	respondNeptuneAnalyticsError(w, http.StatusBadRequest, "ValidationException", err.Error())
}

func decodeNeptuneAnalyticsJSONBody(r *http.Request, out any) error {
	body, err := readBodyBytes(r)
	if err != nil {
		return err
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return nil
	}
	return json.Unmarshal(body, out)
}

func parseNeptuneAnalyticsPagination(r *http.Request) (int, string, error) {
	maxResults := 100
	maxResultsRaw := strings.TrimSpace(r.URL.Query().Get("maxResults"))
	if maxResultsRaw != "" {
		parsed, err := strconv.Atoi(maxResultsRaw)
		if err != nil || parsed < 1 {
			return 0, "", validationNeptuneAnalytics("maxResults must be a positive integer")
		}
		maxResults = parsed
	}
	return maxResults, strings.TrimSpace(r.URL.Query().Get("nextToken")), nil
}

func parseNeptuneAnalyticsNextToken(nextToken string, max int) (int, error) {
	if strings.TrimSpace(nextToken) == "" {
		return 0, nil
	}
	start, err := strconv.Atoi(strings.TrimSpace(nextToken))
	if err != nil || start < 0 || start > max {
		return 0, validationNeptuneAnalytics("nextToken is invalid")
	}
	return start, nil
}

func parseNeptuneAnalyticsSkipSnapshot(r *http.Request, bodyValue *bool) (bool, error) {
	queryValue := strings.TrimSpace(r.URL.Query().Get("skipSnapshot"))
	if queryValue != "" {
		parsed, err := strconv.ParseBool(queryValue)
		if err != nil {
			return false, validationNeptuneAnalytics("skipSnapshot must be true or false")
		}
		return parsed, nil
	}
	if bodyValue != nil {
		return *bodyValue, nil
	}
	return false, validationNeptuneAnalytics("skipSnapshot is required")
}

func validateNeptuneAnalyticsName(name, field string) error {
	if name == "" {
		return validationNeptuneAnalytics(field + " is required")
	}
	if len(name) > 63 {
		return validationNeptuneAnalytics(field + " must be between 1 and 63 characters")
	}
	for i, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' {
			if i == 0 && !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')) {
				return validationNeptuneAnalytics(field + " must start with a letter")
			}
			continue
		}
		return validationNeptuneAnalytics(field + " contains invalid characters")
	}
	if strings.HasSuffix(name, "-") {
		return validationNeptuneAnalytics(field + " cannot end with '-'")
	}
	if strings.Contains(name, "--") {
		return validationNeptuneAnalytics(field + " cannot contain consecutive '-'")
	}
	return nil
}

func decodeNeptuneAnalyticsPathValue(value string) (string, error) {
	decoded, err := url.PathUnescape(value)
	if err != nil {
		return "", err
	}
	decoded = strings.TrimSpace(decoded)
	if decoded == "" {
		return "", errors.New("empty path value")
	}
	return decoded, nil
}

func validationNeptuneAnalytics(msg string) error {
	return &neptuneAnalyticsAPIError{Status: http.StatusBadRequest, Code: "ValidationException", Message: msg}
}

func notFoundNeptuneAnalytics(msg string) error {
	return &neptuneAnalyticsAPIError{Status: http.StatusNotFound, Code: "ResourceNotFoundException", Message: msg}
}

func conflictNeptuneAnalytics(msg string) error {
	return &neptuneAnalyticsAPIError{Status: http.StatusConflict, Code: "ConflictException", Message: msg}
}

func cloneNeptuneAnalyticsTags(tags map[string]string) map[string]string {
	if len(tags) == 0 {
		return nil
	}
	out := make(map[string]string, len(tags))
	for k, v := range tags {
		key := strings.TrimSpace(k)
		if key == "" {
			continue
		}
		out[key] = strings.TrimSpace(v)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func neptuneAnalyticsGraphArn(graphID string) string {
	return "arn:aws:neptune-graph:us-east-1:123456789012:graph/" + graphID
}

func neptuneAnalyticsSnapshotArn(snapshotID string) string {
	return "arn:aws:neptune-graph:us-east-1:123456789012:snapshot/" + snapshotID
}

func neptuneAnalyticsGraphEndpoint(graphID string) string {
	return graphID + ".neptune-graph.us-east-1.amazonaws.com"
}

func neptuneAnalyticsFinalSnapshotName(graphName string) string {
	return fmt.Sprintf("%s-final-%d", graphName, time.Now().UTC().UnixNano())
}

func neptuneAnalyticsGraphPayload(graph neptuneAnalyticsGraph) map[string]any {
	out := map[string]any{
		"id":                 graph.ID,
		"name":               graph.Name,
		"arn":                graph.Arn,
		"status":             graph.Status,
		"statusReason":       graph.StatusReason,
		"createTime":         graph.CreateTime,
		"provisionedMemory":  graph.ProvisionedMemory,
		"endpoint":           graph.Endpoint,
		"publicConnectivity": graph.PublicConnectivity,
		"replicaCount":       graph.ReplicaCount,
		"kmsKeyIdentifier":   graph.KMSKeyIdentifier,
		"deletionProtection": graph.DeletionProtection,
		"buildNumber":        graph.BuildNumber,
	}
	if graph.VectorDimension != nil {
		out["vectorSearchConfiguration"] = map[string]any{"dimension": *graph.VectorDimension}
	}
	if graph.SourceSnapshotID != "" {
		out["sourceSnapshotId"] = graph.SourceSnapshotID
	}
	return out
}

func neptuneAnalyticsGraphSummariesPayload(graphs []neptuneAnalyticsGraph) []map[string]any {
	out := make([]map[string]any, 0, len(graphs))
	for _, graph := range graphs {
		item := map[string]any{
			"id":                 graph.ID,
			"name":               graph.Name,
			"arn":                graph.Arn,
			"status":             graph.Status,
			"provisionedMemory":  graph.ProvisionedMemory,
			"publicConnectivity": graph.PublicConnectivity,
			"endpoint":           graph.Endpoint,
			"replicaCount":       graph.ReplicaCount,
			"kmsKeyIdentifier":   graph.KMSKeyIdentifier,
			"deletionProtection": graph.DeletionProtection,
		}
		out = append(out, item)
	}
	return out
}

func neptuneAnalyticsSnapshotPayload(snapshot neptuneAnalyticsSnapshot) map[string]any {
	return map[string]any{
		"id":                 snapshot.ID,
		"name":               snapshot.Name,
		"arn":                snapshot.Arn,
		"sourceGraphId":      snapshot.SourceGraphID,
		"snapshotCreateTime": snapshot.SnapshotCreateTime,
		"status":             snapshot.Status,
		"kmsKeyIdentifier":   snapshot.KMSKeyIdentifier,
	}
}

func neptuneAnalyticsSnapshotPayloadList(snapshots []neptuneAnalyticsSnapshot) []map[string]any {
	out := make([]map[string]any, 0, len(snapshots))
	for _, snapshot := range snapshots {
		out = append(out, neptuneAnalyticsSnapshotPayload(snapshot))
	}
	return out
}
