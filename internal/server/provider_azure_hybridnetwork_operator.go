package server

import (
	"encoding/json"
	"maps"
	"net/http"
	"slices"
	"strconv"
	"strings"
)

// The operator's resources reference the publisher's: a configuration
// group value its schema, a network function its definition version, a
// site network service its design version, site and values. A reference
// with idType Secret hides its ID, so the ID is write-only and copied back
// for Open references.

const (
	azureHNCGSRef  = "configurationGroupSchemaResourceReference"
	azureHNNFDVRef = "networkFunctionDefinitionVersionResourceReference"
	azureHNNSDVRef = "networkServiceDesignVersionResourceReference"
)

func azureHNRegisterOperator(tags func(map[string]any) (map[string]any, map[string]any, string)) {
	ns := azureHNNamespace
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "configurationGroupValues",
		ValidateName:        azureHNName("configuration group value", azureHNNamePattern),
		Validate:            azureHNCGVValidate,
		WriteOnlyProperties: []string{"secretConfigurationValue", azureHNCGSRef + ".id"},
		PatchBody:           tags,
		Reconcile:           azureHNKeepRefID(azureHNCGSRef),
		Linked:              azureHNCGVLinked,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "networkFunctions",
		ValidateName:        azureHNName("network function", azureHNNoSpace),
		Validate:            azureHNNFValidate,
		WriteOnlyProperties: []string{"secretDeploymentValues", azureHNNFDVRef + ".id"},
		PatchBody:           tags,
		Reconcile:           azureHNKeepRefID(azureHNNFDVRef),
		Linked:              azureHNNFLinked,
		Actions:             map[string]azureARMAction{"executerequest": azureHNExecuteRequest},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "sites",
		ValidateName: azureHNName("site", azureHNNamePattern),
		Validate:     azureHNSiteValidate,
		PatchBody:    tags,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "siteNetworkServices",
		ValidateName: azureHNName("site network service", azureHNNamePattern),
		Validate:     azureHNSNSValidate,
		ReadOnlySKU: func(sku map[string]any) map[string]any {
			if sku["name"] == nil {
				return nil
			}
			return map[string]any{"tier": sku["name"]}
		},
		WriteOnlyProperties: []string{azureHNNSDVRef + ".id"},
		PatchBody:           tags,
		Reconcile:           azureHNKeepRefID(azureHNNSDVRef),
		Linked:              azureHNSNSLinked,
	})
}

// azureHNRefShape checks a deployment resource reference: an ID and an
// idType of Open (the default) or Secret.
func azureHNRefShape(props map[string]any, field string) string {
	ref, ok := props[field].(map[string]any)
	if !ok || azureARMString(ref["id"]) == "" {
		return "properties." + field + ".id is required."
	}
	return azureLabEnum(ref, "properties."+field+".", "idType", []string{"Open", "Secret"})
}

// azureHNValues checks that the Open or Secret values field, whichever
// configurationType selects, holds a JSON object, and the other is absent.
func azureHNValues(props map[string]any, open, secret string, required bool) string {
	field, other := open, secret
	switch props["configurationType"] {
	case "Open":
	case "Secret":
		field, other = secret, open
	default:
		return "properties.configurationType must be Open or Secret."
	}
	if _, present := props[other]; present {
		return "properties." + other + " isn't allowed with configurationType " + azureARMString(props["configurationType"]) + "; use " + field + "."
	}
	raw, present := props[field]
	if !present && !required {
		return ""
	}
	_, msg := azureHNJSONObject(raw, "properties."+field)
	return msg
}

// azureHNValuesOf returns the values a record's configurationType selects.
func azureHNValuesOf(props map[string]any, open, secret string) map[string]any {
	field := open
	if props["configurationType"] == "Secret" {
		field = secret
	}
	values, _ := azureHNJSONObject(props[field], field)
	return values
}

// azureHNKeepRefID copies the stored ID of an Open reference into a write
// whose own copy was dropped as write-only: a PATCH keeps it, and a PUT's
// Linked hook replaces it with the ID sent.
func azureHNKeepRefID(field string) func(*Server, *azureARMResource, *azureARMResource) (map[string]any, string) {
	return func(_ *Server, existing, rec *azureARMResource) (map[string]any, string) {
		if existing == nil {
			return nil, ""
		}
		id := azureARMObject(existing.Properties[field])["id"]
		if id == nil || azureARMObject(rec.Properties[field])["idType"] == "Secret" {
			return nil, ""
		}
		return map[string]any{field: map[string]any{"id": id}}, ""
	}
}

