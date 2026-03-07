package server

import (
	"strconv"
	"strings"
	"time"

	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	workstationspb "cloud.google.com/go/workstations/apiv1/workstationspb"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	gcpWorkstationsListWorkstationClustersMethod       = "/google.cloud.workstations.v1.Workstations/ListWorkstationClusters"
	gcpWorkstationsGetWorkstationClusterMethod         = "/google.cloud.workstations.v1.Workstations/GetWorkstationCluster"
	gcpWorkstationsCreateWorkstationClusterMethod      = "/google.cloud.workstations.v1.Workstations/CreateWorkstationCluster"
	gcpWorkstationsUpdateWorkstationClusterMethod      = "/google.cloud.workstations.v1.Workstations/UpdateWorkstationCluster"
	gcpWorkstationsDeleteWorkstationClusterMethod      = "/google.cloud.workstations.v1.Workstations/DeleteWorkstationCluster"
	gcpWorkstationsListWorkstationConfigsMethod        = "/google.cloud.workstations.v1.Workstations/ListWorkstationConfigs"
	gcpWorkstationsListUsableWorkstationConfigsMethod  = "/google.cloud.workstations.v1.Workstations/ListUsableWorkstationConfigs"
	gcpWorkstationsGetWorkstationConfigMethod          = "/google.cloud.workstations.v1.Workstations/GetWorkstationConfig"
	gcpWorkstationsCreateWorkstationConfigMethod       = "/google.cloud.workstations.v1.Workstations/CreateWorkstationConfig"
	gcpWorkstationsUpdateWorkstationConfigMethod       = "/google.cloud.workstations.v1.Workstations/UpdateWorkstationConfig"
	gcpWorkstationsDeleteWorkstationConfigMethod       = "/google.cloud.workstations.v1.Workstations/DeleteWorkstationConfig"
	gcpWorkstationsListWorkstationsMethod              = "/google.cloud.workstations.v1.Workstations/ListWorkstations"
	gcpWorkstationsListUsableWorkstationsMethod        = "/google.cloud.workstations.v1.Workstations/ListUsableWorkstations"
	gcpWorkstationsGetWorkstationMethod                = "/google.cloud.workstations.v1.Workstations/GetWorkstation"
	gcpWorkstationsCreateWorkstationMethod             = "/google.cloud.workstations.v1.Workstations/CreateWorkstation"
	gcpWorkstationsUpdateWorkstationMethod             = "/google.cloud.workstations.v1.Workstations/UpdateWorkstation"
	gcpWorkstationsDeleteWorkstationMethod             = "/google.cloud.workstations.v1.Workstations/DeleteWorkstation"
	gcpWorkstationsStartWorkstationMethod              = "/google.cloud.workstations.v1.Workstations/StartWorkstation"
	gcpWorkstationsStopWorkstationMethod               = "/google.cloud.workstations.v1.Workstations/StopWorkstation"
	gcpWorkstationsGenerateAccessTokenMethod           = "/google.cloud.workstations.v1.Workstations/GenerateAccessToken"
	gcpStage4WorkstationsMaxListPageSize               = 1000
	gcpStage4WorkstationsDefaultAccessTokenExpiryHours = 1
	gcpStage4WorkstationsMaxAccessTokenExpiryHours     = 24
)

