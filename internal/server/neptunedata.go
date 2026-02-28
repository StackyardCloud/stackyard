package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	neturl "net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type neptuneDataError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleNeptuneDataRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isNeptuneDataRESTCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "neptune-db")
	if !ok {
		respondNeptuneDataError(w, status, code, msg)
		return true
	}

	path := strings.TrimSpace(r.URL.Path)
	switch {
	// Stage 1: read/status surface.
	case r.Method == http.MethodGet && path == "/status":
		s.handleNeptuneDataGetEngineStatus(w, r)
	case r.Method == http.MethodGet && path == "/gremlin/status":
		s.handleNeptuneDataListGremlinQueries(w, r)
	case r.Method == http.MethodGet && strings.HasPrefix(path, "/gremlin/status/"):
		s.handleNeptuneDataGetGremlinQueryStatus(w, r)
	case r.Method == http.MethodGet && path == "/opencypher/status":
		s.handleNeptuneDataListOpenCypherQueries(w, r)
	case r.Method == http.MethodGet && strings.HasPrefix(path, "/opencypher/status/"):
		s.handleNeptuneDataGetOpenCypherQueryStatus(w, r)
	case r.Method == http.MethodGet && path == "/propertygraph/statistics/summary":
		s.handleNeptuneDataGetPropertygraphSummary(w, r)
	case r.Method == http.MethodGet && path == "/rdf/statistics/summary":
		s.handleNeptuneDataGetRDFGraphSummary(w, r)

	// Stage 2: query execution and cancellation.
	case r.Method == http.MethodPost && path == "/gremlin":
		s.handleNeptuneDataExecuteGremlinQuery(w, r)
	case r.Method == http.MethodPost && path == "/gremlin/explain":
		s.handleNeptuneDataExecuteGremlinExplainQuery(w, r)
	case r.Method == http.MethodPost && path == "/gremlin/profile":
		s.handleNeptuneDataExecuteGremlinProfileQuery(w, r)
	case r.Method == http.MethodDelete && strings.HasPrefix(path, "/gremlin/status/"):
		s.handleNeptuneDataCancelGremlinQuery(w, r)
	case r.Method == http.MethodPost && path == "/opencypher":
		s.handleNeptuneDataExecuteOpenCypherQuery(w, r)
	case r.Method == http.MethodPost && path == "/opencypher/explain":
		s.handleNeptuneDataExecuteOpenCypherExplainQuery(w, r)
	case r.Method == http.MethodDelete && strings.HasPrefix(path, "/opencypher/status/"):
		s.handleNeptuneDataCancelOpenCypherQuery(w, r)

	// Stage 3: loader workflows.
	case r.Method == http.MethodPost && path == "/loader":
		s.handleNeptuneDataStartLoaderJob(w, r)
	case r.Method == http.MethodGet && path == "/loader":
		s.handleNeptuneDataListLoaderJobs(w, r)
	case r.Method == http.MethodGet && strings.HasPrefix(path, "/loader/"):
		s.handleNeptuneDataGetLoaderJobStatus(w, r)
	case r.Method == http.MethodDelete && strings.HasPrefix(path, "/loader/"):
		s.handleNeptuneDataCancelLoaderJob(w, r)

	// Stage 4: statistics and stream workflows.
	case path == "/propertygraph/statistics":
		s.handleNeptuneDataPropertygraphStatistics(w, r)
	case r.Method == http.MethodGet && path == "/propertygraph/stream":
		s.handleNeptuneDataGetPropertygraphStream(w, r)
	case path == "/sparql/statistics":
		s.handleNeptuneDataSparqlStatistics(w, r)
	case r.Method == http.MethodGet && path == "/sparql/stream":
		s.handleNeptuneDataGetSparqlStream(w, r)

	// Stage 5: ML data-processing, model, and endpoint lifecycle.
	case path == "/ml/dataprocessing":
		s.handleNeptuneDataMLDataProcessing(w, r)
	case strings.HasPrefix(path, "/ml/dataprocessing/"):
		s.handleNeptuneDataMLDataProcessingItem(w, r)
	case path == "/ml/modeltraining":
		s.handleNeptuneDataMLModelTraining(w, r)
	case strings.HasPrefix(path, "/ml/modeltraining/"):
		s.handleNeptuneDataMLModelTrainingItem(w, r)
	case path == "/ml/modeltransform":
		s.handleNeptuneDataMLModelTransform(w, r)
	case strings.HasPrefix(path, "/ml/modeltransform/"):
		s.handleNeptuneDataMLModelTransformItem(w, r)
	case path == "/ml/endpoints":
		s.handleNeptuneDataMLEndpoints(w, r)
	case strings.HasPrefix(path, "/ml/endpoints/"):
		s.handleNeptuneDataMLEndpointItem(w, r)

	// Stage 6: compatibility hardening, including fast reset path.
	case r.Method == http.MethodPost && path == "/system":
		s.handleNeptuneDataExecuteFastReset(w, r)

	default:
		respondNeptuneDataError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
	}
	return true
}

func isNeptuneDataRESTCandidate(r *http.Request) bool {
	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "neptune-db" {
		return false
	}
	if service == "neptune-db" {
		return true
	}

	path := strings.TrimSpace(r.URL.Path)
	for _, prefix := range []string{
		"/status",
		"/loader",
		"/gremlin",
		"/opencypher",
		"/propertygraph",
		"/rdf",
		"/sparql",
		"/ml",
		"/system",
	} {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".neptune-db.") || strings.Contains(host, ".neptune.") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#neptunedata") || strings.Contains(userAgent, " neptunedata/") {
		return true
	}
	amzUserAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Amz-User-Agent")))
	return strings.Contains(amzUserAgent, "command#neptunedata") || strings.Contains(amzUserAgent, " neptunedata/")
}

func respondNeptuneDataJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondNeptuneDataError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondNeptuneDataJSON(w, status, neptuneDataError{Type: code, Message: msg})
}

func (s *Server) handleNeptuneDataGetEngineStatus(w http.ResponseWriter, _ *http.Request) {
	respondNeptuneDataJSON(w, http.StatusOK, map[string]any{
		"status":          "healthy",
		"startTime":       time.Now().UTC().Add(-30 * time.Minute).Format(time.RFC3339),
		"dbEngineVersion": "neptune-1.3.0.0",
		"role":            "writer",
		"dfeQueryEngine":  "enabled",
		"gremlin":         map[string]any{"version": "tinkerpop-3.7.1"},
		"sparql":          map[string]any{"version": "1.1"},
		"opencypher":      map[string]any{"version": "Neptune-OC-1.4.2"},
		"labMode":         map[string]any{},
		"features":        map[string]any{},
		"settings":        map[string]any{},
	})
}

func (s *Server) handleNeptuneDataListGremlinQueries(w http.ResponseWriter, r *http.Request) {
	includeWaiting := strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("includeWaiting")), "true")
	queries := s.neptunedata.ListQueries("gremlin", includeWaiting)
	respondNeptuneDataJSON(w, http.StatusOK, neptuneDataListQueriesResponse(queries))
}

func (s *Server) handleNeptuneDataGetGremlinQueryStatus(w http.ResponseWriter, r *http.Request) {
	queryID := pathSuffix(r.URL.Path, "/gremlin/status/")
	if queryID == "" {
		respondNeptuneDataError(w, http.StatusBadRequest, "BadRequestException", "query id is required")
		return
	}
	query := s.neptunedata.GetOrCreateQuery("gremlin", queryID)
	respondNeptuneDataJSON(w, http.StatusOK, neptuneDataQueryStatusResponse(query))
}

func (s *Server) handleNeptuneDataCancelGremlinQuery(w http.ResponseWriter, r *http.Request) {
	queryID := pathSuffix(r.URL.Path, "/gremlin/status/")
	if queryID == "" {
		respondNeptuneDataError(w, http.StatusBadRequest, "BadRequestException", "query id is required")
		return
	}
	query := s.neptunedata.CancelQuery("gremlin", queryID)
	respondNeptuneDataJSON(w, http.StatusOK, map[string]any{
		"queryId":     query.ID,
		"queryStatus": query.Status,
		"cancelled":   true,
	})
}

