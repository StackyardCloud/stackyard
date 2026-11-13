package server

import (
	"strings"
	"time"
)

// Microsoft.ResourceHealth availability statuses and service health events,
// computed from emulator state: every resource is Available and no events
// are active.
func init() {
	const ns = "Microsoft.ResourceHealth"
	registerAzureARMView(ns, "availabilityStatuses", azureResourceHealthStatuses)
	registerAzureARMView(ns, "events", func(*Server, string) []map[string]any { return nil })
}

// azureResourceHealthStatuses returns the "current" status of the resource
// at scope, or of every top-level resource in a subscription or resource
// group scope.
func azureResourceHealthStatuses(s *Server, scope string) []map[string]any {
	if scope == "" {
		return nil
	}
	if strings.Contains(strings.ToLower(scope), "/providers/") {
		// ponytail: any resource ID is Available, since several Azure services keep their own stores; check existence when all of them share one.
		return []map[string]any{azureAvailabilityStatus(scope, "", time.Now().UTC())}
	}
	prefix := azureARMKey(scope + "/")
	records := s.azureARMResourcesWhere(func(rec *azureARMResource) bool {
		namespace, typeName, _ := strings.Cut(rec.Type, "/")
		typ, ok := lookupAzureARMResourceType(namespace, typeName)
		// Tracked resources only: proxy and extension resources have no health.
		return ok && !typ.Proxy && strings.HasPrefix(azureARMKey(rec.ID), prefix)
	})
	out := make([]map[string]any, 0, len(records))
	for _, rec := range records {
		out = append(out, azureAvailabilityStatus(rec.ID, rec.Location, rec.CreatedAt))
	}
	return out
}

func azureAvailabilityStatus(resourceID, location string, since time.Time) map[string]any {
	status := map[string]any{
		"id":   resourceID + "/providers/Microsoft.ResourceHealth/availabilityStatuses/current",
		"name": "current",
		"type": "Microsoft.ResourceHealth/AvailabilityStatuses",
		"properties": map[string]any{
			"availabilityState": "Available",
			"title":             "Available",
			"summary":           "There aren't any known Azure platform problems affecting this resource.",
			"reasonChronicity":  "Persistent",
			"occuredTime":       since.Format(time.RFC3339),
			"reportedTime":      time.Now().UTC().Format(time.RFC3339),
		},
	}
	if location != "" {
		status["location"] = location
	}
	return status
}
