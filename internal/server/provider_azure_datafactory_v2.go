package server

import (
	"net/http"
	"strings"
	"time"
)

// Azure Data Factory v2 (Microsoft.DataFactory 2018-06-01): factories with
// their linked services, datasets, pipelines, triggers, data flows,
// integration runtimes, managed virtual networks and private endpoints,
// global parameters, credentials and private endpoint connections, served
// by the shared ARM handler, plus pipeline runs and their activity runs,
// trigger control, integration runtime control and the operations list.
// Nothing executes: a pipeline run succeeds the moment it is created, with
// one succeeded activity run per pipeline activity
// (azure_datafactory_runs.go).

const (
	azureDataFactoryV2APIVersion = "2018-06-01"
	azureADFNamespace            = "Microsoft.DataFactory"
	azureADFFactoryType          = azureADFNamespace + "/factories"
)

func init() {
	ns := azureADFNamespace
	// Data Factory answers create-or-update with 200 for new and existing
	// resources alike.
	child := func(typ string, validate func(map[string]any) string) azureARMResourceType {
		return azureARMResourceType{Namespace: ns, Type: "factories/" + typ, Proxy: true, ETag: true, NoPatch: true, CreateStatus: http.StatusOK, Validate: validate}
	}
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "factories", ETag: true, CreateStatus: http.StatusOK,
		ValidateName: azureADFValidateName,
		Validate: azureDataPlatAll(
			azureDataPlatEnum("publicNetworkAccess", "Enabled", "Disabled"),
			azureADFValidateIdentity,
		),
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			return map[string]any{"createTime": rec.CreatedAt.Format(time.RFC3339Nano), "version": azureDataFactoryV2APIVersion}
		},
		Actions: map[string]azureARMAction{
			"getgithubaccesstoken": azureADFGitHubAccessToken,
			"getdataplaneaccess":   azureADFDataPlaneAccess,
		},
		ServerActions: map[string]func(*Server, *azureARMResource, map[string]any) (int, any){
			"querypipelineruns": azureADFQueryPipelineRuns,
			"querytriggerruns":  azureADFQueryTriggerRuns,
			"querytriggers":     azureADFQueryTriggers,
		},
	})
	registerAzureARMResourceType(child("linkedservices", azureDataPlatRequired("type")))
	registerAzureARMResourceType(child("datasets", azureDataPlatRequired("type", "linkedServiceName.referenceName")))
	pipelines := child("pipelines", azureADFPipelineValidate)
	pipelines.ServerActions = map[string]func(*Server, *azureARMResource, map[string]any) (int, any){"createrun": azureADFCreateRun}
	registerAzureARMResourceType(pipelines)
	triggers := child("triggers", azureDataPlatRequired("type"))
	triggers.StateProperties = map[string]any{"runtimeState": "Stopped"}
	triggers.Transitions = map[string]map[string]any{"start": {"runtimeState": "Started"}, "stop": {"runtimeState": "Stopped"}}
	triggers.Actions = map[string]azureARMAction{
		"subscribetoevents":          azureADFTriggerSubscription("Enabled"),
		"unsubscribefromevents":      azureADFTriggerSubscription("Disabled"),
		"geteventsubscriptionstatus": azureADFTriggerSubscription(""),
	}
	registerAzureARMResourceType(triggers)
	registerAzureARMResourceType(child("dataflows", azureDataPlatAll(
		azureDataPlatRequired("type"),
		azureDataPlatEnum("type", "MappingDataFlow", "Flowlet", "WranglingDataFlow"),
	)))
	runtimes := child("integrationRuntimes", azureDataPlatAll(
		azureDataPlatRequired("type"),
		azureDataPlatEnum("type", "Managed", "SelfHosted"),
	))
	runtimes.NoPatch = false
	runtimes.StateProperties = map[string]any{"state": "Stopped"}
	runtimes.Transitions = map[string]map[string]any{"start": {"state": "Started"}, "stop": {"state": "Stopped"}}
	runtimes.Actions = azureADFRuntimeActions()
	registerAzureARMResourceType(runtimes)
	registerAzureARMView(ns, "factories/integrationRuntimes/outboundNetworkDependenciesEndpoints", func(*Server, string) []map[string]any {
		return []map[string]any{{
			"category":  "Azure Data Factory (Management)",
			"endpoints": []any{map[string]any{"domainName": "*.frontend.clouddatahub.net", "endpointDetails": []any{map[string]any{"port": 443}}}},
		}}
	})
	registerAzureARMResourceType(child("managedVirtualNetworks", nil))
	registerAzureARMResourceType(child("managedVirtualNetworks/managedPrivateEndpoints", azureDataPlatRequired("privateLinkResourceId", "groupId")))
	globalParameters := child("globalParameters", nil)
	globalParameters.ValidateName = azureDataPlatOnlyName("default")
	registerAzureARMResourceType(globalParameters)
	registerAzureARMResourceType(child("credentials", azureDataPlatAll(
		azureDataPlatRequired("type"),
		azureDataPlatEnum("type", "ManagedIdentity", "ServicePrincipal"),
	)))
	registerAzureARMResourceType(child("privateEndpointConnections", azureDataPlatAll(
		azureDataPlatRequired("privateLinkServiceConnectionState.status"),
		azureDataPlatEnum("privateLinkServiceConnectionState.status", "Pending", "Approved", "Rejected", "Disconnected"),
	)))
	registerAzureARMLocationAction(ns, "configureFactoryRepo", azureADFConfigureFactoryRepo)
	registerAzureARMView(ns, "operations", azureDataPlatOperations(ns, "Microsoft Data Factory",
		"factories/read", "factories/write", "factories/delete",
		"factories/linkedservices/read", "factories/linkedservices/write", "factories/linkedservices/delete",
		"factories/datasets/read", "factories/datasets/write", "factories/datasets/delete",
		"factories/pipelines/read", "factories/pipelines/write", "factories/pipelines/delete", "factories/pipelines/createrun/action",
		"factories/pipelineruns/read", "factories/pipelineruns/cancel/action", "factories/pipelineruns/queryactivityruns/action",
		"factories/querypipelineruns/action", "factories/querytriggerruns/action",
		"factories/triggers/read", "factories/triggers/write", "factories/triggers/delete", "factories/triggers/start/action", "factories/triggers/stop/action",
		"factories/dataflows/read", "factories/dataflows/write", "factories/dataflows/delete",
		"factories/integrationRuntimes/read", "factories/integrationRuntimes/write", "factories/integrationRuntimes/delete",
		"factories/integrationRuntimes/start/action", "factories/integrationRuntimes/stop/action",
		"locations/configureFactoryRepo/action"))
}

