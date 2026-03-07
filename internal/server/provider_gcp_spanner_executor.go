package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const gcpSpannerExecutorExecuteActionAsyncPath = "/gcp/google.spanner.executor.v1.SpannerExecutorProxy/ExecuteActionAsync"

var gcpSpannerExecutorReferenceTime = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

type gcpSpannerExecutorOneofField struct {
	name    string
	aliases []string
}

var gcpSpannerExecutorActionFields = []gcpSpannerExecutorOneofField{
	{name: "start", aliases: []string{"start"}},
	{name: "finish", aliases: []string{"finish"}},
	{name: "read", aliases: []string{"read"}},
	{name: "query", aliases: []string{"query"}},
	{name: "mutation", aliases: []string{"mutation"}},
	{name: "dml", aliases: []string{"dml"}},
	{name: "batchDml", aliases: []string{"batchDml", "batch_dml"}},
	{name: "write", aliases: []string{"write"}},
	{name: "partitionedUpdate", aliases: []string{"partitionedUpdate", "partitioned_update"}},
	{name: "admin", aliases: []string{"admin"}},
	{name: "startBatchTxn", aliases: []string{"startBatchTxn", "start_batch_txn"}},
	{name: "closeBatchTxn", aliases: []string{"closeBatchTxn", "close_batch_txn"}},
	{name: "generateDbPartitionsRead", aliases: []string{"generateDbPartitionsRead", "generate_db_partitions_read"}},
	{name: "generateDbPartitionsQuery", aliases: []string{"generateDbPartitionsQuery", "generate_db_partitions_query"}},
	{name: "executePartition", aliases: []string{"executePartition", "execute_partition"}},
	{name: "executeChangeStreamQuery", aliases: []string{"executeChangeStreamQuery", "execute_change_stream_query"}},
	{name: "queryCancellation", aliases: []string{"queryCancellation", "query_cancellation"}},
}

var gcpSpannerExecutorAdminActionFields = []gcpSpannerExecutorOneofField{
	{name: "createUserInstanceConfig", aliases: []string{"createUserInstanceConfig", "create_user_instance_config"}},
	{name: "updateUserInstanceConfig", aliases: []string{"updateUserInstanceConfig", "update_user_instance_config"}},
	{name: "deleteUserInstanceConfig", aliases: []string{"deleteUserInstanceConfig", "delete_user_instance_config"}},
	{name: "getCloudInstanceConfig", aliases: []string{"getCloudInstanceConfig", "get_cloud_instance_config"}},
	{name: "listInstanceConfigs", aliases: []string{"listInstanceConfigs", "list_instance_configs"}},
	{name: "createCloudInstance", aliases: []string{"createCloudInstance", "create_cloud_instance"}},
	{name: "updateCloudInstance", aliases: []string{"updateCloudInstance", "update_cloud_instance"}},
	{name: "deleteCloudInstance", aliases: []string{"deleteCloudInstance", "delete_cloud_instance"}},
	{name: "listCloudInstances", aliases: []string{"listCloudInstances", "list_cloud_instances"}},
	{name: "getCloudInstance", aliases: []string{"getCloudInstance", "get_cloud_instance"}},
	{name: "createCloudDatabase", aliases: []string{"createCloudDatabase", "create_cloud_database"}},
	{name: "updateCloudDatabaseDdl", aliases: []string{"updateCloudDatabaseDdl", "update_cloud_database_ddl"}},
	{name: "updateCloudDatabase", aliases: []string{"updateCloudDatabase", "update_cloud_database"}},
	{name: "dropCloudDatabase", aliases: []string{"dropCloudDatabase", "drop_cloud_database"}},
	{name: "listCloudDatabases", aliases: []string{"listCloudDatabases", "list_cloud_databases"}},
	{name: "listCloudDatabaseOperations", aliases: []string{"listCloudDatabaseOperations", "list_cloud_database_operations"}},
	{name: "restoreCloudDatabase", aliases: []string{"restoreCloudDatabase", "restore_cloud_database"}},
	{name: "getCloudDatabase", aliases: []string{"getCloudDatabase", "get_cloud_database"}},
	{name: "createCloudBackup", aliases: []string{"createCloudBackup", "create_cloud_backup"}},
	{name: "copyCloudBackup", aliases: []string{"copyCloudBackup", "copy_cloud_backup"}},
	{name: "getCloudBackup", aliases: []string{"getCloudBackup", "get_cloud_backup"}},
	{name: "updateCloudBackup", aliases: []string{"updateCloudBackup", "update_cloud_backup"}},
	{name: "deleteCloudBackup", aliases: []string{"deleteCloudBackup", "delete_cloud_backup"}},
	{name: "listCloudBackups", aliases: []string{"listCloudBackups", "list_cloud_backups"}},
	{name: "listCloudBackupOperations", aliases: []string{"listCloudBackupOperations", "list_cloud_backup_operations"}},
	{name: "getOperation", aliases: []string{"getOperation", "get_operation"}},
	{name: "cancelOperation", aliases: []string{"cancelOperation", "cancel_operation"}},
	{name: "changeQuorumCloudDatabase", aliases: []string{"changeQuorumCloudDatabase", "change_quorum_cloud_database"}},
}

func (s *Server) handleGCPSpannerExecutorRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_spanner_executor(w, r) {
		return true
	}

	path := strings.TrimSpace(rawRequestPath(r))
	if !isGCPSpannerExecutorPath(path, hasGCPSpannerExecutorHint(r)) {
		return false
	}

	if r.Method != http.MethodPost {
		return false
	}

	if path != gcpSpannerExecutorExecuteActionAsyncPath {
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	}

	body, valid := decodeGCPSpannerExecutorJSONBody(w, r, path)
	if !valid {
		return true
	}

	actionID, ok := gcpSpannerExecutorInt32(body, "actionId", "action_id")
	if !ok {
		respondGCPSpannerExecutorInvalidArgument(w, path, "actionId is required")
		return true
	}
	if actionID <= 0 {
		respondGCPSpannerExecutorInvalidArgument(w, path, "actionId must be positive")
		return true
	}
	if actionID > 1_000_000 {
		respondGCPSpannerExecutorInvalidArgument(w, path, "actionId is out of range")
		return true
	}

	action := gcpSpannerExecutorBodyMap(body, "action")
	if len(action) == 0 {
		respondGCPSpannerExecutorInvalidArgument(w, path, "action is required")
		return true
	}

	databasePath := strings.TrimSpace(gcpSpannerExecutorString(action, "databasePath", "database_path"))
	project := "stackyard"
	instance := "stackyard-instance"
	database := "stackyard-db"
	if databasePath != "" {
		p, i, d, validName := parseGCPSpannerDatabaseName(databasePath)
		if !validName {
			respondGCPSpannerExecutorInvalidArgument(w, path, "action.databasePath is invalid")
			return true
		}
		project, instance, database = p, i, d
		if isGCPSpannerMissingResource(project, instance, database) {
			respondGCPSpannerExecutorNotFound(w, path, "database not found")
			return true
		}
	}

	actionKind, payload, errMessage := gcpSpannerExecutorSelectOneofMap(action, gcpSpannerExecutorActionFields, "action")
	if errMessage != "" {
		respondGCPSpannerExecutorInvalidArgument(w, path, errMessage)
		return true
	}
	if actionKind == "" {
		respondGCPSpannerExecutorInvalidArgument(w, path, "action kind is required")
		return true
	}

	outcome, status, errToken, message := gcpSpannerExecutorBuildActionOutcome(path, actionKind, payload, project, instance, database)
	if status != http.StatusOK {
		respondGCPSpannerExecutorError(w, status, errToken, path, message)
		return true
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"actionId": actionID,
		"outcome":  outcome,
	})
	return true
}

func hasGCPSpannerExecutorHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "spanner_executor", "spanner-executor", "spanner-executor-apiv1", "spanner_executor_apiv1", "cloud-spanner-executor", "cloud_spanner_executor", "cloudspannerexecutor", "gcp-cloud-spanner-executor":
		return true
	}
	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-spanner-executor-apiv1") || strings.Contains(ua, "cloud.google.com/go/spanner/executor")
}

