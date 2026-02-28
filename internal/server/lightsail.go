package server

import (
	"bytes"
	"encoding/json"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/stackyard/stackyard/internal/services/lightsail"
)

type lightsailError struct {
	Code    string `json:"code,omitempty"`
	Docs    string `json:"docs,omitempty"`
	Message string `json:"message"`
	Tip     string `json:"tip,omitempty"`
}

func (s *Server) handleLightsailJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isLightsailJSONCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "lightsail")
	if !ok {
		respondLightsailError(w, status, code, msg)
		return true
	}

	action := parseLightsailTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "missing X-Amz-Target")
		return true
	}
	if _, known := lightsailOperationByName[action]; !known {
		respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "unknown action")
		return true
	}

	payload, err := parseLightsailPayload(r)
	if err != nil {
		respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "invalid JSON body")
		return true
	}

	switch action {
	case "CreateInstances":
		ops, err := s.lightsail.CreateInstances(
			lightsailString(payload["availabilityZone"]),
			lightsailString(payload["blueprintId"]),
			lightsailString(payload["bundleId"]),
			lightsailStringSlice(payload["instanceNames"]),
			lightsailTagsToMap(payload["tags"]),
		)
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operations": lightsailOperationsPayload(ops)})
		return true
	case "GetBlueprints":
		includeInactive, _ := lightsailBool(payload["includeInactive"])
		page, err := s.lightsail.GetBlueprints(
			includeInactive,
			lightsailString(payload["appCategory"]),
			lightsailString(payload["pageToken"]),
		)
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		blueprints := make([]map[string]any, 0, len(page.Blueprints))
		for _, blueprint := range page.Blueprints {
			blueprints = append(blueprints, lightsailBlueprintPayload(blueprint))
		}
		response := map[string]any{"blueprints": blueprints}
		if page.NextPageToken != "" {
			response["nextPageToken"] = page.NextPageToken
		}
		respondLightsailJSON(w, http.StatusOK, response)
		return true
	case "GetBundles":
		includeInactive, _ := lightsailBool(payload["includeInactive"])
		page, err := s.lightsail.GetBundles(
			includeInactive,
			lightsailString(payload["appCategory"]),
			lightsailString(payload["pageToken"]),
		)
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		bundles := make([]map[string]any, 0, len(page.Bundles))
		for _, bundle := range page.Bundles {
			bundles = append(bundles, lightsailBundlePayload(bundle))
		}
		response := map[string]any{"bundles": bundles}
		if page.NextPageToken != "" {
			response["nextPageToken"] = page.NextPageToken
		}
		respondLightsailJSON(w, http.StatusOK, response)
		return true
	case "GetActiveNames":
		activeNames, nextPageToken, err := s.lightsail.GetActiveNames(lightsailString(payload["pageToken"]))
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		response := map[string]any{"activeNames": activeNames}
		if nextPageToken != "" {
			response["nextPageToken"] = nextPageToken
		}
		respondLightsailJSON(w, http.StatusOK, response)
		return true
	case "GetSetupHistory":
		page, err := s.lightsail.GetSetupHistory(
			lightsailString(payload["resourceName"]),
			lightsailString(payload["pageToken"]),
		)
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		response := map[string]any{
			"setupHistory": lightsailSetupHistoryPayload(page.SetupHistory),
		}
		if page.NextPageToken != "" {
			response["nextPageToken"] = page.NextPageToken
		}
		respondLightsailJSON(w, http.StatusOK, response)
		return true
	case "GetCostEstimate":
		startTime, ok := lightsailTime(payload["startTime"])
		if !ok {
			respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "startTime is required")
			return true
		}
		endTime, ok := lightsailTime(payload["endTime"])
		if !ok {
			respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "endTime is required")
			return true
		}
		estimates, err := s.lightsail.GetCostEstimate(
			lightsailString(payload["resourceName"]),
			startTime,
			endTime,
		)
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{
			"resourcesBudgetEstimate": lightsailResourceBudgetEstimatesPayload(estimates),
		})
		return true
	case "IsVpcPeered":
		respondLightsailJSON(w, http.StatusOK, map[string]any{
			"isPeered": s.lightsail.IsVpcPeered(),
		})
		return true
	case "PeerVpc":
		op, err := s.lightsail.PeerVpc()
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operation": lightsailOperationPayload(op)})
		return true
	case "UnpeerVpc":
		op, err := s.lightsail.UnpeerVpc()
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operation": lightsailOperationPayload(op)})
		return true
	case "CreateCloudFormationStack":
		entries, ok := lightsailInstanceEntries(payload["instances"])
		if !ok || len(entries) == 0 {
			respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "instances is required")
			return true
		}
		ops, err := s.lightsail.CreateCloudFormationStack(entries)
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operations": lightsailOperationsPayload(ops)})
		return true
	case "GetCloudFormationStackRecords":
		page, err := s.lightsail.GetCloudFormationStackRecords(lightsailString(payload["pageToken"]))
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		response := map[string]any{
			"cloudFormationStackRecords": lightsailCloudFormationStackRecordsPayload(page.CloudFormationStackRecords),
		}
		if page.NextPageToken != "" {
			response["nextPageToken"] = page.NextPageToken
		}
		respondLightsailJSON(w, http.StatusOK, response)
		return true
	case "CreateGUISessionAccessDetails":
		details, err := s.lightsail.CreateGUISessionAccessDetails(lightsailString(payload["resourceName"]))
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{
			"failureReason":      details.FailureReason,
			"percentageComplete": details.PercentageComplete,
			"resourceName":       details.ResourceName,
			"sessions":           lightsailSessionsPayload(details.Sessions),
			"status":             details.Status,
		})
		return true
	case "StartGUISession":
		ops, err := s.lightsail.StartGUISession(lightsailString(payload["resourceName"]))
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operations": lightsailOperationsPayload(ops)})
		return true
	case "StopGUISession":
		ops, err := s.lightsail.StopGUISession(lightsailString(payload["resourceName"]))
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operations": lightsailOperationsPayload(ops)})
		return true
	case "GetInstanceMetricData":
		startTime, ok := lightsailTime(payload["startTime"])
		if !ok {
			respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "startTime is required")
			return true
		}
		endTime, ok := lightsailTime(payload["endTime"])
		if !ok {
			respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "endTime is required")
			return true
		}
		period, ok := lightsailInt32(payload["period"])
		if !ok {
			respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "period is required")
			return true
		}
		statistics := lightsailStringSlice(payload["statistics"])
		if len(statistics) == 0 {
			respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "statistics is required")
			return true
		}
		metricName, metricData, err := s.lightsail.GetInstanceMetricData(lightsail.InstanceMetricInput{
			InstanceName: lightsailString(payload["instanceName"]),
			EndTime:      endTime,
			MetricName:   lightsailString(payload["metricName"]),
			Period:       period,
			StartTime:    startTime,
			Statistics:   statistics,
			Unit:         lightsailString(payload["unit"]),
		})
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{
			"metricData": lightsailInstanceMetricDatapointsPayload(metricData),
			"metricName": metricName,
		})
		return true
	case "GetInstance":
		instance, found := s.lightsail.GetInstance(lightsailString(payload["instanceName"]))
		if !found {
			respondLightsailError(w, http.StatusNotFound, "NotFoundException", "instance not found")
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"instance": lightsailInstancePayload(instance)})
		return true
	case "GetInstances":
		instances := s.lightsail.GetInstances()
		out := make([]map[string]any, 0, len(instances))
		for _, instance := range instances {
			out = append(out, lightsailInstancePayload(instance))
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"instances": out})
		return true
	case "GetInstanceState":
		code, name, found := s.lightsail.GetInstanceState(lightsailString(payload["instanceName"]))
		if !found {
			respondLightsailError(w, http.StatusNotFound, "NotFoundException", "instance not found")
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{
			"state": map[string]any{
				"code": code,
				"name": name,
			},
		})
		return true
	case "DeleteInstance":
		ops, err := s.lightsail.DeleteInstance(lightsailString(payload["instanceName"]))
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operations": lightsailOperationsPayload(ops)})
		return true
	case "StartInstance":
		ops, err := s.lightsail.StartInstance(lightsailString(payload["instanceName"]))
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operations": lightsailOperationsPayload(ops)})
		return true
	case "StopInstance":
		ops, err := s.lightsail.StopInstance(lightsailString(payload["instanceName"]))
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operations": lightsailOperationsPayload(ops)})
		return true
	case "RebootInstance":
		ops, err := s.lightsail.RebootInstance(lightsailString(payload["instanceName"]))
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operations": lightsailOperationsPayload(ops)})
		return true
	case "OpenInstancePublicPorts":
		portInfo, ok := lightsailPortInfo(payload["portInfo"])
		if !ok {
			respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "portInfo is required")
			return true
		}
		op, err := s.lightsail.OpenInstancePublicPorts(lightsailString(payload["instanceName"]), portInfo)
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operation": lightsailOperationPayload(op)})
		return true
	case "CloseInstancePublicPorts":
		portInfo, ok := lightsailPortInfo(payload["portInfo"])
		if !ok {
			respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "portInfo is required")
			return true
		}
		op, err := s.lightsail.CloseInstancePublicPorts(lightsailString(payload["instanceName"]), portInfo)
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operation": lightsailOperationPayload(op)})
		return true
	case "PutInstancePublicPorts":
		portInfos := lightsailPortInfos(payload["portInfos"])
		if len(portInfos) == 0 {
			respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "portInfos is required")
			return true
		}
		op, err := s.lightsail.PutInstancePublicPorts(lightsailString(payload["instanceName"]), portInfos)
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operation": lightsailOperationPayload(op)})
		return true
	case "GetInstancePortStates":
		portStates, err := s.lightsail.GetInstancePortStates(lightsailString(payload["instanceName"]))
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"portStates": lightsailPortStatesPayload(portStates)})
		return true
	case "GetInstanceAccessDetails":
		accessDetails, err := s.lightsail.GetInstanceAccessDetails(
			lightsailString(payload["instanceName"]),
			lightsailString(payload["protocol"]),
		)
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"accessDetails": lightsailInstanceAccessDetailsPayload(accessDetails)})
		return true
	case "UpdateInstanceMetadataOptions":
		httpPutResponseHopLimit, err := lightsailOptionalInt32(payload["httpPutResponseHopLimit"])
		if err != nil {
			respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "httpPutResponseHopLimit must be an integer")
			return true
		}
		op, err := s.lightsail.UpdateInstanceMetadataOptions(
			lightsailString(payload["instanceName"]),
			lightsailString(payload["httpEndpoint"]),
			lightsailString(payload["httpProtocolIpv6"]),
			lightsailString(payload["httpTokens"]),
			httpPutResponseHopLimit,
		)
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operation": lightsailOperationPayload(op)})
		return true
	case "CreateInstancesFromSnapshot":
		ops, err := s.lightsail.CreateInstancesFromSnapshot(
			lightsailString(payload["availabilityZone"]),
			lightsailString(payload["bundleId"]),
			lightsailStringSlice(payload["instanceNames"]),
			lightsailString(payload["instanceSnapshotName"]),
			lightsailString(payload["sourceInstanceName"]),
			lightsailTagsToMap(payload["tags"]),
		)
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operations": lightsailOperationsPayload(ops)})
		return true
	case "CreateDisk":
		sizeInGb, ok := lightsailInt32(payload["sizeInGb"])
		if !ok {
			respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "sizeInGb is required")
			return true
		}
		ops, err := s.lightsail.CreateDisk(
			lightsailString(payload["availabilityZone"]),
			lightsailString(payload["diskName"]),
			sizeInGb,
			lightsailTagsToMap(payload["tags"]),
		)
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operations": lightsailOperationsPayload(ops)})
		return true
	case "GetDisk":
		disk, found := s.lightsail.GetDisk(lightsailString(payload["diskName"]))
		if !found {
			respondLightsailError(w, http.StatusNotFound, "NotFoundException", "disk not found")
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"disk": lightsailDiskPayload(disk)})
		return true
	case "GetDisks":
		disks := s.lightsail.GetDisks()
		out := make([]map[string]any, 0, len(disks))
		for _, disk := range disks {
			out = append(out, lightsailDiskPayload(disk))
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"disks": out})
		return true
	case "DeleteDisk":
		ops, err := s.lightsail.DeleteDisk(lightsailString(payload["diskName"]))
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operations": lightsailOperationsPayload(ops)})
		return true
	case "AttachDisk":
		autoMounting, _ := lightsailBool(payload["autoMounting"])
		ops, err := s.lightsail.AttachDisk(
			lightsailString(payload["diskName"]),
			lightsailString(payload["diskPath"]),
			lightsailString(payload["instanceName"]),
			autoMounting,
		)
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operations": lightsailOperationsPayload(ops)})
		return true
	case "DetachDisk":
		ops, err := s.lightsail.DetachDisk(lightsailString(payload["diskName"]))
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operations": lightsailOperationsPayload(ops)})
		return true
	case "CreateDiskSnapshot":
		ops, err := s.lightsail.CreateDiskSnapshot(
			lightsailString(payload["diskName"]),
			lightsailString(payload["instanceName"]),
			lightsailString(payload["diskSnapshotName"]),
			lightsailTagsToMap(payload["tags"]),
		)
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operations": lightsailOperationsPayload(ops)})
		return true
	case "GetDiskSnapshot":
		snapshot, found := s.lightsail.GetDiskSnapshot(lightsailString(payload["diskSnapshotName"]))
		if !found {
			respondLightsailError(w, http.StatusNotFound, "NotFoundException", "disk snapshot not found")
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"diskSnapshot": lightsailDiskSnapshotPayload(snapshot)})
		return true
	case "GetDiskSnapshots":
		snapshots := s.lightsail.GetDiskSnapshots()
		out := make([]map[string]any, 0, len(snapshots))
		for _, snapshot := range snapshots {
			out = append(out, lightsailDiskSnapshotPayload(snapshot))
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"diskSnapshots": out})
		return true
	case "DeleteDiskSnapshot":
		ops, err := s.lightsail.DeleteDiskSnapshot(lightsailString(payload["diskSnapshotName"]))
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operations": lightsailOperationsPayload(ops)})
		return true
	case "CreateDiskFromSnapshot":
		sizeInGb, ok := lightsailInt32(payload["sizeInGb"])
		if !ok {
			respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "sizeInGb is required")
			return true
		}
		ops, err := s.lightsail.CreateDiskFromSnapshot(
			lightsailString(payload["availabilityZone"]),
			lightsailString(payload["diskName"]),
			lightsailString(payload["diskSnapshotName"]),
			lightsailString(payload["sourceDiskName"]),
			sizeInGb,
			lightsailTagsToMap(payload["tags"]),
		)
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operations": lightsailOperationsPayload(ops)})
		return true
	case "CopySnapshot":
		ops, err := s.lightsail.CopySnapshot(
			lightsailString(payload["sourceRegion"]),
			lightsailString(payload["targetSnapshotName"]),
			lightsailString(payload["sourceSnapshotName"]),
			lightsailString(payload["sourceResourceName"]),
		)
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operations": lightsailOperationsPayload(ops)})
		return true
	case "ExportSnapshot":
		ops, err := s.lightsail.ExportSnapshot(lightsailString(payload["sourceSnapshotName"]))
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operations": lightsailOperationsPayload(ops)})
		return true
	case "GetExportSnapshotRecords":
		records := s.lightsail.GetExportSnapshotRecords()
		out := make([]map[string]any, 0, len(records))
		for _, record := range records {
			out = append(out, lightsailExportSnapshotRecordPayload(record))
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"exportSnapshotRecords": out})
		return true
	case "CreateLoadBalancer":
		instancePort, ok := lightsailInt32(payload["instancePort"])
		if !ok {
			respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "instancePort is required")
			return true
		}
		ops, err := s.lightsail.CreateLoadBalancer(
			lightsailString(payload["loadBalancerName"]),
			instancePort,
			lightsailString(payload["certificateName"]),
			lightsailString(payload["certificateDomainName"]),
			lightsailStringSlice(payload["certificateAlternativeNames"]),
			lightsailString(payload["healthCheckPath"]),
			lightsailString(payload["ipAddressType"]),
			lightsailString(payload["tlsPolicyName"]),
			lightsailTagsToMap(payload["tags"]),
		)
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operations": lightsailOperationsPayload(ops)})
		return true
	case "AttachInstancesToLoadBalancer":
		instanceNames := lightsailStringSlice(payload["instanceNames"])
		if len(instanceNames) == 0 {
			respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "instanceNames is required")
			return true
		}
		ops, err := s.lightsail.AttachInstancesToLoadBalancer(
			lightsailString(payload["loadBalancerName"]),
			instanceNames,
		)
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operations": lightsailOperationsPayload(ops)})
		return true
	case "DetachInstancesFromLoadBalancer":
		instanceNames := lightsailStringSlice(payload["instanceNames"])
		if len(instanceNames) == 0 {
			respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "instanceNames is required")
			return true
		}
		ops, err := s.lightsail.DetachInstancesFromLoadBalancer(
			lightsailString(payload["loadBalancerName"]),
			instanceNames,
		)
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operations": lightsailOperationsPayload(ops)})
		return true
	case "GetLoadBalancer":
		loadBalancer, found := s.lightsail.GetLoadBalancer(lightsailString(payload["loadBalancerName"]))
		if !found {
			respondLightsailError(w, http.StatusNotFound, "NotFoundException", "load balancer not found")
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"loadBalancer": lightsailLoadBalancerPayload(loadBalancer)})
		return true
	case "GetLoadBalancers":
		page, err := s.lightsail.GetLoadBalancers(lightsailString(payload["pageToken"]))
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		out := make([]map[string]any, 0, len(page.LoadBalancers))
		for _, loadBalancer := range page.LoadBalancers {
			out = append(out, lightsailLoadBalancerPayload(loadBalancer))
		}
		response := map[string]any{"loadBalancers": out}
		if page.NextPageToken != "" {
			response["nextPageToken"] = page.NextPageToken
		}
		respondLightsailJSON(w, http.StatusOK, response)
		return true
	case "UpdateLoadBalancerAttribute":
		ops, err := s.lightsail.UpdateLoadBalancerAttribute(
			lightsailString(payload["loadBalancerName"]),
			lightsailString(payload["attributeName"]),
			lightsailString(payload["attributeValue"]),
		)
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operations": lightsailOperationsPayload(ops)})
		return true
	case "GetLoadBalancerMetricData":
		startTime, ok := lightsailTime(payload["startTime"])
		if !ok {
			respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "startTime is required")
			return true
		}
		endTime, ok := lightsailTime(payload["endTime"])
		if !ok {
			respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "endTime is required")
			return true
		}
		period, ok := lightsailInt32(payload["period"])
		if !ok {
			respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "period is required")
			return true
		}
		statistics := lightsailStringSlice(payload["statistics"])
		if len(statistics) == 0 {
			respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "statistics is required")
			return true
		}
		metricName, metricData, err := s.lightsail.GetLoadBalancerMetricData(lightsail.LoadBalancerMetricInput{
			LoadBalancerName: lightsailString(payload["loadBalancerName"]),
			EndTime:          endTime,
			MetricName:       lightsailString(payload["metricName"]),
			Period:           period,
			StartTime:        startTime,
			Statistics:       statistics,
			Unit:             lightsailString(payload["unit"]),
		})
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{
			"metricData": lightsailLoadBalancerMetricDatapointsPayload(metricData),
			"metricName": metricName,
		})
		return true
	case "DeleteLoadBalancer":
		ops, err := s.lightsail.DeleteLoadBalancer(lightsailString(payload["loadBalancerName"]))
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operations": lightsailOperationsPayload(ops)})
		return true
	case "CreateLoadBalancerTlsCertificate":
		ops, err := s.lightsail.CreateLoadBalancerTLSCertificate(
			lightsailString(payload["loadBalancerName"]),
			lightsailString(payload["certificateName"]),
			lightsailString(payload["certificateDomainName"]),
			lightsailStringSlice(payload["certificateAlternativeNames"]),
			lightsailTagsToMap(payload["tags"]),
		)
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operations": lightsailOperationsPayload(ops)})
		return true
	case "GetLoadBalancerTlsCertificates":
		certs := s.lightsail.GetLoadBalancerTLSCertificates(lightsailString(payload["loadBalancerName"]))
		out := make([]map[string]any, 0, len(certs))
		for _, cert := range certs {
			out = append(out, lightsailLoadBalancerTLSCertificatePayload(cert))
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"tlsCertificates": out})
		return true
	case "GetLoadBalancerTlsPolicies":
		policies := s.lightsail.GetLoadBalancerTLSPolicies()
		out := make([]map[string]any, 0, len(policies))
		for _, policy := range policies {
			out = append(out, lightsailLoadBalancerTLSPolicyPayload(policy))
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"tlsPolicies": out})
		return true
	case "AttachLoadBalancerTlsCertificate":
		ops, err := s.lightsail.AttachLoadBalancerTLSCertificate(
			lightsailString(payload["loadBalancerName"]),
			lightsailString(payload["certificateName"]),
		)
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operations": lightsailOperationsPayload(ops)})
		return true
	case "DeleteLoadBalancerTlsCertificate":
		force, _ := lightsailBool(payload["force"])
		ops, err := s.lightsail.DeleteLoadBalancerTLSCertificate(
			lightsailString(payload["loadBalancerName"]),
			lightsailString(payload["certificateName"]),
			force,
		)
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operations": lightsailOperationsPayload(ops)})
		return true
	case "SetupInstanceHttps":
		ops, err := s.lightsail.SetupInstanceHTTPS(
			lightsailString(payload["certificateProvider"]),
			lightsailStringSlice(payload["domainNames"]),
			lightsailString(payload["emailAddress"]),
			lightsailString(payload["instanceName"]),
		)
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operations": lightsailOperationsPayload(ops)})
		return true
	case "CreateCertificate":
		certificate, ops, err := s.lightsail.CreateCertificate(
			lightsailString(payload["certificateName"]),
			lightsailString(payload["domainName"]),
			lightsailStringSlice(payload["subjectAlternativeNames"]),
			lightsailTagsToMap(payload["tags"]),
		)
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{
			"certificate": lightsailCertificateSummaryPayload(certificate, true),
			"operations":  lightsailOperationsPayload(ops),
		})
		return true
	case "GetCertificates":
		includeDetails, _ := lightsailBool(payload["includeCertificateDetails"])
		certificates := s.lightsail.GetCertificates(
			lightsailString(payload["certificateName"]),
			lightsailStringSlice(payload["certificateStatuses"]),
		)
		out := make([]map[string]any, 0, len(certificates))
		for _, certificate := range certificates {
			out = append(out, lightsailCertificateSummaryPayload(certificate, includeDetails))
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"certificates": out})
		return true
	case "DeleteCertificate":
		ops, err := s.lightsail.DeleteCertificate(lightsailString(payload["certificateName"]))
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operations": lightsailOperationsPayload(ops)})
		return true
	case "AttachCertificateToDistribution":
		op, err := s.lightsail.AttachCertificateToDistribution(
			lightsailString(payload["certificateName"]),
			lightsailString(payload["distributionName"]),
		)
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operation": lightsailOperationPayload(op)})
		return true
	case "DetachCertificateFromDistribution":
		op, err := s.lightsail.DetachCertificateFromDistribution(lightsailString(payload["distributionName"]))
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operation": lightsailOperationPayload(op)})
		return true
	case "SetIpAddressType":
		var acceptBundleUpdate *bool
		if value, ok := payload["acceptBundleUpdate"]; ok {
			v, valid := lightsailBool(value)
			if !valid {
				respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "acceptBundleUpdate must be a boolean")
				return true
			}
			acceptBundleUpdate = &v
		}
		ops, err := s.lightsail.SetIPAddressType(
			lightsailString(payload["resourceName"]),
			lightsailString(payload["resourceType"]),
			lightsailString(payload["ipAddressType"]),
			acceptBundleUpdate,
		)
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operations": lightsailOperationsPayload(ops)})
		return true
	case "CreateDomain":
		op, err := s.lightsail.CreateDomain(
			lightsailString(payload["domainName"]),
			lightsailTagsToMap(payload["tags"]),
		)
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operation": lightsailOperationPayload(op)})
		return true
	case "GetDomain":
		domain, found := s.lightsail.GetDomain(lightsailString(payload["domainName"]))
		if !found {
			respondLightsailError(w, http.StatusNotFound, "NotFoundException", "domain not found")
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"domain": lightsailDomainPayload(domain)})
		return true
	case "GetDomains":
		domains := s.lightsail.GetDomains()
		out := make([]map[string]any, 0, len(domains))
		for _, domain := range domains {
			out = append(out, lightsailDomainPayload(domain))
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"domains": out})
		return true
	case "DeleteDomain":
		op, err := s.lightsail.DeleteDomain(lightsailString(payload["domainName"]))
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operation": lightsailOperationPayload(op)})
		return true
	case "CreateDomainEntry":
		domainEntry, ok := lightsailDomainEntry(payload["domainEntry"])
		if !ok {
			respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "domainEntry is required")
			return true
		}
		op, err := s.lightsail.CreateDomainEntry(lightsailString(payload["domainName"]), domainEntry)
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operation": lightsailOperationPayload(op)})
		return true
	case "UpdateDomainEntry":
		domainEntry, ok := lightsailDomainEntry(payload["domainEntry"])
		if !ok {
			respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "domainEntry is required")
			return true
		}
		ops, err := s.lightsail.UpdateDomainEntry(lightsailString(payload["domainName"]), domainEntry)
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operations": lightsailOperationsPayload(ops)})
		return true
	case "DeleteDomainEntry":
		domainEntry, ok := lightsailDomainEntry(payload["domainEntry"])
		if !ok {
			respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "domainEntry is required")
			return true
		}
		op, err := s.lightsail.DeleteDomainEntry(lightsailString(payload["domainName"]), domainEntry)
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operation": lightsailOperationPayload(op)})
		return true
	case "CreateDistribution":
		defaultCacheBehavior, ok := lightsailDistributionCacheBehavior(payload["defaultCacheBehavior"])
		if !ok {
			respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "defaultCacheBehavior is required")
			return true
		}
		origin, ok := lightsailDistributionOrigin(payload["origin"])
		if !ok {
			respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "origin is required")
			return true
		}
		distribution, op, err := s.lightsail.CreateDistribution(lightsail.DistributionCreateInput{
			BundleID:                        lightsailString(payload["bundleId"]),
			DefaultCacheBehavior:            defaultCacheBehavior,
			DistributionName:                lightsailString(payload["distributionName"]),
			Origin:                          origin,
			CacheBehaviorSettings:           lightsailDistributionCacheSettings(payload["cacheBehaviorSettings"]),
			CacheBehaviors:                  lightsailDistributionCacheBehaviors(payload["cacheBehaviors"]),
			CertificateName:                 lightsailString(payload["certificateName"]),
			IPAddressType:                   lightsailString(payload["ipAddressType"]),
			Tags:                            lightsailTagsToMap(payload["tags"]),
			ViewerMinimumTLSProtocolVersion: lightsailString(payload["viewerMinimumTlsProtocolVersion"]),
		})
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{
			"distribution": lightsailDistributionPayload(distribution),
			"operation":    lightsailOperationPayload(op),
		})
		return true
	case "GetDistributions":
		distributions := s.lightsail.GetDistributions(lightsailString(payload["distributionName"]))
		out := make([]map[string]any, 0, len(distributions))
		for _, distribution := range distributions {
			out = append(out, lightsailDistributionPayload(distribution))
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"distributions": out})
		return true
	case "DeleteDistribution":
		op, err := s.lightsail.DeleteDistribution(lightsailString(payload["distributionName"]))
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operation": lightsailOperationPayload(op)})
		return true
	case "UpdateDistribution":
		update := lightsail.DistributionUpdateInput{
			DistributionName:                lightsailString(payload["distributionName"]),
			ViewerMinimumTLSProtocolVersion: lightsailString(payload["viewerMinimumTlsProtocolVersion"]),
		}
		if value, ok := payload["cacheBehaviorSettings"]; ok {
			cacheSettings := lightsailDistributionCacheSettings(value)
			update.CacheBehaviorSettings = &cacheSettings
		}
		if value, ok := payload["cacheBehaviors"]; ok {
			update.CacheBehaviors = lightsailDistributionCacheBehaviors(value)
			update.HasCacheBehaviors = true
		}
		if value, ok := payload["certificateName"]; ok {
			certName := lightsailString(value)
			update.CertificateName = &certName
		}
		if value, ok := payload["defaultCacheBehavior"]; ok {
			defaultCacheBehavior, valid := lightsailDistributionCacheBehavior(value)
			if !valid {
				respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "defaultCacheBehavior is invalid")
				return true
			}
			update.DefaultCacheBehavior = &defaultCacheBehavior
		}
		if value, ok := payload["isEnabled"]; ok {
			isEnabled, valid := lightsailBool(value)
			if !valid {
				respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "isEnabled must be a boolean")
				return true
			}
			update.IsEnabled = &isEnabled
		}
		if value, ok := payload["origin"]; ok {
			origin, valid := lightsailDistributionOrigin(value)
			if !valid {
				respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "origin is invalid")
				return true
			}
			update.Origin = &origin
		}
		if value, ok := payload["useDefaultCertificate"]; ok {
			useDefaultCertificate, valid := lightsailBool(value)
			if !valid {
				respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "useDefaultCertificate must be a boolean")
				return true
			}
			update.UseDefaultCertificate = &useDefaultCertificate
		}
		op, err := s.lightsail.UpdateDistribution(update)
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operation": lightsailOperationPayload(op)})
		return true
	case "ResetDistributionCache":
		reset, op, err := s.lightsail.ResetDistributionCache(lightsailString(payload["distributionName"]))
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{
			"createTime": lightsailTimestamp(reset.CreateTime),
			"status":     reset.Status,
			"operation":  lightsailOperationPayload(op),
		})
		return true
	case "GetDistributionMetricData":
		startTime, ok := lightsailTime(payload["startTime"])
		if !ok {
			respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "startTime is required")
			return true
		}
		endTime, ok := lightsailTime(payload["endTime"])
		if !ok {
			respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "endTime is required")
			return true
		}
		period, ok := lightsailInt32(payload["period"])
		if !ok {
			respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "period is required")
			return true
		}
		statistics := lightsailStringSlice(payload["statistics"])
		if len(statistics) == 0 {
			respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "statistics is required")
			return true
		}
		metricName, metricData, err := s.lightsail.GetDistributionMetricData(lightsail.DistributionMetricInput{
			DistributionName: lightsailString(payload["distributionName"]),
			EndTime:          endTime,
			MetricName:       lightsailString(payload["metricName"]),
			Period:           period,
			StartTime:        startTime,
			Statistics:       statistics,
			Unit:             lightsailString(payload["unit"]),
		})
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{
			"metricData": lightsailDistributionMetricDatapointsPayload(metricData),
			"metricName": metricName,
		})
		return true
	case "GetDistributionBundles":
		bundles := s.lightsail.GetDistributionBundles()
		out := make([]map[string]any, 0, len(bundles))
		for _, bundle := range bundles {
			out = append(out, lightsailDistributionBundlePayload(bundle))
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"bundles": out})
		return true
	case "GetDistributionLatestCacheReset":
		reset, found, err := s.lightsail.GetDistributionLatestCacheReset(lightsailString(payload["distributionName"]))
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		out := map[string]any{}
		if found {
			out["createTime"] = lightsailTimestamp(reset.CreateTime)
			out["status"] = reset.Status
		}
		respondLightsailJSON(w, http.StatusOK, out)
		return true
	case "UpdateDistributionBundle":
		op, err := s.lightsail.UpdateDistributionBundle(
			lightsailString(payload["distributionName"]),
			lightsailString(payload["bundleId"]),
		)
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operation": lightsailOperationPayload(op)})
		return true
	case "CreateBucket":
		var enableObjectVersioning *bool
		if value, ok := payload["enableObjectVersioning"]; ok {
			v, valid := lightsailBool(value)
			if !valid {
				respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "enableObjectVersioning must be a boolean")
				return true
			}
			enableObjectVersioning = &v
		}
		bucket, ops, err := s.lightsail.CreateBucket(
			lightsailString(payload["bucketName"]),
			lightsailString(payload["bundleId"]),
			enableObjectVersioning,
			lightsailTagsToMap(payload["tags"]),
		)
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{
			"bucket":     lightsailBucketPayload(bucket),
			"operations": lightsailOperationsPayload(ops),
		})
		return true
	case "GetBuckets":
		includeConnectedResources, _ := lightsailBool(payload["includeConnectedResources"])
		buckets := s.lightsail.GetBuckets(
			lightsailString(payload["bucketName"]),
			includeConnectedResources,
		)
		out := make([]map[string]any, 0, len(buckets))
		for _, bucket := range buckets {
			out = append(out, lightsailBucketPayload(bucket))
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{
			"accountLevelBpaSync": map[string]any{
				"bpaImpactsLightsail": false,
				"lastSyncedAt":        lightsailTimestamp(time.Now().UTC()),
				"status":              "InSync",
			},
			"buckets": out,
		})
		return true
	case "UpdateBucket":
		update := lightsail.BucketUpdateInput{
			BucketName: lightsailString(payload["bucketName"]),
		}
		if value, ok := payload["accessLogConfig"]; ok {
			accessLogConfig, valid := lightsailBucketAccessLogConfig(value)
			if !valid {
				respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "accessLogConfig is invalid")
				return true
			}
			update.AccessLogConfig = &accessLogConfig
		}
		if value, ok := payload["accessRules"]; ok {
			accessRules, valid := lightsailBucketAccessRules(value)
			if !valid {
				respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "accessRules is invalid")
				return true
			}
			update.AccessRules = &accessRules
		}
		if value, ok := payload["readonlyAccessAccounts"]; ok {
			update.ReadonlyAccessAccounts = lightsailStringSlice(value)
			update.HasReadonlyAccessAccounts = true
		}
		if value, ok := payload["versioning"]; ok {
			versioning := lightsailString(value)
			update.Versioning = &versioning
		}
		bucket, ops, err := s.lightsail.UpdateBucket(update)
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{
			"bucket":     lightsailBucketPayload(bucket),
			"operations": lightsailOperationsPayload(ops),
		})
		return true
	case "DeleteBucket":
		forceDelete, _ := lightsailBool(payload["forceDelete"])
		ops, err := s.lightsail.DeleteBucket(
			lightsailString(payload["bucketName"]),
			forceDelete,
		)
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operations": lightsailOperationsPayload(ops)})
		return true
	case "SetResourceAccessForBucket":
		ops, err := s.lightsail.SetResourceAccessForBucket(
			lightsailString(payload["bucketName"]),
			lightsailString(payload["resourceName"]),
			lightsailString(payload["access"]),
		)
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operations": lightsailOperationsPayload(ops)})
		return true
	case "GetBucketBundles":
		includeInactive, _ := lightsailBool(payload["includeInactive"])
		bundles := s.lightsail.GetBucketBundles(includeInactive)
		out := make([]map[string]any, 0, len(bundles))
		for _, bundle := range bundles {
			out = append(out, lightsailBucketBundlePayload(bundle))
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"bundles": out})
		return true
	case "GetBucketMetricData":
		startTime, ok := lightsailTime(payload["startTime"])
		if !ok {
			respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "startTime is required")
			return true
		}
		endTime, ok := lightsailTime(payload["endTime"])
		if !ok {
			respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "endTime is required")
			return true
		}
		period, ok := lightsailInt32(payload["period"])
		if !ok {
			respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "period is required")
			return true
		}
		statistics := lightsailStringSlice(payload["statistics"])
		if len(statistics) == 0 {
			respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "statistics is required")
			return true
		}
		metricName, metricData, err := s.lightsail.GetBucketMetricData(lightsail.BucketMetricInput{
			BucketName: lightsailString(payload["bucketName"]),
			EndTime:    endTime,
			MetricName: lightsailString(payload["metricName"]),
			Period:     period,
			StartTime:  startTime,
			Statistics: statistics,
			Unit:       lightsailString(payload["unit"]),
		})
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{
			"metricData": lightsailBucketMetricDatapointsPayload(metricData),
			"metricName": metricName,
		})
		return true
	case "UpdateBucketBundle":
		ops, err := s.lightsail.UpdateBucketBundle(
			lightsailString(payload["bucketName"]),
			lightsailString(payload["bundleId"]),
		)
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operations": lightsailOperationsPayload(ops)})
		return true
	case "CreateBucketAccessKey":
		accessKey, ops, err := s.lightsail.CreateBucketAccessKey(lightsailString(payload["bucketName"]))
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{
			"accessKey":  lightsailBucketAccessKeyPayload(accessKey, true),
			"operations": lightsailOperationsPayload(ops),
		})
		return true
	case "GetBucketAccessKeys":
		accessKeys, err := s.lightsail.GetBucketAccessKeys(lightsailString(payload["bucketName"]))
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		out := make([]map[string]any, 0, len(accessKeys))
		for _, accessKey := range accessKeys {
			out = append(out, lightsailBucketAccessKeyPayload(accessKey, false))
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"accessKeys": out})
		return true
	case "DeleteBucketAccessKey":
		ops, err := s.lightsail.DeleteBucketAccessKey(
			lightsailString(payload["bucketName"]),
			lightsailString(payload["accessKeyId"]),
		)
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operations": lightsailOperationsPayload(ops)})
		return true
	case "CreateContactMethod":
		ops, err := s.lightsail.CreateContactMethod(
			lightsailString(payload["contactEndpoint"]),
			lightsailString(payload["protocol"]),
		)
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operations": lightsailOperationsPayload(ops)})
		return true
	case "GetContactMethods":
		contactMethods := s.lightsail.GetContactMethods(lightsailStringSlice(payload["protocols"]))
		out := make([]map[string]any, 0, len(contactMethods))
		for _, contactMethod := range contactMethods {
			out = append(out, lightsailContactMethodPayload(contactMethod))
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"contactMethods": out})
		return true
	case "DeleteContactMethod":
		ops, err := s.lightsail.DeleteContactMethod(lightsailString(payload["protocol"]))
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operations": lightsailOperationsPayload(ops)})
		return true
	case "SendContactMethodVerification":
		ops, err := s.lightsail.SendContactMethodVerification(lightsailString(payload["protocol"]))
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operations": lightsailOperationsPayload(ops)})
		return true
	case "CreateContainerService":
		scale, ok := lightsailInt32(payload["scale"])
		if !ok {
			respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "scale is required")
			return true
		}
		containerService, err := s.lightsail.CreateContainerService(
			lightsailString(payload["serviceName"]),
			lightsailString(payload["power"]),
			scale,
			lightsailStringSliceMap(payload["publicDomainNames"]),
			lightsailTagsToMap(payload["tags"]),
		)
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{
			"containerService": lightsailContainerServicePayload(containerService),
		})
		return true
	case "GetContainerServices":
		containerServices := s.lightsail.GetContainerServices(lightsailString(payload["serviceName"]))
		out := make([]map[string]any, 0, len(containerServices))
		for _, containerService := range containerServices {
			out = append(out, lightsailContainerServicePayload(containerService))
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"containerServices": out})
		return true
	case "UpdateContainerService":
		update := lightsail.ContainerServiceUpdateInput{
			ServiceName: lightsailString(payload["serviceName"]),
		}
		if value, ok := payload["power"]; ok {
			power := lightsailString(value)
			update.Power = &power
		}
		if value, ok := payload["scale"]; ok {
			scale, valid := lightsailInt32(value)
			if !valid {
				respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "scale must be an integer")
				return true
			}
			update.Scale = &scale
		}
		if value, ok := payload["isDisabled"]; ok {
			isDisabled, valid := lightsailBool(value)
			if !valid {
				respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "isDisabled must be a boolean")
				return true
			}
			update.IsDisabled = &isDisabled
		}
		if value, ok := payload["publicDomainNames"]; ok {
			update.PublicDomainNames = lightsailStringSliceMap(value)
			update.HasPublicDomainNames = true
		}
		containerService, err := s.lightsail.UpdateContainerService(update)
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{
			"containerService": lightsailContainerServicePayload(containerService),
		})
		return true
	case "DeleteContainerService":
		if err := s.lightsail.DeleteContainerService(lightsailString(payload["serviceName"])); err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{})
		return true
	case "GetContainerAPIMetadata":
		respondLightsailJSON(w, http.StatusOK, map[string]any{"metadata": s.lightsail.GetContainerAPIMetadata()})
		return true
	case "GetContainerServiceMetricData":
		startTime, ok := lightsailTime(payload["startTime"])
		if !ok {
			respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "startTime is required")
			return true
		}
		endTime, ok := lightsailTime(payload["endTime"])
		if !ok {
			respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "endTime is required")
			return true
		}
		period, ok := lightsailInt32(payload["period"])
		if !ok {
			respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "period is required")
			return true
		}
		statistics := lightsailStringSlice(payload["statistics"])
		if len(statistics) == 0 {
			respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "statistics is required")
			return true
		}
		metricName, metricData, err := s.lightsail.GetContainerServiceMetricData(lightsail.ContainerServiceMetricInput{
			ServiceName: lightsailString(payload["serviceName"]),
			EndTime:     endTime,
			MetricName:  lightsailString(payload["metricName"]),
			Period:      period,
			StartTime:   startTime,
			Statistics:  statistics,
		})
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{
			"metricData": lightsailContainerServiceMetricDatapointsPayload(metricData),
			"metricName": metricName,
		})
		return true
	case "GetContainerServicePowers":
		powers := s.lightsail.GetContainerServicePowers()
		out := make([]map[string]any, 0, len(powers))
		for _, power := range powers {
			out = append(out, lightsailContainerServicePowerPayload(power))
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"powers": out})
		return true
	case "CreateContainerServiceRegistryLogin":
		registryLogin := s.lightsail.CreateContainerServiceRegistryLogin()
		respondLightsailJSON(w, http.StatusOK, map[string]any{
			"registryLogin": lightsailContainerServiceRegistryLoginPayload(registryLogin),
		})
		return true
	case "CreateContainerServiceDeployment":
		containers := lightsailContainerDefinitions(payload["containers"])
		if len(containers) == 0 {
			respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "containers is required")
			return true
		}
		publicEndpoint, ok := lightsailEndpointRequest(payload["publicEndpoint"])
		if !ok {
			respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "publicEndpoint is invalid")
			return true
		}
		containerService, err := s.lightsail.CreateContainerServiceDeployment(
			lightsailString(payload["serviceName"]),
			containers,
			publicEndpoint,
		)
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{
			"containerService": lightsailContainerServicePayload(containerService),
		})
		return true
	case "GetContainerServiceDeployments":
		deployments, err := s.lightsail.GetContainerServiceDeployments(lightsailString(payload["serviceName"]))
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{
			"deployments": lightsailContainerServiceDeploymentsPayload(deployments),
		})
		return true
	case "RegisterContainerImage":
		containerImage, err := s.lightsail.RegisterContainerImage(
			lightsailString(payload["serviceName"]),
			lightsailString(payload["label"]),
			lightsailString(payload["digest"]),
		)
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{
			"containerImage": lightsailContainerImagePayload(containerImage),
		})
		return true
	case "GetContainerImages":
		containerImages, err := s.lightsail.GetContainerImages(lightsailString(payload["serviceName"]))
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{
			"containerImages": lightsailContainerImagesPayload(containerImages),
		})
		return true
	case "DeleteContainerImage":
		err := s.lightsail.DeleteContainerImage(
			lightsailString(payload["serviceName"]),
			lightsailString(payload["image"]),
		)
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{})
		return true
	case "GetContainerLog":
		var startTimePtr *time.Time
		if _, exists := payload["startTime"]; exists {
			startTime, ok := lightsailTime(payload["startTime"])
			if !ok {
				respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "startTime is invalid")
				return true
			}
			startTimePtr = &startTime
		}
		var endTimePtr *time.Time
		if _, exists := payload["endTime"]; exists {
			endTime, ok := lightsailTime(payload["endTime"])
			if !ok {
				respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "endTime is invalid")
				return true
			}
			endTimePtr = &endTime
		}
		logEvents, nextPageToken, err := s.lightsail.GetContainerLog(lightsail.ContainerLogInput{
			ServiceName:   lightsailString(payload["serviceName"]),
			ContainerName: lightsailString(payload["containerName"]),
			StartTime:     startTimePtr,
			EndTime:       endTimePtr,
			FilterPattern: lightsailString(payload["filterPattern"]),
			PageToken:     lightsailString(payload["pageToken"]),
		})
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		response := map[string]any{
			"logEvents": lightsailContainerServiceLogEventsPayload(logEvents),
		}
		if nextPageToken != "" {
			response["nextPageToken"] = nextPageToken
		}
		respondLightsailJSON(w, http.StatusOK, response)
		return true
	case "CreateRelationalDatabase":
		publiclyAccessible, err := lightsailOptionalBool(payload["publiclyAccessible"])
		if err != nil {
			respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "publiclyAccessible must be a boolean")
			return true
		}
		ops, err := s.lightsail.CreateRelationalDatabase(lightsail.RelationalDatabaseCreateInput{
			RelationalDatabaseName:        lightsailString(payload["relationalDatabaseName"]),
			AvailabilityZone:              lightsailString(payload["availabilityZone"]),
			MasterDatabaseName:            lightsailString(payload["masterDatabaseName"]),
			MasterUsername:                lightsailString(payload["masterUsername"]),
			MasterUserPassword:            lightsailString(payload["masterUserPassword"]),
			RelationalDatabaseBlueprintID: lightsailString(payload["relationalDatabaseBlueprintId"]),
			RelationalDatabaseBundleID:    lightsailString(payload["relationalDatabaseBundleId"]),
			PreferredBackupWindow:         lightsailString(payload["preferredBackupWindow"]),
			PreferredMaintenanceWindow:    lightsailString(payload["preferredMaintenanceWindow"]),
			PubliclyAccessible:            publiclyAccessible,
			Tags:                          lightsailTagsToMap(payload["tags"]),
		})
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operations": lightsailOperationsPayload(ops)})
		return true
	case "GetRelationalDatabaseBlueprints":
		page, err := s.lightsail.GetRelationalDatabaseBlueprints(lightsailString(payload["pageToken"]))
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		blueprints := make([]map[string]any, 0, len(page.Blueprints))
		for _, blueprint := range page.Blueprints {
			blueprints = append(blueprints, lightsailRelationalDatabaseBlueprintPayload(blueprint))
		}
		response := map[string]any{"blueprints": blueprints}
		if page.NextPageToken != "" {
			response["nextPageToken"] = page.NextPageToken
		}
		respondLightsailJSON(w, http.StatusOK, response)
		return true
	case "GetRelationalDatabaseBundles":
		includeInactive, _ := lightsailBool(payload["includeInactive"])
		page, err := s.lightsail.GetRelationalDatabaseBundles(includeInactive, lightsailString(payload["pageToken"]))
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		bundles := make([]map[string]any, 0, len(page.Bundles))
		for _, bundle := range page.Bundles {
			bundles = append(bundles, lightsailRelationalDatabaseBundlePayload(bundle))
		}
		response := map[string]any{"bundles": bundles}
		if page.NextPageToken != "" {
			response["nextPageToken"] = page.NextPageToken
		}
		respondLightsailJSON(w, http.StatusOK, response)
		return true
	case "GetRelationalDatabase":
		relationalDatabase, found := s.lightsail.GetRelationalDatabase(lightsailString(payload["relationalDatabaseName"]))
		if !found {
			respondLightsailError(w, http.StatusNotFound, "NotFoundException", "relational database not found")
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{
			"relationalDatabase": lightsailRelationalDatabasePayload(relationalDatabase),
		})
		return true
	case "GetRelationalDatabases":
		page, err := s.lightsail.GetRelationalDatabases(lightsailString(payload["pageToken"]))
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		relationalDatabasesOut := make([]map[string]any, 0, len(page.RelationalDatabases))
		for _, relationalDatabase := range page.RelationalDatabases {
			relationalDatabasesOut = append(relationalDatabasesOut, lightsailRelationalDatabasePayload(relationalDatabase))
		}
		response := map[string]any{"relationalDatabases": relationalDatabasesOut}
		if page.NextPageToken != "" {
			response["nextPageToken"] = page.NextPageToken
		}
		respondLightsailJSON(w, http.StatusOK, response)
		return true
	case "UpdateRelationalDatabase":
		update := lightsail.RelationalDatabaseUpdateInput{
			RelationalDatabaseName: lightsailString(payload["relationalDatabaseName"]),
		}
		if value, ok := payload["applyImmediately"]; ok {
			parsed, valid := lightsailBool(value)
			if !valid {
				respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "applyImmediately must be a boolean")
				return true
			}
			update.ApplyImmediately = &parsed
		}
		if value, ok := payload["caCertificateIdentifier"]; ok {
			parsed := lightsailString(value)
			update.CACertificateIdentifier = &parsed
		}
		if value, ok := payload["disableBackupRetention"]; ok {
			parsed, valid := lightsailBool(value)
			if !valid {
				respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "disableBackupRetention must be a boolean")
				return true
			}
			update.DisableBackupRetention = &parsed
		}
		if value, ok := payload["enableBackupRetention"]; ok {
			parsed, valid := lightsailBool(value)
			if !valid {
				respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "enableBackupRetention must be a boolean")
				return true
			}
			update.EnableBackupRetention = &parsed
		}
		if value, ok := payload["masterUserPassword"]; ok {
			parsed := lightsailString(value)
			update.MasterUserPassword = &parsed
		}
		if value, ok := payload["preferredBackupWindow"]; ok {
			parsed := lightsailString(value)
			update.PreferredBackupWindow = &parsed
		}
		if value, ok := payload["preferredMaintenanceWindow"]; ok {
			parsed := lightsailString(value)
			update.PreferredMaintenanceWindow = &parsed
		}
		if value, ok := payload["publiclyAccessible"]; ok {
			parsed, valid := lightsailBool(value)
			if !valid {
				respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "publiclyAccessible must be a boolean")
				return true
			}
			update.PubliclyAccessible = &parsed
		}
		if value, ok := payload["relationalDatabaseBlueprintId"]; ok {
			parsed := lightsailString(value)
			update.RelationalDatabaseBlueprintID = &parsed
		}
		if value, ok := payload["rotateMasterUserPassword"]; ok {
			parsed, valid := lightsailBool(value)
			if !valid {
				respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "rotateMasterUserPassword must be a boolean")
				return true
			}
			update.RotateMasterUserPassword = &parsed
		}
		ops, err := s.lightsail.UpdateRelationalDatabase(update)
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operations": lightsailOperationsPayload(ops)})
		return true
	case "DeleteRelationalDatabase":
		skipFinalSnapshot, err := lightsailOptionalBool(payload["skipFinalSnapshot"])
		if err != nil {
			respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "skipFinalSnapshot must be a boolean")
			return true
		}
		ops, err := s.lightsail.DeleteRelationalDatabase(lightsail.RelationalDatabaseDeleteInput{
			RelationalDatabaseName:              lightsailString(payload["relationalDatabaseName"]),
			FinalRelationalDatabaseSnapshotName: lightsailString(payload["finalRelationalDatabaseSnapshotName"]),
			SkipFinalSnapshot:                   skipFinalSnapshot,
		})
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operations": lightsailOperationsPayload(ops)})
		return true
	case "RebootRelationalDatabase":
		ops, err := s.lightsail.RebootRelationalDatabase(lightsailString(payload["relationalDatabaseName"]))
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operations": lightsailOperationsPayload(ops)})
		return true
	case "StartRelationalDatabase":
		ops, err := s.lightsail.StartRelationalDatabase(lightsailString(payload["relationalDatabaseName"]))
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operations": lightsailOperationsPayload(ops)})
		return true
	case "StopRelationalDatabase":
		ops, err := s.lightsail.StopRelationalDatabase(
			lightsailString(payload["relationalDatabaseName"]),
			lightsailString(payload["relationalDatabaseSnapshotName"]),
		)
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operations": lightsailOperationsPayload(ops)})
		return true
	case "CreateRelationalDatabaseSnapshot":
		ops, err := s.lightsail.CreateRelationalDatabaseSnapshot(
			lightsailString(payload["relationalDatabaseName"]),
			lightsailString(payload["relationalDatabaseSnapshotName"]),
			lightsailTagsToMap(payload["tags"]),
		)
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operations": lightsailOperationsPayload(ops)})
		return true
	case "GetRelationalDatabaseSnapshot":
		relationalDatabaseSnapshot, found := s.lightsail.GetRelationalDatabaseSnapshot(lightsailString(payload["relationalDatabaseSnapshotName"]))
		if !found {
			respondLightsailError(w, http.StatusNotFound, "NotFoundException", "relational database snapshot not found")
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{
			"relationalDatabaseSnapshot": lightsailRelationalDatabaseSnapshotPayload(relationalDatabaseSnapshot),
		})
		return true
	case "GetRelationalDatabaseSnapshots":
		page, err := s.lightsail.GetRelationalDatabaseSnapshots(lightsailString(payload["pageToken"]))
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		snapshots := make([]map[string]any, 0, len(page.RelationalDatabaseSnapshots))
		for _, snapshot := range page.RelationalDatabaseSnapshots {
			snapshots = append(snapshots, lightsailRelationalDatabaseSnapshotPayload(snapshot))
		}
		response := map[string]any{"relationalDatabaseSnapshots": snapshots}
		if page.NextPageToken != "" {
			response["nextPageToken"] = page.NextPageToken
		}
		respondLightsailJSON(w, http.StatusOK, response)
		return true
	case "DeleteRelationalDatabaseSnapshot":
		ops, err := s.lightsail.DeleteRelationalDatabaseSnapshot(lightsailString(payload["relationalDatabaseSnapshotName"]))
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operations": lightsailOperationsPayload(ops)})
		return true
	case "CreateRelationalDatabaseFromSnapshot":
		publiclyAccessible, err := lightsailOptionalBool(payload["publiclyAccessible"])
		if err != nil {
			respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "publiclyAccessible must be a boolean")
			return true
		}
		useLatestRestorableTime, err := lightsailOptionalBool(payload["useLatestRestorableTime"])
		if err != nil {
			respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "useLatestRestorableTime must be a boolean")
			return true
		}
		var restoreTimePtr *time.Time
		if _, exists := payload["restoreTime"]; exists {
			restoreTime, ok := lightsailTime(payload["restoreTime"])
			if !ok {
				respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "restoreTime is invalid")
				return true
			}
			restoreTimePtr = &restoreTime
		}
		ops, err := s.lightsail.CreateRelationalDatabaseFromSnapshot(lightsail.RelationalDatabaseFromSnapshotInput{
			RelationalDatabaseName:         lightsailString(payload["relationalDatabaseName"]),
			AvailabilityZone:               lightsailString(payload["availabilityZone"]),
			PubliclyAccessible:             publiclyAccessible,
			RelationalDatabaseBundleID:     lightsailString(payload["relationalDatabaseBundleId"]),
			RelationalDatabaseSnapshotName: lightsailString(payload["relationalDatabaseSnapshotName"]),
			RestoreTime:                    restoreTimePtr,
			SourceRelationalDatabaseName:   lightsailString(payload["sourceRelationalDatabaseName"]),
			Tags:                           lightsailTagsToMap(payload["tags"]),
			UseLatestRestorableTime:        useLatestRestorableTime,
		})
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operations": lightsailOperationsPayload(ops)})
		return true
	case "GetRelationalDatabaseEvents":
		durationInMinutes, err := lightsailOptionalInt32(payload["durationInMinutes"])
		if err != nil {
			respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "durationInMinutes must be an integer")
			return true
		}
		page, err := s.lightsail.GetRelationalDatabaseEvents(
			lightsailString(payload["relationalDatabaseName"]),
			durationInMinutes,
			lightsailString(payload["pageToken"]),
		)
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		events := make([]map[string]any, 0, len(page.RelationalDatabaseEvents))
		for _, event := range page.RelationalDatabaseEvents {
			events = append(events, lightsailRelationalDatabaseEventPayload(event))
		}
		response := map[string]any{"relationalDatabaseEvents": events}
		if page.NextPageToken != "" {
			response["nextPageToken"] = page.NextPageToken
		}
		respondLightsailJSON(w, http.StatusOK, response)
		return true
	case "GetRelationalDatabaseLogStreams":
		logStreams, err := s.lightsail.GetRelationalDatabaseLogStreams(lightsailString(payload["relationalDatabaseName"]))
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"logStreams": logStreams})
		return true
	case "GetRelationalDatabaseLogEvents":
		startFromHead, err := lightsailOptionalBool(payload["startFromHead"])
		if err != nil {
			respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "startFromHead must be a boolean")
			return true
		}
		var startTimePtr *time.Time
		if _, exists := payload["startTime"]; exists {
			startTime, ok := lightsailTime(payload["startTime"])
			if !ok {
				respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "startTime is invalid")
				return true
			}
			startTimePtr = &startTime
		}
		var endTimePtr *time.Time
		if _, exists := payload["endTime"]; exists {
			endTime, ok := lightsailTime(payload["endTime"])
			if !ok {
				respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "endTime is invalid")
				return true
			}
			endTimePtr = &endTime
		}
		page, err := s.lightsail.GetRelationalDatabaseLogEvents(lightsail.RelationalDatabaseLogEventsInput{
			RelationalDatabaseName: lightsailString(payload["relationalDatabaseName"]),
			LogStreamName:          lightsailString(payload["logStreamName"]),
			StartTime:              startTimePtr,
			EndTime:                endTimePtr,
			PageToken:              lightsailString(payload["pageToken"]),
			StartFromHead:          startFromHead,
		})
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		response := map[string]any{
			"resourceLogEvents": lightsailRelationalDatabaseLogEventsPayload(page.ResourceLogEvents),
		}
		if page.NextBackwardToken != "" {
			response["nextBackwardToken"] = page.NextBackwardToken
		}
		if page.NextForwardToken != "" {
			response["nextForwardToken"] = page.NextForwardToken
		}
		respondLightsailJSON(w, http.StatusOK, response)
		return true
	case "GetRelationalDatabaseMasterUserPassword":
		createdAt, masterUserPassword, err := s.lightsail.GetRelationalDatabaseMasterUserPassword(
			lightsailString(payload["relationalDatabaseName"]),
			lightsailString(payload["passwordVersion"]),
		)
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{
			"createdAt":          lightsailTimestamp(createdAt),
			"masterUserPassword": masterUserPassword,
		})
		return true
	case "GetRelationalDatabaseMetricData":
		startTime, ok := lightsailTime(payload["startTime"])
		if !ok {
			respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "startTime is required")
			return true
		}
		endTime, ok := lightsailTime(payload["endTime"])
		if !ok {
			respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "endTime is required")
			return true
		}
		period, ok := lightsailInt32(payload["period"])
		if !ok {
			respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "period is required")
			return true
		}
		statistics := lightsailStringSlice(payload["statistics"])
		if len(statistics) == 0 {
			respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "statistics is required")
			return true
		}
		metricName, metricData, err := s.lightsail.GetRelationalDatabaseMetricData(lightsail.RelationalDatabaseMetricInput{
			RelationalDatabaseName: lightsailString(payload["relationalDatabaseName"]),
			EndTime:                endTime,
			MetricName:             lightsailString(payload["metricName"]),
			Period:                 period,
			StartTime:              startTime,
			Statistics:             statistics,
			Unit:                   lightsailString(payload["unit"]),
		})
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{
			"metricData": lightsailRelationalDatabaseMetricDatapointsPayload(metricData),
			"metricName": metricName,
		})
		return true
	case "GetRelationalDatabaseParameters":
		page, err := s.lightsail.GetRelationalDatabaseParameters(
			lightsailString(payload["relationalDatabaseName"]),
			lightsailString(payload["pageToken"]),
		)
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		parameters := make([]map[string]any, 0, len(page.Parameters))
		for _, parameter := range page.Parameters {
			parameters = append(parameters, lightsailRelationalDatabaseParameterPayload(parameter))
		}
		response := map[string]any{"parameters": parameters}
		if page.NextPageToken != "" {
			response["nextPageToken"] = page.NextPageToken
		}
		respondLightsailJSON(w, http.StatusOK, response)
		return true
	case "UpdateRelationalDatabaseParameters":
		parameters, ok := lightsailRelationalDatabaseParameters(payload["parameters"])
		if !ok || len(parameters) == 0 {
			respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "parameters is required")
			return true
		}
		ops, err := s.lightsail.UpdateRelationalDatabaseParameters(
			lightsailString(payload["relationalDatabaseName"]),
			parameters,
		)
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operations": lightsailOperationsPayload(ops)})
		return true
	case "PutAlarm":
		evaluationPeriods, ok := lightsailInt32(payload["evaluationPeriods"])
		if !ok {
			respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "evaluationPeriods is required")
			return true
		}
		threshold, ok := lightsailFloat64(payload["threshold"])
		if !ok {
			respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "threshold is required")
			return true
		}
		datapointsToAlarm, err := lightsailOptionalInt32(payload["datapointsToAlarm"])
		if err != nil {
			respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "datapointsToAlarm must be an integer")
			return true
		}
		var notificationEnabledPtr *bool
		if v, ok := lightsailBool(payload["notificationEnabled"]); ok {
			notificationEnabledPtr = &v
		}
		ops, err := s.lightsail.PutAlarm(
			lightsailString(payload["alarmName"]),
			lightsailString(payload["comparisonOperator"]),
			lightsailString(payload["metricName"]),
			lightsailString(payload["monitoredResourceName"]),
			evaluationPeriods,
			threshold,
			lightsailStringSlice(payload["contactProtocols"]),
			datapointsToAlarm,
			notificationEnabledPtr,
			lightsailStringSlice(payload["notificationTriggers"]),
			lightsailString(payload["treatMissingData"]),
		)
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operations": lightsailOperationsPayload(ops)})
		return true
	case "GetAlarms":
		alarms := s.lightsail.GetAlarms(
			lightsailString(payload["alarmName"]),
			lightsailString(payload["monitoredResourceName"]),
		)
		out := make([]map[string]any, 0, len(alarms))
		for _, alarm := range alarms {
			out = append(out, lightsailAlarmPayload(alarm))
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"alarms": out})
		return true
	case "TestAlarm":
		ops, err := s.lightsail.TestAlarm(
			lightsailString(payload["alarmName"]),
			lightsailString(payload["state"]),
		)
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operations": lightsailOperationsPayload(ops)})
		return true
	case "DeleteAlarm":
		ops, err := s.lightsail.DeleteAlarm(lightsailString(payload["alarmName"]))
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operations": lightsailOperationsPayload(ops)})
		return true
	case "GetAutoSnapshots":
		autoSnapshots, resourceType, err := s.lightsail.GetAutoSnapshots(lightsailString(payload["resourceName"]))
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		out := make([]map[string]any, 0, len(autoSnapshots))
		for _, snapshot := range autoSnapshots {
			out = append(out, lightsailAutoSnapshotPayload(snapshot))
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{
			"autoSnapshots": out,
			"resourceName":  lightsailString(payload["resourceName"]),
			"resourceType":  resourceType,
		})
		return true
	case "DeleteAutoSnapshot":
		ops, err := s.lightsail.DeleteAutoSnapshot(
			lightsailString(payload["resourceName"]),
			lightsailString(payload["date"]),
		)
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operations": lightsailOperationsPayload(ops)})
		return true
	case "EnableAddOn":
		addOnType, snapshotTimeOfDay, ok := lightsailAddOnRequest(payload["addOnRequest"])
		if !ok {
			respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", "addOnRequest is required")
			return true
		}
		ops, err := s.lightsail.EnableAddOn(
			lightsailString(payload["resourceName"]),
			addOnType,
			snapshotTimeOfDay,
		)
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operations": lightsailOperationsPayload(ops)})
		return true
	case "DisableAddOn":
		ops, err := s.lightsail.DisableAddOn(
			lightsailString(payload["resourceName"]),
			lightsailString(payload["addOnType"]),
		)
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operations": lightsailOperationsPayload(ops)})
		return true
	case "CreateKeyPair":
		keyPair, op, err := s.lightsail.CreateKeyPair(
			lightsailString(payload["keyPairName"]),
			lightsailTagsToMap(payload["tags"]),
		)
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{
			"keyPair":          lightsailKeyPairPayload(keyPair),
			"operation":        lightsailOperationPayload(op),
			"privateKeyBase64": keyPair.PrivateKeyBase64,
			"publicKeyBase64":  keyPair.PublicKeyBase64,
		})
		return true
	case "ImportKeyPair":
		op, err := s.lightsail.ImportKeyPair(
			lightsailString(payload["keyPairName"]),
			lightsailString(payload["publicKeyBase64"]),
		)
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operation": lightsailOperationPayload(op)})
		return true
	case "GetKeyPair":
		keyPair, found := s.lightsail.GetKeyPair(lightsailString(payload["keyPairName"]))
		if !found {
			respondLightsailError(w, http.StatusNotFound, "NotFoundException", "key pair not found")
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"keyPair": lightsailKeyPairPayload(keyPair)})
		return true
	case "GetKeyPairs":
		includeDefault, _ := lightsailBool(payload["includeDefaultKeyPair"])
		keyPairs := s.lightsail.GetKeyPairs(includeDefault)
		out := make([]map[string]any, 0, len(keyPairs))
		for _, keyPair := range keyPairs {
			out = append(out, lightsailKeyPairPayload(keyPair))
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"keyPairs": out})
		return true
	case "DeleteKeyPair":
		op, err := s.lightsail.DeleteKeyPair(
			lightsailString(payload["keyPairName"]),
			lightsailString(payload["expectedFingerprint"]),
		)
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operation": lightsailOperationPayload(op)})
		return true
	case "DownloadDefaultKeyPair":
		createdAt, privateKeyBase64, publicKeyBase64, err := s.lightsail.DownloadDefaultKeyPair()
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{
			"createdAt":        lightsailTimestamp(createdAt),
			"privateKeyBase64": privateKeyBase64,
			"publicKeyBase64":  publicKeyBase64,
		})
		return true
	case "DeleteKnownHostKeys":
		ops, err := s.lightsail.DeleteKnownHostKeys(lightsailString(payload["instanceName"]))
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operations": lightsailOperationsPayload(ops)})
		return true
	case "CreateInstanceSnapshot":
		ops, err := s.lightsail.CreateInstanceSnapshot(
			lightsailString(payload["instanceName"]),
			lightsailString(payload["instanceSnapshotName"]),
			lightsailTagsToMap(payload["tags"]),
		)
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operations": lightsailOperationsPayload(ops)})
		return true
	case "GetInstanceSnapshot":
		snapshot, found := s.lightsail.GetInstanceSnapshot(lightsailString(payload["instanceSnapshotName"]))
		if !found {
			respondLightsailError(w, http.StatusNotFound, "NotFoundException", "instance snapshot not found")
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"instanceSnapshot": lightsailInstanceSnapshotPayload(snapshot)})
		return true
	case "GetInstanceSnapshots":
		snapshots := s.lightsail.GetInstanceSnapshots()
		out := make([]map[string]any, 0, len(snapshots))
		for _, snapshot := range snapshots {
			out = append(out, lightsailInstanceSnapshotPayload(snapshot))
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"instanceSnapshots": out})
		return true
	case "DeleteInstanceSnapshot":
		ops, err := s.lightsail.DeleteInstanceSnapshot(lightsailString(payload["instanceSnapshotName"]))
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operations": lightsailOperationsPayload(ops)})
		return true
	case "AllocateStaticIp":
		ops, err := s.lightsail.AllocateStaticIP(lightsailString(payload["staticIpName"]))
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operations": lightsailOperationsPayload(ops)})
		return true
	case "GetStaticIp":
		ip, found := s.lightsail.GetStaticIP(lightsailString(payload["staticIpName"]))
		if !found {
			respondLightsailError(w, http.StatusNotFound, "NotFoundException", "static ip not found")
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"staticIp": lightsailStaticIpPayload(ip)})
		return true
	case "GetStaticIps":
		ips := s.lightsail.GetStaticIPs()
		out := make([]map[string]any, 0, len(ips))
		for _, ip := range ips {
			out = append(out, lightsailStaticIpPayload(ip))
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"staticIps": out})
		return true
	case "AttachStaticIp":
		ops, err := s.lightsail.AttachStaticIP(lightsailString(payload["staticIpName"]), lightsailString(payload["instanceName"]))
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operations": lightsailOperationsPayload(ops)})
		return true
	case "DetachStaticIp":
		ops, err := s.lightsail.DetachStaticIP(lightsailString(payload["staticIpName"]))
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operations": lightsailOperationsPayload(ops)})
		return true
	case "ReleaseStaticIp":
		ops, err := s.lightsail.ReleaseStaticIP(lightsailString(payload["staticIpName"]))
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operations": lightsailOperationsPayload(ops)})
		return true
	case "TagResource":
		resourceName := firstNonEmpty(
			lightsailString(payload["resourceName"]),
			s.lightsail.ResourceNameFromARN(lightsailString(payload["resourceArn"])),
		)
		ops, err := s.lightsail.TagResource(resourceName, lightsailTagsToMap(payload["tags"]))
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operations": lightsailOperationsPayload(ops)})
		return true
	case "UntagResource":
		resourceName := firstNonEmpty(
			lightsailString(payload["resourceName"]),
			s.lightsail.ResourceNameFromARN(lightsailString(payload["resourceArn"])),
		)
		ops, err := s.lightsail.UntagResource(resourceName, lightsailStringSlice(payload["tagKeys"]))
		if err != nil {
			respondLightsailErrorForErr(w, err)
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operations": lightsailOperationsPayload(ops)})
		return true
	case "GetRegions":
		includeAZ, _ := lightsailBool(payload["includeAvailabilityZones"])
		includeDBAZ, _ := lightsailBool(payload["includeRelationalDatabaseAvailabilityZones"])
		regions := s.lightsail.GetRegions(includeAZ, includeDBAZ)
		respondLightsailJSON(w, http.StatusOK, map[string]any{"regions": lightsailRegionsPayload(regions)})
		return true
	case "GetOperation":
		op, found := s.lightsail.GetOperation(lightsailString(payload["operationId"]))
		if !found {
			respondLightsailError(w, http.StatusNotFound, "NotFoundException", "operation not found")
			return true
		}
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operation": lightsailOperationPayload(op)})
		return true
	case "GetOperations":
		ops := s.lightsail.GetOperations()
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operations": lightsailOperationsPayload(ops)})
		return true
	case "GetOperationsForResource":
		ops := s.lightsail.GetOperationsForResource(lightsailString(payload["resourceName"]))
		respondLightsailJSON(w, http.StatusOK, map[string]any{"operations": lightsailOperationsPayload(ops)})
		return true
	default:
		respondLightsailError(w, http.StatusNotImplemented, "NotImplementedException", action+" is not implemented")
		return true
	}
}

func respondLightsailJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondLightsailError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondLightsailJSON(w, status, lightsailError{
		Code:    code,
		Message: msg,
	})
}

func respondLightsailErrorForErr(w http.ResponseWriter, err error) {
	switch err {
	case lightsail.ErrInvalidParameter:
		respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", err.Error())
	case lightsail.ErrAlreadyExists:
		respondLightsailError(w, http.StatusBadRequest, "InvalidInputException", err.Error())
	case lightsail.ErrNotFound:
		respondLightsailError(w, http.StatusNotFound, "NotFoundException", err.Error())
	default:
		respondLightsailError(w, http.StatusBadRequest, "OperationFailureException", err.Error())
	}
}

func isLightsailJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.HasPrefix(target, "Lightsail_20161128.") {
		return true
	}
	contentType := strings.ToLower(r.Header.Get("Content-Type"))
	if strings.Contains(contentType, "application/x-amz-json-1.1") || strings.Contains(contentType, "application/x-amz-json-1.0") {
		return strings.HasPrefix(target, "Lightsail")
	}
	return false
}

func parseLightsailTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "Lightsail_20161128.") {
		return strings.TrimPrefix(target, "Lightsail_20161128.")
	}
	if strings.HasPrefix(target, "Lightsail.") {
		return strings.TrimPrefix(target, "Lightsail.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parseLightsailPayload(r *http.Request) (map[string]any, error) {
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

func lightsailString(v any) string {
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(s)
}

func lightsailStringSlice(v any) []string {
	items, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		value := lightsailString(item)
		if value == "" {
			continue
		}
		out = append(out, value)
	}
	return out
}

func lightsailBool(v any) (bool, bool) {
	b, ok := v.(bool)
	return b, ok
}

func lightsailOptionalInt32(v any) (*int32, error) {
	if v == nil {
		return nil, nil
	}
	switch value := v.(type) {
	case float64:
		out := int32(value)
		if float64(out) != value {
			return nil, lightsail.ErrInvalidParameter
		}
		return &out, nil
	case int32:
		out := value
		return &out, nil
	case int:
		out := int32(value)
		return &out, nil
	default:
		return nil, lightsail.ErrInvalidParameter
	}
}

func lightsailOptionalBool(v any) (*bool, error) {
	if v == nil {
		return nil, nil
	}
	switch value := v.(type) {
	case bool:
		out := value
		return &out, nil
	default:
		return nil, lightsail.ErrInvalidParameter
	}
}

func lightsailPortInfo(v any) (lightsail.PortInfo, bool) {
	obj, ok := v.(map[string]any)
	if !ok {
		return lightsail.PortInfo{}, false
	}
	fromPort, fromSet := lightsailInt32(obj["fromPort"])
	toPort, toSet := lightsailInt32(obj["toPort"])
	info := lightsail.PortInfo{
		FromPort:        fromPort,
		ToPort:          toPort,
		Protocol:        lightsailString(obj["protocol"]),
		Cidrs:           lightsailStringSlice(obj["cidrs"]),
		Ipv6Cidrs:       lightsailStringSlice(obj["ipv6Cidrs"]),
		CidrListAliases: lightsailStringSlice(obj["cidrListAliases"]),
	}
	if !fromSet || !toSet || info.Protocol == "" {
		return lightsail.PortInfo{}, false
	}
	return info, true
}

func lightsailPortInfos(v any) []lightsail.PortInfo {
	items, ok := v.([]any)
	if !ok {
		return []lightsail.PortInfo{}
	}
	out := make([]lightsail.PortInfo, 0, len(items))
	for _, item := range items {
		info, ok := lightsailPortInfo(item)
		if !ok {
			continue
		}
		out = append(out, info)
	}
	return out
}

func lightsailInt32(v any) (int32, bool) {
	switch value := v.(type) {
	case float64:
		out := int32(value)
		if float64(out) != value {
			return 0, false
		}
		return out, true
	case int32:
		return value, true
	case int:
		return int32(value), true
	default:
		return 0, false
	}
}

func lightsailInt64(v any) (int64, bool) {
	switch value := v.(type) {
	case float64:
		out := int64(value)
		if float64(out) != value {
			return 0, false
		}
		return out, true
	case int64:
		return value, true
	case int:
		return int64(value), true
	case int32:
		return int64(value), true
	default:
		return 0, false
	}
}

func lightsailTime(v any) (time.Time, bool) {
	switch value := v.(type) {
	case float64:
		seconds, fractional := math.Modf(value)
		nanos := int64(fractional * float64(time.Second))
		return time.Unix(int64(seconds), nanos).UTC(), true
	case int64:
		return time.Unix(value, 0).UTC(), true
	case int:
		return time.Unix(int64(value), 0).UTC(), true
	case string:
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return time.Time{}, false
		}
		if numeric, err := strconv.ParseFloat(trimmed, 64); err == nil {
			seconds, fractional := math.Modf(numeric)
			nanos := int64(fractional * float64(time.Second))
			return time.Unix(int64(seconds), nanos).UTC(), true
		}
		if ts, err := time.Parse(time.RFC3339, trimmed); err == nil {
			return ts.UTC(), true
		}
		return time.Time{}, false
	default:
		return time.Time{}, false
	}
}

