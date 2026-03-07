package server

import (
	"encoding/json"
	"net/http"
)

func (s *Server) handleGCPMapsRoutingRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPMapsRoutingPath(path) {
		return false
	}

	if r.Method != http.MethodPost {
		return false
	}

	var reqBody map[string]any
	if r.Body != nil {
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
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

	switch path {
	case "/gcp/directions/v2:computeRoutes", "/gcp/google.maps.routing.v2.Routes/ComputeRoutes":
		if _, ok := reqBody["origin"].(map[string]any); !ok {
			respondJSON(w, http.StatusBadRequest, map[string]any{
				"error":    "InvalidArgument",
				"message":  "origin must be provided",
				"provider": providerGCP,
				"path":     path,
			})
			return true
		}
		if _, ok := reqBody["destination"].(map[string]any); !ok {
			respondJSON(w, http.StatusBadRequest, map[string]any{
				"error":    "InvalidArgument",
				"message":  "destination must be provided",
				"provider": providerGCP,
				"path":     path,
			})
			return true
		}
		respondJSON(w, http.StatusOK, map[string]any{
			"routes": []any{
				map[string]any{
					"distanceMeters": 7421,
					"duration":       "812s",
					"legs": []any{
						map[string]any{
							"distanceMeters": 7421,
							"duration":       "812s",
						},
					},
				},
			},
		})
		return true
	default:
		if !hasNonEmptyArrayField(reqBody, "origins") {
			respondJSON(w, http.StatusBadRequest, map[string]any{
				"error":    "InvalidArgument",
				"message":  "origins must be a non-empty array",
				"provider": providerGCP,
				"path":     path,
			})
			return true
		}
		if !hasNonEmptyArrayField(reqBody, "destinations") {
			respondJSON(w, http.StatusBadRequest, map[string]any{
				"error":    "InvalidArgument",
				"message":  "destinations must be a non-empty array",
				"provider": providerGCP,
				"path":     path,
			})
			return true
		}
		respondJSON(w, http.StatusOK, []any{
			map[string]any{
				"originIndex":      0,
				"destinationIndex": 0,
				"status": map[string]any{
					"code": 0,
				},
				"distanceMeters": 7421,
				"duration":       "812s",
				"condition":      "ROUTE_EXISTS",
			},
		})
		return true
	}
}

func isGCPMapsRoutingPath(path string) bool {
	if path == "/gcp/directions/v2:computeRoutes" {
		return true
	}
	if path == "/gcp/distanceMatrix/v2:computeRouteMatrix" {
		return true
	}
	return path == "/gcp/google.maps.routing.v2.Routes/ComputeRoutes" ||
		path == "/gcp/google.maps.routing.v2.Routes/ComputeRouteMatrix"
}
