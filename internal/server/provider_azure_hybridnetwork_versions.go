package server

import (
	"encoding/json"
	"maps"
	"slices"
	"strconv"
	"strings"
)

// azureHNJSONObject parses a property that holds a JSON object as a
// string, such as a schema or deployment values.
func azureHNJSONObject(raw any, path string) (map[string]any, string) {
	text, ok := raw.(string)
	if !ok {
		return nil, path + " must be a string holding a JSON object."
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(text), &out); err != nil || out == nil {
		return nil, path + " must be a well-formed JSON object."
	}
	return out, ""
}

// azureHNSchemaMissing returns the path of the first property a JSON
// schema requires that value lacks, recursing into the object properties
// the schema describes, or "".
func azureHNSchemaMissing(schema, value map[string]any, path string) string {
	for _, raw := range azureHNList(schema["required"]) {
		if key := azureARMString(raw); key != "" {
			if _, ok := value[key]; !ok {
				return path + key
			}
		}
	}
	props := azureARMObject(schema["properties"])
	for _, key := range slices.Sorted(maps.Keys(props)) {
		if child, ok := value[key].(map[string]any); ok {
			if missing := azureHNSchemaMissing(azureARMObject(props[key]), child, path+key+"."); missing != "" {
				return missing
			}
		}
	}
	return ""
}

// azureHNCheckValues checks JSON values against a schema held as a
// string; an empty schema accepts anything.
func azureHNCheckValues(schemaText string, values map[string]any, what string) string {
	if schemaText == "" {
		return ""
	}
	schema, msg := azureHNJSONObject(schemaText, what+" schema")
	if msg != "" {
		return ""
	}
	if missing := azureHNSchemaMissing(schema, values, ""); missing != "" {
		return what + " lacks " + missing + ", which its schema requires."
	}
	return ""
}

// azureHNRefLocked returns the stored resource of the given type that a
// reference's ID names. Caller holds providerStorageMu.
func (s *Server) azureHNRefLocked(id, typ, what string) (*azureARMResource, string) {
	if id == "" {
		return nil, what + " ID is required."
	}
	rec := s.azureARMResources[azureARMKey(id)]
	if rec == nil || !strings.EqualFold(rec.Type, azureHNNamespace+"/"+typ) {
		return nil, "The " + what + " '" + id + "' was not found."
	}
	return rec, ""
}

// azureHNSamePublisher checks that a version's artifact store or schema
// belongs to the version's own publisher.
func azureHNSamePublisher(rec, ref *azureARMResource, what string) string {
	publisher := func(id string) string {
		if parts := strings.Split(id, "/"); len(parts) > 8 {
			return strings.ToLower(strings.Join(parts[:9], "/"))
		}
		return ""
	}
	if publisher(rec.ID) != publisher(ref.ID) {
		return "The " + what + " " + ref.Name + " belongs to another publisher; a version can use only its own publisher's resources."
	}
	return ""
}

func azureHNCGSValidate(payload map[string]any) string {
	if msg := azureHNDescribed(payload); msg != "" {
		return msg
	}
	_, msg := azureHNJSONObject(azureARMObject(payload["properties"])["schemaDefinition"], "properties.schemaDefinition")
	return msg
}

// azureHNNFDVApps maps each NFVI type to the artifact types its network
// function applications may have, and the profile naming the artifact.
var azureHNNFDVApps = map[string]map[string][2]string{
	"AzureArcKubernetes": {"HelmPackage": {"helmArtifactProfile", "helmPackageName"}},
	"AzureCore":          {"ArmTemplate": {"templateArtifactProfile", "templateName"}, "VhdImageFile": {"vhdArtifactProfile", "vhdName"}},
	"AzureOperatorNexus": {"ArmTemplate": {"templateArtifactProfile", "templateName"}, "ImageFile": {"imageArtifactProfile", "imageName"}},
}

