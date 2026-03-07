package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func (s *Server) handleGCPMapsFleetEngineRouter(w http.ResponseWriter, r *http.Request) bool {
	path := strings.TrimSpace(r.URL.Path)
	if path == "" {
		path = rawRequestPath(r)
	}
	if !isGCPMapsFleetEnginePath(path) {
		return false
	}

	if strings.HasPrefix(path, "/gcp/maps.fleetengine.v1.TripService/") ||
		strings.HasPrefix(path, "/gcp/maps.fleetengine.v1.VehicleService/") {
		switch r.Method {
		case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
			respondProviderNotImplemented(w, providerGCP, path)
			return true
		default:
			return false
		}
	}

	if !strings.HasPrefix(path, "/gcp/v1/providers/") {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPMapsFleetEngineListTrips(w, r, path) {
			return true
		}
		if handleGCPMapsFleetEngineGetTrip(w, path) {
			return true
		}
		if handleGCPMapsFleetEngineListVehicles(w, r, path) {
			return true
		}
		if handleGCPMapsFleetEngineGetVehicle(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPMapsFleetEngineCreateTrip(w, r, path) {
			return true
		}
		if handleGCPMapsFleetEngineSearchTrips(w, r, path) {
			return true
		}
		if handleGCPMapsFleetEngineReportBillableTrip(w, r, path) {
			return true
		}
		if handleGCPMapsFleetEngineCreateVehicle(w, r, path) {
			return true
		}
		if handleGCPMapsFleetEngineSearchVehicles(w, r, path) {
			return true
		}
		if handleGCPMapsFleetEngineUpdateVehicleAttributes(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPatch:
		if handleGCPMapsFleetEngineUpdateTrip(w, r, path) {
			return true
		}
		if handleGCPMapsFleetEngineUpdateVehicle(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPMapsFleetEngineDeleteTrip(w, path) {
			return true
		}
		if handleGCPMapsFleetEngineDeleteVehicle(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func isGCPMapsFleetEnginePath(path string) bool {
	if strings.HasPrefix(path, "/gcp/maps.fleetengine.v1.TripService/") ||
		strings.HasPrefix(path, "/gcp/maps.fleetengine.v1.VehicleService/") {
		return true
	}

	if !strings.HasPrefix(path, "/gcp/v1/providers/") {
		return false
	}

	return strings.Contains(path, "/trips/") ||
		strings.HasSuffix(path, "/trips") ||
		strings.Contains(path, "/trips:") ||
		strings.Contains(path, "/vehicles/") ||
		strings.HasSuffix(path, "/vehicles") ||
		strings.Contains(path, "/vehicles:")
}

func handleGCPMapsFleetEngineCreateTrip(w http.ResponseWriter, r *http.Request, path string) bool {
	provider, _, list, ok := parseGCPMapsFleetEngineCollectionPath(path, "trips")
	if !ok || !list {
		return false
	}

	body, valid := decodeGCPMapsFleetEngineJSONBody(w, r, path)
	if !valid {
		return true
	}
	if len(body) == 0 {
		respondGCPMapsFleetEngineInvalidArgument(w, path, "trip payload is required")
		return true
	}

	tripID := strings.TrimSpace(r.URL.Query().Get("tripId"))
	if tripID == "" {
		respondGCPMapsFleetEngineInvalidArgument(w, path, "tripId is required")
		return true
	}

	respondJSON(w, http.StatusOK, gcpMapsFleetEngineTrip(provider, tripID))
	return true
}

func handleGCPMapsFleetEngineGetTrip(w http.ResponseWriter, path string) bool {
	provider, tripID, list, ok := parseGCPMapsFleetEngineCollectionPath(path, "trips")
	if !ok || list {
		return false
	}
	respondJSON(w, http.StatusOK, gcpMapsFleetEngineTrip(provider, tripID))
	return true
}

func handleGCPMapsFleetEngineUpdateTrip(w http.ResponseWriter, r *http.Request, path string) bool {
	provider, tripID, list, ok := parseGCPMapsFleetEngineCollectionPath(path, "trips")
	if !ok || list {
		return false
	}

	body, valid := decodeGCPMapsFleetEngineJSONBody(w, r, path)
	if !valid {
		return true
	}
	name, _ := body["name"].(string)
	if strings.TrimSpace(name) == "" {
		respondGCPMapsFleetEngineInvalidArgument(w, path, "name is required for update")
		return true
	}

	respondJSON(w, http.StatusOK, gcpMapsFleetEngineTrip(provider, tripID))
	return true
}

func handleGCPMapsFleetEngineDeleteTrip(w http.ResponseWriter, path string) bool {
	_, _, list, ok := parseGCPMapsFleetEngineCollectionPath(path, "trips")
	if !ok || list {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"deleted": true,
	})
	return true
}

func handleGCPMapsFleetEngineListTrips(w http.ResponseWriter, r *http.Request, path string) bool {
	provider, _, list, ok := parseGCPMapsFleetEngineCollectionPath(path, "trips")
	if !ok || !list {
		return false
	}

	pageSize, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondGCPMapsFleetEngineInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return true
	}
	start := 0
	pageToken := strings.TrimSpace(r.URL.Query().Get("pageToken"))
	if pageToken != "" {
		start, err = parseOptionalNonNegativeInt(pageToken)
		if err != nil {
			respondGCPMapsFleetEngineInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
			return true
		}
	}

	trips := []map[string]any{
		gcpMapsFleetEngineTrip(provider, "trip-1"),
	}
	if start > len(trips) {
		respondGCPMapsFleetEngineInvalidArgument(w, path, "pageToken is out of range")
		return true
	}
	end := len(trips)
	if pageSize > 0 && start+pageSize < end {
		end = start + pageSize
	}
	nextPageToken := ""
	if end < len(trips) {
		nextPageToken = strconv.Itoa(end)
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"trips":         trips[start:end],
		"nextPageToken": nextPageToken,
	})
	return true
}

func handleGCPMapsFleetEngineSearchTrips(w http.ResponseWriter, r *http.Request, path string) bool {
	provider, ok := parseGCPMapsFleetEngineCollectionActionPath(path, "trips", "search")
	if !ok {
		return false
	}

	body, valid := decodeGCPMapsFleetEngineJSONBody(w, r, path)
	if !valid {
		return true
	}
	vehicleID, _ := body["vehicleId"].(string)
	if strings.TrimSpace(vehicleID) == "" {
		respondGCPMapsFleetEngineInvalidArgument(w, path, "vehicleId is required")
		return true
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"trips": []any{
			gcpMapsFleetEngineTrip(provider, "trip-1"),
		},
		"nextPageToken": "",
	})
	return true
}

func handleGCPMapsFleetEngineReportBillableTrip(w http.ResponseWriter, r *http.Request, path string) bool {
	provider, tripID, ok := parseGCPMapsFleetEngineResourceActionPath(path, "trips", "reportBillable")
	if !ok {
		return false
	}

	body, valid := decodeGCPMapsFleetEngineJSONBody(w, r, path)
	if !valid {
		return true
	}
	countryCode, _ := body["countryCode"].(string)
	if strings.TrimSpace(countryCode) == "" {
		respondGCPMapsFleetEngineInvalidArgument(w, path, "countryCode is required")
		return true
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"reported": true,
		"name":     fmt.Sprintf("providers/%s/trips/%s", provider, tripID),
	})
	return true
}

func handleGCPMapsFleetEngineCreateVehicle(w http.ResponseWriter, r *http.Request, path string) bool {
	provider, _, list, ok := parseGCPMapsFleetEngineCollectionPath(path, "vehicles")
	if !ok || !list {
		return false
	}

	body, valid := decodeGCPMapsFleetEngineJSONBody(w, r, path)
	if !valid {
		return true
	}
	if len(body) == 0 {
		respondGCPMapsFleetEngineInvalidArgument(w, path, "vehicle payload is required")
		return true
	}

	vehicleID := strings.TrimSpace(r.URL.Query().Get("vehicleId"))
	if vehicleID == "" {
		respondGCPMapsFleetEngineInvalidArgument(w, path, "vehicleId is required")
		return true
	}

	respondJSON(w, http.StatusOK, gcpMapsFleetEngineVehicle(provider, vehicleID))
	return true
}

func handleGCPMapsFleetEngineGetVehicle(w http.ResponseWriter, path string) bool {
	provider, vehicleID, list, ok := parseGCPMapsFleetEngineCollectionPath(path, "vehicles")
	if !ok || list {
		return false
	}
	respondJSON(w, http.StatusOK, gcpMapsFleetEngineVehicle(provider, vehicleID))
	return true
}

func handleGCPMapsFleetEngineUpdateVehicle(w http.ResponseWriter, r *http.Request, path string) bool {
	provider, vehicleID, list, ok := parseGCPMapsFleetEngineCollectionPath(path, "vehicles")
	if !ok || list {
		return false
	}

	body, valid := decodeGCPMapsFleetEngineJSONBody(w, r, path)
	if !valid {
		return true
	}
	name, _ := body["name"].(string)
	if strings.TrimSpace(name) == "" {
		respondGCPMapsFleetEngineInvalidArgument(w, path, "name is required for update")
		return true
	}

	respondJSON(w, http.StatusOK, gcpMapsFleetEngineVehicle(provider, vehicleID))
	return true
}

func handleGCPMapsFleetEngineDeleteVehicle(w http.ResponseWriter, path string) bool {
	_, _, list, ok := parseGCPMapsFleetEngineCollectionPath(path, "vehicles")
	if !ok || list {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"deleted": true,
	})
	return true
}

