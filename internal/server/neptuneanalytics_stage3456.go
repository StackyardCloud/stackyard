package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

type neptuneAnalyticsStartImportTaskRequest struct {
	ImportOptions     map[string]any `json:"importOptions"`
	FailOnError       *bool          `json:"failOnError"`
	Source            string         `json:"source"`
	Format            string         `json:"format"`
	ParquetType       string         `json:"parquetType"`
	BlankNodeHandling string         `json:"blankNodeHandling"`
	RoleArn           string         `json:"roleArn"`
}

type neptuneAnalyticsCreateGraphUsingImportTaskRequest struct {
	GraphName                 string            `json:"graphName"`
	Tags                      map[string]string `json:"tags"`
	PublicConnectivity        *bool             `json:"publicConnectivity"`
	KMSKeyIdentifier          string            `json:"kmsKeyIdentifier"`
	VectorSearchConfiguration *struct {
		Dimension *int `json:"dimension"`
	} `json:"vectorSearchConfiguration"`
	ReplicaCount         *int           `json:"replicaCount"`
	DeletionProtection   *bool          `json:"deletionProtection"`
	ImportOptions        map[string]any `json:"importOptions"`
	MaxProvisionedMemory *int           `json:"maxProvisionedMemory"`
	MinProvisionedMemory *int           `json:"minProvisionedMemory"`
	FailOnError          *bool          `json:"failOnError"`
	Source               string         `json:"source"`
	Format               string         `json:"format"`
	ParquetType          string         `json:"parquetType"`
	BlankNodeHandling    string         `json:"blankNodeHandling"`
	RoleArn              string         `json:"roleArn"`
}

type neptuneAnalyticsStartExportTaskRequest struct {
	GraphIdentifier  string            `json:"graphIdentifier"`
	RoleArn          string            `json:"roleArn"`
	Format           string            `json:"format"`
	Destination      string            `json:"destination"`
	KMSKeyIdentifier string            `json:"kmsKeyIdentifier"`
	ParquetType      string            `json:"parquetType"`
	ExportFilter     map[string]any    `json:"exportFilter"`
	Tags             map[string]string `json:"tags"`
}

type neptuneAnalyticsExecuteQueryRequest struct {
	QueryString              string         `json:"query"`
	Language                 string         `json:"language"`
	Parameters               map[string]any `json:"parameters"`
	PlanCache                string         `json:"planCache"`
	ExplainMode              string         `json:"explain"`
	QueryTimeoutMilliseconds *int           `json:"queryTimeoutMilliseconds"`
}

type neptuneAnalyticsCreatePrivateEndpointRequest struct {
	VpcID               string   `json:"vpcId"`
	SubnetIDs           []string `json:"subnetIds"`
	VpcSecurityGroupIDs []string `json:"vpcSecurityGroupIds"`
}

type neptuneAnalyticsTagResourceRequest struct {
	Tags map[string]string `json:"tags"`
}

type neptuneAnalyticsResetGraphRequest struct {
	SkipSnapshot *bool `json:"skipSnapshot"`
}

type neptuneAnalyticsImportTask struct {
	GraphID              string
	TaskID               string
	Source               string
	Format               string
	ParquetType          string
	RoleArn              string
	Status               string
	StatusReason         string
	ImportOptions        map[string]any
	AttemptNumber        int
	StartTime            time.Time
	TimeElapsedSeconds   int64
	ProgressPercentage   int
	ErrorCount           int
	ErrorDetails         string
	StatementCount       int64
	DictionaryEntryCount int64
}

type neptuneAnalyticsExportTask struct {
	GraphID            string
	RoleArn            string
	TaskID             string
	Status             string
	Format             string
	Destination        string
	KMSKeyIdentifier   string
	ParquetType        string
	StatusReason       string
	ExportFilter       map[string]any
	StartTime          time.Time
	TimeElapsedSeconds int64
	ProgressPercentage int
	NumVerticesWritten int64
	NumEdgesWritten    int64
}

type neptuneAnalyticsQuery struct {
	ID          string
	GraphID     string
	QueryString string
	Waited      int
	Elapsed     int
	State       string
	Language    string
	Payload     []byte
	CreatedAt   time.Time
}

type neptuneAnalyticsPrivateEndpoint struct {
	GraphID             string
	VpcID               string
	SubnetIDs           []string
	VpcSecurityGroupIDs []string
	Status              string
	VPCEndpointID       string
	Arn                 string
	Tags                map[string]string
}

func (s *Server) handleNeptuneAnalyticsImportTasks(w http.ResponseWriter, r *http.Request, segments []string) {
	if len(segments) == 1 {
		switch r.Method {
		case http.MethodPost:
			s.handleNeptuneAnalyticsCreateGraphUsingImportTask(w, r)
			return
		case http.MethodGet:
			s.handleNeptuneAnalyticsListImportTasks(w, r)
			return
		default:
			respondNeptuneAnalyticsError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
			return
		}
	}

	if len(segments) == 2 {
		taskIdentifier, err := decodeNeptuneAnalyticsPathValue(segments[1])
		if err != nil {
			respondNeptuneAnalyticsError(w, http.StatusBadRequest, "ValidationException", "taskIdentifier is required")
			return
		}
		switch r.Method {
		case http.MethodGet:
			task, err := s.neptuneanalytics.GetImportTask(taskIdentifier)
			if err != nil {
				respondNeptuneAnalyticsErrorForErr(w, err)
				return
			}
			respondNeptuneAnalyticsJSON(w, http.StatusOK, neptuneAnalyticsImportTaskPayload(task, true))
			return
		case http.MethodDelete:
			task, err := s.neptuneanalytics.CancelImportTask(taskIdentifier)
			if err != nil {
				respondNeptuneAnalyticsErrorForErr(w, err)
				return
			}
			respondNeptuneAnalyticsJSON(w, http.StatusOK, neptuneAnalyticsImportTaskPayload(task, false))
			return
		default:
			respondNeptuneAnalyticsError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
			return
		}
	}

	respondNeptuneAnalyticsError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
}