func isGCPSpannerExecutorPath(path string, includeHint bool) bool {
	if path == gcpSpannerExecutorExecuteActionAsyncPath {
		return true
	}
	if includeHint {
		return strings.HasPrefix(path, "/gcp/google.spanner.executor.v1.SpannerExecutorProxy/")
	}
	return false
}

func decodeGCPSpannerExecutorJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		return map[string]any{}, true
	}
	defer r.Body.Close()

	dec := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	dec.UseNumber()

	var body map[string]any
	if err := dec.Decode(&body); err != nil {
		if err == io.EOF {
			return map[string]any{}, true
		}
		respondGCPSpannerExecutorInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func gcpSpannerExecutorBuildActionOutcome(path, actionKind string, payload map[string]any, project, instance, database string) (map[string]any, int, string, string) {
	outcome := map[string]any{
		"status": gcpSpannerExecutorStatusFixture(0, "OK"),
	}

	switch actionKind {
	case "start":
		return outcome, http.StatusOK, "", ""
	case "finish":
		mode := strings.TrimSpace(gcpSpannerExecutorString(payload, "mode"))
		if strings.EqualFold(mode, "MODE_UNSPECIFIED") {
			return nil, http.StatusBadRequest, "InvalidArgument", "finish.mode is invalid"
		}
		outcome["commitTime"] = gcpSpannerExecutorReferenceTime.Format(time.RFC3339)
		outcome["transactionRestarted"] = false
		return outcome, http.StatusOK, "", ""
	case "read":
		table := strings.TrimSpace(gcpSpannerExecutorString(payload, "table"))
		if table == "" {
			return nil, http.StatusBadRequest, "InvalidArgument", "action.read.table is required"
		}
		columns, ok := gcpSpannerExecutorStringSlice(payload["column"])
		if !ok || len(columns) == 0 {
			columns, ok = gcpSpannerExecutorStringSlice(payload["columns"])
			if !ok || len(columns) == 0 {
				return nil, http.StatusBadRequest, "InvalidArgument", "action.read.column must contain at least one column"
			}
		}
		if _, ok := payload["keys"].(map[string]any); !ok {
			return nil, http.StatusBadRequest, "InvalidArgument", "action.read.keys is required"
		}
		if rawLimit, ok := payload["limit"]; ok {
			limit, valid := gcpSpannerExecutorInt64FromAny(rawLimit)
			if !valid || limit <= 0 {
				return nil, http.StatusBadRequest, "InvalidArgument", "action.read.limit must be positive"
			}
		}
		outcome["readResult"] = gcpSpannerExecutorReadResultFixture(table, columns)
		return outcome, http.StatusOK, "", ""
	case "query":
		sql := strings.TrimSpace(gcpSpannerExecutorString(payload, "sql"))
		if sql == "" {
			return nil, http.StatusBadRequest, "InvalidArgument", "action.query.sql is required"
		}
		outcome["queryResult"] = gcpSpannerExecutorQueryResultFixture()
		return outcome, http.StatusOK, "", ""
	case "mutation":
		mods, ok := payload["mod"].([]any)
		if !ok || len(mods) == 0 {
			return nil, http.StatusBadRequest, "InvalidArgument", "action.mutation.mod must contain at least one mutation"
		}
		return outcome, http.StatusOK, "", ""
	case "dml":
		update := gcpSpannerExecutorBodyMap(payload, "update")
		if len(update) == 0 {
			return nil, http.StatusBadRequest, "InvalidArgument", "action.dml.update is required"
		}
		if strings.TrimSpace(gcpSpannerExecutorString(update, "sql")) == "" {
			return nil, http.StatusBadRequest, "InvalidArgument", "action.dml.update.sql is required"
		}
		outcome["dmlRowsModified"] = []any{float64(1)}
		return outcome, http.StatusOK, "", ""
	case "batchDml":
		updates, ok := payload["updates"].([]any)
		if !ok || len(updates) == 0 {
			return nil, http.StatusBadRequest, "InvalidArgument", "action.batchDml.updates must contain at least one statement"
		}
		dmlRows := make([]any, 0, len(updates))
		for idx, raw := range updates {
			query, ok := raw.(map[string]any)
			if !ok {
				return nil, http.StatusBadRequest, "InvalidArgument", "action.batchDml.updates entries must be objects"
			}
			if strings.TrimSpace(gcpSpannerExecutorString(query, "sql")) == "" {
				return nil, http.StatusBadRequest, "InvalidArgument", fmt.Sprintf("action.batchDml.updates[%d].sql is required", idx)
			}
			dmlRows = append(dmlRows, float64(1))
		}
		outcome["dmlRowsModified"] = dmlRows
		return outcome, http.StatusOK, "", ""
	case "write":
		mutation := gcpSpannerExecutorBodyMap(payload, "mutation")
		if len(mutation) == 0 {
			return nil, http.StatusBadRequest, "InvalidArgument", "action.write.mutation is required"
		}
		mods, ok := mutation["mod"].([]any)
		if !ok || len(mods) == 0 {
			return nil, http.StatusBadRequest, "InvalidArgument", "action.write.mutation.mod must contain at least one mutation"
		}
		return outcome, http.StatusOK, "", ""
	case "partitionedUpdate":
		update := gcpSpannerExecutorBodyMap(payload, "update")
		if len(update) == 0 {
			return nil, http.StatusBadRequest, "InvalidArgument", "action.partitionedUpdate.update is required"
		}
		if strings.TrimSpace(gcpSpannerExecutorString(update, "sql")) == "" {
			return nil, http.StatusBadRequest, "InvalidArgument", "action.partitionedUpdate.update.sql is required"
		}
		outcome["dmlRowsModified"] = []any{float64(7)}
		return outcome, http.StatusOK, "", ""
	case "startBatchTxn":
		tid := strings.TrimSpace(gcpSpannerExecutorString(payload, "tid"))
		_, hasBatchTime := payload["batchTxnTime"]
		if tid == "" && !hasBatchTime {
			return nil, http.StatusBadRequest, "InvalidArgument", "action.startBatchTxn requires either tid or batchTxnTime"
		}
		outcome["batchTxnId"] = "YmF0Y2gtdHgtMQ=="
		return outcome, http.StatusOK, "", ""
	case "closeBatchTxn":
		return outcome, http.StatusOK, "", ""
	case "generateDbPartitionsRead":
		read := gcpSpannerExecutorBodyMap(payload, "read")
		if len(read) == 0 {
			return nil, http.StatusBadRequest, "InvalidArgument", "action.generateDbPartitionsRead.read is required"
		}
		table := strings.TrimSpace(gcpSpannerExecutorString(read, "table"))
		if table == "" {
			return nil, http.StatusBadRequest, "InvalidArgument", "action.generateDbPartitionsRead.read.table is required"
		}
		outcome["dbPartition"] = []any{
			map[string]any{
				"partition":      "cGFydGl0aW9uLTE=",
				"partitionToken": "dG9rZW4tMQ==",
				"table":          table,
			},
		}
		return outcome, http.StatusOK, "", ""
	case "generateDbPartitionsQuery":
		query := gcpSpannerExecutorBodyMap(payload, "query")
		if len(query) == 0 {
			return nil, http.StatusBadRequest, "InvalidArgument", "action.generateDbPartitionsQuery.query is required"
		}
		if strings.TrimSpace(gcpSpannerExecutorString(query, "sql")) == "" {
			return nil, http.StatusBadRequest, "InvalidArgument", "action.generateDbPartitionsQuery.query.sql is required"
		}
		outcome["dbPartition"] = []any{
			map[string]any{
				"partition":      "cGFydGl0aW9uLXF1ZXJ5LTE=",
				"partitionToken": "dG9rZW4tcXVlcnktMQ==",
			},
		}
		return outcome, http.StatusOK, "", ""
	case "executePartition":
		partition := gcpSpannerExecutorBodyMap(payload, "partition")
		if len(partition) == 0 {
			return nil, http.StatusBadRequest, "InvalidArgument", "action.executePartition.partition is required"
		}
		serialized := strings.TrimSpace(gcpSpannerExecutorString(partition, "partition"))
		token := strings.TrimSpace(gcpSpannerExecutorString(partition, "partitionToken", "partition_token"))
		if serialized == "" && token == "" {
			return nil, http.StatusBadRequest, "InvalidArgument", "action.executePartition.partition must contain partition or partitionToken"
		}
		table := strings.TrimSpace(gcpSpannerExecutorString(partition, "table"))
		if table == "" {
			table = "Users"
		}
		outcome["readResult"] = gcpSpannerExecutorReadResultFixture(table, []string{"id", "name"})
		return outcome, http.StatusOK, "", ""
	case "executeChangeStreamQuery":
		name := strings.TrimSpace(gcpSpannerExecutorString(payload, "name"))
		if name == "" {
			return nil, http.StatusBadRequest, "InvalidArgument", "action.executeChangeStreamQuery.name is required"
		}
		if _, ok := payload["startTime"]; !ok {
			return nil, http.StatusBadRequest, "InvalidArgument", "action.executeChangeStreamQuery.startTime is required"
		}
		outcome["changeStreamRecords"] = gcpSpannerExecutorChangeStreamFixture()
		return outcome, http.StatusOK, "", ""
	case "queryCancellation":
		longRunningSQL := strings.TrimSpace(gcpSpannerExecutorString(payload, "longRunningSql", "long_running_sql"))
		if longRunningSQL == "" {
			return nil, http.StatusBadRequest, "InvalidArgument", "action.queryCancellation.longRunningSql is required"
		}
		cancelQuery := strings.TrimSpace(gcpSpannerExecutorString(payload, "cancelQuery", "cancel_query"))
		if cancelQuery == "" {
			return nil, http.StatusBadRequest, "InvalidArgument", "action.queryCancellation.cancelQuery is required"
		}
		if strings.Contains(strings.ToLower(longRunningSQL), "already-cancelled") {
			return nil, http.StatusBadRequest, "FailedPrecondition", "query is already cancelled"
		}
		outcome["queryResult"] = gcpSpannerExecutorQueryResultFixture()
		return outcome, http.StatusOK, "", ""
	case "admin":
		adminOutcome, adminStatus, adminErrToken, adminMessage := gcpSpannerExecutorBuildAdminOutcome(payload, project, instance, database)
		if adminStatus != http.StatusOK {
			return nil, adminStatus, adminErrToken, adminMessage
		}
		for key, value := range adminOutcome {
			outcome[key] = value
		}
		return outcome, http.StatusOK, "", ""
	default:
		return nil, http.StatusNotImplemented, "NotImplemented", "action kind is not implemented"
	}
}

func gcpSpannerExecutorBuildAdminOutcome(payload map[string]any, project, instance, database string) (map[string]any, int, string, string) {
	actionKind, actionPayload, errMessage := gcpSpannerExecutorSelectOneofMap(payload, gcpSpannerExecutorAdminActionFields, "action.admin")
	if errMessage != "" {
		return nil, http.StatusBadRequest, "InvalidArgument", errMessage
	}
	if actionKind == "" {
		return nil, http.StatusBadRequest, "InvalidArgument", "action.admin action kind is required"
	}

	if (strings.HasPrefix(actionKind, "get") || strings.HasPrefix(actionKind, "delete") || actionKind == "cancelOperation" || actionKind == "dropCloudDatabase") &&
		strings.TrimSpace(gcpSpannerExecutorString(actionPayload, "name")) == "" {
		return nil, http.StatusBadRequest, "InvalidArgument", fmt.Sprintf("action.admin.%s.name is required", actionKind)
	}
	if strings.HasPrefix(actionKind, "create") && gcpSpannerExecutorContainsToken(actionPayload, "existing") {
		return nil, http.StatusConflict, "AlreadyExists", "resource already exists"
	}
	if gcpSpannerExecutorContainsToken(actionPayload, "missing") {
		return nil, http.StatusNotFound, "NotFound", "resource not found"
	}
	if strings.HasPrefix(actionKind, "update") && gcpSpannerExecutorContainsToken(actionPayload, "locked") {
		return nil, http.StatusBadRequest, "FailedPrecondition", "resource is locked"
	}

	instanceConfigFixture := gcpSpannerExecutorInstanceConfigFixture(project, "custom-stackyard-primary")
	instanceFixture := gcpSpannerExecutorInstanceFixture(project, instance)
	databaseFixture := gcpSpannerExecutorDatabaseFixture(project, instance, database)
	backupFixture := gcpSpannerExecutorBackupFixture(project, instance, "backup-1", database)
	operationFixture := gcpSpannerExecutorOperationFixture(project, instance, "spanner-executor-op-1")

	adminResult := map[string]any{}

	switch actionKind {
	case "createUserInstanceConfig", "updateUserInstanceConfig", "deleteUserInstanceConfig", "getCloudInstanceConfig", "listInstanceConfigs":
		adminResult["instanceConfigResponse"] = map[string]any{
			"listedInstanceConfigs": []any{instanceConfigFixture},
			"nextPageToken":         "",
			"instanceConfig":        instanceConfigFixture,
		}
	case "createCloudInstance", "updateCloudInstance", "deleteCloudInstance", "listCloudInstances", "getCloudInstance":
		adminResult["instanceResponse"] = map[string]any{
			"listedInstances": []any{instanceFixture},
			"nextPageToken":   "",
			"instance":        instanceFixture,
		}
	case "createCloudDatabase", "updateCloudDatabaseDdl", "updateCloudDatabase", "dropCloudDatabase", "listCloudDatabases", "listCloudDatabaseOperations", "restoreCloudDatabase", "getCloudDatabase", "changeQuorumCloudDatabase":
		adminResult["databaseResponse"] = map[string]any{
			"listedDatabases":          []any{databaseFixture},
			"listedDatabaseOperations": []any{operationFixture},
			"nextPageToken":            "",
			"database":                 databaseFixture,
		}
	case "createCloudBackup", "copyCloudBackup", "getCloudBackup", "updateCloudBackup", "deleteCloudBackup", "listCloudBackups", "listCloudBackupOperations":
		adminResult["backupResponse"] = map[string]any{
			"listedBackups":          []any{backupFixture},
			"listedBackupOperations": []any{operationFixture},
			"nextPageToken":          "",
			"backup":                 backupFixture,
		}
	case "getOperation", "cancelOperation":
		adminResult["operationResponse"] = map[string]any{
			"listedOperations": []any{operationFixture},
			"nextPageToken":    "",
			"operation":        operationFixture,
		}
	default:
		return nil, http.StatusNotImplemented, "NotImplemented", "admin action kind is not implemented"
	}

	return map[string]any{"adminResult": adminResult}, http.StatusOK, "", ""
}

func gcpSpannerExecutorReadResultFixture(table string, columns []string) map[string]any {
	fields := make([]any, 0, len(columns))
	for _, column := range columns {
		fields = append(fields, map[string]any{
			"name": column,
			"type": map[string]any{
				"code": "STRING",
			},
		})
	}

	values := []any{
		map[string]any{"stringValue": "1"},
		map[string]any{"stringValue": "Stackyard"},
	}

	return map[string]any{
		"table": table,
		"row": []any{
			map[string]any{
				"value": values,
			},
		},
		"rowType": map[string]any{
			"fields": fields,
		},
	}
}

func gcpSpannerExecutorQueryResultFixture() map[string]any {
	return map[string]any{
		"row": []any{
			map[string]any{
				"value": []any{
					map[string]any{"stringValue": "stackyard-query-result"},
				},
			},
		},
		"rowType": map[string]any{
			"fields": []any{
				map[string]any{
					"name": "value",
					"type": map[string]any{
						"code": "STRING",
					},
				},
			},
		},
	}
}

func gcpSpannerExecutorChangeStreamFixture() []any {
	return []any{
		map[string]any{
			"heartbeat": map[string]any{
				"timestamp": gcpSpannerExecutorReferenceTime.Format(time.RFC3339),
			},
		},
		map[string]any{
			"dataChange": map[string]any{
				"recordSequence":  "1",
				"tableName":       "Users",
				"commitTimestamp": gcpSpannerExecutorReferenceTime.Add(time.Second).Format(time.RFC3339),
			},
		},
	}
}

func gcpSpannerExecutorStatusFixture(code int32, message string) map[string]any {
	return map[string]any{
		"code":    code,
		"message": message,
	}
}

func gcpSpannerExecutorInstanceConfigFixture(project, configID string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("projects/%s/instanceConfigs/%s", project, configID),
		"displayName": "Stackyard Instance Config",
		"replicas": []any{
			map[string]any{
				"location": "regional-us-central1",
				"type":     "READ_WRITE",
			},
		},
	}
}