func lightsailFloat64(v any) (float64, bool) {
	switch value := v.(type) {
	case float64:
		return value, true
	case int:
		return float64(value), true
	case int32:
		return float64(value), true
	default:
		return 0, false
	}
}

func lightsailAddOnRequest(v any) (addOnType, snapshotTimeOfDay string, ok bool) {
	obj, ok := v.(map[string]any)
	if !ok {
		return "", "", false
	}
	addOnType = lightsailString(obj["addOnType"])
	if addOnType == "" {
		return "", "", false
	}
	if autoSnapshotReq, ok := obj["autoSnapshotAddOnRequest"].(map[string]any); ok {
		snapshotTimeOfDay = lightsailString(autoSnapshotReq["snapshotTimeOfDay"])
	}
	return addOnType, snapshotTimeOfDay, true
}

func lightsailDomainEntry(v any) (lightsail.DomainEntry, bool) {
	obj, ok := v.(map[string]any)
	if !ok {
		return lightsail.DomainEntry{}, false
	}
	entry := lightsail.DomainEntry{
		ID:      lightsailString(obj["id"]),
		IsAlias: false,
		Name:    lightsailString(obj["name"]),
		Options: lightsailStringMap(obj["options"]),
		Target:  lightsailString(obj["target"]),
		Type:    lightsailString(obj["type"]),
	}
	if alias, ok := lightsailBool(obj["isAlias"]); ok {
		entry.IsAlias = alias
	}
	if entry.Name == "" || entry.Type == "" {
		return lightsail.DomainEntry{}, false
	}
	return entry, true
}

