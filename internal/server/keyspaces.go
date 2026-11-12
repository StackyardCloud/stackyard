package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strings"
	"time"

	keyspacessvc "github.com/stackyard/stackyard/internal/services/keyspaces"
)

type keyspacesError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleKeyspacesJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isKeyspacesJSONCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "cassandra")
	if !ok {
		respondKeyspacesError(w, status, code, msg)
		return true
	}

	action := parseKeyspacesTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondKeyspacesError(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, known := keyspacesOperationByName[action]; !known {
		respondKeyspacesError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseKeyspacesPayload(r)
	if err != nil {
		respondKeyspacesError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	switch action {
	case "CreateKeyspace":
		arn, err := s.keyspaces.CreateKeyspace(
			keyspacesString(payload["keyspaceName"]),
			keyspacesReplicationSpecification(payload["replicationSpecification"]),
			keyspacesTagsToMap(payload["tags"]),
		)
		if err != nil {
			respondKeyspacesErrorForErr(w, err)
			return true
		}
		respondKeyspacesJSON(w, http.StatusOK, map[string]any{"resourceArn": arn})
		return true

	case "GetKeyspace":
		keyspace, err := s.keyspaces.GetKeyspace(keyspacesString(payload["keyspaceName"]))
		if err != nil {
			respondKeyspacesErrorForErr(w, err)
			return true
		}
		response := map[string]any{
			"keyspaceName":        keyspace.KeyspaceName,
			"resourceArn":         keyspace.ResourceARN,
			"replicationStrategy": keyspace.ReplicationStrategy,
			"replicationRegions":  keyspace.ReplicationRegions,
		}
		respondKeyspacesJSON(w, http.StatusOK, response)
		return true

	case "ListKeyspaces":
		keyspaces, nextToken, err := s.keyspaces.ListKeyspaces(
			keyspacesString(payload["nextToken"]),
			keyspacesInt(payload["maxResults"]),
		)
		if err != nil {
			respondKeyspacesErrorForErr(w, err)
			return true
		}
		summaries := make([]map[string]any, 0, len(keyspaces))
		for _, ks := range keyspaces {
			summaries = append(summaries, map[string]any{
				"keyspaceName":        ks.KeyspaceName,
				"resourceArn":         ks.ResourceARN,
				"replicationStrategy": ks.ReplicationStrategy,
				"replicationRegions":  ks.ReplicationRegions,
			})
		}
		response := map[string]any{"keyspaces": summaries}
		if nextToken != "" {
			response["nextToken"] = nextToken
		}
		respondKeyspacesJSON(w, http.StatusOK, response)
		return true

	case "UpdateKeyspace":
		arn, err := s.keyspaces.UpdateKeyspace(
			keyspacesString(payload["keyspaceName"]),
			keyspacesReplicationSpecification(payload["replicationSpecification"]),
		)
		if err != nil {
			respondKeyspacesErrorForErr(w, err)
			return true
		}
		respondKeyspacesJSON(w, http.StatusOK, map[string]any{"resourceArn": arn})
		return true

	case "DeleteKeyspace":
		if err := s.keyspaces.DeleteKeyspace(keyspacesString(payload["keyspaceName"])); err != nil {
			respondKeyspacesErrorForErr(w, err)
			return true
		}
		respondKeyspacesJSON(w, http.StatusOK, map[string]any{})
		return true

	case "CreateTable":
		arn, err := s.keyspaces.CreateTable(keyspacessvc.CreateTableInput{
			KeyspaceName:             keyspacesString(payload["keyspaceName"]),
			TableName:                keyspacesString(payload["tableName"]),
			SchemaDefinition:         keyspacesMap(payload["schemaDefinition"]),
			Comment:                  keyspacesMap(payload["comment"]),
			CapacitySpecification:    keyspacesMap(payload["capacitySpecification"]),
			EncryptionSpecification:  keyspacesMap(payload["encryptionSpecification"]),
			PointInTimeRecovery:      keyspacesMap(payload["pointInTimeRecovery"]),
			TTL:                      keyspacesMap(payload["ttl"]),
			DefaultTimeToLive:        keyspacesOptionalInt(payload["defaultTimeToLive"]),
			Tags:                     keyspacesTagsToMap(payload["tags"]),
			ClientSideTimestamps:     keyspacesMap(payload["clientSideTimestamps"]),
			AutoScalingSpecification: keyspacesMap(payload["autoScalingSpecification"]),
			ReplicaSpecifications:    keyspacesSliceOfMaps(payload["replicaSpecifications"]),
		})
		if err != nil {
			respondKeyspacesErrorForErr(w, err)
			return true
		}
		respondKeyspacesJSON(w, http.StatusOK, map[string]any{"resourceArn": arn})
		return true

	case "GetTable":
		table, err := s.keyspaces.GetTable(
			keyspacesString(payload["keyspaceName"]),
			keyspacesString(payload["tableName"]),
		)
		if err != nil {
			respondKeyspacesErrorForErr(w, err)
			return true
		}
		response := map[string]any{
			"keyspaceName":      table.KeyspaceName,
			"tableName":         table.TableName,
			"resourceArn":       table.ResourceARN,
			"creationTimestamp": keyspacesTimestamp(table.CreationTimestamp),
			"status":            table.Status,
		}
		if table.SchemaDefinition != nil {
			response["schemaDefinition"] = table.SchemaDefinition
		}
		if table.CapacitySpecification != nil {
			response["capacitySpecification"] = table.CapacitySpecification
		}
		if table.EncryptionSpecification != nil {
			response["encryptionSpecification"] = table.EncryptionSpecification
		}
		if table.PointInTimeRecovery != nil {
			response["pointInTimeRecovery"] = table.PointInTimeRecovery
		}
		if table.TTL != nil {
			response["ttl"] = table.TTL
		}
		if table.DefaultTimeToLive != nil {
			response["defaultTimeToLive"] = *table.DefaultTimeToLive
		}
		if table.Comment != nil {
			response["comment"] = table.Comment
		}
		if table.ClientSideTimestamps != nil {
			response["clientSideTimestamps"] = table.ClientSideTimestamps
		}
		if table.AutoScalingSpecification != nil {
			response["autoScalingSpecification"] = table.AutoScalingSpecification
		}
		if len(table.ReplicaSpecifications) > 0 {
			response["replicaSpecifications"] = table.ReplicaSpecifications
		}
		respondKeyspacesJSON(w, http.StatusOK, response)
		return true

	case "ListTables":
		tables, nextToken, err := s.keyspaces.ListTables(
			keyspacesString(payload["keyspaceName"]),
			keyspacesString(payload["nextToken"]),
			keyspacesInt(payload["maxResults"]),
		)
		if err != nil {
			respondKeyspacesErrorForErr(w, err)
			return true
		}
		summaries := make([]map[string]any, 0, len(tables))
		for _, tbl := range tables {
			summaries = append(summaries, map[string]any{
				"keyspaceName": tbl.KeyspaceName,
				"tableName":    tbl.TableName,
				"resourceArn":  tbl.ResourceARN,
			})
		}
		response := map[string]any{"tables": summaries}
		if nextToken != "" {
			response["nextToken"] = nextToken
		}
		respondKeyspacesJSON(w, http.StatusOK, response)
		return true

	case "UpdateTable":
		arn, err := s.keyspaces.UpdateTable(keyspacessvc.UpdateTableInput{
			KeyspaceName:             keyspacesString(payload["keyspaceName"]),
			TableName:                keyspacesString(payload["tableName"]),
			AddColumns:               keyspacesSliceOfMaps(payload["addColumns"]),
			CapacitySpecification:    keyspacesMap(payload["capacitySpecification"]),
			EncryptionSpecification:  keyspacesMap(payload["encryptionSpecification"]),
			PointInTimeRecovery:      keyspacesMap(payload["pointInTimeRecovery"]),
			TTL:                      keyspacesMap(payload["ttl"]),
			DefaultTimeToLive:        keyspacesOptionalInt(payload["defaultTimeToLive"]),
			ClientSideTimestamps:     keyspacesMap(payload["clientSideTimestamps"]),
			AutoScalingSpecification: keyspacesMap(payload["autoScalingSpecification"]),
			ReplicaSpecifications:    keyspacesSliceOfMaps(payload["replicaSpecifications"]),
		})
		if err != nil {
			respondKeyspacesErrorForErr(w, err)
			return true
		}
		respondKeyspacesJSON(w, http.StatusOK, map[string]any{"resourceArn": arn})
		return true

	case "DeleteTable":
		if err := s.keyspaces.DeleteTable(
			keyspacesString(payload["keyspaceName"]),
			keyspacesString(payload["tableName"]),
		); err != nil {
			respondKeyspacesErrorForErr(w, err)
			return true
		}
		respondKeyspacesJSON(w, http.StatusOK, map[string]any{})
		return true

	case "RestoreTable":
		arn, err := s.keyspaces.RestoreTable(keyspacessvc.RestoreTableInput{
			SourceKeyspaceName:              keyspacesString(payload["sourceKeyspaceName"]),
			SourceTableName:                 keyspacesString(payload["sourceTableName"]),
			TargetKeyspaceName:              keyspacesString(payload["targetKeyspaceName"]),
			TargetTableName:                 keyspacesString(payload["targetTableName"]),
			RestoreTimestamp:                keyspacesOptionalTimestamp(payload["restoreTimestamp"]),
			CapacitySpecificationOverride:   keyspacesMap(payload["capacitySpecificationOverride"]),
			EncryptionSpecificationOverride: keyspacesMap(payload["encryptionSpecificationOverride"]),
			PointInTimeRecoveryOverride:     keyspacesMap(payload["pointInTimeRecoveryOverride"]),
			TagsOverride:                    keyspacesTagsToMap(payload["tagsOverride"]),
			AutoScalingSpecification:        keyspacesMap(payload["autoScalingSpecification"]),
			ReplicaSpecifications:           keyspacesSliceOfMaps(payload["replicaSpecifications"]),
		})
		if err != nil {
			respondKeyspacesErrorForErr(w, err)
			return true
		}
		respondKeyspacesJSON(w, http.StatusOK, map[string]any{"restoredTableARN": arn})
		return true

	case "GetTableAutoScalingSettings":
		settings, err := s.keyspaces.GetTableAutoScalingSettings(
			keyspacesString(payload["keyspaceName"]),
			keyspacesString(payload["tableName"]),
		)
		if err != nil {
			respondKeyspacesErrorForErr(w, err)
			return true
		}
		response := map[string]any{
			"keyspaceName": settings.KeyspaceName,
			"tableName":    settings.TableName,
			"resourceArn":  settings.ResourceARN,
		}
		if settings.AutoScalingSpecification != nil {
			response["autoScalingSpecification"] = settings.AutoScalingSpecification
		}
		if len(settings.ReplicaSpecifications) > 0 {
			response["replicaSpecifications"] = settings.ReplicaSpecifications
		}
		respondKeyspacesJSON(w, http.StatusOK, response)
		return true

	case "CreateType":
		keyspaceARN, typeName, err := s.keyspaces.CreateType(
			keyspacesString(payload["keyspaceName"]),
			keyspacesString(payload["typeName"]),
			keyspacesSliceOfMaps(payload["fieldDefinitions"]),
		)
		if err != nil {
			respondKeyspacesErrorForErr(w, err)
			return true
		}
		respondKeyspacesJSON(w, http.StatusOK, map[string]any{
			"keyspaceArn": keyspaceARN,
			"typeName":    typeName,
		})
		return true

	case "GetType":
		entry, err := s.keyspaces.GetType(
			keyspacesString(payload["keyspaceName"]),
			keyspacesString(payload["typeName"]),
		)
		if err != nil {
			respondKeyspacesErrorForErr(w, err)
			return true
		}
		response := map[string]any{
			"keyspaceName": entry.KeyspaceName,
			"typeName":     entry.TypeName,
			"keyspaceArn":  entry.KeyspaceARN,
		}
		if len(entry.FieldDefinitions) > 0 {
			response["fieldDefinitions"] = entry.FieldDefinitions
		}
		if !entry.LastModifiedTimestamp.IsZero() {
			response["lastModifiedTimestamp"] = keyspacesTimestamp(entry.LastModifiedTimestamp)
		}
		if entry.Status != "" {
			response["status"] = entry.Status
		}
		if len(entry.DirectReferringTables) > 0 {
			response["directReferringTables"] = entry.DirectReferringTables
		}
		if len(entry.DirectParentTypes) > 0 {
			response["directParentTypes"] = entry.DirectParentTypes
		}
		if entry.MaxNestingDepth > 0 {
			response["maxNestingDepth"] = entry.MaxNestingDepth
		}
		respondKeyspacesJSON(w, http.StatusOK, response)
		return true

	case "ListTypes":
		types, nextToken, err := s.keyspaces.ListTypes(
			keyspacesString(payload["keyspaceName"]),
			keyspacesString(payload["nextToken"]),
			keyspacesInt(payload["maxResults"]),
		)
		if err != nil {
			respondKeyspacesErrorForErr(w, err)
			return true
		}
		response := map[string]any{"types": types}
		if nextToken != "" {
			response["nextToken"] = nextToken
		}
		respondKeyspacesJSON(w, http.StatusOK, response)
		return true

	case "DeleteType":
		keyspaceARN, typeName, err := s.keyspaces.DeleteType(
			keyspacesString(payload["keyspaceName"]),
			keyspacesString(payload["typeName"]),
		)
		if err != nil {
			respondKeyspacesErrorForErr(w, err)
			return true
		}
		respondKeyspacesJSON(w, http.StatusOK, map[string]any{
			"keyspaceArn": keyspaceARN,
			"typeName":    typeName,
		})
		return true

	case "TagResource":
		if err := s.keyspaces.TagResource(
			keyspacesString(payload["resourceArn"]),
			keyspacesTagsToMap(payload["tags"]),
		); err != nil {
			respondKeyspacesErrorForErr(w, err)
			return true
		}
		respondKeyspacesJSON(w, http.StatusOK, map[string]any{})
		return true

	case "UntagResource":
		if err := s.keyspaces.UntagResource(
			keyspacesString(payload["resourceArn"]),
			keyspacesTagKeys(payload["tags"]),
		); err != nil {
			respondKeyspacesErrorForErr(w, err)
			return true
		}
		respondKeyspacesJSON(w, http.StatusOK, map[string]any{})
		return true

	case "ListTagsForResource":
		tags, nextToken, err := s.keyspaces.ListTagsForResource(
			keyspacesString(payload["resourceArn"]),
			keyspacesString(payload["nextToken"]),
			keyspacesInt(payload["maxResults"]),
		)
		if err != nil {
			respondKeyspacesErrorForErr(w, err)
			return true
		}
		response := map[string]any{
			"tags": keyspacesTagListFromMap(tags),
		}
		if nextToken != "" {
			response["nextToken"] = nextToken
		}
		respondKeyspacesJSON(w, http.StatusOK, response)
		return true
	}

	respondKeyspacesError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
	return true
}

func isKeyspacesJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}

	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.HasPrefix(target, "KeyspacesService.") {
		return true
	}

	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.0") || strings.Contains(contentType, "application/x-amz-json-1.1") {
		return strings.HasPrefix(target, "KeyspacesService")
	}

	if strings.TrimSpace(sigV4ServiceHint(r)) == "cassandra" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	return strings.Contains(host, ".keyspaces.") || strings.HasPrefix(host, "keyspaces.") || strings.Contains(host, ".cassandra.")
}

func parseKeyspacesTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "KeyspacesService.") {
		return strings.TrimPrefix(target, "KeyspacesService.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func respondKeyspacesJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.0")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondKeyspacesError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondKeyspacesJSON(w, status, keyspacesError{Type: code, Message: msg})
}

func respondKeyspacesErrorForErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, keyspacessvc.ErrInvalidParameter):
		respondKeyspacesError(w, http.StatusBadRequest, "ValidationException", err.Error())
	case errors.Is(err, keyspacessvc.ErrAlreadyExists):
		respondKeyspacesError(w, http.StatusBadRequest, "ConflictException", err.Error())
	case errors.Is(err, keyspacessvc.ErrNotFound):
		respondKeyspacesError(w, http.StatusBadRequest, "ResourceNotFoundException", err.Error())
	case errors.Is(err, keyspacessvc.ErrConflict):
		respondKeyspacesError(w, http.StatusBadRequest, "ConflictException", err.Error())
	default:
		respondKeyspacesError(w, http.StatusBadRequest, "ValidationException", err.Error())
	}
}

func parseKeyspacesPayload(r *http.Request) (map[string]any, error) {
	body, err := readBodyBytes(r)
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return map[string]any{}, nil
	}

	var value any
	if err := json.Unmarshal(body, &value); err != nil {
		return nil, err
	}
	obj, ok := value.(map[string]any)
	if !ok {
		return map[string]any{}, nil
	}
	return obj, nil
}

