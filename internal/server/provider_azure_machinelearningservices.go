package server

import (
	"net/http"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Microsoft.MachineLearningServices workspaces with their computes,
// datastores, managed online endpoints and deployments, served by the shared
// ARM handler. No compute nodes start, no model is served at the scoring URI
// and no job runs.
func init() {
	const ns = "Microsoft.MachineLearningServices"
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "workspaces",
		Validate:           azureMLWorkspaceValidate,
		ReadOnlyProperties: azureMLWorkspaceProperties,
		Actions: map[string]azureARMAction{
			"listkeys":               azureMLWorkspaceKeys,
			"liststorageaccountkeys": azureMLWorkspaceKeys,
			"resynckeys":             func(*azureARMResource, map[string]any) (int, any) { return http.StatusOK, nil },
		},
	})
	instanceOnly := func(rec *azureARMResource, _ map[string]any) (int, any) {
		if rec.Properties["computeType"] != "ComputeInstance" {
			return http.StatusBadRequest, azureARMErrorBody("BadRequest", "Only compute instances can be started, stopped or restarted.")
		}
		return http.StatusAccepted, nil
	}
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "workspaces/computes",
		ValidateName:       azureMLComputeValidateName,
		Validate:           azureMLComputeValidate,
		ReadOnlyProperties: azureMLComputeProperties,
		Actions: map[string]azureARMAction{
			"start": instanceOnly, "stop": instanceOnly, "restart": instanceOnly,
			"listnodes": func(*azureARMResource, map[string]any) (int, any) {
				return http.StatusOK, map[string]any{"nodes": []any{}, "nextLink": nil}
			},
		},
		WriteOnlyProperties: []string{
			"properties.administratorAccount.password",
			"properties.administratorAccount.privateKeyData",
			"properties.userAccountCredentials.adminUserPassword",
			"properties.databricksAccessToken",
			"properties.sslConfiguration.key",
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "workspaces/datastores", Proxy: true,
		Validate:            azureMLDatastoreValidate,
		ReadOnlyProperties:  azureMLDatastoreProperties,
		WriteOnlyProperties: []string{"credentials.secrets"},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "workspaces/onlineEndpoints",
		ValidateName:       azureMLEndpointValidateName,
		Validate:           azureEnumRequired("authMode", "AMLToken", "Key", "AADToken"),
		ReadOnlyProperties: azureMLEndpointProperties,
		Actions: map[string]azureARMAction{
			"listkeys":       azureMLEndpointKeys,
			"regeneratekeys": azureMLEndpointRegenerateKeys,
			"token":          azureMLEndpointToken,
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "workspaces/onlineEndpoints/deployments",
		ValidateName: azureMLEndpointValidateName,
		Validate:     azureEnumRequired("endpointComputeType", "Managed", "Kubernetes", "AzureMLCompute"),
		Actions: map[string]azureARMAction{
			"getlogs": func(*azureARMResource, map[string]any) (int, any) {
				return http.StatusOK, map[string]any{"content": ""}
			},
		},
	})
}

// azureEnumRequired is azureEnumProperty for a property that must be set.
func azureEnumRequired(key string, values ...string) func(map[string]any) string {
	return func(payload map[string]any) string {
		if !slices.Contains(values, azureARMString(azureARMObject(payload["properties"])[key])) {
			return "properties." + key + " must be one of " + strings.Join(values, ", ") + "."
		}
		return ""
	}
}

var (
	azureMLComputeNamePattern  = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9-]{1,22}[a-zA-Z0-9]$`)
	azureMLEndpointNamePattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9-]{1,30}[a-zA-Z0-9]$`)
	azureMLAttachedComputes    = []string{"AKS", "Kubernetes", "VirtualMachine", "Databricks", "DataFactory", "HDInsight", "SynapseSpark", "DataLakeAnalytics"}
)

func azureMLComputeValidateName(name string) string {
	if !azureMLComputeNamePattern.MatchString(name) {
		return "Compute names must be 3-24 letters, digits or hyphens, start with a letter and end with a letter or digit."
	}
	return ""
}

func azureMLEndpointValidateName(name string) string {
	if !azureMLEndpointNamePattern.MatchString(name) {
		return "Endpoint and deployment names must be 3-32 letters, digits or hyphens, start with a letter and end with a letter or digit."
	}
	return ""
}