func gcpSpannerExecutorInstanceFixture(project, instanceID string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("projects/%s/instances/%s", project, instanceID),
		"config":      fmt.Sprintf("projects/%s/instanceConfigs/custom-stackyard-primary", project),
		"displayName": "Stackyard Instance",
		"nodeCount":   float64(1),
		"state":       "READY",
	}
}

func gcpSpannerExecutorDatabaseFixture(project, instance, database string) map[string]any {
	return map[string]any{
		"name":                 fmt.Sprintf("projects/%s/instances/%s/databases/%s", project, instance, database),
		"state":                "READY",
		"enableDropProtection": false,
	}
}

func gcpSpannerExecutorBackupFixture(project, instance, backupID, database string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("projects/%s/instances/%s/backups/%s", project, instance, backupID),
		"database":    fmt.Sprintf("projects/%s/instances/%s/databases/%s", project, instance, database),
		"versionTime": gcpSpannerExecutorReferenceTime.Format(time.RFC3339),
		"state":       "READY",
	}
}

func gcpSpannerExecutorOperationFixture(project, instance, operationID string) map[string]any {
	return map[string]any{
		"name": fmt.Sprintf("projects/%s/instances/%s/operations/%s", project, instance, operationID),
		"done": true,
	}
}

func gcpSpannerExecutorSelectOneofMap(body map[string]any, fields []gcpSpannerExecutorOneofField, context string) (string, map[string]any, string) {
	seen := 0
	selectedField := ""
	var selectedRaw any

	for _, field := range fields {
		found := false
		for _, alias := range field.aliases {
			raw, ok := body[alias]
			if !ok {
				continue
			}
			found = true
			selectedRaw = raw
			break
		}
		if !found {
			continue
		}
		seen++
		if selectedField == "" {
			selectedField = field.name
		}
	}

	if seen == 0 {
		return "", nil, ""
	}
	if seen > 1 {
		return "", nil, fmt.Sprintf("%s must define exactly one action kind", context)
	}
	payload, ok := selectedRaw.(map[string]any)
	if !ok {
		return "", nil, fmt.Sprintf("%s.%s must be an object", context, selectedField)
	}
	return selectedField, payload, ""
}