func keyspacesString(value any) string {
	if raw, ok := value.(string); ok {
		return strings.TrimSpace(raw)
	}
	return ""
}

func keyspacesInt(value any) int {
	switch raw := value.(type) {
	case float64:
		return int(raw)
	case float32:
		return int(raw)
	case int:
		return raw
	case int32:
		return int(raw)
	case int64:
		return int(raw)
	default:
		return 0
	}
}

func keyspacesOptionalInt(value any) *int {
	switch raw := value.(type) {
	case float64:
		out := int(raw)
		return &out
	case float32:
		out := int(raw)
		return &out
	case int:
		out := raw
		return &out
	case int32:
		out := int(raw)
		return &out
	case int64:
		out := int(raw)
		return &out
	default:
		return nil
	}
}

func keyspacesMap(value any) map[string]any {
	raw, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	out := make(map[string]any, len(raw))
	for k, v := range raw {
		out[k] = v
	}
	return out
}

func keyspacesSliceOfMaps(value any) []map[string]any {
	raw, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]map[string]any, 0, len(raw))
	for _, entry := range raw {
		entryMap, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		copied := make(map[string]any, len(entryMap))
		for k, v := range entryMap {
			copied[k] = v
		}
		out = append(out, copied)
	}
	return out
}

func keyspacesTagListFromMap(tags map[string]string) []map[string]any {
	if len(tags) == 0 {
		return []map[string]any{}
	}
	keys := make([]string, 0, len(tags))
	for k := range tags {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	out := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, map[string]any{
			"key":   key,
			"value": tags[key],
		})
	}
	return out
}