func (s *Server) handleNeptuneAnalyticsExportTasks(w http.ResponseWriter, r *http.Request, segments []string) {
	if len(segments) == 1 {
		switch r.Method {
		case http.MethodPost:
			s.handleNeptuneAnalyticsStartExportTask(w, r)
			return
		case http.MethodGet:
			s.handleNeptuneAnalyticsListExportTasks(w, r)
			return
		default:
			respondNeptuneAnalyticsError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
			return
		}
	}

	if len(segments) == 2 {
		taskIdentifier, err := decodeNeptuneAnalyticsPathValue(segments[1])
		if err != nil {
			respondNeptuneAnalyticsError(w, http.StatusBadRequest, "ValidationException", "taskIdentifier is required")
			return
		}
		switch r.Method {
		case http.MethodGet:
			task, err := s.neptuneanalytics.GetExportTask(taskIdentifier)
			if err != nil {
				respondNeptuneAnalyticsErrorForErr(w, err)
				return
			}
			respondNeptuneAnalyticsJSON(w, http.StatusOK, neptuneAnalyticsExportTaskPayload(task, true))
			return
		case http.MethodDelete:
			task, err := s.neptuneanalytics.CancelExportTask(taskIdentifier)
			if err != nil {
				respondNeptuneAnalyticsErrorForErr(w, err)
				return
			}
			respondNeptuneAnalyticsJSON(w, http.StatusOK, neptuneAnalyticsExportTaskPayload(task, false))
			return
		default:
			respondNeptuneAnalyticsError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
			return
		}
	}

	respondNeptuneAnalyticsError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
}

func (s *Server) handleNeptuneAnalyticsQueries(w http.ResponseWriter, r *http.Request, segments []string) {
	if len(segments) == 1 {
		switch r.Method {
		case http.MethodPost:
			s.handleNeptuneAnalyticsExecuteQuery(w, r)
			return
		case http.MethodGet:
			s.handleNeptuneAnalyticsListQueries(w, r)
			return
		default:
			respondNeptuneAnalyticsError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
			return
		}
	}

	if len(segments) == 2 {
		queryID, err := decodeNeptuneAnalyticsPathValue(segments[1])
		if err != nil {
			respondNeptuneAnalyticsError(w, http.StatusBadRequest, "ValidationException", "queryId is required")
			return
		}
		graphIdentifier, err := neptuneAnalyticsRequiredGraphIdentifierHeader(r)
		if err != nil {
			respondNeptuneAnalyticsErrorForErr(w, err)
			return
		}
		switch r.Method {
		case http.MethodGet:
			query, err := s.neptuneanalytics.GetQuery(graphIdentifier, queryID)
			if err != nil {
				respondNeptuneAnalyticsErrorForErr(w, err)
				return
			}
			respondNeptuneAnalyticsJSON(w, http.StatusOK, neptuneAnalyticsQueryPayload(query))
			return
		case http.MethodDelete:
			if err := s.neptuneanalytics.CancelQuery(graphIdentifier, queryID); err != nil {
				respondNeptuneAnalyticsErrorForErr(w, err)
				return
			}
			respondNeptuneAnalyticsJSON(w, http.StatusOK, map[string]any{})
			return
		default:
			respondNeptuneAnalyticsError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
			return
		}
	}

	respondNeptuneAnalyticsError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
}

func (s *Server) handleNeptuneAnalyticsGetGraphSummary(w http.ResponseWriter, r *http.Request, segments []string) {
	if len(segments) != 1 || r.Method != http.MethodGet {
		respondNeptuneAnalyticsError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
		return
	}
	graphIdentifier, err := neptuneAnalyticsRequiredGraphIdentifierHeader(r)
	if err != nil {
		respondNeptuneAnalyticsErrorForErr(w, err)
		return
	}

	mode := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("mode")))
	if mode == "" {
		mode = "BASIC"
	}
	if mode != "BASIC" && mode != "DETAILED" {
		respondNeptuneAnalyticsError(w, http.StatusBadRequest, "ValidationException", "mode must be BASIC or DETAILED")
		return
	}

	payload, err := s.neptuneanalytics.GetGraphSummary(graphIdentifier, mode)
	if err != nil {
		respondNeptuneAnalyticsErrorForErr(w, err)
		return
	}
	respondNeptuneAnalyticsJSON(w, http.StatusOK, payload)
}

