package server

import (
	"net/http"
	"strings"
)

const azureDtl20180915APIVersion = "2018-09-15"

// handleAzureDtl20180915Router owns Microsoft.DevTestLab ARM routes and enforces
// strict api-version handling before staged method implementations are added.
func (s *Server) handleAzureDtl20180915Router(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/azure/") {
		return false
	}

	segments := splitPathSegments(strings.TrimPrefix(path, "/azure/"))
	if !azureDtl20180915OwnsPath(segments) {
		return false
	}

	if !azureDtl20180915HasExpectedAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "api-version must be 2018-09-15")
		return true
	}

	route := azureDtl20180915MatchRoute(segments)
	if route == nil {
		respondJSON(w, http.StatusNotFound, map[string]any{
			"error":    "NotFound",
			"message":  "route is not supported for Azure Dev Test Labs 2018-09-15",
			"provider": providerAzure,
			"path":     path,
		})
		return true
	}

	if _, ok := route.methods[r.Method]; !ok {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return true
	}

	respondAzureImplemented(w, path)
	return true
}

func azureDtl20180915HasExpectedAPIVersion(r *http.Request) bool {
	values, ok := r.URL.Query()["api-version"]
	if !ok || len(values) != 1 {
		return false
	}
	return strings.TrimSpace(values[0]) == azureDtl20180915APIVersion
}

func azureDtl20180915OwnsPath(segments []string) bool {
	for i := 0; i+1 < len(segments); i++ {
		if strings.EqualFold(segments[i], "providers") && strings.EqualFold(segments[i+1], "Microsoft.DevTestLab") {
			return true
		}
	}
	return false
}

func azureDtl20180915MatchRoute(segments []string) *azureDtl20180915RouteDef {
	for idx := range azureDtl20180915Routes {
		route := &azureDtl20180915Routes[idx]
		if len(route.segments) != len(segments) {
			continue
		}
		matched := true
		for i := range route.segments {
			templateSegment := route.segments[i]
			actualSegment := strings.TrimSpace(segments[i])
			if actualSegment == "" {
				matched = false
				break
			}
			if strings.HasPrefix(templateSegment, "{") && strings.HasSuffix(templateSegment, "}") {
				continue
			}
			if !strings.EqualFold(templateSegment, actualSegment) {
				matched = false
				break
			}
		}
		if matched {
			return route
		}
	}
	return nil
}

type azureDtl20180915RouteDef struct {
	template string
	methods  map[string]struct{}
	segments []string
}