func keyspacesTagKeys(value any) []string {
	switch raw := value.(type) {
	case []any:
		out := make([]string, 0, len(raw))
		for _, entry := range raw {
			if key := keyspacesTagKey(entry); key != "" {
				out = append(out, key)
			}
		}
		return out
	case []string:
		out := make([]string, 0, len(raw))
		for _, key := range raw {
			key = strings.TrimSpace(key)
			if key != "" {
				out = append(out, key)
			}
		}
		return out
	default:
		return nil
	}
}

func keyspacesTagsToMap(value any) map[string]string {
	raw, ok := value.([]any)
	if !ok {
		return nil
	}
	out := map[string]string{}
	for _, entry := range raw {
		key := keyspacesTagKey(entry)
		if key == "" {
			continue
		}
		value := keyspacesTagValue(entry)
		out[key] = value
	}
	return out
}

func keyspacesTagKey(entry any) string {
	switch raw := entry.(type) {
	case string:
		return strings.TrimSpace(raw)
	case map[string]any:
		if key := keyspacesString(raw["key"]); key != "" {
			return key
		}
		if key := keyspacesString(raw["Key"]); key != "" {
			return key
		}
	}
	return ""
}

func keyspacesTagValue(entry any) string {
	raw, ok := entry.(map[string]any)
	if !ok {
		return ""
	}
	if value := keyspacesString(raw["value"]); value != "" {
		return value
	}
	if value := keyspacesString(raw["Value"]); value != "" {
		return value
	}
	return ""
}

