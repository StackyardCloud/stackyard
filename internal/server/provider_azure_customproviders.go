package server

import (
	"fmt"
	"slices"
	"strings"
)

// Microsoft.CustomProviders resource providers and associations, served by
// the shared ARM handler. Providers are only registered: the emulator never
// calls their endpoints, so custom actions and resource types are not
// served.
func init() {
	const ns = "Microsoft.CustomProviders"
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "resourceProviders",
		ValidateName: func(name string) string {
			if len(name) < 3 || len(name) > 64 {
				return "Custom resource provider names must be 3 to 64 characters."
			}
			return ""
		},
		Validate: azureCustomProviderValidate,
		// Updates change tags only.
		PatchBody: func(payload map[string]any) (map[string]any, map[string]any, string) {
			body := map[string]any{}
			if tags, present := payload["tags"]; present {
				body["tags"] = tags
			}
			return body, nil, ""
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "associations", Extension: true,
		Validate: func(payload map[string]any) string {
			target := strings.ToLower(azureARMString(azureARMObject(payload["properties"])["targetResourceId"]))
			if !strings.Contains(target, "/providers/microsoft.customproviders/resourceproviders/") {
				return "properties.targetResourceId must be a custom resource provider ID."
			}
			return ""
		},
		Linked: azureCustomProviderAssociationLinked,
	})
}

// azureCustomProviderValidate checks the manifest's actions, resource types
// and validations.
func azureCustomProviderValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	for _, routes := range []struct {
		key     string
		routing []string
	}{
		{"actions", []string{"Proxy"}},
		{"resourceTypes", []string{"Proxy", "Proxy,Cache"}},
	} {
		var names []string
		for i, raw := range azureARMSlice(props[routes.key]) {
			route := azureARMObject(raw)
			at := fmt.Sprintf("properties.%s[%d]", routes.key, i)
			name := azureARMString(route["name"])
			switch {
			case strings.TrimSpace(name) == "" || strings.Contains(name, "/"):
				return at + ".name is required and can't contain '/'."
			case slices.ContainsFunc(names, func(n string) bool { return strings.EqualFold(n, name) }):
				return at + ".name " + name + " is listed twice."
			case !strings.HasPrefix(azureARMString(route["endpoint"]), "https://") || len(azureARMString(route["endpoint"])) <= len("https://"):
				return at + ".endpoint must be an HTTPS URL."
			}
			if raw, present := route["routingType"]; present && !slices.Contains(routes.routing, azureARMString(raw)) {
				return at + ".routingType must be " + strings.Join(routes.routing, " or ") + "."
			}
			names = append(names, name)
		}
	}
	for i, raw := range azureARMSlice(props["validations"]) {
		validation := azureARMObject(raw)
		if kind, present := validation["validationType"]; present && kind != "Swagger" {
			return fmt.Sprintf("properties.validations[%d].validationType must be Swagger.", i)
		}
		if !strings.HasPrefix(azureARMString(validation["specification"]), "https://raw.githubusercontent.com/") {
			return fmt.Sprintf("properties.validations[%d].specification must be a raw.githubusercontent.com URL.", i)
		}
	}
	return ""
}

// azureCustomProviderAssociationLinked requires the target provider to
// exist.
func azureCustomProviderAssociationLinked(s *Server, rec *azureARMResource) (map[string]any, []*azureARMResource, string) {
	target := azureARMString(rec.Properties["targetResourceId"])
	if s.azureARMResources[azureARMKey(target)] == nil {
		return nil, nil, "Custom resource provider " + target + " was not found."
	}
	return nil, nil, ""
}
