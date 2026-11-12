package server

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

const azureApplicationInsightsPrefix = "/azure/application-insights/v1/apps/"

var azureApplicationInsightsSupportedEventTypes = map[string]string{
	"traces":              "traces",
	"customevents":        "customEvents",
	"pageviews":           "pageViews",
	"requests":            "requests",
	"dependencies":        "dependencies",
	"exceptions":          "exceptions",
	"availabilityresults": "availabilityResults",
	"$all":                "$all",
}

var azureApplicationInsightsSupportedMetrics = map[string]map[string]any{
	"requests/count": {
		"aggregations": []string{"count", "sum", "avg", "min", "max"},
		"supportedGroupBy": []string{
			"request/name",
			"cloud/roleName",
		},
		"base": 42.0,
	},
	"requests/duration": {
		"aggregations": []string{"avg", "min", "max"},
		"supportedGroupBy": []string{
			"request/name",
		},
		"base": 123.4,
	},
	"requests/failed": {
		"aggregations": []string{"count", "sum"},
		"supportedGroupBy": []string{
			"resultCode",
		},
		"base": 3.0,
	},
	"pageviews/count": {
		"aggregations": []string{"count", "sum"},
		"supportedGroupBy": []string{
			"page/name",
		},
		"base": 17.0,
	},
	"dependencies/count": {
		"aggregations": []string{"count", "sum"},
		"supportedGroupBy": []string{
			"dependency/type",
		},
		"base": 9.0,
	},
	"exceptions/count": {
		"aggregations": []string{"count", "sum"},
		"supportedGroupBy": []string{
			"exception/type",
		},
		"base": 2.0,
	},
	"availabilityresults/availabilitypercentage": {
		"aggregations": []string{"avg", "min", "max"},
		"supportedGroupBy": []string{
			"availability/location",
		},
		"base": 99.5,
	},
}

func (s *Server) handleAzureApplicationInsightsRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, azureApplicationInsightsPrefix) {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}

	relative := strings.TrimPrefix(path, azureApplicationInsightsPrefix)
	segments := splitPathSegments(strings.Trim(strings.TrimSpace(relative), "/"))
	if len(segments) < 2 {
		return false
	}

	appID := strings.TrimSpace(segments[0])
	if appID == "" {
		azureApplicationInsightsError(w, http.StatusBadRequest, path, "BadArgumentError", "appId is required")
		return true
	}

	switch strings.ToLower(strings.TrimSpace(segments[1])) {
	case "query":
		s.handleAzureApplicationInsightsQuery(w, r, path, appID, segments[2:])
		return true
	case "events":
		s.handleAzureApplicationInsightsEvents(w, r, path, appID, segments[2:])
		return true
	case "metrics":
		s.handleAzureApplicationInsightsMetrics(w, r, path, appID, segments[2:])
		return true
	case "metadata":
		s.handleAzureApplicationInsightsMetadata(w, r, path, appID, segments[2:])
		return true
	default:
		return false
	}
}