func gcpStage4GRPCWorkstations(path string, grpcReqBody []byte) ([]byte, string, string, bool) {
	switch path {
	case gcpWorkstationsListWorkstationClustersMethod:
		return gcpStage4GRPCWorkstationsListWorkstationClusters(grpcReqBody)
	case gcpWorkstationsGetWorkstationClusterMethod:
		return gcpStage4GRPCWorkstationsGetWorkstationCluster(grpcReqBody)
	case gcpWorkstationsCreateWorkstationClusterMethod:
		return gcpStage4GRPCWorkstationsCreateWorkstationCluster(grpcReqBody)
	case gcpWorkstationsUpdateWorkstationClusterMethod:
		return gcpStage4GRPCWorkstationsUpdateWorkstationCluster(grpcReqBody)
	case gcpWorkstationsDeleteWorkstationClusterMethod:
		return gcpStage4GRPCWorkstationsDeleteWorkstationCluster(grpcReqBody)
	case gcpWorkstationsListWorkstationConfigsMethod:
		return gcpStage4GRPCWorkstationsListWorkstationConfigs(grpcReqBody)
	case gcpWorkstationsListUsableWorkstationConfigsMethod:
		return gcpStage4GRPCWorkstationsListUsableWorkstationConfigs(grpcReqBody)
	case gcpWorkstationsGetWorkstationConfigMethod:
		return gcpStage4GRPCWorkstationsGetWorkstationConfig(grpcReqBody)
	case gcpWorkstationsCreateWorkstationConfigMethod:
		return gcpStage4GRPCWorkstationsCreateWorkstationConfig(grpcReqBody)
	case gcpWorkstationsUpdateWorkstationConfigMethod:
		return gcpStage4GRPCWorkstationsUpdateWorkstationConfig(grpcReqBody)
	case gcpWorkstationsDeleteWorkstationConfigMethod:
		return gcpStage4GRPCWorkstationsDeleteWorkstationConfig(grpcReqBody)
	case gcpWorkstationsListWorkstationsMethod:
		return gcpStage4GRPCWorkstationsListWorkstations(grpcReqBody)
	case gcpWorkstationsListUsableWorkstationsMethod:
		return gcpStage4GRPCWorkstationsListUsableWorkstations(grpcReqBody)
	case gcpWorkstationsGetWorkstationMethod:
		return gcpStage4GRPCWorkstationsGetWorkstation(grpcReqBody)
	case gcpWorkstationsCreateWorkstationMethod:
		return gcpStage4GRPCWorkstationsCreateWorkstation(grpcReqBody)
	case gcpWorkstationsUpdateWorkstationMethod:
		return gcpStage4GRPCWorkstationsUpdateWorkstation(grpcReqBody)
	case gcpWorkstationsDeleteWorkstationMethod:
		return gcpStage4GRPCWorkstationsDeleteWorkstation(grpcReqBody)
	case gcpWorkstationsStartWorkstationMethod:
		return gcpStage4GRPCWorkstationsStartWorkstation(grpcReqBody)
	case gcpWorkstationsStopWorkstationMethod:
		return gcpStage4GRPCWorkstationsStopWorkstation(grpcReqBody)
	case gcpWorkstationsGenerateAccessTokenMethod:
		return gcpStage4GRPCWorkstationsGenerateAccessToken(grpcReqBody)
	default:
		return grpcUnimplemented("method-not-implemented")
	}
}