func (s *Server) handleNeptuneAnalyticsTags(w http.ResponseWriter, r *http.Request, segments []string) {
	if len(segments) != 2 {
		respondNeptuneAnalyticsError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
		return
	}
	resourceARN, err := decodeNeptuneAnalyticsPathValue(segments[1])
	if err != nil {
		respondNeptuneAnalyticsError(w, http.StatusBadRequest, "ValidationException", "resourceArn is required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		tags, err := s.neptuneanalytics.ListTagsForResource(resourceARN)
		if err != nil {
			respondNeptuneAnalyticsErrorForErr(w, err)
			return
		}
		respondNeptuneAnalyticsJSON(w, http.StatusOK, map[string]any{"tags": tags})
		return
	case http.MethodPost:
		var req neptuneAnalyticsTagResourceRequest
		if err := decodeNeptuneAnalyticsJSONBody(r, &req); err != nil {
			respondNeptuneAnalyticsError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
			return
		}
		if err := s.neptuneanalytics.TagResource(resourceARN, req.Tags); err != nil {
			respondNeptuneAnalyticsErrorForErr(w, err)
			return
		}
		respondNeptuneAnalyticsJSON(w, http.StatusOK, map[string]any{})
		return
	case http.MethodDelete:
		tagKeys := r.URL.Query()["tagKeys"]
		if len(tagKeys) == 1 && strings.Contains(tagKeys[0], ",") {
			tagKeys = strings.Split(tagKeys[0], ",")
		}
		if err := s.neptuneanalytics.UntagResource(resourceARN, tagKeys); err != nil {
			respondNeptuneAnalyticsErrorForErr(w, err)
			return
		}
		respondNeptuneAnalyticsJSON(w, http.StatusOK, map[string]any{})
		return
	default:
		respondNeptuneAnalyticsError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
		return
	}
}

func (s *Server) handleNeptuneAnalyticsStartImportTask(w http.ResponseWriter, r *http.Request, graphIdentifier string) {
	var req neptuneAnalyticsStartImportTaskRequest
	if err := decodeNeptuneAnalyticsJSONBody(r, &req); err != nil {
		respondNeptuneAnalyticsError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return
	}
	task, err := s.neptuneanalytics.StartImportTask(graphIdentifier, req)
	if err != nil {
		respondNeptuneAnalyticsErrorForErr(w, err)
		return
	}
	respondNeptuneAnalyticsJSON(w, http.StatusOK, neptuneAnalyticsImportTaskPayload(task, false))
}

func (s *Server) handleNeptuneAnalyticsCreateGraphUsingImportTask(w http.ResponseWriter, r *http.Request) {
	var req neptuneAnalyticsCreateGraphUsingImportTaskRequest
	if err := decodeNeptuneAnalyticsJSONBody(r, &req); err != nil {
		respondNeptuneAnalyticsError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return
	}
	task, err := s.neptuneanalytics.CreateGraphUsingImportTask(req)
	if err != nil {
		respondNeptuneAnalyticsErrorForErr(w, err)
		return
	}
	respondNeptuneAnalyticsJSON(w, http.StatusOK, neptuneAnalyticsImportTaskPayload(task, false))
}

func (s *Server) handleNeptuneAnalyticsListImportTasks(w http.ResponseWriter, r *http.Request) {
	maxResults, nextToken, err := parseNeptuneAnalyticsPagination(r)
	if err != nil {
		respondNeptuneAnalyticsErrorForErr(w, err)
		return
	}
	tasks, outNextToken, err := s.neptuneanalytics.ListImportTasks(maxResults, nextToken)
	if err != nil {
		respondNeptuneAnalyticsErrorForErr(w, err)
		return
	}
	out := map[string]any{"tasks": neptuneAnalyticsImportTaskSummariesPayload(tasks)}
	if outNextToken != "" {
		out["nextToken"] = outNextToken
	}
	respondNeptuneAnalyticsJSON(w, http.StatusOK, out)
}

func (s *Server) handleNeptuneAnalyticsStartExportTask(w http.ResponseWriter, r *http.Request) {
	var req neptuneAnalyticsStartExportTaskRequest
	if err := decodeNeptuneAnalyticsJSONBody(r, &req); err != nil {
		respondNeptuneAnalyticsError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return
	}
	task, err := s.neptuneanalytics.StartExportTask(req)
	if err != nil {
		respondNeptuneAnalyticsErrorForErr(w, err)
		return
	}
	respondNeptuneAnalyticsJSON(w, http.StatusOK, neptuneAnalyticsExportTaskPayload(task, false))
}

func (s *Server) handleNeptuneAnalyticsListExportTasks(w http.ResponseWriter, r *http.Request) {
	maxResults, nextToken, err := parseNeptuneAnalyticsPagination(r)
	if err != nil {
		respondNeptuneAnalyticsErrorForErr(w, err)
		return
	}
	tasks, outNextToken, err := s.neptuneanalytics.ListExportTasks(maxResults, nextToken)
	if err != nil {
		respondNeptuneAnalyticsErrorForErr(w, err)
		return
	}
	out := map[string]any{"tasks": neptuneAnalyticsExportTaskSummariesPayload(tasks)}
	if outNextToken != "" {
		out["nextToken"] = outNextToken
	}
	respondNeptuneAnalyticsJSON(w, http.StatusOK, out)
}

func (s *Server) handleNeptuneAnalyticsExecuteQuery(w http.ResponseWriter, r *http.Request) {
	graphIdentifier, err := neptuneAnalyticsRequiredGraphIdentifierHeader(r)
	if err != nil {
		respondNeptuneAnalyticsErrorForErr(w, err)
		return
	}
	var req neptuneAnalyticsExecuteQueryRequest
	if err := decodeNeptuneAnalyticsJSONBody(r, &req); err != nil {
		respondNeptuneAnalyticsError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return
	}
	query, err := s.neptuneanalytics.ExecuteQuery(graphIdentifier, req)
	if err != nil {
		respondNeptuneAnalyticsErrorForErr(w, err)
		return
	}
	respondNeptuneAnalyticsBytes(w, http.StatusOK, "application/json", query.Payload)
}

func (s *Server) handleNeptuneAnalyticsListQueries(w http.ResponseWriter, r *http.Request) {
	graphIdentifier, err := neptuneAnalyticsRequiredGraphIdentifierHeader(r)
	if err != nil {
		respondNeptuneAnalyticsErrorForErr(w, err)
		return
	}
	maxResultsRaw := strings.TrimSpace(r.URL.Query().Get("maxResults"))
	if maxResultsRaw == "" {
		respondNeptuneAnalyticsError(w, http.StatusBadRequest, "ValidationException", "maxResults is required")
		return
	}
	maxResults, err := strconv.Atoi(maxResultsRaw)
	if err != nil || maxResults < 1 {
		respondNeptuneAnalyticsError(w, http.StatusBadRequest, "ValidationException", "maxResults must be a positive integer")
		return
	}
	state := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("state")))
	if state == "" {
		state = "ALL"
	}
	queries, err := s.neptuneanalytics.ListQueries(graphIdentifier, maxResults, state)
	if err != nil {
		respondNeptuneAnalyticsErrorForErr(w, err)
		return
	}
	respondNeptuneAnalyticsJSON(w, http.StatusOK, map[string]any{"queries": neptuneAnalyticsQuerySummariesPayload(queries)})
}

func (s *Server) handleNeptuneAnalyticsResetGraph(w http.ResponseWriter, r *http.Request, graphIdentifier string) {
	var req neptuneAnalyticsResetGraphRequest
	if err := decodeNeptuneAnalyticsJSONBody(r, &req); err != nil {
		respondNeptuneAnalyticsError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return
	}
	if req.SkipSnapshot == nil {
		respondNeptuneAnalyticsError(w, http.StatusBadRequest, "ValidationException", "skipSnapshot is required")
		return
	}
	graph, err := s.neptuneanalytics.ResetGraph(graphIdentifier, *req.SkipSnapshot)
	if err != nil {
		respondNeptuneAnalyticsErrorForErr(w, err)
		return
	}
	respondNeptuneAnalyticsJSON(w, http.StatusOK, neptuneAnalyticsGraphPayload(graph))
}