func (s *Server) handleAzureApplicationInsightsQuery(w http.ResponseWriter, r *http.Request, path, appID string, tail []string) {
	if len(tail) != 0 {
		azureApplicationInsightsError(w, http.StatusNotFound, path, "PathNotFoundError", "query route not found")
		return
	}
	switch r.Method {
	case http.MethodGet:
		query := strings.TrimSpace(r.URL.Query().Get("query"))
		if query == "" {
			azureApplicationInsightsError(w, http.StatusBadRequest, path, "BadArgumentError", "query is required")
			return
		}
		respondJSON(w, http.StatusOK, azureApplicationInsightsQueryResultsFixture(appID, query, strings.TrimSpace(r.URL.Query().Get("timespan"))))
	case http.MethodPost:
		body, err := readBodyBytes(r)
		if err != nil {
			azureApplicationInsightsError(w, http.StatusBadRequest, path, "BadArgumentError", "unable to read request body")
			return
		}
		var payload map[string]any
		if len(strings.TrimSpace(string(body))) > 0 {
			if err := json.Unmarshal(body, &payload); err != nil {
				azureApplicationInsightsError(w, http.StatusBadRequest, path, "BadArgumentError", "request body must be valid JSON")
				return
			}
		}
		query := strings.TrimSpace(azureApplicationInsightsString(payload["query"]))
		if query == "" {
			azureApplicationInsightsError(w, http.StatusBadRequest, path, "BadArgumentError", "query is required")
			return
		}
		respondJSON(w, http.StatusOK, azureApplicationInsightsQueryResultsFixture(appID, query, strings.TrimSpace(azureApplicationInsightsString(payload["timespan"]))))
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAzureApplicationInsightsEvents(w http.ResponseWriter, r *http.Request, path, appID string, tail []string) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if len(tail) == 0 {
		azureApplicationInsightsError(w, http.StatusBadRequest, path, "BadArgumentError", "eventType is required")
		return
	}

	if len(tail) == 1 && tail[0] == "$metadata" {
		w.Header().Set("Content-Type", "application/xml;charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(azureApplicationInsightsEventsMetadataXML()))
		return
	}

	eventType := azureApplicationInsightsCanonicalEventType(tail[0])
	if eventType == "" {
		azureApplicationInsightsError(w, http.StatusBadRequest, path, "BadArgumentError", "unsupported eventType")
		return
	}

	if len(tail) == 1 {
		response := map[string]any{
			"@odata.context": fmt.Sprintf("https://api.applicationinsights.io/v1/apps/%s/events/$metadata#%s", appID, eventType),
			"value":          azureApplicationInsightsEventListFixture(appID, eventType),
		}
		if strings.EqualFold(r.URL.Query().Get("$count"), "true") {
			response["@odata.count"] = len(response["value"].([]map[string]any))
		}
		respondJSON(w, http.StatusOK, response)
		return
	}

	eventID := strings.TrimSpace(tail[1])
	if eventID == "" {
		azureApplicationInsightsError(w, http.StatusBadRequest, path, "BadArgumentError", "eventId is required")
		return
	}
	respondJSON(w, http.StatusOK, azureApplicationInsightsSingleEventFixture(appID, eventType, eventID))
}

func (s *Server) handleAzureApplicationInsightsMetrics(w http.ResponseWriter, r *http.Request, path, appID string, tail []string) {
	switch r.Method {
	case http.MethodGet:
		switch {
		case len(tail) == 1 && strings.EqualFold(tail[0], "metadata"):
			respondJSON(w, http.StatusOK, azureApplicationInsightsMetricMetadataFixture(appID))
			return
		case len(tail) >= 1:
			metricID := strings.ToLower(strings.Trim(strings.Join(tail, "/"), "/"))
			if metricID == "" {
				azureApplicationInsightsError(w, http.StatusBadRequest, path, "BadArgumentError", "metricId is required")
				return
			}
			if _, ok := azureApplicationInsightsSupportedMetrics[metricID]; !ok {
				azureApplicationInsightsError(w, http.StatusNotFound, path, "PathNotFoundError", "metric not found")
				return
			}
			respondJSON(w, http.StatusOK, azureApplicationInsightsMetricResultFixture(appID, metricID, r.URL.Query()))
			return
		default:
			azureApplicationInsightsError(w, http.StatusBadRequest, path, "BadArgumentError", "metricId is required")
			return
		}
	case http.MethodPost:
		if len(tail) != 0 {
			azureApplicationInsightsError(w, http.StatusNotFound, path, "PathNotFoundError", "metrics route not found")
			return
		}
		body, err := readBodyBytes(r)
		if err != nil {
			azureApplicationInsightsError(w, http.StatusBadRequest, path, "BadArgumentError", "unable to read request body")
			return
		}
		requests, ok := azureApplicationInsightsParseMetricBatch(body)
		if !ok || len(requests) == 0 {
			azureApplicationInsightsError(w, http.StatusBadRequest, path, "BadArgumentError", "request body must contain at least one metric request")
			return
		}
		results := make([]map[string]any, 0, len(requests))
		for _, metricID := range requests {
			normalized := strings.ToLower(strings.TrimSpace(metricID))
			if _, exists := azureApplicationInsightsSupportedMetrics[normalized]; !exists {
				azureApplicationInsightsError(w, http.StatusNotFound, path, "PathNotFoundError", "metric not found")
				return
			}
			results = append(results, azureApplicationInsightsMetricResultFixture(appID, normalized, r.URL.Query()))
		}
		respondJSON(w, http.StatusOK, map[string]any{"value": results})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAzureApplicationInsightsMetadata(w http.ResponseWriter, r *http.Request, path, appID string, tail []string) {
	if len(tail) != 0 {
		azureApplicationInsightsError(w, http.StatusNotFound, path, "PathNotFoundError", "metadata route not found")
		return
	}
	switch r.Method {
	case http.MethodGet, http.MethodPost:
		if r.Method == http.MethodPost {
			body, err := readBodyBytes(r)
			if err != nil {
				azureApplicationInsightsError(w, http.StatusBadRequest, path, "BadArgumentError", "unable to read request body")
				return
			}
			if trimmed := strings.TrimSpace(string(body)); trimmed != "" {
				var payload map[string]any
				if err := json.Unmarshal(body, &payload); err != nil {
					azureApplicationInsightsError(w, http.StatusBadRequest, path, "BadArgumentError", "request body must be valid JSON")
					return
				}
			}
		}
		respondJSON(w, http.StatusOK, azureApplicationInsightsMetadataFixture(appID))
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func azureApplicationInsightsError(w http.ResponseWriter, status int, path, code, message string) {
	if strings.TrimSpace(code) == "" {
		code = "BadArgumentError"
	}
	if strings.TrimSpace(message) == "" {
		message = "invalid request"
	}
	respondJSON(w, status, map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
			"details": []map[string]any{
				{
					"code":    code,
					"message": message,
				},
			},
		},
		"provider": providerAzure,
		"path":     path,
	})
}

func azureApplicationInsightsQueryResultsFixture(appID, query, timespan string) map[string]any {
	return map[string]any{
		"tables": []map[string]any{
			{
				"name": "PrimaryResult",
				"columns": []map[string]any{
					{"name": "timestamp", "type": "datetime"},
					{"name": "appId", "type": "string"},
					{"name": "query", "type": "string"},
					{"name": "timespan", "type": "string"},
					{"name": "itemCount", "type": "real"},
				},
				"rows": [][]any{
					{
						"2026-01-02T03:04:05Z",
						appID,
						query,
						azureApplicationInsightsDefaultString(timespan, "PT1H"),
						42.0,
					},
				},
			},
		},
	}
}

func azureApplicationInsightsEventListFixture(appID, eventType string) []map[string]any {
	eventID := azureApplicationInsightsDeterministicID(appID + ":" + eventType)
	return []map[string]any{
		azureApplicationInsightsSingleEventFixture(appID, eventType, eventID),
	}
}

func azureApplicationInsightsSingleEventFixture(appID, eventType, eventID string) map[string]any {
	return map[string]any{
		"id":                     eventID,
		"eventType":              eventType,
		"timestamp":              "2026-01-02T03:04:05Z",
		"appId":                  appID,
		"itemType":               eventType,
		"operation_Name":         "GET /health",
		"operation_Id":           azureApplicationInsightsDeterministicID("operation:" + appID + ":" + eventType),
		"session_Id":             "stackyard-session",
		"user_Id":                "stackyard-user",
		"client_City":            "New York",
		"client_CountryOrRegion": "United States",
		"application_Version":    "1.0.0",
		"success":                true,
		"properties": map[string]any{
			"source": "stackyard",
		},
	}
}

func azureApplicationInsightsEventsMetadataXML() string {
	return `<?xml version="1.0" encoding="utf-8"?>
<edmx:Edmx Version="4.0" xmlns:edmx="http://docs.oasis-open.org/odata/ns/edmx">
  <edmx:DataServices>
    <Schema Namespace="ApplicationInsights" xmlns="http://docs.oasis-open.org/odata/ns/edm">
      <EntityType Name="event">
        <Key>
          <PropertyRef Name="id" />
        </Key>
        <Property Name="id" Type="Edm.String" Nullable="false" />
        <Property Name="eventType" Type="Edm.String" />
        <Property Name="timestamp" Type="Edm.DateTimeOffset" />
      </EntityType>
    </Schema>
  </edmx:DataServices>
</edmx:Edmx>`
}

func azureApplicationInsightsMetricMetadataFixture(appID string) map[string]any {
	metrics := make([]map[string]any, 0, len(azureApplicationInsightsSupportedMetrics))
	for metricID, meta := range azureApplicationInsightsSupportedMetrics {
		metrics = append(metrics, map[string]any{
			"id":                        metricID,
			"supportedAggregationTypes": meta["aggregations"],
			"supportedGroupBy":          meta["supportedGroupBy"],
			"defaultAggregation":        azureApplicationInsightsDefaultAggregation(meta["aggregations"]),
			"applicationId":             appID,
		})
	}
	return map[string]any{"value": metrics}
}

func azureApplicationInsightsMetricResultFixture(appID, metricID string, values url.Values) map[string]any {
	meta := azureApplicationInsightsSupportedMetrics[metricID]
	base := meta["base"].(float64)
	aggregation := azureApplicationInsightsDefaultString(strings.TrimSpace(values.Get("aggregation")), azureApplicationInsightsDefaultAggregation(meta["aggregations"]))
	segment := azureApplicationInsightsDefaultString(strings.TrimSpace(values.Get("segment")), "overall")
	return map[string]any{
		"id":          metricID,
		"application": appID,
		"timespan":    azureApplicationInsightsDefaultString(strings.TrimSpace(values.Get("timespan")), "PT1H"),
		"interval":    azureApplicationInsightsDefaultString(strings.TrimSpace(values.Get("interval")), "PT1H"),
		"value": []map[string]any{
			{
				"segment": segment,
				"timeseries": []map[string]any{
					{
						"metadata": []map[string]any{
							{
								"name":  "aggregation",
								"value": aggregation,
							},
						},
						"data": []map[string]any{
							{
								"timeStamp": "2026-01-02T03:00:00Z",
								aggregation: base,
							},
						},
					},
				},
			},
		},
	}
}

func azureApplicationInsightsMetadataFixture(appID string) map[string]any {
	return map[string]any{
		"tables": []map[string]any{
			{
				"name":        "requests",
				"displayName": "Requests",
				"columns": []map[string]any{
					{"name": "timestamp", "type": "datetime"},
					{"name": "name", "type": "string"},
					{"name": "duration", "type": "real"},
					{"name": "success", "type": "bool"},
				},
			},
			{
				"name":        "traces",
				"displayName": "Traces",
				"columns": []map[string]any{
					{"name": "timestamp", "type": "datetime"},
					{"name": "message", "type": "string"},
					{"name": "severityLevel", "type": "int"},
				},
			},
		},
		"tableGroups": []map[string]any{
			{
				"name":    "application",
				"display": "Application telemetry",
				"tables":  []string{"requests", "traces"},
			},
		},
		"applications": []map[string]any{
			{
				"id":          appID,
				"displayName": "stackyard-application-insights",
			},
		},
	}
}

func azureApplicationInsightsParseMetricBatch(body []byte) ([]string, bool) {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return nil, false
	}

	var arrayPayload []map[string]any
	if err := json.Unmarshal(body, &arrayPayload); err == nil {
		out := make([]string, 0, len(arrayPayload))
		for _, item := range arrayPayload {
			id := strings.ToLower(strings.TrimSpace(azureApplicationInsightsString(item["id"])))
			if id != "" {
				out = append(out, id)
			}
		}
		return out, true
	}

	var objectPayload map[string]any
	if err := json.Unmarshal(body, &objectPayload); err != nil {
		return nil, false
	}
	if raw, ok := objectPayload["metrics"].([]any); ok {
		out := make([]string, 0, len(raw))
		for _, item := range raw {
			asMap, ok := item.(map[string]any)
			if !ok {
				continue
			}
			id := strings.ToLower(strings.TrimSpace(azureApplicationInsightsString(asMap["id"])))
			if id != "" {
				out = append(out, id)
			}
		}
		return out, true
	}
	return nil, false
}

func azureApplicationInsightsCanonicalEventType(raw string) string {
	return azureApplicationInsightsSupportedEventTypes[strings.ToLower(strings.TrimSpace(raw))]
}

func azureApplicationInsightsDefaultAggregation(raw any) string {
	values, _ := raw.([]string)
	if len(values) == 0 {
		return "count"
	}
	return values[0]
}

func azureApplicationInsightsString(v any) string {
	switch value := v.(type) {
	case string:
		return value
	default:
		return ""
	}
}

func azureApplicationInsightsDefaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func azureApplicationInsightsDeterministicID(seed string) string {
	sum := sha1.Sum([]byte(seed))
	return hex.EncodeToString(sum[:8])
}