func azureMLWorkspaceValidate(payload map[string]any) string {
	kind := azureARMString(payload["kind"])
	if kind != "" && !slices.Contains([]string{"Default", "Hub", "Project", "FeatureStore"}, kind) {
		return "kind must be Default, Hub, Project or FeatureStore."
	}
	props := azureARMObject(payload["properties"])
	if kind == "Project" {
		if !strings.Contains(strings.ToLower(azureARMString(props["hubResourceId"])), "/providers/microsoft.machinelearningservices/workspaces/") {
			return "properties.hubResourceId must be a hub workspace ID."
		}
	} else {
		for _, dep := range [][2]string{{"storageAccount", "microsoft.storage/storageaccounts"}, {"keyVault", "microsoft.keyvault/vaults"}} {
			if !strings.Contains(strings.ToLower(azureARMString(props[dep[0]])), "/providers/"+dep[1]+"/") {
				return "properties." + dep[0] + " must be a " + dep[1] + " resource ID."
			}
		}
	}
	for _, dep := range [][2]string{{"applicationInsights", "microsoft.insights/components"}, {"containerRegistry", "microsoft.containerregistry/registries"}} {
		if id, present := props[dep[0]]; present && !strings.Contains(strings.ToLower(azureARMString(id)), "/providers/"+dep[1]+"/") {
			return "properties." + dep[0] + " must be a " + dep[1] + " resource ID."
		}
	}
	return azureEnumProperty("publicNetworkAccess", "Enabled", "Disabled")(payload)
}

// azureMLWorkspaceProperties reports the workspace's stable ID and its
// regional discovery and MLflow tracking endpoints.
func azureMLWorkspaceProperties(rec *azureARMResource) map[string]any {
	host := azureLocationHost(rec.Location) + ".api.azureml.ms"
	props := map[string]any{
		"workspaceId":       uuid.NewSHA1(uuid.NameSpaceURL, []byte(azureARMKey(rec.ID))).String(),
		"tenantId":          azureDefaultTenantID,
		"discoveryUrl":      "https://" + host + "/discovery",
		"mlFlowTrackingUri": "azureml://" + host + "/mlflow/v1.0" + rec.ID,
	}
	if _, present := rec.Properties["publicNetworkAccess"]; !present {
		props["publicNetworkAccess"] = "Enabled"
	}
	return props
}

// azureLocationHost turns a location such as "East US" into its host
// label, "eastus".
func azureLocationHost(location string) string {
	return strings.ToLower(strings.ReplaceAll(location, " ", ""))
}

func azureMLWorkspaceKeys(rec *azureARMResource, _ map[string]any) (int, any) {
	return http.StatusOK, map[string]any{
		"userStorageArmId":              rec.Properties["storageAccount"],
		"userStorageKey":                azureARMDerivedKey(rec.ID, "storage"),
		"appInsightsInstrumentationKey": uuid.NewSHA1(uuid.NameSpaceURL, []byte(azureARMKey(rec.ID)+"#insights")).String(),
	}
}

func azureMLComputeValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	computeType := azureARMString(props["computeType"])
	inner := azureARMObject(props["properties"])
	switch {
	case computeType == "AmlCompute":
		scale := azureARMObject(inner["scaleSettings"])
		maxNodes, ok := scale["maxNodeCount"].(float64)
		if !ok || maxNodes != float64(int(maxNodes)) || maxNodes < 0 {
			return "properties.properties.scaleSettings.maxNodeCount must be a non-negative integer."
		}
		if raw, present := scale["minNodeCount"]; present {
			if minNodes, ok := raw.(float64); !ok || minNodes != float64(int(minNodes)) || minNodes < 0 || minNodes > maxNodes {
				return "properties.properties.scaleSettings.minNodeCount must be an integer from 0 to maxNodeCount."
			}
		}
	case computeType == "ComputeInstance":
	case slices.Contains(azureMLAttachedComputes, computeType):
		if strings.TrimSpace(azureARMString(props["resourceId"])) == "" {
			return "properties.resourceId is required to attach a " + computeType + " compute."
		}
	default:
		return "properties.computeType must be AmlCompute, ComputeInstance or an attachable compute type."
	}
	return ""
}

// azureMLComputeProperties reports where the compute runs and whether it is
// attached, and gives created computes their default VM size; compute
// instances are always Running.
func azureMLComputeProperties(rec *azureARMResource) map[string]any {
	computeType := azureARMString(rec.Properties["computeType"])
	attached := slices.Contains(azureMLAttachedComputes, computeType)
	props := map[string]any{"isAttachedCompute": attached, "provisioningErrors": []any{}}
	if _, present := rec.Properties["computeLocation"]; !present {
		props["computeLocation"] = rec.Location
	}
	if attached {
		return props
	}
	inner := map[string]any{}
	for key, value := range azureARMObject(rec.Properties["properties"]) {
		inner[key] = value
	}
	if _, present := inner["vmSize"]; !present {
		inner["vmSize"] = "STANDARD_DS3_V2"
	}
	if computeType == "ComputeInstance" {
		inner["state"] = "Running"
	} else {
		inner["currentNodeCount"] = 0
	}
	props["properties"] = inner
	return props
}

