package server

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
)

const azureComputeRecommenderAPIVersion = "2025-06-05"

var azureComputeRecommenderSubscriptionPattern = regexp.MustCompile(`^[0-9a-fA-F-]{36}$`)

type azureComputeRecommenderGenerateRequest struct {
	AvailabilityZones []string
	DesiredCount      int
	DesiredLocations  []string
	DesiredSizes      []map[string]any
}

func (s *Server) handleAzureComputeRecommenderRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/azure/providers/") && !strings.HasPrefix(path, "/azure/subscriptions/") {
		return false
	}

	segments := splitPathSegments(strings.TrimPrefix(path, "/azure/"))
	if len(segments) == 0 || !azureComputeRecommenderIsPath(segments) {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}
	if !azureComputeRecommenderAllowedMethod(r.Method) {
		return false
	}

	if len(segments) == 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.Compute") &&
		strings.EqualFold(segments[2], "operations") {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return true
		}
		respondJSON(w, http.StatusOK, map[string]any{"value": azureComputeRecommenderOperationsFixture()})
		return true
	}

	subscriptionID, location, tail, ok := azureComputeRecommenderRouteContext(segments)
	if !ok || len(tail) == 0 {
		return false
	}
	if !azureComputeRecommenderValidSubscriptionID(subscriptionID) {
		respondAzureInvalidRequest(w, path, "subscriptionId is invalid")
		return true
	}
	if strings.TrimSpace(location) == "" {
		respondAzureInvalidRequest(w, path, "location is required")
		return true
	}
	if !strings.EqualFold(tail[0], "placementScores") || len(tail) < 2 || !strings.EqualFold(tail[1], "spot") {
		return false
	}

	if len(tail) == 2 {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return true
		}
		respondJSON(w, http.StatusOK, azureComputeRecommenderMetadataFixture(subscriptionID, location))
		return true
	}
	if len(tail) == 3 && strings.EqualFold(tail[2], "generate") {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return true
		}
		s.handleAzureComputeRecommenderGenerate(w, r, path, subscriptionID, location)
		return true
	}

	respondAzureImplemented(w, path)
	return true
}

func azureComputeRecommenderIsPath(segments []string) bool {
	if len(segments) == 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.Compute") &&
		strings.EqualFold(segments[2], "operations") {
		return true
	}
	_, _, tail, ok := azureComputeRecommenderRouteContext(segments)
	return ok && len(tail) >= 2 &&
		strings.EqualFold(tail[0], "placementScores") &&
		strings.EqualFold(tail[1], "spot")
}

func azureComputeRecommenderAllowedMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodPost:
		return true
	default:
		return false
	}
}

func azureComputeRecommenderRouteContext(segments []string) (subscriptionID, location string, tail []string, ok bool) {
	if len(segments) < 7 || !strings.EqualFold(segments[0], "subscriptions") {
		return "", "", nil, false
	}
	subscriptionID = strings.TrimSpace(segments[1])
	for idx := 2; idx+3 < len(segments); idx++ {
		if strings.EqualFold(segments[idx], "providers") &&
			strings.EqualFold(segments[idx+1], "Microsoft.Compute") &&
			strings.EqualFold(segments[idx+2], "locations") {
			location = strings.TrimSpace(segments[idx+3])
			return subscriptionID, location, segments[idx+4:], true
		}
	}
	return "", "", nil, false
}

func (s *Server) handleAzureComputeRecommenderGenerate(w http.ResponseWriter, r *http.Request, path, subscriptionID, location string) {
	_, payload, ok := azureComputeRecommenderReadRequestBody(w, r, path)
	if !ok {
		return
	}
	req, err := azureComputeRecommenderParseGenerateRequest(payload)
	if err != nil {
		respondAzureInvalidRequest(w, path, err.Error())
		return
	}

	responseLocation := fmt.Sprintf("/subscriptions/%s/providers/Microsoft.Compute/locations/%s/placementScores/spot/generateResults/stackyard", subscriptionID, location)
	w.Header().Set("Location", responseLocation)
	respondJSON(w, http.StatusOK, azureComputeRecommenderGenerateFixture(subscriptionID, location, req))
}

func azureComputeRecommenderParseGenerateRequest(payload map[string]any) (azureComputeRecommenderGenerateRequest, error) {
	req := azureComputeRecommenderGenerateRequest{
		AvailabilityZones: azureComputeRecommenderStringSlice(payload["availabilityZones"]),
		DesiredLocations:  azureComputeRecommenderStringSlice(payload["desiredLocations"]),
		DesiredSizes:      azureComputeRecommenderSizeList(payload["desiredSizes"]),
	}

	desiredCount, ok := azureComputeRecommenderInt(payload["desiredCount"])
	if !ok {
		return req, fmt.Errorf("desiredCount is required and must be an integer")
	}
	if desiredCount < 1 {
		return req, fmt.Errorf("desiredCount must be greater than 0")
	}
	req.DesiredCount = desiredCount
	if len(req.DesiredLocations) == 0 {
		return req, fmt.Errorf("desiredLocations is required")
	}
	if len(req.DesiredSizes) == 0 {
		return req, fmt.Errorf("desiredSizes is required")
	}
	for _, size := range req.DesiredSizes {
		if strings.TrimSpace(azureComputeRecommenderString(size["sku"])) == "" {
			return req, fmt.Errorf("each desiredSizes entry must include sku")
		}
	}
	return req, nil
}

