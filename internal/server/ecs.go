package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	ecssvc "github.com/stackyard/stackyard/internal/services/ecs"
)

type ecsError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleECSJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isECSJSONCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "ecs")
	if !ok {
		respondECSError(w, status, code, msg)
		return true
	}

	action := parseECSTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondECSError(w, http.StatusBadRequest, "ClientException", "missing X-Amz-Target")
		return true
	}
	if _, known := ecsOperationByName[action]; !known {
		respondECSError(w, http.StatusBadRequest, "ClientException", "unknown action")
		return true
	}

	payload, err := parseECSPayload(r)
	if err != nil {
		respondECSError(w, http.StatusBadRequest, "ClientException", "invalid JSON body")
		return true
	}

	switch action {
	case "PutAccountSetting":
		setting, err := s.ecs.PutAccountSetting(
			ecsString(payload["name"]),
			ecsString(payload["value"]),
			ecsString(payload["principalArn"]),
		)
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		respondECSJSON(w, http.StatusOK, map[string]any{
			"setting": ecsAccountSettingPayload(setting),
		})
		return true
	case "PutAccountSettingDefault":
		setting, err := s.ecs.PutAccountSettingDefault(
			ecsString(payload["name"]),
			ecsString(payload["value"]),
		)
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		respondECSJSON(w, http.StatusOK, map[string]any{
			"setting": ecsAccountSettingPayload(setting),
		})
		return true
	case "ListAccountSettings":
		effectiveSettings, _ := ecsBool(payload["effectiveSettings"])
		settings, err := s.ecs.ListAccountSettings(
			ecsStringSlice(payload["name"]),
			effectiveSettings,
			ecsString(payload["principalArn"]),
		)
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		out := make([]map[string]any, 0, len(settings))
		for _, setting := range settings {
			out = append(out, ecsAccountSettingPayload(setting))
		}
		respondECSJSON(w, http.StatusOK, map[string]any{
			"settings": out,
		})
		return true
	case "DeleteAccountSetting":
		if err := s.ecs.DeleteAccountSetting(
			ecsString(payload["name"]),
			ecsString(payload["principalArn"]),
		); err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		respondECSJSON(w, http.StatusOK, map[string]any{})
		return true
	case "CreateCluster":
		settings, err := ecsClusterSettings(payload["settings"])
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		tags, err := ecsTagsToMap(payload["tags"])
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		cluster, err := s.ecs.CreateCluster(
			ecsString(payload["clusterName"]),
			settings,
			tags,
		)
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		respondECSJSON(w, http.StatusOK, map[string]any{
			"cluster": ecsClusterPayload(cluster),
		})
		return true
	case "DescribeClusters":
		clusters, failures, err := s.ecs.DescribeClusters(ecsStringSlice(payload["clusters"]))
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		clusterOut := make([]map[string]any, 0, len(clusters))
		for _, cluster := range clusters {
			clusterOut = append(clusterOut, ecsClusterPayload(cluster))
		}
		failureOut := make([]map[string]any, 0, len(failures))
		for _, failure := range failures {
			failureOut = append(failureOut, ecsFailurePayload(failure))
		}
		respondECSJSON(w, http.StatusOK, map[string]any{
			"clusters": clusterOut,
			"failures": failureOut,
		})
		return true
	case "ListClusters":
		maxResults, _ := ecsInt32(payload["maxResults"])
		clusterARNs, nextToken, err := s.ecs.ListClusters(
			ecsString(payload["nextToken"]),
			maxResults,
		)
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		response := map[string]any{
			"clusterArns": clusterARNs,
		}
		if nextToken != "" {
			response["nextToken"] = nextToken
		}
		respondECSJSON(w, http.StatusOK, response)
		return true
	case "UpdateClusterSettings":
		settings, err := ecsClusterSettings(payload["settings"])
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		cluster, err := s.ecs.UpdateClusterSettings(
			ecsString(payload["cluster"]),
			settings,
		)
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		respondECSJSON(w, http.StatusOK, map[string]any{
			"cluster": ecsClusterPayload(cluster),
		})
		return true
	case "UpdateCluster":
		settings, err := ecsClusterSettings(payload["settings"])
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		configuration, _, err := ecsOptionalMap(payload, "configuration")
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		serviceConnectDefaults, hasServiceConnectDefaults, err := ecsOptionalMap(payload, "serviceConnectDefaults")
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		var serviceConnectNamespace string
		if hasServiceConnectDefaults {
			serviceConnectNamespace = ecsString(serviceConnectDefaults["namespace"])
		}
		cluster, err := s.ecs.UpdateCluster(
			ecsString(payload["cluster"]),
			settings,
			configuration,
			serviceConnectNamespace,
			hasServiceConnectDefaults,
		)
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		respondECSJSON(w, http.StatusOK, map[string]any{
			"cluster": ecsClusterPayload(cluster),
		})
		return true
	case "DeleteCluster":
		cluster, err := s.ecs.DeleteCluster(ecsString(payload["cluster"]))
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		respondECSJSON(w, http.StatusOK, map[string]any{
			"cluster": ecsClusterPayload(cluster),
		})
		return true
	case "CreateCapacityProvider":
		autoScalingGroupProvider := ecsMap(payload["autoScalingGroupProvider"])
		managedScaling := ecsMap(autoScalingGroupProvider["managedScaling"])
		tags, err := ecsTagsToMap(payload["tags"])
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		cp, err := s.ecs.CreateCapacityProvider(
			ecsString(payload["name"]),
			ecsString(autoScalingGroupProvider["autoScalingGroupArn"]),
			ecsString(managedScaling["status"]),
			ecsString(autoScalingGroupProvider["managedTerminationProtection"]),
			tags,
		)
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		respondECSJSON(w, http.StatusOK, map[string]any{
			"capacityProvider": ecsCapacityProviderPayload(cp),
		})
		return true
	case "DescribeCapacityProviders":
		cps, failures, err := s.ecs.DescribeCapacityProviders(ecsStringSlice(payload["capacityProviders"]))
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		cpOut := make([]map[string]any, 0, len(cps))
		for _, cp := range cps {
			cpOut = append(cpOut, ecsCapacityProviderPayload(cp))
		}
		failureOut := make([]map[string]any, 0, len(failures))
		for _, failure := range failures {
			failureOut = append(failureOut, ecsFailurePayload(failure))
		}
		respondECSJSON(w, http.StatusOK, map[string]any{
			"capacityProviders": cpOut,
			"failures":          failureOut,
		})
		return true
	case "UpdateCapacityProvider":
		autoScalingGroupProvider := ecsMap(payload["autoScalingGroupProvider"])
		managedScaling := ecsMap(autoScalingGroupProvider["managedScaling"])
		cp, err := s.ecs.UpdateCapacityProvider(
			ecsString(payload["name"]),
			ecsString(autoScalingGroupProvider["autoScalingGroupArn"]),
			ecsString(managedScaling["status"]),
			ecsString(autoScalingGroupProvider["managedTerminationProtection"]),
		)
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		respondECSJSON(w, http.StatusOK, map[string]any{
			"capacityProvider": ecsCapacityProviderPayload(cp),
		})
		return true
	case "DeleteCapacityProvider":
		ref := ecsString(payload["capacityProvider"])
		if ref == "" {
			ref = ecsString(payload["name"])
		}
		cp, err := s.ecs.DeleteCapacityProvider(ref)
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		respondECSJSON(w, http.StatusOK, map[string]any{
			"capacityProvider": ecsCapacityProviderPayload(cp),
		})
		return true
	case "PutClusterCapacityProviders":
		strategy, err := ecsCapacityProviderStrategy(payload["defaultCapacityProviderStrategy"])
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		cluster, err := s.ecs.PutClusterCapacityProviders(
			ecsString(payload["cluster"]),
			ecsStringSlice(payload["capacityProviders"]),
			strategy,
		)
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		respondECSJSON(w, http.StatusOK, map[string]any{
			"cluster": ecsClusterPayload(cluster),
		})
		return true
	case "RegisterTaskDefinition":
		containerDefinitions, err := ecsContainerDefinitions(payload["containerDefinitions"])
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		tags, err := ecsTagsToMap(payload["tags"])
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		taskDefinition, err := s.ecs.RegisterTaskDefinition(ecssvc.TaskDefinitionInput{
			Family:                  ecsString(payload["family"]),
			NetworkMode:             ecsString(payload["networkMode"]),
			Cpu:                     ecsString(payload["cpu"]),
			Memory:                  ecsString(payload["memory"]),
			ExecutionRoleArn:        ecsString(payload["executionRoleArn"]),
			TaskRoleArn:             ecsString(payload["taskRoleArn"]),
			RequiresCompatibilities: ecsStringSlice(payload["requiresCompatibilities"]),
			ContainerDefinitions:    containerDefinitions,
			Tags:                    tags,
		})
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		respondECSJSON(w, http.StatusOK, map[string]any{
			"taskDefinition": ecsTaskDefinitionPayload(taskDefinition),
		})
		return true
	case "DescribeTaskDefinition":
		taskDefinition, err := s.ecs.DescribeTaskDefinition(ecsString(payload["taskDefinition"]))
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		respondECSJSON(w, http.StatusOK, map[string]any{
			"taskDefinition": ecsTaskDefinitionPayload(taskDefinition),
			"tags":           ecsMapToTags(taskDefinition.Tags),
		})
		return true
	case "ListTaskDefinitions":
		maxResults, _ := ecsInt32(payload["maxResults"])
		taskDefinitionARNs, nextToken, err := s.ecs.ListTaskDefinitions(
			ecsString(payload["familyPrefix"]),
			ecsString(payload["status"]),
			ecsString(payload["sort"]),
			ecsString(payload["nextToken"]),
			maxResults,
		)
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		response := map[string]any{
			"taskDefinitionArns": taskDefinitionARNs,
		}
		if nextToken != "" {
			response["nextToken"] = nextToken
		}
		respondECSJSON(w, http.StatusOK, response)
		return true
	case "ListTaskDefinitionFamilies":
		maxResults, _ := ecsInt32(payload["maxResults"])
		families, nextToken, err := s.ecs.ListTaskDefinitionFamilies(
			ecsString(payload["familyPrefix"]),
			ecsString(payload["status"]),
			ecsString(payload["nextToken"]),
			maxResults,
		)
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		response := map[string]any{
			"families": families,
		}
		if nextToken != "" {
			response["nextToken"] = nextToken
		}
		respondECSJSON(w, http.StatusOK, response)
		return true
	case "DeregisterTaskDefinition":
		taskDefinition, err := s.ecs.DeregisterTaskDefinition(ecsString(payload["taskDefinition"]))
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		respondECSJSON(w, http.StatusOK, map[string]any{
			"taskDefinition": ecsTaskDefinitionPayload(taskDefinition),
		})
		return true
	case "DeleteTaskDefinitions":
		taskDefinitions, failures, err := s.ecs.DeleteTaskDefinitions(ecsStringSlice(payload["taskDefinitions"]))
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		taskDefOut := make([]map[string]any, 0, len(taskDefinitions))
		for _, taskDefinition := range taskDefinitions {
			taskDefOut = append(taskDefOut, ecsTaskDefinitionPayload(taskDefinition))
		}
		failureOut := make([]map[string]any, 0, len(failures))
		for _, failure := range failures {
			failureOut = append(failureOut, ecsFailurePayload(failure))
		}
		respondECSJSON(w, http.StatusOK, map[string]any{
			"taskDefinitions": taskDefOut,
			"failures":        failureOut,
		})
		return true
	case "CreateService":
		tags, err := ecsTagsToMap(payload["tags"])
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		desiredCount, hasDesiredCount := ecsInt32(payload["desiredCount"])
		service, err := s.ecs.CreateService(
			ecsString(payload["cluster"]),
			ecsString(payload["serviceName"]),
			ecsString(payload["taskDefinition"]),
			ecsString(payload["launchType"]),
			desiredCount,
			hasDesiredCount,
			tags,
		)
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		respondECSJSON(w, http.StatusOK, map[string]any{
			"service": ecsServicePayload(service),
		})
		return true
	case "CreateExpressGatewayService":
		tags, err := ecsTagsToMap(payload["tags"])
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		primaryContainer, _, err := ecsOptionalMap(payload, "primaryContainer")
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		networkConfiguration, hasNetworkConfiguration, err := ecsOptionalMap(payload, "networkConfiguration")
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		scalingTarget, hasScalingTarget, err := ecsOptionalMap(payload, "scalingTarget")
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		if !hasNetworkConfiguration {
			networkConfiguration = nil
		}
		if !hasScalingTarget {
			scalingTarget = nil
		}
		service, err := s.ecs.CreateExpressGatewayService(
			ecsString(payload["cluster"]),
			ecsString(payload["serviceName"]),
			ecsString(payload["executionRoleArn"]),
			ecsString(payload["infrastructureRoleArn"]),
			ecsString(payload["taskRoleArn"]),
			ecsString(payload["cpu"]),
			ecsString(payload["memory"]),
			ecsString(payload["healthCheckPath"]),
			primaryContainer,
			networkConfiguration,
			scalingTarget,
			tags,
		)
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		respondECSJSON(w, http.StatusOK, map[string]any{
			"service": ecsExpressGatewayServicePayload(service),
		})
		return true
	case "DescribeExpressGatewayService":
		service, err := s.ecs.DescribeExpressGatewayService(
			ecsString(payload["serviceArn"]),
			ecsIncludeTags(payload["include"]),
		)
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		respondECSJSON(w, http.StatusOK, map[string]any{
			"service": ecsExpressGatewayServicePayload(service),
		})
		return true
	case "UpdateExpressGatewayService":
		primaryContainer, hasPrimaryContainer, err := ecsOptionalMap(payload, "primaryContainer")
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		networkConfiguration, hasNetworkConfiguration, err := ecsOptionalMap(payload, "networkConfiguration")
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		scalingTarget, hasScalingTarget, err := ecsOptionalMap(payload, "scalingTarget")
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		if !hasPrimaryContainer {
			primaryContainer = nil
		}
		if !hasNetworkConfiguration {
			networkConfiguration = nil
		}
		if !hasScalingTarget {
			scalingTarget = nil
		}
		service, err := s.ecs.UpdateExpressGatewayService(
			ecsString(payload["serviceArn"]),
			ecsString(payload["executionRoleArn"]),
			ecsString(payload["taskRoleArn"]),
			ecsString(payload["cpu"]),
			ecsString(payload["memory"]),
			ecsString(payload["healthCheckPath"]),
			primaryContainer,
			networkConfiguration,
			scalingTarget,
		)
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		respondECSJSON(w, http.StatusOK, map[string]any{
			"service": ecsUpdatedExpressGatewayServicePayload(service),
		})
		return true
	case "DeleteExpressGatewayService":
		service, err := s.ecs.DeleteExpressGatewayService(ecsString(payload["serviceArn"]))
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		respondECSJSON(w, http.StatusOK, map[string]any{
			"service": ecsExpressGatewayServicePayload(service),
		})
		return true
	case "DescribeServices":
		services, failures, err := s.ecs.DescribeServices(
			ecsString(payload["cluster"]),
			ecsStringSlice(payload["services"]),
		)
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		serviceOut := make([]map[string]any, 0, len(services))
		for _, service := range services {
			serviceOut = append(serviceOut, ecsServicePayload(service))
		}
		failureOut := make([]map[string]any, 0, len(failures))
		for _, failure := range failures {
			failureOut = append(failureOut, ecsFailurePayload(failure))
		}
		respondECSJSON(w, http.StatusOK, map[string]any{
			"services": serviceOut,
			"failures": failureOut,
		})
		return true
	case "ListServices":
		maxResults, _ := ecsInt32(payload["maxResults"])
		serviceARNs, nextToken, err := s.ecs.ListServices(
			ecsString(payload["cluster"]),
			ecsString(payload["nextToken"]),
			maxResults,
		)
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		response := map[string]any{
			"serviceArns": serviceARNs,
		}
		if nextToken != "" {
			response["nextToken"] = nextToken
		}
		respondECSJSON(w, http.StatusOK, response)
		return true
	case "ListServicesByLaunchType":
		maxResults, _ := ecsInt32(payload["maxResults"])
		serviceARNs, nextToken, err := s.ecs.ListServicesByLaunchType(
			ecsString(payload["cluster"]),
			ecsString(payload["launchType"]),
			ecsString(payload["nextToken"]),
			maxResults,
		)
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		response := map[string]any{
			"serviceArns": serviceARNs,
		}
		if nextToken != "" {
			response["nextToken"] = nextToken
		}
		respondECSJSON(w, http.StatusOK, response)
		return true
	case "ListServicesByNamespace":
		maxResults, _ := ecsInt32(payload["maxResults"])
		namespace := ecsString(payload["namespace"])
		if namespace == "" {
			namespace = ecsString(payload["namespaceName"])
		}
		serviceARNs, nextToken, err := s.ecs.ListServicesByNamespace(
			ecsString(payload["cluster"]),
			namespace,
			ecsString(payload["nextToken"]),
			maxResults,
		)
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		response := map[string]any{
			"serviceArns": serviceARNs,
		}
		if nextToken != "" {
			response["nextToken"] = nextToken
		}
		respondECSJSON(w, http.StatusOK, response)
		return true
	case "UpdateService":
		desiredCount, hasDesiredCount := ecsInt32(payload["desiredCount"])
		service, err := s.ecs.UpdateService(
			ecsString(payload["cluster"]),
			ecsString(payload["service"]),
			ecsString(payload["taskDefinition"]),
			ecsString(payload["launchType"]),
			desiredCount,
			hasDesiredCount,
		)
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		respondECSJSON(w, http.StatusOK, map[string]any{
			"service": ecsServicePayload(service),
		})
		return true
	case "DeleteService":
		service, err := s.ecs.DeleteService(
			ecsString(payload["cluster"]),
			ecsString(payload["service"]),
		)
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		respondECSJSON(w, http.StatusOK, map[string]any{
			"service": ecsServicePayload(service),
		})
		return true
	case "ListServiceDeployments":
		maxResults, _ := ecsInt32(payload["maxResults"])
		deployments, nextToken, err := s.ecs.ListServiceDeployments(
			ecsString(payload["cluster"]),
			ecsString(payload["service"]),
			ecsString(payload["nextToken"]),
			maxResults,
		)
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		deploymentOut := make([]map[string]any, 0, len(deployments))
		for _, deployment := range deployments {
			deploymentOut = append(deploymentOut, ecsServiceDeploymentPayload(deployment))
		}
		response := map[string]any{
			"serviceDeployments": deploymentOut,
		}
		if nextToken != "" {
			response["nextToken"] = nextToken
		}
		respondECSJSON(w, http.StatusOK, response)
		return true
	case "DescribeServiceDeployments":
		deployments, failures, err := s.ecs.DescribeServiceDeployments(
			ecsString(payload["cluster"]),
			ecsString(payload["service"]),
			ecsStringSlice(payload["serviceDeployments"]),
		)
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		deploymentOut := make([]map[string]any, 0, len(deployments))
		for _, deployment := range deployments {
			deploymentOut = append(deploymentOut, ecsServiceDeploymentPayload(deployment))
		}
		failureOut := make([]map[string]any, 0, len(failures))
		for _, failure := range failures {
			failureOut = append(failureOut, ecsFailurePayload(failure))
		}
		respondECSJSON(w, http.StatusOK, map[string]any{
			"serviceDeployments": deploymentOut,
			"failures":           failureOut,
		})
		return true
	case "StopServiceDeployment":
		deployment, err := s.ecs.StopServiceDeployment(
			ecsString(payload["cluster"]),
			ecsString(payload["service"]),
			ecsString(payload["serviceDeployment"]),
			ecsString(payload["stopReason"]),
		)
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		respondECSJSON(w, http.StatusOK, map[string]any{
			"serviceDeployment": ecsServiceDeploymentPayload(deployment),
		})
		return true
	case "ListServiceDeploymentsByCreatedAt":
		maxResults, _ := ecsInt32(payload["maxResults"])
		deployments, nextToken, err := s.ecs.ListServiceDeploymentsByCreatedAt(
			ecsString(payload["cluster"]),
			ecsString(payload["service"]),
			ecsString(payload["sortOrder"]),
			ecsString(payload["nextToken"]),
			maxResults,
		)
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		deploymentOut := make([]map[string]any, 0, len(deployments))
		for _, deployment := range deployments {
			deploymentOut = append(deploymentOut, ecsServiceDeploymentPayload(deployment))
		}
		response := map[string]any{
			"serviceDeployments": deploymentOut,
		}
		if nextToken != "" {
			response["nextToken"] = nextToken
		}
		respondECSJSON(w, http.StatusOK, response)
		return true
	case "ListServiceDeploymentsByServiceRevision":
		maxResults, _ := ecsInt32(payload["maxResults"])
		deployments, nextToken, err := s.ecs.ListServiceDeploymentsByServiceRevision(
			ecsString(payload["cluster"]),
			ecsString(payload["service"]),
			ecsString(payload["serviceRevision"]),
			ecsString(payload["nextToken"]),
			maxResults,
		)
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		deploymentOut := make([]map[string]any, 0, len(deployments))
		for _, deployment := range deployments {
			deploymentOut = append(deploymentOut, ecsServiceDeploymentPayload(deployment))
		}
		response := map[string]any{
			"serviceDeployments": deploymentOut,
		}
		if nextToken != "" {
			response["nextToken"] = nextToken
		}
		respondECSJSON(w, http.StatusOK, response)
		return true
	case "DescribeServiceRevisions":
		revisions, failures, err := s.ecs.DescribeServiceRevisions(
			ecsString(payload["cluster"]),
			ecsString(payload["service"]),
			ecsStringSlice(payload["serviceRevisions"]),
		)
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		revisionOut := make([]map[string]any, 0, len(revisions))
		for _, revision := range revisions {
			revisionOut = append(revisionOut, ecsServiceRevisionPayload(revision))
		}
		failureOut := make([]map[string]any, 0, len(failures))
		for _, failure := range failures {
			failureOut = append(failureOut, ecsFailurePayload(failure))
		}
		respondECSJSON(w, http.StatusOK, map[string]any{
			"serviceRevisions": revisionOut,
			"failures":         failureOut,
		})
		return true
	case "CreateTaskSet":
		scale := ecsMap(payload["scale"])
		scaleValue, hasScaleValue := ecsFloat64(scale["value"])
		taskSet, err := s.ecs.CreateTaskSet(
			ecsString(payload["cluster"]),
			ecsString(payload["service"]),
			ecsString(payload["taskDefinition"]),
			ecsString(payload["launchType"]),
			scaleValue,
			hasScaleValue,
		)
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		respondECSJSON(w, http.StatusOK, map[string]any{
			"taskSet": ecsTaskSetPayload(taskSet),
		})
		return true
	case "DescribeTaskSets":
		taskSets, failures, err := s.ecs.DescribeTaskSets(
			ecsString(payload["cluster"]),
			ecsString(payload["service"]),
			ecsStringSlice(payload["taskSets"]),
		)
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		taskSetOut := make([]map[string]any, 0, len(taskSets))
		for _, taskSet := range taskSets {
			taskSetOut = append(taskSetOut, ecsTaskSetPayload(taskSet))
		}
		failureOut := make([]map[string]any, 0, len(failures))
		for _, failure := range failures {
			failureOut = append(failureOut, ecsFailurePayload(failure))
		}
		respondECSJSON(w, http.StatusOK, map[string]any{
			"taskSets": taskSetOut,
			"failures": failureOut,
		})
		return true
	case "UpdateTaskSet":
		scale := ecsMap(payload["scale"])
		scaleValue, hasScaleValue := ecsFloat64(scale["value"])
		taskSet, err := s.ecs.UpdateTaskSet(
			ecsString(payload["cluster"]),
			ecsString(payload["service"]),
			ecsString(payload["taskSet"]),
			scaleValue,
			hasScaleValue,
		)
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		respondECSJSON(w, http.StatusOK, map[string]any{
			"taskSet": ecsTaskSetPayload(taskSet),
		})
		return true
	case "UpdateServicePrimaryTaskSet":
		service, err := s.ecs.UpdateServicePrimaryTaskSet(
			ecsString(payload["cluster"]),
			ecsString(payload["service"]),
			ecsString(payload["primaryTaskSet"]),
		)
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		respondECSJSON(w, http.StatusOK, map[string]any{
			"service": ecsServicePayload(service),
		})
		return true
	case "DeleteTaskSet":
		taskSet, err := s.ecs.DeleteTaskSet(
			ecsString(payload["cluster"]),
			ecsString(payload["service"]),
			ecsString(payload["taskSet"]),
		)
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		respondECSJSON(w, http.StatusOK, map[string]any{
			"taskSet": ecsTaskSetPayload(taskSet),
		})
		return true
	case "RunTask":
		count, hasCount := ecsInt32(payload["count"])
		tasks, failures, err := s.ecs.RunTask(
			ecsString(payload["cluster"]),
			ecsString(payload["taskDefinition"]),
			ecsString(payload["launchType"]),
			ecsString(payload["startedBy"]),
			ecsString(payload["group"]),
			ecsString(payload["service"]),
			count,
			hasCount,
		)
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		taskOut := make([]map[string]any, 0, len(tasks))
		for _, task := range tasks {
			taskOut = append(taskOut, ecsTaskPayload(task))
		}
		failureOut := make([]map[string]any, 0, len(failures))
		for _, failure := range failures {
			failureOut = append(failureOut, ecsFailurePayload(failure))
		}
		respondECSJSON(w, http.StatusOK, map[string]any{
			"tasks":    taskOut,
			"failures": failureOut,
		})
		return true
	case "StartTask":
		tasks, failures, err := s.ecs.StartTask(
			ecsString(payload["cluster"]),
			ecsString(payload["taskDefinition"]),
			ecsString(payload["startedBy"]),
			ecsString(payload["group"]),
			ecsString(payload["service"]),
			ecsStringSlice(payload["containerInstances"]),
		)
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		taskOut := make([]map[string]any, 0, len(tasks))
		for _, task := range tasks {
			taskOut = append(taskOut, ecsTaskPayload(task))
		}
		failureOut := make([]map[string]any, 0, len(failures))
		for _, failure := range failures {
			failureOut = append(failureOut, ecsFailurePayload(failure))
		}
		respondECSJSON(w, http.StatusOK, map[string]any{
			"tasks":    taskOut,
			"failures": failureOut,
		})
		return true
	case "StopTask":
		task, err := s.ecs.StopTask(
			ecsString(payload["cluster"]),
			ecsString(payload["task"]),
			ecsString(payload["reason"]),
		)
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		respondECSJSON(w, http.StatusOK, map[string]any{
			"task": ecsTaskPayload(task),
		})
		return true
	case "DescribeTasks":
		tasks, failures, err := s.ecs.DescribeTasks(
			ecsString(payload["cluster"]),
			ecsStringSlice(payload["tasks"]),
		)
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		taskOut := make([]map[string]any, 0, len(tasks))
		for _, task := range tasks {
			taskOut = append(taskOut, ecsTaskPayload(task))
		}
		failureOut := make([]map[string]any, 0, len(failures))
		for _, failure := range failures {
			failureOut = append(failureOut, ecsFailurePayload(failure))
		}
		respondECSJSON(w, http.StatusOK, map[string]any{
			"tasks":    taskOut,
			"failures": failureOut,
		})
		return true
	case "ListTasks":
		maxResults, _ := ecsInt32(payload["maxResults"])
		taskARNs, nextToken, err := s.ecs.ListTasks(
			ecsString(payload["cluster"]),
			ecsString(payload["serviceName"]),
			ecsString(payload["family"]),
			ecsString(payload["desiredStatus"]),
			ecsString(payload["launchType"]),
			ecsString(payload["startedBy"]),
			ecsString(payload["containerInstance"]),
			ecsString(payload["nextToken"]),
			maxResults,
		)
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		response := map[string]any{
			"taskArns": taskARNs,
		}
		if nextToken != "" {
			response["nextToken"] = nextToken
		}
		respondECSJSON(w, http.StatusOK, response)
		return true
	case "ExecuteCommand":
		result, err := s.ecs.ExecuteCommand(
			ecsString(payload["cluster"]),
			ecsString(payload["task"]),
			ecsString(payload["container"]),
			ecsString(payload["command"]),
			ecsBoolValue(payload["interactive"]),
		)
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		respondECSJSON(w, http.StatusOK, ecsExecuteCommandPayload(result))
		return true
	case "GetTaskProtection":
		protectedTasks, failures, err := s.ecs.GetTaskProtection(
			ecsString(payload["cluster"]),
			ecsStringSlice(payload["tasks"]),
		)
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		protectedOut := make([]map[string]any, 0, len(protectedTasks))
		for _, protectedTask := range protectedTasks {
			protectedOut = append(protectedOut, ecsTaskProtectionPayload(protectedTask))
		}
		failureOut := make([]map[string]any, 0, len(failures))
		for _, failure := range failures {
			failureOut = append(failureOut, ecsFailurePayload(failure))
		}
		respondECSJSON(w, http.StatusOK, map[string]any{
			"protectedTasks": protectedOut,
			"failures":       failureOut,
		})
		return true
	case "UpdateTaskProtection":
		expiresInMinutes, hasExpires := ecsInt64(payload["expiresInMinutes"])
		protectedTasks, failures, err := s.ecs.UpdateTaskProtection(
			ecsString(payload["cluster"]),
			ecsStringSlice(payload["tasks"]),
			ecsBoolValue(payload["protectionEnabled"]),
			expiresInMinutes,
			hasExpires,
		)
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		protectedOut := make([]map[string]any, 0, len(protectedTasks))
		for _, protectedTask := range protectedTasks {
			protectedOut = append(protectedOut, ecsTaskProtectionPayload(protectedTask))
		}
		failureOut := make([]map[string]any, 0, len(failures))
		for _, failure := range failures {
			failureOut = append(failureOut, ecsFailurePayload(failure))
		}
		respondECSJSON(w, http.StatusOK, map[string]any{
			"protectedTasks": protectedOut,
			"failures":       failureOut,
		})
		return true
	case "DiscoverPollEndpoint":
		result, err := s.ecs.DiscoverPollEndpoint(
			ecsString(payload["cluster"]),
			ecsString(payload["containerInstance"]),
		)
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		respondECSJSON(w, http.StatusOK, map[string]any{
			"endpoint":          result.Endpoint,
			"telemetryEndpoint": result.TelemetryEndpoint,
		})
		return true
	case "RegisterContainerInstance":
		attributes, err := ecsAttributes(payload["attributes"])
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		containerInstance, err := s.ecs.RegisterContainerInstance(
			ecsString(payload["cluster"]),
			ecsString(payload["containerInstanceArn"]),
			ecsString(payload["ec2InstanceId"]),
			attributes,
		)
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		respondECSJSON(w, http.StatusOK, map[string]any{
			"containerInstance": ecsContainerInstancePayload(containerInstance),
		})
		return true
	case "UpdateContainerAgent":
		containerInstanceRef := ecsString(payload["containerInstance"])
		if containerInstanceRef == "" {
			containerInstanceRef = ecsString(payload["containerInstanceArn"])
		}
		containerInstance, err := s.ecs.UpdateContainerAgent(
			ecsString(payload["cluster"]),
			containerInstanceRef,
		)
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		respondECSJSON(w, http.StatusOK, map[string]any{
			"containerInstance": ecsContainerInstancePayload(containerInstance),
		})
		return true
	case "DeregisterContainerInstance":
		containerInstance, err := s.ecs.DeregisterContainerInstance(
			ecsString(payload["cluster"]),
			ecsString(payload["containerInstance"]),
		)
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		respondECSJSON(w, http.StatusOK, map[string]any{
			"containerInstance": ecsContainerInstancePayload(containerInstance),
		})
		return true
	case "DescribeContainerInstances":
		containerInstances, failures, err := s.ecs.DescribeContainerInstances(
			ecsString(payload["cluster"]),
			ecsStringSlice(payload["containerInstances"]),
		)
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		instanceOut := make([]map[string]any, 0, len(containerInstances))
		for _, containerInstance := range containerInstances {
			instanceOut = append(instanceOut, ecsContainerInstancePayload(containerInstance))
		}
		failureOut := make([]map[string]any, 0, len(failures))
		for _, failure := range failures {
			failureOut = append(failureOut, ecsFailurePayload(failure))
		}
		respondECSJSON(w, http.StatusOK, map[string]any{
			"containerInstances": instanceOut,
			"failures":           failureOut,
		})
		return true
	case "ListContainerInstances":
		maxResults, _ := ecsInt32(payload["maxResults"])
		instanceARNs, nextToken, err := s.ecs.ListContainerInstances(
			ecsString(payload["cluster"]),
			ecsString(payload["status"]),
			ecsString(payload["nextToken"]),
			maxResults,
		)
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		response := map[string]any{
			"containerInstanceArns": instanceARNs,
		}
		if nextToken != "" {
			response["nextToken"] = nextToken
		}
		respondECSJSON(w, http.StatusOK, response)
		return true
	case "UpdateContainerInstancesState":
		containerInstances, failures, err := s.ecs.UpdateContainerInstancesState(
			ecsString(payload["cluster"]),
			ecsStringSlice(payload["containerInstances"]),
			ecsString(payload["status"]),
		)
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		instanceOut := make([]map[string]any, 0, len(containerInstances))
		for _, containerInstance := range containerInstances {
			instanceOut = append(instanceOut, ecsContainerInstancePayload(containerInstance))
		}
		failureOut := make([]map[string]any, 0, len(failures))
		for _, failure := range failures {
			failureOut = append(failureOut, ecsFailurePayload(failure))
		}
		respondECSJSON(w, http.StatusOK, map[string]any{
			"containerInstances": instanceOut,
			"failures":           failureOut,
		})
		return true
	case "PutAttributes":
		attributes, err := ecsAttributes(payload["attributes"])
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		out, err := s.ecs.PutAttributes(
			ecsString(payload["cluster"]),
			attributes,
		)
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		attrOut := make([]map[string]any, 0, len(out))
		for _, attribute := range out {
			attrOut = append(attrOut, ecsAttributePayload(attribute))
		}
		respondECSJSON(w, http.StatusOK, map[string]any{"attributes": attrOut})
		return true
	case "ListAttributes":
		maxResults, _ := ecsInt32(payload["maxResults"])
		out, nextToken, err := s.ecs.ListAttributes(
			ecsString(payload["cluster"]),
			ecsString(payload["targetType"]),
			ecsString(payload["targetId"]),
			ecsString(payload["attributeName"]),
			ecsString(payload["nextToken"]),
			maxResults,
		)
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		attrOut := make([]map[string]any, 0, len(out))
		for _, attribute := range out {
			attrOut = append(attrOut, ecsAttributePayload(attribute))
		}
		response := map[string]any{"attributes": attrOut}
		if nextToken != "" {
			response["nextToken"] = nextToken
		}
		respondECSJSON(w, http.StatusOK, response)
		return true
	case "DeleteAttributes":
		attributes, err := ecsAttributes(payload["attributes"])
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		out, err := s.ecs.DeleteAttributes(
			ecsString(payload["cluster"]),
			attributes,
		)
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		attrOut := make([]map[string]any, 0, len(out))
		for _, attribute := range out {
			attrOut = append(attrOut, ecsAttributePayload(attribute))
		}
		respondECSJSON(w, http.StatusOK, map[string]any{"attributes": attrOut})
		return true
	case "TagResource":
		tags, err := ecsTagsToMap(payload["tags"])
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		if err := s.ecs.TagResource(ecsString(payload["resourceArn"]), tags); err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		respondECSJSON(w, http.StatusOK, map[string]any{})
		return true
	case "UntagResource":
		if err := s.ecs.UntagResource(
			ecsString(payload["resourceArn"]),
			ecsStringSlice(payload["tagKeys"]),
		); err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		respondECSJSON(w, http.StatusOK, map[string]any{})
		return true
	case "ListTagsForResource":
		tags, err := s.ecs.ListTagsForResource(ecsString(payload["resourceArn"]))
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		respondECSJSON(w, http.StatusOK, map[string]any{
			"tags": ecsMapToTags(tags),
		})
		return true
	case "SubmitAttachmentStateChanges":
		attachments, err := ecsAttachmentStateChanges(payload["attachments"])
		if err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		if err := s.ecs.SubmitAttachmentStateChanges(
			ecsString(payload["cluster"]),
			ecsString(payload["containerInstance"]),
			attachments,
		); err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		respondECSJSON(w, http.StatusOK, map[string]any{})
		return true
	case "SubmitContainerStateChange":
		if err := s.ecs.SubmitContainerStateChange(
			ecsString(payload["cluster"]),
			ecsString(payload["containerInstance"]),
			ecsString(payload["task"]),
			ecsString(payload["containerName"]),
			ecsString(payload["status"]),
			ecsString(payload["reason"]),
		); err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		respondECSJSON(w, http.StatusOK, map[string]any{})
		return true
	case "SubmitTaskStateChange":
		if err := s.ecs.SubmitTaskStateChange(
			ecsString(payload["cluster"]),
			ecsString(payload["task"]),
			ecsString(payload["status"]),
			ecsString(payload["reason"]),
		); err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		respondECSJSON(w, http.StatusOK, map[string]any{})
		return true
	case "SubmitTaskStateChangeByAgent":
		if err := s.ecs.SubmitTaskStateChangeByAgent(
			ecsString(payload["cluster"]),
			ecsString(payload["task"]),
			ecsString(payload["status"]),
			ecsString(payload["reason"]),
		); err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		respondECSJSON(w, http.StatusOK, map[string]any{})
		return true
	case "SubmitTaskStateChangeByManagedAgents":
		if err := s.ecs.SubmitTaskStateChangeByManagedAgents(
			ecsString(payload["cluster"]),
			ecsString(payload["task"]),
			ecsString(payload["status"]),
			ecsString(payload["reason"]),
		); err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		respondECSJSON(w, http.StatusOK, map[string]any{})
		return true
	case "StartTelemetrySession":
		if err := s.ecs.StartTelemetrySession(
			ecsString(payload["cluster"]),
			ecsString(payload["containerInstance"]),
		); err != nil {
			respondECSErrorForErr(w, err)
			return true
		}
		respondECSJSON(w, http.StatusOK, map[string]any{})
		return true
	default:
		respondECSError(w, http.StatusNotImplemented, "NotImplementedException", action+" is not implemented")
		return true
	}
}

func respondECSJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondECSError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondECSJSON(w, status, ecsError{Type: code, Message: msg})
}

func respondECSErrorForErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ecssvc.ErrInvalidParameter):
		respondECSError(w, http.StatusBadRequest, "ClientException", err.Error())
	case errors.Is(err, ecssvc.ErrAlreadyExists):
		respondECSError(w, http.StatusBadRequest, "ClientException", err.Error())
	case errors.Is(err, ecssvc.ErrClusterNotFound):
		respondECSError(w, http.StatusBadRequest, "ClusterNotFoundException", err.Error())
	case errors.Is(err, ecssvc.ErrCapacityProviderNotFound):
		respondECSError(w, http.StatusBadRequest, "CapacityProviderNotFoundException", err.Error())
	case errors.Is(err, ecssvc.ErrContainerInstanceNotFound):
		respondECSError(w, http.StatusBadRequest, "ClientException", err.Error())
	case errors.Is(err, ecssvc.ErrTaskDefinitionNotFound):
		respondECSError(w, http.StatusBadRequest, "ClientException", err.Error())
	case errors.Is(err, ecssvc.ErrResourceNotFound):
		respondECSError(w, http.StatusBadRequest, "ClientException", err.Error())
	case errors.Is(err, ecssvc.ErrServiceNotFound):
		respondECSError(w, http.StatusBadRequest, "ServiceNotFoundException", err.Error())
	case errors.Is(err, ecssvc.ErrServiceDeploymentNotFound):
		respondECSError(w, http.StatusBadRequest, "ClientException", err.Error())
	case errors.Is(err, ecssvc.ErrServiceRevisionNotFound):
		respondECSError(w, http.StatusBadRequest, "ClientException", err.Error())
	case errors.Is(err, ecssvc.ErrTaskSetNotFound):
		respondECSError(w, http.StatusBadRequest, "ClientException", err.Error())
	case errors.Is(err, ecssvc.ErrTaskNotFound):
		respondECSError(w, http.StatusBadRequest, "ClientException", err.Error())
	default:
		respondECSError(w, http.StatusInternalServerError, "ServerException", err.Error())
	}
}

func isECSJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.HasPrefix(target, "AmazonEC2ContainerService_V20141113.") {
		return true
	}
	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.1") || strings.Contains(contentType, "application/x-amz-json-1.0") {
		return strings.HasPrefix(target, "AmazonEC2ContainerService")
	}
	return false
}

func parseECSTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "AmazonEC2ContainerService_V20141113.") {
		return strings.TrimPrefix(target, "AmazonEC2ContainerService_V20141113.")
	}
	if strings.HasPrefix(target, "AmazonEC2ContainerService.") {
		return strings.TrimPrefix(target, "AmazonEC2ContainerService.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parseECSPayload(r *http.Request) (map[string]any, error) {
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

func ecsAccountSettingPayload(setting ecssvc.AccountSetting) map[string]any {
	out := map[string]any{
		"name":  setting.Name,
		"value": setting.Value,
		"type":  setting.Type,
	}
	if setting.PrincipalArn != "" {
		out["principalArn"] = setting.PrincipalArn
	}
	return out
}

func ecsClusterPayload(cluster ecssvc.Cluster) map[string]any {
	settings := make([]map[string]any, 0, len(cluster.Settings))
	for _, setting := range cluster.Settings {
		settings = append(settings, map[string]any{
			"name":  setting.Name,
			"value": setting.Value,
		})
	}
	statistics := make([]map[string]any, 0, len(cluster.Statistics))
	statKeys := make([]string, 0, len(cluster.Statistics))
	for key := range cluster.Statistics {
		statKeys = append(statKeys, key)
	}
	sort.Strings(statKeys)
	for _, key := range statKeys {
		statistics = append(statistics, map[string]any{
			"name":  key,
			"value": cluster.Statistics[key],
		})
	}
	tags := make([]map[string]any, 0, len(cluster.Tags))
	tagKeys := make([]string, 0, len(cluster.Tags))
	for key := range cluster.Tags {
		tagKeys = append(tagKeys, key)
	}
	sort.Strings(tagKeys)
	for _, key := range tagKeys {
		tags = append(tags, map[string]any{
			"key":   key,
			"value": cluster.Tags[key],
		})
	}
	out := map[string]any{
		"clusterArn":                        cluster.ClusterArn,
		"clusterName":                       cluster.ClusterName,
		"status":                            cluster.Status,
		"settings":                          settings,
		"statistics":                        statistics,
		"tags":                              tags,
		"registeredContainerInstancesCount": cluster.RegisteredContainerInstancesCount,
		"runningTasksCount":                 cluster.RunningTasksCount,
		"pendingTasksCount":                 cluster.PendingTasksCount,
		"activeServicesCount":               cluster.ActiveServicesCount,
		"capacityProviders":                 cluster.CapacityProviders,
		"defaultCapacityProviderStrategy":   ecsCapacityProviderStrategyPayload(cluster.DefaultCapacityProviderStrategy),
		"createdAt":                         ecsTimestamp(cluster.CreatedAt),
		"updatedAt":                         ecsTimestamp(cluster.UpdatedAt),
	}
	if cluster.Configuration != nil {
		out["configuration"] = cloneMapAny(cluster.Configuration)
	}
	if strings.TrimSpace(cluster.ServiceConnectDefaultsNamespace) != "" {
		out["serviceConnectDefaults"] = map[string]any{
			"namespace": cluster.ServiceConnectDefaultsNamespace,
		}
	}
	return out
}

func ecsCapacityProviderPayload(cp ecssvc.CapacityProvider) map[string]any {
	out := map[string]any{
		"capacityProviderArn": cp.Arn,
		"name":                cp.Name,
		"status":              cp.Status,
		"updateStatus":        cp.UpdateStatus,
		"createdAt":           ecsTimestamp(cp.CreatedAt),
		"updatedAt":           ecsTimestamp(cp.UpdatedAt),
	}
	if cp.AutoScalingGroupArn != "" || cp.ManagedScalingStatus != "" || cp.ManagedTerminationProtection != "" {
		autoScalingGroupProvider := map[string]any{
			"autoScalingGroupArn": cp.AutoScalingGroupArn,
		}
		if cp.ManagedScalingStatus != "" {
			autoScalingGroupProvider["managedScaling"] = map[string]any{
				"status": cp.ManagedScalingStatus,
			}
		}
		if cp.ManagedTerminationProtection != "" {
			autoScalingGroupProvider["managedTerminationProtection"] = cp.ManagedTerminationProtection
		}
		out["autoScalingGroupProvider"] = autoScalingGroupProvider
	}
	if len(cp.Tags) > 0 {
		out["tags"] = ecsMapToTags(cp.Tags)
	}
	return out
}

func ecsTaskDefinitionPayload(def ecssvc.TaskDefinition) map[string]any {
	out := map[string]any{
		"taskDefinitionArn":       def.Arn,
		"family":                  def.Family,
		"revision":                def.Revision,
		"status":                  def.Status,
		"networkMode":             def.NetworkMode,
		"cpu":                     def.Cpu,
		"memory":                  def.Memory,
		"executionRoleArn":        def.ExecutionRoleArn,
		"taskRoleArn":             def.TaskRoleArn,
		"requiresCompatibilities": def.RequiresCompatibilities,
		"containerDefinitions":    ecsContainerDefinitionsPayload(def.ContainerDefinitions),
		"registeredAt":            ecsTimestamp(def.RegisteredAt),
	}
	if len(def.RequiresCompatibilities) == 0 {
		out["requiresCompatibilities"] = []string{}
	}
	if len(def.ContainerDefinitions) == 0 {
		out["containerDefinitions"] = []map[string]any{}
	}
	if def.DeregisteredAt != nil {
		out["deregisteredAt"] = ecsTimestamp(*def.DeregisteredAt)
	}
	return out
}

func ecsServicePayload(service ecssvc.ServiceDefinition) map[string]any {
	out := map[string]any{
		"serviceArn":               service.Arn,
		"serviceName":              service.Name,
		"clusterArn":               service.ClusterArn,
		"taskDefinition":           service.TaskDefinitionArn,
		"taskSets":                 []map[string]any{},
		"serviceConnectResources":  []map[string]any{},
		"desiredCount":             service.DesiredCount,
		"runningCount":             service.DesiredCount,
		"pendingCount":             int32(0),
		"status":                   service.Status,
		"launchType":               service.LaunchType,
		"createdAt":                ecsTimestamp(service.CreatedAt),
		"updatedAt":                ecsTimestamp(service.UpdatedAt),
		"deployments":              []map[string]any{},
		"events":                   []map[string]any{},
		"capacityProviderStrategy": []map[string]any{},
	}
	if len(service.Tags) > 0 {
		out["tags"] = ecsMapToTags(service.Tags)
	}
	if service.PrimaryTaskSetArn != "" {
		out["primaryTaskSet"] = service.PrimaryTaskSetArn
	}
	return out
}

func ecsTaskSetPayload(taskSet ecssvc.TaskSet) map[string]any {
	return map[string]any{
		"id":                   taskSet.ID,
		"taskSetArn":           taskSet.Arn,
		"serviceArn":           taskSet.ServiceArn,
		"clusterArn":           taskSet.ClusterArn,
		"taskDefinition":       taskSet.TaskDefinitionArn,
		"computedDesiredCount": taskSet.ComputedDesired,
		"pendingCount":         taskSet.PendingCount,
		"runningCount":         taskSet.RunningCount,
		"status":               taskSet.Status,
		"launchType":           taskSet.LaunchType,
		"scale": map[string]any{
			"value": taskSet.ScaleValue,
			"unit":  taskSet.ScaleUnit,
		},
		"createdAt": ecsTimestamp(taskSet.CreatedAt),
		"updatedAt": ecsTimestamp(taskSet.UpdatedAt),
	}
}

func ecsTaskPayload(task ecssvc.Task) map[string]any {
	out := map[string]any{
		"taskArn":           task.Arn,
		"clusterArn":        task.ClusterArn,
		"taskDefinitionArn": task.TaskDefinitionArn,
		"group":             task.Group,
		"startedBy":         task.StartedBy,
		"lastStatus":        task.LastStatus,
		"desiredStatus":     task.DesiredStatus,
		"launchType":        task.LaunchType,
		"createdAt":         ecsTimestamp(task.CreatedAt),
		"startedAt":         ecsTimestamp(task.StartedAt),
		"attachments":       []map[string]any{},
		"containers":        []map[string]any{},
	}
	if task.ServiceArn != "" {
		if idx := strings.LastIndex(task.ServiceArn, "/"); idx >= 0 && idx+1 < len(task.ServiceArn) {
			out["group"] = "service:" + task.ServiceArn[idx+1:]
		}
	}
	if task.ContainerInstanceArn != "" {
		out["containerInstanceArn"] = task.ContainerInstanceArn
	}
	if task.StoppedAt != nil {
		out["stoppedAt"] = ecsTimestamp(*task.StoppedAt)
	}
	if task.StoppedReason != "" {
		out["stoppedReason"] = task.StoppedReason
	}
	return out
}

func ecsContainerInstancePayload(containerInstance ecssvc.ContainerInstance) map[string]any {
	registeredResources := []map[string]any{}
	remainingResources := []map[string]any{}
	attributes := make([]map[string]any, 0, len(containerInstance.Attributes))
	for _, attribute := range containerInstance.Attributes {
		attributes = append(attributes, ecsAttributePayload(attribute))
	}
	out := map[string]any{
		"containerInstanceArn": containerInstance.Arn,
		"ec2InstanceId":        containerInstance.Ec2InstanceID,
		"status":               containerInstance.Status,
		"agentConnected":       containerInstance.AgentConnected,
		"registeredAt":         ecsTimestamp(containerInstance.RegisteredAt),
		"updatedAt":            ecsTimestamp(containerInstance.UpdatedAt),
		"runningTasksCount":    int32(0),
		"pendingTasksCount":    int32(0),
		"registeredResources":  registeredResources,
		"remainingResources":   remainingResources,
		"attributes":           attributes,
	}
	if strings.TrimSpace(containerInstance.AgentUpdateStatus) != "" {
		out["agentUpdateStatus"] = containerInstance.AgentUpdateStatus
	}
	if containerInstance.Version > 0 {
		out["version"] = containerInstance.Version
	}
	if strings.TrimSpace(containerInstance.VersionInfo.AgentHash) != "" ||
		strings.TrimSpace(containerInstance.VersionInfo.AgentVersion) != "" ||
		strings.TrimSpace(containerInstance.VersionInfo.DockerVersion) != "" {
		out["versionInfo"] = map[string]any{
			"agentHash":     containerInstance.VersionInfo.AgentHash,
			"agentVersion":  containerInstance.VersionInfo.AgentVersion,
			"dockerVersion": containerInstance.VersionInfo.DockerVersion,
		}
	}
	return out
}

func ecsExpressGatewayServicePayload(service ecssvc.ExpressGatewayService) map[string]any {
	activeConfigurations := make([]map[string]any, 0, len(service.ActiveConfigurations))
	for _, cfg := range service.ActiveConfigurations {
		activeConfigurations = append(activeConfigurations, ecsExpressGatewayServiceConfigurationPayload(cfg))
	}
	return map[string]any{
		"activeConfigurations":  activeConfigurations,
		"cluster":               service.Cluster,
		"createdAt":             ecsTimestamp(service.CreatedAt),
		"currentDeployment":     service.CurrentDeployment,
		"infrastructureRoleArn": service.InfrastructureRoleArn,
		"serviceArn":            service.ServiceArn,
		"serviceName":           service.ServiceName,
		"status": map[string]any{
			"statusCode":   service.StatusCode,
			"statusReason": service.StatusReason,
		},
		"tags":      ecsMapToTags(service.Tags),
		"updatedAt": ecsTimestamp(service.UpdatedAt),
	}
}

func ecsExpressGatewayServiceConfigurationPayload(cfg ecssvc.ExpressGatewayServiceConfiguration) map[string]any {
	out := map[string]any{
		"cpu":                cfg.Cpu,
		"createdAt":          ecsTimestamp(cfg.CreatedAt),
		"executionRoleArn":   cfg.ExecutionRoleArn,
		"healthCheckPath":    cfg.HealthCheckPath,
		"ingressPaths":       []map[string]any{},
		"memory":             cfg.Memory,
		"serviceRevisionArn": cfg.ServiceRevisionArn,
		"taskRoleArn":        cfg.TaskRoleArn,
	}
	if cfg.PrimaryContainer != nil {
		out["primaryContainer"] = cloneMapAny(cfg.PrimaryContainer)
	}
	if cfg.NetworkConfiguration != nil {
		out["networkConfiguration"] = cloneMapAny(cfg.NetworkConfiguration)
	}
	if cfg.ScalingTarget != nil {
		out["scalingTarget"] = cloneMapAny(cfg.ScalingTarget)
	}
	return out
}

func ecsUpdatedExpressGatewayServicePayload(service ecssvc.UpdatedExpressGatewayService) map[string]any {
	return map[string]any{
		"cluster":     service.Cluster,
		"createdAt":   ecsTimestamp(service.CreatedAt),
		"serviceArn":  service.ServiceArn,
		"serviceName": service.ServiceName,
		"status": map[string]any{
			"statusCode":   service.StatusCode,
			"statusReason": service.StatusReason,
		},
		"targetConfiguration": ecsExpressGatewayServiceConfigurationPayload(service.TargetConfiguration),
		"updatedAt":           ecsTimestamp(service.UpdatedAt),
	}
}

func ecsAttributePayload(attribute ecssvc.Attribute) map[string]any {
	return map[string]any{
		"name":       attribute.Name,
		"value":      attribute.Value,
		"targetType": attribute.TargetType,
		"targetId":   attribute.TargetID,
	}
}

func ecsTaskProtectionPayload(taskProtection ecssvc.TaskProtection) map[string]any {
	out := map[string]any{
		"taskArn":           taskProtection.TaskArn,
		"protectionEnabled": taskProtection.ProtectionEnabled,
	}
	if taskProtection.ExpirationDate != nil {
		out["expirationDate"] = ecsTimestamp(*taskProtection.ExpirationDate)
	}
	return out
}

func ecsExecuteCommandPayload(result ecssvc.ExecuteCommandResult) map[string]any {
	return map[string]any{
		"clusterArn":    result.ClusterArn,
		"taskArn":       result.TaskArn,
		"containerName": result.ContainerName,
		"interactive":   result.Interactive,
		"session": map[string]any{
			"sessionId":  result.SessionID,
			"streamUrl":  result.StreamURL,
			"tokenValue": result.TokenValue,
		},
	}
}

func ecsServiceDeploymentPayload(deployment ecssvc.ServiceDeploymentSnapshot) map[string]any {
	out := map[string]any{
		"serviceDeploymentArn": deployment.Arn,
		"serviceArn":           deployment.ServiceArn,
		"serviceRevisionArn":   deployment.ServiceRevisionArn,
		"status":               deployment.Status,
		"statusReason":         deployment.StatusReason,
		"createdAt":            ecsTimestamp(deployment.CreatedAt),
		"updatedAt":            ecsTimestamp(deployment.UpdatedAt),
	}
	if deployment.StoppedAt != nil {
		out["stoppedAt"] = ecsTimestamp(*deployment.StoppedAt)
	}
	return out
}

func ecsServiceRevisionPayload(revision ecssvc.ServiceRevisionSnapshot) map[string]any {
	return map[string]any{
		"serviceRevisionArn": revision.Arn,
		"serviceArn":         revision.ServiceArn,
		"taskDefinition":     revision.TaskDefinitionArn,
		"desiredCount":       revision.DesiredCount,
		"createdAt":          ecsTimestamp(revision.CreatedAt),
	}
}

func ecsFailurePayload(failure ecssvc.Failure) map[string]any {
	out := map[string]any{
		"arn":    failure.Arn,
		"reason": failure.Reason,
	}
	if failure.Detail != "" {
		out["detail"] = failure.Detail
	}
	return out
}

func ecsClusterSettings(v any) ([]ecssvc.ClusterSetting, error) {
	items, ok := v.([]any)
	if !ok {
		if v == nil {
			return nil, nil
		}
		return nil, ecssvc.ErrInvalidParameter
	}
	out := make([]ecssvc.ClusterSetting, 0, len(items))
	for _, item := range items {
		entry, ok := item.(map[string]any)
		if !ok {
			return nil, ecssvc.ErrInvalidParameter
		}
		out = append(out, ecssvc.ClusterSetting{
			Name:  ecsString(entry["name"]),
			Value: ecsString(entry["value"]),
		})
	}
	return out, nil
}

func ecsCapacityProviderStrategy(v any) ([]ecssvc.CapacityProviderStrategyItem, error) {
	items, ok := v.([]any)
	if !ok {
		if v == nil {
			return nil, nil
		}
		return nil, ecssvc.ErrInvalidParameter
	}
	out := make([]ecssvc.CapacityProviderStrategyItem, 0, len(items))
	for _, item := range items {
		entry, ok := item.(map[string]any)
		if !ok {
			return nil, ecssvc.ErrInvalidParameter
		}
		base, _ := ecsInt32(entry["base"])
		weight, _ := ecsInt32(entry["weight"])
		out = append(out, ecssvc.CapacityProviderStrategyItem{
			CapacityProvider: ecsString(entry["capacityProvider"]),
			Base:             base,
			Weight:           weight,
		})
	}
	return out, nil
}

func ecsCapacityProviderStrategyPayload(items []ecssvc.CapacityProviderStrategyItem) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, map[string]any{
			"capacityProvider": item.CapacityProvider,
			"base":             item.Base,
			"weight":           item.Weight,
		})
	}
	return out
}