// azureHNResolveLocked resolves a record's reference to a publisher
// resource and returns it with the properties the record reports about
// it. Caller holds providerStorageMu.
func (s *Server) azureHNResolveLocked(rec *azureARMResource, field, typ, what string) (*azureARMResource, map[string]any, string) {
	ref := azureARMObject(rec.Properties[field])
	target, msg := s.azureHNRefLocked(azureARMString(ref["id"]), typ, what)
	if msg != "" {
		return nil, nil, msg
	}
	out := map[string]any{"idType": "Secret"}
	if ref["idType"] != "Secret" {
		out = map[string]any{"idType": "Open", "id": ref["id"]}
	}
	publisher := s.azureARMResources[azureARMKey(azureHNPublisherOf(target.ID))]
	if publisher == nil {
		return nil, nil, "The publisher of " + what + " '" + target.ID + "' was not found."
	}
	return target, map[string]any{field: out, "publisherName": publisher.Name, "publisherScope": publisher.Properties["scope"]}, ""
}

// azureHNDeployable applies Learn's version states: a Deprecated version
// can't be deployed, and a Preview one only from its own subscription.
func azureHNDeployable(version, rec *azureARMResource, what string) string {
	switch version.Properties["versionState"] {
	case "Deprecated":
		return "The " + what + " " + version.Name + " is Deprecated and can't be deployed."
	case "Preview":
		if !strings.EqualFold(version.SubscriptionID, rec.SubscriptionID) {
			return "The " + what + " " + version.Name + " is in Preview, so only its own subscription can deploy it; make it Active to deploy it from others."
		}
	}
	return ""
}

func azureHNCGVValidate(payload map[string]any) string {
	props, msg := azureHNProperties(payload)
	if msg != "" {
		return msg
	}
	if msg := azureHNRefShape(props, azureHNCGSRef); msg != "" {
		return msg
	}
	return azureHNValues(props, "configurationValue", "secretConfigurationValue", true)
}

// azureHNCGVLinked checks the value, secret or not, against its schema.
func azureHNCGVLinked(s *Server, rec *azureARMResource) (map[string]any, []*azureARMResource, string) {
	schema, out, msg := s.azureHNResolveLocked(rec, azureHNCGSRef, azureHNCGSType, "configuration group schema")
	if msg != "" {
		return nil, nil, msg
	}
	if msg := azureHNDeployable(schema, rec, "configuration group schema"); msg != "" {
		return nil, nil, msg
	}
	values := azureHNValuesOf(rec.Properties, "configurationValue", "secretConfigurationValue")
	if msg := azureHNCheckValues(azureARMString(schema.Properties["schemaDefinition"]), values, "The configuration value"); msg != "" {
		return nil, nil, msg
	}
	out["configurationGroupSchemaName"] = schema.Name
	out["configurationGroupSchemaOfferingLocation"] = schema.Location
	return out, nil, ""
}

func azureHNNFValidate(payload map[string]any) string {
	props, msg := azureHNProperties(payload)
	if msg != "" {
		return msg
	}
	if msg := azureNewRelicIdentity(payload); msg != "" {
		return msg
	}
	if msg := azureHNRefShape(props, azureHNNFDVRef); msg != "" {
		return msg
	}
	if msg := azureHNValues(props, "deploymentValues", "secretDeploymentValues", false); msg != "" {
		return msg
	}
	nfvi := azureARMString(props["nfviType"])
	if !slices.Contains(azureHNNFVITypes[1:], nfvi) {
		return "properties.nfviType must be one of " + strings.Join(azureHNNFVITypes[1:], ", ") + "."
	}
	if nfvi != "AzureCore" && !strings.Contains(strings.ToLower(azureARMString(props["nfviId"])), "/providers/microsoft.extendedlocation/customlocations/") {
		return "properties.nfviId must be the custom location ID of the " + nfvi + " cluster."
	}
	if raw, present := props["allowSoftwareUpdate"]; present {
		if _, ok := raw.(bool); !ok {
			return "properties.allowSoftwareUpdate must be a boolean."
		}
	}
	if raw, present := props["roleOverrideValues"]; present {
		if _, ok := raw.([]any); !ok {
			return "properties.roleOverrideValues must be an array."
		}
		for i, item := range azureHNList(raw) {
			path := "properties.roleOverrideValues[" + strconv.Itoa(i) + "]"
			override, msg := azureHNJSONObject(item, path)
			if msg != "" {
				return msg
			}
			if azureARMString(override["name"]) == "" {
				return path + " must name the application it overrides."
			}
		}
	}
	return ""
}