func (s *Server) handleNeptuneDataExecuteGremlinQuery(w http.ResponseWriter, r *http.Request) {
	payload, err := parseNeptuneDataPayload(r)
	if err != nil {
		respondNeptuneDataError(w, http.StatusBadRequest, "BadRequestException", "invalid JSON body")
		return
	}
	queryString := neptuneDataStringField(payload, "gremlin", "gremlinQuery")
	if queryString == "" {
		respondNeptuneDataError(w, http.StatusBadRequest, "BadRequestException", "gremlin query is required")
		return
	}
	query := s.neptunedata.ExecuteQuery("gremlin", queryString)
	respondNeptuneDataJSON(w, http.StatusOK, map[string]any{
		"queryId": query.ID,
		"result":  []any{},
		"meta": map[string]any{
			"queryStatus": query.Status,
		},
	})
}

func (s *Server) handleNeptuneDataExecuteGremlinExplainQuery(w http.ResponseWriter, r *http.Request) {
	payload, err := parseNeptuneDataPayload(r)
	if err != nil {
		respondNeptuneDataError(w, http.StatusBadRequest, "BadRequestException", "invalid JSON body")
		return
	}
	queryString := neptuneDataStringField(payload, "gremlin", "gremlinQuery")
	if queryString == "" {
		respondNeptuneDataError(w, http.StatusBadRequest, "BadRequestException", "gremlin query is required")
		return
	}
	query := s.neptunedata.ExecuteQuery("gremlin", queryString)
	respondNeptuneDataJSON(w, http.StatusOK, map[string]any{
		"queryId": query.ID,
		"output":  "Neptune explain plan unavailable in stage-2 emulation.",
	})
}

func (s *Server) handleNeptuneDataExecuteGremlinProfileQuery(w http.ResponseWriter, r *http.Request) {
	payload, err := parseNeptuneDataPayload(r)
	if err != nil {
		respondNeptuneDataError(w, http.StatusBadRequest, "BadRequestException", "invalid JSON body")
		return
	}
	queryString := neptuneDataStringField(payload, "gremlin", "gremlinQuery")
	if queryString == "" {
		respondNeptuneDataError(w, http.StatusBadRequest, "BadRequestException", "gremlin query is required")
		return
	}
	query := s.neptunedata.ExecuteQuery("gremlin", queryString)
	respondNeptuneDataJSON(w, http.StatusOK, map[string]any{
		"queryId": query.ID,
		"profile": map[string]any{
			"elapsedMillis": time.Since(query.CreatedAt).Milliseconds(),
			"resultCount":   0,
		},
	})
}

func (s *Server) handleNeptuneDataListOpenCypherQueries(w http.ResponseWriter, r *http.Request) {
	includeWaiting := strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("includeWaiting")), "true")
	queries := s.neptunedata.ListQueries("opencypher", includeWaiting)
	respondNeptuneDataJSON(w, http.StatusOK, neptuneDataListQueriesResponse(queries))
}

func (s *Server) handleNeptuneDataGetOpenCypherQueryStatus(w http.ResponseWriter, r *http.Request) {
	queryID := pathSuffix(r.URL.Path, "/opencypher/status/")
	if queryID == "" {
		respondNeptuneDataError(w, http.StatusBadRequest, "BadRequestException", "query id is required")
		return
	}
	query := s.neptunedata.GetOrCreateQuery("opencypher", queryID)
	respondNeptuneDataJSON(w, http.StatusOK, neptuneDataQueryStatusResponse(query))
}

func (s *Server) handleNeptuneDataCancelOpenCypherQuery(w http.ResponseWriter, r *http.Request) {
	queryID := pathSuffix(r.URL.Path, "/opencypher/status/")
	if queryID == "" {
		respondNeptuneDataError(w, http.StatusBadRequest, "BadRequestException", "query id is required")
		return
	}
	query := s.neptunedata.CancelQuery("opencypher", queryID)
	respondNeptuneDataJSON(w, http.StatusOK, map[string]any{
		"queryId":     query.ID,
		"queryStatus": query.Status,
		"cancelled":   true,
	})
}

func (s *Server) handleNeptuneDataExecuteOpenCypherQuery(w http.ResponseWriter, r *http.Request) {
	payload, err := parseNeptuneDataPayload(r)
	if err != nil {
		respondNeptuneDataError(w, http.StatusBadRequest, "BadRequestException", "invalid JSON body")
		return
	}
	queryString := neptuneDataStringField(payload, "query", "openCypherQuery")
	if queryString == "" {
		respondNeptuneDataError(w, http.StatusBadRequest, "BadRequestException", "openCypher query is required")
		return
	}
	query := s.neptunedata.ExecuteQuery("opencypher", queryString)
	respondNeptuneDataJSON(w, http.StatusOK, map[string]any{
		"queryId": query.ID,
		"results": []any{},
		"summary": map[string]any{
			"queryStatus": query.Status,
		},
	})
}

func (s *Server) handleNeptuneDataExecuteOpenCypherExplainQuery(w http.ResponseWriter, r *http.Request) {
	payload, err := parseNeptuneDataPayload(r)
	if err != nil {
		respondNeptuneDataError(w, http.StatusBadRequest, "BadRequestException", "invalid JSON body")
		return
	}
	queryString := neptuneDataStringField(payload, "query", "openCypherQuery")
	if queryString == "" {
		respondNeptuneDataError(w, http.StatusBadRequest, "BadRequestException", "openCypher query is required")
		return
	}
	query := s.neptunedata.ExecuteQuery("opencypher", queryString)
	respondNeptuneDataJSON(w, http.StatusOK, map[string]any{
		"queryId": query.ID,
		"output":  "Neptune openCypher explain output unavailable in stage-2 emulation.",
	})
}

func (s *Server) handleNeptuneDataGetPropertygraphSummary(w http.ResponseWriter, _ *http.Request) {
	respondNeptuneDataJSON(w, http.StatusOK, map[string]any{
		"statusCode": 200,
		"payload": map[string]any{
			"version":                       "v1",
			"lastStatisticsComputationTime": time.Now().UTC().Format(time.RFC3339),
			"graphSummary": map[string]any{
				"numNodes":      0,
				"numEdges":      0,
				"numNodeLabels": 0,
				"numEdgeLabels": 0,
				"nodeLabels":    []string{},
				"edgeLabels":    []string{},
			},
		},
	})
}

func (s *Server) handleNeptuneDataGetRDFGraphSummary(w http.ResponseWriter, _ *http.Request) {
	respondNeptuneDataJSON(w, http.StatusOK, map[string]any{
		"statusCode": 200,
		"payload": map[string]any{
			"version":                       "v1",
			"lastStatisticsComputationTime": time.Now().UTC().Format(time.RFC3339),
			"graphSummary": map[string]any{
				"numDistinctSubjects":   0,
				"numDistinctPredicates": 0,
				"numQuads":              0,
				"numClasses":            0,
				"classes":               []string{},
			},
		},
	})
}

func (s *Server) handleNeptuneDataStartLoaderJob(w http.ResponseWriter, r *http.Request) {
	payload, err := parseNeptuneDataPayload(r)
	if err != nil {
		respondNeptuneDataError(w, http.StatusBadRequest, "BadRequestException", "invalid JSON body")
		return
	}

	source := neptuneDataStringField(payload, "source")
	format := neptuneDataStringField(payload, "format")
	region := neptuneDataStringField(payload, "s3BucketRegion", "region")
	roleArn := neptuneDataStringField(payload, "iamRoleArn")
	if source == "" || format == "" || region == "" || roleArn == "" {
		respondNeptuneDataError(
			w,
			http.StatusBadRequest,
			"BadRequestException",
			"source, format, s3BucketRegion, and iamRoleArn are required",
		)
		return
	}

	job := s.neptunedata.StartLoaderJob(source, format, region, roleArn)
	respondNeptuneDataJSON(w, http.StatusOK, map[string]any{
		"status": "200 OK",
		"payload": map[string]any{
			"loadId":    job.ID,
			"loadIds":   []string{job.ID},
			"fullUri":   job.Source,
			"runNumber": 1,
		},
	})
}

