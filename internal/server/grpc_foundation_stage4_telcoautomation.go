package server

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	telcoautomationpb "cloud.google.com/go/telcoautomation/apiv1/telcoautomationpb"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	gcpTelcoAutomationListOrchestrationClustersMethod  = "/google.cloud.telcoautomation.v1.TelcoAutomation/ListOrchestrationClusters"
	gcpTelcoAutomationGetOrchestrationClusterMethod    = "/google.cloud.telcoautomation.v1.TelcoAutomation/GetOrchestrationCluster"
	gcpTelcoAutomationCreateOrchestrationClusterMethod = "/google.cloud.telcoautomation.v1.TelcoAutomation/CreateOrchestrationCluster"
	gcpTelcoAutomationDeleteOrchestrationClusterMethod = "/google.cloud.telcoautomation.v1.TelcoAutomation/DeleteOrchestrationCluster"
	gcpTelcoAutomationListEdgeSlmsMethod               = "/google.cloud.telcoautomation.v1.TelcoAutomation/ListEdgeSlms"
	gcpTelcoAutomationGetEdgeSlmMethod                 = "/google.cloud.telcoautomation.v1.TelcoAutomation/GetEdgeSlm"
	gcpTelcoAutomationCreateEdgeSlmMethod              = "/google.cloud.telcoautomation.v1.TelcoAutomation/CreateEdgeSlm"
	gcpTelcoAutomationDeleteEdgeSlmMethod              = "/google.cloud.telcoautomation.v1.TelcoAutomation/DeleteEdgeSlm"
	gcpTelcoAutomationCreateBlueprintMethod            = "/google.cloud.telcoautomation.v1.TelcoAutomation/CreateBlueprint"
	gcpTelcoAutomationUpdateBlueprintMethod            = "/google.cloud.telcoautomation.v1.TelcoAutomation/UpdateBlueprint"
	gcpTelcoAutomationGetBlueprintMethod               = "/google.cloud.telcoautomation.v1.TelcoAutomation/GetBlueprint"
	gcpTelcoAutomationDeleteBlueprintMethod            = "/google.cloud.telcoautomation.v1.TelcoAutomation/DeleteBlueprint"
	gcpTelcoAutomationListBlueprintsMethod             = "/google.cloud.telcoautomation.v1.TelcoAutomation/ListBlueprints"
	gcpTelcoAutomationApproveBlueprintMethod           = "/google.cloud.telcoautomation.v1.TelcoAutomation/ApproveBlueprint"
	gcpTelcoAutomationProposeBlueprintMethod           = "/google.cloud.telcoautomation.v1.TelcoAutomation/ProposeBlueprint"
	gcpTelcoAutomationRejectBlueprintMethod            = "/google.cloud.telcoautomation.v1.TelcoAutomation/RejectBlueprint"
	gcpTelcoAutomationListBlueprintRevisionsMethod     = "/google.cloud.telcoautomation.v1.TelcoAutomation/ListBlueprintRevisions"
	gcpTelcoAutomationSearchBlueprintRevisionsMethod   = "/google.cloud.telcoautomation.v1.TelcoAutomation/SearchBlueprintRevisions"
	gcpTelcoAutomationSearchDeploymentRevisionsMethod  = "/google.cloud.telcoautomation.v1.TelcoAutomation/SearchDeploymentRevisions"
	gcpTelcoAutomationDiscardBlueprintChangesMethod    = "/google.cloud.telcoautomation.v1.TelcoAutomation/DiscardBlueprintChanges"
	gcpTelcoAutomationListPublicBlueprintsMethod       = "/google.cloud.telcoautomation.v1.TelcoAutomation/ListPublicBlueprints"
	gcpTelcoAutomationGetPublicBlueprintMethod         = "/google.cloud.telcoautomation.v1.TelcoAutomation/GetPublicBlueprint"
	gcpTelcoAutomationCreateDeploymentMethod           = "/google.cloud.telcoautomation.v1.TelcoAutomation/CreateDeployment"
	gcpTelcoAutomationUpdateDeploymentMethod           = "/google.cloud.telcoautomation.v1.TelcoAutomation/UpdateDeployment"
	gcpTelcoAutomationGetDeploymentMethod              = "/google.cloud.telcoautomation.v1.TelcoAutomation/GetDeployment"
	gcpTelcoAutomationRemoveDeploymentMethod           = "/google.cloud.telcoautomation.v1.TelcoAutomation/RemoveDeployment"
	gcpTelcoAutomationListDeploymentsMethod            = "/google.cloud.telcoautomation.v1.TelcoAutomation/ListDeployments"
	gcpTelcoAutomationListDeploymentRevisionsMethod    = "/google.cloud.telcoautomation.v1.TelcoAutomation/ListDeploymentRevisions"
	gcpTelcoAutomationDiscardDeploymentChangesMethod   = "/google.cloud.telcoautomation.v1.TelcoAutomation/DiscardDeploymentChanges"
	gcpTelcoAutomationApplyDeploymentMethod            = "/google.cloud.telcoautomation.v1.TelcoAutomation/ApplyDeployment"
	gcpTelcoAutomationComputeDeploymentStatusMethod    = "/google.cloud.telcoautomation.v1.TelcoAutomation/ComputeDeploymentStatus"
	gcpTelcoAutomationRollbackDeploymentMethod         = "/google.cloud.telcoautomation.v1.TelcoAutomation/RollbackDeployment"
	gcpTelcoAutomationGetHydratedDeploymentMethod      = "/google.cloud.telcoautomation.v1.TelcoAutomation/GetHydratedDeployment"
	gcpTelcoAutomationListHydratedDeploymentsMethod    = "/google.cloud.telcoautomation.v1.TelcoAutomation/ListHydratedDeployments"
	gcpTelcoAutomationUpdateHydratedDeploymentMethod   = "/google.cloud.telcoautomation.v1.TelcoAutomation/UpdateHydratedDeployment"
	gcpTelcoAutomationApplyHydratedDeploymentMethod    = "/google.cloud.telcoautomation.v1.TelcoAutomation/ApplyHydratedDeployment"
)

