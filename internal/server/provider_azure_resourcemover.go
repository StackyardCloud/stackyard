package server

import (
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Microsoft.Migrate (Azure Resource Mover) move collections and their move
// resources, served by the shared ARM handler. Moves are bookkeeping: each
// step changes a move resource's state at once, and no resource is copied
// to the target region or deleted from the source.
func init() {
	const ns = "Microsoft.Migrate"
	actions := map[string]func(*Server, *azureARMResource, map[string]any) (int, any){}
	for verb := range azureMoveSteps {
		actions[verb] = func(s *Server, rec *azureARMResource, payload map[string]any) (int, any) {
			return s.azureMoveStep(rec, verb, payload)
		}
	}
	actions["resolvedependencies"] = func(_ *Server, rec *azureARMResource, _ map[string]any) (int, any) {
		return http.StatusOK, azureMoveOperation(rec)
	}
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "moveCollections",
		Validate:      azureMoveCollectionValidate,
		ServerActions: actions,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "moveCollections/moveResources", Proxy: true, NoPatch: true,
		// The API answers a create with 200 (or 202), never 201.
		CreateStatus:    http.StatusOK,
		Validate:        azureMoveResourceValidate,
		StateProperties: map[string]any{"moveStatus": map[string]any{"moveState": "PreparePending"}},
		ReadOnlyProperties: func(*azureARMResource) map[string]any {
			return map[string]any{"dependsOn": []any{}, "isResolveRequired": false}
		},
	})
	registerAzureARMView(ns, "moveCollections/unresolvedDependencies", func(*Server, string) []map[string]any { return nil })
}

// azureMoveSteps maps each step to the states it starts from and the state
// it leaves a move resource in. "bulkremove" removes the resources.
var azureMoveSteps = map[string]struct {
	From []string
	To   string
}{
	"prepare":      {[]string{"PreparePending", "PrepareFailed"}, "MovePending"},
	"initiatemove": {[]string{"MovePending", "MoveFailed"}, "CommitPending"},
	"commit":       {[]string{"CommitPending", "CommitFailed"}, "DeleteSourcePending"},
	"discard":      {[]string{"CommitPending", "DiscardFailed"}, "MovePending"},
	"bulkremove":   {},
}

var azureMoveResourceTypes = []string{
	"Microsoft.Compute/virtualMachines", "Microsoft.Compute/availabilitySets", "Microsoft.Network/virtualNetworks",
	"Microsoft.Network/networkInterfaces", "Microsoft.Network/networkSecurityGroups", "Microsoft.Network/loadBalancers",
	"Microsoft.Sql/servers", "Microsoft.Sql/servers/elasticPools", "Microsoft.Sql/servers/databases", "resourceGroups",
	"Microsoft.Network/publicIPAddresses", "Microsoft.KeyVault/vaults", "Microsoft.Compute/diskEncryptionSets",
}

func azureMoveCollectionValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	switch azureARMString(props["moveType"]) {
	case "", "RegionToRegion":
		source, target := azureARMString(props["sourceRegion"]), azureARMString(props["targetRegion"])
		if source == "" || target == "" {
			return "properties.sourceRegion and targetRegion are required for a RegionToRegion move."
		}
		if azureLocationHost(source) == azureLocationHost(target) {
			return "properties.targetRegion must differ from sourceRegion."
		}
	case "RegionToZone":
		if azureARMString(props["moveRegion"]) == "" {
			return "properties.moveRegion is required for a RegionToZone move."
		}
	default:
		return "properties.moveType must be RegionToRegion or RegionToZone."
	}
	return ""
}

func azureMoveResourceValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	source := strings.ToLower(azureARMString(props["sourceId"]))
	if !strings.HasPrefix(source, "/subscriptions/") || !strings.Contains(source, "/resourcegroups/") {
		return "properties.sourceId must be a resource or resource group ID."
	}
	if settings, present := props["resourceSettings"]; present {
		kind := azureARMString(azureARMObject(settings)["resourceType"])
		if !slices.ContainsFunc(azureMoveResourceTypes, func(t string) bool { return strings.EqualFold(t, kind) }) {
			return "properties.resourceSettings.resourceType must be a type Resource Mover can move, such as Microsoft.Compute/virtualMachines."
		}
	}
	return ""
}

func azureMoveOperation(rec *azureARMResource) map[string]any {
	now := time.Now().UTC().Format(time.RFC3339)
	id := uuid.NewString()
	return map[string]any{"id": rec.ID + "/operations/" + id, "name": id, "status": "Succeeded", "startTime": now, "endTime": now, "properties": map[string]any{}}
}

// azureMoveStep applies a step to the listed move resources of collection,
// named by move resource ID or, with moveResourceInputType
// MoveResourceSourceId, by source ID. Every resource must be in a state the
// step starts from, or none changes; validateOnly checks without changing.
func (s *Server) azureMoveStep(collection *azureARMResource, verb string, payload map[string]any) (int, any) {
	listed := azureARMSlice(payload["moveResources"])
	if len(listed) == 0 && verb != "bulkremove" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "moveResources must list at least one move resource.")
	}
	bySource := false
	switch payload["moveResourceInputType"] {
	case nil, "MoveResourceId":
	case "MoveResourceSourceId":
		bySource = true
	default:
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "moveResourceInputType must be MoveResourceId or MoveResourceSourceId.")
	}
	step := azureMoveSteps[verb]

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	prefix := azureARMKey(collection.ID + "/moveResources/")
	targets := make([]*azureARMResource, 0, len(listed))
	for _, raw := range listed {
		want := azureARMString(raw)
		var found *azureARMResource
		for key, rec := range s.azureARMResources {
			if !strings.HasPrefix(key, prefix) || strings.Contains(key[len(prefix):], "/") {
				continue
			}
			if (!bySource && key == azureARMKey(want)) || (bySource && strings.EqualFold(azureARMString(rec.Properties["sourceId"]), want)) {
				found = rec
				break
			}
		}
		if found == nil {
			return http.StatusBadRequest, azureARMErrorBody("BadRequest", "Move resource "+want+" was not found in the move collection.")
		}
		state := azureARMString(azureARMObject(found.Properties["moveStatus"])["moveState"])
		if verb != "bulkremove" && !slices.Contains(step.From, state) {
			return http.StatusBadRequest, azureARMErrorBody("BadRequest", "Move resource "+found.Name+" is "+state+"; "+verb+" needs it to be "+strings.Join(step.From, " or ")+".")
		}
		targets = append(targets, found)
	}
	if payload["validateOnly"] == true {
		return http.StatusOK, azureMoveOperation(collection)
	}
	now := time.Now().UTC()
	for _, rec := range targets {
		if verb == "bulkremove" {
			delete(s.azureARMResources, azureARMKey(rec.ID))
			continue
		}
		updated := *rec
		updated.Properties = azureARMMerge(rec.Properties, map[string]any{"moveStatus": map[string]any{"moveState": step.To}})
		updated.UpdatedAt = now
		s.azureARMResources[azureARMKey(rec.ID)] = &updated
	}
	return http.StatusOK, azureMoveOperation(collection)
}