func lightsailStringMap(v any) map[string]string {
	obj, ok := v.(map[string]any)
	if !ok {
		return map[string]string{}
	}
	out := make(map[string]string, len(obj))
	for key, raw := range obj {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if value, ok := raw.(string); ok {
			out[key] = strings.TrimSpace(value)
		}
	}
	return out
}

func lightsailInstanceEntries(v any) ([]lightsail.InstanceEntry, bool) {
	items, ok := v.([]any)
	if !ok {
		return nil, false
	}
	out := make([]lightsail.InstanceEntry, 0, len(items))
	for _, item := range items {
		obj, ok := item.(map[string]any)
		if !ok {
			return nil, false
		}
		entry := lightsail.InstanceEntry{
			AvailabilityZone: lightsailString(obj["availabilityZone"]),
			InstanceType:     lightsailString(obj["instanceType"]),
			PortInfoSource:   lightsailString(obj["portInfoSource"]),
			SourceName:       lightsailString(obj["sourceName"]),
			UserData:         lightsailString(obj["userData"]),
		}
		if entry.AvailabilityZone == "" || entry.InstanceType == "" || entry.PortInfoSource == "" || entry.SourceName == "" {
			return nil, false
		}
		out = append(out, entry)
	}
	return out, true
}

func lightsailRelationalDatabaseParameters(v any) ([]lightsail.RelationalDatabaseParameter, bool) {
	items, ok := v.([]any)
	if !ok {
		return nil, false
	}
	out := make([]lightsail.RelationalDatabaseParameter, 0, len(items))
	for _, item := range items {
		obj, ok := item.(map[string]any)
		if !ok {
			return nil, false
		}
		parameter := lightsail.RelationalDatabaseParameter{
			AllowedValues:  strings.TrimSpace(lightsailString(obj["allowedValues"])),
			ApplyMethod:    strings.TrimSpace(lightsailString(obj["applyMethod"])),
			ApplyType:      strings.TrimSpace(lightsailString(obj["applyType"])),
			DataType:       strings.TrimSpace(lightsailString(obj["dataType"])),
			Description:    strings.TrimSpace(lightsailString(obj["description"])),
			IsModifiable:   false,
			ParameterName:  strings.TrimSpace(lightsailString(obj["parameterName"])),
			ParameterValue: strings.TrimSpace(lightsailString(obj["parameterValue"])),
		}
		if value, ok := lightsailBool(obj["isModifiable"]); ok {
			parameter.IsModifiable = value
		}
		if parameter.ParameterName == "" {
			return nil, false
		}
		out = append(out, parameter)
	}
	return out, true
}

