package server

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Azure Operator Service Manager (Microsoft.HybridNetwork 2023-09-01):
// publishers with their artifact stores and manifests, network function
// definitions, network service designs and configuration group schemas;
// and the operator's sites, site network services, configuration group
// values and network functions. No artifact is stored and nothing deploys.

const azureHNNamespace = "Microsoft.HybridNetwork"

var (
	azureHNNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)
	// azureHNSemVer is Learn's SemVer 2.0.0 pattern for version names.
	azureHNSemVer  = regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`)
	azureHNNoSpace = regexp.MustCompile(`^\S+$`)
)

const (
	azureHNNFDVType = "publishers/networkFunctionDefinitionGroups/networkFunctionDefinitionVersions"
	azureHNNSDVType = "publishers/networkServiceDesignGroups/networkServiceDesignVersions"
	azureHNCGSType  = "publishers/configurationGroupSchemas"
	azureHNStore    = "publishers/artifactStores"
)

func init() {
	ns := azureHNNamespace
	tags := azureWorkloadsTagsPatch(false)
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "publishers",
		ValidateName: azureHNName("publisher", azureHNNamePattern),
		Validate:     azureHNPublisherValidate,
		PatchBody:    tags,
		Reconcile:    azureHNPublisherReconcile,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: azureHNStore,
		ValidateName: azureHNName("artifact store", azureHNNamePattern),
		Validate:     azureHNStoreValidate,
		PatchBody:    tags,
		Reconcile:    azureHNStoreReconcile,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: azureHNStore + "/artifactManifests",
		ValidateName:    azureHNName("artifact manifest", azureHNNoSpace),
		Validate:        azureHNManifestValidate,
		PatchBody:       tags,
		StateProperties: map[string]any{"artifactManifestState": "Uploading"},
		Reconcile:       azureHNManifestReconcile,
		ServerActions: map[string]func(*Server, *azureARMResource, map[string]any) (int, any){
			"updatestate":    azureHNUpdateState("artifactManifestState", "Uploaded"),
			"listcredential": azureHNListCredential,
		},
	})
	for _, group := range []string{"networkFunctionDefinitionGroups", "networkServiceDesignGroups"} {
		registerAzureARMResourceType(azureARMResourceType{
			Namespace: ns, Type: "publishers/" + group,
			ValidateName: azureHNName("group", azureHNNamePattern),
			Validate:     azureHNDescribed,
			PatchBody:    tags,
		})
	}
	versioned := map[string]struct {
		validate  func(map[string]any) string
		reconcile func(*Server, *azureARMResource) string
		name      func(string) string
	}{
		azureHNNFDVType: {azureHNNFDVValidate, azureHNNFDVReconcile, azureHNName("network function definition version", azureHNSemVer)},
		azureHNNSDVType: {azureHNNSDVValidate, azureHNNSDVReconcile, azureHNName("network service design version", azureHNSemVer)},
		azureHNCGSType:  {azureHNCGSValidate, nil, azureHNName("configuration group schema", azureHNNamePattern)},
	}
	for typ, v := range versioned {
		registerAzureARMResourceType(azureARMResourceType{
			Namespace: ns, Type: typ,
			ValidateName:    v.name,
			Validate:        v.validate,
			PatchBody:       tags,
			StateProperties: map[string]any{"versionState": "Preview"},
			Reconcile:       azureHNVersionReconcile(v.reconcile),
			ServerActions: map[string]func(*Server, *azureARMResource, map[string]any) (int, any){
				"updatestate": azureHNUpdateState("versionState", "Active", "Deprecated"),
			},
		})
	}
	azureHNRegisterOperator(tags)
}

// azureHNName checks a resource name against Learn's pattern and its
// 64-character limit.
func azureHNName(what string, pattern *regexp.Regexp) func(string) string {
	return func(name string) string {
		if len(name) > 64 || !pattern.MatchString(name) {
			return "The " + what + " name '" + name + "' is invalid: it must match " + pattern.String() + " and be at most 64 characters."
		}
		return ""
	}
}

func azureHNProperties(payload map[string]any) (map[string]any, string) {
	if msg := azureDataBoxObjects(payload, "", "properties", "tags"); msg != "" {
		return nil, msg
	}
	return azureARMObject(payload["properties"]), ""
}

func azureHNDescribed(payload map[string]any) string {
	props, msg := azureHNProperties(payload)
	if msg != "" {
		return msg
	}
	if raw, present := props["description"]; present {
		if _, ok := raw.(string); !ok {
			return "properties.description must be a string."
		}
	}
	return ""
}

// azureHNPublisherValidate takes Learn's scope enum (Private) and the
// Public scope its samples use.
func azureHNPublisherValidate(payload map[string]any) string {
	props, msg := azureHNProperties(payload)
	if msg != "" {
		return msg
	}
	if msg := azureLabEnum(props, "properties.", "scope", []string{"Private", "Public"}); msg != "" {
		return msg
	}
	return azureNewRelicIdentity(payload)
}

func azureHNLocation(loc string) string { return strings.ToLower(strings.ReplaceAll(loc, " ", "")) }

// azureHNPublisherReconcile keeps publisher names unique in a region and
// the scope fixed once created.
func azureHNPublisherReconcile(s *Server, existing, rec *azureARMResource) (map[string]any, string) {
	for _, other := range s.azureARMResourcesMatchingLocked(func(r *azureARMResource) bool {
		return strings.EqualFold(r.Type, azureHNNamespace+"/publishers") && strings.EqualFold(r.Name, rec.Name) &&
			!strings.EqualFold(r.ID, rec.ID) && azureHNLocation(r.Location) == azureHNLocation(rec.Location)
	}) {
		return nil, "A publisher named " + rec.Name + " already exists in " + rec.Location + " (" + other.ID + "); publisher names are unique in a region."
	}
	scope := rec.Properties["scope"]
	if existing != nil {
		if scope != nil && scope != existing.Properties["scope"] {
			return nil, "properties.scope can't be changed after creation."
		}
		return map[string]any{"scope": existing.Properties["scope"]}, ""
	}
	if scope == nil {
		return map[string]any{"scope": "Private"}, ""
	}
	return nil, ""
}

func azureHNStoreValidate(payload map[string]any) string {
	props, msg := azureHNProperties(payload)
	if msg != "" {
		return msg
	}
	if msg := azureLabEnum(props, "properties.", "storeType", []string{"AzureContainerRegistry", "AzureStorageAccount"}); msg != "" {
		return msg
	}
	if msg := azureLabEnum(props, "properties.", "replicationStrategy", []string{"SingleReplication"}); msg != "" {
		return msg
	}
	if msg := azureDataBoxObjects(props, "properties.", "managedResourceGroupConfiguration"); msg != "" {
		return msg
	}
	mrg := azureARMObject(props["managedResourceGroupConfiguration"])
	for _, key := range []string{"name", "location"} {
		if v, present := mrg[key]; present && v != nil {
			if _, ok := v.(string); !ok {
				return "properties.managedResourceGroupConfiguration." + key + " must be a string."
			}
		}
	}
	if name := azureARMString(mrg["name"]); name != "" && !azureHNResourceGroupName.MatchString(name) {
		return "properties.managedResourceGroupConfiguration.name must be 1-90 letters, digits, underscores, hyphens, periods or parentheses, not ending in a period."
	}
	return ""
}

// azureHNResourceGroupName is Azure's resource group naming rule.
var azureHNResourceGroupName = regexp.MustCompile(`^[-\w._()]{0,89}[-\w_()]$`)

// azureHNStoreReconcile creates the store's backing registry or storage
// account in its managed resource group; neither can change afterwards.
func azureHNStoreReconcile(_ *Server, existing, rec *azureARMResource) (map[string]any, string) {
	props := rec.Properties
	if existing != nil {
		old := existing.Properties
		for _, key := range []string{"storeType", "replicationStrategy"} {
			if props[key] != nil && !reflect.DeepEqual(props[key], old[key]) {
				return nil, "properties." + key + " can't be changed after creation."
			}
		}
		// Only the fields sent are compared; omitted ones keep their defaults.
		oldMRG := azureARMObject(old["managedResourceGroupConfiguration"])
		for key, value := range azureARMObject(props["managedResourceGroupConfiguration"]) {
			if !reflect.DeepEqual(value, oldMRG[key]) {
				return nil, "properties.managedResourceGroupConfiguration can't be changed after creation."
			}
		}
		return map[string]any{"storeType": old["storeType"], "replicationStrategy": old["replicationStrategy"],
			"managedResourceGroupConfiguration": old["managedResourceGroupConfiguration"], "storageResourceId": old["storageResourceId"]}, ""
	}
	storeType := azureARMString(props["storeType"])
	if storeType == "" {
		storeType = "AzureContainerRegistry"
	}
	mrg := azureARMObject(props["managedResourceGroupConfiguration"])
	name := azureARMString(mrg["name"])
	if name == "" {
		name = "ArtifactStore-" + rec.Name + "-" + azureWorkloadsHash(rec.ID, 8)
	}
	location := azureARMString(mrg["location"])
	if location == "" {
		location = rec.Location
	}
	group := "/subscriptions/" + rec.SubscriptionID + "/resourceGroups/" + name + "/providers/"
	backing := group + "Microsoft.ContainerRegistry/registries/" + azureHNAlnum(rec.Name, 30) + azureWorkloadsHash(rec.ID, 10)
	if storeType == "AzureStorageAccount" {
		backing = group + "Microsoft.Storage/storageAccounts/" + strings.ToLower(azureHNAlnum(rec.Name, 14)) + azureWorkloadsHash(rec.ID, 10)
	}
	return map[string]any{
		"storeType": storeType, "replicationStrategy": "SingleReplication", "storageResourceId": backing,
		"managedResourceGroupConfiguration": map[string]any{"name": name, "location": location},
	}, ""
}

// azureHNAlnum keeps the letters and digits of name, at most n of them.
func azureHNAlnum(name string, n int) string {
	out := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, name)
	return out[:min(len(out), n)]
}

var azureHNArtifactTypes = []string{"OCIArtifact", "VhdImageFile", "ArmTemplate", "ImageFile"}

func azureHNManifestValidate(payload map[string]any) string {
	props, msg := azureHNProperties(payload)
	if msg != "" {
		return msg
	}
	raw, present := props["artifacts"]
	if !present {
		return ""
	}
	artifacts, ok := raw.([]any)
	if !ok {
		return "properties.artifacts must be an array."
	}
	seen := map[string]bool{}
	for i, item := range artifacts {
		artifact, ok := item.(map[string]any)
		path := "properties.artifacts[" + strconv.Itoa(i) + "]."
		if !ok {
			return "properties.artifacts items must be objects."
		}
		for _, key := range []string{"artifactName", "artifactVersion"} {
			if strings.TrimSpace(azureARMString(artifact[key])) == "" {
				return path + key + " is required."
			}
		}
		if !slices.Contains(azureHNArtifactTypes, azureARMString(artifact["artifactType"])) {
			return path + "artifactType must be one of " + strings.Join(azureHNArtifactTypes, ", ") + "."
		}
		key := strings.ToLower(azureARMString(artifact["artifactName"]) + ":" + azureARMString(artifact["artifactVersion"]))
		if seen[key] {
			return "properties.artifacts lists " + azureARMString(artifact["artifactName"]) + " " + azureARMString(artifact["artifactVersion"]) + " twice."
		}
		seen[key] = true
	}
	return ""
}

// azureHNManifestReconcile keeps an uploaded manifest's artifacts, which
// are immutable, and stores only artifacts its store can hold: VHDs in a
// storage account, everything else in a container registry.
func azureHNManifestReconcile(s *Server, existing, rec *azureARMResource) (map[string]any, string) {
	if existing != nil && existing.Properties["artifactManifestState"] == "Uploaded" &&
		!reflect.DeepEqual(rec.Properties["artifacts"], existing.Properties["artifacts"]) {
		return nil, "The artifact manifest " + rec.Name + " is Uploaded; its artifacts are immutable."
	}
	store := s.azureARMResources[azureARMKey(azureWorkloadsParentID(rec.ID))]
	if store == nil {
		return nil, "The artifact store for " + rec.Name + " was not found."
	}
	storage := store.Properties["storeType"] == "AzureStorageAccount"
	for _, item := range azureHNList(rec.Properties["artifacts"]) {
		artifact := azureARMObject(item)
		if (artifact["artifactType"] == "VhdImageFile") != storage {
			return nil, "The " + azureARMString(store.Properties["storeType"]) + " store " + store.Name + " can't hold the " + azureARMString(artifact["artifactType"]) + " artifact " + azureARMString(artifact["artifactName"]) + "."
		}
	}
	return nil, ""
}

func azureHNList(v any) []any {
	items, _ := v.([]any)
	return items
}

// azureHNVersionReconcile makes an Active or Deprecated version, or
// schema, immutable, then runs the type's reference checks.
func azureHNVersionReconcile(refs func(*Server, *azureARMResource) string) func(*Server, *azureARMResource, *azureARMResource) (map[string]any, string) {
	return func(s *Server, existing, rec *azureARMResource) (map[string]any, string) {
		if existing != nil && reflect.DeepEqual(rec.Properties, existing.Properties) {
			return nil, "" // unchanged, such as a tags update
		}
		if existing != nil && existing.Properties["versionState"] != "Preview" {
			return nil, rec.Name + " is " + azureARMString(existing.Properties["versionState"]) + " and immutable; only Preview versions can change."
		}
		if refs != nil {
			return nil, refs(s, rec)
		}
		return nil, ""
	}
}

// azureHNUpdateState sets a state property to one of the states clients
// may request, answering 200 with the new state.
func azureHNUpdateState(key string, allowed ...string) func(*Server, *azureARMResource, map[string]any) (int, any) {
	return func(s *Server, rec *azureARMResource, payload map[string]any) (int, any) {
		state := azureARMString(payload[key])
		if !slices.Contains(allowed, state) {
			return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", key+" must be "+strings.Join(allowed, " or ")+"; other states are set by the service.")
		}
		s.providerStorageMu.Lock()
		defer s.providerStorageMu.Unlock()
		stored := s.azureARMResources[azureARMKey(rec.ID)]
		if stored == nil {
			return http.StatusNotFound, azureARMErrorBody("ResourceNotFound", "The Resource '"+rec.ID+"' was not found.")
		}
		updated := *stored
		updated.UpdatedAt = time.Now().UTC()
		updated.Properties = azureARMMerge(stored.Properties, map[string]any{key: state})
		s.azureARMResources[azureARMKey(rec.ID)] = &updated
		return http.StatusOK, map[string]any{key: state}
	}
}

// azureHNListCredential returns a fresh, short-lived credential for
// pushing a manifest's artifacts: a scoped registry token or container
// SAS URIs. Credentials are generated per call and never stored.
func azureHNListCredential(s *Server, rec *azureARMResource, _ map[string]any) (int, any) {
	s.providerStorageMu.Lock()
	store := s.azureARMResources[azureARMKey(azureWorkloadsParentID(rec.ID))]
	manifest := s.azureARMResources[azureARMKey(rec.ID)]
	s.providerStorageMu.Unlock()
	if store == nil || manifest == nil {
		return http.StatusNotFound, azureARMErrorBody("ResourceNotFound", "The Resource '"+rec.ID+"' was not found.")
	}
	expiry := time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339)
	backing := azureARMString(store.Properties["storageResourceId"])
	account := backing[strings.LastIndex(backing, "/")+1:]
	artifacts := azureHNList(manifest.Properties["artifacts"])
	if store.Properties["storeType"] == "AzureStorageAccount" {
		containers := make([]any, 0, len(artifacts))
		for _, item := range artifacts {
			artifact := azureARMObject(item)
			// Name prefix plus a hash of name:version, so distinct artifacts
			// never share a container and the name starts alphanumeric.
			name, version := azureARMString(artifact["artifactName"]), azureARMString(artifact["artifactVersion"])
			container := strings.ToLower(azureHNAlnum(name, 40))
			if container == "" {
				container = "artifact"
			}
			container += "-" + azureWorkloadsHash(name+":"+version, 10)
			containers = append(containers, map[string]any{
				"containerName":   container,
				"containerSasUri": "https://" + account + ".blob.core.windows.net/" + container + "?sv=2021-08-06&sp=racwl&se=" + expiry + "&sig=" + azureHNSecret(),
			})
		}
		return http.StatusOK, map[string]any{"credentialType": "AzureStorageAccountToken", "storageAccountId": backing, "expiry": expiry, "containerCredentials": containers}
	}
	repositories := make([]any, 0, len(artifacts))
	for _, item := range artifacts {
		repositories = append(repositories, azureARMString(azureARMObject(item)["artifactName"]))
	}
	return http.StatusOK, map[string]any{
		"credentialType": "AzureContainerRegistryScopedToken", "username": strings.ToLower(azureHNAlnum(manifest.Name, 40)) + "-token",
		"acrToken": azureHNSecret(), "expiry": expiry, "acrServerUrl": strings.ToLower(account) + ".azurecr.io", "repositories": repositories,
	}
}

func azureHNSecret() string {
	b := make([]byte, 24)
	_, _ = rand.Read(b) // crypto/rand.Read never returns an error.
	return hex.EncodeToString(b)
}
