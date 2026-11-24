package server

import (
	"net/http"
	"regexp"
	"slices"
	"strings"
	"time"
)

const (
	azureDataMigration20220330PreviewAPIVersion = "2022-03-30-preview"
	azureDMSNamespace                           = "Microsoft.DataMigration"
	azureDMSMaxCommands                         = 50
)

// Azure Database Migration Service (classic, Microsoft.DataMigration
// 2022-03-30-preview) on the shared ARM handler: services with start, stop
// and checkStatus, service tasks, projects with their tasks and files, name
// checks, SKUs and the operations list. No migration runs; tasks wait in
// Queued until a client cancels them.
var (
	azureDMSNameRE  = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,62}$`)
	azureDMSSKUs    = []string{"Standard_1vCores", "Standard_2vCores", "Standard_4vCores", "Premium_4vCores"}
	azureDMSSources = []string{"SQL", "MySQL", "PostgreSql", "MongoDb", "Unknown"}
	azureDMSTargets = []string{"SQLDB", "SQLMI", "AzureDbForMySql", "AzureDbForPostgreSql", "MongoDb", "Unknown"}
	// azureDMSCommandTypes are the task commands the API defines.
	azureDMSCommandTypes = []string{"Migrate.Sync.Complete.Database", "Migrate.SqlServer.AzureDbSqlMi.Complete", "cancel", "finish", "restart"}
	azureDMSTaskTypes    = []string{"ConnectToSource.SqlServer", "ConnectToTarget.SqlDb", "ConnectToTarget.SqlServer", "ConnectToTarget.AzureSqlDbMI", "GetUserTables.Sql", "Migrate.SqlServer.SqlDb", "Migrate.SqlServer.AzureSqlDbMI", "Migrate.MySql.AzureDbForMySql", "Migrate.PostgreSql.AzureDbForPostgreSql.SyncV2", "Migrate.MongoDb", "Service.Check.OCI", "Service.Upload.OCI", "Service.Install.OCI", "Validate.MongoDb"}
)

func init() {
	ns := azureDMSNamespace
	task := func(typ string) azureARMResourceType {
		return azureARMResourceType{
			Namespace: ns, Type: typ, Proxy: true, ETag: true,
			ValidateName:    azureDMSValidateName,
			Validate:        azureDMSTaskValidate,
			StateProperties: map[string]any{"state": "Queued", "commands": []any{}},
			ServerActions: map[string]func(*Server, *azureARMResource, map[string]any) (int, any){
				"command": azureDMSTaskCommand,
				"cancel": func(s *Server, rec *azureARMResource, _ map[string]any) (int, any) {
					return s.azureDMSCancelTask(rec.ID)
				},
			},
		}
	}
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "services", ETag: true,
		ValidateName:       azureDMSValidateName,
		Validate:           azureDMSServiceValidate,
		ReadOnlyProperties: azureDMSServiceProperties,
		StateProperties:    map[string]any{"provisioningState": "Succeeded"},
		Transitions: map[string]map[string]any{
			"start": {"provisioningState": "Succeeded"},
			"stop":  {"provisioningState": "Stopped"},
		},
		Actions: map[string]azureARMAction{"checkstatus": azureDMSCheckStatus},
		ServerActions: map[string]func(*Server, *azureARMResource, map[string]any) (int, any){
			"checknameavailability": azureDMSCheckChildName,
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "services/projects", ETag: true,
		ValidateName: azureDMSValidateName,
		Validate:     azureDMSProjectValidate,
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			return map[string]any{"creationTime": rec.CreatedAt.Format(time.RFC3339)}
		},
	})
	registerAzureARMResourceType(task("services/serviceTasks"))
	registerAzureARMResourceType(task("services/projects/tasks"))
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "services/projects/files", Proxy: true, ETag: true,
		ValidateName: azureDMSValidateName,
		Validate:     azureDMSFileValidate,
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			return map[string]any{"lastModified": rec.UpdatedAt.Format(time.RFC3339), "size": 0}
		},
		Actions: map[string]azureARMAction{"read": azureDMSFileAccess, "readwrite": azureDMSFileAccess},
	})
	registerAzureARMLocationAction(ns, "checkNameAvailability", func(s *Server, sub, _ string, payload map[string]any) (int, any) {
		if typ := azureARMString(payload["type"]); !strings.EqualFold(typ, "services") && !strings.EqualFold(typ, ns+"/services") {
			return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "type must be Microsoft.DataMigration/services.")
		}
		return s.azureDMSCheckName("/subscriptions/"+sub, ns+"/services", payload)
	})
	registerAzureARMView(ns, "skus", func(*Server, string) []map[string]any {
		out := make([]map[string]any, 0, len(azureDMSSKUs))
		for _, sku := range azureDMSSKUs {
			tier, size, _ := strings.Cut(sku, "_")
			out = append(out, map[string]any{
				"resourceType": "services", "name": sku, "tier": tier, "size": strings.TrimSuffix(size, "s"),
				"locations":    []any{"eastus", "westus", "westeurope"},
				"capacity":     map[string]any{"minimum": 1, "maximum": 1, "default": 1, "scaleType": "None"},
				"restrictions": []any{},
			})
		}
		return out
	})
	azureDMSOperationsView(ns, "Microsoft Data Migration",
		[2]string{"services", "Database Migration Service"}, [2]string{"services/projects", "projects"},
		[2]string{"services/serviceTasks", "service tasks"}, [2]string{"services/projects/tasks", "tasks"},
		[2]string{"services/projects/files", "files"})
}

// handleAzureDataMigration20220330PreviewRouter answers paths under the
// provider that the shared ARM handler did not serve.
func (s *Server) handleAzureDataMigration20220330PreviewRouter(w http.ResponseWriter, r *http.Request) bool {
	return azureDMSUnknownRoute(w, r, azureDMSNamespace)
}

func azureDMSValidateName(name string) string {
	if !azureDMSNameRE.MatchString(name) {
		return "Names must be 1-63 characters of letters, digits, '-', '_' and '.', starting with a letter or digit."
	}
	return ""
}

func azureDMSServiceValidate(payload map[string]any) string {
	sku := azureARMString(azureARMObject(payload["sku"])["name"])
	if !slices.Contains(azureDMSSKUs, sku) {
		return "sku.name must be one of " + strings.Join(azureDMSSKUs, ", ") + "."
	}
	props := azureARMObject(payload["properties"])
	if subnet := azureARMString(props["virtualSubnetId"]); subnet != "" && !strings.Contains(strings.ToLower(subnet), "/providers/microsoft.network/virtualnetworks/") {
		return "properties.virtualSubnetId must be a virtual network subnet ID."
	}
	if raw, present := props["autoStopDelay"]; present {
		if _, ok := raw.(string); !ok {
			return "properties.autoStopDelay must be a string."
		}
	}
	return ""
}

func azureDMSServiceProperties(rec *azureARMResource) map[string]any {
	return map[string]any{
		"virtualNicId": "/subscriptions/" + rec.SubscriptionID + "/resourceGroups/" + rec.ResourceGroup + "/providers/Microsoft.Network/networkInterfaces/" + rec.Name + "-nic",
		"publicKey":    "",
	}
}

func azureDMSCheckStatus(rec *azureARMResource, _ map[string]any) (int, any) {
	status := "Online"
	if rec.Properties["provisioningState"] == "Stopped" {
		status = "Stopped"
	}
	return http.StatusOK, map[string]any{
		"agentVersion":       "1.0.0",
		"agentConfiguration": map[string]any{},
		"status":             status,
		"vmSize":             "Standard_D2s_v3",
		"supportedTaskTypes": azureDMSTaskTypes,
	}
}

// azureDMSCheckChildName checks a project or service task name under the service.
func azureDMSCheckChildName(s *Server, rec *azureARMResource, payload map[string]any) (int, any) {
	typ := azureARMString(payload["type"])
	child := typ[strings.LastIndex(typ, "/")+1:]
	switch strings.ToLower(child) {
	case "projects":
		return s.azureDMSCheckName(rec.ID, azureDMSNamespace+"/services/projects", payload)
	case "servicetasks":
		return s.azureDMSCheckName(rec.ID, azureDMSNamespace+"/services/serviceTasks", payload)
	}
	return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "type must be projects or serviceTasks.")
}

func azureDMSProjectValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if !slices.Contains(azureDMSSources, azureARMString(props["sourcePlatform"])) {
		return "properties.sourcePlatform must be one of " + strings.Join(azureDMSSources, ", ") + "."
	}
	if !slices.Contains(azureDMSTargets, azureARMString(props["targetPlatform"])) {
		return "properties.targetPlatform must be one of " + strings.Join(azureDMSTargets, ", ") + "."
	}
	return ""
}

func azureDMSTaskValidate(payload map[string]any) string {
	if strings.TrimSpace(azureARMString(azureARMObject(payload["properties"])["taskType"])) == "" {
		return "properties.taskType is required."
	}
	return ""
}

func azureDMSFileValidate(payload map[string]any) string {
	for _, key := range []string{"extension", "filePath", "mediaType"} {
		if raw, present := azureARMObject(payload["properties"])[key]; present {
			if _, ok := raw.(string); !ok {
				return "properties." + key + " must be a string."
			}
		}
	}
	return ""
}

// azureDMSFileAccess returns a SAS URI for the file; no storage backs it.
func azureDMSFileAccess(rec *azureARMResource, _ map[string]any) (int, any) {
	return http.StatusOK, map[string]any{
		"uri":     "https://stackyarddms.blob.core.windows.net/projects/" + rec.Name + "?sv=2022-11-02&sig=stackyard",
		"headers": map[string]any{"x-ms-blob-type": "BlockBlob"},
	}
}

// azureDMSTaskCommand records a command on the task; cancel also cancels it.
func azureDMSTaskCommand(s *Server, rec *azureARMResource, payload map[string]any) (int, any) {
	commandType := azureARMString(payload["commandType"])
	i := slices.IndexFunc(azureDMSCommandTypes, func(t string) bool { return strings.EqualFold(t, commandType) })
	if i < 0 {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "commandType must be one of "+strings.Join(azureDMSCommandTypes, ", ")+".")
	}
	command := map[string]any{"commandType": azureDMSCommandTypes[i], "state": "Accepted", "errors": []any{}}
	if input, present := payload["input"]; present {
		command["input"] = input
	}
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	current := s.azureARMResources[azureARMKey(rec.ID)]
	if current == nil {
		return http.StatusNotFound, azureARMErrorBody("ResourceNotFound", "The task '"+rec.Name+"' was not found.")
	}
	commands := azureARMSlice(current.Properties["commands"])
	if len(commands) >= azureDMSMaxCommands {
		return http.StatusConflict, azureARMErrorBody("Conflict", "The task already has the maximum number of commands.")
	}
	props := map[string]any{"commands": append(slices.Clone(commands), command)}
	if strings.EqualFold(azureDMSCommandTypes[i], "cancel") {
		props["state"] = "Canceled"
	}
	s.azureDMSUpdateLocked(rec.ID, props)
	return http.StatusOK, command
}

func (s *Server) azureDMSCancelTask(id string) (int, any) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	updated := s.azureDMSUpdateLocked(id, map[string]any{"state": "Canceled"})
	if updated == nil {
		return http.StatusNotFound, azureARMErrorBody("ResourceNotFound", "The task '"+id+"' was not found.")
	}
	return http.StatusOK, updated.fixture()
}