func gcpSpannerExecutorBodyMap(body map[string]any, keys ...string) map[string]any {
	if body == nil {
		return map[string]any{}
	}
	for _, key := range keys {
		raw, ok := body[key]
		if !ok || raw == nil {
			continue
		}
		m, ok := raw.(map[string]any)
		if ok {
			return m
		}
		return map[string]any{}
	}
	return map[string]any{}
}

func gcpSpannerExecutorString(body map[string]any, keys ...string) string {
	if body == nil {
		return ""
	}
	for _, key := range keys {
		raw, ok := body[key]
		if !ok || raw == nil {
			continue
		}
		text, _ := raw.(string)
		if strings.TrimSpace(text) != "" {
			return text
		}
	}
	return ""
}

func gcpSpannerExecutorInt32(body map[string]any, keys ...string) (int32, bool) {
	if body == nil {
		return 0, false
	}
	for _, key := range keys {
		raw, ok := body[key]
		if !ok {
			continue
		}
		value, valid := gcpSpannerExecutorInt64FromAny(raw)
		if !valid {
			return 0, false
		}
		return int32(value), true
	}
	return 0, false
}

func gcpSpannerExecutorInt64FromAny(raw any) (int64, bool) {
	switch v := raw.(type) {
	case json.Number:
		i, err := v.Int64()
		if err != nil {
			return 0, false
		}
		return i, true
	case float64:
		return int64(v), true
	case float32:
		return int64(v), true
	case int:
		return int64(v), true
	case int8:
		return int64(v), true
	case int16:
		return int64(v), true
	case int32:
		return int64(v), true
	case int64:
		return v, true
	case uint:
		return int64(v), true
	case uint8:
		return int64(v), true
	case uint16:
		return int64(v), true
	case uint32:
		return int64(v), true
	case uint64:
		if v > uint64(^uint64(0)>>1) {
			return 0, false
		}
		return int64(v), true
	case string:
		i, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		if err != nil {
			return 0, false
		}
		return i, true
	default:
		return 0, false
	}
}

