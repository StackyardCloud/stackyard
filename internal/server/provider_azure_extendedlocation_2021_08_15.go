package server

import (
	"cmp"
	"strings"
)

const azureExtendedLocation20210815APIVersion = "2021-08-15"

// Azure Arc custom locations (Microsoft.ExtendedLocation 2021-08-15) on the
// shared ARM handler, with the resource types each location's cluster
// extensions enable and the operations list. Network Cloud and other
// services place resources on these locations by ID.

const (
	xlocNamespace     = "Microsoft.ExtendedLocation"
	xlocMaxExtensions = 50
)

var (
	xlocHostID      = azureNCAnyID
	xlocExtensionID = azureNCAnyID
)

func init() {
	ns := xlocNamespace
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "customLocations",
		Validate:            xlocValidate,
		WriteOnlyProperties: []string{"authentication.value"},
		PatchBody: func(payload map[string]any) (map[string]any, map[string]any, string) {
			props := azureARMObject(payload["properties"])
			if _, present := props["clusterExtensionIds"]; present {
				if msg := xlocExtensions(props); msg != "" {
					return nil, nil, msg
				}
			}
			return payload, nil, cmp.Or(
				azureLabEnum(azureARMObject(payload["identity"]), "identity.", "type", []string{"SystemAssigned", "None"}),
				azureLabEnum(props, "properties.", "hostType", []string{"Kubernetes"}),
			)
		},
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			props := map[string]any{}
			if _, set := rec.Properties["hostType"]; !set {
				props["hostType"] = "Kubernetes"
			}
			if _, set := rec.Properties["displayName"]; !set {
				props["displayName"] = rec.Name
			}
			return props
		},
	})
	registerAzureARMView(ns, "customLocations/enabledResourceTypes", func(s *Server, scope string) []map[string]any {
		loc, _ := s.getAzureARMResource(scope, "", "", "")
		if loc == nil {
			return nil
		}
		var out []map[string]any
		for _, raw := range azureARMSlice(loc.Properties["clusterExtensionIds"]) {
			extID := azureARMString(raw)
			name := azureHZGUID(loc.ID + "#" + extID)
			out = append(out, map[string]any{
				"id": scope + "/enabledResourceTypes/" + name, "name": name, "type": ns + "/customLocations/enabledResourceTypes",
				"properties": map[string]any{
					"clusterExtensionId": extID,
					"extensionType":      extID[strings.LastIndex(extID, "/")+1:],
					"typesMetadata":      []any{},
				},
			})
		}
		return out
	})
	registerAzureARMView(ns, "operations", elasOperations("Microsoft Extended Location", ns, "customLocations"))
}

func xlocValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	msg := cmp.Or(
		azureLabEnum(azureARMObject(payload["identity"]), "identity.", "type", []string{"SystemAssigned", "None"}),
		azureWorkloadsRequired(props, "properties.", "hostResourceId", "namespace"),
		azureLabEnum(props, "properties.", "hostType", []string{"Kubernetes"}),
		xlocExtensions(props),
	)
	if msg == "" && !xlocHostID.MatchString(azureARMString(props["hostResourceId"])) {
		msg = "properties.hostResourceId is invalid: it must be a resource ID."
	}
	if auth, present := props["authentication"]; msg == "" && present {
		msg = azureWorkloadsRequired(azureARMObject(auth), "properties.authentication.", "type", "value")
	}
	return msg
}

// xlocExtensions checks properties.clusterExtensionIds: one or more cluster
// extension resource IDs.
func xlocExtensions(props map[string]any) string {
	ids := azureARMSlice(props["clusterExtensionIds"])
	switch {
	case len(ids) == 0:
		return "properties.clusterExtensionIds is required."
	case len(ids) > xlocMaxExtensions:
		return "properties.clusterExtensionIds is invalid: at most 50 extensions."
	}
	for _, raw := range ids {
		id, _ := raw.(string)
		if !xlocExtensionID.MatchString(id) || !strings.Contains(strings.ToLower(id), "/providers/microsoft.kubernetesconfiguration/extensions/") {
			return "properties.clusterExtensionIds is invalid: '" + azureARMString(raw) + "' isn't a cluster extension resource ID."
		}
	}
	return ""
}
