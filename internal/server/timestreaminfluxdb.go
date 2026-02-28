package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	timestreamsvc "github.com/stackyard/stackyard/internal/services/timestreaminfluxdb"
)

type timestreamInfluxDBError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleTimestreamInfluxDBJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isTimestreamInfluxDBJSONCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "timestream-influxdb")
	if !ok {
		respondTimestreamInfluxDBError(w, status, code, msg)
		return true
	}

	action := parseTimestreamInfluxDBTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondTimestreamInfluxDBError(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, known := timestreamInfluxDBOperationByName[action]; !known {
		respondTimestreamInfluxDBError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseTimestreamInfluxDBPayload(r)
	if err != nil {
		respondTimestreamInfluxDBError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	switch action {
	// Stage 2: cluster lifecycle.
	case "CreateDbCluster":
		vpcSubnetIDs, ok := timestreamInfluxDBStringSlice(timestreamInfluxDBFieldValue(payload, "vpcSubnetIds"))
		if !ok {
			respondTimestreamInfluxDBError(w, http.StatusBadRequest, "ValidationException", "invalid vpcSubnetIds")
			return true
		}
		vpcSecurityGroupIDs, ok := timestreamInfluxDBStringSlice(timestreamInfluxDBFieldValue(payload, "vpcSecurityGroupIds"))
		if !ok {
			respondTimestreamInfluxDBError(w, http.StatusBadRequest, "ValidationException", "invalid vpcSecurityGroupIds")
			return true
		}
		tags, ok := timestreamInfluxDBTags(timestreamInfluxDBFieldValue(payload, "tags"))
		if !ok {
			respondTimestreamInfluxDBError(w, http.StatusBadRequest, "ValidationException", "invalid tags")
			return true
		}

		createInput := timestreamsvc.CreateDbClusterInput{
			Name:                       timestreamInfluxDBString(timestreamInfluxDBFieldValue(payload, "name")),
			DbInstanceType:             timestreamInfluxDBString(timestreamInfluxDBFieldValue(payload, "dbInstanceType")),
			VpcSubnetIds:               vpcSubnetIDs,
			VpcSecurityGroupIds:        vpcSecurityGroupIDs,
			DbParameterGroupIdentifier: timestreamInfluxDBString(timestreamInfluxDBFieldValue(payload, "dbParameterGroupIdentifier")),
			DeploymentType:             timestreamInfluxDBString(timestreamInfluxDBFieldValue(payload, "deploymentType")),
			NetworkType:                timestreamInfluxDBString(timestreamInfluxDBFieldValue(payload, "networkType")),
			DbStorageType:              timestreamInfluxDBString(timestreamInfluxDBFieldValue(payload, "dbStorageType")),
			EngineType:                 timestreamInfluxDBString(timestreamInfluxDBFieldValue(payload, "engineType")),
			FailoverMode:               timestreamInfluxDBString(timestreamInfluxDBFieldValue(payload, "failoverMode")),
			Tags:                       tags,
		}
		if portRaw, ok := timestreamInfluxDBField(payload, "port"); ok {
			port, ok := timestreamInfluxDBInt(portRaw)
			if !ok {
				respondTimestreamInfluxDBError(w, http.StatusBadRequest, "ValidationException", "invalid port")
				return true
			}
			createInput.Port = port
		}
		if allocatedStorageRaw, ok := timestreamInfluxDBField(payload, "allocatedStorage"); ok {
			allocatedStorage, ok := timestreamInfluxDBInt(allocatedStorageRaw)
			if !ok {
				respondTimestreamInfluxDBError(w, http.StatusBadRequest, "ValidationException", "invalid allocatedStorage")
				return true
			}
			createInput.AllocatedStorage = allocatedStorage
		}
		if publiclyAccessibleRaw, ok := timestreamInfluxDBField(payload, "publiclyAccessible"); ok {
			value, ok := timestreamInfluxDBBool(publiclyAccessibleRaw)
			if !ok {
				respondTimestreamInfluxDBError(w, http.StatusBadRequest, "ValidationException", "invalid publiclyAccessible")
				return true
			}
			createInput.PubliclyAccessible = &value
		}
		if configRaw, ok := timestreamInfluxDBField(payload, "logDeliveryConfiguration"); ok {
			config, ok := timestreamInfluxDBMap(configRaw)
			if !ok {
				respondTimestreamInfluxDBError(w, http.StatusBadRequest, "ValidationException", "invalid logDeliveryConfiguration")
				return true
			}
			createInput.LogDeliveryConfiguration = config
		}

		cluster, status, err := s.timestream.CreateDbCluster(createInput)
		if err != nil {
			respondTimestreamInfluxDBErrorForErr(w, err)
			return true
		}
		respondTimestreamInfluxDBJSON(w, http.StatusOK, map[string]any{
			"dbClusterId":     cluster.ID,
			"dbClusterStatus": status,
		})
		return true

	case "UpdateDbCluster":
		updateInput := timestreamsvc.UpdateDbClusterInput{
			DbClusterID: timestreamInfluxDBString(timestreamInfluxDBFieldValue(payload, "dbClusterId")),
		}
		if field, ok := timestreamInfluxDBField(payload, "dbParameterGroupIdentifier"); ok {
			value := timestreamInfluxDBString(field)
			updateInput.DbParameterGroupIdentifier = &value
		}
		if field, ok := timestreamInfluxDBField(payload, "port"); ok {
			value, ok := timestreamInfluxDBInt(field)
			if !ok {
				respondTimestreamInfluxDBError(w, http.StatusBadRequest, "ValidationException", "invalid port")
				return true
			}
			updateInput.Port = &value
		}
		if field, ok := timestreamInfluxDBField(payload, "dbInstanceType"); ok {
			value := timestreamInfluxDBString(field)
			updateInput.DbInstanceType = &value
		}
		if field, ok := timestreamInfluxDBField(payload, "failoverMode"); ok {
			value := timestreamInfluxDBString(field)
			updateInput.FailoverMode = &value
		}
		if field, ok := timestreamInfluxDBField(payload, "logDeliveryConfiguration"); ok {
			value, ok := timestreamInfluxDBMap(field)
			if !ok {
				respondTimestreamInfluxDBError(w, http.StatusBadRequest, "ValidationException", "invalid logDeliveryConfiguration")
				return true
			}
			updateInput.LogDeliveryConfiguration = value
			updateInput.LogDeliveryConfigurationSet = true
		}

		status, err := s.timestream.UpdateDbCluster(updateInput)
		if err != nil {
			respondTimestreamInfluxDBErrorForErr(w, err)
			return true
		}
		respondTimestreamInfluxDBJSON(w, http.StatusOK, map[string]any{
			"dbClusterStatus": status,
		})
		return true

	case "RebootDbCluster":
		instanceIDs, ok := timestreamInfluxDBStringSlice(timestreamInfluxDBFieldValue(payload, "instanceIds"))
		if !ok {
			respondTimestreamInfluxDBError(w, http.StatusBadRequest, "ValidationException", "invalid instanceIds")
			return true
		}
		status, err := s.timestream.RebootDbCluster(
			timestreamInfluxDBString(timestreamInfluxDBFieldValue(payload, "dbClusterId")),
			instanceIDs,
		)
		if err != nil {
			respondTimestreamInfluxDBErrorForErr(w, err)
			return true
		}
		respondTimestreamInfluxDBJSON(w, http.StatusOK, map[string]any{
			"dbClusterStatus": status,
		})
		return true

	case "DeleteDbCluster":
		status, err := s.timestream.DeleteDbCluster(timestreamInfluxDBString(timestreamInfluxDBFieldValue(payload, "dbClusterId")))
		if err != nil {
			respondTimestreamInfluxDBErrorForErr(w, err)
			return true
		}
		respondTimestreamInfluxDBJSON(w, http.StatusOK, map[string]any{
			"dbClusterStatus": status,
		})
		return true

	// Stage 3: instance lifecycle.
	case "CreateDbInstance":
		vpcSubnetIDs, ok := timestreamInfluxDBStringSlice(timestreamInfluxDBFieldValue(payload, "vpcSubnetIds"))
		if !ok {
			respondTimestreamInfluxDBError(w, http.StatusBadRequest, "ValidationException", "invalid vpcSubnetIds")
			return true
		}
		vpcSecurityGroupIDs, ok := timestreamInfluxDBStringSlice(timestreamInfluxDBFieldValue(payload, "vpcSecurityGroupIds"))
		if !ok {
			respondTimestreamInfluxDBError(w, http.StatusBadRequest, "ValidationException", "invalid vpcSecurityGroupIds")
			return true
		}
		tags, ok := timestreamInfluxDBTags(timestreamInfluxDBFieldValue(payload, "tags"))
		if !ok {
			respondTimestreamInfluxDBError(w, http.StatusBadRequest, "ValidationException", "invalid tags")
			return true
		}

		createInput := timestreamsvc.CreateDbInstanceInput{
			Name:                       timestreamInfluxDBString(timestreamInfluxDBFieldValue(payload, "name")),
			Password:                   timestreamInfluxDBString(timestreamInfluxDBFieldValue(payload, "password")),
			DbInstanceType:             timestreamInfluxDBString(timestreamInfluxDBFieldValue(payload, "dbInstanceType")),
			VpcSubnetIds:               vpcSubnetIDs,
			VpcSecurityGroupIds:        vpcSecurityGroupIDs,
			DbParameterGroupIdentifier: timestreamInfluxDBString(timestreamInfluxDBFieldValue(payload, "dbParameterGroupIdentifier")),
			DeploymentType:             timestreamInfluxDBString(timestreamInfluxDBFieldValue(payload, "deploymentType")),
			NetworkType:                timestreamInfluxDBString(timestreamInfluxDBFieldValue(payload, "networkType")),
			DbStorageType:              timestreamInfluxDBString(timestreamInfluxDBFieldValue(payload, "dbStorageType")),
			Tags:                       tags,
		}
		if field, ok := timestreamInfluxDBField(payload, "allocatedStorage"); ok {
			value, ok := timestreamInfluxDBInt(field)
			if !ok {
				respondTimestreamInfluxDBError(w, http.StatusBadRequest, "ValidationException", "invalid allocatedStorage")
				return true
			}
			createInput.AllocatedStorage = value
		}
		if field, ok := timestreamInfluxDBField(payload, "port"); ok {
			value, ok := timestreamInfluxDBInt(field)
			if !ok {
				respondTimestreamInfluxDBError(w, http.StatusBadRequest, "ValidationException", "invalid port")
				return true
			}
			createInput.Port = value
		}
		if field, ok := timestreamInfluxDBField(payload, "publiclyAccessible"); ok {
			value, ok := timestreamInfluxDBBool(field)
			if !ok {
				respondTimestreamInfluxDBError(w, http.StatusBadRequest, "ValidationException", "invalid publiclyAccessible")
				return true
			}
			createInput.PubliclyAccessible = &value
		}
		if field, ok := timestreamInfluxDBField(payload, "logDeliveryConfiguration"); ok {
			value, ok := timestreamInfluxDBMap(field)
			if !ok {
				respondTimestreamInfluxDBError(w, http.StatusBadRequest, "ValidationException", "invalid logDeliveryConfiguration")
				return true
			}
			createInput.LogDeliveryConfiguration = value
		}

		instance, err := s.timestream.CreateDbInstance(createInput)
		if err != nil {
			respondTimestreamInfluxDBErrorForErr(w, err)
			return true
		}
		respondTimestreamInfluxDBJSON(w, http.StatusOK, timestreamInfluxDBInstanceToAPI(instance))
		return true

	case "UpdateDbInstance":
		updateInput := timestreamsvc.UpdateDbInstanceInput{
			Identifier: timestreamInfluxDBString(timestreamInfluxDBFieldValue(payload, "identifier")),
		}
		if field, ok := timestreamInfluxDBField(payload, "dbInstanceType"); ok {
			value := timestreamInfluxDBString(field)
			updateInput.DbInstanceType = &value
		}
		if field, ok := timestreamInfluxDBField(payload, "allocatedStorage"); ok {
			value, ok := timestreamInfluxDBInt(field)
			if !ok {
				respondTimestreamInfluxDBError(w, http.StatusBadRequest, "ValidationException", "invalid allocatedStorage")
				return true
			}
			updateInput.AllocatedStorage = &value
		}
		if field, ok := timestreamInfluxDBField(payload, "port"); ok {
			value, ok := timestreamInfluxDBInt(field)
			if !ok {
				respondTimestreamInfluxDBError(w, http.StatusBadRequest, "ValidationException", "invalid port")
				return true
			}
			updateInput.Port = &value
		}
		if field, ok := timestreamInfluxDBField(payload, "dbParameterGroupIdentifier"); ok {
			value := timestreamInfluxDBString(field)
			updateInput.DbParameterGroupIdentifier = &value
		}
		if field, ok := timestreamInfluxDBField(payload, "deploymentType"); ok {
			value := timestreamInfluxDBString(field)
			updateInput.DeploymentType = &value
		}
		if field, ok := timestreamInfluxDBField(payload, "dbStorageType"); ok {
			value := timestreamInfluxDBString(field)
			updateInput.DbStorageType = &value
		}
		if field, ok := timestreamInfluxDBField(payload, "logDeliveryConfiguration"); ok {
			value, ok := timestreamInfluxDBMap(field)
			if !ok {
				respondTimestreamInfluxDBError(w, http.StatusBadRequest, "ValidationException", "invalid logDeliveryConfiguration")
				return true
			}
			updateInput.LogDeliveryConfiguration = value
			updateInput.LogDeliveryConfigurationSet = true
		}

		instance, err := s.timestream.UpdateDbInstance(updateInput)
		if err != nil {
			respondTimestreamInfluxDBErrorForErr(w, err)
			return true
		}
		respondTimestreamInfluxDBJSON(w, http.StatusOK, timestreamInfluxDBInstanceToAPI(instance))
		return true

	case "RebootDbInstance":
		instance, err := s.timestream.RebootDbInstance(timestreamInfluxDBString(timestreamInfluxDBFieldValue(payload, "identifier")))
		if err != nil {
			respondTimestreamInfluxDBErrorForErr(w, err)
			return true
		}
		respondTimestreamInfluxDBJSON(w, http.StatusOK, timestreamInfluxDBInstanceToAPI(instance))
		return true

	case "DeleteDbInstance":
		instance, err := s.timestream.DeleteDbInstance(timestreamInfluxDBString(timestreamInfluxDBFieldValue(payload, "identifier")))
		if err != nil {
			respondTimestreamInfluxDBErrorForErr(w, err)
			return true
		}
		respondTimestreamInfluxDBJSON(w, http.StatusOK, timestreamInfluxDBInstanceToAPI(instance))
		return true

	// Stage 4: parameter group management.
	case "CreateDbParameterGroup":
		parameters, ok := timestreamInfluxDBMap(timestreamInfluxDBFieldValue(payload, "parameters"))
		if !ok {
			respondTimestreamInfluxDBError(w, http.StatusBadRequest, "ValidationException", "invalid parameters")
			return true
		}
		tags, ok := timestreamInfluxDBTags(timestreamInfluxDBFieldValue(payload, "tags"))
		if !ok {
			respondTimestreamInfluxDBError(w, http.StatusBadRequest, "ValidationException", "invalid tags")
			return true
		}
		parameterGroup, err := s.timestream.CreateDbParameterGroup(timestreamsvc.CreateDbParameterGroupInput{
			Name:        timestreamInfluxDBString(timestreamInfluxDBFieldValue(payload, "name")),
			Description: timestreamInfluxDBString(timestreamInfluxDBFieldValue(payload, "description")),
			Parameters:  parameters,
			Tags:        tags,
		})
		if err != nil {
			respondTimestreamInfluxDBErrorForErr(w, err)
			return true
		}
		respondTimestreamInfluxDBJSON(w, http.StatusOK, timestreamInfluxDBParameterGroupToAPI(parameterGroup))
		return true

	// Stage 1: read/list surface.
	case "GetDbCluster":
		cluster, err := s.timestream.GetDbCluster(timestreamInfluxDBString(timestreamInfluxDBFieldValue(payload, "dbClusterId")))
		if err != nil {
			respondTimestreamInfluxDBErrorForErr(w, err)
			return true
		}
		respondTimestreamInfluxDBJSON(w, http.StatusOK, timestreamInfluxDBClusterToAPI(cluster))
		return true

	case "ListDbClusters":
		nextToken, maxResults, err := timestreamInfluxDBListArgs(payload)
		if err != nil {
			respondTimestreamInfluxDBError(w, http.StatusBadRequest, "ValidationException", err.Error())
			return true
		}
		clusters, outToken, err := s.timestream.ListDbClusters(nextToken, maxResults)
		if err != nil {
			respondTimestreamInfluxDBErrorForErr(w, err)
			return true
		}
		items := make([]map[string]any, 0, len(clusters))
		for _, cluster := range clusters {
			items = append(items, timestreamInfluxDBClusterSummaryToAPI(cluster))
		}
		response := map[string]any{"items": items}
		if outToken != "" {
			response["nextToken"] = outToken
		}
		respondTimestreamInfluxDBJSON(w, http.StatusOK, response)
		return true

	case "GetDbInstance":
		instance, err := s.timestream.GetDbInstance(timestreamInfluxDBString(timestreamInfluxDBFieldValue(payload, "identifier")))
		if err != nil {
			respondTimestreamInfluxDBErrorForErr(w, err)
			return true
		}
		respondTimestreamInfluxDBJSON(w, http.StatusOK, timestreamInfluxDBInstanceToAPI(instance))
		return true

	case "ListDbInstances":
		nextToken, maxResults, err := timestreamInfluxDBListArgs(payload)
		if err != nil {
			respondTimestreamInfluxDBError(w, http.StatusBadRequest, "ValidationException", err.Error())
			return true
		}
		instances, outToken, err := s.timestream.ListDbInstances(nextToken, maxResults)
		if err != nil {
			respondTimestreamInfluxDBErrorForErr(w, err)
			return true
		}
		items := make([]map[string]any, 0, len(instances))
		for _, instance := range instances {
			items = append(items, timestreamInfluxDBInstanceSummaryToAPI(instance))
		}
		response := map[string]any{"items": items}
		if outToken != "" {
			response["nextToken"] = outToken
		}
		respondTimestreamInfluxDBJSON(w, http.StatusOK, response)
		return true

	case "ListDbInstancesForCluster":
		nextToken, maxResults, err := timestreamInfluxDBListArgs(payload)
		if err != nil {
			respondTimestreamInfluxDBError(w, http.StatusBadRequest, "ValidationException", err.Error())
			return true
		}
		instances, outToken, err := s.timestream.ListDbInstancesForCluster(
			timestreamInfluxDBString(timestreamInfluxDBFieldValue(payload, "dbClusterId")),
			nextToken,
			maxResults,
		)
		if err != nil {
			respondTimestreamInfluxDBErrorForErr(w, err)
			return true
		}
		items := make([]map[string]any, 0, len(instances))
		for _, instance := range instances {
			items = append(items, timestreamInfluxDBInstanceForClusterSummaryToAPI(instance))
		}
		response := map[string]any{"items": items}
		if outToken != "" {
			response["nextToken"] = outToken
		}
		respondTimestreamInfluxDBJSON(w, http.StatusOK, response)
		return true

	case "GetDbParameterGroup":
		parameterGroup, err := s.timestream.GetDbParameterGroup(timestreamInfluxDBString(timestreamInfluxDBFieldValue(payload, "identifier")))
		if err != nil {
			respondTimestreamInfluxDBErrorForErr(w, err)
			return true
		}
		respondTimestreamInfluxDBJSON(w, http.StatusOK, timestreamInfluxDBParameterGroupToAPI(parameterGroup))
		return true

	case "ListDbParameterGroups":
		nextToken, maxResults, err := timestreamInfluxDBListArgs(payload)
		if err != nil {
			respondTimestreamInfluxDBError(w, http.StatusBadRequest, "ValidationException", err.Error())
			return true
		}
		parameterGroups, outToken, err := s.timestream.ListDbParameterGroups(nextToken, maxResults)
		if err != nil {
			respondTimestreamInfluxDBErrorForErr(w, err)
			return true
		}
		items := make([]map[string]any, 0, len(parameterGroups))
		for _, parameterGroup := range parameterGroups {
			items = append(items, timestreamInfluxDBParameterGroupSummaryToAPI(parameterGroup))
		}
		response := map[string]any{"items": items}
		if outToken != "" {
			response["nextToken"] = outToken
		}
		respondTimestreamInfluxDBJSON(w, http.StatusOK, response)
		return true

	case "ListTagsForResource":
		tags, err := s.timestream.ListTagsForResource(timestreamInfluxDBString(timestreamInfluxDBFieldValue(payload, "resourceArn")))
		if err != nil {
			respondTimestreamInfluxDBErrorForErr(w, err)
			return true
		}
		respondTimestreamInfluxDBJSON(w, http.StatusOK, map[string]any{
			"tags": tags,
		})
		return true

	// Stage 5: tag mutation actions.
	case "TagResource":
		tags, ok := timestreamInfluxDBTags(timestreamInfluxDBFieldValue(payload, "tags"))
		if !ok {
			respondTimestreamInfluxDBError(w, http.StatusBadRequest, "ValidationException", "invalid tags")
			return true
		}
		if err := s.timestream.TagResource(
			timestreamInfluxDBString(timestreamInfluxDBFieldValue(payload, "resourceArn")),
			tags,
		); err != nil {
			respondTimestreamInfluxDBErrorForErr(w, err)
			return true
		}
		respondTimestreamInfluxDBJSON(w, http.StatusOK, map[string]any{})
		return true

	case "UntagResource":
		tagKeys, ok := timestreamInfluxDBStringSlice(timestreamInfluxDBFieldValue(payload, "tagKeys"))
		if !ok {
			respondTimestreamInfluxDBError(w, http.StatusBadRequest, "ValidationException", "invalid tagKeys")
			return true
		}
		if err := s.timestream.UntagResource(
			timestreamInfluxDBString(timestreamInfluxDBFieldValue(payload, "resourceArn")),
			tagKeys,
		); err != nil {
			respondTimestreamInfluxDBErrorForErr(w, err)
			return true
		}
		respondTimestreamInfluxDBJSON(w, http.StatusOK, map[string]any{})
		return true
	}

	respondTimestreamInfluxDBError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
	return true
}

func isTimestreamInfluxDBJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}

	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.HasPrefix(target, "AmazonTimestreamInfluxDB.") {
		return true
	}

	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.0") || strings.Contains(contentType, "application/x-amz-json-1.1") {
		return strings.HasPrefix(target, "AmazonTimestreamInfluxDB")
	}

	if strings.TrimSpace(sigV4ServiceHint(r)) == "timestream-influxdb" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	return strings.Contains(host, ".timestream-influxdb.") || strings.HasPrefix(host, "timestream-influxdb.")
}

func parseTimestreamInfluxDBTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "AmazonTimestreamInfluxDB.") {
		return strings.TrimPrefix(target, "AmazonTimestreamInfluxDB.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parseTimestreamInfluxDBPayload(r *http.Request) (map[string]any, error) {
	body, err := readBodyBytes(r)
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return map[string]any{}, nil
	}
	var payload map[string]any
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	if err := dec.Decode(&payload); err != nil {
		return nil, err
	}
	if payload == nil {
		payload = map[string]any{}
	}
	return payload, nil
}

func respondTimestreamInfluxDBJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.0")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondTimestreamInfluxDBError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondTimestreamInfluxDBJSON(w, status, timestreamInfluxDBError{Type: code, Message: msg})
}

func respondTimestreamInfluxDBErrorForErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, timestreamsvc.ErrInvalidParameter):
		respondTimestreamInfluxDBError(w, http.StatusBadRequest, "ValidationException", err.Error())
	case errors.Is(err, timestreamsvc.ErrNotFound):
		respondTimestreamInfluxDBError(w, http.StatusNotFound, "ResourceNotFoundException", err.Error())
	case errors.Is(err, timestreamsvc.ErrConflict):
		respondTimestreamInfluxDBError(w, http.StatusConflict, "ConflictException", err.Error())
	default:
		respondTimestreamInfluxDBError(w, http.StatusInternalServerError, "InternalServerException", err.Error())
	}
}