func gcpSpannerExecutorStringSlice(raw any) ([]string, bool) {
	items, ok := raw.([]any)
	if !ok {
		return nil, false
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		text, ok := item.(string)
		if !ok {
			return nil, false
		}
		text = strings.TrimSpace(text)
		if text == "" {
			return nil, false
		}
		out = append(out, text)
	}
	return out, true
}

func gcpSpannerExecutorContainsToken(raw any, token string) bool {
	token = strings.ToLower(strings.TrimSpace(token))
	switch value := raw.(type) {
	case string:
		return strings.Contains(strings.ToLower(value), token)
	case []any:
		for _, item := range value {
			if gcpSpannerExecutorContainsToken(item, token) {
				return true
			}
		}
	case map[string]any:
		for key, item := range value {
			if strings.Contains(strings.ToLower(key), token) {
				return true
			}
			if gcpSpannerExecutorContainsToken(item, token) {
				return true
			}
		}
	}
	return false
}

func respondGCPSpannerExecutorInvalidArgument(w http.ResponseWriter, path, message string) {
	respondGCPSpannerExecutorError(w, http.StatusBadRequest, "InvalidArgument", path, message)
}

func respondGCPSpannerExecutorFailedPrecondition(w http.ResponseWriter, path, message string) {
	respondGCPSpannerExecutorError(w, http.StatusBadRequest, "FailedPrecondition", path, message)
}