// azureHNNFLinked deploys a network function from its definition version:
// on the version's NFVI type, with the values its deployParameters schema
// requires and overrides for its own applications. An Active version's
// artifacts must be in Uploaded (immutable) manifests.
func azureHNNFLinked(s *Server, rec *azureARMResource) (map[string]any, []*azureARMResource, string) {
	version, out, msg := s.azureHNResolveLocked(rec, azureHNNFDVRef, azureHNNFDVType, "network function definition version")
	if msg != "" {
		return nil, nil, msg
	}
	if msg := azureHNDeployable(version, rec, "network function definition version"); msg != "" {
		return nil, nil, msg
	}
	props := version.Properties
	template := azureARMObject(props["networkFunctionTemplate"])
	if nfvi := azureARMString(template["nfviType"]); nfvi != rec.Properties["nfviType"] {
		return nil, nil, "properties.nfviType must be " + nfvi + ", the NFVI type of " + version.Name + "."
	}
	values := azureHNValuesOf(rec.Properties, "deploymentValues", "secretDeploymentValues")
	if msg := azureHNCheckValues(azureARMString(props["deployParameters"]), values, "The deployment values"); msg != "" {
		return nil, nil, msg
	}
	apps := azureHNList(template["networkFunctionApplications"])
	for _, item := range azureHNList(rec.Properties["roleOverrideValues"]) {
		override, _ := azureHNJSONObject(item, "")
		name := azureARMString(override["name"])
		if !slices.ContainsFunc(apps, func(app any) bool { return strings.EqualFold(azureARMString(azureARMObject(app)["name"]), name) }) {
			return nil, nil, "properties.roleOverrideValues overrides " + name + ", which isn't an application of " + version.Name + "."
		}
	}
	if props["versionState"] == "Active" {
		if msg := s.azureHNUploadedLocked(apps); msg != "" {
			return nil, nil, msg
		}
	}
	parts := strings.Split(version.ID, "/")
	out["networkFunctionDefinitionGroupName"] = parts[len(parts)-3]
	out["networkFunctionDefinitionVersion"] = version.Name
	out["networkFunctionDefinitionOfferingLocation"] = version.Location
	return out, nil, ""
}

// azureHNUploadedLocked checks that each application's artifact is listed
// by an Uploaded manifest in its artifact store. Caller holds
// providerStorageMu.
func (s *Server) azureHNUploadedLocked(apps []any) string {
	for _, item := range apps {
		app := azureARMObject(item)
		artifact := azureARMObject(app["artifactProfile"])
		store := azureARMString(azureARMObject(artifact["artifactStore"])["id"])
		kind := azureARMString(app["artifactType"])
		var name, version string
		for _, profile := range azureHNNFDVApps {
			if p, ok := profile[kind]; ok {
				name = azureARMString(azureARMObject(artifact[p[0]])[p[1]])
				// templateName → templateVersion etc.; a helm package names
				// a version range, which isn't matched.
				version = azureARMString(azureARMObject(artifact[p[0]])[strings.TrimSuffix(p[1], "Name")+"Version"])
			}
		}
		manifestType := map[string]string{"HelmPackage": "OCIArtifact"}[kind]
		if manifestType == "" {
			manifestType = kind
		}
		prefix := azureARMKey(store) + "/artifactmanifests/"
		uploaded := s.azureARMResourcesMatchingLocked(func(r *azureARMResource) bool {
			return strings.HasPrefix(azureARMKey(r.ID), prefix) && r.Properties["artifactManifestState"] == "Uploaded" &&
				slices.ContainsFunc(azureHNList(r.Properties["artifacts"]), func(a any) bool {
					got := azureARMObject(a)
					return strings.EqualFold(azureARMString(got["artifactName"]), name) && got["artifactType"] == manifestType &&
						(version == "" || got["artifactVersion"] == version)
				})
		})
		if len(uploaded) == 0 {
			return "The definition version is Active, so its artifact " + name + " must be in an Uploaded artifact manifest; mark the manifest Uploaded first."
		}
	}
	return ""
}