func lightsailCopyStringMap(v map[string]string) map[string]string {
	if len(v) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(v))
	for key, value := range v {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		out[key] = value
	}
	return out
}

func lightsailContainerDefinitions(v any) map[string]lightsail.ContainerServiceContainer {
	obj, ok := v.(map[string]any)
	if !ok {
		return map[string]lightsail.ContainerServiceContainer{}
	}
	out := make(map[string]lightsail.ContainerServiceContainer, len(obj))
	for key, raw := range obj {
		containerName := strings.TrimSpace(key)
		if containerName == "" {
			continue
		}
		container, ok := lightsailContainerDefinition(raw)
		if !ok {
			continue
		}
		out[containerName] = container
	}
	return out
}

func lightsailContainerDefinition(v any) (lightsail.ContainerServiceContainer, bool) {
	obj, ok := v.(map[string]any)
	if !ok {
		return lightsail.ContainerServiceContainer{}, false
	}
	return lightsail.ContainerServiceContainer{
		Command:     lightsailStringSlice(obj["command"]),
		Environment: lightsailStringMap(obj["environment"]),
		Image:       lightsailString(obj["image"]),
		Ports:       lightsailStringMap(obj["ports"]),
	}, true
}

func lightsailEndpointRequest(v any) (*lightsail.ContainerServiceEndpoint, bool) {
	if v == nil {
		return nil, true
	}
	obj, ok := v.(map[string]any)
	if !ok {
		return nil, false
	}
	containerName := lightsailString(obj["containerName"])
	containerPort, hasContainerPort := lightsailInt32(obj["containerPort"])
	if containerName == "" || !hasContainerPort {
		return nil, false
	}
	var healthCheck *lightsail.ContainerServiceHealthCheckConfig
	if raw, exists := obj["healthCheck"]; exists && raw != nil {
		parsed, ok := lightsailContainerServiceHealthCheckConfig(raw)
		if !ok {
			return nil, false
		}
		healthCheck = parsed
	}
	return &lightsail.ContainerServiceEndpoint{
		ContainerName: containerName,
		ContainerPort: containerPort,
		HealthCheck:   healthCheck,
	}, true
}

func lightsailContainerServiceHealthCheckConfig(v any) (*lightsail.ContainerServiceHealthCheckConfig, bool) {
	obj, ok := v.(map[string]any)
	if !ok {
		return nil, false
	}

	out := &lightsail.ContainerServiceHealthCheckConfig{}
	if value, exists := obj["healthyThreshold"]; exists {
		parsed, err := lightsailOptionalInt32(value)
		if err != nil {
			return nil, false
		}
		out.HealthyThreshold = parsed
	}
	if value, exists := obj["intervalSeconds"]; exists {
		parsed, err := lightsailOptionalInt32(value)
		if err != nil {
			return nil, false
		}
		out.IntervalSeconds = parsed
	}
	if value, exists := obj["path"]; exists {
		strValue, ok := value.(string)
		if !ok {
			return nil, false
		}
		out.Path = &strValue
	}
	if value, exists := obj["successCodes"]; exists {
		strValue, ok := value.(string)
		if !ok {
			return nil, false
		}
		out.SuccessCodes = &strValue
	}
	if value, exists := obj["timeoutSeconds"]; exists {
		parsed, err := lightsailOptionalInt32(value)
		if err != nil {
			return nil, false
		}
		out.TimeoutSeconds = parsed
	}
	if value, exists := obj["unhealthyThreshold"]; exists {
		parsed, err := lightsailOptionalInt32(value)
		if err != nil {
			return nil, false
		}
		out.UnhealthyThreshold = parsed
	}
	return out, true
}

func lightsailDistributionCacheBehavior(v any) (lightsail.DistributionCacheBehavior, bool) {
	obj, ok := v.(map[string]any)
	if !ok {
		return lightsail.DistributionCacheBehavior{}, false
	}
	behavior := lightsailString(obj["behavior"])
	if behavior == "" {
		return lightsail.DistributionCacheBehavior{}, false
	}
	return lightsail.DistributionCacheBehavior{Behavior: behavior}, true
}

func lightsailDistributionOrigin(v any) (lightsail.DistributionOrigin, bool) {
	obj, ok := v.(map[string]any)
	if !ok {
		return lightsail.DistributionOrigin{}, false
	}
	name := lightsailString(obj["name"])
	if name == "" {
		return lightsail.DistributionOrigin{}, false
	}
	responseTimeout, _ := lightsailInt32(obj["responseTimeout"])
	return lightsail.DistributionOrigin{
		Name:            name,
		ProtocolPolicy:  lightsailString(obj["protocolPolicy"]),
		RegionName:      lightsailString(obj["regionName"]),
		ResourceType:    lightsailString(obj["resourceType"]),
		ResponseTimeout: responseTimeout,
	}, true
}

func lightsailDistributionCacheSettings(v any) lightsail.DistributionCacheSettings {
	obj, ok := v.(map[string]any)
	if !ok {
		return lightsail.DistributionCacheSettings{}
	}
	defaultTTL, _ := lightsailInt64(obj["defaultTTL"])
	maximumTTL, _ := lightsailInt64(obj["maximumTTL"])
	minimumTTL, _ := lightsailInt64(obj["minimumTTL"])
	return lightsail.DistributionCacheSettings{
		AllowedHTTPMethods:    lightsailString(obj["allowedHTTPMethods"]),
		CachedHTTPMethods:     lightsailString(obj["cachedHTTPMethods"]),
		DefaultTTL:            defaultTTL,
		MaximumTTL:            maximumTTL,
		MinimumTTL:            minimumTTL,
		ForwardedCookies:      lightsailDistributionCookieObject(obj["forwardedCookies"]),
		ForwardedHeaders:      lightsailDistributionHeaderObject(obj["forwardedHeaders"]),
		ForwardedQueryStrings: lightsailDistributionQueryStringObject(obj["forwardedQueryStrings"]),
	}
}

func lightsailDistributionCookieObject(v any) lightsail.DistributionCookieObject {
	obj, ok := v.(map[string]any)
	if !ok {
		return lightsail.DistributionCookieObject{}
	}
	return lightsail.DistributionCookieObject{
		Option:           lightsailString(obj["option"]),
		CookiesAllowList: lightsailStringSlice(obj["cookiesAllowList"]),
	}
}

func lightsailDistributionHeaderObject(v any) lightsail.DistributionHeaderObject {
	obj, ok := v.(map[string]any)
	if !ok {
		return lightsail.DistributionHeaderObject{}
	}
	return lightsail.DistributionHeaderObject{
		Option:           lightsailString(obj["option"]),
		HeadersAllowList: lightsailStringSlice(obj["headersAllowList"]),
	}
}

func lightsailDistributionQueryStringObject(v any) lightsail.DistributionQueryStringObject {
	obj, ok := v.(map[string]any)
	if !ok {
		return lightsail.DistributionQueryStringObject{}
	}
	var option *bool
	if b, ok := lightsailBool(obj["option"]); ok {
		option = &b
	} else if s := strings.ToLower(lightsailString(obj["option"])); s != "" {
		switch s {
		case "true", "all":
			v := true
			option = &v
		case "false", "none":
			v := false
			option = &v
		}
	}
	return lightsail.DistributionQueryStringObject{
		Option:                option,
		QueryStringsAllowList: lightsailStringSlice(obj["queryStringsAllowList"]),
	}
}