func azureHNNFDVValidate(payload map[string]any) string {
	if msg := azureHNDescribed(payload); msg != "" {
		return msg
	}
	props := azureARMObject(payload["properties"])
	nfType := azureARMString(props["networkFunctionType"])
	nfvis := map[string][]string{"ContainerizedNetworkFunction": {"AzureArcKubernetes"}, "VirtualNetworkFunction": {"AzureCore", "AzureOperatorNexus"}}[nfType]
	if nfvis == nil {
		return "properties.networkFunctionType must be ContainerizedNetworkFunction or VirtualNetworkFunction."
	}
	if raw, present := props["deployParameters"]; present {
		if _, msg := azureHNJSONObject(raw, "properties.deployParameters"); msg != "" {
			return msg
		}
	}
	template, ok := props["networkFunctionTemplate"].(map[string]any)
	if !ok {
		return "properties.networkFunctionTemplate is required."
	}
	nfvi := azureARMString(template["nfviType"])
	if !slices.Contains(nfvis, nfvi) {
		return "properties.networkFunctionTemplate.nfviType must be " + strings.Join(nfvis, " or ") + " for a " + nfType + "."
	}
	apps, msg := azureHNNamedItems(template["networkFunctionApplications"], "properties.networkFunctionTemplate.networkFunctionApplications")
	if msg != "" {
		return msg
	}
	kinds := map[string]bool{}
	for i, app := range apps {
		path := "properties.networkFunctionTemplate.networkFunctionApplications[" + strconv.Itoa(i) + "]."
		kind := azureARMString(app["artifactType"])
		profile, ok := azureHNNFDVApps[nfvi][kind]
		if !ok {
			return path + "artifactType must be one of " + strings.Join(slices.Sorted(maps.Keys(azureHNNFDVApps[nfvi])), ", ") + " on " + nfvi + "."
		}
		kinds[kind] = true
		artifact := azureARMObject(app["artifactProfile"])
		if azureARMString(azureARMObject(artifact["artifactStore"])["id"]) == "" {
			return path + "artifactProfile.artifactStore.id is required."
		}
		if azureARMString(azureARMObject(artifact[profile[0]])[profile[1]]) == "" {
			return path + "artifactProfile." + profile[0] + "." + profile[1] + " is required."
		}
		rule := azureARMObject(app["deployParametersMappingRuleProfile"])
		if msg := azureLabEnum(rule, path+"deployParametersMappingRuleProfile.", "applicationEnablement", []string{"Unknown", "Enabled", "Disabled"}); msg != "" {
			return msg
		}
	}
	// Learn's onboarding guidance: a VNF on Azure Core deploys at least one
	// VHD image with at least one ARM template.
	if nfvi == "AzureCore" && (!kinds["VhdImageFile"] || !kinds["ArmTemplate"]) {
		return "An AzureCore virtual network function needs at least one VhdImageFile and one ArmTemplate application."
	}
	return ""
}

// azureHNNamedItems checks a non-empty array of objects with unique names
// whose dependsOnProfile lists name only their siblings.
func azureHNNamedItems(raw any, path string) ([]map[string]any, string) {
	list, ok := raw.([]any)
	if !ok || len(list) == 0 {
		return nil, path + " must be a non-empty array."
	}
	items := make([]map[string]any, 0, len(list))
	names := map[string]bool{}
	for i, item := range list {
		m, ok := item.(map[string]any)
		name := strings.ToLower(azureARMString(m["name"]))
		if !ok || name == "" {
			return nil, path + "[" + strconv.Itoa(i) + "] must be an object with a name."
		}
		if names[name] {
			return nil, path + " has two items named " + azureARMString(m["name"]) + "."
		}
		names[name] = true
		items = append(items, m)
	}
	for i, m := range items {
		depends := azureARMObject(m["dependsOnProfile"])
		for _, key := range []string{"installDependsOn", "uninstallDependsOn", "updateDependsOn"} {
			for _, dep := range azureHNList(depends[key]) {
				if !names[strings.ToLower(azureARMString(dep))] {
					return nil, path + "[" + strconv.Itoa(i) + "].dependsOnProfile." + key + " names " + azureARMString(dep) + ", which isn't in the list."
				}
			}
		}
	}
	return items, ""
}

// azureHNNFDVReconcile checks that each application's artifact store
// exists and can hold its artifact.
func azureHNNFDVReconcile(s *Server, rec *azureARMResource) string {
	template := azureARMObject(rec.Properties["networkFunctionTemplate"])
	for _, item := range azureHNList(template["networkFunctionApplications"]) {
		app := azureARMObject(item)
		id := azureARMString(azureARMObject(azureARMObject(app["artifactProfile"])["artifactStore"])["id"])
		store, msg := s.azureHNRefLocked(id, azureHNStore, "artifact store")
		if msg != "" {
			return msg
		}
		if msg := azureHNSamePublisher(rec, store, "artifact store"); msg != "" {
			return msg
		}
		if storage := store.Properties["storeType"] == "AzureStorageAccount"; storage != (app["artifactType"] == "VhdImageFile") {
			return "The " + azureARMString(store.Properties["storeType"]) + " store " + store.Name + " can't hold the " + azureARMString(app["artifactType"]) + " of application " + azureARMString(app["name"]) + "."
		}
	}
	return ""
}