func (s *Server) handleNeptuneDataListLoaderJobs(w http.ResponseWriter, r *http.Request) {
	limit, err := parseNeptuneDataOptionalPositiveInt(r.URL.Query().Get("limit"), 100)
	if err != nil {
		respondNeptuneDataError(w, http.StatusBadRequest, "BadRequestException", "limit must be a positive integer")
		return
	}
	if _, err := parseNeptuneDataOptionalBool(r.URL.Query().Get("includeQueuedLoads")); err != nil {
		respondNeptuneDataError(w, http.StatusBadRequest, "BadRequestException", "includeQueuedLoads must be true or false")
		return
	}

	loadIDs := s.neptunedata.ListLoaderJobs(limit)
	respondNeptuneDataJSON(w, http.StatusOK, map[string]any{
		"status": "200 OK",
		"payload": map[string]any{
			"loadIds": loadIDs,
		},
	})
}

func (s *Server) handleNeptuneDataGetLoaderJobStatus(w http.ResponseWriter, r *http.Request) {
	loadID := pathSuffix(r.URL.Path, "/loader/")
	if loadID == "" {
		respondNeptuneDataError(w, http.StatusBadRequest, "BadRequestException", "load id is required")
		return
	}
	if _, err := parseNeptuneDataOptionalBool(r.URL.Query().Get("details")); err != nil {
		respondNeptuneDataError(w, http.StatusBadRequest, "BadRequestException", "details must be true or false")
		return
	}
	if _, err := parseNeptuneDataOptionalBool(r.URL.Query().Get("errors")); err != nil {
		respondNeptuneDataError(w, http.StatusBadRequest, "BadRequestException", "errors must be true or false")
		return
	}
	page, err := parseNeptuneDataOptionalPositiveInt(r.URL.Query().Get("page"), 1)
	if err != nil {
		respondNeptuneDataError(w, http.StatusBadRequest, "BadRequestException", "page must be a positive integer")
		return
	}
	errorsPerPage, err := parseNeptuneDataOptionalPositiveInt(r.URL.Query().Get("errorsPerPage"), 10)
	if err != nil {
		respondNeptuneDataError(w, http.StatusBadRequest, "BadRequestException", "errorsPerPage must be a positive integer")
		return
	}

	job := s.neptunedata.GetOrCreateLoaderJob(loadID)
	respondNeptuneDataJSON(w, http.StatusOK, map[string]any{
		"status": "200 OK",
		"payload": map[string]any{
			"loadId":          job.ID,
			"overallStatus":   neptuneDataLoaderJobStatusPayload(job),
			"feedCount":       []any{},
			"errors":          neptuneDataLoaderJobErrorsPayload(job, page, errorsPerPage),
			"retryNumber":     0,
			"lastUpdatedTime": job.UpdatedAt.UnixMilli(),
		},
	})
}

func (s *Server) handleNeptuneDataCancelLoaderJob(w http.ResponseWriter, r *http.Request) {
	loadID := pathSuffix(r.URL.Path, "/loader/")
	if loadID == "" {
		respondNeptuneDataError(w, http.StatusBadRequest, "BadRequestException", "load id is required")
		return
	}
	job := s.neptunedata.CancelLoaderJob(loadID)
	respondNeptuneDataJSON(w, http.StatusOK, map[string]any{
		"status": fmt.Sprintf("200 OK (%s)", job.Status),
	})
}

func (s *Server) handleNeptuneDataPropertygraphStatistics(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		stats := s.neptunedata.GetStatistics("propertygraph")
		respondNeptuneDataJSON(w, http.StatusOK, map[string]any{
			"status":  "200 OK",
			"payload": neptuneDataStatisticsPayload(stats),
		})
	case http.MethodPost:
		payload, err := parseNeptuneDataPayload(r)
		if err != nil {
			respondNeptuneDataError(w, http.StatusBadRequest, "BadRequestException", "invalid JSON body")
			return
		}
		mode := neptuneDataStringField(payload, "mode")
		stats, err := s.neptunedata.ManageStatistics("propertygraph", mode)
		if err != nil {
			respondNeptuneDataError(w, http.StatusBadRequest, "BadRequestException", err.Error())
			return
		}
		respondNeptuneDataJSON(w, http.StatusOK, map[string]any{
			"status": "200 OK",
			"payload": map[string]any{
				"statisticsId": stats.ID,
			},
		})
	case http.MethodDelete:
		stats := s.neptunedata.DeleteStatistics("propertygraph")
		respondNeptuneDataJSON(w, http.StatusOK, map[string]any{
			"statusCode": 200,
			"status":     "200 OK",
			"payload": map[string]any{
				"active":       stats.Active,
				"statisticsId": stats.ID,
			},
		})
	default:
		respondNeptuneDataError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
	}
}

func (s *Server) handleNeptuneDataGetPropertygraphStream(w http.ResponseWriter, r *http.Request) {
	limit, err := parseNeptuneDataOptionalPositiveInt(r.URL.Query().Get("limit"), 10)
	if err != nil {
		respondNeptuneDataError(w, http.StatusBadRequest, "BadRequestException", "limit must be a positive integer")
		return
	}
	iteratorType := strings.TrimSpace(r.URL.Query().Get("iteratorType"))
	if iteratorType != "" && !neptuneDataAllowedIteratorType(iteratorType) {
		respondNeptuneDataError(w, http.StatusBadRequest, "BadRequestException", "iteratorType is invalid")
		return
	}
	commitNum := strings.TrimSpace(r.URL.Query().Get("commitNum"))
	opNum := strings.TrimSpace(r.URL.Query().Get("opNum"))
	if commitNum == "" {
		commitNum = "1"
	}
	if opNum == "" {
		opNum = "1"
	}

	recordCount := limit
	if recordCount > 5 {
		recordCount = 5
	}
	nowMillis := time.Now().UTC().UnixMilli()
	records := make([]map[string]any, 0, recordCount)
	for i := 0; i < recordCount; i++ {
		records = append(records, map[string]any{
			"commitTimestampInMillis": nowMillis,
			"eventId": map[string]string{
				"commitNum": commitNum,
				"opNum":     strconv.Itoa(i + 1),
			},
			"data": map[string]any{
				"id":    fmt.Sprintf("vertex-%d", i+1),
				"type":  "vertex",
				"key":   "label",
				"value": "node",
			},
			"op":       "ADD",
			"isLastOp": i+1 == recordCount,
		})
	}

	respondNeptuneDataJSON(w, http.StatusOK, map[string]any{
		"lastEventId": map[string]string{
			"commitNum": commitNum,
			"opNum":     strconv.Itoa(recordCount),
		},
		"lastTrxTimestampInMillis": nowMillis,
		"format":                   "PG_JSON",
		"records":                  records,
		"totalRecords":             len(records),
	})
}

func (s *Server) handleNeptuneDataSparqlStatistics(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		stats := s.neptunedata.GetStatistics("sparql")
		respondNeptuneDataJSON(w, http.StatusOK, map[string]any{
			"status":  "200 OK",
			"payload": neptuneDataStatisticsPayload(stats),
		})
	case http.MethodPost:
		payload, err := parseNeptuneDataPayload(r)
		if err != nil {
			respondNeptuneDataError(w, http.StatusBadRequest, "BadRequestException", "invalid JSON body")
			return
		}
		mode := neptuneDataStringField(payload, "mode")
		stats, err := s.neptunedata.ManageStatistics("sparql", mode)
		if err != nil {
			respondNeptuneDataError(w, http.StatusBadRequest, "BadRequestException", err.Error())
			return
		}
		respondNeptuneDataJSON(w, http.StatusOK, map[string]any{
			"status": "200 OK",
			"payload": map[string]any{
				"statisticsId": stats.ID,
			},
		})
	case http.MethodDelete:
		stats := s.neptunedata.DeleteStatistics("sparql")
		respondNeptuneDataJSON(w, http.StatusOK, map[string]any{
			"statusCode": 200,
			"status":     "200 OK",
			"payload": map[string]any{
				"active":       stats.Active,
				"statisticsId": stats.ID,
			},
		})
	default:
		respondNeptuneDataError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
	}
}