func gcpStage4GRPCTelcoAutomation(path string, grpcReqBody []byte) ([]byte, string, string, bool) {
	switch path {
	case gcpTelcoAutomationListOrchestrationClustersMethod:
		return gcpStage4GRPCTelcoAutomationListOrchestrationClusters(grpcReqBody)
	case gcpTelcoAutomationGetOrchestrationClusterMethod:
		return gcpStage4GRPCTelcoAutomationGetOrchestrationCluster(grpcReqBody)
	case gcpTelcoAutomationCreateOrchestrationClusterMethod:
		return gcpStage4GRPCTelcoAutomationCreateOrchestrationCluster(grpcReqBody)
	case gcpTelcoAutomationDeleteOrchestrationClusterMethod:
		return gcpStage4GRPCTelcoAutomationDeleteOrchestrationCluster(grpcReqBody)
	case gcpTelcoAutomationListEdgeSlmsMethod:
		return gcpStage4GRPCTelcoAutomationListEdgeSlms(grpcReqBody)
	case gcpTelcoAutomationGetEdgeSlmMethod:
		return gcpStage4GRPCTelcoAutomationGetEdgeSlm(grpcReqBody)
	case gcpTelcoAutomationCreateEdgeSlmMethod:
		return gcpStage4GRPCTelcoAutomationCreateEdgeSlm(grpcReqBody)
	case gcpTelcoAutomationDeleteEdgeSlmMethod:
		return gcpStage4GRPCTelcoAutomationDeleteEdgeSlm(grpcReqBody)
	case gcpTelcoAutomationCreateBlueprintMethod:
		return gcpStage4GRPCTelcoAutomationCreateBlueprint(grpcReqBody)
	case gcpTelcoAutomationUpdateBlueprintMethod:
		return gcpStage4GRPCTelcoAutomationUpdateBlueprint(grpcReqBody)
	case gcpTelcoAutomationGetBlueprintMethod:
		return gcpStage4GRPCTelcoAutomationGetBlueprint(grpcReqBody)
	case gcpTelcoAutomationDeleteBlueprintMethod:
		return gcpStage4GRPCTelcoAutomationDeleteBlueprint(grpcReqBody)
	case gcpTelcoAutomationListBlueprintsMethod:
		return gcpStage4GRPCTelcoAutomationListBlueprints(grpcReqBody)
	case gcpTelcoAutomationApproveBlueprintMethod:
		return gcpStage4GRPCTelcoAutomationBlueprintAction(grpcReqBody, "approve")
	case gcpTelcoAutomationProposeBlueprintMethod:
		return gcpStage4GRPCTelcoAutomationBlueprintAction(grpcReqBody, "propose")
	case gcpTelcoAutomationRejectBlueprintMethod:
		return gcpStage4GRPCTelcoAutomationBlueprintAction(grpcReqBody, "reject")
	case gcpTelcoAutomationListBlueprintRevisionsMethod:
		return gcpStage4GRPCTelcoAutomationListBlueprintRevisions(grpcReqBody)
	case gcpTelcoAutomationSearchBlueprintRevisionsMethod:
		return gcpStage4GRPCTelcoAutomationSearchBlueprintRevisions(grpcReqBody)
	case gcpTelcoAutomationSearchDeploymentRevisionsMethod:
		return gcpStage4GRPCTelcoAutomationSearchDeploymentRevisions(grpcReqBody)
	case gcpTelcoAutomationDiscardBlueprintChangesMethod:
		return gcpStage4GRPCTelcoAutomationDiscardBlueprintChanges(grpcReqBody)
	case gcpTelcoAutomationListPublicBlueprintsMethod:
		return gcpStage4GRPCTelcoAutomationListPublicBlueprints(grpcReqBody)
	case gcpTelcoAutomationGetPublicBlueprintMethod:
		return gcpStage4GRPCTelcoAutomationGetPublicBlueprint(grpcReqBody)
	case gcpTelcoAutomationCreateDeploymentMethod:
		return gcpStage4GRPCTelcoAutomationCreateDeployment(grpcReqBody)
	case gcpTelcoAutomationUpdateDeploymentMethod:
		return gcpStage4GRPCTelcoAutomationUpdateDeployment(grpcReqBody)
	case gcpTelcoAutomationGetDeploymentMethod:
		return gcpStage4GRPCTelcoAutomationGetDeployment(grpcReqBody)
	case gcpTelcoAutomationRemoveDeploymentMethod:
		return gcpStage4GRPCTelcoAutomationRemoveDeployment(grpcReqBody)
	case gcpTelcoAutomationListDeploymentsMethod:
		return gcpStage4GRPCTelcoAutomationListDeployments(grpcReqBody)
	case gcpTelcoAutomationListDeploymentRevisionsMethod:
		return gcpStage4GRPCTelcoAutomationListDeploymentRevisions(grpcReqBody)
	case gcpTelcoAutomationDiscardDeploymentChangesMethod:
		return gcpStage4GRPCTelcoAutomationDiscardDeploymentChanges(grpcReqBody)
	case gcpTelcoAutomationApplyDeploymentMethod:
		return gcpStage4GRPCTelcoAutomationApplyDeployment(grpcReqBody)
	case gcpTelcoAutomationComputeDeploymentStatusMethod:
		return gcpStage4GRPCTelcoAutomationComputeDeploymentStatus(grpcReqBody)
	case gcpTelcoAutomationRollbackDeploymentMethod:
		return gcpStage4GRPCTelcoAutomationRollbackDeployment(grpcReqBody)
	case gcpTelcoAutomationGetHydratedDeploymentMethod:
		return gcpStage4GRPCTelcoAutomationGetHydratedDeployment(grpcReqBody)
	case gcpTelcoAutomationListHydratedDeploymentsMethod:
		return gcpStage4GRPCTelcoAutomationListHydratedDeployments(grpcReqBody)
	case gcpTelcoAutomationUpdateHydratedDeploymentMethod:
		return gcpStage4GRPCTelcoAutomationUpdateHydratedDeployment(grpcReqBody)
	case gcpTelcoAutomationApplyHydratedDeploymentMethod:
		return gcpStage4GRPCTelcoAutomationApplyHydratedDeployment(grpcReqBody)
	default:
		return grpcUnimplemented("method-not-implemented")
	}
}