func (s *Server) handleNeptuneAnalyticsPrivateEndpoints(w http.ResponseWriter, r *http.Request, graphIdentifier, vpcID string) {
	if vpcID == "" {
		switch r.Method {
		case http.MethodPost:
			var req neptuneAnalyticsCreatePrivateEndpointRequest
			if err := decodeNeptuneAnalyticsJSONBody(r, &req); err != nil {
				respondNeptuneAnalyticsError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
				return
			}
			endpoint, err := s.neptuneanalytics.CreatePrivateEndpoint(graphIdentifier, req)
			if err != nil {
				respondNeptuneAnalyticsErrorForErr(w, err)
				return
			}
			respondNeptuneAnalyticsJSON(w, http.StatusOK, neptuneAnalyticsPrivateEndpointPayload(endpoint))
			return
		case http.MethodGet:
			maxResults, nextToken, err := parseNeptuneAnalyticsPagination(r)
			if err != nil {
				respondNeptuneAnalyticsErrorForErr(w, err)
				return
			}
			endpoints, outNextToken, err := s.neptuneanalytics.ListPrivateEndpoints(graphIdentifier, maxResults, nextToken)
			if err != nil {
				respondNeptuneAnalyticsErrorForErr(w, err)
				return
			}
			out := map[string]any{"privateGraphEndpoints": neptuneAnalyticsPrivateEndpointPayloads(endpoints)}
			if outNextToken != "" {
				out["nextToken"] = outNextToken
			}
			respondNeptuneAnalyticsJSON(w, http.StatusOK, out)
			return
		default:
			respondNeptuneAnalyticsError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
			return
		}
	}

	switch r.Method {
	case http.MethodGet:
		endpoint, err := s.neptuneanalytics.GetPrivateEndpoint(graphIdentifier, vpcID)
		if err != nil {
			respondNeptuneAnalyticsErrorForErr(w, err)
			return
		}
		respondNeptuneAnalyticsJSON(w, http.StatusOK, neptuneAnalyticsPrivateEndpointPayload(endpoint))
		return
	case http.MethodDelete:
		endpoint, err := s.neptuneanalytics.DeletePrivateEndpoint(graphIdentifier, vpcID)
		if err != nil {
			respondNeptuneAnalyticsErrorForErr(w, err)
			return
		}
		respondNeptuneAnalyticsJSON(w, http.StatusOK, neptuneAnalyticsPrivateEndpointPayload(endpoint))
		return
	default:
		respondNeptuneAnalyticsError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
		return
	}
}