func (s *Server) handleNeptuneDataGetSparqlStream(w http.ResponseWriter, r *http.Request) {
	limit, err := parseNeptuneDataOptionalPositiveInt(r.URL.Query().Get("limit"), 10)
	if err != nil {
		respondNeptuneDataError(w, http.StatusBadRequest, "BadRequestException", "limit must be a positive integer")
		return
	}
	iteratorType := strings.TrimSpace(r.URL.Query().Get("iteratorType"))
	if iteratorType != "" && !neptuneDataAllowedIteratorType(iteratorType) {
		respondNeptuneDataError(w, http.StatusBadRequest, "BadRequestException", "iteratorType is invalid")
		return
	}
	commitNum := strings.TrimSpace(r.URL.Query().Get("commitNum"))
	opNum := strings.TrimSpace(r.URL.Query().Get("opNum"))
	if commitNum == "" {
		commitNum = "1"
	}
	if opNum == "" {
		opNum = "1"
	}

	recordCount := limit
	if recordCount > 5 {
		recordCount = 5
	}
	nowMillis := time.Now().UTC().UnixMilli()
	records := make([]map[string]any, 0, recordCount)
	for i := 0; i < recordCount; i++ {
		records = append(records, map[string]any{
			"commitTimestampInMillis": nowMillis,
			"eventId": map[string]string{
				"commitNum": commitNum,
				"opNum":     strconv.Itoa(i + 1),
			},
			"data": map[string]any{
				"stmt": fmt.Sprintf("<s%d> <p> <o> .", i+1),
			},
			"op":       "ADD",
			"isLastOp": i+1 == recordCount,
		})
	}

	respondNeptuneDataJSON(w, http.StatusOK, map[string]any{
		"lastEventId": map[string]string{
			"commitNum": commitNum,
			"opNum":     strconv.Itoa(recordCount),
		},
		"lastTrxTimestampInMillis": nowMillis,
		"format":                   "RDF_JSON",
		"records":                  records,
		"totalRecords":             len(records),
	})
}

func (s *Server) handleNeptuneDataMLDataProcessing(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		payload, err := parseNeptuneDataPayload(r)
		if err != nil {
			respondNeptuneDataError(w, http.StatusBadRequest, "BadRequestException", "invalid JSON body")
			return
		}
		if neptuneDataStringField(payload, "inputDataS3Location") == "" || neptuneDataStringField(payload, "processedDataS3Location") == "" {
			respondNeptuneDataError(
				w,
				http.StatusBadRequest,
				"BadRequestException",
				"inputDataS3Location and processedDataS3Location are required",
			)
			return
		}
		id := neptuneDataStringField(payload, "id")
		job := s.neptunedata.StartMLJob("dataprocessing", id)
		respondNeptuneDataJSON(w, http.StatusOK, map[string]any{
			"id":                   job.ID,
			"arn":                  job.ARN,
			"creationTimeInMillis": job.CreatedAt.UnixMilli(),
		})
	case http.MethodGet:
		maxItems, err := parseNeptuneDataOptionalPositiveInt(r.URL.Query().Get("maxItems"), 100)
		if err != nil {
			respondNeptuneDataError(w, http.StatusBadRequest, "BadRequestException", "maxItems must be a positive integer")
			return
		}
		ids := s.neptunedata.ListMLJobs("dataprocessing", maxItems)
		respondNeptuneDataJSON(w, http.StatusOK, map[string]any{"ids": ids})
	default:
		respondNeptuneDataError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
	}
}

func (s *Server) handleNeptuneDataMLDataProcessingItem(w http.ResponseWriter, r *http.Request) {
	id := pathSuffix(r.URL.Path, "/ml/dataprocessing/")
	if id == "" {
		respondNeptuneDataError(w, http.StatusBadRequest, "BadRequestException", "id is required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		job := s.neptunedata.GetOrCreateMLJob("dataprocessing", id)
		respondNeptuneDataJSON(w, http.StatusOK, map[string]any{
			"status":        "200 OK",
			"id":            job.ID,
			"processingJob": neptuneDataMLResourcePayload(job),
		})
	case http.MethodDelete:
		job := s.neptunedata.CancelMLJob("dataprocessing", id)
		respondNeptuneDataJSON(w, http.StatusOK, map[string]any{
			"status": fmt.Sprintf("200 OK (%s)", job.Status),
		})
	default:
		respondNeptuneDataError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
	}
}

func (s *Server) handleNeptuneDataMLModelTraining(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		payload, err := parseNeptuneDataPayload(r)
		if err != nil {
			respondNeptuneDataError(w, http.StatusBadRequest, "BadRequestException", "invalid JSON body")
			return
		}
		if neptuneDataStringField(payload, "dataProcessingJobId") == "" || neptuneDataStringField(payload, "trainModelS3Location") == "" {
			respondNeptuneDataError(
				w,
				http.StatusBadRequest,
				"BadRequestException",
				"dataProcessingJobId and trainModelS3Location are required",
			)
			return
		}
		id := neptuneDataStringField(payload, "id")
		job := s.neptunedata.StartMLJob("modeltraining", id)
		respondNeptuneDataJSON(w, http.StatusOK, map[string]any{
			"id":                   job.ID,
			"arn":                  job.ARN,
			"creationTimeInMillis": job.CreatedAt.UnixMilli(),
		})
	case http.MethodGet:
		maxItems, err := parseNeptuneDataOptionalPositiveInt(r.URL.Query().Get("maxItems"), 100)
		if err != nil {
			respondNeptuneDataError(w, http.StatusBadRequest, "BadRequestException", "maxItems must be a positive integer")
			return
		}
		ids := s.neptunedata.ListMLJobs("modeltraining", maxItems)
		respondNeptuneDataJSON(w, http.StatusOK, map[string]any{"ids": ids})
	default:
		respondNeptuneDataError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
	}
}

func (s *Server) handleNeptuneDataMLModelTrainingItem(w http.ResponseWriter, r *http.Request) {
	id := pathSuffix(r.URL.Path, "/ml/modeltraining/")
	if id == "" {
		respondNeptuneDataError(w, http.StatusBadRequest, "BadRequestException", "id is required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		job := s.neptunedata.GetOrCreateMLJob("modeltraining", id)
		respondNeptuneDataJSON(w, http.StatusOK, map[string]any{
			"status":            "200 OK",
			"id":                job.ID,
			"processingJob":     neptuneDataMLResourcePayload(job),
			"hpoJob":            neptuneDataMLResourcePayload(job),
			"modelTransformJob": neptuneDataMLResourcePayload(job),
			"mlModels": []map[string]any{{
				"name": fmt.Sprintf("%s-model", job.ID),
				"arn":  neptuneDataARN("ml-model", job.ID),
			}},
		})
	case http.MethodDelete:
		job := s.neptunedata.CancelMLJob("modeltraining", id)
		respondNeptuneDataJSON(w, http.StatusOK, map[string]any{
			"status": fmt.Sprintf("200 OK (%s)", job.Status),
		})
	default:
		respondNeptuneDataError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
	}
}

func (s *Server) handleNeptuneDataMLModelTransform(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		payload, err := parseNeptuneDataPayload(r)
		if err != nil {
			respondNeptuneDataError(w, http.StatusBadRequest, "BadRequestException", "invalid JSON body")
			return
		}
		if neptuneDataStringField(payload, "modelTransformOutputS3Location") == "" {
			respondNeptuneDataError(w, http.StatusBadRequest, "BadRequestException", "modelTransformOutputS3Location is required")
			return
		}
		id := neptuneDataStringField(payload, "id")
		job := s.neptunedata.StartMLJob("modeltransform", id)
		respondNeptuneDataJSON(w, http.StatusOK, map[string]any{
			"id":                   job.ID,
			"arn":                  job.ARN,
			"creationTimeInMillis": job.CreatedAt.UnixMilli(),
		})
	case http.MethodGet:
		maxItems, err := parseNeptuneDataOptionalPositiveInt(r.URL.Query().Get("maxItems"), 100)
		if err != nil {
			respondNeptuneDataError(w, http.StatusBadRequest, "BadRequestException", "maxItems must be a positive integer")
			return
		}
		ids := s.neptunedata.ListMLJobs("modeltransform", maxItems)
		respondNeptuneDataJSON(w, http.StatusOK, map[string]any{"ids": ids})
	default:
		respondNeptuneDataError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
	}
}