func gcpStage4GRPCTelcoAutomationListOrchestrationClusters(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &telcoautomationpb.ListOrchestrationClustersRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := parseGCPTelcoAutomationLocationName(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if strings.TrimSpace(req.GetFilter()) != "" && !isGCPTelcoAutomationSimpleStateFilter(req.GetFilter(), []string{"ACTIVE", "CREATING", "FAILED", "DELETING"}) {
		return grpcInvalidArgument("filter-invalid")
	}
	if orderBy := strings.TrimSpace(req.GetOrderBy()); orderBy != "" && orderBy != "name" && orderBy != "create_time" && orderBy != "create_time desc" {
		return grpcInvalidArgument("order_by-invalid")
	}

	items := []*telcoautomationpb.OrchestrationCluster{
		gcpStage4TelcoAutomationOrchestrationCluster(project, location, "cluster-1", telcoautomationpb.OrchestrationCluster_ACTIVE),
		gcpStage4TelcoAutomationOrchestrationCluster(project, location, "cluster-2", telcoautomationpb.OrchestrationCluster_CREATING),
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].GetName() < items[j].GetName()
	})
	filter := strings.TrimSpace(req.GetFilter())
	if filter != "" {
		filtered := make([]*telcoautomationpb.OrchestrationCluster, 0, len(items))
		for _, item := range items {
			if gcpStage4TelcoAutomationStateMatchesFilter(filter, item.GetState().String()) {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}

	start, end, nextPageToken, reason, ok := gcpStage4TelcoAutomationPageWindow(req.GetPageSize(), req.GetPageToken(), 1000, len(items))
	if !ok {
		return grpcInvalidArgument(reason)
	}
	return grpcProtoSuccess(&telcoautomationpb.ListOrchestrationClustersResponse{
		OrchestrationClusters: items[start:end],
		NextPageToken:         nextPageToken,
		Unreachable:           []string{},
	})
}

func gcpStage4GRPCTelcoAutomationGetOrchestrationCluster(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &telcoautomationpb.GetOrchestrationClusterRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, clusterID, ok := parseGCPTelcoAutomationOrchestrationClusterName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPTelcoAutomationMissingID(clusterID) {
		return grpcNotFound("orchestration-cluster-not-found")
	}
	state := telcoautomationpb.OrchestrationCluster_ACTIVE
	lowerID := strings.ToLower(clusterID)
	if strings.Contains(lowerID, "creating") {
		state = telcoautomationpb.OrchestrationCluster_CREATING
	}
	if strings.Contains(lowerID, "failed") {
		state = telcoautomationpb.OrchestrationCluster_FAILED
	}
	return grpcProtoSuccess(gcpStage4TelcoAutomationOrchestrationCluster(project, location, clusterID, state))
}

func gcpStage4GRPCTelcoAutomationCreateOrchestrationCluster(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &telcoautomationpb.CreateOrchestrationClusterRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := parseGCPTelcoAutomationLocationName(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	clusterID := strings.TrimSpace(req.GetOrchestrationClusterId())
	if clusterID == "" {
		return grpcInvalidArgument("orchestration_cluster_id-required")
	}
	if !isGCPTelcoAutomationResourceID(clusterID) {
		return grpcInvalidArgument("orchestration_cluster_id-invalid")
	}
	if requestID := strings.TrimSpace(req.GetRequestId()); requestID != "" && !isGCPTelcoAutomationRequestID(requestID) {
		return grpcInvalidArgument("request_id-invalid")
	}
	if req.GetOrchestrationCluster() == nil {
		return grpcInvalidArgument("orchestration_cluster-required")
	}
	expectedName := fmt.Sprintf("projects/%s/locations/%s/orchestrationClusters/%s", project, location, clusterID)
	if provided := strings.TrimSpace(req.GetOrchestrationCluster().GetName()); provided != "" && provided != expectedName {
		return grpcInvalidArgument("orchestration_cluster-name-mismatch")
	}

	response := gcpStage4TelcoAutomationOrchestrationCluster(project, location, clusterID, telcoautomationpb.OrchestrationCluster_ACTIVE)
	gcpStage4TelcoAutomationApplyOrchestrationClusterOverrides(response, req.GetOrchestrationCluster())
	return grpcProtoSuccess(gcpStage4TelcoAutomationOperation(
		project,
		location,
		"createOrchestrationCluster."+clusterID,
		"create",
		expectedName,
		response,
	))
}

func gcpStage4GRPCTelcoAutomationDeleteOrchestrationCluster(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &telcoautomationpb.DeleteOrchestrationClusterRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, clusterID, ok := parseGCPTelcoAutomationOrchestrationClusterName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if requestID := strings.TrimSpace(req.GetRequestId()); requestID != "" && !isGCPTelcoAutomationRequestID(requestID) {
		return grpcInvalidArgument("request_id-invalid")
	}
	if isGCPTelcoAutomationMissingID(clusterID) {
		return grpcNotFound("orchestration-cluster-not-found")
	}
	target := fmt.Sprintf("projects/%s/locations/%s/orchestrationClusters/%s", project, location, clusterID)
	return grpcProtoSuccess(gcpStage4TelcoAutomationOperation(
		project,
		location,
		"deleteOrchestrationCluster."+clusterID,
		"delete",
		target,
		&emptypb.Empty{},
	))
}

func gcpStage4GRPCTelcoAutomationListEdgeSlms(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &telcoautomationpb.ListEdgeSlmsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := parseGCPTelcoAutomationLocationName(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if strings.TrimSpace(req.GetFilter()) != "" && !isGCPTelcoAutomationSimpleStateFilter(req.GetFilter(), []string{"ACTIVE", "CREATING", "FAILED", "DELETING"}) {
		return grpcInvalidArgument("filter-invalid")
	}
	if orderBy := strings.TrimSpace(req.GetOrderBy()); orderBy != "" && orderBy != "name" && orderBy != "create_time" && orderBy != "create_time desc" {
		return grpcInvalidArgument("order_by-invalid")
	}

	items := []*telcoautomationpb.EdgeSlm{
		gcpStage4TelcoAutomationEdgeSlm(project, location, "edgeslm-1", "cluster-1", telcoautomationpb.EdgeSlm_ACTIVE),
		gcpStage4TelcoAutomationEdgeSlm(project, location, "edgeslm-2", "cluster-2", telcoautomationpb.EdgeSlm_CREATING),
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].GetName() < items[j].GetName()
	})
	filter := strings.TrimSpace(req.GetFilter())
	if filter != "" {
		filtered := make([]*telcoautomationpb.EdgeSlm, 0, len(items))
		for _, item := range items {
			if gcpStage4TelcoAutomationStateMatchesFilter(filter, item.GetState().String()) {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}

	start, end, nextPageToken, reason, ok := gcpStage4TelcoAutomationPageWindow(req.GetPageSize(), req.GetPageToken(), 1000, len(items))
	if !ok {
		return grpcInvalidArgument(reason)
	}
	return grpcProtoSuccess(&telcoautomationpb.ListEdgeSlmsResponse{
		EdgeSlms:      items[start:end],
		NextPageToken: nextPageToken,
		Unreachable:   []string{},
	})
}

func gcpStage4GRPCTelcoAutomationGetEdgeSlm(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &telcoautomationpb.GetEdgeSlmRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, edgeID, ok := parseGCPTelcoAutomationEdgeSlmName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPTelcoAutomationMissingID(edgeID) {
		return grpcNotFound("edge_slm-not-found")
	}
	state := telcoautomationpb.EdgeSlm_ACTIVE
	lowerID := strings.ToLower(edgeID)
	if strings.Contains(lowerID, "creating") {
		state = telcoautomationpb.EdgeSlm_CREATING
	}
	if strings.Contains(lowerID, "failed") {
		state = telcoautomationpb.EdgeSlm_FAILED
	}
	return grpcProtoSuccess(gcpStage4TelcoAutomationEdgeSlm(project, location, edgeID, "cluster-1", state))
}

func gcpStage4GRPCTelcoAutomationCreateEdgeSlm(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &telcoautomationpb.CreateEdgeSlmRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := parseGCPTelcoAutomationLocationName(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	edgeID := strings.TrimSpace(req.GetEdgeSlmId())
	if edgeID == "" {
		return grpcInvalidArgument("edge_slm_id-required")
	}
	if !isGCPTelcoAutomationResourceID(edgeID) {
		return grpcInvalidArgument("edge_slm_id-invalid")
	}
	if requestID := strings.TrimSpace(req.GetRequestId()); requestID != "" && !isGCPTelcoAutomationRequestID(requestID) {
		return grpcInvalidArgument("request_id-invalid")
	}
	if req.GetEdgeSlm() == nil {
		return grpcInvalidArgument("edge_slm-required")
	}
	expectedName := fmt.Sprintf("projects/%s/locations/%s/edgeSlms/%s", project, location, edgeID)
	if provided := strings.TrimSpace(req.GetEdgeSlm().GetName()); provided != "" && provided != expectedName {
		return grpcInvalidArgument("edge_slm-name-mismatch")
	}
	clusterName := strings.TrimSpace(req.GetEdgeSlm().GetOrchestrationCluster())
	if clusterName == "" {
		return grpcInvalidArgument("edge_slm-orchestration_cluster-required")
	}
	_, _, clusterID, ok := parseGCPTelcoAutomationOrchestrationClusterName(clusterName)
	if !ok {
		return grpcInvalidArgument("edge_slm-orchestration_cluster-invalid")
	}

	response := gcpStage4TelcoAutomationEdgeSlm(project, location, edgeID, clusterID, telcoautomationpb.EdgeSlm_ACTIVE)
	gcpStage4TelcoAutomationApplyEdgeSlmOverrides(response, req.GetEdgeSlm())
	return grpcProtoSuccess(gcpStage4TelcoAutomationOperation(
		project,
		location,
		"createEdgeSlm."+edgeID,
		"create",
		expectedName,
		response,
	))
}

func gcpStage4GRPCTelcoAutomationDeleteEdgeSlm(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &telcoautomationpb.DeleteEdgeSlmRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, edgeID, ok := parseGCPTelcoAutomationEdgeSlmName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if requestID := strings.TrimSpace(req.GetRequestId()); requestID != "" && !isGCPTelcoAutomationRequestID(requestID) {
		return grpcInvalidArgument("request_id-invalid")
	}
	if isGCPTelcoAutomationMissingID(edgeID) {
		return grpcNotFound("edge_slm-not-found")
	}
	target := fmt.Sprintf("projects/%s/locations/%s/edgeSlms/%s", project, location, edgeID)
	return grpcProtoSuccess(gcpStage4TelcoAutomationOperation(
		project,
		location,
		"deleteEdgeSlm."+edgeID,
		"delete",
		target,
		&emptypb.Empty{},
	))
}

func gcpStage4GRPCTelcoAutomationCreateBlueprint(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &telcoautomationpb.CreateBlueprintRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, clusterID, ok := parseGCPTelcoAutomationOrchestrationClusterName(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetBlueprint() == nil {
		return grpcInvalidArgument("blueprint-required")
	}
	blueprintID := strings.TrimSpace(req.GetBlueprintId())
	if blueprintID == "" {
		blueprintID = "blueprint-created-1"
	}
	if !isGCPTelcoAutomationResourceID(blueprintID) {
		return grpcInvalidArgument("blueprint_id-invalid")
	}
	expectedName := fmt.Sprintf("projects/%s/locations/%s/orchestrationClusters/%s/blueprints/%s", project, location, clusterID, blueprintID)
	if provided := strings.TrimSpace(req.GetBlueprint().GetName()); provided != "" && provided != expectedName {
		return grpcInvalidArgument("blueprint-name-mismatch")
	}
	if strings.TrimSpace(req.GetBlueprint().GetSourceBlueprint()) == "" {
		return grpcInvalidArgument("blueprint-source_blueprint-required")
	}

	response := gcpStage4TelcoAutomationBlueprint(project, location, clusterID, blueprintID, "rev-1", telcoautomationpb.Blueprint_DRAFT)
	gcpStage4TelcoAutomationApplyBlueprintOverrides(response, req.GetBlueprint())
	return grpcProtoSuccess(response)
}

func gcpStage4GRPCTelcoAutomationUpdateBlueprint(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &telcoautomationpb.UpdateBlueprintRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	blueprint := req.GetBlueprint()
	if blueprint == nil {
		return grpcInvalidArgument("blueprint-required")
	}
	project, location, clusterID, blueprintID, revisionID, ok := parseGCPTelcoAutomationBlueprintName(strings.TrimSpace(blueprint.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if revisionID != "" {
		return grpcInvalidArgument("name-revision-not-allowed")
	}
	updateMask := req.GetUpdateMask()
	if updateMask == nil || len(updateMask.GetPaths()) == 0 {
		return grpcInvalidArgument("update_mask-required")
	}
	allowed := map[string]struct{}{
		"display_name": {}, "displayName": {},
		"files": {}, "labels": {}, "*": {},
	}
	for _, field := range updateMask.GetPaths() {
		trimmed := strings.TrimSpace(field)
		if _, ok := allowed[trimmed]; !ok {
			return grpcInvalidArgument("update_mask-invalid")
		}
	}

	response := gcpStage4TelcoAutomationBlueprint(
		project,
		location,
		clusterID,
		blueprintID,
		"rev-2",
		telcoautomationpb.Blueprint_ApprovalState(gcpTelcoAutomationBlueprintStateForName(blueprintID)),
	)
	gcpStage4TelcoAutomationApplyBlueprintOverrides(response, blueprint)
	return grpcProtoSuccess(response)
}

func gcpStage4GRPCTelcoAutomationGetBlueprint(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &telcoautomationpb.GetBlueprintRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, clusterID, blueprintID, revisionID, ok := parseGCPTelcoAutomationBlueprintName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPTelcoAutomationMissingID(blueprintID) {
		return grpcNotFound("blueprint-not-found")
	}
	state := telcoautomationpb.Blueprint_ApprovalState(gcpTelcoAutomationBlueprintStateForName(blueprintID))
	if revisionID == "" {
		switch state {
		case telcoautomationpb.Blueprint_DRAFT:
			revisionID = "rev-1"
		case telcoautomationpb.Blueprint_PROPOSED:
			revisionID = "rev-2"
		default:
			revisionID = "rev-3"
		}
	}
	return grpcProtoSuccess(gcpStage4TelcoAutomationBlueprint(project, location, clusterID, blueprintID, revisionID, state))
}

func gcpStage4GRPCTelcoAutomationDeleteBlueprint(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &telcoautomationpb.DeleteBlueprintRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	_, _, _, _, revisionID, ok := parseGCPTelcoAutomationBlueprintName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if revisionID != "" {
		return grpcInvalidArgument("name-revision-not-allowed")
	}
	return grpcProtoSuccess(&emptypb.Empty{})
}

func gcpStage4GRPCTelcoAutomationListBlueprints(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &telcoautomationpb.ListBlueprintsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, clusterID, ok := parseGCPTelcoAutomationOrchestrationClusterName(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if strings.TrimSpace(req.GetFilter()) != "" && !isGCPTelcoAutomationSimpleStateFilter(req.GetFilter(), []string{"DRAFT", "PROPOSED", "APPROVED"}) {
		return grpcInvalidArgument("filter-invalid")
	}
	items := []*telcoautomationpb.Blueprint{
		gcpStage4TelcoAutomationBlueprint(project, location, clusterID, "blueprint-draft", "rev-1", telcoautomationpb.Blueprint_DRAFT),
		gcpStage4TelcoAutomationBlueprint(project, location, clusterID, "blueprint-proposed", "rev-2", telcoautomationpb.Blueprint_PROPOSED),
		gcpStage4TelcoAutomationBlueprint(project, location, clusterID, "blueprint-approved", "rev-3", telcoautomationpb.Blueprint_APPROVED),
	}
	filter := strings.TrimSpace(req.GetFilter())
	if filter != "" {
		filtered := make([]*telcoautomationpb.Blueprint, 0, len(items))
		for _, item := range items {
			if gcpStage4TelcoAutomationStateMatchesFilter(filter, item.GetApprovalState().String()) {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	start, end, nextPageToken, reason, ok := gcpStage4TelcoAutomationPageWindow(req.GetPageSize(), req.GetPageToken(), 1000, len(items))
	if !ok {
		return grpcInvalidArgument(reason)
	}
	return grpcProtoSuccess(&telcoautomationpb.ListBlueprintsResponse{
		Blueprints:    items[start:end],
		NextPageToken: nextPageToken,
	})
}

func gcpStage4GRPCTelcoAutomationBlueprintAction(grpcReqBody []byte, action string) ([]byte, string, string, bool) {
	var name string
	switch action {
	case "approve":
		req := &telcoautomationpb.ApproveBlueprintRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		name = req.GetName()
	case "propose":
		req := &telcoautomationpb.ProposeBlueprintRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		name = req.GetName()
	case "reject":
		req := &telcoautomationpb.RejectBlueprintRequest{}
		if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
			return grpcInvalidArgument("request-body-invalid")
		}
		name = req.GetName()
	default:
		return grpcUnimplemented("action-not-supported")
	}

	project, location, clusterID, blueprintID, _, ok := parseGCPTelcoAutomationBlueprintName(strings.TrimSpace(name))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPTelcoAutomationMissingID(blueprintID) {
		return grpcNotFound("blueprint-not-found")
	}

	currentState := telcoautomationpb.Blueprint_ApprovalState(gcpTelcoAutomationBlueprintStateForName(blueprintID))
	switch action {
	case "propose":
		if currentState != telcoautomationpb.Blueprint_DRAFT {
			return grpcFailedPrecondition("blueprint-must-be-draft")
		}
		return grpcProtoSuccess(gcpStage4TelcoAutomationBlueprint(project, location, clusterID, blueprintID, "rev-2", telcoautomationpb.Blueprint_PROPOSED))
	case "approve":
		if currentState != telcoautomationpb.Blueprint_PROPOSED {
			return grpcFailedPrecondition("blueprint-must-be-proposed")
		}
		return grpcProtoSuccess(gcpStage4TelcoAutomationBlueprint(project, location, clusterID, blueprintID, "rev-3", telcoautomationpb.Blueprint_APPROVED))
	case "reject":
		if currentState != telcoautomationpb.Blueprint_PROPOSED {
			return grpcFailedPrecondition("blueprint-must-be-proposed")
		}
		return grpcProtoSuccess(gcpStage4TelcoAutomationBlueprint(project, location, clusterID, blueprintID, "rev-2", telcoautomationpb.Blueprint_DRAFT))
	default:
		return grpcUnimplemented("action-not-supported")
	}
}

func gcpStage4GRPCTelcoAutomationListBlueprintRevisions(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &telcoautomationpb.ListBlueprintRevisionsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, clusterID, blueprintID, revisionID, ok := parseGCPTelcoAutomationBlueprintName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if revisionID != "" {
		return grpcInvalidArgument("name-revision-not-allowed")
	}
	items := []*telcoautomationpb.Blueprint{
		gcpStage4TelcoAutomationBlueprint(project, location, clusterID, blueprintID, "rev-1", telcoautomationpb.Blueprint_DRAFT),
		gcpStage4TelcoAutomationBlueprint(project, location, clusterID, blueprintID, "rev-2", telcoautomationpb.Blueprint_APPROVED),
	}
	start, end, nextPageToken, reason, ok := gcpStage4TelcoAutomationPageWindow(req.GetPageSize(), req.GetPageToken(), 1000, len(items))
	if !ok {
		return grpcInvalidArgument(reason)
	}
	return grpcProtoSuccess(&telcoautomationpb.ListBlueprintRevisionsResponse{
		Blueprints:    items[start:end],
		NextPageToken: nextPageToken,
	})
}

func gcpStage4GRPCTelcoAutomationSearchBlueprintRevisions(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &telcoautomationpb.SearchBlueprintRevisionsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, clusterID, ok := parseGCPTelcoAutomationOrchestrationClusterName(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	query := strings.TrimSpace(req.GetQuery())
	if !isGCPTelcoAutomationRevisionSearchQuery(query, true) {
		return grpcInvalidArgument("query-invalid")
	}
	items := []*telcoautomationpb.Blueprint{
		gcpStage4TelcoAutomationBlueprint(project, location, clusterID, "blueprint-draft", "rev-1", telcoautomationpb.Blueprint_DRAFT),
		gcpStage4TelcoAutomationBlueprint(project, location, clusterID, "blueprint-approved", "rev-3", telcoautomationpb.Blueprint_APPROVED),
	}
	items = gcpStage4TelcoAutomationFilterBlueprintRevisionSearch(items, query)
	start, end, nextPageToken, reason, ok := gcpStage4TelcoAutomationPageWindow(req.GetPageSize(), req.GetPageToken(), 100, len(items))
	if !ok {
		return grpcInvalidArgument(reason)
	}
	return grpcProtoSuccess(&telcoautomationpb.SearchBlueprintRevisionsResponse{
		Blueprints:    items[start:end],
		NextPageToken: nextPageToken,
	})
}

func gcpStage4GRPCTelcoAutomationDiscardBlueprintChanges(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &telcoautomationpb.DiscardBlueprintChangesRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if _, _, _, _, _, ok := parseGCPTelcoAutomationBlueprintName(strings.TrimSpace(req.GetName())); !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(&telcoautomationpb.DiscardBlueprintChangesResponse{})
}

func gcpStage4GRPCTelcoAutomationListPublicBlueprints(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &telcoautomationpb.ListPublicBlueprintsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := parseGCPTelcoAutomationLocationName(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	items := []*telcoautomationpb.PublicBlueprint{
		gcpStage4TelcoAutomationPublicBlueprint(project, location, "public-blueprint-1"),
		gcpStage4TelcoAutomationPublicBlueprint(project, location, "public-blueprint-2"),
	}
	start, end, nextPageToken, reason, ok := gcpStage4TelcoAutomationPageWindow(req.GetPageSize(), req.GetPageToken(), 100, len(items))
	if !ok {
		return grpcInvalidArgument(reason)
	}
	return grpcProtoSuccess(&telcoautomationpb.ListPublicBlueprintsResponse{
		PublicBlueprints: items[start:end],
		NextPageToken:    nextPageToken,
	})
}

func gcpStage4GRPCTelcoAutomationGetPublicBlueprint(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &telcoautomationpb.GetPublicBlueprintRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, publicBlueprintID, ok := parseGCPTelcoAutomationPublicBlueprintName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPTelcoAutomationMissingID(publicBlueprintID) {
		return grpcNotFound("public_blueprint-not-found")
	}
	return grpcProtoSuccess(gcpStage4TelcoAutomationPublicBlueprint(project, location, publicBlueprintID))
}

func gcpStage4GRPCTelcoAutomationCreateDeployment(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &telcoautomationpb.CreateDeploymentRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, clusterID, ok := parseGCPTelcoAutomationOrchestrationClusterName(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetDeployment() == nil {
		return grpcInvalidArgument("deployment-required")
	}
	deploymentID := strings.TrimSpace(req.GetDeploymentId())
	if deploymentID == "" {
		deploymentID = "deployment-created-1"
	}
	if !isGCPTelcoAutomationResourceID(deploymentID) {
		return grpcInvalidArgument("deployment_id-invalid")
	}
	expectedName := fmt.Sprintf("projects/%s/locations/%s/orchestrationClusters/%s/deployments/%s", project, location, clusterID, deploymentID)
	if provided := strings.TrimSpace(req.GetDeployment().GetName()); provided != "" && provided != expectedName {
		return grpcInvalidArgument("deployment-name-mismatch")
	}
	if strings.TrimSpace(req.GetDeployment().GetSourceBlueprintRevision()) == "" {
		return grpcInvalidArgument("deployment-source_blueprint_revision-required")
	}
	response := gcpStage4TelcoAutomationDeployment(project, location, clusterID, deploymentID, "rev-1", telcoautomationpb.Deployment_DRAFT)
	gcpStage4TelcoAutomationApplyDeploymentOverrides(response, req.GetDeployment())
	return grpcProtoSuccess(response)
}

func gcpStage4GRPCTelcoAutomationUpdateDeployment(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &telcoautomationpb.UpdateDeploymentRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	deployment := req.GetDeployment()
	if deployment == nil {
		return grpcInvalidArgument("deployment-required")
	}
	project, location, clusterID, deploymentID, revisionID, ok := parseGCPTelcoAutomationDeploymentName(strings.TrimSpace(deployment.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if revisionID != "" {
		return grpcInvalidArgument("name-revision-not-allowed")
	}
	updateMask := req.GetUpdateMask()
	if updateMask == nil || len(updateMask.GetPaths()) == 0 {
		return grpcInvalidArgument("update_mask-required")
	}
	allowed := map[string]struct{}{
		"display_name": {}, "displayName": {},
		"files": {}, "labels": {},
		"source_blueprint_revision": {}, "sourceBlueprintRevision": {},
		"workload_cluster": {}, "workloadCluster": {},
		"*": {},
	}
	for _, field := range updateMask.GetPaths() {
		if _, ok := allowed[strings.TrimSpace(field)]; !ok {
			return grpcInvalidArgument("update_mask-invalid")
		}
	}
	if strings.TrimSpace(deployment.GetSourceBlueprintRevision()) == "" {
		return grpcInvalidArgument("deployment-source_blueprint_revision-required")
	}
	response := gcpStage4TelcoAutomationDeployment(
		project,
		location,
		clusterID,
		deploymentID,
		"rev-2",
		telcoautomationpb.Deployment_State(gcpTelcoAutomationDeploymentStateForName(deploymentID)),
	)
	gcpStage4TelcoAutomationApplyDeploymentOverrides(response, deployment)
	return grpcProtoSuccess(response)
}

func gcpStage4GRPCTelcoAutomationGetDeployment(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &telcoautomationpb.GetDeploymentRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, clusterID, deploymentID, revisionID, ok := parseGCPTelcoAutomationDeploymentName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPTelcoAutomationMissingID(deploymentID) {
		return grpcNotFound("deployment-not-found")
	}
	if revisionID == "" {
		revisionID = "rev-2"
	}
	state := telcoautomationpb.Deployment_State(gcpTelcoAutomationDeploymentStateForName(deploymentID))
	return grpcProtoSuccess(gcpStage4TelcoAutomationDeployment(project, location, clusterID, deploymentID, revisionID, state))
}

func gcpStage4GRPCTelcoAutomationRemoveDeployment(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &telcoautomationpb.RemoveDeploymentRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	_, _, _, deploymentID, _, ok := parseGCPTelcoAutomationDeploymentName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if gcpTelcoAutomationDeploymentStateForName(deploymentID) == 3 {
		return grpcFailedPrecondition("deployment-already-deleting")
	}
	if isGCPTelcoAutomationMissingID(deploymentID) {
		return grpcNotFound("deployment-not-found")
	}
	return grpcProtoSuccess(&emptypb.Empty{})
}

func gcpStage4GRPCTelcoAutomationListDeployments(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &telcoautomationpb.ListDeploymentsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, clusterID, ok := parseGCPTelcoAutomationOrchestrationClusterName(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if strings.TrimSpace(req.GetFilter()) != "" && !isGCPTelcoAutomationSimpleStateFilter(req.GetFilter(), []string{"DRAFT", "APPLIED", "DELETING"}) {
		return grpcInvalidArgument("filter-invalid")
	}
	items := []*telcoautomationpb.Deployment{
		gcpStage4TelcoAutomationDeployment(project, location, clusterID, "deployment-draft", "rev-1", telcoautomationpb.Deployment_DRAFT),
		gcpStage4TelcoAutomationDeployment(project, location, clusterID, "deployment-applied", "rev-2", telcoautomationpb.Deployment_APPLIED),
		gcpStage4TelcoAutomationDeployment(project, location, clusterID, "deployment-deleting", "rev-3", telcoautomationpb.Deployment_DELETING),
	}
	filter := strings.TrimSpace(req.GetFilter())
	if filter != "" {
		filtered := make([]*telcoautomationpb.Deployment, 0, len(items))
		for _, item := range items {
			if gcpStage4TelcoAutomationStateMatchesFilter(filter, item.GetState().String()) {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	start, end, nextPageToken, reason, ok := gcpStage4TelcoAutomationPageWindow(req.GetPageSize(), req.GetPageToken(), 1000, len(items))
	if !ok {
		return grpcInvalidArgument(reason)
	}
	return grpcProtoSuccess(&telcoautomationpb.ListDeploymentsResponse{
		Deployments:   items[start:end],
		NextPageToken: nextPageToken,
	})
}

func gcpStage4GRPCTelcoAutomationListDeploymentRevisions(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &telcoautomationpb.ListDeploymentRevisionsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, clusterID, deploymentID, revisionID, ok := parseGCPTelcoAutomationDeploymentName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if revisionID != "" {
		return grpcInvalidArgument("name-revision-not-allowed")
	}
	items := []*telcoautomationpb.Deployment{
		gcpStage4TelcoAutomationDeployment(project, location, clusterID, deploymentID, "rev-1", telcoautomationpb.Deployment_DRAFT),
		gcpStage4TelcoAutomationDeployment(project, location, clusterID, deploymentID, "rev-2", telcoautomationpb.Deployment_APPLIED),
	}
	start, end, nextPageToken, reason, ok := gcpStage4TelcoAutomationPageWindow(req.GetPageSize(), req.GetPageToken(), 1000, len(items))
	if !ok {
		return grpcInvalidArgument(reason)
	}
	return grpcProtoSuccess(&telcoautomationpb.ListDeploymentRevisionsResponse{
		Deployments:   items[start:end],
		NextPageToken: nextPageToken,
	})
}

func gcpStage4GRPCTelcoAutomationSearchDeploymentRevisions(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &telcoautomationpb.SearchDeploymentRevisionsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, clusterID, ok := parseGCPTelcoAutomationOrchestrationClusterName(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	query := strings.TrimSpace(req.GetQuery())
	if !isGCPTelcoAutomationRevisionSearchQuery(query, false) {
		return grpcInvalidArgument("query-invalid")
	}
	items := []*telcoautomationpb.Deployment{
		gcpStage4TelcoAutomationDeployment(project, location, clusterID, "deployment-draft", "rev-1", telcoautomationpb.Deployment_DRAFT),
		gcpStage4TelcoAutomationDeployment(project, location, clusterID, "deployment-applied", "rev-2", telcoautomationpb.Deployment_APPLIED),
	}
	items = gcpStage4TelcoAutomationFilterDeploymentRevisionSearch(items, query)
	start, end, nextPageToken, reason, ok := gcpStage4TelcoAutomationPageWindow(req.GetPageSize(), req.GetPageToken(), 100, len(items))
	if !ok {
		return grpcInvalidArgument(reason)
	}
	return grpcProtoSuccess(&telcoautomationpb.SearchDeploymentRevisionsResponse{
		Deployments:   items[start:end],
		NextPageToken: nextPageToken,
	})
}

func gcpStage4GRPCTelcoAutomationDiscardDeploymentChanges(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &telcoautomationpb.DiscardDeploymentChangesRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if _, _, _, _, _, ok := parseGCPTelcoAutomationDeploymentName(strings.TrimSpace(req.GetName())); !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(&telcoautomationpb.DiscardDeploymentChangesResponse{})
}

func gcpStage4GRPCTelcoAutomationApplyDeployment(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &telcoautomationpb.ApplyDeploymentRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, clusterID, deploymentID, _, ok := parseGCPTelcoAutomationDeploymentName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if gcpTelcoAutomationDeploymentStateForName(deploymentID) != 1 {
		return grpcFailedPrecondition("deployment-must-be-draft")
	}
	return grpcProtoSuccess(gcpStage4TelcoAutomationDeployment(project, location, clusterID, deploymentID, "rev-applied-1", telcoautomationpb.Deployment_APPLIED))
}

func gcpStage4GRPCTelcoAutomationComputeDeploymentStatus(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &telcoautomationpb.ComputeDeploymentStatusRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, clusterID, deploymentID, _, ok := parseGCPTelcoAutomationDeploymentName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPTelcoAutomationMissingID(deploymentID) {
		return grpcNotFound("deployment-not-found")
	}
	name := fmt.Sprintf("projects/%s/locations/%s/orchestrationClusters/%s/deployments/%s", project, location, clusterID, deploymentID)
	return grpcProtoSuccess(gcpStage4TelcoAutomationComputeDeploymentStatus(name))
}

func gcpStage4GRPCTelcoAutomationRollbackDeployment(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &telcoautomationpb.RollbackDeploymentRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, clusterID, deploymentID, _, ok := parseGCPTelcoAutomationDeploymentName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if strings.TrimSpace(req.GetRevisionId()) == "" {
		return grpcInvalidArgument("revision_id-required")
	}
	if gcpTelcoAutomationDeploymentStateForName(deploymentID) != 2 {
		return grpcFailedPrecondition("deployment-must-be-applied")
	}
	return grpcProtoSuccess(gcpStage4TelcoAutomationDeployment(
		project,
		location,
		clusterID,
		deploymentID,
		strings.TrimSpace(req.GetRevisionId()),
		telcoautomationpb.Deployment_APPLIED,
	))
}

func gcpStage4GRPCTelcoAutomationGetHydratedDeployment(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &telcoautomationpb.GetHydratedDeploymentRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, clusterID, deploymentID, hydratedID, ok := parseGCPTelcoAutomationHydratedDeploymentName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPTelcoAutomationMissingID(hydratedID) {
		return grpcNotFound("hydrated_deployment-not-found")
	}
	state := telcoautomationpb.HydratedDeployment_DRAFT
	if strings.Contains(strings.ToLower(hydratedID), "applied") {
		state = telcoautomationpb.HydratedDeployment_APPLIED
	}
	return grpcProtoSuccess(gcpStage4TelcoAutomationHydratedDeployment(project, location, clusterID, deploymentID, hydratedID, state))
}

func gcpStage4GRPCTelcoAutomationListHydratedDeployments(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &telcoautomationpb.ListHydratedDeploymentsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, clusterID, deploymentID, revisionID, ok := parseGCPTelcoAutomationDeploymentName(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if revisionID != "" {
		return grpcInvalidArgument("parent-revision-not-allowed")
	}
	items := []*telcoautomationpb.HydratedDeployment{
		gcpStage4TelcoAutomationHydratedDeployment(project, location, clusterID, deploymentID, "hydrated-draft", telcoautomationpb.HydratedDeployment_DRAFT),
		gcpStage4TelcoAutomationHydratedDeployment(project, location, clusterID, deploymentID, "hydrated-applied", telcoautomationpb.HydratedDeployment_APPLIED),
	}
	start, end, nextPageToken, reason, ok := gcpStage4TelcoAutomationPageWindow(req.GetPageSize(), req.GetPageToken(), 1000, len(items))
	if !ok {
		return grpcInvalidArgument(reason)
	}
	return grpcProtoSuccess(&telcoautomationpb.ListHydratedDeploymentsResponse{
		HydratedDeployments: items[start:end],
		NextPageToken:       nextPageToken,
	})
}

func gcpStage4GRPCTelcoAutomationUpdateHydratedDeployment(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &telcoautomationpb.UpdateHydratedDeploymentRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	hydrated := req.GetHydratedDeployment()
	if hydrated == nil {
		return grpcInvalidArgument("hydrated_deployment-required")
	}
	project, location, clusterID, deploymentID, hydratedID, ok := parseGCPTelcoAutomationHydratedDeploymentName(strings.TrimSpace(hydrated.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	updateMask := req.GetUpdateMask()
	if updateMask == nil || len(updateMask.GetPaths()) == 0 {
		return grpcInvalidArgument("update_mask-required")
	}
	for _, field := range updateMask.GetPaths() {
		trimmed := strings.TrimSpace(field)
		if trimmed != "files" && trimmed != "*" {
			return grpcInvalidArgument("update_mask-invalid")
		}
	}
	response := gcpStage4TelcoAutomationHydratedDeployment(project, location, clusterID, deploymentID, hydratedID, telcoautomationpb.HydratedDeployment_DRAFT)
	gcpStage4TelcoAutomationApplyHydratedDeploymentOverrides(response, hydrated)
	return grpcProtoSuccess(response)
}

func gcpStage4GRPCTelcoAutomationApplyHydratedDeployment(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &telcoautomationpb.ApplyHydratedDeploymentRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, clusterID, deploymentID, hydratedID, ok := parseGCPTelcoAutomationHydratedDeploymentName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if strings.Contains(strings.ToLower(hydratedID), "applied") {
		return grpcFailedPrecondition("hydrated-deployment-already-applied")
	}
	return grpcProtoSuccess(gcpStage4TelcoAutomationHydratedDeployment(project, location, clusterID, deploymentID, hydratedID, telcoautomationpb.HydratedDeployment_APPLIED))
}

func gcpStage4TelcoAutomationOrchestrationCluster(project, location, clusterID string, state telcoautomationpb.OrchestrationCluster_State) *telcoautomationpb.OrchestrationCluster {
	return &telcoautomationpb.OrchestrationCluster{
		Name: fmt.Sprintf("projects/%s/locations/%s/orchestrationClusters/%s", project, location, clusterID),
		ManagementConfig: &telcoautomationpb.ManagementConfig{
			OneofConfig: &telcoautomationpb.ManagementConfig_FullManagementConfig{
				FullManagementConfig: &telcoautomationpb.FullManagementConfig{},
			},
		},
		CreateTime: gcpStage4TelcoAutomationTimestamp(),
		UpdateTime: gcpStage4TelcoAutomationTimestamp(),
		Labels: map[string]string{
			"env": "staged",
		},
		TnaVersion: "1.0.0",
		State:      state,
	}
}

func gcpStage4TelcoAutomationEdgeSlm(project, location, edgeID, clusterID string, state telcoautomationpb.EdgeSlm_State) *telcoautomationpb.EdgeSlm {
	return &telcoautomationpb.EdgeSlm{
		Name:                 fmt.Sprintf("projects/%s/locations/%s/edgeSlms/%s", project, location, edgeID),
		OrchestrationCluster: fmt.Sprintf("projects/%s/locations/%s/orchestrationClusters/%s", project, location, clusterID),
		CreateTime:           gcpStage4TelcoAutomationTimestamp(),
		UpdateTime:           gcpStage4TelcoAutomationTimestamp(),
		Labels: map[string]string{
			"env": "staged",
		},
		TnaVersion:          "1.0.0",
		State:               state,
		WorkloadClusterType: telcoautomationpb.EdgeSlm_GKE,
	}
}

func gcpStage4TelcoAutomationBlueprint(project, location, clusterID, blueprintID, revisionID string, state telcoautomationpb.Blueprint_ApprovalState) *telcoautomationpb.Blueprint {
	name := fmt.Sprintf("projects/%s/locations/%s/orchestrationClusters/%s/blueprints/%s", project, location, clusterID, blueprintID)
	if revisionID != "" {
		name += "@" + revisionID
	}
	return &telcoautomationpb.Blueprint{
		Name:               name,
		RevisionId:         revisionID,
		SourceBlueprint:    fmt.Sprintf("projects/%s/locations/%s/publicBlueprints/public-blueprint-1", project, location),
		RevisionCreateTime: gcpStage4TelcoAutomationTimestamp(),
		ApprovalState:      state,
		DisplayName:        "Stackyard Blueprint " + blueprintID,
		Repository:         fmt.Sprintf("projects/%s/locations/%s/orchestrationClusters/%s/blueprints/%s/repository", project, location, clusterID, blueprintID),
		Files: []*telcoautomationpb.File{
			{
				Path:     "deployments/main.yaml",
				Content:  "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: stackyard-blueprint",
				Editable: true,
			},
		},
		Labels: map[string]string{
			"env": "staged",
		},
		CreateTime:      gcpStage4TelcoAutomationTimestamp(),
		UpdateTime:      gcpStage4TelcoAutomationTimestamp(),
		SourceProvider:  "Google",
		DeploymentLevel: telcoautomationpb.DeploymentLevel_SINGLE_DEPLOYMENT,
		RollbackSupport: true,
	}
}

func gcpStage4TelcoAutomationPublicBlueprint(project, location, publicBlueprintID string) *telcoautomationpb.PublicBlueprint {
	return &telcoautomationpb.PublicBlueprint{
		Name:            fmt.Sprintf("projects/%s/locations/%s/publicBlueprints/%s", project, location, publicBlueprintID),
		DisplayName:     "Public Blueprint " + publicBlueprintID,
		Description:     "Stackyard staged public blueprint",
		DeploymentLevel: telcoautomationpb.DeploymentLevel_SINGLE_DEPLOYMENT,
		SourceProvider:  "Google",
		RollbackSupport: true,
	}
}

func gcpStage4TelcoAutomationDeployment(project, location, clusterID, deploymentID, revisionID string, state telcoautomationpb.Deployment_State) *telcoautomationpb.Deployment {
	name := fmt.Sprintf("projects/%s/locations/%s/orchestrationClusters/%s/deployments/%s", project, location, clusterID, deploymentID)
	if revisionID != "" {
		name += "@" + revisionID
	}
	return &telcoautomationpb.Deployment{
		Name:                    name,
		RevisionId:              revisionID,
		SourceBlueprintRevision: fmt.Sprintf("projects/%s/locations/%s/orchestrationClusters/%s/blueprints/blueprint-approved@rev-3", project, location, clusterID),
		RevisionCreateTime:      gcpStage4TelcoAutomationTimestamp(),
		State:                   state,
		DisplayName:             "Stackyard Deployment " + deploymentID,
		Repository:              fmt.Sprintf("projects/%s/locations/%s/orchestrationClusters/%s/deployments/%s/repository", project, location, clusterID, deploymentID),
		Files: []*telcoautomationpb.File{
			{
				Path:     "deployments/workload.yaml",
				Content:  "apiVersion: v1\nkind: Namespace\nmetadata:\n  name: stackyard-workload",
				Editable: true,
			},
		},
		Labels: map[string]string{
			"env": "staged",
		},
		CreateTime:      gcpStage4TelcoAutomationTimestamp(),
		UpdateTime:      gcpStage4TelcoAutomationTimestamp(),
		SourceProvider:  "Google",
		DeploymentLevel: telcoautomationpb.DeploymentLevel_SINGLE_DEPLOYMENT,
		RollbackSupport: true,
		WorkloadCluster: fmt.Sprintf("projects/%s/locations/%s/workloadClusters/workload-1", project, location),
	}
}

func gcpStage4TelcoAutomationHydratedDeployment(project, location, clusterID, deploymentID, hydratedID string, state telcoautomationpb.HydratedDeployment_State) *telcoautomationpb.HydratedDeployment {
	return &telcoautomationpb.HydratedDeployment{
		Name:  fmt.Sprintf("projects/%s/locations/%s/orchestrationClusters/%s/deployments/%s/hydratedDeployments/%s", project, location, clusterID, deploymentID, hydratedID),
		State: state,
		Files: []*telcoautomationpb.File{
			{
				Path:     "hydrated/site.yaml",
				Content:  "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: hydrated-site",
				Editable: true,
			},
		},
		WorkloadCluster: fmt.Sprintf("projects/%s/locations/%s/workloadClusters/workload-1", project, location),
	}
}

func gcpStage4TelcoAutomationComputeDeploymentStatus(name string) *telcoautomationpb.ComputeDeploymentStatusResponse {
	siteVersion := &telcoautomationpb.SiteVersion{
		NfVendor:  "stackyard",
		NfType:    "sample",
		NfVersion: "1.0.0",
	}
	return &telcoautomationpb.ComputeDeploymentStatusResponse{
		Name:             name,
		AggregatedStatus: telcoautomationpb.Status_STATUS_ACTIVE,
		ResourceStatuses: []*telcoautomationpb.ResourceStatus{
			{
				Name:              "nfdeploy-sample",
				ResourceNamespace: "default",
				Group:             "nf.google.com",
				Version:           "v1",
				Kind:              "NFDeploy",
				ResourceType:      telcoautomationpb.ResourceType_NF_DEPLOY_RESOURCE,
				Status:            telcoautomationpb.Status_STATUS_ACTIVE,
				NfDeployStatus: &telcoautomationpb.NFDeployStatus{
					TargetedNfs: 1,
					ReadyNfs:    1,
					Sites: []*telcoautomationpb.NFDeploySiteStatus{
						{
							Site:            "site-1",
							PendingDeletion: false,
							Hydration: &telcoautomationpb.HydrationStatus{
								SiteVersion: siteVersion,
								Status:      "READY",
							},
							Workload: &telcoautomationpb.WorkloadStatus{
								SiteVersion: siteVersion,
								Status:      "READY",
							},
						},
					},
				},
			},
		},
	}
}

func gcpStage4TelcoAutomationOperation(project, location, operationID, verb, target string, response proto.Message) *longrunningpb.Operation {
	if response == nil {
		response = &emptypb.Empty{}
	}
	metadata, _ := anypb.New(&telcoautomationpb.OperationMetadata{
		CreateTime:            gcpStage4TelcoAutomationTimestamp(),
		EndTime:               gcpStage4TelcoAutomationTimestamp(),
		Target:                target,
		Verb:                  verb,
		StatusMessage:         "completed",
		RequestedCancellation: false,
		ApiVersion:            "v1",
	})
	responseAny, _ := anypb.New(response)
	return &longrunningpb.Operation{
		Name:     fmt.Sprintf("projects/%s/locations/%s/operations/%s", project, location, operationID),
		Metadata: metadata,
		Done:     true,
		Result: &longrunningpb.Operation_Response{
			Response: responseAny,
		},
	}
}

func gcpStage4TelcoAutomationTimestamp() *timestamppb.Timestamp {
	return timestamppb.New(gcpTelcoAutomationReferenceTime)
}

func gcpStage4TelcoAutomationApplyOrchestrationClusterOverrides(out, in *telcoautomationpb.OrchestrationCluster) {
	if out == nil || in == nil {
		return
	}
	if in.GetName() != "" {
		out.Name = in.GetName()
	}
	if in.GetManagementConfig() != nil {
		out.ManagementConfig = in.GetManagementConfig()
	}
	if labels := in.GetLabels(); len(labels) > 0 {
		out.Labels = gcpStage4TelcoAutomationCloneLabels(labels)
	}
}

func gcpStage4TelcoAutomationApplyEdgeSlmOverrides(out, in *telcoautomationpb.EdgeSlm) {
	if out == nil || in == nil {
		return
	}
	if in.GetName() != "" {
		out.Name = in.GetName()
	}
	if in.GetOrchestrationCluster() != "" {
		out.OrchestrationCluster = in.GetOrchestrationCluster()
	}
	if labels := in.GetLabels(); len(labels) > 0 {
		out.Labels = gcpStage4TelcoAutomationCloneLabels(labels)
	}
	if in.GetWorkloadClusterType() != telcoautomationpb.EdgeSlm_WORKLOAD_CLUSTER_TYPE_UNSPECIFIED {
		out.WorkloadClusterType = in.GetWorkloadClusterType()
	}
}

func gcpStage4TelcoAutomationApplyBlueprintOverrides(out, in *telcoautomationpb.Blueprint) {
	if out == nil || in == nil {
		return
	}
	if in.GetName() != "" {
		out.Name = in.GetName()
	}
	if in.GetSourceBlueprint() != "" {
		out.SourceBlueprint = in.GetSourceBlueprint()
	}
	if in.GetDisplayName() != "" {
		out.DisplayName = in.GetDisplayName()
	}
	if files := in.GetFiles(); len(files) > 0 {
		out.Files = gcpStage4TelcoAutomationCloneFiles(files)
	}
	if labels := in.GetLabels(); len(labels) > 0 {
		out.Labels = gcpStage4TelcoAutomationCloneLabels(labels)
	}
}

func gcpStage4TelcoAutomationApplyDeploymentOverrides(out, in *telcoautomationpb.Deployment) {
	if out == nil || in == nil {
		return
	}
	if in.GetName() != "" {
		out.Name = in.GetName()
	}
	if in.GetSourceBlueprintRevision() != "" {
		out.SourceBlueprintRevision = in.GetSourceBlueprintRevision()
	}
	if in.GetDisplayName() != "" {
		out.DisplayName = in.GetDisplayName()
	}
	if files := in.GetFiles(); len(files) > 0 {
		out.Files = gcpStage4TelcoAutomationCloneFiles(files)
	}
	if labels := in.GetLabels(); len(labels) > 0 {
		out.Labels = gcpStage4TelcoAutomationCloneLabels(labels)
	}
	if in.GetWorkloadCluster() != "" {
		out.WorkloadCluster = in.GetWorkloadCluster()
	}
}

func gcpStage4TelcoAutomationApplyHydratedDeploymentOverrides(out, in *telcoautomationpb.HydratedDeployment) {
	if out == nil || in == nil {
		return
	}
	if in.GetName() != "" {
		out.Name = in.GetName()
	}
	if files := in.GetFiles(); len(files) > 0 {
		out.Files = gcpStage4TelcoAutomationCloneFiles(files)
	}
}

func gcpStage4TelcoAutomationCloneFiles(files []*telcoautomationpb.File) []*telcoautomationpb.File {
	if len(files) == 0 {
		return nil
	}
	out := make([]*telcoautomationpb.File, 0, len(files))
	for _, file := range files {
		if file == nil {
			continue
		}
		out = append(out, &telcoautomationpb.File{
			Path:     file.GetPath(),
			Content:  file.GetContent(),
			Deleted:  file.GetDeleted(),
			Editable: file.GetEditable(),
		})
	}
	return out
}

func gcpStage4TelcoAutomationCloneLabels(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func gcpStage4TelcoAutomationPageWindow(pageSize int32, pageToken string, max, total int) (start, end int, nextPageToken, reason string, ok bool) {
	if pageSize < 0 || int(pageSize) > max {
		return 0, 0, "", "page_size-invalid", false
	}
	start, valid := parseGCPStage4PageToken(pageToken)
	if !valid {
		return 0, 0, "", "page_token-invalid", false
	}
	if start > total {
		return 0, 0, "", "page_token-out-of-range", false
	}
	end = total
	if pageSize > 0 && start+int(pageSize) < end {
		end = start + int(pageSize)
	}
	nextPageToken = ""
	if end < total {
		nextPageToken = strconv.Itoa(end)
	}
	return start, end, nextPageToken, "", true
}

func gcpStage4TelcoAutomationStateMatchesFilter(filter, state string) bool {
	filter = strings.TrimSpace(filter)
	if filter == "" {
		return true
	}
	state = strings.ToUpper(strings.TrimSpace(state))
	for _, part := range strings.Split(filter, " OR ") {
		candidate := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(part), " ", ""))
		if candidate == "STATE="+state {
			return true
		}
	}
	return false
}

func gcpStage4TelcoAutomationFilterBlueprintRevisionSearch(items []*telcoautomationpb.Blueprint, query string) []*telcoautomationpb.Blueprint {
	query = strings.TrimSpace(query)
	if query == "" || query == "latest=true" {
		return items
	}
	if !strings.HasPrefix(query, "name=") {
		return items
	}
	target := strings.TrimSpace(strings.TrimPrefix(query, "name="))
	latestOnly := false
	if strings.HasSuffix(target, " latest=true") {
		target = strings.TrimSpace(strings.TrimSuffix(target, " latest=true"))
		latestOnly = true
	}
	_, _, _, targetID, _, ok := parseGCPTelcoAutomationBlueprintName(target)
	if !ok {
		return []*telcoautomationpb.Blueprint{}
	}
	filtered := make([]*telcoautomationpb.Blueprint, 0, len(items))
	for _, item := range items {
		_, _, _, itemID, _, ok := parseGCPTelcoAutomationBlueprintName(item.GetName())
		if ok && itemID == targetID {
			filtered = append(filtered, item)
		}
	}
	if !latestOnly || len(filtered) <= 1 {
		return filtered
	}
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].GetRevisionId() < filtered[j].GetRevisionId()
	})
	return filtered[len(filtered)-1:]
}

func gcpStage4TelcoAutomationFilterDeploymentRevisionSearch(items []*telcoautomationpb.Deployment, query string) []*telcoautomationpb.Deployment {
	query = strings.TrimSpace(query)
	if query == "" || query == "latest=true" {
		return items
	}
	if !strings.HasPrefix(query, "name=") {
		return items
	}
	target := strings.TrimSpace(strings.TrimPrefix(query, "name="))
	latestOnly := false
	if strings.HasSuffix(target, " latest=true") {
		target = strings.TrimSpace(strings.TrimSuffix(target, " latest=true"))
		latestOnly = true
	}
	_, _, _, targetID, _, ok := parseGCPTelcoAutomationDeploymentName(target)
	if !ok {
		return []*telcoautomationpb.Deployment{}
	}
	filtered := make([]*telcoautomationpb.Deployment, 0, len(items))
	for _, item := range items {
		_, _, _, itemID, _, ok := parseGCPTelcoAutomationDeploymentName(item.GetName())
		if ok && itemID == targetID {
			filtered = append(filtered, item)
		}
	}
	if !latestOnly || len(filtered) <= 1 {
		return filtered
	}
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].GetRevisionId() < filtered[j].GetRevisionId()
	})
	return filtered[len(filtered)-1:]
}