func ecsContainerDefinitions(v any) ([]ecssvc.ContainerDefinition, error) {
	items, ok := v.([]any)
	if !ok {
		if v == nil {
			return nil, nil
		}
		return nil, ecssvc.ErrInvalidParameter
	}
	out := make([]ecssvc.ContainerDefinition, 0, len(items))
	for _, item := range items {
		entry, ok := item.(map[string]any)
		if !ok {
			return nil, ecssvc.ErrInvalidParameter
		}
		out = append(out, ecssvc.ContainerDefinition{
			Name:  ecsString(entry["name"]),
			Image: ecsString(entry["image"]),
		})
	}
	return out, nil
}

func ecsContainerDefinitionsPayload(defs []ecssvc.ContainerDefinition) []map[string]any {
	out := make([]map[string]any, 0, len(defs))
	for _, def := range defs {
		out = append(out, map[string]any{
			"name":  def.Name,
			"image": def.Image,
		})
	}
	return out
}

func ecsTagsToMap(v any) (map[string]string, error) {
	items, ok := v.([]any)
	if !ok {
		if v == nil {
			return map[string]string{}, nil
		}
		return nil, ecssvc.ErrInvalidParameter
	}
	out := make(map[string]string, len(items))
	for _, item := range items {
		entry, ok := item.(map[string]any)
		if !ok {
			return nil, ecssvc.ErrInvalidParameter
		}
		key := ecsString(entry["key"])
		if key == "" {
			key = ecsString(entry["Key"])
		}
		if key == "" {
			return nil, ecssvc.ErrInvalidParameter
		}
		value := ecsString(entry["value"])
		if value == "" {
			value = ecsString(entry["Value"])
		}
		out[key] = value
	}
	return out, nil
}