// azureHNExecuteRequest takes a request for a network function's service
// endpoint; no service runs, so it answers 200 with no body.
func azureHNExecuteRequest(_ *azureARMResource, payload map[string]any) (int, any) {
	bad := func(msg string) (int, any) {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", msg)
	}
	if strings.TrimSpace(azureARMString(payload["serviceEndpoint"])) == "" {
		return bad("serviceEndpoint is required.")
	}
	meta, ok := payload["requestMetadata"].(map[string]any)
	if !ok {
		return bad("requestMetadata is required.")
	}
	if path := azureARMString(meta["relativePath"]); !strings.HasPrefix(path, "/") || strings.HasPrefix(path, "//") {
		return bad("requestMetadata.relativePath must be a path relative to the service endpoint, starting with /.")
	}
	if !slices.Contains([]string{"Post", "Put", "Get", "Patch", "Delete"}, azureARMString(meta["httpMethod"])) {
		return bad("requestMetadata.httpMethod must be one of Post, Put, Get, Patch, Delete.")
	}
	if raw, present := meta["serializedBody"]; present && raw != "" {
		if body, ok := raw.(string); !ok || !json.Valid([]byte(body)) {
			return bad("requestMetadata.serializedBody must be a JSON string.")
		}
	}
	return http.StatusOK, nil
}

func azureHNSiteValidate(payload map[string]any) string {
	props, msg := azureHNProperties(payload)
	if msg != "" {
		return msg
	}
	raw, present := props["nfvis"]
	if !present {
		return ""
	}
	nfvis, ok := raw.([]any)
	if !ok {
		return "properties.nfvis must be an array."
	}
	names := map[string]bool{}
	for i, item := range nfvis {
		nfvi := azureARMObject(item)
		path := "properties.nfvis[" + strconv.Itoa(i) + "]."
		name := azureARMString(nfvi["name"])
		if name == "" {
			return path + "name is required."
		}
		if names[strings.ToLower(name)] {
			return "properties.nfvis has two NFVIs named " + name + "."
		}
		names[strings.ToLower(name)] = true
		switch nfviType := azureARMString(nfvi["nfviType"]); nfviType {
		case "AzureCore":
			if azureARMString(nfvi["location"]) == "" {
				return path + "location is required for an AzureCore NFVI."
			}
		case "AzureArcKubernetes", "AzureOperatorNexus":
			if !strings.Contains(strings.ToLower(azureARMString(azureARMObject(nfvi["customLocationReference"])["id"])), "/providers/microsoft.extendedlocation/customlocations/") {
				return path + "customLocationReference.id must be a custom location ID for an " + nfviType + " NFVI."
			}
		case "Unknown":
		default:
			return path + "nfviType must be one of " + strings.Join(azureHNNFVITypes, ", ") + "."
		}
	}
	return ""
}

func azureHNSNSValidate(payload map[string]any) string {
	props, msg := azureHNProperties(payload)
	if msg != "" {
		return msg
	}
	if msg := azureNewRelicIdentity(payload); msg != "" {
		return msg
	}
	if msg := azureDataBoxObjects(payload, "", "sku"); msg != "" {
		return msg
	}
	if msg := azureLabEnum(azureARMObject(payload["sku"]), "sku.", "name", []string{"Basic", "Standard"}); msg != "" {
		return msg
	}
	if azureARMString(azureARMObject(props["siteReference"])["id"]) == "" {
		return "properties.siteReference.id is required."
	}
	if msg := azureHNRefShape(props, azureHNNSDVRef); msg != "" {
		return msg
	}
	if msg := azureDataBoxObjects(props, "properties.", "desiredStateConfigurationGroupValueReferences", "managedResourceGroupConfiguration"); msg != "" {
		return msg
	}
	for key, raw := range azureARMObject(props["desiredStateConfigurationGroupValueReferences"]) {
		if azureARMString(azureARMObject(raw)["id"]) == "" {
			return "properties.desiredStateConfigurationGroupValueReferences." + key + ".id is required."
		}
	}
	return ""
}