func lightsailDistributionCacheBehaviors(v any) []lightsail.DistributionCacheBehaviorPerPath {
	items, ok := v.([]any)
	if !ok {
		return []lightsail.DistributionCacheBehaviorPerPath{}
	}
	out := make([]lightsail.DistributionCacheBehaviorPerPath, 0, len(items))
	for _, item := range items {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		behavior := lightsailString(entry["behavior"])
		path := lightsailString(entry["path"])
		if behavior == "" || path == "" {
			continue
		}
		out = append(out, lightsail.DistributionCacheBehaviorPerPath{
			Behavior: behavior,
			Path:     path,
		})
	}
	return out
}

func lightsailBucketAccessLogConfig(v any) (lightsail.BucketAccessLogConfig, bool) {
	obj, ok := v.(map[string]any)
	if !ok {
		return lightsail.BucketAccessLogConfig{}, false
	}
	enabled, ok := lightsailBool(obj["enabled"])
	if !ok {
		return lightsail.BucketAccessLogConfig{}, false
	}
	return lightsail.BucketAccessLogConfig{
		Enabled:     enabled,
		Destination: lightsailString(obj["destination"]),
		Prefix:      lightsailString(obj["prefix"]),
	}, true
}

func lightsailBucketAccessRules(v any) (lightsail.BucketAccessRules, bool) {
	obj, ok := v.(map[string]any)
	if !ok {
		return lightsail.BucketAccessRules{}, false
	}
	getObject := lightsailString(obj["getObject"])
	if getObject == "" {
		return lightsail.BucketAccessRules{}, false
	}
	allowPublicOverrides, _ := lightsailBool(obj["allowPublicOverrides"])
	return lightsail.BucketAccessRules{
		AllowPublicOverrides: allowPublicOverrides,
		GetObject:            getObject,
	}, true
}

func lightsailStringSliceMap(v any) map[string][]string {
	obj, ok := v.(map[string]any)
	if !ok {
		return map[string][]string{}
	}
	out := make(map[string][]string, len(obj))
	for key, value := range obj {
		trimmed := strings.TrimSpace(key)
		if trimmed == "" {
			continue
		}
		out[trimmed] = lightsailStringSlice(value)
	}
	return out
}

func lightsailTagsToMap(v any) map[string]string {
	items, ok := v.([]any)
	if !ok {
		return map[string]string{}
	}
	out := map[string]string{}
	for _, item := range items {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		key := firstNonEmpty(lightsailString(entry["key"]), lightsailString(entry["Key"]))
		if key == "" {
			continue
		}
		out[key] = firstNonEmpty(lightsailString(entry["value"]), lightsailString(entry["Value"]))
	}
	return out
}

func lightsailMapToTags(v map[string]string) []map[string]string {
	if len(v) == 0 {
		return []map[string]string{}
	}
	keys := make([]string, 0, len(v))
	for key := range v {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]map[string]string, 0, len(keys))
	for _, key := range keys {
		out = append(out, map[string]string{
			"key":   key,
			"value": v[key],
		})
	}
	return out
}

func lightsailOperationsPayload(ops []lightsail.Operation) []map[string]any {
	out := make([]map[string]any, 0, len(ops))
	for _, op := range ops {
		out = append(out, lightsailOperationPayload(op))
	}
	return out
}

func lightsailOperationPayload(op lightsail.Operation) map[string]any {
	return map[string]any{
		"id":               op.ID,
		"resourceName":     op.ResourceName,
		"resourceType":     op.ResourceType,
		"createdAt":        lightsailTimestamp(op.CreatedAt),
		"location":         lightsailLocationPayload(op.AvailabilityZone, op.Region),
		"operationDetails": op.Details,
		"operationType":    op.OperationType,
		"status":           op.Status,
		"statusChangedAt":  lightsailTimestamp(op.StatusChangedAt),
		"isTerminal":       op.IsTerminal,
	}
}

func lightsailBlueprintPayload(in lightsail.Blueprint) map[string]any {
	return map[string]any{
		"appCategory": in.AppCategory,
		"blueprintId": in.BlueprintID,
		"description": in.Description,
		"group":       in.Group,
		"isActive":    in.IsActive,
		"licenseUrl":  in.LicenseURL,
		"minPower":    in.MinPower,
		"name":        in.Name,
		"platform":    in.Platform,
		"productUrl":  in.ProductURL,
		"type":        in.Type,
		"version":     in.Version,
		"versionCode": in.VersionCode,
	}
}

func lightsailBundlePayload(in lightsail.Bundle) map[string]any {
	return map[string]any{
		"appCategory":            in.AppCategory,
		"bundleId":               in.BundleID,
		"cpuCount":               in.CPUCount,
		"diskSizeInGb":           in.DiskSizeInGb,
		"instanceType":           in.InstanceType,
		"isActive":               in.IsActive,
		"name":                   in.Name,
		"power":                  in.Power,
		"price":                  in.Price,
		"publicIpv4AddressCount": in.PublicIpv4AddressCount,
		"ramSizeInGb":            in.RAMSizeInGb,
		"supportedAppCategories": in.SupportedAppCategories,
		"supportedPlatforms":     in.SupportedPlatforms,
		"transferPerMonthInGb":   in.TransferPerMonthInGb,
	}
}

func lightsailSetupHistoryPayload(in []lightsail.SetupHistory) []map[string]any {
	if len(in) == 0 {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(in))
	for _, item := range in {
		payload := map[string]any{
			"executionDetails": lightsailSetupExecutionDetailsPayload(item.ExecutionDetails),
			"operationId":      item.OperationID,
			"resource":         lightsailSetupHistoryResourcePayload(item.Resource),
			"status":           item.Status,
		}
		if item.Request != nil {
			payload["request"] = lightsailSetupRequestPayload(*item.Request)
		}
		out = append(out, payload)
	}
	return out
}

func lightsailSetupExecutionDetailsPayload(in []lightsail.SetupExecutionDetail) []map[string]any {
	if len(in) == 0 {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(in))
	for _, item := range in {
		out = append(out, map[string]any{
			"command":        item.Command,
			"dateTime":       lightsailTimestamp(item.DateTime),
			"name":           item.Name,
			"standardError":  item.StandardError,
			"standardOutput": item.StandardOutput,
			"status":         item.Status,
			"version":        item.Version,
		})
	}
	return out
}

func lightsailSetupRequestPayload(in lightsail.SetupRequest) map[string]any {
	return map[string]any{
		"certificateProvider": in.CertificateProvider,
		"domainNames":         in.DomainNames,
		"instanceName":        in.InstanceName,
	}
}

func lightsailSetupHistoryResourcePayload(in lightsail.SetupHistoryResource) map[string]any {
	return map[string]any{
		"arn":          in.ARN,
		"createdAt":    lightsailTimestamp(in.CreatedAt),
		"location":     lightsailLocationPayload(in.AvailabilityZone, in.Region),
		"name":         in.Name,
		"resourceType": in.ResourceType,
	}
}

func lightsailResourceBudgetEstimatesPayload(in []lightsail.ResourceBudgetEstimate) []map[string]any {
	if len(in) == 0 {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(in))
	for _, item := range in {
		out = append(out, map[string]any{
			"costEstimates": lightsailCostEstimatesPayload(item.CostEstimates),
			"endTime":       lightsailTimestamp(item.EndTime),
			"resourceName":  item.ResourceName,
			"resourceType":  item.ResourceType,
			"startTime":     lightsailTimestamp(item.StartTime),
		})
	}
	return out
}

func lightsailCostEstimatesPayload(in []lightsail.CostEstimate) []map[string]any {
	if len(in) == 0 {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(in))
	for _, item := range in {
		out = append(out, map[string]any{
			"resultsByTime": lightsailEstimateByTimePayload(item.ResultsByTime),
			"usageType":     item.UsageType,
		})
	}
	return out
}

func lightsailEstimateByTimePayload(in []lightsail.EstimateByTime) []map[string]any {
	if len(in) == 0 {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(in))
	for _, item := range in {
		out = append(out, map[string]any{
			"currency":    item.Currency,
			"pricingUnit": item.PricingUnit,
			"timePeriod": map[string]any{
				"start": lightsailTimestamp(item.StartTime),
				"end":   lightsailTimestamp(item.EndTime),
			},
			"unit":      item.Unit,
			"usageCost": item.UsageCost,
		})
	}
	return out
}

func lightsailCloudFormationStackRecordsPayload(in []lightsail.CloudFormationStackRecord) []map[string]any {
	if len(in) == 0 {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(in))
	for _, item := range in {
		out = append(out, map[string]any{
			"arn":       item.ARN,
			"createdAt": lightsailTimestamp(item.CreatedAt),
			"destinationInfo": map[string]any{
				"id":      item.DestinationInfo.ID,
				"service": item.DestinationInfo.Service,
			},
			"location":     lightsailLocationPayload(item.AvailabilityZone, item.Region),
			"name":         item.Name,
			"resourceType": item.ResourceType,
			"sourceInfo":   lightsailCloudFormationSourceInfoPayload(item.SourceInfo),
			"state":        item.State,
		})
	}
	return out
}

func lightsailCloudFormationSourceInfoPayload(in []lightsail.CloudFormationStackSourceInfo) []map[string]any {
	if len(in) == 0 {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(in))
	for _, item := range in {
		out = append(out, map[string]any{
			"arn":          item.ARN,
			"name":         item.Name,
			"resourceType": item.ResourceType,
		})
	}
	return out
}

func lightsailSessionsPayload(in []lightsail.GUISession) []map[string]any {
	if len(in) == 0 {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(in))
	for _, item := range in {
		out = append(out, map[string]any{
			"isPrimary": item.IsPrimary,
			"name":      item.Name,
			"url":       item.URL,
		})
	}
	return out
}

func lightsailInstanceMetricDatapointsPayload(in []lightsail.InstanceMetricDatapoint) []map[string]any {
	if len(in) == 0 {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(in))
	for _, item := range in {
		point := map[string]any{
			"timestamp": lightsailTimestamp(item.Timestamp),
			"unit":      item.Unit,
		}
		if item.Average != nil {
			point["average"] = *item.Average
		}
		if item.Maximum != nil {
			point["maximum"] = *item.Maximum
		}
		if item.Minimum != nil {
			point["minimum"] = *item.Minimum
		}
		if item.SampleCount != nil {
			point["sampleCount"] = *item.SampleCount
		}
		if item.Sum != nil {
			point["sum"] = *item.Sum
		}
		out = append(out, point)
	}
	return out
}

func lightsailInstancePayload(in lightsail.Instance) map[string]any {
	return map[string]any{
		"arn":              in.ARN,
		"blueprintId":      in.BlueprintID,
		"bundleId":         in.BundleID,
		"createdAt":        lightsailTimestamp(in.CreatedAt),
		"ipv6Addresses":    append([]string(nil), in.IPv6Addresses...),
		"isStaticIp":       in.IsStaticIP,
		"location":         lightsailLocationPayload(in.AvailabilityZone, in.Region),
		"metadataOptions":  lightsailInstanceMetadataOptionsPayload(in.MetadataOptions),
		"name":             in.Name,
		"networking":       map[string]any{"ports": lightsailPortStatesPayload(in.PortStates)},
		"privateIpAddress": in.PrivateIPAddress,
		"publicIpAddress":  in.PublicIPAddress,
		"resourceType":     "Instance",
		"state": map[string]any{
			"code": in.StateCode,
			"name": in.StateName,
		},
		"tags":     lightsailMapToTags(in.Tags),
		"username": in.Username,
	}
}

func lightsailPortStatesPayload(in []lightsail.InstancePortState) []map[string]any {
	out := make([]map[string]any, 0, len(in))
	for _, item := range in {
		out = append(out, map[string]any{
			"fromPort":        item.FromPort,
			"toPort":          item.ToPort,
			"protocol":        item.Protocol,
			"cidrs":           append([]string(nil), item.Cidrs...),
			"ipv6Cidrs":       append([]string(nil), item.Ipv6Cidrs...),
			"cidrListAliases": append([]string(nil), item.CidrListAliases...),
			"state":           item.State,
		})
	}
	return out
}

func lightsailInstanceMetadataOptionsPayload(in lightsail.InstanceMetadataOptions) map[string]any {
	return map[string]any{
		"httpEndpoint":            in.HttpEndpoint,
		"httpProtocolIpv6":        in.HttpProtocolIpv6,
		"httpPutResponseHopLimit": in.HttpPutResponseHopLimit,
		"httpTokens":              in.HttpTokens,
		"state":                   in.State,
	}
}

func lightsailInstanceAccessDetailsPayload(in lightsail.InstanceAccessDetails) map[string]any {
	return map[string]any{
		"certKey":       in.CertKey,
		"expiresAt":     lightsailTimestamp(in.ExpiresAt),
		"hostKeys":      lightsailHostKeysPayload(in.HostKeys),
		"instanceName":  in.InstanceName,
		"ipAddress":     in.IpAddress,
		"ipv6Addresses": append([]string(nil), in.Ipv6Addresses...),
		"password":      in.Password,
		"privateKey":    in.PrivateKey,
		"protocol":      in.Protocol,
		"username":      in.Username,
	}
}

func lightsailHostKeysPayload(in []lightsail.HostKeyAttributes) []map[string]any {
	out := make([]map[string]any, 0, len(in))
	for _, item := range in {
		out = append(out, map[string]any{
			"algorithm":         item.Algorithm,
			"fingerprintSHA1":   item.FingerprintSHA1,
			"fingerprintSHA256": item.FingerprintSHA256,
			"notValidAfter":     lightsailTimestamp(item.NotValidAfter),
			"notValidBefore":    lightsailTimestamp(item.NotValidBefore),
			"publicKey":         item.PublicKey,
			"witnessedAt":       lightsailTimestamp(item.WitnessedAt),
		})
	}
	return out
}

func lightsailInstanceSnapshotPayload(in lightsail.InstanceSnapshot) map[string]any {
	return map[string]any{
		"arn":              in.ARN,
		"createdAt":        lightsailTimestamp(in.CreatedAt),
		"fromBlueprintId":  in.FromBlueprintID,
		"fromBundleId":     in.FromBundleID,
		"fromInstanceArn":  in.FromInstanceARN,
		"fromInstanceName": in.FromInstanceName,
		"location":         lightsailLocationPayload(in.AvailabilityZone, in.Region),
		"name":             in.Name,
		"resourceType":     "InstanceSnapshot",
		"state":            in.State,
		"tags":             lightsailMapToTags(in.Tags),
	}
}

func lightsailStaticIpPayload(in lightsail.StaticIP) map[string]any {
	return map[string]any{
		"arn":          in.ARN,
		"attachedTo":   in.AttachedTo,
		"createdAt":    lightsailTimestamp(in.CreatedAt),
		"ipAddress":    in.IPAddress,
		"isAttached":   strings.TrimSpace(in.AttachedTo) != "",
		"location":     lightsailLocationPayload(in.AvailabilityZone, in.Region),
		"name":         in.Name,
		"resourceType": "StaticIp",
	}
}

func lightsailDiskPayload(in lightsail.Disk) map[string]any {
	return map[string]any{
		"arn":             in.ARN,
		"attachedTo":      in.AttachedTo,
		"autoMountStatus": in.AutoMountStatus,
		"createdAt":       lightsailTimestamp(in.CreatedAt),
		"iops":            in.Iops,
		"isAttached":      in.IsAttached,
		"isSystemDisk":    in.IsSystemDisk,
		"location":        lightsailLocationPayload(in.AvailabilityZone, in.Region),
		"name":            in.Name,
		"path":            in.Path,
		"resourceType":    "Disk",
		"sizeInGb":        in.SizeInGb,
		"state":           in.State,
		"tags":            lightsailMapToTags(in.Tags),
	}
}

func lightsailDiskSnapshotPayload(in lightsail.DiskSnapshot) map[string]any {
	return map[string]any{
		"arn":                in.ARN,
		"createdAt":          lightsailTimestamp(in.CreatedAt),
		"fromDiskArn":        in.FromDiskARN,
		"fromDiskName":       in.FromDiskName,
		"fromInstanceArn":    in.FromInstanceARN,
		"fromInstanceName":   in.FromInstanceName,
		"isFromAutoSnapshot": in.IsFromAutoSnapshot,
		"location":           lightsailLocationPayload(in.AvailabilityZone, in.Region),
		"name":               in.Name,
		"progress":           in.Progress,
		"resourceType":       "DiskSnapshot",
		"sizeInGb":           in.SizeInGb,
		"state":              in.State,
		"supportCode":        in.SupportCode,
		"tags":               lightsailMapToTags(in.Tags),
	}
}

func lightsailExportSnapshotRecordPayload(in lightsail.ExportSnapshotRecord) map[string]any {
	sourceInfo := map[string]any{
		"arn":              in.SourceSnapshotARN,
		"createdAt":        lightsailTimestamp(in.SourceCreatedAt),
		"fromResourceArn":  in.SourceResourceARN,
		"fromResourceName": in.SourceResourceName,
		"name":             in.SourceSnapshotName,
		"resourceType":     in.SourceType,
	}
	if in.SourceDiskSizeInGb > 0 {
		sourceInfo["diskSnapshotInfo"] = map[string]any{"sizeInGb": in.SourceDiskSizeInGb}
	}
	return map[string]any{
		"arn":       in.ARN,
		"createdAt": lightsailTimestamp(in.CreatedAt),
		"destinationInfo": map[string]any{
			"id":      in.DestinationID,
			"service": in.DestinationService,
		},
		"location":     lightsailLocationPayload(in.AvailabilityZone, in.Region),
		"name":         in.Name,
		"resourceType": "ExportSnapshotRecord",
		"sourceInfo":   sourceInfo,
		"state":        in.State,
	}
}