func (s *Server) handleNeptuneDataMLModelTransformItem(w http.ResponseWriter, r *http.Request) {
	id := pathSuffix(r.URL.Path, "/ml/modeltransform/")
	if id == "" {
		respondNeptuneDataError(w, http.StatusBadRequest, "BadRequestException", "id is required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		job := s.neptunedata.GetOrCreateMLJob("modeltransform", id)
		respondNeptuneDataJSON(w, http.StatusOK, map[string]any{
			"status":                  "200 OK",
			"id":                      job.ID,
			"baseProcessingJob":       neptuneDataMLResourcePayload(job),
			"remoteModelTransformJob": neptuneDataMLResourcePayload(job),
			"models": []map[string]any{{
				"name": fmt.Sprintf("%s-model", job.ID),
				"arn":  neptuneDataARN("ml-model", job.ID),
			}},
		})
	case http.MethodDelete:
		job := s.neptunedata.CancelMLJob("modeltransform", id)
		respondNeptuneDataJSON(w, http.StatusOK, map[string]any{
			"status": fmt.Sprintf("200 OK (%s)", job.Status),
		})
	default:
		respondNeptuneDataError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
	}
}

func (s *Server) handleNeptuneDataMLEndpoints(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		payload, err := parseNeptuneDataPayload(r)
		if err != nil {
			respondNeptuneDataError(w, http.StatusBadRequest, "BadRequestException", "invalid JSON body")
			return
		}
		id := neptuneDataStringField(payload, "id")
		modelName := neptuneDataStringField(payload, "modelName")
		trainingJobID := neptuneDataStringField(payload, "mlModelTrainingJobId")
		transformJobID := neptuneDataStringField(payload, "mlModelTransformJobId")
		endpoint := s.neptunedata.CreateMLEndpoint(id, modelName, trainingJobID, transformJobID)
		respondNeptuneDataJSON(w, http.StatusOK, map[string]any{
			"id":                   endpoint.ID,
			"arn":                  endpoint.ARN,
			"creationTimeInMillis": endpoint.CreatedAt.UnixMilli(),
		})
	case http.MethodGet:
		maxItems, err := parseNeptuneDataOptionalPositiveInt(r.URL.Query().Get("maxItems"), 100)
		if err != nil {
			respondNeptuneDataError(w, http.StatusBadRequest, "BadRequestException", "maxItems must be a positive integer")
			return
		}
		ids := s.neptunedata.ListMLEndpoints(maxItems)
		respondNeptuneDataJSON(w, http.StatusOK, map[string]any{"ids": ids})
	default:
		respondNeptuneDataError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
	}
}

func (s *Server) handleNeptuneDataMLEndpointItem(w http.ResponseWriter, r *http.Request) {
	id := pathSuffix(r.URL.Path, "/ml/endpoints/")
	if id == "" {
		respondNeptuneDataError(w, http.StatusBadRequest, "BadRequestException", "id is required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		endpoint := s.neptunedata.GetOrCreateMLEndpoint(id)
		respondNeptuneDataJSON(w, http.StatusOK, map[string]any{
			"status": "200 OK",
			"id":     endpoint.ID,
			"endpoint": map[string]any{
				"name":   endpoint.ID,
				"arn":    endpoint.ARN,
				"status": endpoint.Status,
			},
			"endpointConfig": map[string]any{
				"name": fmt.Sprintf("%s-config", endpoint.ID),
				"arn":  neptuneDataARN("ml-endpoint-config", endpoint.ID),
			},
		})
	case http.MethodDelete:
		endpoint := s.neptunedata.DeleteMLEndpoint(id)
		respondNeptuneDataJSON(w, http.StatusOK, map[string]any{
			"status": fmt.Sprintf("200 OK (%s)", endpoint.Status),
		})
	default:
		respondNeptuneDataError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
	}
}

func (s *Server) handleNeptuneDataExecuteFastReset(w http.ResponseWriter, r *http.Request) {
	payload, err := parseNeptuneDataPayload(r)
	if err != nil {
		respondNeptuneDataError(w, http.StatusBadRequest, "BadRequestException", "invalid JSON body")
		return
	}
	action := neptuneDataStringField(payload, "action")
	if action == "" {
		respondNeptuneDataError(w, http.StatusBadRequest, "BadRequestException", "action is required")
		return
	}
	token := neptuneDataStringField(payload, "token")
	status, outToken := s.neptunedata.ExecuteFastReset(action, token)

	out := map[string]any{"status": status}
	if strings.TrimSpace(outToken) != "" {
		out["payload"] = map[string]any{"token": outToken}
	}
	respondNeptuneDataJSON(w, http.StatusOK, out)
}

func parseNeptuneDataPayload(r *http.Request) (map[string]any, error) {
	body, err := readBodyBytes(r)
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return map[string]any{}, nil
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func pathSuffix(path, prefix string) string {
	suffix := strings.TrimPrefix(strings.TrimSpace(path), strings.TrimSpace(prefix))
	suffix = strings.Trim(suffix, "/")
	if suffix == "" {
		return ""
	}
	decoded, err := neturl.PathUnescape(suffix)
	if err != nil {
		return suffix
	}
	return strings.TrimSpace(decoded)
}

func neptuneDataStringField(payload map[string]any, keys ...string) string {
	for _, key := range keys {
		raw, ok := payload[key]
		if !ok || raw == nil {
			continue
		}
		if value, ok := raw.(string); ok {
			value = strings.TrimSpace(value)
			if value != "" {
				return value
			}
		}
	}
	return ""
}

func parseNeptuneDataOptionalPositiveInt(raw string, defaultValue int) (int, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return defaultValue, nil
	}
	value, err := strconv.Atoi(trimmed)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("invalid integer")
	}
	return value, nil
}

func parseNeptuneDataOptionalBool(raw string) (bool, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return false, nil
	}
	if strings.EqualFold(trimmed, "true") {
		return true, nil
	}
	if strings.EqualFold(trimmed, "false") {
		return false, nil
	}
	return false, fmt.Errorf("invalid boolean")
}

func neptuneDataAllowedIteratorType(value string) bool {
	switch strings.TrimSpace(value) {
	case "AT_SEQUENCE_NUMBER", "AFTER_SEQUENCE_NUMBER", "TRIM_HORIZON", "LATEST":
		return true
	default:
		return false
	}
}

func neptuneDataListQueriesResponse(queries []neptuneDataQuery) map[string]any {
	running := 0
	items := make([]map[string]any, 0, len(queries))
	for _, query := range queries {
		if strings.EqualFold(query.Status, "RUNNING") {
			running++
		}
		items = append(items, neptuneDataQueryToListItem(query))
	}
	return map[string]any{
		"acceptedQueryCount": len(queries),
		"runningQueryCount":  running,
		"queries":            items,
	}
}

func neptuneDataQueryStatusResponse(query neptuneDataQuery) map[string]any {
	out := neptuneDataQueryToListItem(query)
	out["queryStatus"] = query.Status
	return out
}