func ecsAttributes(v any) ([]ecssvc.Attribute, error) {
	items, ok := v.([]any)
	if !ok {
		if v == nil {
			return nil, nil
		}
		return nil, ecssvc.ErrInvalidParameter
	}
	out := make([]ecssvc.Attribute, 0, len(items))
	for _, item := range items {
		entry, ok := item.(map[string]any)
		if !ok {
			return nil, ecssvc.ErrInvalidParameter
		}
		name := ecsString(entry["name"])
		if name == "" {
			name = ecsString(entry["Name"])
		}
		value := ecsString(entry["value"])
		if value == "" {
			value = ecsString(entry["Value"])
		}
		targetType := ecsString(entry["targetType"])
		if targetType == "" {
			targetType = ecsString(entry["TargetType"])
		}
		targetID := ecsString(entry["targetId"])
		if targetID == "" {
			targetID = ecsString(entry["targetID"])
		}
		out = append(out, ecssvc.Attribute{
			Name:       name,
			Value:      value,
			TargetType: targetType,
			TargetID:   targetID,
		})
	}
	return out, nil
}

func ecsAttachmentStateChanges(v any) ([]ecssvc.AttachmentStateChange, error) {
	items, ok := v.([]any)
	if !ok {
		if v == nil {
			return nil, nil
		}
		return nil, ecssvc.ErrInvalidParameter
	}
	out := make([]ecssvc.AttachmentStateChange, 0, len(items))
	for _, item := range items {
		entry, ok := item.(map[string]any)
		if !ok {
			return nil, ecssvc.ErrInvalidParameter
		}
		attachmentARN := ecsString(entry["attachmentArn"])
		if attachmentARN == "" {
			attachmentARN = ecsString(entry["attachmentARN"])
		}
		status := ecsString(entry["status"])
		if attachmentARN == "" || status == "" {
			return nil, ecssvc.ErrInvalidParameter
		}
		out = append(out, ecssvc.AttachmentStateChange{
			AttachmentArn: attachmentARN,
			Status:        status,
		})
	}
	return out, nil
}