func gcpStage4GRPCWorkstationsListWorkstationClusters(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &workstationspb.ListWorkstationClustersRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := gcpStage4ParseWorkstationsLocationParent(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	start, end, next, reason, ok := gcpStage4WorkstationsPageWindow(req.GetPageSize(), req.GetPageToken(), gcpStage4WorkstationsMaxListPageSize, 2)
	if !ok {
		return grpcInvalidArgument(reason)
	}

	items := []*workstationspb.WorkstationCluster{
		gcpStage4WorkstationsCluster(project, location, "cluster-1"),
		gcpStage4WorkstationsCluster(project, location, "cluster-2"),
	}
	return grpcProtoSuccess(&workstationspb.ListWorkstationClustersResponse{
		WorkstationClusters: items[start:end],
		NextPageToken:       next,
		Unreachable:         []string{},
	})
}

func gcpStage4GRPCWorkstationsGetWorkstationCluster(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &workstationspb.GetWorkstationClusterRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, clusterID, ok := parseGCPWorkstationsClusterName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if gcpStage4WorkstationsIsMissingID(clusterID) {
		return grpcNotFound("workstation-cluster-not-found")
	}
	return grpcProtoSuccess(gcpStage4WorkstationsCluster(project, location, clusterID))
}

func gcpStage4GRPCWorkstationsCreateWorkstationCluster(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &workstationspb.CreateWorkstationClusterRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := gcpStage4ParseWorkstationsLocationParent(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	clusterID := strings.TrimSpace(req.GetWorkstationClusterId())
	if clusterID == "" {
		return grpcInvalidArgument("workstation_cluster_id-required")
	}
	if !gcpWorkstationsIDPattern.MatchString(clusterID) {
		return grpcInvalidArgument("workstation_cluster_id-invalid")
	}
	cluster := req.GetWorkstationCluster()
	if cluster == nil {
		return grpcInvalidArgument("workstation_cluster-required")
	}
	expectedName := gcpWorkstationsClusterName(project, location, clusterID)
	if name := strings.TrimSpace(cluster.GetName()); name != "" && name != expectedName {
		return grpcInvalidArgument("workstation_cluster-name-mismatch")
	}
	if strings.TrimSpace(cluster.GetNetwork()) == "" {
		return grpcInvalidArgument("workstation_cluster-network-required")
	}
	if strings.Contains(strings.ToLower(clusterID), "existing") {
		return grpcAlreadyExists("workstation-cluster-already-exists")
	}
	return grpcProtoSuccess(gcpStage4WorkstationsOperation(project, location, "createWorkstationCluster."+clusterID, expectedName, "create", false))
}

func gcpStage4GRPCWorkstationsUpdateWorkstationCluster(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &workstationspb.UpdateWorkstationClusterRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	cluster := req.GetWorkstationCluster()
	if cluster == nil {
		return grpcInvalidArgument("workstation_cluster-required")
	}
	project, location, clusterID, ok := parseGCPWorkstationsClusterName(strings.TrimSpace(cluster.GetName()))
	if !ok {
		return grpcInvalidArgument("workstation_cluster-name-required")
	}
	if !gcpStage4WorkstationsValidUpdateMask(req.GetUpdateMask()) {
		return grpcInvalidArgument("update_mask-invalid")
	}
	expectedName := gcpWorkstationsClusterName(project, location, clusterID)
	return grpcProtoSuccess(gcpStage4WorkstationsOperation(project, location, "updateWorkstationCluster."+clusterID, expectedName, "update", false))
}

func gcpStage4GRPCWorkstationsDeleteWorkstationCluster(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &workstationspb.DeleteWorkstationClusterRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, clusterID, ok := parseGCPWorkstationsClusterName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if !req.GetForce() && strings.Contains(strings.ToLower(clusterID), "inuse") {
		return grpcFailedPrecondition("workstation-cluster-not-empty")
	}
	return grpcProtoSuccess(gcpStage4WorkstationsOperation(project, location, "deleteWorkstationCluster."+clusterID, gcpWorkstationsClusterName(project, location, clusterID), "delete", false))
}

func gcpStage4GRPCWorkstationsListWorkstationConfigs(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &workstationspb.ListWorkstationConfigsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, clusterID, ok := gcpStage4ParseWorkstationsConfigParent(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	start, end, next, reason, ok := gcpStage4WorkstationsPageWindow(req.GetPageSize(), req.GetPageToken(), gcpStage4WorkstationsMaxListPageSize, 2)
	if !ok {
		return grpcInvalidArgument(reason)
	}

	items := []*workstationspb.WorkstationConfig{
		gcpStage4WorkstationsConfig(project, location, clusterID, "config-1"),
		gcpStage4WorkstationsConfig(project, location, clusterID, "config-2"),
	}
	return grpcProtoSuccess(&workstationspb.ListWorkstationConfigsResponse{
		WorkstationConfigs: items[start:end],
		NextPageToken:      next,
		Unreachable:        []string{},
	})
}

func gcpStage4GRPCWorkstationsListUsableWorkstationConfigs(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &workstationspb.ListUsableWorkstationConfigsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, clusterID, ok := gcpStage4ParseWorkstationsConfigParent(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	start, end, next, reason, ok := gcpStage4WorkstationsPageWindow(req.GetPageSize(), req.GetPageToken(), gcpStage4WorkstationsMaxListPageSize, 1)
	if !ok {
		return grpcInvalidArgument(reason)
	}

	items := []*workstationspb.WorkstationConfig{
		gcpStage4WorkstationsConfig(project, location, clusterID, "config-1"),
	}
	return grpcProtoSuccess(&workstationspb.ListUsableWorkstationConfigsResponse{
		WorkstationConfigs: items[start:end],
		NextPageToken:      next,
		Unreachable:        []string{},
	})
}

func gcpStage4GRPCWorkstationsGetWorkstationConfig(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &workstationspb.GetWorkstationConfigRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, clusterID, configID, ok := parseGCPWorkstationsConfigName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if gcpStage4WorkstationsIsMissingID(configID) {
		return grpcNotFound("workstation-config-not-found")
	}
	return grpcProtoSuccess(gcpStage4WorkstationsConfig(project, location, clusterID, configID))
}

func gcpStage4GRPCWorkstationsCreateWorkstationConfig(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &workstationspb.CreateWorkstationConfigRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, clusterID, ok := gcpStage4ParseWorkstationsConfigParent(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	configID := strings.TrimSpace(req.GetWorkstationConfigId())
	if configID == "" {
		return grpcInvalidArgument("workstation_config_id-required")
	}
	if !gcpWorkstationsIDPattern.MatchString(configID) {
		return grpcInvalidArgument("workstation_config_id-invalid")
	}
	config := req.GetWorkstationConfig()
	if config == nil {
		return grpcInvalidArgument("workstation_config-required")
	}
	expectedName := gcpWorkstationsConfigName(project, location, clusterID, configID)
	if name := strings.TrimSpace(config.GetName()); name != "" && name != expectedName {
		return grpcInvalidArgument("workstation_config-name-mismatch")
	}
	if config.GetHost() == nil {
		return grpcInvalidArgument("workstation_config-host-required")
	}
	if strings.Contains(strings.ToLower(configID), "existing") {
		return grpcAlreadyExists("workstation-config-already-exists")
	}
	return grpcProtoSuccess(gcpStage4WorkstationsOperation(project, location, "createWorkstationConfig."+configID, expectedName, "create", false))
}

func gcpStage4GRPCWorkstationsUpdateWorkstationConfig(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &workstationspb.UpdateWorkstationConfigRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	config := req.GetWorkstationConfig()
	if config == nil {
		return grpcInvalidArgument("workstation_config-required")
	}
	project, location, clusterID, configID, ok := parseGCPWorkstationsConfigName(strings.TrimSpace(config.GetName()))
	if !ok {
		return grpcInvalidArgument("workstation_config-name-required")
	}
	if !gcpStage4WorkstationsValidUpdateMask(req.GetUpdateMask()) {
		return grpcInvalidArgument("update_mask-invalid")
	}
	return grpcProtoSuccess(gcpStage4WorkstationsOperation(project, location, "updateWorkstationConfig."+configID, gcpWorkstationsConfigName(project, location, clusterID, configID), "update", false))
}

func gcpStage4GRPCWorkstationsDeleteWorkstationConfig(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &workstationspb.DeleteWorkstationConfigRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, clusterID, configID, ok := parseGCPWorkstationsConfigName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if !req.GetForce() && strings.Contains(strings.ToLower(configID), "inuse") {
		return grpcFailedPrecondition("workstation-config-not-empty")
	}
	return grpcProtoSuccess(gcpStage4WorkstationsOperation(project, location, "deleteWorkstationConfig."+configID, gcpWorkstationsConfigName(project, location, clusterID, configID), "delete", false))
}

func gcpStage4GRPCWorkstationsListWorkstations(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &workstationspb.ListWorkstationsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, clusterID, configID, ok := gcpStage4ParseWorkstationsWorkstationParent(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	start, end, next, reason, ok := gcpStage4WorkstationsPageWindow(req.GetPageSize(), req.GetPageToken(), gcpStage4WorkstationsMaxListPageSize, 2)
	if !ok {
		return grpcInvalidArgument(reason)
	}

	items := []*workstationspb.Workstation{
		gcpStage4WorkstationsWorkstation(project, location, clusterID, configID, "workstation-running"),
		gcpStage4WorkstationsWorkstation(project, location, clusterID, configID, "workstation-stopped"),
	}
	return grpcProtoSuccess(&workstationspb.ListWorkstationsResponse{
		Workstations:  items[start:end],
		NextPageToken: next,
		Unreachable:   []string{},
	})
}

func gcpStage4GRPCWorkstationsListUsableWorkstations(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &workstationspb.ListUsableWorkstationsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, clusterID, configID, ok := gcpStage4ParseWorkstationsWorkstationParent(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	start, end, next, reason, ok := gcpStage4WorkstationsPageWindow(req.GetPageSize(), req.GetPageToken(), gcpStage4WorkstationsMaxListPageSize, 1)
	if !ok {
		return grpcInvalidArgument(reason)
	}

	items := []*workstationspb.Workstation{
		gcpStage4WorkstationsWorkstation(project, location, clusterID, configID, "workstation-running"),
	}
	return grpcProtoSuccess(&workstationspb.ListUsableWorkstationsResponse{
		Workstations:  items[start:end],
		NextPageToken: next,
		Unreachable:   []string{},
	})
}

func gcpStage4GRPCWorkstationsGetWorkstation(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &workstationspb.GetWorkstationRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, clusterID, configID, workstationID, ok := parseGCPWorkstationsWorkstationName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if gcpStage4WorkstationsIsMissingID(workstationID) {
		return grpcNotFound("workstation-not-found")
	}
	return grpcProtoSuccess(gcpStage4WorkstationsWorkstation(project, location, clusterID, configID, workstationID))
}

func gcpStage4GRPCWorkstationsCreateWorkstation(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &workstationspb.CreateWorkstationRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, clusterID, configID, ok := gcpStage4ParseWorkstationsWorkstationParent(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	workstationID := strings.TrimSpace(req.GetWorkstationId())
	if workstationID == "" {
		return grpcInvalidArgument("workstation_id-required")
	}
	if !gcpWorkstationsIDPattern.MatchString(workstationID) {
		return grpcInvalidArgument("workstation_id-invalid")
	}
	workstation := req.GetWorkstation()
	if workstation == nil {
		return grpcInvalidArgument("workstation-required")
	}
	expectedName := gcpWorkstationsWorkstationName(project, location, clusterID, configID, workstationID)
	if name := strings.TrimSpace(workstation.GetName()); name != "" && name != expectedName {
		return grpcInvalidArgument("workstation-name-mismatch")
	}
	if strings.Contains(strings.ToLower(workstationID), "existing") {
		return grpcAlreadyExists("workstation-already-exists")
	}
	return grpcProtoSuccess(gcpStage4WorkstationsOperation(project, location, "createWorkstation."+workstationID, expectedName, "create", false))
}

func gcpStage4GRPCWorkstationsUpdateWorkstation(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &workstationspb.UpdateWorkstationRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	workstation := req.GetWorkstation()
	if workstation == nil {
		return grpcInvalidArgument("workstation-required")
	}
	project, location, clusterID, configID, workstationID, ok := parseGCPWorkstationsWorkstationName(strings.TrimSpace(workstation.GetName()))
	if !ok {
		return grpcInvalidArgument("workstation-name-required")
	}
	if !gcpStage4WorkstationsValidUpdateMask(req.GetUpdateMask()) {
		return grpcInvalidArgument("update_mask-invalid")
	}
	return grpcProtoSuccess(gcpStage4WorkstationsOperation(project, location, "updateWorkstation."+workstationID, gcpWorkstationsWorkstationName(project, location, clusterID, configID, workstationID), "update", false))
}

func gcpStage4GRPCWorkstationsDeleteWorkstation(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &workstationspb.DeleteWorkstationRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, clusterID, configID, workstationID, ok := parseGCPWorkstationsWorkstationName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(gcpStage4WorkstationsOperation(project, location, "deleteWorkstation."+workstationID, gcpWorkstationsWorkstationName(project, location, clusterID, configID, workstationID), "delete", false))
}

func gcpStage4GRPCWorkstationsStartWorkstation(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &workstationspb.StartWorkstationRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, clusterID, configID, workstationID, ok := parseGCPWorkstationsWorkstationName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	state := gcpStage4WorkstationsStateForID(workstationID)
	if state == workstationspb.Workstation_STATE_RUNNING || state == workstationspb.Workstation_STATE_STARTING {
		return grpcFailedPrecondition("workstation-must-be-stopped-before-start")
	}
	return grpcProtoSuccess(gcpStage4WorkstationsOperation(project, location, "startWorkstation."+workstationID, gcpWorkstationsWorkstationName(project, location, clusterID, configID, workstationID), "start", false))
}

func gcpStage4GRPCWorkstationsStopWorkstation(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &workstationspb.StopWorkstationRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, clusterID, configID, workstationID, ok := parseGCPWorkstationsWorkstationName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	state := gcpStage4WorkstationsStateForID(workstationID)
	if state == workstationspb.Workstation_STATE_STOPPED || state == workstationspb.Workstation_STATE_STOPPING {
		return grpcFailedPrecondition("workstation-must-be-running-before-stop")
	}
	return grpcProtoSuccess(gcpStage4WorkstationsOperation(project, location, "stopWorkstation."+workstationID, gcpWorkstationsWorkstationName(project, location, clusterID, configID, workstationID), "stop", false))
}

func gcpStage4GRPCWorkstationsGenerateAccessToken(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &workstationspb.GenerateAccessTokenRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, clusterID, configID, workstationID, ok := parseGCPWorkstationsWorkstationName(strings.TrimSpace(req.GetWorkstation()))
	if !ok {
		return grpcInvalidArgument("workstation-required")
	}
	state := gcpStage4WorkstationsStateForID(workstationID)
	if state != workstationspb.Workstation_STATE_RUNNING {
		return grpcFailedPrecondition("workstation-must-be-running-to-generate-access-token")
	}
	expireTime, valid := gcpStage4WorkstationsResolveTokenExpiry(req)
	if !valid {
		return grpcInvalidArgument("access_token_expiration-invalid")
	}
	_ = project
	_ = location
	_ = clusterID
	_ = configID
	return grpcProtoSuccess(&workstationspb.GenerateAccessTokenResponse{
		AccessToken: "ya29.stackyard-workstations-" + workstationID,
		ExpireTime:  expireTime,
	})
}

func gcpStage4WorkstationsCluster(project, location, clusterID string) *workstationspb.WorkstationCluster {
	return &workstationspb.WorkstationCluster{
		Name:           gcpWorkstationsClusterName(project, location, clusterID),
		DisplayName:    strings.ToUpper(clusterID[:1]) + clusterID[1:],
		Uid:            "wsc-" + clusterID,
		Reconciling:    false,
		Annotations:    map[string]string{"stackyard.dev/stage": "staged"},
		Labels:         map[string]string{"env": "staged"},
		CreateTime:     timestamppb.New(gcpWorkstationsReferenceTime),
		UpdateTime:     timestamppb.New(gcpWorkstationsReferenceTime.Add(2 * time.Minute)),
		Etag:           "etag-" + clusterID,
		Network:        "projects/" + project + "/global/networks/default",
		Subnetwork:     "projects/" + project + "/regions/" + location + "/subnetworks/default",
		ControlPlaneIp: "10.10.0.10",
		Degraded:       false,
		Conditions:     nil,
	}
}

func gcpStage4WorkstationsConfig(project, location, clusterID, configID string) *workstationspb.WorkstationConfig {
	return &workstationspb.WorkstationConfig{
		Name:           gcpWorkstationsConfigName(project, location, clusterID, configID),
		DisplayName:    "Config " + configID,
		Uid:            "wscfg-" + configID,
		Reconciling:    false,
		Annotations:    map[string]string{"stackyard.dev/stage": "staged"},
		Labels:         map[string]string{"env": "staged"},
		CreateTime:     timestamppb.New(gcpWorkstationsReferenceTime.Add(5 * time.Minute)),
		UpdateTime:     timestamppb.New(gcpWorkstationsReferenceTime.Add(6 * time.Minute)),
		Etag:           "etag-" + configID,
		IdleTimeout:    durationpb.New(1200 * time.Second),
		RunningTimeout: durationpb.New(43200 * time.Second),
		Host: &workstationspb.WorkstationConfig_Host{
			Config: &workstationspb.WorkstationConfig_Host_GceInstance_{
				GceInstance: &workstationspb.WorkstationConfig_Host_GceInstance{
					MachineType:    "e2-standard-4",
					PoolSize:       1,
					BootDiskSizeGb: 50,
				},
			},
		},
		Container: &workstationspb.WorkstationConfig_Container{
			Image:      "us-docker.pkg.dev/cloud-workstations-images/predefined/code-oss:latest",
			WorkingDir: "/home/workstation",
		},
		Degraded:   false,
		Conditions: nil,
	}
}

func gcpStage4WorkstationsWorkstation(project, location, clusterID, configID, workstationID string) *workstationspb.Workstation {
	return &workstationspb.Workstation{
		Name:        gcpWorkstationsWorkstationName(project, location, clusterID, configID, workstationID),
		DisplayName: "Workstation " + workstationID,
		Uid:         "ws-" + workstationID,
		Reconciling: false,
		Annotations: map[string]string{"stackyard.dev/stage": "staged"},
		Labels:      map[string]string{"env": "staged"},
		CreateTime:  timestamppb.New(gcpWorkstationsReferenceTime.Add(10 * time.Minute)),
		UpdateTime:  timestamppb.New(gcpWorkstationsReferenceTime.Add(12 * time.Minute)),
		StartTime:   timestamppb.New(gcpWorkstationsReferenceTime.Add(15 * time.Minute)),
		Etag:        "etag-" + workstationID,
		State:       gcpStage4WorkstationsStateForID(workstationID),
		Host:        workstationID + ".ws.stackyard.local",
	}
}

func gcpStage4WorkstationsOperation(project, location, operationID, target, verb string, done bool) *longrunningpb.Operation {
	meta, _ := anypb.New(&workstationspb.OperationMetadata{
		CreateTime:            timestamppb.New(gcpWorkstationsReferenceTime.Add(20 * time.Minute)),
		EndTime:               timestamppb.New(gcpWorkstationsReferenceTime.Add(21 * time.Minute)),
		Target:                target,
		Verb:                  verb,
		StatusMessage:         "staged",
		RequestedCancellation: false,
		ApiVersion:            "v1",
	})
	response, _ := anypb.New(&emptypb.Empty{})
	return &longrunningpb.Operation{
		Name:     gcpWorkstationsOperationName(project, location, operationID),
		Metadata: meta,
		Done:     done,
		Result: &longrunningpb.Operation_Response{
			Response: response,
		},
	}
}

func gcpStage4ParseWorkstationsLocationParent(parent string) (project, location string, ok bool) {
	parts := strings.Split(strings.Trim(parent, "/"), "/")
	if len(parts) != 4 || parts[0] != "projects" || parts[2] != "locations" {
		return "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	if project == "" || location == "" {
		return "", "", false
	}
	return project, location, true
}

func gcpStage4ParseWorkstationsConfigParent(parent string) (project, location, clusterID string, ok bool) {
	project, location, clusterID, ok = parseGCPWorkstationsClusterName(parent)
	return project, location, clusterID, ok
}

func gcpStage4ParseWorkstationsWorkstationParent(parent string) (project, location, clusterID, configID string, ok bool) {
	project, location, clusterID, configID, ok = parseGCPWorkstationsConfigName(parent)
	return project, location, clusterID, configID, ok
}

func gcpStage4WorkstationsPageWindow(pageSize int32, pageToken string, max, total int) (start, end int, nextPageToken, reason string, ok bool) {
	if pageSize < 0 {
		return 0, 0, "", "page_size-invalid", false
	}
	if pageSize > int32(max) {
		return 0, 0, "", "page_size-invalid", false
	}
	start, ok = parseGCPStage4PageToken(pageToken)
	if !ok {
		return 0, 0, "", "page_token-invalid", false
	}
	if start > total {
		return 0, 0, "", "page_token-out-of-range", false
	}

	end = total
	if pageSize > 0 && start+int(pageSize) < end {
		end = start + int(pageSize)
	}
	if end < total {
		nextPageToken = strconv.Itoa(end)
	}
	return start, end, nextPageToken, "", true
}

func gcpStage4WorkstationsValidUpdateMask(mask any) bool {
	switch m := mask.(type) {
	case interface{ GetPaths() []string }:
		paths := m.GetPaths()
		if len(paths) == 0 {
			return false
		}
		joined := strings.Join(paths, ",")
		parsed, ok := parseGCPWorkstationsUpdateMask(joined)
		return ok && len(parsed) > 0
	default:
		return false
	}
}

func gcpStage4WorkstationsStateForID(workstationID string) workstationspb.Workstation_State {
	switch gcpWorkstationsStateForID(workstationID) {
	case "STATE_STARTING":
		return workstationspb.Workstation_STATE_STARTING
	case "STATE_RUNNING":
		return workstationspb.Workstation_STATE_RUNNING
	case "STATE_STOPPING":
		return workstationspb.Workstation_STATE_STOPPING
	case "STATE_STOPPED":
		return workstationspb.Workstation_STATE_STOPPED
	default:
		return workstationspb.Workstation_STATE_UNSPECIFIED
	}
}

func gcpStage4WorkstationsResolveTokenExpiry(req *workstationspb.GenerateAccessTokenRequest) (*timestamppb.Timestamp, bool) {
	now := gcpWorkstationsReferenceTime
	expiry := now.Add(gcpStage4WorkstationsDefaultAccessTokenExpiryHours * time.Hour)

	if expireAt := req.GetExpireTime(); expireAt != nil {
		t := expireAt.AsTime()
		if t.Before(now) || t.After(now.Add(gcpStage4WorkstationsMaxAccessTokenExpiryHours*time.Hour)) {
			return nil, false
		}
		expiry = t
	}
	if ttl := req.GetTtl(); ttl != nil {
		d := ttl.AsDuration()
		if d <= 0 || d > gcpStage4WorkstationsMaxAccessTokenExpiryHours*time.Hour {
			return nil, false
		}
		expiry = now.Add(d)
	}
	return timestamppb.New(expiry), true
}

func gcpStage4WorkstationsIsMissingID(id string) bool {
	return strings.Contains(strings.ToLower(strings.TrimSpace(id)), "missing")
}