var azureDtl20180915Routes = []azureDtl20180915RouteDef{
	{
		template: "/providers/Microsoft.DevTestLab/operations",
		methods:  map[string]struct{}{"GET": {}},
		segments: []string{"providers", "Microsoft.DevTestLab", "operations"},
	},
	{
		template: "/subscriptions/{subscriptionId}/providers/Microsoft.DevTestLab/labs",
		methods:  map[string]struct{}{"GET": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "providers", "Microsoft.DevTestLab", "labs"},
	},
	{
		template: "/subscriptions/{subscriptionId}/providers/Microsoft.DevTestLab/locations/{locationName}/operations/{name}",
		methods:  map[string]struct{}{"GET": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "providers", "Microsoft.DevTestLab", "locations", "{locationName}", "operations", "{name}"},
	},
	{
		template: "/subscriptions/{subscriptionId}/providers/Microsoft.DevTestLab/schedules",
		methods:  map[string]struct{}{"GET": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "providers", "Microsoft.DevTestLab", "schedules"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs",
		methods:  map[string]struct{}{"GET": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{labName}/artifactsources",
		methods:  map[string]struct{}{"GET": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{labName}", "artifactsources"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{labName}/artifactsources/{artifactSourceName}/armtemplates",
		methods:  map[string]struct{}{"GET": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{labName}", "artifactsources", "{artifactSourceName}", "armtemplates"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{labName}/artifactsources/{artifactSourceName}/armtemplates/{name}",
		methods:  map[string]struct{}{"GET": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{labName}", "artifactsources", "{artifactSourceName}", "armtemplates", "{name}"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{labName}/artifactsources/{artifactSourceName}/artifacts",
		methods:  map[string]struct{}{"GET": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{labName}", "artifactsources", "{artifactSourceName}", "artifacts"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{labName}/artifactsources/{artifactSourceName}/artifacts/{name}",
		methods:  map[string]struct{}{"GET": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{labName}", "artifactsources", "{artifactSourceName}", "artifacts", "{name}"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{labName}/artifactsources/{artifactSourceName}/artifacts/{name}/generateArmTemplate",
		methods:  map[string]struct{}{"POST": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{labName}", "artifactsources", "{artifactSourceName}", "artifacts", "{name}", "generateArmTemplate"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{labName}/artifactsources/{name}",
		methods:  map[string]struct{}{"GET": {}, "PUT": {}, "PATCH": {}, "DELETE": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{labName}", "artifactsources", "{name}"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{labName}/costs/{name}",
		methods:  map[string]struct{}{"GET": {}, "PUT": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{labName}", "costs", "{name}"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{labName}/customimages",
		methods:  map[string]struct{}{"GET": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{labName}", "customimages"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{labName}/customimages/{name}",
		methods:  map[string]struct{}{"GET": {}, "PUT": {}, "PATCH": {}, "DELETE": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{labName}", "customimages", "{name}"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{labName}/formulas",
		methods:  map[string]struct{}{"GET": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{labName}", "formulas"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{labName}/formulas/{name}",
		methods:  map[string]struct{}{"GET": {}, "PUT": {}, "PATCH": {}, "DELETE": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{labName}", "formulas", "{name}"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{labName}/galleryimages",
		methods:  map[string]struct{}{"GET": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{labName}", "galleryimages"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{labName}/notificationchannels",
		methods:  map[string]struct{}{"GET": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{labName}", "notificationchannels"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{labName}/notificationchannels/{name}",
		methods:  map[string]struct{}{"GET": {}, "PUT": {}, "PATCH": {}, "DELETE": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{labName}", "notificationchannels", "{name}"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{labName}/notificationchannels/{name}/notify",
		methods:  map[string]struct{}{"POST": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{labName}", "notificationchannels", "{name}", "notify"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{labName}/policysets/{name}/evaluatePolicies",
		methods:  map[string]struct{}{"POST": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{labName}", "policysets", "{name}", "evaluatePolicies"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{labName}/policysets/{policySetName}/policies",
		methods:  map[string]struct{}{"GET": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{labName}", "policysets", "{policySetName}", "policies"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{labName}/policysets/{policySetName}/policies/{name}",
		methods:  map[string]struct{}{"GET": {}, "PUT": {}, "PATCH": {}, "DELETE": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{labName}", "policysets", "{policySetName}", "policies", "{name}"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{labName}/schedules",
		methods:  map[string]struct{}{"GET": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{labName}", "schedules"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{labName}/schedules/{name}",
		methods:  map[string]struct{}{"GET": {}, "PUT": {}, "PATCH": {}, "DELETE": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{labName}", "schedules", "{name}"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{labName}/schedules/{name}/execute",
		methods:  map[string]struct{}{"POST": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{labName}", "schedules", "{name}", "execute"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{labName}/schedules/{name}/listApplicable",
		methods:  map[string]struct{}{"POST": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{labName}", "schedules", "{name}", "listApplicable"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{labName}/servicerunners/{name}",
		methods:  map[string]struct{}{"GET": {}, "PUT": {}, "DELETE": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{labName}", "servicerunners", "{name}"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{labName}/users",
		methods:  map[string]struct{}{"GET": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{labName}", "users"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{labName}/users/{name}",
		methods:  map[string]struct{}{"GET": {}, "PUT": {}, "PATCH": {}, "DELETE": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{labName}", "users", "{name}"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{labName}/users/{userName}/disks",
		methods:  map[string]struct{}{"GET": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{labName}", "users", "{userName}", "disks"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{labName}/users/{userName}/disks/{name}",
		methods:  map[string]struct{}{"GET": {}, "PUT": {}, "PATCH": {}, "DELETE": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{labName}", "users", "{userName}", "disks", "{name}"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{labName}/users/{userName}/disks/{name}/attach",
		methods:  map[string]struct{}{"POST": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{labName}", "users", "{userName}", "disks", "{name}", "attach"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{labName}/users/{userName}/disks/{name}/detach",
		methods:  map[string]struct{}{"POST": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{labName}", "users", "{userName}", "disks", "{name}", "detach"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{labName}/users/{userName}/environments",
		methods:  map[string]struct{}{"GET": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{labName}", "users", "{userName}", "environments"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{labName}/users/{userName}/environments/{name}",
		methods:  map[string]struct{}{"GET": {}, "PUT": {}, "PATCH": {}, "DELETE": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{labName}", "users", "{userName}", "environments", "{name}"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{labName}/users/{userName}/secrets",
		methods:  map[string]struct{}{"GET": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{labName}", "users", "{userName}", "secrets"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{labName}/users/{userName}/secrets/{name}",
		methods:  map[string]struct{}{"GET": {}, "PUT": {}, "PATCH": {}, "DELETE": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{labName}", "users", "{userName}", "secrets", "{name}"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{labName}/users/{userName}/servicefabrics",
		methods:  map[string]struct{}{"GET": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{labName}", "users", "{userName}", "servicefabrics"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{labName}/users/{userName}/servicefabrics/{name}",
		methods:  map[string]struct{}{"GET": {}, "PUT": {}, "PATCH": {}, "DELETE": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{labName}", "users", "{userName}", "servicefabrics", "{name}"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{labName}/users/{userName}/servicefabrics/{name}/listApplicableSchedules",
		methods:  map[string]struct{}{"POST": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{labName}", "users", "{userName}", "servicefabrics", "{name}", "listApplicableSchedules"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{labName}/users/{userName}/servicefabrics/{name}/start",
		methods:  map[string]struct{}{"POST": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{labName}", "users", "{userName}", "servicefabrics", "{name}", "start"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{labName}/users/{userName}/servicefabrics/{name}/stop",
		methods:  map[string]struct{}{"POST": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{labName}", "users", "{userName}", "servicefabrics", "{name}", "stop"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{labName}/users/{userName}/servicefabrics/{serviceFabricName}/schedules",
		methods:  map[string]struct{}{"GET": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{labName}", "users", "{userName}", "servicefabrics", "{serviceFabricName}", "schedules"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{labName}/users/{userName}/servicefabrics/{serviceFabricName}/schedules/{name}",
		methods:  map[string]struct{}{"GET": {}, "PUT": {}, "PATCH": {}, "DELETE": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{labName}", "users", "{userName}", "servicefabrics", "{serviceFabricName}", "schedules", "{name}"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{labName}/users/{userName}/servicefabrics/{serviceFabricName}/schedules/{name}/execute",
		methods:  map[string]struct{}{"POST": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{labName}", "users", "{userName}", "servicefabrics", "{serviceFabricName}", "schedules", "{name}", "execute"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{labName}/virtualmachines",
		methods:  map[string]struct{}{"GET": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{labName}", "virtualmachines"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{labName}/virtualmachines/{name}",
		methods:  map[string]struct{}{"GET": {}, "PUT": {}, "PATCH": {}, "DELETE": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{labName}", "virtualmachines", "{name}"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{labName}/virtualmachines/{name}/addDataDisk",
		methods:  map[string]struct{}{"POST": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{labName}", "virtualmachines", "{name}", "addDataDisk"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{labName}/virtualmachines/{name}/applyArtifacts",
		methods:  map[string]struct{}{"POST": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{labName}", "virtualmachines", "{name}", "applyArtifacts"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{labName}/virtualmachines/{name}/claim",
		methods:  map[string]struct{}{"POST": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{labName}", "virtualmachines", "{name}", "claim"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{labName}/virtualmachines/{name}/detachDataDisk",
		methods:  map[string]struct{}{"POST": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{labName}", "virtualmachines", "{name}", "detachDataDisk"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{labName}/virtualmachines/{name}/getRdpFileContents",
		methods:  map[string]struct{}{"POST": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{labName}", "virtualmachines", "{name}", "getRdpFileContents"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{labName}/virtualmachines/{name}/listApplicableSchedules",
		methods:  map[string]struct{}{"POST": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{labName}", "virtualmachines", "{name}", "listApplicableSchedules"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{labName}/virtualmachines/{name}/redeploy",
		methods:  map[string]struct{}{"POST": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{labName}", "virtualmachines", "{name}", "redeploy"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{labName}/virtualmachines/{name}/resize",
		methods:  map[string]struct{}{"POST": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{labName}", "virtualmachines", "{name}", "resize"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{labName}/virtualmachines/{name}/restart",
		methods:  map[string]struct{}{"POST": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{labName}", "virtualmachines", "{name}", "restart"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{labName}/virtualmachines/{name}/start",
		methods:  map[string]struct{}{"POST": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{labName}", "virtualmachines", "{name}", "start"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{labName}/virtualmachines/{name}/stop",
		methods:  map[string]struct{}{"POST": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{labName}", "virtualmachines", "{name}", "stop"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{labName}/virtualmachines/{name}/transferDisks",
		methods:  map[string]struct{}{"POST": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{labName}", "virtualmachines", "{name}", "transferDisks"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{labName}/virtualmachines/{name}/unClaim",
		methods:  map[string]struct{}{"POST": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{labName}", "virtualmachines", "{name}", "unClaim"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{labName}/virtualmachines/{virtualMachineName}/schedules",
		methods:  map[string]struct{}{"GET": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{labName}", "virtualmachines", "{virtualMachineName}", "schedules"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{labName}/virtualmachines/{virtualMachineName}/schedules/{name}",
		methods:  map[string]struct{}{"GET": {}, "PUT": {}, "PATCH": {}, "DELETE": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{labName}", "virtualmachines", "{virtualMachineName}", "schedules", "{name}"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{labName}/virtualmachines/{virtualMachineName}/schedules/{name}/execute",
		methods:  map[string]struct{}{"POST": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{labName}", "virtualmachines", "{virtualMachineName}", "schedules", "{name}", "execute"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{labName}/virtualnetworks",
		methods:  map[string]struct{}{"GET": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{labName}", "virtualnetworks"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{labName}/virtualnetworks/{name}",
		methods:  map[string]struct{}{"GET": {}, "PUT": {}, "PATCH": {}, "DELETE": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{labName}", "virtualnetworks", "{name}"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{name}",
		methods:  map[string]struct{}{"GET": {}, "PUT": {}, "PATCH": {}, "DELETE": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{name}"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{name}/claimAnyVm",
		methods:  map[string]struct{}{"POST": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{name}", "claimAnyVm"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{name}/createEnvironment",
		methods:  map[string]struct{}{"POST": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{name}", "createEnvironment"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{name}/exportResourceUsage",
		methods:  map[string]struct{}{"POST": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{name}", "exportResourceUsage"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{name}/generateUploadUri",
		methods:  map[string]struct{}{"POST": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{name}", "generateUploadUri"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{name}/importVirtualMachine",
		methods:  map[string]struct{}{"POST": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{name}", "importVirtualMachine"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/labs/{name}/listVhds",
		methods:  map[string]struct{}{"POST": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "labs", "{name}", "listVhds"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/schedules",
		methods:  map[string]struct{}{"GET": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "schedules"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/schedules/{name}",
		methods:  map[string]struct{}{"GET": {}, "PUT": {}, "PATCH": {}, "DELETE": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "schedules", "{name}"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/schedules/{name}/execute",
		methods:  map[string]struct{}{"POST": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "schedules", "{name}", "execute"},
	},
	{
		template: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.DevTestLab/schedules/{name}/retarget",
		methods:  map[string]struct{}{"POST": {}},
		segments: []string{"subscriptions", "{subscriptionId}", "resourceGroups", "{resourceGroupName}", "providers", "Microsoft.DevTestLab", "schedules", "{name}", "retarget"},
	},
}
