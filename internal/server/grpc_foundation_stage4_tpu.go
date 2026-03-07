package server

import (
	"fmt"
	"sort"
	"strings"
	"time"

	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	tpupb "cloud.google.com/go/tpu/apiv1/tpupb"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	gcpTPUListNodesMethod              = "/google.cloud.tpu.v1.Tpu/ListNodes"
	gcpTPUGetNodeMethod                = "/google.cloud.tpu.v1.Tpu/GetNode"
	gcpTPUCreateNodeMethod             = "/google.cloud.tpu.v1.Tpu/CreateNode"
	gcpTPUDeleteNodeMethod             = "/google.cloud.tpu.v1.Tpu/DeleteNode"
	gcpTPUReimageNodeMethod            = "/google.cloud.tpu.v1.Tpu/ReimageNode"
	gcpTPUStopNodeMethod               = "/google.cloud.tpu.v1.Tpu/StopNode"
	gcpTPUStartNodeMethod              = "/google.cloud.tpu.v1.Tpu/StartNode"
	gcpTPUListTensorFlowVersionsMethod = "/google.cloud.tpu.v1.Tpu/ListTensorFlowVersions"
	gcpTPUGetTensorFlowVersionMethod   = "/google.cloud.tpu.v1.Tpu/GetTensorFlowVersion"
	gcpTPUListAcceleratorTypesMethod   = "/google.cloud.tpu.v1.Tpu/ListAcceleratorTypes"
	gcpTPUGetAcceleratorTypeMethod     = "/google.cloud.tpu.v1.Tpu/GetAcceleratorType"
)

func gcpStage4GRPCTPU(path string, grpcReqBody []byte) ([]byte, string, string, bool) {
	switch path {
	case gcpTPUListNodesMethod:
		return gcpStage4GRPCTPUListNodes(grpcReqBody)
	case gcpTPUGetNodeMethod:
		return gcpStage4GRPCTPUGetNode(grpcReqBody)
	case gcpTPUCreateNodeMethod:
		return gcpStage4GRPCTPUCreateNode(grpcReqBody)
	case gcpTPUDeleteNodeMethod:
		return gcpStage4GRPCTPUDeleteNode(grpcReqBody)
	case gcpTPUReimageNodeMethod:
		return gcpStage4GRPCTPUReimageNode(grpcReqBody)
	case gcpTPUStopNodeMethod:
		return gcpStage4GRPCTPUStopNode(grpcReqBody)
	case gcpTPUStartNodeMethod:
		return gcpStage4GRPCTPUStartNode(grpcReqBody)
	case gcpTPUListTensorFlowVersionsMethod:
		return gcpStage4GRPCTPUListTensorFlowVersions(grpcReqBody)
	case gcpTPUGetTensorFlowVersionMethod:
		return gcpStage4GRPCTPUGetTensorFlowVersion(grpcReqBody)
	case gcpTPUListAcceleratorTypesMethod:
		return gcpStage4GRPCTPUListAcceleratorTypes(grpcReqBody)
	case gcpTPUGetAcceleratorTypeMethod:
		return gcpStage4GRPCTPUGetAcceleratorType(grpcReqBody)
	default:
		return grpcUnimplemented("method-not-implemented")
	}
}

func gcpStage4GRPCTPUListNodes(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &tpupb.ListNodesRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := parseGCPTPULocationName(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	start, end, next, reason, ok := gcpStage4TPUPageWindow(req.GetPageSize(), req.GetPageToken(), 1000, 2)
	if !ok {
		return grpcInvalidArgument(reason)
	}

	items := []*tpupb.Node{
		gcpStage4TPUNode(project, location, "node-1", "READY"),
		gcpStage4TPUNode(project, location, "node-stopped", "STOPPED"),
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].GetName() < items[j].GetName()
	})

	return grpcProtoSuccess(&tpupb.ListNodesResponse{
		Nodes:         items[start:end],
		NextPageToken: next,
	})
}

