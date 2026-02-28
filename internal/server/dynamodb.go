package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	dynamodbsvc "github.com/stackyard/stackyard/internal/services/dynamodb"
)

type dynamodbError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleDynamoDBJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isDynamoDBJSONCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "dynamodb")
	if !ok {
		respondDynamoDBError(w, status, code, msg)
		return true
	}

	action := parseDynamoDBTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondDynamoDBError(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, known := dynamodbOperationByName[action]; !known {
		respondDynamoDBError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseDynamoDBPayload(r)
	if err != nil {
		respondDynamoDBError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	switch action {
	case "CreateTable":
		table, err := s.dynamodb.CreateTable(
			dynamodbString(payload["TableName"]),
			dynamodbAttrDefinitions(payload["AttributeDefinitions"]),
			dynamodbKeySchema(payload["KeySchema"]),
			dynamodbString(payload["BillingMode"]),
		)
		if err != nil {
			respondDynamoDBErrorForErr(w, err)
			return true
		}
		respondDynamoDBJSON(w, http.StatusOK, map[string]any{"TableDescription": table})
		return true

	case "DescribeTable":
		table, err := s.dynamodb.DescribeTable(dynamodbString(payload["TableName"]))
		if err != nil {
			respondDynamoDBErrorForErr(w, err)
			return true
		}
		respondDynamoDBJSON(w, http.StatusOK, map[string]any{"Table": table})
		return true

	case "ListTables":
		limit, ok := dynamodbInt(payload["Limit"])
		if !ok {
			respondDynamoDBError(w, http.StatusBadRequest, "ValidationException", "invalid Limit")
			return true
		}
		tables, lastEvaluated, err := s.dynamodb.ListTables(limit, dynamodbString(payload["ExclusiveStartTableName"]))
		if err != nil {
			respondDynamoDBErrorForErr(w, err)
			return true
		}
		response := map[string]any{"TableNames": tables}
		if strings.TrimSpace(lastEvaluated) != "" {
			response["LastEvaluatedTableName"] = lastEvaluated
		}
		respondDynamoDBJSON(w, http.StatusOK, response)
		return true

	case "UpdateTable":
		table, err := s.dynamodb.UpdateTable(dynamodbString(payload["TableName"]), dynamodbString(payload["BillingMode"]))
		if err != nil {
			respondDynamoDBErrorForErr(w, err)
			return true
		}
		respondDynamoDBJSON(w, http.StatusOK, map[string]any{"TableDescription": table})
		return true

	case "DeleteTable":
		table, err := s.dynamodb.DeleteTable(dynamodbString(payload["TableName"]))
		if err != nil {
			respondDynamoDBErrorForErr(w, err)
			return true
		}
		respondDynamoDBJSON(w, http.StatusOK, map[string]any{"TableDescription": table})
		return true

	case "DescribeLimits":
		respondDynamoDBJSON(w, http.StatusOK, s.dynamodb.DescribeLimits())
		return true

	case "PutItem":
		item, ok := dynamodbMap(payload["Item"])
		if !ok {
			respondDynamoDBError(w, http.StatusBadRequest, "ValidationException", "invalid Item")
			return true
		}
		attrs, err := s.dynamodb.PutItem(dynamodbString(payload["TableName"]), item, dynamodbString(payload["ReturnValues"]))
		if err != nil {
			respondDynamoDBErrorForErr(w, err)
			return true
		}
		response := map[string]any{}
		if len(attrs) > 0 {
			response["Attributes"] = attrs
		}
		respondDynamoDBJSON(w, http.StatusOK, response)
		return true

	case "GetItem":
		key, ok := dynamodbMap(payload["Key"])
		if !ok {
			respondDynamoDBError(w, http.StatusBadRequest, "ValidationException", "invalid Key")
			return true
		}
		item, err := s.dynamodb.GetItem(dynamodbString(payload["TableName"]), key)
		if err != nil {
			respondDynamoDBErrorForErr(w, err)
			return true
		}
		response := map[string]any{}
		if len(item) > 0 {
			response["Item"] = item
		}
		respondDynamoDBJSON(w, http.StatusOK, response)
		return true

	case "DeleteItem":
		key, ok := dynamodbMap(payload["Key"])
		if !ok {
			respondDynamoDBError(w, http.StatusBadRequest, "ValidationException", "invalid Key")
			return true
		}
		attrs, err := s.dynamodb.DeleteItem(dynamodbString(payload["TableName"]), key, dynamodbString(payload["ReturnValues"]))
		if err != nil {
			respondDynamoDBErrorForErr(w, err)
			return true
		}
		response := map[string]any{}
		if len(attrs) > 0 {
			response["Attributes"] = attrs
		}
		respondDynamoDBJSON(w, http.StatusOK, response)
		return true

	case "UpdateItem":
		key, ok := dynamodbMap(payload["Key"])
		if !ok {
			respondDynamoDBError(w, http.StatusBadRequest, "ValidationException", "invalid Key")
			return true
		}
		attrs, err := s.dynamodb.UpdateItem(
			dynamodbString(payload["TableName"]),
			key,
			dynamodbMapAny(payload["AttributeUpdates"]),
			dynamodbString(payload["UpdateExpression"]),
			dynamodbStringMap(payload["ExpressionAttributeNames"]),
			dynamodbMapAny(payload["ExpressionAttributeValues"]),
			dynamodbString(payload["ReturnValues"]),
		)
		if err != nil {
			respondDynamoDBErrorForErr(w, err)
			return true
		}
		response := map[string]any{}
		if len(attrs) > 0 {
			response["Attributes"] = attrs
		}
		respondDynamoDBJSON(w, http.StatusOK, response)
		return true

	case "BatchWriteItem":
		requestItems := dynamodbMapAny(payload["RequestItems"])
		if len(requestItems) == 0 {
			respondDynamoDBError(w, http.StatusBadRequest, "ValidationException", "invalid RequestItems")
			return true
		}
		_, err := s.dynamodb.BatchWriteItem(requestItems)
		if err != nil {
			respondDynamoDBErrorForErr(w, err)
			return true
		}
		respondDynamoDBJSON(w, http.StatusOK, map[string]any{"UnprocessedItems": map[string]any{}})
		return true

	case "BatchGetItem":
		requestItems := dynamodbMapAny(payload["RequestItems"])
		if len(requestItems) == 0 {
			respondDynamoDBError(w, http.StatusBadRequest, "ValidationException", "invalid RequestItems")
			return true
		}
		responses, err := s.dynamodb.BatchGetItem(requestItems)
		if err != nil {
			respondDynamoDBErrorForErr(w, err)
			return true
		}
		respondDynamoDBJSON(w, http.StatusOK, map[string]any{"Responses": responses, "UnprocessedKeys": map[string]any{}})
		return true

	case "Query":
		out, err := s.dynamodb.Query(dynamodbsvc.QueryInput{
			TableName:                 dynamodbString(payload["TableName"]),
			KeyConditionExpression:    dynamodbString(payload["KeyConditionExpression"]),
			KeyConditions:             dynamodbMapAny(payload["KeyConditions"]),
			ExpressionAttributeNames:  dynamodbStringMap(payload["ExpressionAttributeNames"]),
			ExpressionAttributeValues: dynamodbMapAny(payload["ExpressionAttributeValues"]),
			Limit:                     dynamodbIntValue(payload["Limit"]),
		})
		if err != nil {
			respondDynamoDBErrorForErr(w, err)
			return true
		}
		respondDynamoDBJSON(w, http.StatusOK, map[string]any{"Items": out.Items, "Count": out.Count, "ScannedCount": out.ScannedCount})
		return true

	case "Scan":
		out, err := s.dynamodb.Scan(dynamodbString(payload["TableName"]), dynamodbIntValue(payload["Limit"]))
		if err != nil {
			respondDynamoDBErrorForErr(w, err)
			return true
		}
		respondDynamoDBJSON(w, http.StatusOK, map[string]any{"Items": out.Items, "Count": out.Count, "ScannedCount": out.ScannedCount})
		return true

	case "ExecuteStatement":
		limit, ok := dynamodbInt(payload["Limit"])
		if !ok {
			respondDynamoDBError(w, http.StatusBadRequest, "ValidationException", "invalid Limit")
			return true
		}
		out, err := s.dynamodb.ExecuteStatement(dynamodbString(payload["Statement"]), dynamodbAnySlice(payload["Parameters"]), limit)
		if err != nil {
			respondDynamoDBErrorForErr(w, err)
			return true
		}
		respondDynamoDBJSON(w, http.StatusOK, map[string]any{"Items": out.Items, "Count": out.Count, "ScannedCount": out.ScannedCount})
		return true

	case "BatchExecuteStatement":
		statements, ok := dynamodbMapSlice(payload["Statements"])
		if !ok {
			respondDynamoDBError(w, http.StatusBadRequest, "ValidationException", "invalid Statements")
			return true
		}
		out, err := s.dynamodb.BatchExecuteStatement(statements)
		if err != nil {
			respondDynamoDBErrorForErr(w, err)
			return true
		}
		responses := make([]map[string]any, 0, len(out))
		for _, result := range out {
			responses = append(responses, map[string]any{
				"Items":        result.Items,
				"Count":        result.Count,
				"ScannedCount": result.ScannedCount,
			})
		}
		respondDynamoDBJSON(w, http.StatusOK, map[string]any{"Responses": responses})
		return true

	case "ExecuteTransaction":
		statements, ok := dynamodbMapSlice(payload["TransactStatements"])
		if !ok {
			respondDynamoDBError(w, http.StatusBadRequest, "ValidationException", "invalid TransactStatements")
			return true
		}
		out, err := s.dynamodb.ExecuteTransaction(statements)
		if err != nil {
			respondDynamoDBErrorForErr(w, err)
			return true
		}
		responses := make([]map[string]any, 0, len(out))
		for _, result := range out {
			responses = append(responses, map[string]any{
				"Items":        result.Items,
				"Count":        result.Count,
				"ScannedCount": result.ScannedCount,
			})
		}
		respondDynamoDBJSON(w, http.StatusOK, map[string]any{"Responses": responses})
		return true

	case "TransactGetItems":
		transactItems, ok := dynamodbMapSlice(payload["TransactItems"])
		if !ok {
			respondDynamoDBError(w, http.StatusBadRequest, "ValidationException", "invalid TransactItems")
			return true
		}
		out, err := s.dynamodb.TransactGetItems(transactItems)
		if err != nil {
			respondDynamoDBErrorForErr(w, err)
			return true
		}
		respondDynamoDBJSON(w, http.StatusOK, map[string]any{"Responses": out})
		return true

	case "TransactWriteItems":
		transactItems, ok := dynamodbMapSlice(payload["TransactItems"])
		if !ok {
			respondDynamoDBError(w, http.StatusBadRequest, "ValidationException", "invalid TransactItems")
			return true
		}
		if err := s.dynamodb.TransactWriteItems(transactItems); err != nil {
			respondDynamoDBErrorForErr(w, err)
			return true
		}
		respondDynamoDBJSON(w, http.StatusOK, map[string]any{})
		return true

	case "CreateBackup":
		backup, err := s.dynamodb.CreateBackup(dynamodbString(payload["TableName"]), dynamodbString(payload["BackupName"]))
		if err != nil {
			respondDynamoDBErrorForErr(w, err)
			return true
		}
		respondDynamoDBJSON(w, http.StatusOK, map[string]any{"BackupDetails": backup})
		return true

	case "DescribeBackup":
		backup, err := s.dynamodb.DescribeBackup(dynamodbString(payload["BackupArn"]))
		if err != nil {
			respondDynamoDBErrorForErr(w, err)
			return true
		}
		respondDynamoDBJSON(w, http.StatusOK, map[string]any{"BackupDescription": map[string]any{"BackupDetails": backup}})
		return true

	case "DeleteBackup":
		backup, err := s.dynamodb.DeleteBackup(dynamodbString(payload["BackupArn"]))
		if err != nil {
			respondDynamoDBErrorForErr(w, err)
			return true
		}
		respondDynamoDBJSON(w, http.StatusOK, map[string]any{"BackupDescription": map[string]any{"BackupDetails": backup}})
		return true

	case "ListBackups":
		limit, ok := dynamodbInt(payload["Limit"])
		if !ok {
			respondDynamoDBError(w, http.StatusBadRequest, "ValidationException", "invalid Limit")
			return true
		}
		backups, lastEvaluated, err := s.dynamodb.ListBackups(
			dynamodbString(payload["TableName"]),
			limit,
			dynamodbString(payload["ExclusiveStartBackupArn"]),
		)
		if err != nil {
			respondDynamoDBErrorForErr(w, err)
			return true
		}
		response := map[string]any{"BackupSummaries": backups}
		if strings.TrimSpace(lastEvaluated) != "" {
			response["LastEvaluatedBackupArn"] = lastEvaluated
		}
		respondDynamoDBJSON(w, http.StatusOK, response)
		return true

	case "RestoreTableFromBackup":
		table, err := s.dynamodb.RestoreTableFromBackup(
			dynamodbString(payload["TargetTableName"]),
			dynamodbString(payload["BackupArn"]),
		)
		if err != nil {
			respondDynamoDBErrorForErr(w, err)
			return true
		}
		respondDynamoDBJSON(w, http.StatusOK, map[string]any{"TableDescription": table})
		return true

	case "RestoreTableToPointInTime":
		table, err := s.dynamodb.RestoreTableToPointInTime(
			dynamodbString(payload["SourceTableName"]),
			dynamodbString(payload["TargetTableName"]),
		)
		if err != nil {
			respondDynamoDBErrorForErr(w, err)
			return true
		}
		respondDynamoDBJSON(w, http.StatusOK, map[string]any{"TableDescription": table})
		return true

	case "CreateGlobalTable":
		table, err := s.dynamodb.CreateGlobalTable(
			dynamodbString(payload["GlobalTableName"]),
			dynamodbReplicationGroupRegions(payload["ReplicationGroup"]),
		)
		if err != nil {
			respondDynamoDBErrorForErr(w, err)
			return true
		}
		respondDynamoDBJSON(w, http.StatusOK, map[string]any{"GlobalTableDescription": table})
		return true

	case "DescribeGlobalTable":
		table, err := s.dynamodb.DescribeGlobalTable(dynamodbString(payload["GlobalTableName"]))
		if err != nil {
			respondDynamoDBErrorForErr(w, err)
			return true
		}
		respondDynamoDBJSON(w, http.StatusOK, map[string]any{"GlobalTableDescription": table})
		return true

	case "ListGlobalTables":
		limit, ok := dynamodbInt(payload["Limit"])
		if !ok {
			respondDynamoDBError(w, http.StatusBadRequest, "ValidationException", "invalid Limit")
			return true
		}
		tables, lastEvaluated, err := s.dynamodb.ListGlobalTables(limit, dynamodbString(payload["ExclusiveStartGlobalTableName"]))
		if err != nil {
			respondDynamoDBErrorForErr(w, err)
			return true
		}
		response := map[string]any{"GlobalTables": tables}
		if strings.TrimSpace(lastEvaluated) != "" {
			response["LastEvaluatedGlobalTableName"] = lastEvaluated
		}
		respondDynamoDBJSON(w, http.StatusOK, response)
		return true

	case "UpdateGlobalTable":
		addRegions, deleteRegions, ok := dynamodbReplicaUpdateRegions(payload["ReplicaUpdates"])
		if !ok {
			respondDynamoDBError(w, http.StatusBadRequest, "ValidationException", "invalid ReplicaUpdates")
			return true
		}
		table, err := s.dynamodb.UpdateGlobalTable(dynamodbString(payload["GlobalTableName"]), addRegions, deleteRegions)
		if err != nil {
			respondDynamoDBErrorForErr(w, err)
			return true
		}
		respondDynamoDBJSON(w, http.StatusOK, map[string]any{"GlobalTableDescription": table})
		return true

	case "DescribeGlobalTableSettings":
		settings, err := s.dynamodb.DescribeGlobalTableSettings(dynamodbString(payload["GlobalTableName"]))
		if err != nil {
			respondDynamoDBErrorForErr(w, err)
			return true
		}
		respondDynamoDBJSON(w, http.StatusOK, map[string]any{"GlobalTableSettingsDescription": settings})
		return true

	case "UpdateGlobalTableSettings":
		replicaSettings, ok := dynamodbMapSlice(payload["ReplicaSettingsUpdate"])
		if !ok {
			respondDynamoDBError(w, http.StatusBadRequest, "ValidationException", "invalid ReplicaSettingsUpdate")
			return true
		}
		settings, err := s.dynamodb.UpdateGlobalTableSettings(dynamodbString(payload["GlobalTableName"]), replicaSettings)
		if err != nil {
			respondDynamoDBErrorForErr(w, err)
			return true
		}
		respondDynamoDBJSON(w, http.StatusOK, map[string]any{"GlobalTableSettingsDescription": settings})
		return true

	case "DescribeTableReplicaAutoScaling":
		description, err := s.dynamodb.DescribeTableReplicaAutoScaling(dynamodbString(payload["TableName"]))
		if err != nil {
			respondDynamoDBErrorForErr(w, err)
			return true
		}
		respondDynamoDBJSON(w, http.StatusOK, map[string]any{"TableAutoScalingDescription": description})
		return true

	case "UpdateTableReplicaAutoScaling":
		tableName := dynamodbString(payload["TableName"])
		updates := dynamodbMapAny(payload)
		delete(updates, "TableName")
		description, err := s.dynamodb.UpdateTableReplicaAutoScaling(tableName, updates)
		if err != nil {
			respondDynamoDBErrorForErr(w, err)
			return true
		}
		respondDynamoDBJSON(w, http.StatusOK, map[string]any{"TableAutoScalingDescription": description})
		return true

	case "ExportTableToPointInTime":
		exportDescription, err := s.dynamodb.ExportTableToPointInTime(
			dynamodbString(payload["TableArn"]),
			dynamodbString(payload["S3Bucket"]),
			dynamodbString(payload["ExportFormat"]),
			dynamodbString(payload["ClientToken"]),
		)
		if err != nil {
			respondDynamoDBErrorForErr(w, err)
			return true
		}
		respondDynamoDBJSON(w, http.StatusOK, map[string]any{"ExportDescription": exportDescription})
		return true

	case "DescribeExport":
		exportDescription, err := s.dynamodb.DescribeExport(dynamodbString(payload["ExportArn"]))
		if err != nil {
			respondDynamoDBErrorForErr(w, err)
			return true
		}
		respondDynamoDBJSON(w, http.StatusOK, map[string]any{"ExportDescription": exportDescription})
		return true

	case "ListExports":
		limit, ok := dynamodbInt(payload["Limit"])
		if !ok {
			respondDynamoDBError(w, http.StatusBadRequest, "ValidationException", "invalid Limit")
			return true
		}
		exports, lastEvaluated, err := s.dynamodb.ListExports(
			dynamodbString(payload["TableArn"]),
			limit,
			dynamodbString(payload["ExclusiveStartExportArn"]),
		)
		if err != nil {
			respondDynamoDBErrorForErr(w, err)
			return true
		}
		response := map[string]any{"ExportSummaries": exports}
		if strings.TrimSpace(lastEvaluated) != "" {
			response["LastEvaluatedExportArn"] = lastEvaluated
		}
		respondDynamoDBJSON(w, http.StatusOK, response)
		return true

	case "ImportTable":
		importDescription, err := s.dynamodb.ImportTable(
			dynamodbImportTableName(payload["TableCreationParameters"]),
			dynamodbString(payload["InputFormat"]),
			dynamodbString(payload["ClientToken"]),
		)
		if err != nil {
			respondDynamoDBErrorForErr(w, err)
			return true
		}
		respondDynamoDBJSON(w, http.StatusOK, map[string]any{"ImportTableDescription": importDescription})
		return true

	case "DescribeImport":
		importDescription, err := s.dynamodb.DescribeImport(dynamodbString(payload["ImportArn"]))
		if err != nil {
			respondDynamoDBErrorForErr(w, err)
			return true
		}
		respondDynamoDBJSON(w, http.StatusOK, map[string]any{"ImportTableDescription": importDescription})
		return true

	case "ListImports":
		limit, ok := dynamodbInt(payload["Limit"])
		if !ok {
			respondDynamoDBError(w, http.StatusBadRequest, "ValidationException", "invalid Limit")
			return true
		}
		imports, lastEvaluated, err := s.dynamodb.ListImports(
			dynamodbString(payload["TableArn"]),
			limit,
			dynamodbString(payload["ExclusiveStartImportArn"]),
		)
		if err != nil {
			respondDynamoDBErrorForErr(w, err)
			return true
		}
		response := map[string]any{"ImportSummaryList": imports}
		if strings.TrimSpace(lastEvaluated) != "" {
			response["LastEvaluatedImportArn"] = lastEvaluated
		}
		respondDynamoDBJSON(w, http.StatusOK, response)
		return true

	case "EnableKinesisStreamingDestination":
		description, err := s.dynamodb.EnableKinesisStreamingDestination(
			dynamodbString(payload["TableName"]),
			dynamodbString(payload["StreamArn"]),
		)
		if err != nil {
			respondDynamoDBErrorForErr(w, err)
			return true
		}
		respondDynamoDBJSON(w, http.StatusOK, map[string]any{"DestinationStatus": description["DestinationStatus"], "TableName": description["TableName"], "StreamArn": description["StreamArn"]})
		return true

	case "DisableKinesisStreamingDestination":
		description, err := s.dynamodb.DisableKinesisStreamingDestination(
			dynamodbString(payload["TableName"]),
			dynamodbString(payload["StreamArn"]),
		)
		if err != nil {
			respondDynamoDBErrorForErr(w, err)
			return true
		}
		respondDynamoDBJSON(w, http.StatusOK, map[string]any{"DestinationStatus": description["DestinationStatus"], "TableName": description["TableName"], "StreamArn": description["StreamArn"]})
		return true

	case "UpdateKinesisStreamingDestination":
		description, err := s.dynamodb.UpdateKinesisStreamingDestination(
			dynamodbString(payload["TableName"]),
			dynamodbString(payload["StreamArn"]),
		)
		if err != nil {
			respondDynamoDBErrorForErr(w, err)
			return true
		}
		respondDynamoDBJSON(w, http.StatusOK, map[string]any{"DestinationStatus": description["DestinationStatus"], "TableName": description["TableName"], "StreamArn": description["StreamArn"]})
		return true

	case "DescribeKinesisStreamingDestination":
		description, err := s.dynamodb.DescribeKinesisStreamingDestination(dynamodbString(payload["TableName"]))
		if err != nil {
			respondDynamoDBErrorForErr(w, err)
			return true
		}
		respondDynamoDBJSON(w, http.StatusOK, description)
		return true

	case "DescribeEndpoints":
		respondDynamoDBJSON(w, http.StatusOK, s.dynamodb.DescribeEndpoints())
		return true

	case "UpdateContinuousBackups":
		description, err := s.dynamodb.UpdateContinuousBackups(
			dynamodbString(payload["TableName"]),
			dynamodbPointInTimeEnabled(payload["PointInTimeRecoverySpecification"]),
		)
		if err != nil {
			respondDynamoDBErrorForErr(w, err)
			return true
		}
		respondDynamoDBJSON(w, http.StatusOK, map[string]any{"ContinuousBackupsDescription": description})
		return true

	case "DescribeContinuousBackups":
		description, err := s.dynamodb.DescribeContinuousBackups(dynamodbString(payload["TableName"]))
		if err != nil {
			respondDynamoDBErrorForErr(w, err)
			return true
		}
		respondDynamoDBJSON(w, http.StatusOK, map[string]any{"ContinuousBackupsDescription": description})
		return true

	case "UpdateTimeToLive":
		spec, _ := payload["TimeToLiveSpecification"].(map[string]any)
		description, err := s.dynamodb.UpdateTimeToLive(
			dynamodbString(payload["TableName"]),
			dynamodbString(spec["AttributeName"]),
			dynamodbBool(spec["Enabled"]),
		)
		if err != nil {
			respondDynamoDBErrorForErr(w, err)
			return true
		}
		respondDynamoDBJSON(w, http.StatusOK, map[string]any{"TimeToLiveSpecification": description})
		return true

	case "DescribeTimeToLive":
		description, err := s.dynamodb.DescribeTimeToLive(dynamodbString(payload["TableName"]))
		if err != nil {
			respondDynamoDBErrorForErr(w, err)
			return true
		}
		respondDynamoDBJSON(w, http.StatusOK, map[string]any{"TimeToLiveDescription": description})
		return true

	case "UpdateContributorInsights":
		description, err := s.dynamodb.UpdateContributorInsights(
			dynamodbString(payload["TableName"]),
			dynamodbString(payload["IndexName"]),
			strings.EqualFold(dynamodbString(payload["ContributorInsightsAction"]), "ENABLE"),
		)
		if err != nil {
			respondDynamoDBErrorForErr(w, err)
			return true
		}
		respondDynamoDBJSON(w, http.StatusOK, map[string]any{"ContributorInsightsStatus": description["ContributorInsightsStatus"], "TableName": description["TableName"], "IndexName": description["IndexName"]})
		return true

	case "DescribeContributorInsights":
		description, err := s.dynamodb.DescribeContributorInsights(dynamodbString(payload["TableName"]), dynamodbString(payload["IndexName"]))
		if err != nil {
			respondDynamoDBErrorForErr(w, err)
			return true
		}
		respondDynamoDBJSON(w, http.StatusOK, description)
		return true

	case "ListContributorInsights":
		summaries, err := s.dynamodb.ListContributorInsights(dynamodbString(payload["TableName"]))
		if err != nil {
			respondDynamoDBErrorForErr(w, err)
			return true
		}
		respondDynamoDBJSON(w, http.StatusOK, map[string]any{"ContributorInsightsSummaries": summaries})
		return true

	case "TagResource":
		if err := s.dynamodb.TagResource(dynamodbString(payload["ResourceArn"]), dynamodbTagList(payload["Tags"])); err != nil {
			respondDynamoDBErrorForErr(w, err)
			return true
		}
		respondDynamoDBJSON(w, http.StatusOK, map[string]any{})
		return true

	case "UntagResource":
		if err := s.dynamodb.UntagResource(dynamodbString(payload["ResourceArn"]), dynamodbStringSlice(payload["TagKeys"])); err != nil {
			respondDynamoDBErrorForErr(w, err)
			return true
		}
		respondDynamoDBJSON(w, http.StatusOK, map[string]any{})
		return true

	case "ListTagsOfResource":
		tags, err := s.dynamodb.ListTagsOfResource(dynamodbString(payload["ResourceArn"]))
		if err != nil {
			respondDynamoDBErrorForErr(w, err)
			return true
		}
		respondDynamoDBJSON(w, http.StatusOK, map[string]any{"Tags": tags})
		return true

	case "PutResourcePolicy":
		revisionID, err := s.dynamodb.PutResourcePolicy(
			dynamodbString(payload["ResourceArn"]),
			dynamodbString(payload["Policy"]),
			dynamodbString(payload["ExpectedRevisionId"]),
		)
		if err != nil {
			respondDynamoDBErrorForErr(w, err)
			return true
		}
		respondDynamoDBJSON(w, http.StatusOK, map[string]any{"RevisionId": revisionID})
		return true

	case "GetResourcePolicy":
		policy, revisionID, err := s.dynamodb.GetResourcePolicy(dynamodbString(payload["ResourceArn"]))
		if err != nil {
			respondDynamoDBErrorForErr(w, err)
			return true
		}
		respondDynamoDBJSON(w, http.StatusOK, map[string]any{"Policy": policy, "RevisionId": revisionID})
		return true

	case "DeleteResourcePolicy":
		revisionID, err := s.dynamodb.DeleteResourcePolicy(
			dynamodbString(payload["ResourceArn"]),
			dynamodbString(payload["ExpectedRevisionId"]),
		)
		if err != nil {
			respondDynamoDBErrorForErr(w, err)
			return true
		}
		respondDynamoDBJSON(w, http.StatusOK, map[string]any{"RevisionId": revisionID})
		return true
	}

	respondDynamoDBError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
	return true
}

func isDynamoDBJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.HasPrefix(target, "DynamoDB_20120810.") {
		return true
	}
	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json") {
		return strings.TrimSpace(sigV4ServiceHint(r)) == "dynamodb"
	}
	return false
}

