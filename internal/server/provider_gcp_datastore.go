package server

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

func (s *Server) handleGCPDatastoreRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	normalizedPath := normalizeGCPDatastorePath(path)
	if !strings.HasPrefix(normalizedPath, "/gcp/v1/projects/") {
		return false
	}
	if !isGCPDatastorePath(normalizedPath) {
		return false
	}

	if r.Method != http.MethodPost {
		return false
	}

	project, database, action, ok := parseGCPDatastoreActionPath(normalizedPath)
	if !ok {
		return false
	}

	var reqBody map[string]any
	if r.Body != nil {
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&reqBody); err != nil && err.Error() != "EOF" {
			respondJSON(w, http.StatusBadRequest, map[string]any{
				"error":    "InvalidArgument",
				"message":  "request body must be valid JSON",
				"provider": providerGCP,
				"path":     path,
			})
			return true
		}
	}

	switch action {
	case "lookup":
		respondJSON(w, http.StatusOK, gcpDatastoreLookupResponse(project, database))
		return true
	case "runQuery":
		respondJSON(w, http.StatusOK, gcpDatastoreRunQueryResponse(project, database))
		return true
	case "runAggregationQuery":
		respondJSON(w, http.StatusOK, gcpDatastoreRunAggregationQueryResponse())
		return true
	case "beginTransaction":
		respondJSON(w, http.StatusOK, map[string]any{
			"transaction": base64.StdEncoding.EncodeToString([]byte("tx-1")),
		})
		return true
	case "commit":
		respondJSON(w, http.StatusOK, map[string]any{
			"mutationResults": []any{},
			"indexUpdates":    "0",
			"commitTime":      time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339Nano),
		})
		return true
	case "rollback":
		respondJSON(w, http.StatusOK, map[string]any{
			"rolledBack": true,
		})
		return true
	case "allocateIds":
		respondJSON(w, http.StatusOK, map[string]any{
			"keys": []any{gcpDatastoreKey(project, database, "order-allocated-1")},
		})
		return true
	case "reserveIds":
		respondJSON(w, http.StatusOK, map[string]any{
			"reserved": true,
		})
		return true
	default:
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	}
}

func isGCPDatastorePath(path string) bool {
	return strings.Contains(path, ":lookup") ||
		strings.Contains(path, ":runQuery") ||
		strings.Contains(path, ":runAggregationQuery") ||
		strings.Contains(path, ":beginTransaction") ||
		strings.Contains(path, ":commit") ||
		strings.Contains(path, ":rollback") ||
		strings.Contains(path, ":allocateIds") ||
		strings.Contains(path, ":reserveIds")
}

func normalizeGCPDatastorePath(path string) string {
	normalized := strings.ReplaceAll(path, "%3A", ":")
	normalized = strings.ReplaceAll(normalized, "%3a", ":")
	return normalized
}

func parseGCPDatastoreActionPath(path string) (project, database, action string, ok bool) {
	const prefix = "/gcp/v1/projects/"
	if !strings.HasPrefix(path, prefix) {
		return "", "", "", false
	}
	remainder := strings.TrimPrefix(path, prefix)
	resourcePath, action, found := strings.Cut(remainder, ":")
	if !found {
		return "", "", "", false
	}
	action = strings.TrimSpace(action)
	switch action {
	case "lookup", "runQuery", "runAggregationQuery", "beginTransaction", "commit", "rollback", "allocateIds", "reserveIds":
	default:
		return "", "", "", false
	}

	parts := strings.Split(strings.Trim(resourcePath, "/"), "/")
	switch len(parts) {
	case 1:
		project = strings.TrimSpace(parts[0])
		database = "(default)"
	case 3:
		project = strings.TrimSpace(parts[0])
		if parts[1] != "databases" {
			return "", "", "", false
		}
		database = strings.TrimSpace(parts[2])
	default:
		return "", "", "", false
	}

	if project == "" || database == "" {
		return "", "", "", false
	}
	return project, database, action, true
}

func gcpDatastoreLookupResponse(project, database string) map[string]any {
	return map[string]any{
		"found": []any{
			map[string]any{
				"entity": map[string]any{
					"key": gcpDatastoreKey(project, database, "order-1"),
					"properties": map[string]any{
						"amount": map[string]any{
							"integerValue": "42",
						},
						"state": map[string]any{
							"stringValue": "CREATED",
						},
					},
				},
				"version": "1",
			},
		},
		"missing":  []any{},
		"deferred": []any{},
	}
}

func gcpDatastoreRunQueryResponse(project, database string) map[string]any {
	return map[string]any{
		"batch": map[string]any{
			"entityResultType": "FULL",
			"entityResults": []any{
				map[string]any{
					"entity": map[string]any{
						"key": gcpDatastoreKey(project, database, "order-1"),
						"properties": map[string]any{
							"amount": map[string]any{
								"integerValue": "42",
							},
							"state": map[string]any{
								"stringValue": "CREATED",
							},
						},
					},
				},
			},
			"moreResults": "NO_MORE_RESULTS",
		},
	}
}

func gcpDatastoreRunAggregationQueryResponse() map[string]any {
	return map[string]any{
		"batch": map[string]any{
			"aggregationResults": []any{
				map[string]any{
					"aggregateProperties": map[string]any{
						"total_orders": map[string]any{
							"integerValue": "1",
						},
					},
				},
			},
			"moreResults": "NO_MORE_RESULTS",
		},
	}
}

func gcpDatastoreKey(project, database, name string) map[string]any {
	return map[string]any{
		"partitionId": map[string]any{
			"projectId":  project,
			"databaseId": database,
		},
		"path": []any{
			map[string]any{
				"kind": "Order",
				"name": name,
			},
		},
	}
}