func gcpStage4GRPCTPUGetNode(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &tpupb.GetNodeRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, nodeID, ok := parseGCPTPUNodeName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPTPUMissingID(nodeID) {
		return grpcNotFound("node-not-found")
	}
	return grpcProtoSuccess(gcpStage4TPUNode(project, location, nodeID, gcpTPUStateForNodeID(nodeID)))
}

func gcpStage4GRPCTPUCreateNode(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &tpupb.CreateNodeRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := parseGCPTPULocationName(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	nodeID := strings.TrimSpace(req.GetNodeId())
	if nodeID == "" {
		return grpcInvalidArgument("node_id-required")
	}
	if !gcpTPUNodeIDRegex.MatchString(nodeID) {
		return grpcInvalidArgument("node_id-invalid")
	}
	if strings.Contains(strings.ToLower(nodeID), "existing") {
		return grpcAlreadyExists("node-already-exists")
	}
	if req.GetNode() == nil {
		return grpcInvalidArgument("node-required")
	}
	if strings.TrimSpace(req.GetNode().GetAcceleratorType()) == "" {
		return grpcInvalidArgument("node-accelerator_type-required")
	}
	if strings.TrimSpace(req.GetNode().GetTensorflowVersion()) == "" {
		return grpcInvalidArgument("node-tensorflow_version-required")
	}
	expectedName := gcpTPUNodeName(project, location, nodeID)
	if provided := strings.TrimSpace(req.GetNode().GetName()); provided != "" && provided != expectedName {
		return grpcInvalidArgument("node-name-mismatch")
	}

	response := gcpStage4TPUNode(project, location, nodeID, "READY")
	gcpStage4TPUApplyNodeOverrides(response, req.GetNode())
	return grpcProtoSuccess(gcpStage4TPUOperation(project, location, "createNode."+nodeID, expectedName, "create", response))
}

func gcpStage4GRPCTPUDeleteNode(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &tpupb.DeleteNodeRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, nodeID, ok := parseGCPTPUNodeName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPTPUMissingID(nodeID) {
		return grpcNotFound("node-not-found")
	}
	response := gcpStage4TPUNode(project, location, nodeID, "STOPPED")
	return grpcProtoSuccess(gcpStage4TPUOperation(project, location, "deleteNode."+nodeID, gcpTPUNodeName(project, location, nodeID), "delete", response))
}

func gcpStage4GRPCTPUReimageNode(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &tpupb.ReimageNodeRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, nodeID, ok := parseGCPTPUNodeName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPTPUMissingID(nodeID) {
		return grpcNotFound("node-not-found")
	}
	state := gcpTPUStateForNodeID(nodeID)
	if state == "CREATING" || state == "DELETING" || state == "STARTING" || state == "STOPPING" {
		return grpcFailedPrecondition("node-not-ready-for-reimage")
	}
	version := strings.TrimSpace(req.GetTensorflowVersion())
	if version != "" && !isGCPTPUValidTensorFlowVersionInput(project, location, version) {
		return grpcInvalidArgument("tensorflow_version-invalid")
	}
	response := gcpStage4TPUNode(project, location, nodeID, "READY")
	if version != "" {
		if strings.Contains(version, "/") {
			_, _, versionID, _ := parseGCPTPUTensorFlowVersionName(version)
			response.TensorflowVersion = versionID
		} else {
			response.TensorflowVersion = version
		}
	}
	return grpcProtoSuccess(gcpStage4TPUOperation(project, location, "reimageNode."+nodeID, gcpTPUNodeName(project, location, nodeID), "reimage", response))
}

func gcpStage4GRPCTPUStopNode(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &tpupb.StopNodeRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, nodeID, ok := parseGCPTPUNodeName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPTPUMissingID(nodeID) {
		return grpcNotFound("node-not-found")
	}
	if gcpTPUStateForNodeID(nodeID) != "READY" {
		return grpcFailedPrecondition("node-must-be-ready-to-stop")
	}
	response := gcpStage4TPUNode(project, location, nodeID, "STOPPED")
	return grpcProtoSuccess(gcpStage4TPUOperation(project, location, "stopNode."+nodeID, gcpTPUNodeName(project, location, nodeID), "stop", response))
}

func gcpStage4GRPCTPUStartNode(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &tpupb.StartNodeRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, nodeID, ok := parseGCPTPUNodeName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPTPUMissingID(nodeID) {
		return grpcNotFound("node-not-found")
	}
	if gcpTPUStateForNodeID(nodeID) != "STOPPED" {
		return grpcFailedPrecondition("node-must-be-stopped-to-start")
	}
	response := gcpStage4TPUNode(project, location, nodeID, "READY")
	return grpcProtoSuccess(gcpStage4TPUOperation(project, location, "startNode."+nodeID, gcpTPUNodeName(project, location, nodeID), "start", response))
}

func gcpStage4GRPCTPUListTensorFlowVersions(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &tpupb.ListTensorFlowVersionsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := parseGCPTPULocationName(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	start, end, next, reason, ok := gcpStage4TPUPageWindow(req.GetPageSize(), req.GetPageToken(), 1000, 2)
	if !ok {
		return grpcInvalidArgument(reason)
	}
	items := []*tpupb.TensorFlowVersion{
		{Name: fmt.Sprintf("projects/%s/locations/%s/tensorflowVersions/v2-alpha", project, location), Version: "v2-alpha"},
		{Name: fmt.Sprintf("projects/%s/locations/%s/tensorflowVersions/tpu-vm-tf-2.15.0", project, location), Version: "tpu-vm-tf-2.15.0"},
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].GetName() < items[j].GetName()
	})
	return grpcProtoSuccess(&tpupb.ListTensorFlowVersionsResponse{
		TensorflowVersions: items[start:end],
		NextPageToken:      next,
	})
}