func azureADFValidateName(name string) string {
	if len(name) < 3 || len(name) > 63 || strings.IndexFunc(name, func(r rune) bool {
		return !(r == '-' || r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z')
	}) >= 0 || strings.HasPrefix(name, "-") || strings.HasSuffix(name, "-") {
		return "The factory name '" + name + "' must be 3 to 63 letters, digits and hyphens, starting and ending with a letter or digit."
	}
	return ""
}

func azureADFValidateIdentity(payload map[string]any) string {
	switch azureARMString(azureARMObject(payload["identity"])["type"]) {
	case "", "SystemAssigned", "UserAssigned", "SystemAssigned,UserAssigned":
		return ""
	}
	return "identity.type must be SystemAssigned, UserAssigned or SystemAssigned,UserAssigned."
}

// azureADFPipelineValidate checks that each activity names itself and its
// type, since activity runs are reported per activity.
func azureADFPipelineValidate(payload map[string]any) string {
	raw, present := azureARMObject(payload["properties"])["activities"]
	if !present || raw == nil {
		return ""
	}
	activities, ok := raw.([]any)
	if !ok {
		return "properties.activities must be an array."
	}
	seen := map[string]bool{}
	for _, item := range activities {
		activity := azureARMObject(item)
		name := azureARMString(activity["name"])
		switch {
		case name == "" || azureARMString(activity["type"]) == "":
			return "every activity in properties.activities needs a name and type."
		case seen[name]:
			return "the activity name '" + name + "' is used twice."
		}
		seen[name] = true
	}
	return ""
}

func azureADFGitHubAccessToken(_ *azureARMResource, payload map[string]any) (int, any) {
	if azureARMString(payload["gitHubAccessCode"]) == "" || azureARMString(payload["gitHubAccessTokenBaseUrl"]) == "" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "gitHubAccessCode and gitHubAccessTokenBaseUrl are required.")
	}
	return http.StatusOK, map[string]any{"gitHubAccessToken": "gho_" + strings.ReplaceAll(azureDataPlatGUID(azureARMString(payload["gitHubAccessCode"])), "-", "")}
}