func (s *neptuneAnalyticsStore) StartImportTask(graphIdentifier string, req neptuneAnalyticsStartImportTaskRequest) (neptuneAnalyticsImportTask, error) {
	source := strings.TrimSpace(req.Source)
	if source == "" {
		return neptuneAnalyticsImportTask{}, validationNeptuneAnalytics("source is required")
	}
	roleArn := strings.TrimSpace(req.RoleArn)
	if roleArn == "" {
		return neptuneAnalyticsImportTask{}, validationNeptuneAnalytics("roleArn is required")
	}
	format := strings.ToUpper(strings.TrimSpace(req.Format))
	if format == "" {
		format = "CSV"
	}
	if format != "CSV" && format != "OPEN_CYPHER" && format != "PARQUET" && format != "NTRIPLES" {
		return neptuneAnalyticsImportTask{}, validationNeptuneAnalytics("format is invalid")
	}
	parquetType := strings.ToUpper(strings.TrimSpace(req.ParquetType))
	if parquetType == "" {
		parquetType = "COLUMNAR"
	}
	if parquetType != "COLUMNAR" {
		return neptuneAnalyticsImportTask{}, validationNeptuneAnalytics("parquetType is invalid")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	graphID, ok := s.resolveGraphIDLocked(graphIdentifier)
	if !ok {
		return neptuneAnalyticsImportTask{}, notFoundNeptuneAnalytics("graph was not found")
	}

	s.nextImportTaskID++
	taskID := fmt.Sprintf("import-%012d", s.nextImportTaskID)
	task := neptuneAnalyticsImportTask{
		GraphID:              graphID,
		TaskID:               taskID,
		Source:               source,
		Format:               format,
		ParquetType:          parquetType,
		RoleArn:              roleArn,
		Status:               "SUCCEEDED",
		StatusReason:         "",
		ImportOptions:        req.ImportOptions,
		AttemptNumber:        1,
		StartTime:            time.Now().UTC(),
		TimeElapsedSeconds:   1,
		ProgressPercentage:   100,
		ErrorCount:           0,
		ErrorDetails:         "",
		StatementCount:       0,
		DictionaryEntryCount: 0,
	}
	s.importTasks[taskID] = task
	return task, nil
}

func (s *neptuneAnalyticsStore) CreateGraphUsingImportTask(req neptuneAnalyticsCreateGraphUsingImportTaskRequest) (neptuneAnalyticsImportTask, error) {
	if strings.TrimSpace(req.Source) == "" {
		return neptuneAnalyticsImportTask{}, validationNeptuneAnalytics("source is required")
	}
	if strings.TrimSpace(req.RoleArn) == "" {
		return neptuneAnalyticsImportTask{}, validationNeptuneAnalytics("roleArn is required")
	}
	provisioned := 128
	if req.MaxProvisionedMemory != nil && *req.MaxProvisionedMemory > 0 {
		provisioned = *req.MaxProvisionedMemory
	}
	if req.MinProvisionedMemory != nil && *req.MinProvisionedMemory > provisioned {
		provisioned = *req.MinProvisionedMemory
	}
	graph, err := s.CreateGraph(neptuneAnalyticsGraphCreateRequest{
		GraphName:                 req.GraphName,
		Tags:                      req.Tags,
		PublicConnectivity:        req.PublicConnectivity,
		KMSKeyIdentifier:          req.KMSKeyIdentifier,
		VectorSearchConfiguration: req.VectorSearchConfiguration,
		ReplicaCount:              req.ReplicaCount,
		DeletionProtection:        req.DeletionProtection,
		ProvisionedMemory:         &provisioned,
	})
	if err != nil {
		return neptuneAnalyticsImportTask{}, err
	}
	task, err := s.StartImportTask(graph.ID, neptuneAnalyticsStartImportTaskRequest{
		ImportOptions:     req.ImportOptions,
		FailOnError:       req.FailOnError,
		Source:            req.Source,
		Format:            req.Format,
		ParquetType:       req.ParquetType,
		BlankNodeHandling: req.BlankNodeHandling,
		RoleArn:           req.RoleArn,
	})
	if err != nil {
		return neptuneAnalyticsImportTask{}, err
	}
	return task, nil
}

func (s *neptuneAnalyticsStore) GetImportTask(taskIdentifier string) (neptuneAnalyticsImportTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	taskID := strings.TrimSpace(taskIdentifier)
	task, ok := s.importTasks[taskID]
	if !ok {
		return neptuneAnalyticsImportTask{}, notFoundNeptuneAnalytics("import task was not found")
	}
	return task, nil
}

func (s *neptuneAnalyticsStore) ListImportTasks(maxResults int, nextToken string) ([]neptuneAnalyticsImportTask, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ids := make([]string, 0, len(s.importTasks))
	for id := range s.importTasks {
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
	out := make([]neptuneAnalyticsImportTask, 0, end-start)
	for _, id := range ids[start:end] {
		out = append(out, s.importTasks[id])
	}
	if end < len(ids) {
		return out, strconv.Itoa(end), nil
	}
	return out, "", nil
}

func (s *neptuneAnalyticsStore) CancelImportTask(taskIdentifier string) (neptuneAnalyticsImportTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	taskID := strings.TrimSpace(taskIdentifier)
	task, ok := s.importTasks[taskID]
	if !ok {
		return neptuneAnalyticsImportTask{}, notFoundNeptuneAnalytics("import task was not found")
	}
	task.Status = "CANCELLED"
	task.StatusReason = "Import task was cancelled."
	s.importTasks[taskID] = task
	return task, nil
}

func (s *neptuneAnalyticsStore) StartExportTask(req neptuneAnalyticsStartExportTaskRequest) (neptuneAnalyticsExportTask, error) {
	graphIdentifier := strings.TrimSpace(req.GraphIdentifier)
	if graphIdentifier == "" {
		return neptuneAnalyticsExportTask{}, validationNeptuneAnalytics("graphIdentifier is required")
	}
	roleArn := strings.TrimSpace(req.RoleArn)
	if roleArn == "" {
		return neptuneAnalyticsExportTask{}, validationNeptuneAnalytics("roleArn is required")
	}
	format := strings.ToUpper(strings.TrimSpace(req.Format))
	if format == "" {
		return neptuneAnalyticsExportTask{}, validationNeptuneAnalytics("format is required")
	}
	if format != "PARQUET" && format != "CSV" {
		return neptuneAnalyticsExportTask{}, validationNeptuneAnalytics("format is invalid")
	}
	destination := strings.TrimSpace(req.Destination)
	if destination == "" {
		return neptuneAnalyticsExportTask{}, validationNeptuneAnalytics("destination is required")
	}
	kmsKey := strings.TrimSpace(req.KMSKeyIdentifier)
	if kmsKey == "" {
		return neptuneAnalyticsExportTask{}, validationNeptuneAnalytics("kmsKeyIdentifier is required")
	}
	parquetType := strings.ToUpper(strings.TrimSpace(req.ParquetType))
	if parquetType == "" {
		parquetType = "COLUMNAR"
	}
	if parquetType != "COLUMNAR" {
		return neptuneAnalyticsExportTask{}, validationNeptuneAnalytics("parquetType is invalid")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	graphID, ok := s.resolveGraphIDLocked(graphIdentifier)
	if !ok {
		return neptuneAnalyticsExportTask{}, notFoundNeptuneAnalytics("graph was not found")
	}

	s.nextExportTaskID++
	taskID := fmt.Sprintf("export-%012d", s.nextExportTaskID)
	task := neptuneAnalyticsExportTask{
		GraphID:            graphID,
		RoleArn:            roleArn,
		TaskID:             taskID,
		Status:             "SUCCEEDED",
		Format:             format,
		Destination:        destination,
		KMSKeyIdentifier:   kmsKey,
		ParquetType:        parquetType,
		StatusReason:       "",
		ExportFilter:       req.ExportFilter,
		StartTime:          time.Now().UTC(),
		TimeElapsedSeconds: 1,
		ProgressPercentage: 100,
		NumVerticesWritten: 0,
		NumEdgesWritten:    0,
	}
	s.exportTasks[taskID] = task
	return task, nil
}

func (s *neptuneAnalyticsStore) GetExportTask(taskIdentifier string) (neptuneAnalyticsExportTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	taskID := strings.TrimSpace(taskIdentifier)
	task, ok := s.exportTasks[taskID]
	if !ok {
		return neptuneAnalyticsExportTask{}, notFoundNeptuneAnalytics("export task was not found")
	}
	return task, nil
}

func (s *neptuneAnalyticsStore) ListExportTasks(maxResults int, nextToken string) ([]neptuneAnalyticsExportTask, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ids := make([]string, 0, len(s.exportTasks))
	for id := range s.exportTasks {
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
	out := make([]neptuneAnalyticsExportTask, 0, end-start)
	for _, id := range ids[start:end] {
		out = append(out, s.exportTasks[id])
	}
	if end < len(ids) {
		return out, strconv.Itoa(end), nil
	}
	return out, "", nil
}

func (s *neptuneAnalyticsStore) CancelExportTask(taskIdentifier string) (neptuneAnalyticsExportTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	taskID := strings.TrimSpace(taskIdentifier)
	task, ok := s.exportTasks[taskID]
	if !ok {
		return neptuneAnalyticsExportTask{}, notFoundNeptuneAnalytics("export task was not found")
	}
	task.Status = "CANCELLED"
	task.StatusReason = "Export task was cancelled."
	s.exportTasks[taskID] = task
	return task, nil
}

func (s *neptuneAnalyticsStore) ExecuteQuery(graphIdentifier string, req neptuneAnalyticsExecuteQueryRequest) (neptuneAnalyticsQuery, error) {
	queryString := strings.TrimSpace(req.QueryString)
	if queryString == "" {
		return neptuneAnalyticsQuery{}, validationNeptuneAnalytics("query is required")
	}
	language := strings.ToUpper(strings.TrimSpace(req.Language))
	if language == "" {
		return neptuneAnalyticsQuery{}, validationNeptuneAnalytics("language is required")
	}
	if language != "OPEN_CYPHER" {
		return neptuneAnalyticsQuery{}, validationNeptuneAnalytics("language must be OPEN_CYPHER")
	}
	planCache := strings.ToUpper(strings.TrimSpace(req.PlanCache))
	if planCache != "" && planCache != "ENABLED" && planCache != "DISABLED" && planCache != "AUTO" {
		return neptuneAnalyticsQuery{}, validationNeptuneAnalytics("planCache is invalid")
	}
	explainMode := strings.ToUpper(strings.TrimSpace(req.ExplainMode))
	if explainMode != "" && explainMode != "STATIC" && explainMode != "DETAILS" {
		return neptuneAnalyticsQuery{}, validationNeptuneAnalytics("explain is invalid")
	}
	if req.QueryTimeoutMilliseconds != nil && *req.QueryTimeoutMilliseconds < 1 {
		return neptuneAnalyticsQuery{}, validationNeptuneAnalytics("queryTimeoutMilliseconds must be >= 1")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	graphID, ok := s.resolveGraphIDLocked(graphIdentifier)
	if !ok {
		return neptuneAnalyticsQuery{}, notFoundNeptuneAnalytics("graph was not found")
	}
	s.nextQueryID++
	queryID := fmt.Sprintf("query-%012d", s.nextQueryID)
	payload, _ := json.Marshal(map[string]any{
		"queryId": queryID,
		"results": []any{},
	})
	query := neptuneAnalyticsQuery{
		ID:          queryID,
		GraphID:     graphID,
		QueryString: queryString,
		Waited:      0,
		Elapsed:     1,
		State:       "SUCCEEDED",
		Language:    language,
		Payload:     payload,
		CreatedAt:   time.Now().UTC(),
	}
	s.queries[queryID] = query
	return query, nil
}

func (s *neptuneAnalyticsStore) GetQuery(graphIdentifier, queryID string) (neptuneAnalyticsQuery, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	graphID, ok := s.resolveGraphIDLocked(graphIdentifier)
	if !ok {
		return neptuneAnalyticsQuery{}, notFoundNeptuneAnalytics("graph was not found")
	}
	query, ok := s.queries[strings.TrimSpace(queryID)]
	if !ok || query.GraphID != graphID {
		return neptuneAnalyticsQuery{}, notFoundNeptuneAnalytics("query was not found")
	}
	return query, nil
}

func (s *neptuneAnalyticsStore) ListQueries(graphIdentifier string, maxResults int, state string) ([]neptuneAnalyticsQuery, error) {
	state = strings.ToUpper(strings.TrimSpace(state))
	if state == "" {
		state = "ALL"
	}
	if state != "ALL" && state != "RUNNING" && state != "WAITING" && state != "CANCELLING" {
		return nil, validationNeptuneAnalytics("state is invalid")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	graphID, ok := s.resolveGraphIDLocked(graphIdentifier)
	if !ok {
		return nil, notFoundNeptuneAnalytics("graph was not found")
	}
	ids := make([]string, 0, len(s.queries))
	for id, query := range s.queries {
		if query.GraphID != graphID {
			continue
		}
		if state != "ALL" && query.State != state {
			continue
		}
		ids = append(ids, id)
	}
	sort.Strings(ids)
	if maxResults < 1 {
		maxResults = 1
	}
	if len(ids) > maxResults {
		ids = ids[:maxResults]
	}
	out := make([]neptuneAnalyticsQuery, 0, len(ids))
	for _, id := range ids {
		out = append(out, s.queries[id])
	}
	return out, nil
}

func (s *neptuneAnalyticsStore) CancelQuery(graphIdentifier, queryID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	graphID, ok := s.resolveGraphIDLocked(graphIdentifier)
	if !ok {
		return notFoundNeptuneAnalytics("graph was not found")
	}
	queryID = strings.TrimSpace(queryID)
	query, ok := s.queries[queryID]
	if !ok || query.GraphID != graphID {
		return notFoundNeptuneAnalytics("query was not found")
	}
	query.State = "CANCELLED"
	s.queries[queryID] = query
	return nil
}

func (s *neptuneAnalyticsStore) GetGraphSummary(graphIdentifier, mode string) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	graphID, ok := s.resolveGraphIDLocked(graphIdentifier)
	if !ok {
		return nil, notFoundNeptuneAnalytics("graph was not found")
	}
	graph := s.graphs[graphID]
	graphSummary := map[string]any{
		"numNodes":                0,
		"numEdges":                0,
		"numNodeLabels":           0,
		"numEdgeLabels":           0,
		"nodeLabels":              []string{},
		"edgeLabels":              []string{},
		"numNodeProperties":       0,
		"numEdgeProperties":       0,
		"nodeProperties":          []string{},
		"edgeProperties":          []string{},
		"totalNodePropertyValues": 0,
		"totalEdgePropertyValues": 0,
		"nodeStructures":          []map[string]any{},
		"edgeStructures":          []map[string]any{},
	}
	if mode == "BASIC" {
		delete(graphSummary, "nodeStructures")
		delete(graphSummary, "edgeStructures")
	}
	return map[string]any{
		"version":                       "v1",
		"lastStatisticsComputationTime": time.Now().UTC(),
		"graphSummary":                  graphSummary,
		"graph":                         neptuneAnalyticsGraphSummariesPayload([]neptuneAnalyticsGraph{graph})[0],
	}, nil
}

func (s *neptuneAnalyticsStore) CreatePrivateEndpoint(graphIdentifier string, req neptuneAnalyticsCreatePrivateEndpointRequest) (neptuneAnalyticsPrivateEndpoint, error) {
	vpcID := strings.TrimSpace(req.VpcID)
	if vpcID == "" {
		vpcID = "vpc-0123456789abcdef0"
	}
	subnetIDs := make([]string, 0, len(req.SubnetIDs))
	for _, subnetID := range req.SubnetIDs {
		trimmed := strings.TrimSpace(subnetID)
		if trimmed != "" {
			subnetIDs = append(subnetIDs, trimmed)
		}
	}
	if len(subnetIDs) == 0 {
		subnetIDs = []string{"subnet-0123456789abcdef0"}
	}
	vpcSecurityGroupIDs := make([]string, 0, len(req.VpcSecurityGroupIDs))
	for _, sgID := range req.VpcSecurityGroupIDs {
		trimmed := strings.TrimSpace(sgID)
		if trimmed != "" {
			vpcSecurityGroupIDs = append(vpcSecurityGroupIDs, trimmed)
		}
	}
	if len(vpcSecurityGroupIDs) == 0 {
		vpcSecurityGroupIDs = []string{"sg-0123456789abcdef0"}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	graphID, ok := s.resolveGraphIDLocked(graphIdentifier)
	if !ok {
		return neptuneAnalyticsPrivateEndpoint{}, notFoundNeptuneAnalytics("graph was not found")
	}
	if s.privateEndpoints[graphID] == nil {
		s.privateEndpoints[graphID] = map[string]neptuneAnalyticsPrivateEndpoint{}
	}
	if _, exists := s.privateEndpoints[graphID][vpcID]; exists {
		return neptuneAnalyticsPrivateEndpoint{}, conflictNeptuneAnalytics("private endpoint already exists for vpcId")
	}
	s.nextPrivateEndpointID++
	endpoint := neptuneAnalyticsPrivateEndpoint{
		GraphID:             graphID,
		VpcID:               vpcID,
		SubnetIDs:           subnetIDs,
		VpcSecurityGroupIDs: vpcSecurityGroupIDs,
		Status:              "AVAILABLE",
		VPCEndpointID:       fmt.Sprintf("vpce-%012d", s.nextPrivateEndpointID),
		Arn:                 neptuneAnalyticsPrivateEndpointArn(graphID, vpcID),
		Tags:                map[string]string{},
	}
	s.privateEndpoints[graphID][vpcID] = endpoint
	s.resourceTags[endpoint.Arn] = map[string]string{}
	return endpoint, nil
}

func (s *neptuneAnalyticsStore) GetPrivateEndpoint(graphIdentifier, vpcID string) (neptuneAnalyticsPrivateEndpoint, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	graphID, ok := s.resolveGraphIDLocked(graphIdentifier)
	if !ok {
		return neptuneAnalyticsPrivateEndpoint{}, notFoundNeptuneAnalytics("graph was not found")
	}
	endpoint, ok := s.privateEndpoints[graphID][strings.TrimSpace(vpcID)]
	if !ok {
		return neptuneAnalyticsPrivateEndpoint{}, notFoundNeptuneAnalytics("private graph endpoint was not found")
	}
	return endpoint, nil
}

func (s *neptuneAnalyticsStore) ListPrivateEndpoints(graphIdentifier string, maxResults int, nextToken string) ([]neptuneAnalyticsPrivateEndpoint, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	graphID, ok := s.resolveGraphIDLocked(graphIdentifier)
	if !ok {
		return nil, "", notFoundNeptuneAnalytics("graph was not found")
	}
	endpointsMap := s.privateEndpoints[graphID]
	if endpointsMap == nil {
		return []neptuneAnalyticsPrivateEndpoint{}, "", nil
	}
	vpcIDs := make([]string, 0, len(endpointsMap))
	for vpcID := range endpointsMap {
		vpcIDs = append(vpcIDs, vpcID)
	}
	sort.Strings(vpcIDs)
	start, err := parseNeptuneAnalyticsNextToken(nextToken, len(vpcIDs))
	if err != nil {
		return nil, "", err
	}
	if maxResults <= 0 {
		maxResults = 100
	}
	end := start + maxResults
	if end > len(vpcIDs) {
		end = len(vpcIDs)
	}
	out := make([]neptuneAnalyticsPrivateEndpoint, 0, end-start)
	for _, vpcID := range vpcIDs[start:end] {
		out = append(out, endpointsMap[vpcID])
	}
	if end < len(vpcIDs) {
		return out, strconv.Itoa(end), nil
	}
	return out, "", nil
}

func (s *neptuneAnalyticsStore) DeletePrivateEndpoint(graphIdentifier, vpcID string) (neptuneAnalyticsPrivateEndpoint, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	graphID, ok := s.resolveGraphIDLocked(graphIdentifier)
	if !ok {
		return neptuneAnalyticsPrivateEndpoint{}, notFoundNeptuneAnalytics("graph was not found")
	}
	endpoint, ok := s.privateEndpoints[graphID][strings.TrimSpace(vpcID)]
	if !ok {
		return neptuneAnalyticsPrivateEndpoint{}, notFoundNeptuneAnalytics("private graph endpoint was not found")
	}
	delete(s.privateEndpoints[graphID], endpoint.VpcID)
	delete(s.resourceTags, endpoint.Arn)
	endpoint.Status = "DELETING"
	return endpoint, nil
}

func (s *neptuneAnalyticsStore) ResetGraph(graphIdentifier string, skipSnapshot bool) (neptuneAnalyticsGraph, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	graphID, ok := s.resolveGraphIDLocked(graphIdentifier)
	if !ok {
		return neptuneAnalyticsGraph{}, notFoundNeptuneAnalytics("graph was not found")
	}
	graph := s.graphs[graphID]
	if !skipSnapshot {
		autoName := neptuneAnalyticsFinalSnapshotName(graph.Name)
		for s.snapshotNameToID[autoName] != "" {
			autoName = neptuneAnalyticsFinalSnapshotName(graph.Name)
		}
		_, _ = s.createSnapshotLocked(neptuneAnalyticsSnapshotCreateRequest{
			GraphIdentifier: graph.ID,
			SnapshotName:    autoName,
			Tags:            cloneNeptuneAnalyticsTags(graph.Tags),
		})
	}
	graph.Status = "AVAILABLE"
	graph.StatusReason = "Graph reset completed."
	s.graphs[graphID] = graph
	return graph, nil
}

func (s *neptuneAnalyticsStore) ListTagsForResource(resourceARN string) (map[string]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	resourceARN = strings.TrimSpace(resourceARN)
	if !s.resourceExistsLocked(resourceARN) {
		return nil, notFoundNeptuneAnalytics("resource was not found")
	}
	tags := cloneNeptuneAnalyticsTags(s.resourceTags[resourceARN])
	if tags == nil {
		tags = map[string]string{}
	}
	return tags, nil
}

func (s *neptuneAnalyticsStore) TagResource(resourceARN string, tags map[string]string) error {
	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" {
		return validationNeptuneAnalytics("resourceArn is required")
	}
	if len(tags) == 0 {
		return validationNeptuneAnalytics("tags are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.resourceExistsLocked(resourceARN) {
		return notFoundNeptuneAnalytics("resource was not found")
	}
	if s.resourceTags[resourceARN] == nil {
		s.resourceTags[resourceARN] = map[string]string{}
	}
	for key, value := range tags {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		s.resourceTags[resourceARN][trimmedKey] = strings.TrimSpace(value)
	}
	s.syncResourceTagsLocked(resourceARN)
	return nil
}

func (s *neptuneAnalyticsStore) UntagResource(resourceARN string, tagKeys []string) error {
	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" {
		return validationNeptuneAnalytics("resourceArn is required")
	}
	if len(tagKeys) == 0 {
		return validationNeptuneAnalytics("tagKeys are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.resourceExistsLocked(resourceARN) {
		return notFoundNeptuneAnalytics("resource was not found")
	}
	tags := s.resourceTags[resourceARN]
	for _, key := range tagKeys {
		trimmed := strings.TrimSpace(key)
		if trimmed == "" {
			continue
		}
		delete(tags, trimmed)
	}
	s.resourceTags[resourceARN] = tags
	s.syncResourceTagsLocked(resourceARN)
	return nil
}

func (s *neptuneAnalyticsStore) resourceExistsLocked(resourceARN string) bool {
	if _, ok := s.resourceTags[resourceARN]; ok {
		return true
	}
	for _, graph := range s.graphs {
		if graph.Arn == resourceARN {
			return true
		}
	}
	for _, snapshot := range s.snapshots {
		if snapshot.Arn == resourceARN {
			return true
		}
	}
	for _, endpoints := range s.privateEndpoints {
		for _, endpoint := range endpoints {
			if endpoint.Arn == resourceARN {
				return true
			}
		}
	}
	return false
}

func (s *neptuneAnalyticsStore) syncResourceTagsLocked(resourceARN string) {
	for id, graph := range s.graphs {
		if graph.Arn == resourceARN {
			graph.Tags = cloneNeptuneAnalyticsTags(s.resourceTags[resourceARN])
			s.graphs[id] = graph
			return
		}
	}
	for id, snapshot := range s.snapshots {
		if snapshot.Arn == resourceARN {
			snapshot.Tags = cloneNeptuneAnalyticsTags(s.resourceTags[resourceARN])
			s.snapshots[id] = snapshot
			return
		}
	}
	for graphID, endpoints := range s.privateEndpoints {
		for vpcID, endpoint := range endpoints {
			if endpoint.Arn == resourceARN {
				endpoint.Tags = cloneNeptuneAnalyticsTags(s.resourceTags[resourceARN])
				endpoints[vpcID] = endpoint
				s.privateEndpoints[graphID] = endpoints
				return
			}
		}
	}
}

func neptuneAnalyticsRequiredGraphIdentifierHeader(r *http.Request) (string, error) {
	graphIdentifier := strings.TrimSpace(r.Header.Get("graphIdentifier"))
	if graphIdentifier == "" {
		return "", validationNeptuneAnalytics("graphIdentifier header is required")
	}
	return graphIdentifier, nil
}

func neptuneAnalyticsImportTaskPayload(task neptuneAnalyticsImportTask, includeDetails bool) map[string]any {
	out := map[string]any{
		"graphId": task.GraphID,
		"taskId":  task.TaskID,
		"source":  task.Source,
		"format":  task.Format,
		"roleArn": task.RoleArn,
		"status":  task.Status,
	}
	if task.ImportOptions != nil {
		out["importOptions"] = task.ImportOptions
	}
	if includeDetails {
		out["attemptNumber"] = task.AttemptNumber
		out["statusReason"] = task.StatusReason
		out["importTaskDetails"] = map[string]any{
			"status":               task.Status,
			"startTime":            task.StartTime,
			"timeElapsedSeconds":   task.TimeElapsedSeconds,
			"progressPercentage":   task.ProgressPercentage,
			"errorCount":           task.ErrorCount,
			"errorDetails":         task.ErrorDetails,
			"statementCount":       task.StatementCount,
			"dictionaryEntryCount": task.DictionaryEntryCount,
		}
	}
	return out
}

func neptuneAnalyticsImportTaskSummaryPayload(task neptuneAnalyticsImportTask) map[string]any {
	return map[string]any{
		"graphId": task.GraphID,
		"taskId":  task.TaskID,
		"source":  task.Source,
		"format":  task.Format,
		"roleArn": task.RoleArn,
		"status":  task.Status,
	}
}

func neptuneAnalyticsImportTaskSummariesPayload(tasks []neptuneAnalyticsImportTask) []map[string]any {
	out := make([]map[string]any, 0, len(tasks))
	for _, task := range tasks {
		out = append(out, neptuneAnalyticsImportTaskSummaryPayload(task))
	}
	return out
}

func neptuneAnalyticsExportTaskPayload(task neptuneAnalyticsExportTask, includeDetails bool) map[string]any {
	out := map[string]any{
		"graphId":          task.GraphID,
		"roleArn":          task.RoleArn,
		"taskId":           task.TaskID,
		"status":           task.Status,
		"format":           task.Format,
		"destination":      task.Destination,
		"kmsKeyIdentifier": task.KMSKeyIdentifier,
		"parquetType":      task.ParquetType,
		"statusReason":     task.StatusReason,
	}
	if task.ExportFilter != nil {
		out["exportFilter"] = task.ExportFilter
	}
	if includeDetails {
		out["exportTaskDetails"] = map[string]any{
			"startTime":          task.StartTime,
			"timeElapsedSeconds": task.TimeElapsedSeconds,
			"progressPercentage": task.ProgressPercentage,
			"numVerticesWritten": task.NumVerticesWritten,
			"numEdgesWritten":    task.NumEdgesWritten,
		}
	}
	return out
}

func neptuneAnalyticsExportTaskSummaryPayload(task neptuneAnalyticsExportTask) map[string]any {
	return map[string]any{
		"graphId":          task.GraphID,
		"roleArn":          task.RoleArn,
		"taskId":           task.TaskID,
		"status":           task.Status,
		"format":           task.Format,
		"destination":      task.Destination,
		"kmsKeyIdentifier": task.KMSKeyIdentifier,
		"parquetType":      task.ParquetType,
		"statusReason":     task.StatusReason,
	}
}

func neptuneAnalyticsExportTaskSummariesPayload(tasks []neptuneAnalyticsExportTask) []map[string]any {
	out := make([]map[string]any, 0, len(tasks))
	for _, task := range tasks {
		out = append(out, neptuneAnalyticsExportTaskSummaryPayload(task))
	}
	return out
}

func neptuneAnalyticsQueryPayload(query neptuneAnalyticsQuery) map[string]any {
	return map[string]any{
		"id":          query.ID,
		"queryString": query.QueryString,
		"waited":      query.Waited,
		"elapsed":     query.Elapsed,
		"state":       query.State,
	}
}

func neptuneAnalyticsQuerySummariesPayload(queries []neptuneAnalyticsQuery) []map[string]any {
	out := make([]map[string]any, 0, len(queries))
	for _, query := range queries {
		out = append(out, neptuneAnalyticsQueryPayload(query))
	}
	return out
}

func neptuneAnalyticsPrivateEndpointPayload(endpoint neptuneAnalyticsPrivateEndpoint) map[string]any {
	return map[string]any{
		"vpcId":         endpoint.VpcID,
		"subnetIds":     endpoint.SubnetIDs,
		"status":        endpoint.Status,
		"vpcEndpointId": endpoint.VPCEndpointID,
	}
}

func neptuneAnalyticsPrivateEndpointPayloads(endpoints []neptuneAnalyticsPrivateEndpoint) []map[string]any {
	out := make([]map[string]any, 0, len(endpoints))
	for _, endpoint := range endpoints {
		out = append(out, neptuneAnalyticsPrivateEndpointPayload(endpoint))
	}
	return out
}

func neptuneAnalyticsPrivateEndpointArn(graphID, vpcID string) string {
	return "arn:aws:neptune-graph:us-east-1:123456789012:graph-endpoint/" + graphID + "/" + vpcID
}

func respondNeptuneAnalyticsBytes(w http.ResponseWriter, status int, contentType string, payload []byte) {
	if strings.TrimSpace(contentType) == "" {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(status)
	_, _ = w.Write(payload)
}