func gcpStage4GRPCTPUGetTensorFlowVersion(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &tpupb.GetTensorFlowVersionRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, versionID, ok := parseGCPTPUTensorFlowVersionName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPTPUMissingID(versionID) {
		return grpcNotFound("tensorflow-version-not-found")
	}
	return grpcProtoSuccess(&tpupb.TensorFlowVersion{
		Name:    fmt.Sprintf("projects/%s/locations/%s/tensorflowVersions/%s", project, location, versionID),
		Version: versionID,
	})
}

func gcpStage4GRPCTPUListAcceleratorTypes(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &tpupb.ListAcceleratorTypesRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := parseGCPTPULocationName(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	start, end, next, reason, ok := gcpStage4TPUPageWindow(req.GetPageSize(), req.GetPageToken(), 1000, 2)
	if !ok {
		return grpcInvalidArgument(reason)
	}
	items := []*tpupb.AcceleratorType{
		{Name: fmt.Sprintf("projects/%s/locations/%s/acceleratorTypes/v3-8", project, location), Type: "v3-8"},
		{Name: fmt.Sprintf("projects/%s/locations/%s/acceleratorTypes/v4-8", project, location), Type: "v4-8"},
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].GetName() < items[j].GetName()
	})
	return grpcProtoSuccess(&tpupb.ListAcceleratorTypesResponse{
		AcceleratorTypes: items[start:end],
		NextPageToken:    next,
	})
}

func gcpStage4GRPCTPUGetAcceleratorType(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &tpupb.GetAcceleratorTypeRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, acceleratorTypeID, ok := parseGCPTPUAcceleratorTypeName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPTPUMissingID(acceleratorTypeID) {
		return grpcNotFound("accelerator-type-not-found")
	}
	return grpcProtoSuccess(&tpupb.AcceleratorType{
		Name: fmt.Sprintf("projects/%s/locations/%s/acceleratorTypes/%s", project, location, acceleratorTypeID),
		Type: acceleratorTypeID,
	})
}

func gcpStage4TPUPageWindow(pageSize int32, pageToken string, max, total int) (start, end int, nextPageToken, reason string, ok bool) {
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
	nextPageToken = ""
	if end < total {
		nextPageToken = fmt.Sprintf("%d", end)
	}
	return start, end, nextPageToken, "", true
}