func lightsailLoadBalancerPayload(in lightsail.LoadBalancer) map[string]any {
	configurationOptions := map[string]string{}
	for key, value := range in.ConfigurationOptions {
		configurationOptions[key] = value
	}
	return map[string]any{
		"arn":                     in.ARN,
		"configurationOptions":    configurationOptions,
		"createdAt":               lightsailTimestamp(in.CreatedAt),
		"dnsName":                 in.DNSName,
		"healthCheckPath":         in.HealthCheckPath,
		"httpsRedirectionEnabled": in.HTTPSRedirectionEnabled,
		"instanceHealthSummary":   lightsailLoadBalancerInstanceHealthSummariesPayload(in.InstanceHealthSummary),
		"instancePort":            in.InstancePort,
		"ipAddressType":           in.IPAddressType,
		"location":                lightsailLocationPayload(in.AvailabilityZone, in.Region),
		"name":                    in.Name,
		"protocol":                in.Protocol,
		"publicPorts":             append([]int32(nil), in.PublicPorts...),
		"resourceType":            in.ResourceType,
		"state":                   in.State,
		"supportCode":             in.SupportCode,
		"tags":                    lightsailMapToTags(in.Tags),
		"tlsCertificateSummaries": lightsailLoadBalancerTLSCertificateSummariesPayload(in.TLSCertificateSummaries),
		"tlsPolicyName":           in.TLSPolicyName,
	}
}

func lightsailLoadBalancerInstanceHealthSummariesPayload(in []lightsail.LoadBalancerInstanceHealthSummary) []map[string]any {
	out := make([]map[string]any, 0, len(in))
	for _, item := range in {
		entry := map[string]any{
			"instanceHealth": item.InstanceHealth,
			"instanceName":   item.InstanceName,
		}
		if item.InstanceHealthReason != "" {
			entry["instanceHealthReason"] = item.InstanceHealthReason
		}
		out = append(out, entry)
	}
	return out
}

func lightsailLoadBalancerTLSCertificateSummariesPayload(in []lightsail.LoadBalancerTLSCertificateSummary) []map[string]any {
	out := make([]map[string]any, 0, len(in))
	for _, item := range in {
		out = append(out, map[string]any{
			"isAttached": item.IsAttached,
			"name":       item.Name,
		})
	}
	return out
}

func lightsailLoadBalancerMetricDatapointsPayload(in []lightsail.LoadBalancerMetricDatapoint) []map[string]any {
	out := make([]map[string]any, 0, len(in))
	for _, item := range in {
		point := map[string]any{
			"timestamp": lightsailTimestamp(item.Timestamp),
			"unit":      item.Unit,
		}
		if item.Average != nil {
			point["average"] = *item.Average
		}
		if item.Maximum != nil {
			point["maximum"] = *item.Maximum
		}
		if item.Minimum != nil {
			point["minimum"] = *item.Minimum
		}
		if item.SampleCount != nil {
			point["sampleCount"] = *item.SampleCount
		}
		if item.Sum != nil {
			point["sum"] = *item.Sum
		}
		out = append(out, point)
	}
	return out
}

func lightsailLoadBalancerTLSCertificatePayload(in lightsail.LoadBalancerTLSCertificate) map[string]any {
	return map[string]any{
		"arn":                     in.ARN,
		"createdAt":               lightsailTimestamp(in.CreatedAt),
		"domainName":              in.DomainName,
		"isAttached":              in.IsAttached,
		"issuedAt":                lightsailTimestamp(in.IssuedAt),
		"issuer":                  in.Issuer,
		"keyAlgorithm":            in.KeyAlgorithm,
		"loadBalancerName":        in.LoadBalancerName,
		"location":                lightsailLocationPayload(in.AvailabilityZone, in.Region),
		"name":                    in.Name,
		"notAfter":                lightsailTimestamp(in.NotAfter),
		"notBefore":               lightsailTimestamp(in.NotBefore),
		"resourceType":            in.ResourceType,
		"status":                  in.Status,
		"subject":                 in.Subject,
		"subjectAlternativeNames": append([]string(nil), in.SubjectAlternativeNames...),
		"supportCode":             in.SupportCode,
		"tags":                    lightsailMapToTags(in.Tags),
	}
}

func lightsailLoadBalancerTLSPolicyPayload(in lightsail.LoadBalancerTLSPolicy) map[string]any {
	return map[string]any{
		"name":        in.Name,
		"description": in.Description,
		"isDefault":   in.IsDefault,
		"ciphers":     append([]string(nil), in.Ciphers...),
		"protocols":   append([]string(nil), in.Protocols...),
	}
}

func lightsailCertificateSummaryPayload(in lightsail.Certificate, includeDetails bool) map[string]any {
	out := map[string]any{
		"certificateArn":  in.ARN,
		"certificateName": in.Name,
		"domainName":      in.DomainName,
		"tags":            lightsailMapToTags(in.Tags),
	}
	if includeDetails {
		out["certificateDetail"] = lightsailCertificateDetailPayload(in)
	}
	return out
}

func lightsailCertificateDetailPayload(in lightsail.Certificate) map[string]any {
	out := map[string]any{
		"arn":                     in.ARN,
		"createdAt":               lightsailTimestamp(in.CreatedAt),
		"domainName":              in.DomainName,
		"domainValidationRecords": lightsailCertificateDomainValidationRecordsPayload(in.DomainValidationRecords),
		"eligibleToRenew":         in.EligibleToRenew,
		"inUseResourceCount":      in.InUseResourceCount,
		"issuedAt":                lightsailTimestamp(in.IssuedAt),
		"issuerCA":                in.IssuerCA,
		"keyAlgorithm":            in.KeyAlgorithm,
		"name":                    in.Name,
		"notAfter":                lightsailTimestamp(in.NotAfter),
		"notBefore":               lightsailTimestamp(in.NotBefore),
		"serialNumber":            in.SerialNumber,
		"status":                  in.Status,
		"subjectAlternativeNames": append([]string(nil), in.SubjectAlternativeNames...),
		"supportCode":             in.SupportCode,
		"tags":                    lightsailMapToTags(in.Tags),
	}
	if strings.TrimSpace(in.RequestFailureReason) != "" {
		out["requestFailureReason"] = in.RequestFailureReason
	}
	if strings.TrimSpace(in.RevocationReason) != "" {
		out["revocationReason"] = in.RevocationReason
	}
	if !in.RevokedAt.IsZero() {
		out["revokedAt"] = lightsailTimestamp(in.RevokedAt)
	}
	if !in.RenewalSummary.UpdatedAt.IsZero() || strings.TrimSpace(in.RenewalSummary.RenewalStatus) != "" || strings.TrimSpace(in.RenewalSummary.RenewalStatusReason) != "" || len(in.RenewalSummary.DomainValidationRecords) > 0 {
		out["renewalSummary"] = lightsailCertificateRenewalSummaryPayload(in.RenewalSummary)
	}
	return out
}

func lightsailCertificateDomainValidationRecordsPayload(in []lightsail.CertificateDomainValidationRecord) []map[string]any {
	out := make([]map[string]any, 0, len(in))
	for _, item := range in {
		entry := map[string]any{
			"domainName":       item.DomainName,
			"validationStatus": item.ValidationStatus,
		}
		if strings.TrimSpace(item.DNSRecordCreationCode) != "" || strings.TrimSpace(item.DNSRecordCreationText) != "" {
			entry["dnsRecordCreationState"] = map[string]any{
				"code":    item.DNSRecordCreationCode,
				"message": item.DNSRecordCreationText,
			}
		}
		if strings.TrimSpace(item.ResourceRecordName) != "" || strings.TrimSpace(item.ResourceRecordType) != "" || strings.TrimSpace(item.ResourceRecordValue) != "" {
			entry["resourceRecord"] = map[string]any{
				"name":  item.ResourceRecordName,
				"type":  item.ResourceRecordType,
				"value": item.ResourceRecordValue,
			}
		}
		out = append(out, entry)
	}
	return out
}

func lightsailCertificateRenewalSummaryPayload(in lightsail.CertificateRenewalSummary) map[string]any {
	out := map[string]any{
		"domainValidationRecords": lightsailCertificateDomainValidationRecordsPayload(in.DomainValidationRecords),
		"renewalStatus":           in.RenewalStatus,
	}
	if strings.TrimSpace(in.RenewalStatusReason) != "" {
		out["renewalStatusReason"] = in.RenewalStatusReason
	}
	if !in.UpdatedAt.IsZero() {
		out["updatedAt"] = lightsailTimestamp(in.UpdatedAt)
	}
	return out
}

func lightsailDomainPayload(in lightsail.Domain) map[string]any {
	return map[string]any{
		"arn":           in.ARN,
		"createdAt":     lightsailTimestamp(in.CreatedAt),
		"domainEntries": lightsailDomainEntriesPayload(in.DomainEntries),
		"location":      lightsailLocationPayload(in.AvailabilityZone, in.Region),
		"name":          in.Name,
		"resourceType":  firstNonEmpty(in.ResourceType, "Domain"),
		"supportCode":   in.SupportCode,
		"tags":          lightsailMapToTags(in.Tags),
	}
}

func lightsailDomainEntriesPayload(in []lightsail.DomainEntry) []map[string]any {
	out := make([]map[string]any, 0, len(in))
	for _, entry := range in {
		out = append(out, lightsailDomainEntryPayload(entry))
	}
	return out
}

func lightsailDomainEntryPayload(in lightsail.DomainEntry) map[string]any {
	return map[string]any{
		"id":      in.ID,
		"isAlias": in.IsAlias,
		"name":    in.Name,
		"options": in.Options,
		"target":  in.Target,
		"type":    in.Type,
	}
}

func lightsailDistributionPayload(in lightsail.Distribution) map[string]any {
	return map[string]any{
		"ableToUpdateBundle":              in.AbleToUpdateBundle,
		"alternativeDomainNames":          append([]string(nil), in.AlternativeDomainNames...),
		"arn":                             in.ARN,
		"bundleId":                        in.BundleID,
		"cacheBehaviorSettings":           lightsailDistributionCacheSettingsPayload(in.CacheBehaviorSettings),
		"cacheBehaviors":                  lightsailDistributionCacheBehaviorsPayload(in.CacheBehaviors),
		"certificateName":                 in.CertificateName,
		"createdAt":                       lightsailTimestamp(in.CreatedAt),
		"defaultCacheBehavior":            lightsailDistributionCacheBehaviorPayload(in.DefaultCacheBehavior),
		"domainName":                      in.DomainName,
		"ipAddressType":                   in.IPAddressType,
		"isEnabled":                       in.IsEnabled,
		"location":                        lightsailLocationPayload(in.AvailabilityZone, in.Region),
		"name":                            in.Name,
		"origin":                          lightsailDistributionOriginPayload(in.Origin),
		"originPublicDNS":                 in.OriginPublicDNS,
		"resourceType":                    firstNonEmpty(in.ResourceType, "Distribution"),
		"status":                          in.Status,
		"supportCode":                     in.SupportCode,
		"tags":                            lightsailMapToTags(in.Tags),
		"viewerMinimumTlsProtocolVersion": in.ViewerMinimumTLSProtocolVersion,
	}
}

func lightsailDistributionCacheBehaviorPayload(in lightsail.DistributionCacheBehavior) map[string]any {
	return map[string]any{
		"behavior": in.Behavior,
	}
}

func lightsailDistributionOriginPayload(in lightsail.DistributionOrigin) map[string]any {
	return map[string]any{
		"name":            in.Name,
		"protocolPolicy":  in.ProtocolPolicy,
		"regionName":      in.RegionName,
		"resourceType":    in.ResourceType,
		"responseTimeout": in.ResponseTimeout,
	}
}

func lightsailDistributionCacheSettingsPayload(in lightsail.DistributionCacheSettings) map[string]any {
	return map[string]any{
		"allowedHTTPMethods": in.AllowedHTTPMethods,
		"cachedHTTPMethods":  in.CachedHTTPMethods,
		"defaultTTL":         in.DefaultTTL,
		"forwardedCookies": map[string]any{
			"option":           in.ForwardedCookies.Option,
			"cookiesAllowList": append([]string(nil), in.ForwardedCookies.CookiesAllowList...),
		},
		"forwardedHeaders": map[string]any{
			"option":           in.ForwardedHeaders.Option,
			"headersAllowList": append([]string(nil), in.ForwardedHeaders.HeadersAllowList...),
		},
		"forwardedQueryStrings": map[string]any{
			"option":                boolValueOrDefault(in.ForwardedQueryStrings.Option, false),
			"queryStringsAllowList": append([]string(nil), in.ForwardedQueryStrings.QueryStringsAllowList...),
		},
		"maximumTTL": in.MaximumTTL,
		"minimumTTL": in.MinimumTTL,
	}
}

func lightsailDistributionCacheBehaviorsPayload(in []lightsail.DistributionCacheBehaviorPerPath) []map[string]any {
	out := make([]map[string]any, 0, len(in))
	for _, item := range in {
		out = append(out, map[string]any{
			"behavior": item.Behavior,
			"path":     item.Path,
		})
	}
	return out
}

func lightsailDistributionBundlePayload(in lightsail.DistributionBundle) map[string]any {
	return map[string]any{
		"bundleId":             in.BundleID,
		"isActive":             in.IsActive,
		"name":                 in.Name,
		"price":                in.Price,
		"transferPerMonthInGb": in.TransferPerMonthInGb,
	}
}

func lightsailDistributionMetricDatapointsPayload(in []lightsail.DistributionMetricDatapoint) []map[string]any {
	out := make([]map[string]any, 0, len(in))
	for _, item := range in {
		point := map[string]any{
			"timestamp": lightsailTimestamp(item.Timestamp),
			"unit":      item.Unit,
		}
		if item.Average != nil {
			point["average"] = *item.Average
		}
		if item.Maximum != nil {
			point["maximum"] = *item.Maximum
		}
		if item.Minimum != nil {
			point["minimum"] = *item.Minimum
		}
		if item.SampleCount != nil {
			point["sampleCount"] = *item.SampleCount
		}
		if item.Sum != nil {
			point["sum"] = *item.Sum
		}
		out = append(out, point)
	}
	return out
}

func lightsailAlarmPayload(in lightsail.Alarm) map[string]any {
	return map[string]any{
		"arn":                in.ARN,
		"comparisonOperator": in.ComparisonOperator,
		"contactProtocols":   append([]string(nil), in.ContactProtocols...),
		"createdAt":          lightsailTimestamp(in.CreatedAt),
		"datapointsToAlarm":  in.DatapointsToAlarm,
		"evaluationPeriods":  in.EvaluationPeriods,
		"location":           lightsailLocationPayload(in.AvailabilityZone, in.Region),
		"metricName":         in.MetricName,
		"monitoredResourceInfo": map[string]any{
			"arn":          in.MonitoredResource.ARN,
			"name":         in.MonitoredResource.Name,
			"resourceType": in.MonitoredResource.ResourceType,
		},
		"name":                 in.Name,
		"notificationEnabled":  in.NotificationEnabled,
		"notificationTriggers": append([]string(nil), in.NotificationTriggers...),
		"period":               in.Period,
		"resourceType":         in.ResourceType,
		"state":                in.State,
		"statistic":            in.Statistic,
		"supportCode":          in.SupportCode,
		"threshold":            in.Threshold,
		"treatMissingData":     in.TreatMissingData,
		"unit":                 in.Unit,
	}
}

func lightsailAutoSnapshotPayload(in lightsail.AutoSnapshotDetails) map[string]any {
	out := make([]map[string]any, 0, len(in.FromAttachedDisks))
	for _, item := range in.FromAttachedDisks {
		out = append(out, map[string]any{
			"path":     item.Path,
			"sizeInGb": item.SizeInGb,
		})
	}
	return map[string]any{
		"createdAt":         lightsailTimestamp(in.CreatedAt),
		"date":              in.Date,
		"fromAttachedDisks": out,
		"status":            in.Status,
	}
}

func lightsailKeyPairPayload(in lightsail.KeyPair) map[string]any {
	return map[string]any{
		"arn":          in.ARN,
		"createdAt":    lightsailTimestamp(in.CreatedAt),
		"fingerprint":  in.Fingerprint,
		"location":     lightsailLocationPayload(in.AvailabilityZone, in.Region),
		"name":         in.Name,
		"resourceType": "KeyPair",
		"tags":         lightsailMapToTags(in.Tags),
	}
}

func lightsailBucketPayload(in lightsail.Bucket) map[string]any {
	out := map[string]any{
		"ableToUpdateBundle":       in.AbleToUpdateBundle,
		"arn":                      in.ARN,
		"bundleId":                 in.BundleID,
		"createdAt":                lightsailTimestamp(in.CreatedAt),
		"location":                 lightsailLocationPayload(in.AvailabilityZone, in.Region),
		"name":                     in.Name,
		"objectVersioning":         in.ObjectVersioning,
		"readonlyAccessAccounts":   append([]string(nil), in.ReadonlyAccessAccounts...),
		"resourceType":             firstNonEmpty(in.ResourceType, "Bucket"),
		"resourcesReceivingAccess": lightsailBucketResourcesReceivingAccessPayload(in.ResourcesReceivingAccess),
		"state": map[string]any{
			"code":    in.State.Code,
			"message": in.State.Message,
		},
		"supportCode": in.SupportCode,
		"tags":        lightsailMapToTags(in.Tags),
		"url":         in.URL,
	}
	if in.AccessRules != nil {
		out["accessRules"] = map[string]any{
			"allowPublicOverrides": in.AccessRules.AllowPublicOverrides,
			"getObject":            in.AccessRules.GetObject,
		}
	}
	if in.AccessLogConfig != nil {
		out["accessLogConfig"] = map[string]any{
			"destination": in.AccessLogConfig.Destination,
			"enabled":     in.AccessLogConfig.Enabled,
			"prefix":      in.AccessLogConfig.Prefix,
		}
	}
	return out
}