func ecsOptionalMap(payload map[string]any, key string) (map[string]any, bool, error) {
	raw, ok := payload[key]
	if !ok {
		return nil, false, nil
	}
	if raw == nil {
		return map[string]any{}, true, nil
	}
	obj, isMap := raw.(map[string]any)
	if !isMap {
		return nil, false, ecssvc.ErrInvalidParameter
	}
	return obj, true, nil
}

func ecsIncludeTags(v any) bool {
	for _, include := range ecsStringSlice(v) {
		if strings.EqualFold(strings.TrimSpace(include), "TAGS") {
			return true
		}
	}
	return false
}

func ecsMapToTags(tags map[string]string) []map[string]any {
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
		out = append(out, map[string]any{"key": key, "value": tags[key]})
	}
	return out
}

func cloneMapAny(in map[string]any) map[string]any {
	if len(in) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = cloneValueAny(value)
	}
	return out
}

func cloneValueAny(v any) any {
	switch typed := v.(type) {
	case map[string]any:
		return cloneMapAny(typed)
	case []any:
		out := make([]any, len(typed))
		for i, value := range typed {
			out[i] = cloneValueAny(value)
		}
		return out
	default:
		return v
	}
}

func ecsMap(v any) map[string]any {
	obj, ok := v.(map[string]any)
	if !ok {
		return map[string]any{}
	}
	return obj
}