var azureMLDatastoreRequired = map[string][]string{
	"AzureBlob":         {"accountName", "containerName"},
	"AzureFile":         {"accountName", "fileShareName"},
	"AzureDataLakeGen2": {"accountName", "filesystem"},
	"AzureDataLakeGen1": {"storeName"},
	"OneLake":           {"oneLakeWorkspaceName"},
}

func azureMLDatastoreValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	required, ok := azureMLDatastoreRequired[azureARMString(props["datastoreType"])]
	if !ok {
		return "properties.datastoreType must be AzureBlob, AzureFile, AzureDataLakeGen1, AzureDataLakeGen2 or OneLake."
	}
	for _, key := range required {
		if strings.TrimSpace(azureARMString(props[key])) == "" {
			return "properties." + key + " is required."
		}
	}
	credentials := azureARMString(azureARMObject(props["credentials"])["credentialsType"])
	if !slices.Contains([]string{"AccountKey", "Sas", "ServicePrincipal", "Certificate", "None"}, credentials) {
		return "properties.credentials.credentialsType must be AccountKey, Sas, ServicePrincipal, Certificate or None."
	}
	return ""
}

func azureMLDatastoreProperties(rec *azureARMResource) map[string]any {
	props := map[string]any{"isDefault": false}
	if datastoreType := azureARMString(rec.Properties["datastoreType"]); datastoreType == "AzureBlob" || datastoreType == "AzureFile" || datastoreType == "AzureDataLakeGen2" {
		for key, value := range map[string]any{"endpoint": "core.windows.net", "protocol": "https"} {
			if _, present := rec.Properties[key]; !present {
				props[key] = value
			}
		}
	}
	return props
}

// azureMLEndpointProperties reports the endpoint's regional scoring and
// Swagger URIs.
func azureMLEndpointProperties(rec *azureARMResource) map[string]any {
	base := "https://" + strings.ToLower(rec.Name) + "." + azureLocationHost(rec.Location) + ".inference.ml.azure.com"
	props := map[string]any{"scoringUri": base + "/score", "swaggerUri": base + "/swagger.json"}
	if _, present := rec.Properties["traffic"]; !present {
		props["traffic"] = map[string]any{}
	}
	return props
}

func azureMLEndpointKeys(rec *azureARMResource, _ map[string]any) (int, any) {
	if rec.Properties["authMode"] != "Key" {
		return http.StatusBadRequest, azureARMErrorBody("BadRequest", "Keys are available only for endpoints with authMode Key.")
	}
	return http.StatusOK, map[string]any{
		"primaryKey":   azureARMDerivedKey(rec.ID, "primary"),
		"secondaryKey": azureARMDerivedKey(rec.ID, "secondary"),
	}
}

// ponytail: regenerated keys stay the same derived values; keep a key
// generation counter in state if a client checks that keys change.
func azureMLEndpointRegenerateKeys(rec *azureARMResource, payload map[string]any) (int, any) {
	if rec.Properties["authMode"] != "Key" {
		return http.StatusBadRequest, azureARMErrorBody("BadRequest", "Keys are available only for endpoints with authMode Key.")
	}
	if keyType := azureARMString(payload["keyType"]); keyType != "Primary" && keyType != "Secondary" {
		return http.StatusBadRequest, azureARMErrorBody("BadRequest", "keyType must be Primary or Secondary.")
	}
	return http.StatusAccepted, nil
}

func azureMLEndpointToken(rec *azureARMResource, _ map[string]any) (int, any) {
	if rec.Properties["authMode"] != "AMLToken" {
		return http.StatusBadRequest, azureARMErrorBody("BadRequest", "Tokens are available only for endpoints with authMode AMLToken.")
	}
	now := time.Now().UTC()
	return http.StatusOK, map[string]any{
		"accessToken":         azureARMDerivedKey(rec.ID, "token"),
		"tokenType":           "Bearer",
		"expiryTimeUtc":       now.Add(time.Hour).Unix(),
		"refreshAfterTimeUtc": now.Add(30 * time.Minute).Unix(),
	}
}
