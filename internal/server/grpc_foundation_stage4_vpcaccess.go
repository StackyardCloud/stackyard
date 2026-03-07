package server

import (
	"fmt"
	"strconv"
	"strings"

	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	vpcaccesspb "cloud.google.com/go/vpcaccess/apiv1/vpcaccesspb"
)

const (
	gcpVPCAccessListConnectorsMethod  = "/google.cloud.vpcaccess.v1.VpcAccessService/ListConnectors"
	gcpVPCAccessGetConnectorMethod    = "/google.cloud.vpcaccess.v1.VpcAccessService/GetConnector"
	gcpVPCAccessCreateConnectorMethod = "/google.cloud.vpcaccess.v1.VpcAccessService/CreateConnector"
	gcpVPCAccessDeleteConnectorMethod = "/google.cloud.vpcaccess.v1.VpcAccessService/DeleteConnector"
)

func gcpStage4GRPCVPCAccess(path string, grpcReqBody []byte) ([]byte, string, string, bool) {
	switch path {
	case gcpVPCAccessListConnectorsMethod:
		return gcpStage4GRPCVPCAccessListConnectors(grpcReqBody)
	case gcpVPCAccessGetConnectorMethod:
		return gcpStage4GRPCVPCAccessGetConnector(grpcReqBody)
	case gcpVPCAccessCreateConnectorMethod:
		return gcpStage4GRPCVPCAccessCreateConnector(grpcReqBody)
	case gcpVPCAccessDeleteConnectorMethod:
		return gcpStage4GRPCVPCAccessDeleteConnector(grpcReqBody)
	default:
		return gcpStage4GRPCVMMigrationDynamic(path)
	}
}

func gcpStage4GRPCVPCAccessListConnectors(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &vpcaccesspb.ListConnectorsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := parseGCPStage4VPCAccessParent(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetPageSize() < 0 || req.GetPageSize() > 1000 {
		return grpcInvalidArgument("page_size-invalid")
	}
	start, valid := parseGCPStage4PageToken(req.GetPageToken())
	if !valid {
		return grpcInvalidArgument("page_token-invalid")
	}

	items := []*vpcaccesspb.Connector{
		gcpStage4VPCAccessConnector(project, location, "connector-1"),
		gcpStage4VPCAccessConnector(project, location, "connector-2"),
	}
	if start > len(items) {
		return grpcInvalidArgument("page_token-out-of-range")
	}
	end := len(items)
	if req.GetPageSize() > 0 && start+int(req.GetPageSize()) < end {
		end = start + int(req.GetPageSize())
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}

	return grpcProtoSuccess(&vpcaccesspb.ListConnectorsResponse{
		Connectors:    items[start:end],
		NextPageToken: next,
	})
}

func gcpStage4GRPCVPCAccessGetConnector(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &vpcaccesspb.GetConnectorRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, connectorID, ok := parseGCPStage4VPCAccessConnectorName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPVPCAccessMissingID(connectorID) {
		return grpcNotFound("connector-not-found")
	}
	return grpcProtoSuccess(gcpStage4VPCAccessConnector(project, location, connectorID))
}

func gcpStage4GRPCVPCAccessCreateConnector(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &vpcaccesspb.CreateConnectorRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := parseGCPStage4VPCAccessParent(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	connectorID := strings.TrimSpace(req.GetConnectorId())
	if connectorID == "" {
		return grpcInvalidArgument("connector_id-required")
	}
	if req.GetConnector() == nil {
		return grpcInvalidArgument("connector-required")
	}
	if strings.Contains(strings.ToLower(connectorID), "existing") {
		return grpcAlreadyExists("connector-already-exists")
	}

	expectedName := fmt.Sprintf("projects/%s/locations/%s/connectors/%s", project, location, connectorID)
	if connectorName := strings.TrimSpace(req.GetConnector().GetName()); connectorName != "" && connectorName != expectedName {
		return grpcInvalidArgument("connector_name-mismatch")
	}

	subnetName := strings.TrimSpace(req.GetConnector().GetSubnet().GetName())
	if strings.TrimSpace(req.GetConnector().GetNetwork()) == "" && subnetName == "" {
		return grpcInvalidArgument("connector_network_or_subnet-required")
	}
	if req.GetConnector().GetMaxThroughput() > 0 && req.GetConnector().GetMinThroughput() > 0 && req.GetConnector().GetMaxThroughput() < req.GetConnector().GetMinThroughput() {
		return grpcInvalidArgument("connector_max_throughput-lt-min")
	}
	if req.GetConnector().GetMaxInstances() > 0 && req.GetConnector().GetMinInstances() > 0 && req.GetConnector().GetMaxInstances() < req.GetConnector().GetMinInstances() {
		return grpcInvalidArgument("connector_max_instances-lt-min")
	}

	return grpcProtoSuccess(gcpStage4VPCAccessOperation(project, location, "createConnector."+connectorID))
}

func gcpStage4GRPCVPCAccessDeleteConnector(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &vpcaccesspb.DeleteConnectorRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, connectorID, ok := parseGCPStage4VPCAccessConnectorName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPVPCAccessMissingID(connectorID) {
		return grpcNotFound("connector-not-found")
	}
	if strings.Contains(strings.ToLower(connectorID), "in-use") {
		return grpcFailedPrecondition("connector-in-use")
	}
	return grpcProtoSuccess(gcpStage4VPCAccessOperation(project, location, "deleteConnector."+connectorID))
}

func parseGCPStage4VPCAccessParent(parent string) (project, location string, ok bool) {
	project, location, tail, parsed := parseGCPVPCAccessResourceName(strings.TrimSpace(parent))
	if !parsed || len(tail) != 0 {
		return "", "", false
	}
	return project, location, true
}

func parseGCPStage4VPCAccessConnectorName(name string) (project, location, connectorID string, ok bool) {
	project, location, connectorID, ok = parseGCPVPCAccessConnectorName(strings.TrimSpace(name))
	if !ok {
		return "", "", "", false
	}
	return project, location, connectorID, true
}

func gcpStage4VPCAccessConnector(project, location, connectorID string) *vpcaccesspb.Connector {
	return &vpcaccesspb.Connector{
		Name:          fmt.Sprintf("projects/%s/locations/%s/connectors/%s", project, location, connectorID),
		Network:       "default",
		IpCidrRange:   "10.8.0.0/28",
		State:         vpcaccesspb.Connector_READY,
		MinThroughput: 200,
		MaxThroughput: 300,
		MachineType:   "e2-micro",
		MinInstances:  2,
		MaxInstances:  3,
		ConnectedProjects: []string{
			project,
		},
		Subnet: &vpcaccesspb.Connector_Subnet{
			Name:      "default",
			ProjectId: project,
		},
	}
}

func gcpStage4VPCAccessOperation(project, location, operationID string) *longrunningpb.Operation {
	return &longrunningpb.Operation{
		Name: fmt.Sprintf("projects/%s/locations/%s/operations/%s", project, location, operationID),
		Done: false,
	}
}