var azureHNNFVITypes = []string{"Unknown", "AzureArcKubernetes", "AzureCore", "AzureOperatorNexus"}

func azureHNNSDVValidate(payload map[string]any) string {
	if msg := azureHNDescribed(payload); msg != "" {
		return msg
	}
	props := azureARMObject(payload["properties"])
	if msg := azureDataBoxObjects(props, "properties.", "configurationGroupSchemaReferences", "nfvisFromSite"); msg != "" {
		return msg
	}
	for key, raw := range azureARMObject(props["configurationGroupSchemaReferences"]) {
		if azureARMString(azureARMObject(raw)["id"]) == "" {
			return "properties.configurationGroupSchemaReferences." + key + ".id is required."
		}
	}
	for key, raw := range azureARMObject(props["nfvisFromSite"]) {
		nfvi := azureARMObject(raw)
		if azureARMString(nfvi["name"]) == "" {
			return "properties.nfvisFromSite." + key + ".name is required."
		}
		if !slices.Contains(azureHNNFVITypes, azureARMString(nfvi["type"])) {
			return "properties.nfvisFromSite." + key + ".type must be one of " + strings.Join(azureHNNFVITypes, ", ") + "."
		}
	}
	if _, present := props["resourceElementTemplates"]; !present {
		return ""
	}
	templates, msg := azureHNNamedItems(props["resourceElementTemplates"], "properties.resourceElementTemplates")
	if msg != "" {
		return msg
	}
	for i, ret := range templates {
		path := "properties.resourceElementTemplates[" + strconv.Itoa(i) + "]."
		if !slices.Contains([]string{"ArmResourceDefinition", "NetworkFunctionDefinition"}, azureARMString(ret["type"])) {
			return path + "type must be ArmResourceDefinition or NetworkFunctionDefinition."
		}
		config := azureARMObject(ret["configuration"])
		if msg := azureLabEnum(config, path+"configuration.", "templateType", []string{"ArmTemplate"}); msg != "" {
			return msg
		}
		if raw, present := config["parameterValues"]; present {
			if _, msg := azureHNJSONObject(raw, path+"configuration.parameterValues"); msg != "" {
				return msg
			}
		}
		artifact := azureARMObject(config["artifactProfile"])
		if azureARMString(artifact["artifactName"]) == "" || azureARMString(azureARMObject(artifact["artifactStoreReference"])["id"]) == "" {
			return path + "configuration.artifactProfile needs artifactName and artifactStoreReference.id."
		}
	}
	return ""
}

// azureHNNSDVReconcile checks that the schemas and artifact stores a
// design references exist.
func azureHNNSDVReconcile(s *Server, rec *azureARMResource) string {
	refs := azureARMObject(rec.Properties["configurationGroupSchemaReferences"])
	for _, key := range slices.Sorted(maps.Keys(refs)) {
		schema, msg := s.azureHNRefLocked(azureARMString(azureARMObject(refs[key])["id"]), azureHNCGSType, "configuration group schema")
		if msg == "" {
			msg = azureHNSamePublisher(rec, schema, "configuration group schema")
		}
		if msg != "" {
			return msg
		}
	}
	for _, item := range azureHNList(rec.Properties["resourceElementTemplates"]) {
		artifact := azureARMObject(azureARMObject(azureARMObject(item)["configuration"])["artifactProfile"])
		store, msg := s.azureHNRefLocked(azureARMString(azureARMObject(artifact["artifactStoreReference"])["id"]), azureHNStore, "artifact store")
		if msg == "" {
			msg = azureHNSamePublisher(rec, store, "artifact store")
		}
		if msg != "" {
			return msg
		}
	}
	return ""
}

// azureHNPublisherOf returns the publisher ID a publisher child's ID
// starts with.
func azureHNPublisherOf(id string) string {
	parts := strings.Split(id, "/")
	return strings.Join(parts[:min(len(parts), 9)], "/")
}