func azureComputeRecommenderMetadataFixture(subscriptionID, location string) map[string]any {
	return map[string]any{
		"id":   fmt.Sprintf("/subscriptions/%s/providers/Microsoft.Compute/locations/%s/placementScores/spot", subscriptionID, location),
		"name": "spot",
		"type": "Microsoft.Compute/placementScores",
		"properties": map[string]any{
			"supportedResourceTypes": []string{
				"Microsoft.Compute/virtualMachines",
				"Microsoft.Compute/virtualMachineScaleSets",
			},
			"supportedScoreValues": []string{
				"High",
				"Medium",
				"Low",
				"DataNotFoundOrStale",
			},
		},
	}
}

func azureComputeRecommenderGenerateFixture(subscriptionID, location string, req azureComputeRecommenderGenerateRequest) map[string]any {
	placementScores := make([]map[string]any, 0, len(req.DesiredLocations)*len(req.DesiredSizes)*max(1, len(req.AvailabilityZones)))

	locations := append([]string{}, req.DesiredLocations...)
	sort.Strings(locations)
	sizes := make([]map[string]any, len(req.DesiredSizes))
	copy(sizes, req.DesiredSizes)
	sort.Slice(sizes, func(i, j int) bool {
		return strings.Compare(strings.ToLower(azureComputeRecommenderString(sizes[i]["sku"])), strings.ToLower(azureComputeRecommenderString(sizes[j]["sku"]))) < 0
	})

	zones := append([]string{}, req.AvailabilityZones...)
	if len(zones) == 0 {
		zones = []string{""}
	}
	sort.Strings(zones)

	for _, region := range locations {
		for _, size := range sizes {
			sku := azureComputeRecommenderString(size["sku"])
			for _, zone := range zones {
				score := azureComputeRecommenderScore(region, sku, zone, req.DesiredCount)
				placementScores = append(placementScores, map[string]any{
					"availabilityZone": zone,
					"isQuotaAvailable": score != "Low",
					"region":           region,
					"score":            score,
					"sku":              sku,
				})
			}
		}
	}

	return map[string]any{
		"id":                fmt.Sprintf("/subscriptions/%s/providers/Microsoft.Compute/locations/%s/placementScores/spot/generateResults/stackyard", subscriptionID, location),
		"name":              "spot",
		"type":              "Microsoft.Compute/placementScores/generate",
		"availabilityZones": req.AvailabilityZones,
		"desiredCount":      req.DesiredCount,
		"desiredLocations":  req.DesiredLocations,
		"desiredSizes":      req.DesiredSizes,
		"placementScores":   placementScores,
	}
}

func azureComputeRecommenderScore(region, sku, zone string, desiredCount int) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(strings.ToLower(region + "|" + sku + "|" + zone)))
	value := int(h.Sum32()) + desiredCount
	switch value % 4 {
	case 0:
		return "High"
	case 1:
		return "Medium"
	case 2:
		return "Low"
	default:
		return "DataNotFoundOrStale"
	}
}

func azureComputeRecommenderOperationsFixture() []map[string]any {
	return []map[string]any{
		{
			"name":       "Microsoft.Compute/placementScores/read",
			"origin":     "system",
			"actionType": "Internal",
			"display": map[string]any{
				"provider":  "Microsoft Compute Recommender",
				"resource":  "Placement Scores",
				"operation": "Get Spot Placement Score metadata",
			},
			"isDataAction": false,
		},
		{
			"name":       "Microsoft.Compute/placementScores/generate/action",
			"origin":     "system",
			"actionType": "Internal",
			"display": map[string]any{
				"provider":  "Microsoft Compute Recommender",
				"resource":  "Placement Scores",
				"operation": "Generate Spot Placement Scores",
			},
			"isDataAction": false,
		},
	}
}

func azureComputeRecommenderReadRequestBody(w http.ResponseWriter, r *http.Request, path string) ([]byte, map[string]any, bool) {
	if r.Body == nil {
		return nil, map[string]any{}, true
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondAzureInvalidRequest(w, path, "failed to read request body")
		return nil, nil, false
	}
	if len(strings.TrimSpace(string(body))) == 0 {
		return body, map[string]any{}, true
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		respondAzureInvalidRequest(w, path, "request body must be valid JSON")
		return nil, nil, false
	}
	return body, payload, true
}

func azureComputeRecommenderValidSubscriptionID(value string) bool {
	return azureComputeRecommenderSubscriptionPattern.MatchString(strings.TrimSpace(value))
}

func azureComputeRecommenderString(value any) string {
	if s, ok := value.(string); ok {
		return s
	}
	return ""
}

func azureComputeRecommenderStringSlice(value any) []string {
	raw, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	seen := map[string]struct{}{}
	for _, item := range raw {
		s, ok := item.(string)
		if !ok {
			continue
		}
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		key := strings.ToLower(s)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, s)
	}
	return out
}

func azureComputeRecommenderSizeList(value any) []map[string]any {
	raw, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, entry)
	}
	return out
}

func azureComputeRecommenderInt(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int32:
		return int(typed), true
	case int64:
		return int(typed), true
	case float64:
		if typed == float64(int(typed)) {
			return int(typed), true
		}
	}
	return 0, false
}