func handleGCPMapsFleetEngineListVehicles(w http.ResponseWriter, r *http.Request, path string) bool {
	provider, _, list, ok := parseGCPMapsFleetEngineCollectionPath(path, "vehicles")
	if !ok || !list {
		return false
	}

	pageSize, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondGCPMapsFleetEngineInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return true
	}
	start := 0
	pageToken := strings.TrimSpace(r.URL.Query().Get("pageToken"))
	if pageToken != "" {
		start, err = parseOptionalNonNegativeInt(pageToken)
		if err != nil {
			respondGCPMapsFleetEngineInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
			return true
		}
	}

	vehicles := []map[string]any{
		gcpMapsFleetEngineVehicle(provider, "vehicle-1"),
	}
	if start > len(vehicles) {
		respondGCPMapsFleetEngineInvalidArgument(w, path, "pageToken is out of range")
		return true
	}
	end := len(vehicles)
	if pageSize > 0 && start+pageSize < end {
		end = start + pageSize
	}
	nextPageToken := ""
	if end < len(vehicles) {
		nextPageToken = strconv.Itoa(end)
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"vehicles":      vehicles[start:end],
		"nextPageToken": nextPageToken,
		"totalSize":     len(vehicles),
	})
	return true
}

func handleGCPMapsFleetEngineSearchVehicles(w http.ResponseWriter, r *http.Request, path string) bool {
	provider, ok := parseGCPMapsFleetEngineCollectionActionPath(path, "vehicles", "search")
	if !ok {
		return false
	}

	body, valid := decodeGCPMapsFleetEngineJSONBody(w, r, path)
	if !valid {
		return true
	}
	if _, ok := body["pickupPoint"].(map[string]any); !ok {
		respondGCPMapsFleetEngineInvalidArgument(w, path, "pickupPoint is required")
		return true
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"matches": []any{
			map[string]any{
				"vehicle":  gcpMapsFleetEngineVehicle(provider, "vehicle-1"),
				"tripType": 1,
				"vehiclePickupStraightLineDistanceMeters": 275,
			},
		},
	})
	return true
}