func azureADFDataPlaneAccess(rec *azureARMResource, payload map[string]any) (int, any) {
	if azureARMString(payload["permissions"]) == "" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "permissions is required.")
	}
	return http.StatusOK, map[string]any{
		"policy":       payload,
		"accessToken":  azureDataPlatGUID(rec.ID + "/dataplane"),
		"dataPlaneUrl": "https://" + strings.ToLower(rec.Location) + ".frontend.clouddatahub.net",
	}
}

// azureADFTriggerSubscription answers the event subscription operations of
// a trigger with the status they leave, or, for "", the current one.
// ponytail: the status is not stored; it follows the trigger's runtime
// state. Store it if a client subscribes without starting the trigger.
func azureADFTriggerSubscription(status string) azureARMAction {
	return func(rec *azureARMResource, _ map[string]any) (int, any) {
		current := status
		if current == "" {
			current = "Disabled"
			if rec.Properties["runtimeState"] == "Started" {
				current = "Enabled"
			}
		}
		return http.StatusOK, map[string]any{"triggerName": rec.Name, "status": current}
	}
}

// azureADFConfigureFactoryRepo sets or clears a factory's repository
// configuration.
func azureADFConfigureFactoryRepo(s *Server, sub, _ string, payload map[string]any) (int, any) {
	factoryID := azureARMString(payload["factoryResourceId"])
	if !strings.HasPrefix(strings.ToLower(factoryID), strings.ToLower("/subscriptions/"+sub+"/")) {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "factoryResourceId must be a factory in this subscription.")
	}
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	factory := s.azureDataPlatStoredLocked(factoryID, azureADFFactoryType)
	if factory == nil {
		return http.StatusNotFound, azureARMErrorBody("ResourceNotFound", "The factory '"+factoryID+"' was not found.")
	}
	updated := *factory
	updated.Properties = azureARMMerge(factory.Properties, nil)
	if repo := azureARMObject(payload["repoConfiguration"]); len(repo) > 0 {
		if azureARMString(repo["type"]) != "FactoryVSTSConfiguration" && azureARMString(repo["type"]) != "FactoryGitHubConfiguration" {
			return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "repoConfiguration.type must be FactoryVSTSConfiguration or FactoryGitHubConfiguration.")
		}
		updated.Properties["repoConfiguration"] = repo
	} else {
		delete(updated.Properties, "repoConfiguration")
	}
	updated.UpdatedAt = time.Now().UTC()
	s.azureARMResources[azureARMKey(factory.ID)] = &updated
	return http.StatusOK, updated.fixture()
}

// handleAzureDataFactoryV2Router serves a factory's pipeline runs, which
// are not ARM resources: GET .../pipelineruns/{runId} and POST
// .../pipelineruns/{runId}/{cancel,queryActivityruns}.
func (s *Server) handleAzureDataFactoryV2Router(w http.ResponseWriter, r *http.Request) bool {
	seg := splitPathSegments(strings.TrimPrefix(rawRequestPath(r), "/azure/"))
	if len(seg) < 10 || len(seg) > 11 || !strings.EqualFold(seg[0], "subscriptions") || !strings.EqualFold(seg[2], "resourceGroups") ||
		!strings.EqualFold(seg[5], azureADFNamespace) || !strings.EqualFold(seg[6], "factories") || !strings.EqualFold(seg[8], "pipelineruns") {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		azureARMError(w, http.StatusBadRequest, "InvalidApiVersionParameter", "invalid api-version query parameter")
		return true
	}
	factoryID := "/subscriptions/" + seg[1] + "/resourceGroups/" + seg[3] + "/providers/" + azureADFNamespace + "/factories/" + seg[7]
	s.serveAzureADFRun(w, r, factoryID, seg[9], seg[10:])
	return true
}