func parseDynamoDBTarget(target string) string {
	if target == "" {
		return ""
	}
	parts := strings.Split(target, ".")
	if len(parts) != 2 {
		return ""
	}
	if strings.TrimSpace(parts[0]) != "DynamoDB_20120810" {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func parseDynamoDBPayload(r *http.Request) (map[string]any, error) {
	bodyBytes, err := readBodyBytes(r)
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(bodyBytes)) == 0 {
		return map[string]any{}, nil
	}
	var out map[string]any
	if err := json.Unmarshal(bodyBytes, &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = map[string]any{}
	}
	return out, nil
}

func respondDynamoDBJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.0")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondDynamoDBError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondDynamoDBJSON(w, status, dynamodbError{Type: code, Message: msg})
}

func respondDynamoDBErrorForErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, dynamodbsvc.ErrValidation):
		respondDynamoDBError(w, http.StatusBadRequest, "ValidationException", err.Error())
	case errors.Is(err, dynamodbsvc.ErrResourceNotFound):
		respondDynamoDBError(w, http.StatusBadRequest, "ResourceNotFoundException", err.Error())
	case errors.Is(err, dynamodbsvc.ErrResourceInUse):
		respondDynamoDBError(w, http.StatusBadRequest, "ResourceInUseException", err.Error())
	case errors.Is(err, dynamodbsvc.ErrConditionalCheckFailed):
		respondDynamoDBError(w, http.StatusBadRequest, "ConditionalCheckFailedException", err.Error())
	default:
		respondDynamoDBError(w, http.StatusInternalServerError, "InternalServerError", err.Error())
	}
}