func handleGCPMapsFleetEngineUpdateVehicleAttributes(w http.ResponseWriter, r *http.Request, path string) bool {
	_, vehicleID, ok := parseGCPMapsFleetEngineResourceActionPath(path, "vehicles", "updateAttributes")
	if !ok {
		return false
	}

	body, valid := decodeGCPMapsFleetEngineJSONBody(w, r, path)
	if !valid {
		return true
	}
	attrs, _ := body["attributes"].([]any)
	if len(attrs) == 0 {
		respondGCPMapsFleetEngineInvalidArgument(w, path, "attributes must be a non-empty array")
		return true
	}

	responseAttributes := make([]map[string]any, 0, len(attrs))
	for _, rawAttr := range attrs {
		attrMap, _ := rawAttr.(map[string]any)
		key, _ := attrMap["key"].(string)
		value, _ := attrMap["value"].(string)
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" {
			key = "env"
		}
		if value == "" {
			value = "local"
		}
		responseAttributes = append(responseAttributes, map[string]any{
			"key":   key,
			"value": value,
		})
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"vehicle":    fmt.Sprintf("vehicles/%s", vehicleID),
		"attributes": responseAttributes,
	})
	return true
}

func decodeGCPMapsFleetEngineJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	var body map[string]any
	if r.Body == nil {
		return map[string]any{}, true
	}

	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	if err := decoder.Decode(&body); err != nil && err.Error() != "EOF" {
		respondGCPMapsFleetEngineInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func parseGCPMapsFleetEngineCollectionPath(path, collection string) (provider, resource string, list, ok bool) {
	trimmed := strings.Trim(path, "/")
	parts := strings.Split(trimmed, "/")
	// /gcp/v1/providers/{provider}/{collection}
	// /gcp/v1/providers/{provider}/{collection}/{resource}
	if len(parts) < 5 || len(parts) > 6 {
		return "", "", false, false
	}
	if parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "providers" || parts[4] != collection {
		return "", "", false, false
	}

	provider = strings.TrimSpace(parts[3])
	if provider == "" {
		return "", "", false, false
	}
	if len(parts) == 5 {
		return provider, "", true, true
	}

	resource = strings.TrimSpace(parts[5])
	if resource == "" {
		return "", "", false, false
	}
	return provider, resource, false, true
}

func parseGCPMapsFleetEngineCollectionActionPath(path, collection, action string) (provider string, ok bool) {
	trimmed := strings.Trim(strings.TrimSpace(path), "/")
	if !strings.HasPrefix(trimmed, "gcp/v1/providers/") {
		return "", false
	}
	rest := strings.TrimPrefix(trimmed, "gcp/v1/providers/")
	provider, tail, found := strings.Cut(rest, "/")
	if !found {
		return "", false
	}
	provider = strings.TrimSpace(provider)
	if provider == "" {
		return "", false
	}

	want := collection + ":" + strings.ToLower(action)
	normalizedTail := strings.ToLower(tail)
	normalizedTail = strings.ReplaceAll(normalizedTail, "%3a", ":")
	if normalizedTail != want {
		return "", false
	}
	return provider, true
}

func parseGCPMapsFleetEngineResourceActionPath(path, collection, action string) (provider, resource string, ok bool) {
	trimmed := strings.Trim(strings.TrimSpace(path), "/")
	parts := strings.Split(trimmed, "/")
	// /gcp/v1/providers/{provider}/{collection}/{resource}:{action}
	if len(parts) != 6 {
		return "", "", false
	}
	if parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "providers" || parts[4] != collection {
		return "", "", false
	}

	provider = strings.TrimSpace(parts[3])
	if provider == "" {
		return "", "", false
	}

	resourceAction := strings.TrimSpace(parts[5])
	resourceAction = strings.ReplaceAll(resourceAction, "%3A", ":")
	resourceAction = strings.ReplaceAll(resourceAction, "%3a", ":")
	resource, tailAction, found := strings.Cut(resourceAction, ":")
	if !found {
		return "", "", false
	}
	resource = strings.TrimSpace(resource)
	tailAction = strings.TrimSpace(tailAction)
	if resource == "" || !strings.EqualFold(tailAction, action) {
		return "", "", false
	}
	return provider, resource, true
}

func respondGCPMapsFleetEngineInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func gcpMapsFleetEngineTrip(provider, tripID string) map[string]any {
	return map[string]any{
		"name":       fmt.Sprintf("providers/%s/trips/%s", provider, tripID),
		"tripStatus": 1,
		"tripType":   1,
		"vehicleId":  "vehicle-1",
		"pickupPoint": map[string]any{
			"point": map[string]any{
				"latitude":  37.7749,
				"longitude": -122.4194,
			},
		},
	}
}

func gcpMapsFleetEngineVehicle(provider, vehicleID string) map[string]any {
	return map[string]any{
		"name":               fmt.Sprintf("providers/%s/vehicles/%s", provider, vehicleID),
		"vehicleState":       1,
		"supportedTripTypes": []any{1},
		"maximumCapacity":    4,
		"vehicleType": map[string]any{
			"category": 1,
		},
		"attributes": []any{
			map[string]any{
				"key":   "env",
				"value": "stackyard",
			},
		},
	}
}