func timestreamInfluxDBField(payload map[string]any, key string) (any, bool) {
	for k, v := range payload {
		if strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			return v, true
		}
	}
	return nil, false
}

func timestreamInfluxDBFieldValue(payload map[string]any, key string) any {
	value, _ := timestreamInfluxDBField(payload, key)
	return value
}

func timestreamInfluxDBString(value any) string {
	if str, ok := value.(string); ok {
		return strings.TrimSpace(str)
	}
	return ""
}

func timestreamInfluxDBInt(value any) (int, bool) {
	switch typed := value.(type) {
	case nil:
		return 0, false
	case int:
		return typed, true
	case int32:
		return int(typed), true
	case int64:
		return int(typed), true
	case float64:
		return int(typed), float64(int(typed)) == typed
	case json.Number:
		i, err := typed.Int64()
		if err == nil {
			return int(i), true
		}
		f, err := typed.Float64()
		if err != nil {
			return 0, false
		}
		return int(f), float64(int(f)) == f
	case string:
		i, err := strconv.Atoi(strings.TrimSpace(typed))
		if err != nil {
			return 0, false
		}
		return i, true
	default:
		return 0, false
	}
}

func timestreamInfluxDBBool(value any) (bool, bool) {
	switch typed := value.(type) {
	case bool:
		return typed, true
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "true":
			return true, true
		case "false":
			return false, true
		default:
			return false, false
		}
	default:
		return false, false
	}
}