func neptuneDataQueryToListItem(query neptuneDataQuery) map[string]any {
	elapsed := query.UpdatedAt.Sub(query.CreatedAt).Milliseconds()
	if elapsed < 0 {
		elapsed = 0
	}
	return map[string]any{
		"queryId":     query.ID,
		"queryString": query.QueryString,
		"queryEvalStats": map[string]any{
			"waited":    0,
			"elapsed":   elapsed,
			"cancelled": query.Cancelled,
			"subqueries": map[string]any{
				"total": 0,
			},
		},
	}
}

func neptuneDataLoaderJobStatusPayload(job neptuneDataLoaderJob) map[string]any {
	elapsed := job.UpdatedAt.Sub(job.CreatedAt).Milliseconds()
	if elapsed < 0 {
		elapsed = 0
	}
	return map[string]any{
		"loadId":                 job.ID,
		"status":                 job.Status,
		"feedCount":              []any{},
		"totalTimeSpent":         elapsed,
		"fullUri":                job.Source,
		"runNumber":              1,
		"retryNumber":            0,
		"totalDuplicates":        0,
		"parsingErrors":          0,
		"datatypeMismatchErrors": 0,
		"insertErrors":           len(job.Errors),
	}
}

func neptuneDataLoaderJobErrorsPayload(job neptuneDataLoaderJob, page, errorsPerPage int) map[string]any {
	if len(job.Errors) == 0 {
		return map[string]any{
			"startIndex":   0,
			"endIndex":     0,
			"totalRecords": 0,
			"errorLogs":    []any{},
		}
	}
	if page < 1 {
		page = 1
	}
	if errorsPerPage < 1 {
		errorsPerPage = 10
	}
	start := (page - 1) * errorsPerPage
	if start > len(job.Errors) {
		start = len(job.Errors)
	}
	end := start + errorsPerPage
	if end > len(job.Errors) {
		end = len(job.Errors)
	}
	logs := make([]map[string]any, 0, end-start)
	for _, item := range job.Errors[start:end] {
		logs = append(logs, item)
	}
	return map[string]any{
		"startIndex":   start,
		"endIndex":     end,
		"totalRecords": len(job.Errors),
		"errorLogs":    logs,
	}
}

func neptuneDataStatisticsPayload(stats neptuneDataStatisticsState) map[string]any {
	return map[string]any{
		"autoCompute":  stats.AutoCompute,
		"active":       stats.Active,
		"statisticsId": stats.ID,
		"date":         stats.UpdatedAt.UTC().Format(time.RFC3339),
		"note":         stats.Note,
		"signatureInfo": map[string]any{
			"signatureCount": 0,
			"instanceCount":  0,
			"predicateCount": 0,
		},
	}
}

func neptuneDataMLResourcePayload(job neptuneDataMLJob) map[string]any {
	return map[string]any{
		"name":             job.ID,
		"arn":              job.ARN,
		"status":           job.Status,
		"outputLocation":   fmt.Sprintf("s3://stackyard-neptunedata/%s", job.Kind),
		"failureReason":    "",
		"cloudwatchLogUrl": fmt.Sprintf("https://console.aws.amazon.com/cloudwatch/home#logsV2:log-groups/log-group/stackyard/%s", job.Kind),
	}
}

func neptuneDataARN(resource, id string) string {
	return fmt.Sprintf("arn:aws:neptune-db:us-east-1:000000000000:%s/%s", resource, strings.TrimSpace(id))
}

type neptuneDataStore struct {
	mu sync.Mutex

	nextID int64

	gremlin    map[string]*neptuneDataQuery
	openCypher map[string]*neptuneDataQuery

	loaderJobs map[string]*neptuneDataLoaderJob

	mlDataProcessing map[string]*neptuneDataMLJob
	mlModelTraining  map[string]*neptuneDataMLJob
	mlModelTransform map[string]*neptuneDataMLJob
	mlEndpoints      map[string]*neptuneDataMLEndpoint

	propertygraphStats neptuneDataStatisticsState
	sparqlStats        neptuneDataStatisticsState

	fastResetToken string
}

