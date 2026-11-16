package server

import (
	"net/http"
	"path"
	"slices"
	"strings"

	"github.com/google/uuid"
)

// Microsoft.StorageMover storage movers with their agents, endpoints,
// projects and job definitions, served by the shared ARM handler. No agent
// runs and no data moves: starting a job returns the ID of a job run that
// is never stored.
func init() {
	const ns = "Microsoft.StorageMover"
	registerAzureARMResourceType(azureARMResourceType{Namespace: ns, Type: "storageMovers"})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "storageMovers/agents", Proxy: true,
		Validate:           azureStorageMoverAgentValidate,
		ReadOnlyProperties: azureStorageMoverAgentProperties,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "storageMovers/endpoints", Proxy: true,
		Validate: azureStorageMoverEndpointValidate,
	})
	registerAzureARMResourceType(azureARMResourceType{Namespace: ns, Type: "storageMovers/projects", Proxy: true})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "storageMovers/projects/jobDefinitions", Proxy: true,
		Validate:           azureStorageMoverJobDefinitionValidate,
		ReadOnlyProperties: azureStorageMoverJobDefinitionProperties,
		Actions: map[string]azureARMAction{
			"startjob": azureStorageMoverStartJob,
			"stopjob":  azureStorageMoverJobRun,
		},
	})
}

func azureStorageMoverAgentValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if !strings.Contains(strings.ToLower(azureARMString(props["arcResourceId"])), "/providers/microsoft.hybridcompute/machines/") {
		return "properties.arcResourceId must be an Arc-enabled machine ID."
	}
	if _, err := uuid.Parse(azureARMString(props["arcVmUuid"])); err != nil {
		return "properties.arcVmUuid must be a GUID."
	}
	return ""
}

// azureStorageMoverAgentProperties reports the agent online with a fixed
// small VM's resources.
func azureStorageMoverAgentProperties(rec *azureARMResource) map[string]any {
	return map[string]any{
		"agentStatus":      "Online",
		"agentVersion":     "2.0.358",
		"localIPAddress":   azureStorageCacheAddresses(rec, 1)[0],
		"memoryInMB":       8192,
		"numberOfCores":    4,
		"uptimeInSeconds":  0,
		"lastStatusUpdate": rec.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

// azureStorageMoverEndpointFields lists, per endpoint type, the properties
// it requires.
var azureStorageMoverEndpointFields = map[string][]string{
	"AzureStorageBlobContainer": {"storageAccountResourceId", "blobContainerName"},
	"AzureStorageSmbFileShare":  {"storageAccountResourceId", "fileShareName"},
	"NfsMount":                  {"host", "export"},
	"SmbMount":                  {"host", "shareName"},
}

func azureStorageMoverEndpointValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	endpointType := azureARMString(props["endpointType"])
	required, ok := azureStorageMoverEndpointFields[endpointType]
	if !ok {
		return "properties.endpointType must be AzureStorageBlobContainer, AzureStorageSmbFileShare, NfsMount or SmbMount."
	}
	for _, key := range required {
		if strings.TrimSpace(azureARMString(props[key])) == "" {
			return "properties." + key + " is required for " + endpointType + " endpoints."
		}
	}
	if account, present := props["storageAccountResourceId"]; present && !strings.Contains(strings.ToLower(azureARMString(account)), "/providers/microsoft.storage/storageaccounts/") {
		return "properties.storageAccountResourceId must be a storage account ID."
	}
	if endpointType == "NfsMount" {
		if msg := azureEnumProperty("nfsVersion", "NFSauto", "NFSv3", "NFSv4")(payload); msg != "" {
			return msg
		}
	}
	// SMB credentials are Key Vault secret URIs, never the secrets.
	credentials := azureARMObject(props["credentials"])
	for _, key := range []string{"usernameUri", "passwordUri"} {
		if uri, present := credentials[key]; present && !strings.HasPrefix(strings.ToLower(azureARMString(uri)), "https://") {
			return "properties.credentials." + key + " must be a Key Vault secret URI."
		}
	}
	return ""
}

func azureStorageMoverJobDefinitionValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if !slices.Contains([]string{"Additive", "Mirror"}, azureARMString(props["copyMode"])) {
		return "properties.copyMode must be Additive or Mirror."
	}
	for _, key := range []string{"sourceName", "targetName"} {
		if strings.TrimSpace(azureARMString(props[key])) == "" {
			return "properties." + key + " is required."
		}
	}
	return ""
}

// azureStorageMoverJobDefinitionProperties resolves the endpoint and agent
// names to IDs under the job definition's storage mover.
func azureStorageMoverJobDefinitionProperties(rec *azureARMResource) map[string]any {
	mover := path.Dir(path.Dir(path.Dir(path.Dir(rec.ID))))
	props := map[string]any{
		"sourceResourceId": mover + "/endpoints/" + azureARMString(rec.Properties["sourceName"]),
		"targetResourceId": mover + "/endpoints/" + azureARMString(rec.Properties["targetName"]),
	}
	if agent := azureARMString(rec.Properties["agentName"]); agent != "" {
		props["agentResourceId"] = mover + "/agents/" + agent
	}
	return props
}

func azureStorageMoverStartJob(rec *azureARMResource, payload map[string]any) (int, any) {
	if azureARMString(rec.Properties["agentName"]) == "" {
		return http.StatusBadRequest, azureARMErrorBody("BadRequest", "A job definition needs an agentName to run.")
	}
	return azureStorageMoverJobRun(rec, payload)
}

func azureStorageMoverJobRun(rec *azureARMResource, _ map[string]any) (int, any) {
	run := uuid.NewSHA1(uuid.NameSpaceURL, []byte(azureARMKey(rec.ID)+"#run")).String()
	return http.StatusOK, map[string]any{"jobRunResourceId": rec.ID + "/jobRuns/" + run}
}