func ecsString(v any) string {
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(s)
}

func ecsStringSlice(v any) []string {
	items, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
			out = append(out, strings.TrimSpace(s))
		}
	}
	return out
}

func ecsBool(v any) (bool, bool) {
	b, ok := v.(bool)
	return b, ok
}

func ecsBoolValue(v any) bool {
	b, ok := ecsBool(v)
	if !ok {
		return false
	}
	return b
}

func ecsInt32(v any) (int32, bool) {
	switch n := v.(type) {
	case float64:
		return int32(n), true
	case int:
		return int32(n), true
	case int32:
		return n, true
	case int64:
		return int32(n), true
	case string:
		if strings.TrimSpace(n) == "" {
			return 0, false
		}
		parsed, err := strconv.ParseInt(strings.TrimSpace(n), 10, 32)
		if err != nil {
			return 0, false
		}
		return int32(parsed), true
	default:
		return 0, false
	}
}

func ecsInt64(v any) (int64, bool) {
	switch n := v.(type) {
	case float64:
		return int64(n), true
	case int:
		return int64(n), true
	case int32:
		return int64(n), true
	case int64:
		return n, true
	case string:
		if strings.TrimSpace(n) == "" {
			return 0, false
		}
		parsed, err := strconv.ParseInt(strings.TrimSpace(n), 10, 64)
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func ecsFloat64(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case string:
		if strings.TrimSpace(n) == "" {
			return 0, false
		}
		parsed, err := strconv.ParseFloat(strings.TrimSpace(n), 64)
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func ecsTimestamp(t time.Time) float64 {
	if t.IsZero() {
		return 0
	}
	return float64(t.UTC().UnixNano()) / float64(time.Second)
}
