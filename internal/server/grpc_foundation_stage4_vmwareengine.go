package server

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	vmwareenginepb "cloud.google.com/go/vmwareengine/apiv1/vmwareenginepb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	gcpVMwareEngineListPrivateCloudsMethod  = "/google.cloud.vmwareengine.v1.VmwareEngine/ListPrivateClouds"
	gcpVMwareEngineGetPrivateCloudMethod    = "/google.cloud.vmwareengine.v1.VmwareEngine/GetPrivateCloud"
	gcpVMwareEngineCreatePrivateCloudMethod = "/google.cloud.vmwareengine.v1.VmwareEngine/CreatePrivateCloud"
)

func gcpStage4GRPCVMwareEngine(path string, grpcReqBody []byte) ([]byte, string, string, bool) {
	switch path {
	case gcpVMwareEngineListPrivateCloudsMethod:
		return gcpStage4GRPCVMwareEngineListPrivateClouds(grpcReqBody)
	case gcpVMwareEngineGetPrivateCloudMethod:
		return gcpStage4GRPCVMwareEngineGetPrivateCloud(grpcReqBody)
	case gcpVMwareEngineCreatePrivateCloudMethod:
		return gcpStage4GRPCVMwareEngineCreatePrivateCloud(grpcReqBody)
	default:
		return gcpStage4GRPCVMMigrationDynamic(path)
	}
}

func gcpStage4GRPCVMwareEngineListPrivateClouds(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &vmwareenginepb.ListPrivateCloudsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := parseGCPStage4VMwareEngineParent(req.GetParent())
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
	items := []*vmwareenginepb.PrivateCloud{
		gcpStage4VMwareEnginePrivateCloud(project, location, "private-cloud-1"),
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
	return grpcProtoSuccess(&vmwareenginepb.ListPrivateCloudsResponse{
		PrivateClouds: items[start:end],
		NextPageToken: next,
		Unreachable:   []string{},
	})
}

func gcpStage4GRPCVMwareEngineGetPrivateCloud(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &vmwareenginepb.GetPrivateCloudRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, privateCloudID, ok := parseGCPStage4VMwareEnginePrivateCloudName(req.GetName())
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPVMwareEngineMissingID(privateCloudID) {
		return grpcNotFound("private_cloud-not-found")
	}
	return grpcProtoSuccess(gcpStage4VMwareEnginePrivateCloud(project, location, privateCloudID))
}

func gcpStage4GRPCVMwareEngineCreatePrivateCloud(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &vmwareenginepb.CreatePrivateCloudRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := parseGCPStage4VMwareEngineParent(req.GetParent())
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	privateCloudID := strings.TrimSpace(req.GetPrivateCloudId())
	if privateCloudID == "" {
		return grpcInvalidArgument("private_cloud_id-required")
	}
	if req.GetPrivateCloud() == nil {
		return grpcInvalidArgument("private_cloud-required")
	}
	if strings.Contains(strings.ToLower(privateCloudID), "existing") {
		return grpcAlreadyExists("private_cloud-already-exists")
	}
	expectedName := fmt.Sprintf("projects/%s/locations/%s/privateClouds/%s", project, location, privateCloudID)
	if privateCloudName := strings.TrimSpace(req.GetPrivateCloud().GetName()); privateCloudName != "" && privateCloudName != expectedName {
		return grpcInvalidArgument("private_cloud_name-mismatch")
	}
	return grpcProtoSuccess(gcpStage4VMwareEngineOperation(project, location, "createPrivateCloud."+privateCloudID))
}

func gcpStage4VMwareEnginePrivateCloud(project, location, privateCloudID string) *vmwareenginepb.PrivateCloud {
	return &vmwareenginepb.PrivateCloud{
		Name:        fmt.Sprintf("projects/%s/locations/%s/privateClouds/%s", project, location, privateCloudID),
		Description: "Stackyard VMware Engine private cloud fixture",
		CreateTime:  timestamppb.New(gcpStage4ReferenceTime),
		UpdateTime:  timestamppb.New(gcpStage4ReferenceTime.Add(5 * time.Minute)),
	}
}

func gcpStage4VMwareEngineOperation(project, location, operationID string) *longrunningpb.Operation {
	return &longrunningpb.Operation{
		Name: fmt.Sprintf("projects/%s/locations/%s/operations/%s", project, location, operationID),
		Done: false,
	}
}

func parseGCPStage4VMwareEngineParent(parent string) (project, location string, ok bool) {
	project, location, tail, parsed := parseGCPVMwareEngineResourceName(strings.TrimSpace(parent))
	if !parsed || len(tail) != 0 {
		return "", "", false
	}
	return project, location, true
}

func parseGCPStage4VMwareEnginePrivateCloudName(name string) (project, location, privateCloudID string, ok bool) {
	project, location, tail, parsed := parseGCPVMwareEngineResourceName(strings.TrimSpace(name))
	if !parsed || len(tail) != 2 || tail[0] != "privateClouds" {
		return "", "", "", false
	}
	privateCloudID = strings.TrimSpace(tail[1])
	if privateCloudID == "" {
		return "", "", "", false
	}
	return project, location, privateCloudID, true
}