func respondGCPSpannerExecutorNotFound(w http.ResponseWriter, path, message string) {
	respondGCPSpannerExecutorError(w, http.StatusNotFound, "NotFound", path, message)
}

func respondGCPSpannerExecutorAlreadyExists(w http.ResponseWriter, path, message string) {
	respondGCPSpannerExecutorError(w, http.StatusConflict, "AlreadyExists", path, message)
}

func respondGCPSpannerExecutorNotImplemented(w http.ResponseWriter, path, message string) {
	respondGCPSpannerExecutorError(w, http.StatusNotImplemented, "NotImplemented", path, message)
}

func respondGCPSpannerExecutorError(w http.ResponseWriter, status int, err, path, message string) {
	switch err {
	case "FailedPrecondition":
		if status == http.StatusOK {
			status = http.StatusBadRequest
		}
	case "NotFound":
		if status == http.StatusOK {
			status = http.StatusNotFound
		}
	case "AlreadyExists":
		if status == http.StatusOK {
			status = http.StatusConflict
		}
	case "NotImplemented":
		if status == http.StatusOK {
			status = http.StatusNotImplemented
		}
	default:
		if status == http.StatusOK {
			status = http.StatusBadRequest
		}
	}
	respondJSON(w, status, map[string]any{
		"error":    err,
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func handleGCPContractProbe_spanner_executor(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "spanner_executor") {
		return false
	}
	if r.URL.Query().Get("pageSize") == "bad" {
		respondGCPSpannerExecutorInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return true
	}
	if r.URL.Query().Get("alreadyExists") == "1" {
		respondGCPSpannerExecutorAlreadyExists(w, path, "resource already exists")
		return true
	}
	if r.URL.Query().Get("failedPrecondition") == "1" {
		respondGCPSpannerExecutorFailedPrecondition(w, path, "precondition failed")
		return true
	}
	if r.URL.Query().Get("typedSuccess") == "1" {
		respondJSON(w, http.StatusOK, map[string]any{
			"name":     "projects/stackyard/locations/us-central1/spanner_executor/sample",
			"service":  "spanner_executor",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}
	return false
}