func dynamodbString(v any) string {
	s, _ := v.(string)
	return strings.TrimSpace(s)
}

func dynamodbInt(v any) (int, bool) {
	switch value := v.(type) {
	case nil:
		return 0, true
	case float64:
		return int(value), value == float64(int(value))
	case int:
		return value, true
	case string:
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return 0, true
		}
		n, err := strconv.Atoi(trimmed)
		if err != nil {
			return 0, false
		}
		return n, true
	default:
		return 0, false
	}
}

func dynamodbIntValue(v any) int {
	n, _ := dynamodbInt(v)
	return n
}

func dynamodbMap(v any) (map[string]any, bool) {
	m, ok := v.(map[string]any)
	if !ok {
		return nil, false
	}
	return m, true
}

func dynamodbMapAny(v any) map[string]any {
	m, _ := v.(map[string]any)
	if m == nil {
		return map[string]any{}
	}
	return m
}

func dynamodbMapSlice(v any) ([]map[string]any, bool) {
	if v == nil {
		return nil, true
	}
	items, ok := v.([]any)
	if !ok {
		return nil, false
	}
	out := make([]map[string]any, 0, len(items))
	for _, raw := range items {
		m, ok := raw.(map[string]any)
		if !ok {
			return nil, false
		}
		out = append(out, m)
	}
	return out, true
}