// azureHNSNSLinked deploys a network service design to a site in the
// service's region: one configuration group value per schema the design
// names, each made from that schema, and a site NFVI for each NFVI the
// design deploys to. An Active design needs Active schemas.
func azureHNSNSLinked(s *Server, rec *azureARMResource) (map[string]any, []*azureARMResource, string) {
	props := rec.Properties
	site, msg := s.azureHNRefLocked(azureARMString(azureARMObject(props["siteReference"])["id"]), "sites", "site")
	if msg != "" {
		return nil, nil, msg
	}
	if azureHNLocation(site.Location) != azureHNLocation(rec.Location) {
		return nil, nil, "The site " + site.Name + " is in " + site.Location + "; a site network service can only use a site in its own region, " + rec.Location + "."
	}
	design, out, msg := s.azureHNResolveLocked(rec, azureHNNSDVRef, azureHNNSDVType, "network service design version")
	if msg != "" {
		return nil, nil, msg
	}
	if msg := azureHNDeployable(design, rec, "network service design version"); msg != "" {
		return nil, nil, msg
	}
	if msg := s.azureHNSNSValuesLocked(design, azureARMObject(props["desiredStateConfigurationGroupValueReferences"])); msg != "" {
		return nil, nil, msg
	}
	if msg := azureHNSiteNFVIs(design, site); msg != "" {
		return nil, nil, msg
	}
	parts := strings.Split(design.ID, "/")
	out["networkServiceDesignGroupName"] = parts[len(parts)-3]
	out["networkServiceDesignVersionName"] = design.Name
	out["networkServiceDesignVersionOfferingLocation"] = design.Location
	out["lastStateNetworkServiceDesignVersionName"] = design.Name
	if desired := props["desiredStateConfigurationGroupValueReferences"]; desired != nil {
		out["lastStateConfigurationGroupValueReferences"] = desired
	}
	if props["managedResourceGroupConfiguration"] == nil {
		out["managedResourceGroupConfiguration"] = map[string]any{"name": rec.Name + "-mrg-" + azureWorkloadsHash(rec.ID, 8), "location": rec.Location}
	}
	return out, nil, ""
}

// azureHNSNSValuesLocked matches the desired values to the design's
// schemas. Caller holds providerStorageMu.
func (s *Server) azureHNSNSValuesLocked(design *azureARMResource, desired map[string]any) string {
	schemas := azureARMObject(design.Properties["configurationGroupSchemaReferences"])
	for _, key := range slices.Sorted(maps.Keys(desired)) {
		if _, ok := schemas[key]; !ok {
			return "properties.desiredStateConfigurationGroupValueReferences." + key + " isn't a configuration group of " + design.Name + "."
		}
	}
	for _, key := range slices.Sorted(maps.Keys(schemas)) {
		ref, ok := desired[key]
		if !ok {
			return "properties.desiredStateConfigurationGroupValueReferences needs a value for " + key + ", a configuration group of " + design.Name + "."
		}
		schema, msg := s.azureHNRefLocked(azureARMString(azureARMObject(schemas[key])["id"]), azureHNCGSType, "configuration group schema")
		if msg != "" {
			return msg
		}
		if design.Properties["versionState"] == "Active" && schema.Properties["versionState"] != "Active" {
			return "The design version is Active, so its configuration group schema " + schema.Name + " must be Active too."
		}
		value, msg := s.azureHNRefLocked(azureARMString(azureARMObject(ref)["id"]), "configurationGroupValues", "configuration group value")
		if msg != "" {
			return msg
		}
		publisher := s.azureARMResources[azureARMKey(azureHNPublisherOf(schema.ID))]
		if !strings.EqualFold(azureARMString(value.Properties["configurationGroupSchemaName"]), schema.Name) ||
			publisher == nil || !strings.EqualFold(azureARMString(value.Properties["publisherName"]), publisher.Name) ||
			azureHNLocation(azureARMString(value.Properties["configurationGroupSchemaOfferingLocation"])) != azureHNLocation(schema.Location) {
			return "The configuration group value " + value.Name + " isn't made from " + schema.Name + ", the schema " + design.Name + " uses for " + key + "."
		}
	}
	return ""
}

// azureHNSiteNFVIs checks that the site has each NFVI the design deploys
// to, by name and type.
func azureHNSiteNFVIs(design, site *azureARMResource) string {
	nfvis := azureARMObject(design.Properties["nfvisFromSite"])
	for _, key := range slices.Sorted(maps.Keys(nfvis)) {
		want := azureARMObject(nfvis[key])
		if !slices.ContainsFunc(azureHNList(site.Properties["nfvis"]), func(item any) bool {
			nfvi := azureARMObject(item)
			return strings.EqualFold(azureARMString(nfvi["name"]), azureARMString(want["name"])) && nfvi["nfviType"] == want["type"]
		}) {
			return "The site " + site.Name + " has no " + azureARMString(want["type"]) + " NFVI named " + azureARMString(want["name"]) + ", which " + design.Name + " deploys to."
		}
	}
	return ""
}