func keyspacesReplicationSpecification(value any) keyspacessvc.ReplicationSpecification {
	specMap, ok := value.(map[string]any)
	if !ok {
		return keyspacessvc.ReplicationSpecification{}
	}
	spec := keyspacessvc.ReplicationSpecification{
		ReplicationStrategy: keyspacesString(specMap["replicationStrategy"]),
	}
	if rawRegions, ok := specMap["regionList"].([]any); ok {
		regions := make([]string, 0, len(rawRegions))
		for _, rawRegion := range rawRegions {
			region := keyspacesString(rawRegion)
			if region != "" {
				regions = append(regions, region)
			}
		}
		spec.RegionList = regions
	}
	return spec
}

func keyspacesOptionalTimestamp(value any) *time.Time {
	switch raw := value.(type) {
	case string:
		if raw == "" {
			return nil
		}
		if parsed, err := time.Parse(time.RFC3339Nano, raw); err == nil {
			t := parsed.UTC()
			return &t
		}
		if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
			t := parsed.UTC()
			return &t
		}
	case float64:
		secs := int64(raw)
		t := time.Unix(secs, 0).UTC()
		return &t
	case int64:
		t := time.Unix(raw, 0).UTC()
		return &t
	case int:
		t := time.Unix(int64(raw), 0).UTC()
		return &t
	}
	return nil
}

func keyspacesTimestamp(t time.Time) float64 {
	return float64(t.UnixNano()) / float64(time.Second)
}
