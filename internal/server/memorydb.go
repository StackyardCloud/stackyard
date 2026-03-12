package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"

	memorydbsvc "github.com/stackyard/stackyard/internal/services/memorydb"
)

type memorydbError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleMemoryDBJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isMemoryDBJSONCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "memorydb")
	if !ok {
		respondMemoryDBError(w, status, code, msg)
		return true
	}

	action := parseMemoryDBTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondMemoryDBError(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, known := memorydbOperationByName[action]; !known {
		respondMemoryDBError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseMemoryDBPayload(r)
	if err != nil {
		respondMemoryDBError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	switch action {
	case "CreateACL":
		acl, err := s.memorydb.CreateACL(
			memorydbString(memorydbField(payload, "ACLName")),
			memorydbStringList(memorydbField(payload, "UserNames")),
			memorydbTagsToMap(memorydbField(payload, "Tags")),
		)
		if err != nil {
			respondMemoryDBErrorForErr(w, err)
			return true
		}
		respondMemoryDBJSON(w, http.StatusOK, map[string]any{"ACL": memorydbACLToAPI(acl)})
		return true

	case "DescribeACLs":
		acls, nextToken, err := s.memorydb.DescribeACLs(
			memorydbString(memorydbField(payload, "ACLName")),
			memorydbString(memorydbField(payload, "NextToken")),
			memorydbInt(memorydbField(payload, "MaxResults")),
		)
		if err != nil {
			respondMemoryDBErrorForErr(w, err)
			return true
		}
		out := make([]map[string]any, 0, len(acls))
		for _, acl := range acls {
			out = append(out, memorydbACLToAPI(acl))
		}
		response := map[string]any{"ACLs": out}
		if nextToken != "" {
			response["NextToken"] = nextToken
		}
		respondMemoryDBJSON(w, http.StatusOK, response)
		return true

	case "UpdateACL":
		acl, err := s.memorydb.UpdateACL(
			memorydbString(memorydbField(payload, "ACLName")),
			memorydbStringList(memorydbField(payload, "UserNamesToAdd")),
			memorydbStringList(memorydbField(payload, "UserNamesToRemove")),
		)
		if err != nil {
			respondMemoryDBErrorForErr(w, err)
			return true
		}
		respondMemoryDBJSON(w, http.StatusOK, map[string]any{"ACL": memorydbACLToAPI(acl)})
		return true

	case "DeleteACL":
		acl, err := s.memorydb.DeleteACL(memorydbString(memorydbField(payload, "ACLName")))
		if err != nil {
			respondMemoryDBErrorForErr(w, err)
			return true
		}
		respondMemoryDBJSON(w, http.StatusOK, map[string]any{"ACL": memorydbACLToAPI(acl)})
		return true

	case "CreateUser":
		authType, passwordCount := memorydbAuthenticationInput(memorydbField(payload, "AuthenticationMode"))
		user, err := s.memorydb.CreateUser(memorydbsvc.CreateUserInput{
			UserName:           memorydbString(memorydbField(payload, "UserName")),
			AccessString:       memorydbString(memorydbField(payload, "AccessString")),
			AuthenticationType: authType,
			PasswordCount:      passwordCount,
			Tags:               memorydbTagsToMap(memorydbField(payload, "Tags")),
		})
		if err != nil {
			respondMemoryDBErrorForErr(w, err)
			return true
		}
		respondMemoryDBJSON(w, http.StatusOK, map[string]any{"User": memorydbUserToAPI(user)})
		return true

	case "DescribeUsers":
		users, nextToken, err := s.memorydb.DescribeUsers(
			memorydbString(memorydbField(payload, "UserName")),
			memorydbString(memorydbField(payload, "NextToken")),
			memorydbInt(memorydbField(payload, "MaxResults")),
		)
		if err != nil {
			respondMemoryDBErrorForErr(w, err)
			return true
		}
		out := make([]map[string]any, 0, len(users))
		for _, user := range users {
			out = append(out, memorydbUserToAPI(user))
		}
		response := map[string]any{"Users": out}
		if nextToken != "" {
			response["NextToken"] = nextToken
		}
		respondMemoryDBJSON(w, http.StatusOK, response)
		return true

	case "UpdateUser":
		var accessPtr *string
		if value, ok := memorydbOptionalStringField(payload, "AccessString"); ok {
			accessPtr = &value
		}
		var authTypePtr *string
		var pwCountPtr *int
		if raw, ok := memorydbOptionalField(payload, "AuthenticationMode"); ok {
			authType, passwordCount := memorydbAuthenticationInput(raw)
			authTypePtr = &authType
			pwCountPtr = &passwordCount
		}
		user, err := s.memorydb.UpdateUser(memorydbsvc.UpdateUserInput{
			UserName:           memorydbString(memorydbField(payload, "UserName")),
			AccessString:       accessPtr,
			AuthenticationType: authTypePtr,
			PasswordCount:      pwCountPtr,
		})
		if err != nil {
			respondMemoryDBErrorForErr(w, err)
			return true
		}
		respondMemoryDBJSON(w, http.StatusOK, map[string]any{"User": memorydbUserToAPI(user)})
		return true

	case "DeleteUser":
		user, err := s.memorydb.DeleteUser(memorydbString(memorydbField(payload, "UserName")))
		if err != nil {
			respondMemoryDBErrorForErr(w, err)
			return true
		}
		respondMemoryDBJSON(w, http.StatusOK, map[string]any{"User": memorydbUserToAPI(user)})
		return true

	case "CreateSubnetGroup":
		group, err := s.memorydb.CreateSubnetGroup(
			memorydbString(memorydbField(payload, "SubnetGroupName")),
			memorydbString(memorydbField(payload, "Description")),
			memorydbStringList(memorydbField(payload, "SubnetIds")),
			memorydbTagsToMap(memorydbField(payload, "Tags")),
		)
		if err != nil {
			respondMemoryDBErrorForErr(w, err)
			return true
		}
		respondMemoryDBJSON(w, http.StatusOK, map[string]any{"SubnetGroup": memorydbSubnetGroupToAPI(group)})
		return true

	case "DescribeSubnetGroups":
		groups, nextToken, err := s.memorydb.DescribeSubnetGroups(
			memorydbString(memorydbField(payload, "SubnetGroupName")),
			memorydbString(memorydbField(payload, "NextToken")),
			memorydbInt(memorydbField(payload, "MaxResults")),
		)
		if err != nil {
			respondMemoryDBErrorForErr(w, err)
			return true
		}
		out := make([]map[string]any, 0, len(groups))
		for _, group := range groups {
			out = append(out, memorydbSubnetGroupToAPI(group))
		}
		response := map[string]any{"SubnetGroups": out}
		if nextToken != "" {
			response["NextToken"] = nextToken
		}
		respondMemoryDBJSON(w, http.StatusOK, response)
		return true

	case "UpdateSubnetGroup":
		group, err := s.memorydb.UpdateSubnetGroup(
			memorydbString(memorydbField(payload, "SubnetGroupName")),
			memorydbString(memorydbField(payload, "Description")),
			memorydbStringList(memorydbField(payload, "SubnetIds")),
		)
		if err != nil {
			respondMemoryDBErrorForErr(w, err)
			return true
		}
		respondMemoryDBJSON(w, http.StatusOK, map[string]any{"SubnetGroup": memorydbSubnetGroupToAPI(group)})
		return true

	case "DeleteSubnetGroup":
		group, err := s.memorydb.DeleteSubnetGroup(memorydbString(memorydbField(payload, "SubnetGroupName")))
		if err != nil {
			respondMemoryDBErrorForErr(w, err)
			return true
		}
		respondMemoryDBJSON(w, http.StatusOK, map[string]any{"SubnetGroup": memorydbSubnetGroupToAPI(group)})
		return true

	case "CreateCluster":
		cluster, err := s.memorydb.CreateCluster(memorydbsvc.CreateClusterInput{
			ClusterName:             memorydbString(memorydbField(payload, "ClusterName")),
			NodeType:                memorydbString(memorydbField(payload, "NodeType")),
			MultiRegionClusterName:  memorydbString(memorydbField(payload, "MultiRegionClusterName")),
			ParameterGroupName:      memorydbString(memorydbField(payload, "ParameterGroupName")),
			Description:             memorydbString(memorydbField(payload, "Description")),
			NumShards:               memorydbInt(memorydbField(payload, "NumShards")),
			SubnetGroupName:         memorydbString(memorydbField(payload, "SubnetGroupName")),
			SecurityGroupIDs:        memorydbStringList(memorydbField(payload, "SecurityGroupIds")),
			MaintenanceWindow:       memorydbString(memorydbField(payload, "MaintenanceWindow")),
			Port:                    memorydbInt(memorydbField(payload, "Port")),
			SnsTopicArn:             memorydbString(memorydbField(payload, "SnsTopicArn")),
			TLSEnabled:              memorydbOptionalBoolFieldValue(payload, "TLSEnabled"),
			KmsKeyID:                memorydbString(memorydbField(payload, "KmsKeyId")),
			SnapshotRetentionLimit:  memorydbInt(memorydbField(payload, "SnapshotRetentionLimit")),
			SnapshotWindow:          memorydbString(memorydbField(payload, "SnapshotWindow")),
			ACLName:                 memorydbString(memorydbField(payload, "ACLName")),
			Engine:                  memorydbString(memorydbField(payload, "Engine")),
			EngineVersion:           memorydbString(memorydbField(payload, "EngineVersion")),
			AutoMinorVersionUpgrade: memorydbOptionalBoolFieldValue(payload, "AutoMinorVersionUpgrade"),
			DataTiering:             memorydbOptionalBoolFieldValue(payload, "DataTiering"),
		})
		if err != nil {
			respondMemoryDBErrorForErr(w, err)
			return true
		}
		respondMemoryDBJSON(w, http.StatusOK, map[string]any{"Cluster": memorydbClusterToAPI(cluster)})
		return true

	case "DescribeClusters":
		clusters, nextToken, err := s.memorydb.DescribeClusters(
			memorydbString(memorydbField(payload, "ClusterName")),
			memorydbString(memorydbField(payload, "NextToken")),
			memorydbInt(memorydbField(payload, "MaxResults")),
		)
		if err != nil {
			respondMemoryDBErrorForErr(w, err)
			return true
		}
		out := make([]map[string]any, 0, len(clusters))
		for _, cluster := range clusters {
			out = append(out, memorydbClusterToAPI(cluster))
		}
		response := map[string]any{"Clusters": out}
		if nextToken != "" {
			response["NextToken"] = nextToken
		}
		respondMemoryDBJSON(w, http.StatusOK, response)
		return true

	case "UpdateCluster":
		update := memorydbsvc.UpdateClusterInput{
			ClusterName:         memorydbString(memorydbField(payload, "ClusterName")),
			SecurityGroupIDs:    memorydbStringList(memorydbField(payload, "SecurityGroupIds")),
			SecurityGroupIDsSet: memorydbFieldExists(payload, "SecurityGroupIds"),
		}
		if value, ok := memorydbOptionalStringField(payload, "Description"); ok {
			update.Description = &value
		}
		if value, ok := memorydbOptionalStringField(payload, "MaintenanceWindow"); ok {
			update.MaintenanceWindow = &value
		}
		if value, ok := memorydbOptionalStringField(payload, "SnsTopicArn"); ok {
			update.SnsTopicArn = &value
		}
		if value, ok := memorydbOptionalStringField(payload, "SnsTopicStatus"); ok {
			update.SnsTopicStatus = &value
		}
		if value, ok := memorydbOptionalStringField(payload, "ParameterGroupName"); ok {
			update.ParameterGroupName = &value
		}
		if value, ok := memorydbOptionalStringField(payload, "SnapshotWindow"); ok {
			update.SnapshotWindow = &value
		}
		if value, ok := memorydbOptionalIntField(payload, "SnapshotRetentionLimit"); ok {
			update.SnapshotRetentionLimit = &value
		}
		if value, ok := memorydbOptionalStringField(payload, "NodeType"); ok {
			update.NodeType = &value
		}
		if value, ok := memorydbOptionalStringField(payload, "Engine"); ok {
			update.Engine = &value
		}
		if value, ok := memorydbOptionalStringField(payload, "EngineVersion"); ok {
			update.EngineVersion = &value
		}
		if shardCfg := memorydbMap(memorydbField(payload, "ShardConfiguration")); shardCfg != nil {
			if value, ok := memorydbOptionalIntField(shardCfg, "ShardCount"); ok {
				update.ShardCount = &value
			}
		}
		if value, ok := memorydbOptionalStringField(payload, "ACLName"); ok {
			update.ACLName = &value
		}

		cluster, err := s.memorydb.UpdateCluster(update)
		if err != nil {
			respondMemoryDBErrorForErr(w, err)
			return true
		}
		respondMemoryDBJSON(w, http.StatusOK, map[string]any{"Cluster": memorydbClusterToAPI(cluster)})
		return true

	case "DeleteCluster":
		cluster, err := s.memorydb.DeleteCluster(memorydbString(memorydbField(payload, "ClusterName")))
		if err != nil {
			respondMemoryDBErrorForErr(w, err)
			return true
		}
		respondMemoryDBJSON(w, http.StatusOK, map[string]any{"Cluster": memorydbClusterToAPI(cluster)})
		return true

	case "CreateParameterGroup":
		pg, err := s.memorydb.CreateParameterGroup(memorydbsvc.CreateParameterGroupInput{
			Name:        memorydbString(memorydbField(payload, "ParameterGroupName")),
			Family:      memorydbString(memorydbField(payload, "Family")),
			Description: memorydbString(memorydbField(payload, "Description")),
			Tags:        memorydbTagsToMap(memorydbField(payload, "Tags")),
		})
		if err != nil {
			respondMemoryDBErrorForErr(w, err)
			return true
		}
		respondMemoryDBJSON(w, http.StatusOK, map[string]any{"ParameterGroup": memorydbParameterGroupToAPI(pg)})
		return true

	case "DescribeParameterGroups":
		groups, nextToken, err := s.memorydb.DescribeParameterGroups(
			memorydbString(memorydbField(payload, "ParameterGroupName")),
			memorydbString(memorydbField(payload, "NextToken")),
			memorydbInt(memorydbField(payload, "MaxResults")),
		)
		if err != nil {
			respondMemoryDBErrorForErr(w, err)
			return true
		}
		out := make([]map[string]any, 0, len(groups))
		for _, group := range groups {
			out = append(out, memorydbParameterGroupToAPI(group))
		}
		response := map[string]any{"ParameterGroups": out}
		if nextToken != "" {
			response["NextToken"] = nextToken
		}
		respondMemoryDBJSON(w, http.StatusOK, response)
		return true

	case "UpdateParameterGroup":
		pg, err := s.memorydb.UpdateParameterGroup(
			memorydbString(memorydbField(payload, "ParameterGroupName")),
			memorydbParameterNameValues(memorydbField(payload, "ParameterNameValues")),
		)
		if err != nil {
			respondMemoryDBErrorForErr(w, err)
			return true
		}
		respondMemoryDBJSON(w, http.StatusOK, map[string]any{"ParameterGroup": memorydbParameterGroupToAPI(pg)})
		return true

	case "DeleteParameterGroup":
		pg, err := s.memorydb.DeleteParameterGroup(memorydbString(memorydbField(payload, "ParameterGroupName")))
		if err != nil {
			respondMemoryDBErrorForErr(w, err)
			return true
		}
		respondMemoryDBJSON(w, http.StatusOK, map[string]any{"ParameterGroup": memorydbParameterGroupToAPI(pg)})
		return true

	case "DescribeParameters":
		params, nextToken, err := s.memorydb.DescribeParameters(
			memorydbString(memorydbField(payload, "ParameterGroupName")),
			memorydbString(memorydbField(payload, "NextToken")),
			memorydbInt(memorydbField(payload, "MaxResults")),
		)
		if err != nil {
			respondMemoryDBErrorForErr(w, err)
			return true
		}
		out := make([]map[string]any, 0, len(params))
		for _, param := range params {
			out = append(out, memorydbParameterToAPI(param))
		}
		response := map[string]any{"Parameters": out}
		if nextToken != "" {
			response["NextToken"] = nextToken
		}
		respondMemoryDBJSON(w, http.StatusOK, response)
		return true

	case "ResetParameterGroup":
		allParams := memorydbBool(memorydbField(payload, "AllParameters"))
		pg, err := s.memorydb.ResetParameterGroup(
			memorydbString(memorydbField(payload, "ParameterGroupName")),
			allParams,
			memorydbStringList(memorydbField(payload, "ParameterNames")),
		)
		if err != nil {
			respondMemoryDBErrorForErr(w, err)
			return true
		}
		respondMemoryDBJSON(w, http.StatusOK, map[string]any{"ParameterGroup": memorydbParameterGroupToAPI(pg)})
		return true

	case "CreateSnapshot":
		snapshot, err := s.memorydb.CreateSnapshot(memorydbsvc.CreateSnapshotInput{
			ClusterName:  memorydbString(memorydbField(payload, "ClusterName")),
			SnapshotName: memorydbString(memorydbField(payload, "SnapshotName")),
			KmsKeyID:     memorydbString(memorydbField(payload, "KmsKeyId")),
			Tags:         memorydbTagsToMap(memorydbField(payload, "Tags")),
		})
		if err != nil {
			respondMemoryDBErrorForErr(w, err)
			return true
		}
		respondMemoryDBJSON(w, http.StatusOK, map[string]any{"Snapshot": memorydbSnapshotToAPI(snapshot)})
		return true

	case "CopySnapshot":
		snapshot, err := s.memorydb.CopySnapshot(memorydbsvc.CopySnapshotInput{
			SourceSnapshotName: memorydbString(memorydbField(payload, "SourceSnapshotName")),
			TargetSnapshotName: memorydbString(memorydbField(payload, "TargetSnapshotName")),
			KmsKeyID:           memorydbString(memorydbField(payload, "KmsKeyId")),
			Tags:               memorydbTagsToMap(memorydbField(payload, "Tags")),
		})
		if err != nil {
			respondMemoryDBErrorForErr(w, err)
			return true
		}
		respondMemoryDBJSON(w, http.StatusOK, map[string]any{"Snapshot": memorydbSnapshotToAPI(snapshot)})
		return true

	case "DescribeSnapshots":
		snapshots, nextToken, err := s.memorydb.DescribeSnapshots(
			memorydbString(memorydbField(payload, "ClusterName")),
			memorydbString(memorydbField(payload, "SnapshotName")),
			memorydbString(memorydbField(payload, "Source")),
			memorydbString(memorydbField(payload, "NextToken")),
			memorydbInt(memorydbField(payload, "MaxResults")),
		)
		if err != nil {
			respondMemoryDBErrorForErr(w, err)
			return true
		}
		out := make([]map[string]any, 0, len(snapshots))
		for _, snapshot := range snapshots {
			out = append(out, memorydbSnapshotToAPI(snapshot))
		}
		response := map[string]any{"Snapshots": out}
		if nextToken != "" {
			response["NextToken"] = nextToken
		}
		respondMemoryDBJSON(w, http.StatusOK, response)
		return true

	case "DeleteSnapshot":
		snapshot, err := s.memorydb.DeleteSnapshot(memorydbString(memorydbField(payload, "SnapshotName")))
		if err != nil {
			respondMemoryDBErrorForErr(w, err)
			return true
		}
		respondMemoryDBJSON(w, http.StatusOK, map[string]any{"Snapshot": memorydbSnapshotToAPI(snapshot)})
		return true

	case "CreateMultiRegionCluster":
		mrc, err := s.memorydb.CreateMultiRegionCluster(memorydbsvc.CreateMultiRegionClusterInput{
			NameSuffix:                    memorydbString(memorydbField(payload, "MultiRegionClusterNameSuffix")),
			Description:                   memorydbString(memorydbField(payload, "Description")),
			Engine:                        memorydbString(memorydbField(payload, "Engine")),
			EngineVersion:                 memorydbString(memorydbField(payload, "EngineVersion")),
			NodeType:                      memorydbString(memorydbField(payload, "NodeType")),
			MultiRegionParameterGroupName: memorydbString(memorydbField(payload, "MultiRegionParameterGroupName")),
			NumShards:                     memorydbInt(memorydbField(payload, "NumShards")),
			TLSEnabled:                    memorydbOptionalBoolFieldValue(payload, "TLSEnabled"),
			Tags:                          memorydbTagsToMap(memorydbField(payload, "Tags")),
		})
		if err != nil {
			respondMemoryDBErrorForErr(w, err)
			return true
		}
		respondMemoryDBJSON(w, http.StatusOK, map[string]any{"MultiRegionCluster": memorydbMultiRegionClusterToAPI(mrc)})
		return true

	case "DescribeMultiRegionClusters":
		clusters, nextToken, err := s.memorydb.DescribeMultiRegionClusters(
			memorydbString(memorydbField(payload, "MultiRegionClusterName")),
			memorydbString(memorydbField(payload, "NextToken")),
			memorydbInt(memorydbField(payload, "MaxResults")),
		)
		if err != nil {
			respondMemoryDBErrorForErr(w, err)
			return true
		}
		out := make([]map[string]any, 0, len(clusters))
		for _, cluster := range clusters {
			out = append(out, memorydbMultiRegionClusterToAPI(cluster))
		}
		response := map[string]any{"MultiRegionClusters": out}
		if nextToken != "" {
			response["NextToken"] = nextToken
		}
		respondMemoryDBJSON(w, http.StatusOK, response)
		return true

	case "UpdateMultiRegionCluster":
		update := memorydbsvc.UpdateMultiRegionClusterInput{
			Name: memorydbString(memorydbField(payload, "MultiRegionClusterName")),
		}
		if value, ok := memorydbOptionalStringField(payload, "NodeType"); ok {
			update.NodeType = &value
		}
		if value, ok := memorydbOptionalStringField(payload, "Description"); ok {
			update.Description = &value
		}
		if value, ok := memorydbOptionalStringField(payload, "EngineVersion"); ok {
			update.EngineVersion = &value
		}
		if shardCfg := memorydbMap(memorydbField(payload, "ShardConfiguration")); shardCfg != nil {
			if value, ok := memorydbOptionalIntField(shardCfg, "ShardCount"); ok {
				update.ShardCount = &value
			}
		}
		if value, ok := memorydbOptionalStringField(payload, "MultiRegionParameterGroupName"); ok {
			update.MultiRegionParameterGroupName = &value
		}

		mrc, err := s.memorydb.UpdateMultiRegionCluster(update)
		if err != nil {
			respondMemoryDBErrorForErr(w, err)
			return true
		}
		respondMemoryDBJSON(w, http.StatusOK, map[string]any{"MultiRegionCluster": memorydbMultiRegionClusterToAPI(mrc)})
		return true

	case "DeleteMultiRegionCluster":
		mrc, err := s.memorydb.DeleteMultiRegionCluster(memorydbString(memorydbField(payload, "MultiRegionClusterName")))
		if err != nil {
			respondMemoryDBErrorForErr(w, err)
			return true
		}
		respondMemoryDBJSON(w, http.StatusOK, map[string]any{"MultiRegionCluster": memorydbMultiRegionClusterToAPI(mrc)})
		return true

	case "ListAllowedMultiRegionClusterUpdates":
		scaleUp, scaleDown, err := s.memorydb.ListAllowedMultiRegionClusterUpdates(memorydbString(memorydbField(payload, "MultiRegionClusterName")))
		if err != nil {
			respondMemoryDBErrorForErr(w, err)
			return true
		}
		respondMemoryDBJSON(w, http.StatusOK, map[string]any{
			"ScaleUpNodeTypes":   scaleUp,
			"ScaleDownNodeTypes": scaleDown,
		})
		return true

	case "BatchUpdateCluster":
		serviceUpdateName := ""
		if serviceUpdate := memorydbMap(memorydbField(payload, "ServiceUpdate")); serviceUpdate != nil {
			serviceUpdateName = memorydbString(memorydbField(serviceUpdate, "ServiceUpdateNameToApply"))
		}
		processed, unprocessed, err := s.memorydb.BatchUpdateCluster(
			memorydbStringList(memorydbField(payload, "ClusterNames")),
			serviceUpdateName,
		)
		if err != nil {
			respondMemoryDBErrorForErr(w, err)
			return true
		}
		processedOut := make([]map[string]any, 0, len(processed))
		for _, cluster := range processed {
			processedOut = append(processedOut, memorydbClusterToAPI(cluster))
		}
		unprocessedOut := make([]map[string]any, 0, len(unprocessed))
		for _, entry := range unprocessed {
			unprocessedOut = append(unprocessedOut, memorydbUnprocessedClusterToAPI(entry))
		}
		respondMemoryDBJSON(w, http.StatusOK, map[string]any{
			"ProcessedClusters":   processedOut,
			"UnprocessedClusters": unprocessedOut,
		})
		return true

	case "FailoverShard":
		cluster, err := s.memorydb.FailoverShard(
			memorydbString(memorydbField(payload, "ClusterName")),
			memorydbString(memorydbField(payload, "ShardName")),
		)
		if err != nil {
			respondMemoryDBErrorForErr(w, err)
			return true
		}
		respondMemoryDBJSON(w, http.StatusOK, map[string]any{"Cluster": memorydbClusterToAPI(cluster)})
		return true

	case "DescribeEngineVersions":
		versions, nextToken, err := s.memorydb.DescribeEngineVersions(
			memorydbString(memorydbField(payload, "Engine")),
			memorydbString(memorydbField(payload, "EngineVersion")),
			memorydbString(memorydbField(payload, "ParameterGroupFamily")),
			memorydbString(memorydbField(payload, "NextToken")),
			memorydbInt(memorydbField(payload, "MaxResults")),
			memorydbBool(memorydbField(payload, "DefaultOnly")),
		)
		if err != nil {
			respondMemoryDBErrorForErr(w, err)
			return true
		}
		out := make([]map[string]any, 0, len(versions))
		for _, version := range versions {
			out = append(out, memorydbEngineVersionToAPI(version))
		}
		response := map[string]any{"EngineVersions": out}
		if nextToken != "" {
			response["NextToken"] = nextToken
		}
		respondMemoryDBJSON(w, http.StatusOK, response)
		return true

	case "ListAllowedNodeTypeUpdates":
		scaleUp, scaleDown, err := s.memorydb.ListAllowedNodeTypeUpdates(memorydbString(memorydbField(payload, "ClusterName")))
		if err != nil {
			respondMemoryDBErrorForErr(w, err)
			return true
		}
		respondMemoryDBJSON(w, http.StatusOK, map[string]any{
			"ScaleUpNodeTypes":   scaleUp,
			"ScaleDownNodeTypes": scaleDown,
		})
		return true

	case "DescribeReservedNodes":
		nodes, nextToken, err := s.memorydb.DescribeReservedNodes(
			memorydbString(memorydbField(payload, "ReservationId")),
			memorydbString(memorydbField(payload, "ReservedNodesOfferingId")),
			memorydbString(memorydbField(payload, "NodeType")),
			memorydbString(memorydbField(payload, "Duration")),
			memorydbString(memorydbField(payload, "OfferingType")),
			memorydbString(memorydbField(payload, "NextToken")),
			memorydbInt(memorydbField(payload, "MaxResults")),
		)
		if err != nil {
			respondMemoryDBErrorForErr(w, err)
			return true
		}
		out := make([]map[string]any, 0, len(nodes))
		for _, node := range nodes {
			out = append(out, memorydbReservedNodeToAPI(node))
		}
		response := map[string]any{"ReservedNodes": out}
		if nextToken != "" {
			response["NextToken"] = nextToken
		}
		respondMemoryDBJSON(w, http.StatusOK, response)
		return true

	case "DescribeReservedNodesOfferings":
		offerings, nextToken, err := s.memorydb.DescribeReservedNodesOfferings(
			memorydbString(memorydbField(payload, "ReservedNodesOfferingId")),
			memorydbString(memorydbField(payload, "NodeType")),
			memorydbString(memorydbField(payload, "Duration")),
			memorydbString(memorydbField(payload, "OfferingType")),
			memorydbString(memorydbField(payload, "NextToken")),
			memorydbInt(memorydbField(payload, "MaxResults")),
		)
		if err != nil {
			respondMemoryDBErrorForErr(w, err)
			return true
		}
		out := make([]map[string]any, 0, len(offerings))
		for _, offering := range offerings {
			out = append(out, memorydbReservedNodesOfferingToAPI(offering))
		}
		response := map[string]any{"ReservedNodesOfferings": out}
		if nextToken != "" {
			response["NextToken"] = nextToken
		}
		respondMemoryDBJSON(w, http.StatusOK, response)
		return true

	case "PurchaseReservedNodesOffering":
		node, err := s.memorydb.PurchaseReservedNodesOffering(
			memorydbString(memorydbField(payload, "ReservedNodesOfferingId")),
			memorydbString(memorydbField(payload, "ReservationId")),
			memorydbInt(memorydbField(payload, "NodeCount")),
			memorydbTagsToMap(memorydbField(payload, "Tags")),
		)
		if err != nil {
			respondMemoryDBErrorForErr(w, err)
			return true
		}
		respondMemoryDBJSON(w, http.StatusOK, map[string]any{"ReservedNode": memorydbReservedNodeToAPI(node)})
		return true

	case "TagResource":
		tagList, err := s.memorydb.TagResource(
			memorydbString(memorydbField(payload, "ResourceArn")),
			memorydbTagsToMap(memorydbField(payload, "Tags")),
		)
		if err != nil {
			respondMemoryDBErrorForErr(w, err)
			return true
		}
		respondMemoryDBJSON(w, http.StatusOK, map[string]any{"TagList": memorydbTagListFromMap(tagList)})
		return true

	case "UntagResource":
		tagList, err := s.memorydb.UntagResource(
			memorydbString(memorydbField(payload, "ResourceArn")),
			memorydbStringList(memorydbField(payload, "TagKeys")),
		)
		if err != nil {
			respondMemoryDBErrorForErr(w, err)
			return true
		}
		respondMemoryDBJSON(w, http.StatusOK, map[string]any{"TagList": memorydbTagListFromMap(tagList)})
		return true

	case "ListTags":
		tagList, nextToken, err := s.memorydb.ListTags(
			memorydbString(memorydbField(payload, "ResourceArn")),
			memorydbString(memorydbField(payload, "NextToken")),
			memorydbInt(memorydbField(payload, "MaxResults")),
		)
		if err != nil {
			respondMemoryDBErrorForErr(w, err)
			return true
		}
		response := map[string]any{"TagList": memorydbTagListFromMap(tagList)}
		if nextToken != "" {
			response["NextToken"] = nextToken
		}
		respondMemoryDBJSON(w, http.StatusOK, response)
		return true

	case "DescribeEvents":
		events, nextToken, err := s.memorydb.DescribeEvents(
			memorydbString(memorydbField(payload, "SourceName")),
			memorydbString(memorydbField(payload, "SourceType")),
			memorydbString(memorydbField(payload, "StartTime")),
			memorydbString(memorydbField(payload, "EndTime")),
			memorydbInt(memorydbField(payload, "Duration")),
			memorydbString(memorydbField(payload, "NextToken")),
			memorydbInt(memorydbField(payload, "MaxResults")),
		)
		if err != nil {
			respondMemoryDBErrorForErr(w, err)
			return true
		}
		out := make([]map[string]any, 0, len(events))
		for _, event := range events {
			out = append(out, memorydbEventToAPI(event))
		}
		response := map[string]any{"Events": out}
		if nextToken != "" {
			response["NextToken"] = nextToken
		}
		respondMemoryDBJSON(w, http.StatusOK, response)
		return true

	case "DescribeServiceUpdates":
		updates, nextToken, err := s.memorydb.DescribeServiceUpdates(
			memorydbString(memorydbField(payload, "ServiceUpdateName")),
			memorydbStringList(memorydbField(payload, "ClusterNames")),
			memorydbStringList(memorydbField(payload, "Status")),
			memorydbString(memorydbField(payload, "NextToken")),
			memorydbInt(memorydbField(payload, "MaxResults")),
		)
		if err != nil {
			respondMemoryDBErrorForErr(w, err)
			return true
		}
		out := make([]map[string]any, 0, len(updates))
		for _, update := range updates {
			out = append(out, memorydbServiceUpdateToAPI(update))
		}
		response := map[string]any{"ServiceUpdates": out}
		if nextToken != "" {
			response["NextToken"] = nextToken
		}
		respondMemoryDBJSON(w, http.StatusOK, response)
		return true
	}

	respondMemoryDBError(w, http.StatusNotImplemented, "NotImplementedException", "operation is not implemented")
	return true
}

func isMemoryDBJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}

	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.HasPrefix(target, "AmazonMemoryDB.") {
		return true
	}

	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.1") || strings.Contains(contentType, "application/x-amz-json-1.0") {
		return strings.HasPrefix(target, "AmazonMemoryDB")
	}

	if strings.TrimSpace(sigV4ServiceHint(r)) == "memorydb" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	return strings.Contains(host, ".memorydb.") || strings.HasPrefix(host, "memorydb.")
}

func parseMemoryDBTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "AmazonMemoryDB.") {
		return strings.TrimPrefix(target, "AmazonMemoryDB.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func respondMemoryDBJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondMemoryDBError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondMemoryDBJSON(w, status, memorydbError{Type: code, Message: msg})
}

func respondMemoryDBErrorForErr(w http.ResponseWriter, err error) {
	var faultErr *memorydbsvc.FaultError
	if errors.As(err, &faultErr) {
		respondMemoryDBError(w, http.StatusBadRequest, faultErr.Code, faultErr.Message)
		return
	}
	respondMemoryDBError(w, http.StatusBadRequest, "InvalidParameterValueException", err.Error())
}

func parseMemoryDBPayload(r *http.Request) (map[string]any, error) {
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

func memorydbACLToAPI(acl memorydbsvc.ACL) map[string]any {
	out := map[string]any{
		"Name":                 acl.Name,
		"Status":               acl.Status,
		"UserNames":            memorydbOrEmptyStrings(acl.UserNames),
		"MinimumEngineVersion": acl.MinimumEngineVersion,
		"ARN":                  acl.ARN,
	}
	if len(acl.Clusters) > 0 {
		out["Clusters"] = acl.Clusters
	}
	return out
}

func memorydbUserToAPI(user memorydbsvc.User) map[string]any {
	out := map[string]any{
		"Name":                 user.Name,
		"Status":               user.Status,
		"AccessString":         user.AccessString,
		"ACLNames":             memorydbOrEmptyStrings(user.ACLNames),
		"MinimumEngineVersion": user.MinimumEngineVersion,
		"ARN":                  user.ARN,
	}
	if strings.TrimSpace(user.AuthenticationType) != "" {
		out["Authentication"] = map[string]any{
			"Type":          user.AuthenticationType,
			"PasswordCount": user.PasswordCount,
		}
	}
	return out
}

func memorydbSubnetGroupToAPI(group memorydbsvc.SubnetGroup) map[string]any {
	subnets := make([]map[string]any, 0, len(group.SubnetIDs))
	for _, subnetID := range group.SubnetIDs {
		subnets = append(subnets, map[string]any{
			"Identifier": subnetID,
			"AvailabilityZone": map[string]any{
				"Name": "us-east-1a",
			},
		})
	}

	return map[string]any{
		"Name":        group.Name,
		"Description": group.Description,
		"VpcId":       group.VpcID,
		"Subnets":     subnets,
		"ARN":         group.ARN,
	}
}

func memorydbClusterToAPI(cluster memorydbsvc.Cluster) map[string]any {
	securityGroups := make([]map[string]any, 0, len(cluster.SecurityGroupIDs))
	for _, groupID := range cluster.SecurityGroupIDs {
		securityGroups = append(securityGroups, map[string]any{
			"SecurityGroupId": groupID,
			"Status":          "active",
		})
	}

	endpoint := map[string]any{
		"Address": fmt.Sprintf("%s.%s.memorydb.amazonaws.com", cluster.Name, "us-east-1"),
		"Port":    cluster.Port,
	}

	return map[string]any{
		"Name":                    cluster.Name,
		"Description":             cluster.Description,
		"Status":                  cluster.Status,
		"NumberOfShards":          cluster.NumberOfShards,
		"ClusterEndpoint":         endpoint,
		"NodeType":                cluster.NodeType,
		"EngineVersion":           cluster.EngineVersion,
		"ParameterGroupName":      cluster.ParameterGroupName,
		"SecurityGroups":          securityGroups,
		"SubnetGroupName":         cluster.SubnetGroupName,
		"TLSEnabled":              cluster.TLSEnabled,
		"KmsKeyId":                cluster.KmsKeyID,
		"ARN":                     cluster.ARN,
		"SnsTopicArn":             cluster.SnsTopicArn,
		"SnsTopicStatus":          cluster.SnsTopicStatus,
		"SnapshotRetentionLimit":  cluster.SnapshotRetentionLimit,
		"MaintenanceWindow":       cluster.MaintenanceWindow,
		"SnapshotWindow":          cluster.SnapshotWindow,
		"ACLName":                 cluster.ACLName,
		"AutoMinorVersionUpgrade": cluster.AutoMinorVersionUpgrade,
		"DataTiering":             cluster.DataTiering,
	}
}

func memorydbParameterGroupToAPI(group memorydbsvc.ParameterGroup) map[string]any {
	return map[string]any{
		"Name":        group.Name,
		"Family":      group.Family,
		"Description": group.Description,
		"ARN":         group.ARN,
	}
}

func memorydbParameterToAPI(param memorydbsvc.Parameter) map[string]any {
	return map[string]any{
		"Name":                 param.Name,
		"Value":                param.Value,
		"Description":          param.Description,
		"DataType":             param.DataType,
		"AllowedValues":        param.AllowedValues,
		"MinimumEngineVersion": param.MinimumEngineVersion,
	}
}

func memorydbSnapshotToAPI(snapshot memorydbsvc.Snapshot) map[string]any {
	out := map[string]any{
		"Name":        snapshot.Name,
		"Status":      snapshot.Status,
		"Source":      snapshot.Source,
		"KmsKeyId":    snapshot.KmsKeyID,
		"ARN":         snapshot.ARN,
		"DataTiering": snapshot.DataTiering,
	}
	if strings.TrimSpace(snapshot.ClusterName) != "" {
		out["ClusterConfiguration"] = map[string]any{"Name": snapshot.ClusterName}
	}
	return out
}

func memorydbMultiRegionClusterToAPI(cluster memorydbsvc.MultiRegionCluster) map[string]any {
	regional := make([]map[string]any, 0, len(cluster.Clusters))
	for _, entry := range cluster.Clusters {
		regional = append(regional, map[string]any{
			"ClusterName": entry.ClusterName,
			"Region":      entry.Region,
			"Status":      entry.Status,
			"ARN":         entry.ARN,
		})
	}
	return map[string]any{
		"MultiRegionClusterName":        cluster.Name,
		"Description":                   cluster.Description,
		"Status":                        cluster.Status,
		"NodeType":                      cluster.NodeType,
		"Engine":                        cluster.Engine,
		"EngineVersion":                 cluster.EngineVersion,
		"NumberOfShards":                cluster.NumberOfShards,
		"Clusters":                      regional,
		"MultiRegionParameterGroupName": cluster.MultiRegionParameterGroupName,
		"TLSEnabled":                    cluster.TLSEnabled,
		"ARN":                           cluster.ARN,
	}
}

func memorydbUnprocessedClusterToAPI(cluster memorydbsvc.UnprocessedCluster) map[string]any {
	return map[string]any{
		"ClusterName":  cluster.ClusterName,
		"ErrorType":    cluster.ErrorType,
		"ErrorMessage": cluster.ErrorMessage,
	}
}

func memorydbEngineVersionToAPI(version memorydbsvc.EngineVersion) map[string]any {
	return map[string]any{
		"EngineVersion":        version.EngineVersion,
		"EnginePatchVersion":   version.EnginePatchVersion,
		"ParameterGroupFamily": version.ParameterGroupFamily,
	}
}

func memorydbRecurringChargeToAPI(charge memorydbsvc.RecurringCharge) map[string]any {
	return map[string]any{
		"RecurringChargeAmount":    charge.Amount,
		"RecurringChargeFrequency": charge.Frequency,
	}
}

func memorydbReservedNodeToAPI(node memorydbsvc.ReservedNode) map[string]any {
	charges := make([]map[string]any, 0, len(node.RecurringCharges))
	for _, charge := range node.RecurringCharges {
		charges = append(charges, memorydbRecurringChargeToAPI(charge))
	}
	return map[string]any{
		"ReservationId":           node.ReservationID,
		"ReservedNodesOfferingId": node.ReservedNodesOfferingID,
		"NodeType":                node.NodeType,
		"StartTime":               node.StartTime,
		"Duration":                node.Duration,
		"FixedPrice":              node.FixedPrice,
		"NodeCount":               node.NodeCount,
		"OfferingType":            node.OfferingType,
		"State":                   node.State,
		"RecurringCharges":        charges,
		"ARN":                     node.ARN,
	}
}

func memorydbReservedNodesOfferingToAPI(offering memorydbsvc.ReservedNodeOffering) map[string]any {
	charges := make([]map[string]any, 0, len(offering.RecurringCharges))
	for _, charge := range offering.RecurringCharges {
		charges = append(charges, memorydbRecurringChargeToAPI(charge))
	}
	return map[string]any{
		"ReservedNodesOfferingId": offering.ReservedNodesOfferingID,
		"NodeType":                offering.NodeType,
		"Duration":                offering.Duration,
		"FixedPrice":              offering.FixedPrice,
		"OfferingType":            offering.OfferingType,
		"RecurringCharges":        charges,
	}
}

func memorydbEventToAPI(event memorydbsvc.Event) map[string]any {
	return map[string]any{
		"SourceName": event.SourceName,
		"SourceType": event.SourceType,
		"Message":    event.Message,
		"Date":       event.Date,
	}
}

func memorydbServiceUpdateToAPI(update memorydbsvc.ServiceUpdate) map[string]any {
	return map[string]any{
		"ClusterName":         update.ClusterName,
		"ServiceUpdateName":   update.ServiceUpdateName,
		"ReleaseDate":         update.ReleaseDate,
		"Description":         update.Description,
		"Status":              update.Status,
		"Type":                update.Type,
		"NodesUpdated":        update.NodesUpdated,
		"AutoUpdateStartDate": update.AutoUpdateStartDate,
	}
}

func memorydbTagListFromMap(tags map[string]string) []map[string]any {
	if len(tags) == 0 {
		return []map[string]any{}
	}
	keys := make([]string, 0, len(tags))
	for key := range tags {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, map[string]any{
			"Key":   key,
			"Value": tags[key],
		})
	}
	return out
}

func memorydbField(payload map[string]any, key string) any {
	if payload == nil {
		return nil
	}
	if value, ok := payload[key]; ok {
		return value
	}
	for existing, value := range payload {
		if strings.EqualFold(existing, key) {
			return value
		}
	}
	return nil
}

func memorydbOptionalField(payload map[string]any, key string) (any, bool) {
	if payload == nil {
		return nil, false
	}
	if value, ok := payload[key]; ok {
		return value, true
	}
	for existing, value := range payload {
		if strings.EqualFold(existing, key) {
			return value, true
		}
	}
	return nil, false
}

func memorydbFieldExists(payload map[string]any, key string) bool {
	_, ok := memorydbOptionalField(payload, key)
	return ok
}

func memorydbMap(value any) map[string]any {
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

func memorydbString(value any) string {
	raw, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(raw)
}

func memorydbOptionalStringField(payload map[string]any, key string) (string, bool) {
	value, ok := memorydbOptionalField(payload, key)
	if !ok {
		return "", false
	}
	return memorydbString(value), true
}

func memorydbInt(value any) int {
	switch raw := value.(type) {
	case int:
		return raw
	case int32:
		return int(raw)
	case int64:
		return int(raw)
	case float32:
		return int(raw)
	case float64:
		return int(raw)
	default:
		return 0
	}
}

func memorydbOptionalIntField(payload map[string]any, key string) (int, bool) {
	value, ok := memorydbOptionalField(payload, key)
	if !ok {
		return 0, false
	}
	return memorydbInt(value), true
}

func memorydbBool(value any) bool {
	switch raw := value.(type) {
	case bool:
		return raw
	case string:
		trimmed := strings.TrimSpace(strings.ToLower(raw))
		return trimmed == "true" || trimmed == "1"
	default:
		return false
	}
}

func memorydbOptionalBoolFieldValue(payload map[string]any, key string) *bool {
	value, ok := memorydbOptionalField(payload, key)
	if !ok {
		return nil
	}
	boolValue := memorydbBool(value)
	return &boolValue
}

func memorydbStringList(value any) []string {
	switch raw := value.(type) {
	case []string:
		out := make([]string, 0, len(raw))
		for _, entry := range raw {
			if trimmed := strings.TrimSpace(entry); trimmed != "" {
				out = append(out, trimmed)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(raw))
		for _, entry := range raw {
			if trimmed := memorydbString(entry); trimmed != "" {
				out = append(out, trimmed)
			}
		}
		return out
	default:
		return nil
	}
}

func memorydbTagsToMap(value any) map[string]string {
	raw, ok := value.([]any)
	if !ok {
		return nil
	}
	out := map[string]string{}
	for _, entry := range raw {
		entryMap, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		key := memorydbString(memorydbField(entryMap, "Key"))
		if key == "" {
			key = memorydbString(memorydbField(entryMap, "key"))
		}
		if key == "" {
			continue
		}
		value := memorydbString(memorydbField(entryMap, "Value"))
		if value == "" {
			value = memorydbString(memorydbField(entryMap, "value"))
		}
		out[key] = value
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func memorydbParameterNameValues(value any) map[string]string {
	raw, ok := value.([]any)
	if !ok {
		return nil
	}
	out := map[string]string{}
	for _, entry := range raw {
		entryMap, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		name := memorydbString(memorydbField(entryMap, "ParameterName"))
		if name == "" {
			continue
		}
		out[name] = memorydbString(memorydbField(entryMap, "ParameterValue"))
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func memorydbAuthenticationInput(value any) (string, int) {
	mode, ok := value.(map[string]any)
	if !ok {
		return "", 0
	}
	authType := memorydbString(memorydbField(mode, "Type"))
	passwordCount := len(memorydbStringList(memorydbField(mode, "Passwords")))
	if passwordCount == 0 {
		passwordCount = 1
	}
	return authType, passwordCount
}

func memorydbOrEmptyStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	out := make([]string, len(values))
	copy(out, values)
	return out
}