func dynamodbAnySlice(v any) []any {
	items, _ := v.([]any)
	if items == nil {
		return nil
	}
	return items
}

func dynamodbReplicationGroupRegions(v any) []string {
	items, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, raw := range items {
		m, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if region := dynamodbString(m["RegionName"]); region != "" {
			out = append(out, region)
		}
	}
	return out
}

func dynamodbImportTableName(v any) string {
	m, ok := v.(map[string]any)
	if !ok {
		return ""
	}
	return dynamodbString(m["TableName"])
}

func dynamodbPointInTimeEnabled(v any) bool {
	m, ok := v.(map[string]any)
	if !ok {
		return false
	}
	return dynamodbBool(m["PointInTimeRecoveryEnabled"])
}

func dynamodbBool(v any) bool {
	switch value := v.(type) {
	case bool:
		return value
	case string:
		return strings.EqualFold(strings.TrimSpace(value), "true")
	default:
		return false
	}
}

func dynamodbTagList(v any) []map[string]string {
	items, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]map[string]string, 0, len(items))
	for _, raw := range items {
		m, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		key := dynamodbString(m["Key"])
		if key == "" {
			continue
		}
		out = append(out, map[string]string{
			"Key":   key,
			"Value": dynamodbString(m["Value"]),
		})
	}
	return out
}