type neptuneDataQuery struct {
	ID          string
	QueryString string
	Status      string
	Cancelled   bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type neptuneDataLoaderJob struct {
	ID      string
	Source  string
	Format  string
	Region  string
	RoleArn string
	Status  string
	Errors  []map[string]any

	CreatedAt time.Time
	UpdatedAt time.Time
}

type neptuneDataMLJob struct {
	ID     string
	ARN    string
	Kind   string
	Status string

	CreatedAt time.Time
	UpdatedAt time.Time
}

type neptuneDataMLEndpoint struct {
	ID     string
	ARN    string
	Status string

	ModelName      string
	TrainingJobID  string
	TransformJobID string

	CreatedAt time.Time
	UpdatedAt time.Time
}

type neptuneDataStatisticsState struct {
	ID          string
	AutoCompute bool
	Active      bool
	Note        string
	UpdatedAt   time.Time
}

func newNeptuneDataStore() *neptuneDataStore {
	now := time.Now().UTC()
	return &neptuneDataStore{
		gremlin:          map[string]*neptuneDataQuery{},
		openCypher:       map[string]*neptuneDataQuery{},
		loaderJobs:       map[string]*neptuneDataLoaderJob{},
		mlDataProcessing: map[string]*neptuneDataMLJob{},
		mlModelTraining:  map[string]*neptuneDataMLJob{},
		mlModelTransform: map[string]*neptuneDataMLJob{},
		mlEndpoints:      map[string]*neptuneDataMLEndpoint{},
		propertygraphStats: neptuneDataStatisticsState{
			ID:          "stackyard-pg-statistics-000001",
			AutoCompute: true,
			Active:      true,
			Note:        "Statistics generated by Stackyard.",
			UpdatedAt:   now,
		},
		sparqlStats: neptuneDataStatisticsState{
			ID:          "stackyard-sparql-statistics-000001",
			AutoCompute: true,
			Active:      true,
			Note:        "Statistics generated by Stackyard.",
			UpdatedAt:   now,
		},
	}
}

func (s *neptuneDataStore) ExecuteQuery(language, queryString string) neptuneDataQuery {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextID++
	now := time.Now().UTC()
	id := fmt.Sprintf("stackyard-%s-%06d", strings.ToLower(strings.TrimSpace(language)), s.nextID)
	query := &neptuneDataQuery{
		ID:          id,
		QueryString: strings.TrimSpace(queryString),
		Status:      "SUCCEEDED",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	s.queryMapLocked(language)[id] = query
	return cloneNeptuneDataQuery(query)
}

func (s *neptuneDataStore) ListQueries(language string, _ bool) []neptuneDataQuery {
	s.mu.Lock()
	defer s.mu.Unlock()

	queries := make([]neptuneDataQuery, 0, len(s.queryMapLocked(language)))
	for _, query := range s.queryMapLocked(language) {
		queries = append(queries, cloneNeptuneDataQuery(query))
	}
	sort.Slice(queries, func(i, j int) bool {
		return queries[i].UpdatedAt.After(queries[j].UpdatedAt)
	})
	return queries
}

func (s *neptuneDataStore) GetOrCreateQuery(language, queryID string) neptuneDataQuery {
	s.mu.Lock()
	defer s.mu.Unlock()
	return cloneNeptuneDataQuery(s.getOrCreateQueryLocked(language, queryID))
}

func (s *neptuneDataStore) CancelQuery(language, queryID string) neptuneDataQuery {
	s.mu.Lock()
	defer s.mu.Unlock()

	query := s.getOrCreateQueryLocked(language, queryID)
	query.Status = "CANCELLED"
	query.Cancelled = true
	query.UpdatedAt = time.Now().UTC()
	return cloneNeptuneDataQuery(query)
}

func (s *neptuneDataStore) getOrCreateQueryLocked(language, queryID string) *neptuneDataQuery {
	id := strings.TrimSpace(queryID)
	if id == "" {
		s.nextID++
		id = fmt.Sprintf("stackyard-%s-%06d", strings.ToLower(strings.TrimSpace(language)), s.nextID)
	}
	if existing, ok := s.queryMapLocked(language)[id]; ok {
		return existing
	}
	now := time.Now().UTC()
	query := &neptuneDataQuery{
		ID:          id,
		QueryString: "stage1-placeholder-query",
		Status:      "SUCCEEDED",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	s.queryMapLocked(language)[id] = query
	return query
}

func (s *neptuneDataStore) queryMapLocked(language string) map[string]*neptuneDataQuery {
	if strings.EqualFold(strings.TrimSpace(language), "opencypher") {
		return s.openCypher
	}
	return s.gremlin
}

func (s *neptuneDataStore) StartLoaderJob(source, format, region, roleArn string) neptuneDataLoaderJob {
	s.mu.Lock()
	defer s.mu.Unlock()

	job := &neptuneDataLoaderJob{
		ID:        s.nextIdentifierLocked("loader"),
		Source:    strings.TrimSpace(source),
		Format:    strings.TrimSpace(format),
		Region:    strings.TrimSpace(region),
		RoleArn:   strings.TrimSpace(roleArn),
		Status:    "LOAD_COMPLETED",
		Errors:    nil,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	s.loaderJobs[job.ID] = job
	return cloneNeptuneDataLoaderJob(job)
}

func (s *neptuneDataStore) ListLoaderJobs(limit int) []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	items := make([]neptuneDataLoaderJob, 0, len(s.loaderJobs))
	for _, job := range s.loaderJobs {
		items = append(items, cloneNeptuneDataLoaderJob(job))
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].UpdatedAt.After(items[j].UpdatedAt)
	})
	if limit <= 0 || limit > len(items) {
		limit = len(items)
	}
	ids := make([]string, 0, limit)
	for _, item := range items[:limit] {
		ids = append(ids, item.ID)
	}
	return ids
}

func (s *neptuneDataStore) GetOrCreateLoaderJob(loadID string) neptuneDataLoaderJob {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := strings.TrimSpace(loadID)
	if id == "" {
		id = s.nextIdentifierLocked("loader")
	}
	if existing, ok := s.loaderJobs[id]; ok {
		return cloneNeptuneDataLoaderJob(existing)
	}
	now := time.Now().UTC()
	job := &neptuneDataLoaderJob{
		ID:        id,
		Source:    "s3://stackyard-neptunedata/placeholder",
		Format:    "csv",
		Region:    "us-east-1",
		RoleArn:   "arn:aws:iam::000000000000:role/stackyard-neptunedata",
		Status:    "LOAD_COMPLETED",
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.loaderJobs[id] = job
	return cloneNeptuneDataLoaderJob(job)
}

func (s *neptuneDataStore) CancelLoaderJob(loadID string) neptuneDataLoaderJob {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := strings.TrimSpace(loadID)
	if id == "" {
		id = s.nextIdentifierLocked("loader")
	}
	job, ok := s.loaderJobs[id]
	if !ok {
		now := time.Now().UTC()
		job = &neptuneDataLoaderJob{
			ID:        id,
			Source:    "s3://stackyard-neptunedata/placeholder",
			Format:    "csv",
			Region:    "us-east-1",
			RoleArn:   "arn:aws:iam::000000000000:role/stackyard-neptunedata",
			CreatedAt: now,
			UpdatedAt: now,
		}
		s.loaderJobs[id] = job
	}
	job.Status = "LOAD_CANCELLED_BY_USER"
	job.UpdatedAt = time.Now().UTC()
	return cloneNeptuneDataLoaderJob(job)
}

func (s *neptuneDataStore) StartMLJob(kind, id string) neptuneDataMLJob {
	s.mu.Lock()
	defer s.mu.Unlock()

	jobs := s.mlJobMapLocked(kind)
	trimmedID := strings.TrimSpace(id)
	if trimmedID != "" {
		if existing, ok := jobs[trimmedID]; ok {
			return cloneNeptuneDataMLJob(existing)
		}
	}
	if trimmedID == "" {
		trimmedID = s.nextIdentifierLocked(kind)
	}
	now := time.Now().UTC()
	job := &neptuneDataMLJob{
		ID:        trimmedID,
		ARN:       neptuneDataARN("ml-job", trimmedID),
		Kind:      kind,
		Status:    "COMPLETED",
		CreatedAt: now,
		UpdatedAt: now,
	}
	jobs[trimmedID] = job
	return cloneNeptuneDataMLJob(job)
}

func (s *neptuneDataStore) ListMLJobs(kind string, maxItems int) []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	ids := make([]string, 0, len(s.mlJobMapLocked(kind)))
	for id := range s.mlJobMapLocked(kind) {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	if maxItems <= 0 || maxItems > len(ids) {
		maxItems = len(ids)
	}
	return append([]string(nil), ids[:maxItems]...)
}

func (s *neptuneDataStore) GetOrCreateMLJob(kind, id string) neptuneDataMLJob {
	s.mu.Lock()
	defer s.mu.Unlock()

	jobs := s.mlJobMapLocked(kind)
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		trimmedID = s.nextIdentifierLocked(kind)
	}
	if existing, ok := jobs[trimmedID]; ok {
		return cloneNeptuneDataMLJob(existing)
	}
	now := time.Now().UTC()
	job := &neptuneDataMLJob{
		ID:        trimmedID,
		ARN:       neptuneDataARN("ml-job", trimmedID),
		Kind:      kind,
		Status:    "COMPLETED",
		CreatedAt: now,
		UpdatedAt: now,
	}
	jobs[trimmedID] = job
	return cloneNeptuneDataMLJob(job)
}

func (s *neptuneDataStore) CancelMLJob(kind, id string) neptuneDataMLJob {
	s.mu.Lock()
	defer s.mu.Unlock()

	jobs := s.mlJobMapLocked(kind)
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		trimmedID = s.nextIdentifierLocked(kind)
	}
	job, ok := jobs[trimmedID]
	if !ok {
		now := time.Now().UTC()
		job = &neptuneDataMLJob{
			ID:        trimmedID,
			ARN:       neptuneDataARN("ml-job", trimmedID),
			Kind:      kind,
			Status:    "COMPLETED",
			CreatedAt: now,
			UpdatedAt: now,
		}
		jobs[trimmedID] = job
	}
	job.Status = "STOPPED"
	job.UpdatedAt = time.Now().UTC()
	return cloneNeptuneDataMLJob(job)
}

func (s *neptuneDataStore) mlJobMapLocked(kind string) map[string]*neptuneDataMLJob {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "dataprocessing":
		return s.mlDataProcessing
	case "modeltraining":
		return s.mlModelTraining
	case "modeltransform":
		return s.mlModelTransform
	default:
		return s.mlDataProcessing
	}
}