func gcpStage4TPUNode(project, location, nodeID, state string) *tpupb.Node {
	health := tpupb.Node_HEALTHY
	if strings.Contains(strings.ToLower(nodeID), "unhealthy") {
		health = tpupb.Node_TIMEOUT
	}
	port := int32(8470)
	if state == "STOPPED" {
		port = 0
	}
	return &tpupb.Node{
		Name:              gcpTPUNodeName(project, location, nodeID),
		Description:       "Stackyard TPU node " + nodeID,
		AcceleratorType:   "v3-8",
		TensorflowVersion: "v2-alpha",
		Network:           fmt.Sprintf("projects/%s/global/networks/default", project),
		CidrBlock:         "10.240.0.0/29",
		ServiceAccount:    "stackyard-tpu@" + project + ".iam.gserviceaccount.com",
		CreateTime:        timestamppb.New(gcpStage4ReferenceTime),
		State:             gcpStage4TPUStateEnum(state),
		Health:            health,
		Labels: map[string]string{
			"env":   "staged",
			"owner": "stackyard",
		},
		NetworkEndpoints: []*tpupb.NetworkEndpoint{
			{
				IpAddress: "10.240.0.2",
				Port:      port,
			},
		},
	}
}

func gcpStage4TPUStateEnum(state string) tpupb.Node_State {
	switch strings.ToUpper(strings.TrimSpace(state)) {
	case "CREATING":
		return tpupb.Node_CREATING
	case "DELETING":
		return tpupb.Node_DELETING
	case "STARTING":
		return tpupb.Node_STARTING
	case "STOPPING":
		return tpupb.Node_STOPPING
	case "STOPPED":
		return tpupb.Node_STOPPED
	default:
		return tpupb.Node_READY
	}
}

func gcpStage4TPUApplyNodeOverrides(dst, src *tpupb.Node) {
	if dst == nil || src == nil {
		return
	}
	if description := strings.TrimSpace(src.GetDescription()); description != "" {
		dst.Description = description
	}
	if acceleratorType := strings.TrimSpace(src.GetAcceleratorType()); acceleratorType != "" {
		dst.AcceleratorType = acceleratorType
	}
	if tensorflowVersion := strings.TrimSpace(src.GetTensorflowVersion()); tensorflowVersion != "" {
		dst.TensorflowVersion = tensorflowVersion
	}
	if len(src.GetLabels()) > 0 {
		labels := map[string]string{}
		for key, value := range src.GetLabels() {
			key = strings.TrimSpace(key)
			value = strings.TrimSpace(value)
			if key != "" && value != "" {
				labels[key] = value
			}
		}
		if len(labels) > 0 {
			dst.Labels = labels
		}
	}
}

func gcpStage4TPUOperation(project, location, operationID, target, verb string, response *tpupb.Node) *longrunningpb.Operation {
	metadataAny, err := anypb.New(&tpupb.OperationMetadata{
		CreateTime:      timestamppb.New(gcpStage4ReferenceTime),
		EndTime:         timestamppb.New(gcpStage4ReferenceTime.Add(2 * time.Second)),
		Target:          target,
		Verb:            verb,
		StatusDetail:    "completed",
		CancelRequested: false,
		ApiVersion:      "v1",
	})
	if err != nil {
		metadataAny = nil
	}

	if response == nil {
		response = &tpupb.Node{Name: target}
	}
	responseAny, err := anypb.New(response)
	if err != nil {
		responseAny = nil
	}

	out := &longrunningpb.Operation{
		Name: fmt.Sprintf("projects/%s/locations/%s/operations/%s", project, location, operationID),
		Done: true,
	}
	if metadataAny != nil {
		out.Metadata = metadataAny
	}
	if responseAny != nil {
		out.Result = &longrunningpb.Operation_Response{Response: responseAny}
	}
	return out
}