func dynamodbStringSlice(v any) []string {
	items, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, raw := range items {
		if s, ok := raw.(string); ok {
			trimmed := strings.TrimSpace(s)
			if trimmed != "" {
				out = append(out, trimmed)
			}
		}
	}
	return out
}

func dynamodbReplicaUpdateRegions(v any) ([]string, []string, bool) {
	items, ok := dynamodbMapSlice(v)
	if !ok {
		return nil, nil, false
	}
	addRegions := make([]string, 0)
	deleteRegions := make([]string, 0)
	for _, item := range items {
		if createReq, ok := item["Create"].(map[string]any); ok {
			if region := dynamodbString(createReq["RegionName"]); region != "" {
				addRegions = append(addRegions, region)
			}
		}
		if deleteReq, ok := item["Delete"].(map[string]any); ok {
			if region := dynamodbString(deleteReq["RegionName"]); region != "" {
				deleteRegions = append(deleteRegions, region)
			}
		}
	}
	return addRegions, deleteRegions, true
}

func dynamodbStringMap(v any) map[string]string {
	m, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, raw := range m {
		s, _ := raw.(string)
		if strings.TrimSpace(k) == "" {
			continue
		}
		out[k] = strings.TrimSpace(s)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func dynamodbAttrDefinitions(v any) []dynamodbsvc.AttributeDefinition {
	items, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]dynamodbsvc.AttributeDefinition, 0, len(items))
	for _, raw := range items {
		m, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		name := dynamodbString(m["AttributeName"])
		typ := dynamodbString(m["AttributeType"])
		if name == "" || typ == "" {
			continue
		}
		out = append(out, dynamodbsvc.AttributeDefinition{AttributeName: name, AttributeType: typ})
	}
	return out
}

func dynamodbKeySchema(v any) []dynamodbsvc.KeySchemaElement {
	items, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]dynamodbsvc.KeySchemaElement, 0, len(items))
	for _, raw := range items {
		m, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		name := dynamodbString(m["AttributeName"])
		typ := strings.ToUpper(dynamodbString(m["KeyType"]))
		if name == "" || typ == "" {
			continue
		}
		out = append(out, dynamodbsvc.KeySchemaElement{AttributeName: name, KeyType: typ})
	}
	return out
}