func (s *neptuneDataStore) CreateMLEndpoint(id, modelName, trainingJobID, transformJobID string) neptuneDataMLEndpoint {
	s.mu.Lock()
	defer s.mu.Unlock()

	trimmedID := strings.TrimSpace(id)
	if trimmedID != "" {
		if existing, ok := s.mlEndpoints[trimmedID]; ok {
			return cloneNeptuneDataMLEndpoint(existing)
		}
	}
	if trimmedID == "" {
		trimmedID = s.nextIdentifierLocked("ml-endpoint")
	}
	now := time.Now().UTC()
	endpoint := &neptuneDataMLEndpoint{
		ID:             trimmedID,
		ARN:            neptuneDataARN("ml-endpoint", trimmedID),
		Status:         "InService",
		ModelName:      strings.TrimSpace(modelName),
		TrainingJobID:  strings.TrimSpace(trainingJobID),
		TransformJobID: strings.TrimSpace(transformJobID),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	s.mlEndpoints[trimmedID] = endpoint
	return cloneNeptuneDataMLEndpoint(endpoint)
}

func (s *neptuneDataStore) ListMLEndpoints(maxItems int) []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	ids := make([]string, 0, len(s.mlEndpoints))
	for id := range s.mlEndpoints {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	if maxItems <= 0 || maxItems > len(ids) {
		maxItems = len(ids)
	}
	return append([]string(nil), ids[:maxItems]...)
}

func (s *neptuneDataStore) GetOrCreateMLEndpoint(id string) neptuneDataMLEndpoint {
	s.mu.Lock()
	defer s.mu.Unlock()

	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		trimmedID = s.nextIdentifierLocked("ml-endpoint")
	}
	if existing, ok := s.mlEndpoints[trimmedID]; ok {
		return cloneNeptuneDataMLEndpoint(existing)
	}
	now := time.Now().UTC()
	endpoint := &neptuneDataMLEndpoint{
		ID:        trimmedID,
		ARN:       neptuneDataARN("ml-endpoint", trimmedID),
		Status:    "InService",
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.mlEndpoints[trimmedID] = endpoint
	return cloneNeptuneDataMLEndpoint(endpoint)
}

func (s *neptuneDataStore) DeleteMLEndpoint(id string) neptuneDataMLEndpoint {
	s.mu.Lock()
	defer s.mu.Unlock()

	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		trimmedID = s.nextIdentifierLocked("ml-endpoint")
	}
	endpoint, ok := s.mlEndpoints[trimmedID]
	if !ok {
		now := time.Now().UTC()
		endpoint = &neptuneDataMLEndpoint{
			ID:        trimmedID,
			ARN:       neptuneDataARN("ml-endpoint", trimmedID),
			Status:    "InService",
			CreatedAt: now,
			UpdatedAt: now,
		}
		s.mlEndpoints[trimmedID] = endpoint
	}
	endpoint.Status = "Deleting"
	endpoint.UpdatedAt = time.Now().UTC()
	return cloneNeptuneDataMLEndpoint(endpoint)
}

func (s *neptuneDataStore) GetStatistics(kind string) neptuneDataStatisticsState {
	s.mu.Lock()
	defer s.mu.Unlock()

	if strings.EqualFold(strings.TrimSpace(kind), "sparql") {
		return cloneNeptuneDataStatisticsState(s.sparqlStats)
	}
	return cloneNeptuneDataStatisticsState(s.propertygraphStats)
}

func (s *neptuneDataStore) ManageStatistics(kind, mode string) (neptuneDataStatisticsState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	trimmedMode := strings.TrimSpace(mode)
	if trimmedMode == "" {
		trimmedMode = "refresh"
	}
	switch trimmedMode {
	case "disableAutoCompute", "enableAutoCompute", "refresh":
	default:
		return neptuneDataStatisticsState{}, fmt.Errorf("mode is invalid")
	}

	stats := s.statisticsPointerLocked(kind)
	switch trimmedMode {
	case "disableAutoCompute":
		stats.AutoCompute = false
		stats.Note = "Auto compute disabled by ManageStatistics."
	case "enableAutoCompute":
		stats.AutoCompute = true
		stats.Note = "Auto compute enabled by ManageStatistics."
	case "refresh":
		stats.ID = s.nextIdentifierLocked(statsPrefixForKind(kind))
		stats.Active = true
		stats.Note = "Statistics refresh requested."
	}
	stats.UpdatedAt = time.Now().UTC()
	return cloneNeptuneDataStatisticsState(*stats), nil
}

func (s *neptuneDataStore) DeleteStatistics(kind string) neptuneDataStatisticsState {
	s.mu.Lock()
	defer s.mu.Unlock()

	stats := s.statisticsPointerLocked(kind)
	stats.Active = false
	stats.UpdatedAt = time.Now().UTC()
	stats.Note = "Statistics deleted."
	stats.ID = s.nextIdentifierLocked(statsPrefixForKind(kind))
	return cloneNeptuneDataStatisticsState(*stats)
}

func (s *neptuneDataStore) statisticsPointerLocked(kind string) *neptuneDataStatisticsState {
	if strings.EqualFold(strings.TrimSpace(kind), "sparql") {
		return &s.sparqlStats
	}
	return &s.propertygraphStats
}

func statsPrefixForKind(kind string) string {
	if strings.EqualFold(strings.TrimSpace(kind), "sparql") {
		return "sparql-statistics"
	}
	return "pg-statistics"
}

func (s *neptuneDataStore) ExecuteFastReset(action, token string) (string, string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	trimmedAction := strings.TrimSpace(action)
	trimmedToken := strings.TrimSpace(token)
	switch trimmedAction {
	case "performDatabaseReset":
		if trimmedToken == "" {
			trimmedToken = s.fastResetToken
		}
		if trimmedToken == "" {
			trimmedToken = s.nextIdentifierLocked("fast-reset-token")
		}
		s.fastResetToken = ""
		return "200 OK", ""
	case "initiateDatabaseReset":
		if trimmedToken == "" {
			trimmedToken = s.nextIdentifierLocked("fast-reset-token")
		}
		s.fastResetToken = trimmedToken
		return "200 OK", trimmedToken
	default:
		if trimmedToken == "" {
			trimmedToken = s.nextIdentifierLocked("fast-reset-token")
		}
		s.fastResetToken = trimmedToken
		return "200 OK", trimmedToken
	}
}

func (s *neptuneDataStore) nextIdentifierLocked(prefix string) string {
	s.nextID++
	return fmt.Sprintf("stackyard-%s-%06d", strings.Trim(strings.ToLower(strings.TrimSpace(prefix)), "-"), s.nextID)
}

func cloneNeptuneDataQuery(query *neptuneDataQuery) neptuneDataQuery {
	if query == nil {
		return neptuneDataQuery{}
	}
	return neptuneDataQuery{
		ID:          query.ID,
		QueryString: query.QueryString,
		Status:      query.Status,
		Cancelled:   query.Cancelled,
		CreatedAt:   query.CreatedAt,
		UpdatedAt:   query.UpdatedAt,
	}
}

func cloneNeptuneDataLoaderJob(job *neptuneDataLoaderJob) neptuneDataLoaderJob {
	if job == nil {
		return neptuneDataLoaderJob{}
	}
	out := neptuneDataLoaderJob{
		ID:        job.ID,
		Source:    job.Source,
		Format:    job.Format,
		Region:    job.Region,
		RoleArn:   job.RoleArn,
		Status:    job.Status,
		CreatedAt: job.CreatedAt,
		UpdatedAt: job.UpdatedAt,
	}
	if len(job.Errors) > 0 {
		out.Errors = make([]map[string]any, 0, len(job.Errors))
		for _, item := range job.Errors {
			cloned := make(map[string]any, len(item))
			for k, v := range item {
				cloned[k] = v
			}
			out.Errors = append(out.Errors, cloned)
		}
	}
	return out
}

func cloneNeptuneDataMLJob(job *neptuneDataMLJob) neptuneDataMLJob {
	if job == nil {
		return neptuneDataMLJob{}
	}
	return neptuneDataMLJob{
		ID:        job.ID,
		ARN:       job.ARN,
		Kind:      job.Kind,
		Status:    job.Status,
		CreatedAt: job.CreatedAt,
		UpdatedAt: job.UpdatedAt,
	}
}

func cloneNeptuneDataMLEndpoint(endpoint *neptuneDataMLEndpoint) neptuneDataMLEndpoint {
	if endpoint == nil {
		return neptuneDataMLEndpoint{}
	}
	return neptuneDataMLEndpoint{
		ID:             endpoint.ID,
		ARN:            endpoint.ARN,
		Status:         endpoint.Status,
		ModelName:      endpoint.ModelName,
		TrainingJobID:  endpoint.TrainingJobID,
		TransformJobID: endpoint.TransformJobID,
		CreatedAt:      endpoint.CreatedAt,
		UpdatedAt:      endpoint.UpdatedAt,
	}
}

func cloneNeptuneDataStatisticsState(stats neptuneDataStatisticsState) neptuneDataStatisticsState {
	return neptuneDataStatisticsState{
		ID:          stats.ID,
		AutoCompute: stats.AutoCompute,
		Active:      stats.Active,
		Note:        stats.Note,
		UpdatedAt:   stats.UpdatedAt,
	}
}