func lightsailBucketResourcesReceivingAccessPayload(in []lightsail.BucketResourceReceivingAccess) []map[string]any {
	out := make([]map[string]any, 0, len(in))
	for _, item := range in {
		out = append(out, map[string]any{
			"name":         item.Name,
			"resourceType": item.ResourceType,
		})
	}
	return out
}

func lightsailBucketBundlePayload(in lightsail.BucketBundle) map[string]any {
	return map[string]any{
		"bundleId":             in.BundleID,
		"isActive":             in.IsActive,
		"name":                 in.Name,
		"price":                in.Price,
		"storagePerMonthInGb":  in.StoragePerMonthInGb,
		"transferPerMonthInGb": in.TransferPerMonthInGb,
	}
}

func lightsailBucketMetricDatapointsPayload(in []lightsail.BucketMetricDatapoint) []map[string]any {
	out := make([]map[string]any, 0, len(in))
	for _, item := range in {
		point := map[string]any{
			"timestamp": lightsailTimestamp(item.Timestamp),
			"unit":      item.Unit,
		}
		if item.Average != nil {
			point["average"] = *item.Average
		}
		if item.Maximum != nil {
			point["maximum"] = *item.Maximum
		}
		if item.Minimum != nil {
			point["minimum"] = *item.Minimum
		}
		if item.SampleCount != nil {
			point["sampleCount"] = *item.SampleCount
		}
		if item.Sum != nil {
			point["sum"] = *item.Sum
		}
		out = append(out, point)
	}
	return out
}

func lightsailBucketAccessKeyPayload(in lightsail.BucketAccessKey, includeSecret bool) map[string]any {
	out := map[string]any{
		"accessKeyId": in.AccessKeyID,
		"createdAt":   lightsailTimestamp(in.CreatedAt),
		"status":      in.Status,
	}
	if in.LastUsed != nil {
		lastUsed := map[string]any{
			"region":      in.LastUsed.Region,
			"serviceName": in.LastUsed.ServiceName,
		}
		if in.LastUsed.LastUsedDate != nil {
			lastUsed["lastUsedDate"] = lightsailTimestamp(*in.LastUsed.LastUsedDate)
		}
		out["lastUsed"] = lastUsed
	}
	if includeSecret && strings.TrimSpace(in.SecretAccessKey) != "" {
		out["secretAccessKey"] = in.SecretAccessKey
	}
	return out
}

func lightsailContactMethodPayload(in lightsail.ContactMethod) map[string]any {
	return map[string]any{
		"arn":             in.ARN,
		"contactEndpoint": in.ContactEndpoint,
		"createdAt":       lightsailTimestamp(in.CreatedAt),
		"location":        lightsailLocationPayload(in.AvailabilityZone, in.Region),
		"name":            in.Name,
		"protocol":        in.Protocol,
		"resourceType":    firstNonEmpty(in.ResourceType, "ContactMethod"),
		"status":          in.Status,
		"supportCode":     in.SupportCode,
	}
}

func lightsailContainerServicePayload(in lightsail.ContainerService) map[string]any {
	return map[string]any{
		"arn":                  in.ARN,
		"containerServiceName": in.Name,
		"createdAt":            lightsailTimestamp(in.CreatedAt),
		"isDisabled":           in.IsDisabled,
		"location":             lightsailLocationPayload(in.AvailabilityZone, in.Region),
		"power":                in.Power,
		"powerId":              in.PowerID,
		"principalArn":         in.PrincipalARN,
		"privateDomainName":    in.PrivateDomainName,
		"publicDomainNames":    in.PublicDomainNames,
		"resourceType":         firstNonEmpty(in.ResourceType, "ContainerService"),
		"scale":                in.Scale,
		"state":                in.State,
		"tags":                 lightsailMapToTags(in.Tags),
		"url":                  in.URL,
	}
}

func lightsailContainerServiceDeploymentsPayload(in []lightsail.ContainerServiceDeployment) []map[string]any {
	out := make([]map[string]any, 0, len(in))
	for _, deployment := range in {
		out = append(out, lightsailContainerServiceDeploymentPayload(deployment))
	}
	return out
}

func lightsailContainerServiceDeploymentPayload(in lightsail.ContainerServiceDeployment) map[string]any {
	out := map[string]any{
		"containers": lightsailContainerServiceContainersPayload(in.Containers),
		"createdAt":  lightsailTimestamp(in.CreatedAt),
		"state":      in.State,
		"version":    in.Version,
	}
	if in.PublicEndpoint != nil {
		out["publicEndpoint"] = lightsailContainerServiceEndpointPayload(in.PublicEndpoint)
	}
	return out
}

func lightsailContainerServiceContainersPayload(in map[string]lightsail.ContainerServiceContainer) map[string]any {
	if len(in) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for name, container := range in {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			continue
		}
		out[trimmed] = lightsailContainerServiceContainerPayload(container)
	}
	return out
}

func lightsailContainerServiceContainerPayload(in lightsail.ContainerServiceContainer) map[string]any {
	return map[string]any{
		"command":     append([]string(nil), in.Command...),
		"environment": lightsailCopyStringMap(in.Environment),
		"image":       in.Image,
		"ports":       lightsailCopyStringMap(in.Ports),
	}
}

func lightsailContainerServiceEndpointPayload(in *lightsail.ContainerServiceEndpoint) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := map[string]any{
		"containerName": in.ContainerName,
		"containerPort": in.ContainerPort,
	}
	if in.HealthCheck != nil {
		out["healthCheck"] = lightsailContainerServiceHealthCheckPayload(in.HealthCheck)
	}
	return out
}

func lightsailContainerServiceHealthCheckPayload(in *lightsail.ContainerServiceHealthCheckConfig) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := map[string]any{}
	if in.HealthyThreshold != nil {
		out["healthyThreshold"] = *in.HealthyThreshold
	}
	if in.IntervalSeconds != nil {
		out["intervalSeconds"] = *in.IntervalSeconds
	}
	if in.Path != nil {
		out["path"] = *in.Path
	}
	if in.SuccessCodes != nil {
		out["successCodes"] = *in.SuccessCodes
	}
	if in.TimeoutSeconds != nil {
		out["timeoutSeconds"] = *in.TimeoutSeconds
	}
	if in.UnhealthyThreshold != nil {
		out["unhealthyThreshold"] = *in.UnhealthyThreshold
	}
	return out
}

func lightsailContainerServiceMetricDatapointsPayload(in []lightsail.ContainerServiceMetricDatapoint) []map[string]any {
	out := make([]map[string]any, 0, len(in))
	for _, item := range in {
		point := map[string]any{
			"timestamp": lightsailTimestamp(item.Timestamp),
			"unit":      item.Unit,
		}
		if item.Average != nil {
			point["average"] = *item.Average
		}
		if item.Maximum != nil {
			point["maximum"] = *item.Maximum
		}
		if item.Minimum != nil {
			point["minimum"] = *item.Minimum
		}
		if item.SampleCount != nil {
			point["sampleCount"] = *item.SampleCount
		}
		if item.Sum != nil {
			point["sum"] = *item.Sum
		}
		out = append(out, point)
	}
	return out
}

func lightsailContainerImagePayload(in lightsail.ContainerImage) map[string]any {
	return map[string]any{
		"createdAt": lightsailTimestamp(in.CreatedAt),
		"digest":    in.Digest,
		"image":     in.Image,
	}
}

func lightsailContainerImagesPayload(in []lightsail.ContainerImage) []map[string]any {
	out := make([]map[string]any, 0, len(in))
	for _, image := range in {
		out = append(out, lightsailContainerImagePayload(image))
	}
	return out
}

func lightsailContainerServiceLogEventsPayload(in []lightsail.ContainerServiceLogEvent) []map[string]any {
	out := make([]map[string]any, 0, len(in))
	for _, event := range in {
		out = append(out, map[string]any{
			"createdAt": lightsailTimestamp(event.CreatedAt),
			"message":   event.Message,
		})
	}
	return out
}

func lightsailContainerServicePowerPayload(in lightsail.ContainerServicePower) map[string]any {
	return map[string]any{
		"cpuCount":    in.CPUCount,
		"isActive":    in.IsActive,
		"name":        in.Name,
		"powerId":     in.PowerID,
		"price":       in.Price,
		"ramSizeInGb": in.RAMSizeInGb,
	}
}

func lightsailContainerServiceRegistryLoginPayload(in lightsail.ContainerServiceRegistryLogin) map[string]any {
	return map[string]any{
		"expiresAt": lightsailTimestamp(in.ExpiresAt),
		"password":  in.Password,
		"registry":  in.Registry,
		"username":  in.Username,
	}
}

func lightsailRelationalDatabaseBlueprintPayload(in lightsail.RelationalDatabaseBlueprint) map[string]any {
	return map[string]any{
		"blueprintId":              in.BlueprintID,
		"engine":                   in.Engine,
		"engineDescription":        in.EngineDescription,
		"engineVersion":            in.EngineVersion,
		"engineVersionDescription": in.EngineVersionDescription,
		"isEngineDefault":          in.IsEngineDefault,
	}
}

func lightsailRelationalDatabaseBundlePayload(in lightsail.RelationalDatabaseBundle) map[string]any {
	return map[string]any{
		"bundleId":             in.BundleID,
		"cpuCount":             in.CPUCount,
		"diskSizeInGb":         in.DiskSizeInGb,
		"isActive":             in.IsActive,
		"isEncrypted":          in.IsEncrypted,
		"name":                 in.Name,
		"price":                in.Price,
		"ramSizeInGb":          in.RAMSizeInGb,
		"transferPerMonthInGb": in.TransferPerMonthInGb,
	}
}

func lightsailRelationalDatabaseMetricDatapointsPayload(in []lightsail.RelationalDatabaseMetricDatapoint) []map[string]any {
	out := make([]map[string]any, 0, len(in))
	for _, item := range in {
		point := map[string]any{
			"timestamp": lightsailTimestamp(item.Timestamp),
			"unit":      item.Unit,
		}
		if item.Average != nil {
			point["average"] = *item.Average
		}
		if item.Maximum != nil {
			point["maximum"] = *item.Maximum
		}
		if item.Minimum != nil {
			point["minimum"] = *item.Minimum
		}
		if item.SampleCount != nil {
			point["sampleCount"] = *item.SampleCount
		}
		if item.Sum != nil {
			point["sum"] = *item.Sum
		}
		out = append(out, point)
	}
	return out
}

func lightsailRelationalDatabasePayload(in lightsail.RelationalDatabase) map[string]any {
	out := map[string]any{
		"arn":                           in.ARN,
		"backupRetentionEnabled":        in.BackupRetentionEnabled,
		"caCertificateIdentifier":       in.CACertificateIdentifier,
		"createdAt":                     lightsailTimestamp(in.CreatedAt),
		"engine":                        in.Engine,
		"engineVersion":                 in.EngineVersion,
		"hardware":                      lightsailRelationalDatabaseHardwarePayload(in),
		"latestRestorableTime":          lightsailTimestamp(in.LatestRestorableTime),
		"location":                      lightsailLocationPayload(in.AvailabilityZone, in.Region),
		"masterDatabaseName":            in.MasterDatabaseName,
		"masterEndpoint":                lightsailRelationalDatabaseEndpointPayload(in),
		"masterUsername":                in.MasterUsername,
		"name":                          in.Name,
		"parameterApplyStatus":          in.ParameterApplyStatus,
		"pendingMaintenanceActions":     lightsailRelationalDatabasePendingMaintenanceActionsPayload(in),
		"preferredBackupWindow":         in.PreferredBackupWindow,
		"preferredMaintenanceWindow":    in.PreferredMaintenanceWindow,
		"publiclyAccessible":            in.PubliclyAccessible,
		"relationalDatabaseBlueprintId": in.BlueprintID,
		"relationalDatabaseBundleId":    in.BundleID,
		"resourceType":                  "RelationalDatabase",
		"secondaryAvailabilityZone":     in.SecondaryAvailabilityZone,
		"state":                         in.State,
		"supportCode":                   in.SupportCode,
		"tags":                          lightsailMapToTags(in.Tags),
	}
	if in.PendingModifiedValues != nil {
		out["pendingModifiedValues"] = lightsailRelationalDatabasePendingModifiedValuesPayload(in.PendingModifiedValues)
	}
	return out
}

func lightsailRelationalDatabaseHardwarePayload(in lightsail.RelationalDatabase) map[string]any {
	return map[string]any{
		"cpuCount":     in.CPUCount,
		"diskSizeInGb": in.DiskSizeInGb,
		"ramSizeInGb":  in.RAMSizeInGb,
	}
}

func lightsailRelationalDatabaseEndpointPayload(in lightsail.RelationalDatabase) map[string]any {
	return map[string]any{
		"address": in.MasterEndpointAddress,
		"port":    in.MasterEndpointPort,
	}
}

func lightsailRelationalDatabasePendingModifiedValuesPayload(in *lightsail.PendingModifiedRelationalDatabaseValues) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := map[string]any{}
	if in.BackupRetentionEnabled != nil {
		out["backupRetentionEnabled"] = *in.BackupRetentionEnabled
	}
	if in.EngineVersion != nil {
		out["engineVersion"] = *in.EngineVersion
	}
	if in.MasterUserPassword != nil {
		out["masterUserPassword"] = *in.MasterUserPassword
	}
	return out
}

func lightsailRelationalDatabasePendingMaintenanceActionsPayload(in lightsail.RelationalDatabase) []map[string]any {
	if len(in.PendingMaintenanceActions) == 0 {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(in.PendingMaintenanceActions))
	for _, action := range in.PendingMaintenanceActions {
		item := map[string]any{
			"action": action,
		}
		if in.PendingMaintenanceActionCode != "" {
			item["currentApplyDate"] = lightsailTimestamp(in.CreatedAt)
			item["description"] = in.PendingMaintenanceActionCode
		}
		out = append(out, item)
	}
	return out
}

func lightsailRelationalDatabaseSnapshotPayload(in lightsail.RelationalDatabaseSnapshot) map[string]any {
	return map[string]any{
		"arn":                               in.ARN,
		"createdAt":                         lightsailTimestamp(in.CreatedAt),
		"engine":                            in.Engine,
		"engineVersion":                     in.EngineVersion,
		"fromRelationalDatabaseArn":         in.FromRelationalDatabaseARN,
		"fromRelationalDatabaseBlueprintId": in.FromRelationalDatabaseBlueprintID,
		"fromRelationalDatabaseBundleId":    in.FromRelationalDatabaseBundleID,
		"fromRelationalDatabaseName":        in.FromRelationalDatabaseName,
		"location":                          lightsailLocationPayload(in.AvailabilityZone, in.Region),
		"name":                              in.Name,
		"resourceType":                      "RelationalDatabaseSnapshot",
		"sizeInGb":                          in.SizeInGb,
		"state":                             in.State,
		"supportCode":                       in.SupportCode,
		"tags":                              lightsailMapToTags(in.Tags),
	}
}

func lightsailRelationalDatabaseEventPayload(in lightsail.RelationalDatabaseEvent) map[string]any {
	return map[string]any{
		"createdAt":       lightsailTimestamp(in.CreatedAt),
		"eventCategories": append([]string(nil), in.EventCategories...),
		"message":         in.Message,
		"resource":        in.Resource,
	}
}

func lightsailRelationalDatabaseLogEventsPayload(in []lightsail.RelationalDatabaseLogEvent) []map[string]any {
	out := make([]map[string]any, 0, len(in))
	for _, event := range in {
		out = append(out, map[string]any{
			"createdAt": lightsailTimestamp(event.CreatedAt),
			"message":   event.Message,
		})
	}
	return out
}

func lightsailRelationalDatabaseParameterPayload(in lightsail.RelationalDatabaseParameter) map[string]any {
	return map[string]any{
		"allowedValues":  in.AllowedValues,
		"applyMethod":    in.ApplyMethod,
		"applyType":      in.ApplyType,
		"dataType":       in.DataType,
		"description":    in.Description,
		"isModifiable":   in.IsModifiable,
		"parameterName":  in.ParameterName,
		"parameterValue": in.ParameterValue,
	}
}

func lightsailRegionsPayload(regions []lightsail.Region) []map[string]any {
	out := make([]map[string]any, 0, len(regions))
	for _, region := range regions {
		item := map[string]any{
			"name":          region.Name,
			"displayName":   region.DisplayName,
			"description":   region.Description,
			"continentCode": region.ContinentCode,
		}
		if len(region.AvailabilityZones) > 0 {
			item["availabilityZones"] = lightsailAvailabilityZonesPayload(region.AvailabilityZones)
		}
		if len(region.DatabaseZones) > 0 {
			item["relationalDatabaseAvailabilityZones"] = lightsailAvailabilityZonesPayload(region.DatabaseZones)
		}
		out = append(out, item)
	}
	return out
}

func lightsailAvailabilityZonesPayload(zones []string) []map[string]any {
	out := make([]map[string]any, 0, len(zones))
	for _, zone := range zones {
		zone = strings.TrimSpace(zone)
		if zone == "" {
			continue
		}
		out = append(out, map[string]any{
			"zoneName": zone,
			"state":    "available",
		})
	}
	return out
}

func lightsailLocationPayload(availabilityZone, region string) map[string]any {
	return map[string]any{
		"availabilityZone": availabilityZone,
		"regionName":       region,
	}
}

func boolValueOrDefault(in *bool, fallback bool) bool {
	if in == nil {
		return fallback
	}
	return *in
}

func lightsailTimestamp(ts time.Time) float64 {
	if ts.IsZero() {
		return 0
	}
	return float64(ts.UnixNano()) / float64(time.Second)
}