func timestreamInfluxDBStringSlice(value any) ([]string, bool) {
	if value == nil {
		return nil, true
	}
	rawItems, ok := value.([]any)
	if !ok {
		return nil, false
	}
	out := make([]string, 0, len(rawItems))
	for _, rawItem := range rawItems {
		item, ok := rawItem.(string)
		if !ok {
			return nil, false
		}
		out = append(out, strings.TrimSpace(item))
	}
	return out, true
}

func timestreamInfluxDBMap(value any) (map[string]any, bool) {
	if value == nil {
		return nil, true
	}
	typed, ok := value.(map[string]any)
	if !ok {
		return nil, false
	}
	out := make(map[string]any, len(typed))
	for k, v := range typed {
		out[k] = v
	}
	return out, true
}

func timestreamInfluxDBTags(value any) (map[string]string, bool) {
	if value == nil {
		return map[string]string{}, true
	}
	raw, ok := value.(map[string]any)
	if !ok {
		return nil, false
	}
	tags := make(map[string]string, len(raw))
	for key, rawValue := range raw {
		value, ok := rawValue.(string)
		if !ok {
			return nil, false
		}
		tags[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return tags, true
}

func timestreamInfluxDBListArgs(payload map[string]any) (string, int, error) {
	nextToken := timestreamInfluxDBString(timestreamInfluxDBFieldValue(payload, "nextToken"))
	maxResults := 0
	if field, ok := timestreamInfluxDBField(payload, "maxResults"); ok {
		value, ok := timestreamInfluxDBInt(field)
		if !ok {
			return "", 0, errors.New("invalid maxResults")
		}
		maxResults = value
	}
	return nextToken, maxResults, nil
}

func timestreamInfluxDBClusterSummaryToAPI(cluster timestreamsvc.DbCluster) map[string]any {
	return map[string]any{
		"id":               cluster.ID,
		"name":             cluster.Name,
		"arn":              cluster.Arn,
		"status":           cluster.Status,
		"endpoint":         cluster.Endpoint,
		"readerEndpoint":   cluster.ReaderEndpoint,
		"port":             cluster.Port,
		"deploymentType":   cluster.DeploymentType,
		"dbInstanceType":   cluster.DbInstanceType,
		"networkType":      cluster.NetworkType,
		"dbStorageType":    cluster.DbStorageType,
		"allocatedStorage": cluster.AllocatedStorage,
		"engineType":       cluster.EngineType,
	}
}

func timestreamInfluxDBClusterToAPI(cluster timestreamsvc.DbCluster) map[string]any {
	result := timestreamInfluxDBClusterSummaryToAPI(cluster)
	result["publiclyAccessible"] = cluster.PubliclyAccessible
	result["dbParameterGroupIdentifier"] = cluster.DbParameterGroupIdentifier
	result["vpcSubnetIds"] = cluster.VpcSubnetIds
	result["vpcSecurityGroupIds"] = cluster.VpcSecurityGroupIds
	result["influxAuthParametersSecretArn"] = cluster.InfluxAuthParametersSecretArn
	result["failoverMode"] = cluster.FailoverMode
	if cluster.LogDeliveryConfiguration != nil {
		result["logDeliveryConfiguration"] = cluster.LogDeliveryConfiguration
	}
	return result
}

func timestreamInfluxDBInstanceSummaryToAPI(instance timestreamsvc.DbInstance) map[string]any {
	return map[string]any{
		"id":               instance.ID,
		"name":             instance.Name,
		"arn":              instance.Arn,
		"status":           instance.Status,
		"endpoint":         instance.Endpoint,
		"port":             instance.Port,
		"networkType":      instance.NetworkType,
		"dbInstanceType":   instance.DbInstanceType,
		"dbStorageType":    instance.DbStorageType,
		"allocatedStorage": instance.AllocatedStorage,
		"deploymentType":   instance.DeploymentType,
	}
}

func timestreamInfluxDBInstanceForClusterSummaryToAPI(instance timestreamsvc.DbInstance) map[string]any {
	result := timestreamInfluxDBInstanceSummaryToAPI(instance)
	result["instanceMode"] = instance.InstanceMode
	if len(instance.InstanceModes) > 0 {
		result["instanceModes"] = instance.InstanceModes
	}
	return result
}

func timestreamInfluxDBInstanceToAPI(instance timestreamsvc.DbInstance) map[string]any {
	result := timestreamInfluxDBInstanceForClusterSummaryToAPI(instance)
	result["vpcSubnetIds"] = instance.VpcSubnetIds
	result["publiclyAccessible"] = instance.PubliclyAccessible
	result["vpcSecurityGroupIds"] = instance.VpcSecurityGroupIds
	result["dbParameterGroupIdentifier"] = instance.DbParameterGroupIdentifier
	result["availabilityZone"] = instance.AvailabilityZone
	result["secondaryAvailabilityZone"] = instance.SecondaryAvailabilityZone
	result["influxAuthParametersSecretArn"] = instance.InfluxAuthParametersSecretArn
	result["dbClusterId"] = instance.DbClusterID
	if instance.LogDeliveryConfiguration != nil {
		result["logDeliveryConfiguration"] = instance.LogDeliveryConfiguration
	}
	return result
}

func timestreamInfluxDBParameterGroupSummaryToAPI(parameterGroup timestreamsvc.DbParameterGroup) map[string]any {
	return map[string]any{
		"id":          parameterGroup.ID,
		"name":        parameterGroup.Name,
		"arn":         parameterGroup.Arn,
		"description": parameterGroup.Description,
	}
}

func timestreamInfluxDBParameterGroupToAPI(parameterGroup timestreamsvc.DbParameterGroup) map[string]any {
	result := timestreamInfluxDBParameterGroupSummaryToAPI(parameterGroup)
	result["parameters"] = parameterGroup.Parameters
	return result
}
