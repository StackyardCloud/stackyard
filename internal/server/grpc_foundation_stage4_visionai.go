package server

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	visionaipb "cloud.google.com/go/visionai/apiv1/visionaipb"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	gcpVisionAIHealthCheckMethod         = "/google.cloud.visionai.v1.HealthCheckService/HealthCheck"
	gcpVisionAIListStreamsMethod         = "/google.cloud.visionai.v1.StreamsService/ListStreams"
	gcpVisionAIGetStreamMethod           = "/google.cloud.visionai.v1.StreamsService/GetStream"
	gcpVisionAICreateStreamMethod        = "/google.cloud.visionai.v1.StreamsService/CreateStream"
	gcpVisionAIListApplicationsMethod    = "/google.cloud.visionai.v1.AppPlatform/ListApplications"
	gcpVisionAIListPublicOperatorsMethod = "/google.cloud.visionai.v1.LiveVideoAnalytics/ListPublicOperators"
	gcpVisionAICreateCorpusMethod        = "/google.cloud.visionai.v1.Warehouse/CreateCorpus"
	gcpVisionAIListCorporaMethod         = "/google.cloud.visionai.v1.Warehouse/ListCorpora"
	gcpVisionAIGetCorpusMethod           = "/google.cloud.visionai.v1.Warehouse/GetCorpus"
	gcpVisionAIAcquireLeaseMethod        = "/google.cloud.visionai.v1.StreamingService/AcquireLease"
	gcpVisionAIRenewLeaseMethod          = "/google.cloud.visionai.v1.StreamingService/RenewLease"
	gcpVisionAIReleaseLeaseMethod        = "/google.cloud.visionai.v1.StreamingService/ReleaseLease"
)

func gcpStage4GRPCVisionAI(path string, grpcReqBody []byte) ([]byte, string, string, bool) {
	switch path {
	case gcpVisionAIHealthCheckMethod:
		return gcpStage4GRPCVisionAIHealthCheck(grpcReqBody)
	case gcpVisionAIListStreamsMethod:
		return gcpStage4GRPCVisionAIListStreams(grpcReqBody)
	case gcpVisionAIGetStreamMethod:
		return gcpStage4GRPCVisionAIGetStream(grpcReqBody)
	case gcpVisionAICreateStreamMethod:
		return gcpStage4GRPCVisionAICreateStream(grpcReqBody)
	case gcpVisionAIListApplicationsMethod:
		return gcpStage4GRPCVisionAIListApplications(grpcReqBody)
	case gcpVisionAIListPublicOperatorsMethod:
		return gcpStage4GRPCVisionAIListPublicOperators(grpcReqBody)
	case gcpVisionAICreateCorpusMethod:
		return gcpStage4GRPCVisionAICreateCorpus(grpcReqBody)
	case gcpVisionAIListCorporaMethod:
		return gcpStage4GRPCVisionAIListCorpora(grpcReqBody)
	case gcpVisionAIGetCorpusMethod:
		return gcpStage4GRPCVisionAIGetCorpus(grpcReqBody)
	case gcpVisionAIAcquireLeaseMethod:
		return gcpStage4GRPCVisionAIAcquireLease(grpcReqBody)
	case gcpVisionAIRenewLeaseMethod:
		return gcpStage4GRPCVisionAIRenewLease(grpcReqBody)
	case gcpVisionAIReleaseLeaseMethod:
		return gcpStage4GRPCVisionAIReleaseLease(grpcReqBody)
	default:
		return grpcUnimplemented("method-not-implemented")
	}
}

func gcpStage4GRPCVisionAIHealthCheck(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &visionaipb.HealthCheckRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	cluster := strings.TrimSpace(req.GetCluster())
	if cluster == "" {
		return grpcInvalidArgument("cluster-required")
	}
	if !strings.Contains(cluster, "/clusters/") {
		return grpcInvalidArgument("cluster-invalid")
	}
	return grpcProtoSuccess(&visionaipb.HealthCheckResponse{
		Healthy: true,
		Reason:  "",
		ClusterInfo: &visionaipb.ClusterInfo{
			StreamsCount:   2,
			ProcessesCount: 1,
		},
	})
}

func gcpStage4GRPCVisionAIListStreams(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &visionaipb.ListStreamsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := parseGCPVisionAIParentName(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}

	items := []*visionaipb.Stream{
		gcpStage4VisionAIStream(project, location, "stream-1"),
		gcpStage4VisionAIStream(project, location, "stream-2"),
	}
	start, end, nextToken, reason, valid := gcpStage4VisionAIPagination(req.GetPageSize(), req.GetPageToken(), 100, 1000, len(items))
	if !valid {
		return grpcInvalidArgument(reason)
	}
	return grpcProtoSuccess(&visionaipb.ListStreamsResponse{
		Streams:       items[start:end],
		NextPageToken: nextToken,
	})
}

func gcpStage4GRPCVisionAIGetStream(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &visionaipb.GetStreamRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, streamID, ok := parseGCPVisionAIStreamName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPVisionAIMissingID(streamID) {
		return grpcNotFound("stream-not-found")
	}
	return grpcProtoSuccess(gcpStage4VisionAIStream(project, location, streamID))
}

func gcpStage4GRPCVisionAICreateStream(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &visionaipb.CreateStreamRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := parseGCPVisionAIParentName(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	streamID := strings.TrimSpace(req.GetStreamId())
	if !isGCPVisionAIResourceID(streamID) {
		return grpcInvalidArgument("stream_id-required")
	}
	stream := req.GetStream()
	if stream == nil {
		return grpcInvalidArgument("stream-required")
	}
	displayName := strings.TrimSpace(stream.GetDisplayName())
	if displayName == "" {
		return grpcInvalidArgument("stream.display_name-required")
	}
	expectedName := fmt.Sprintf("projects/%s/locations/%s/streams/%s", project, location, streamID)
	if name := strings.TrimSpace(stream.GetName()); name != "" && name != expectedName {
		return grpcInvalidArgument("stream.name-mismatch")
	}

	response := gcpStage4VisionAIStream(project, location, streamID)
	response.DisplayName = displayName
	return grpcProtoSuccess(gcpStage4VisionAIOperation(
		project,
		location,
		"createStream."+streamID,
		expectedName,
		"create",
		response,
	))
}

func gcpStage4GRPCVisionAIListApplications(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &visionaipb.ListApplicationsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := parseGCPVisionAIParentName(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	items := []*visionaipb.Application{
		gcpStage4VisionAIApplication(project, location, "application-1"),
		gcpStage4VisionAIApplication(project, location, "application-2"),
	}
	start, end, nextToken, reason, valid := gcpStage4VisionAIPagination(req.GetPageSize(), req.GetPageToken(), 100, 1000, len(items))
	if !valid {
		return grpcInvalidArgument(reason)
	}
	return grpcProtoSuccess(&visionaipb.ListApplicationsResponse{
		Applications:  items[start:end],
		NextPageToken: nextToken,
	})
}

func gcpStage4GRPCVisionAIListPublicOperators(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &visionaipb.ListPublicOperatorsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := parseGCPVisionAIParentName(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	items := []*visionaipb.Operator{
		gcpStage4VisionAIOperator(project, location, "public-operator-1"),
		gcpStage4VisionAIOperator(project, location, "public-operator-2"),
	}
	start, end, nextToken, reason, valid := gcpStage4VisionAIPagination(req.GetPageSize(), req.GetPageToken(), 100, 1000, len(items))
	if !valid {
		return grpcInvalidArgument(reason)
	}
	return grpcProtoSuccess(&visionaipb.ListPublicOperatorsResponse{
		Operators:     items[start:end],
		NextPageToken: nextToken,
	})
}

func gcpStage4GRPCVisionAICreateCorpus(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &visionaipb.CreateCorpusRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := parseGCPVisionAIParentName(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	corpus := req.GetCorpus()
	if corpus == nil {
		return grpcInvalidArgument("corpus-required")
	}
	displayName := strings.TrimSpace(corpus.GetDisplayName())
	if displayName == "" {
		return grpcInvalidArgument("corpus.display_name-required")
	}
	response := gcpStage4VisionAICorpus(project, location, "corpus-1")
	response.DisplayName = displayName
	return grpcProtoSuccess(gcpStage4VisionAIOperation(
		project,
		location,
		"createCorpus.corpus-1",
		response.GetName(),
		"create",
		response,
	))
}

func gcpStage4GRPCVisionAIListCorpora(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &visionaipb.ListCorporaRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := parseGCPVisionAIParentName(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	items := []*visionaipb.Corpus{
		gcpStage4VisionAICorpus(project, location, "corpus-1"),
		gcpStage4VisionAICorpus(project, location, "corpus-2"),
	}
	start, end, nextToken, reason, valid := gcpStage4VisionAIPagination(req.GetPageSize(), req.GetPageToken(), 10, 20, len(items))
	if !valid {
		return grpcInvalidArgument(reason)
	}
	return grpcProtoSuccess(&visionaipb.ListCorporaResponse{
		Corpora:       items[start:end],
		NextPageToken: nextToken,
	})
}

func gcpStage4GRPCVisionAIGetCorpus(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &visionaipb.GetCorpusRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, corpusID, ok := parseGCPVisionAICorpusName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPVisionAIMissingID(corpusID) {
		return grpcNotFound("corpus-not-found")
	}
	return grpcProtoSuccess(gcpStage4VisionAICorpus(project, location, corpusID))
}

func gcpStage4GRPCVisionAIAcquireLease(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &visionaipb.AcquireLeaseRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	series := strings.TrimSpace(req.GetSeries())
	owner := strings.TrimSpace(req.GetOwner())
	if series == "" || owner == "" {
		return grpcInvalidArgument("series-owner-required")
	}
	lease := gcpStage4VisionAILease("lease-1", series, owner)
	if req.GetLeaseType() != visionaipb.LeaseType_LEASE_TYPE_UNSPECIFIED {
		lease.LeaseType = req.GetLeaseType()
	}
	if term := req.GetTerm(); term != nil && term.AsDuration() > 0 {
		lease.ExpireTime = timestamppb.New(gcpStage4ReferenceTime.Add(term.AsDuration()))
	}
	return grpcProtoSuccess(lease)
}

func gcpStage4GRPCVisionAIRenewLease(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &visionaipb.RenewLeaseRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	leaseID := strings.TrimSpace(req.GetId())
	series := strings.TrimSpace(req.GetSeries())
	owner := strings.TrimSpace(req.GetOwner())
	if leaseID == "" || series == "" || owner == "" {
		return grpcInvalidArgument("id-series-owner-required")
	}
	lease := gcpStage4VisionAILease(leaseID, series, owner)
	if term := req.GetTerm(); term != nil && term.AsDuration() > 0 {
		lease.ExpireTime = timestamppb.New(gcpStage4ReferenceTime.Add(term.AsDuration()))
	}
	return grpcProtoSuccess(lease)
}

func gcpStage4GRPCVisionAIReleaseLease(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &visionaipb.ReleaseLeaseRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	leaseID := strings.TrimSpace(req.GetId())
	series := strings.TrimSpace(req.GetSeries())
	owner := strings.TrimSpace(req.GetOwner())
	if leaseID == "" || series == "" || owner == "" {
		return grpcInvalidArgument("id-series-owner-required")
	}
	return grpcProtoSuccess(&visionaipb.ReleaseLeaseResponse{})
}

func gcpStage4VisionAIPagination(pageSize int32, pageToken string, defaultSize, maxSize, total int) (start, end int, nextToken, reason string, ok bool) {
	size := int(pageSize)
	if size < 0 {
		return 0, 0, "", "page_size-invalid", false
	}
	if size == 0 {
		size = defaultSize
	}
	if size > maxSize {
		return 0, 0, "", "page_size-too-large", false
	}

	start = 0
	if token := strings.TrimSpace(pageToken); token != "" {
		parsed, err := strconv.Atoi(token)
		if err != nil || parsed < 0 {
			return 0, 0, "", "page_token-invalid", false
		}
		start = parsed
	}
	if start > total {
		return 0, 0, "", "page_token-out-of-range", false
	}

	end = total
	if start+size < end {
		end = start + size
	}
	nextToken = ""
	if end < total {
		nextToken = strconv.Itoa(end)
	}
	return start, end, nextToken, "", true
}

func gcpStage4VisionAIStream(project, location, streamID string) *visionaipb.Stream {
	return &visionaipb.Stream{
		Name:                fmt.Sprintf("projects/%s/locations/%s/streams/%s", project, location, streamID),
		DisplayName:         "Stackyard Stream " + streamID,
		CreateTime:          timestamppb.New(gcpStage4ReferenceTime),
		UpdateTime:          timestamppb.New(gcpStage4ReferenceTime.Add(2 * time.Second)),
		Labels:              map[string]string{"env": "staged"},
		Annotations:         map[string]string{"source": "stackyard"},
		EnableHlsPlayback:   true,
		MediaWarehouseAsset: fmt.Sprintf("projects/%s/locations/%s/corpora/corpus-1/assets/asset-1", project, location),
	}
}

func gcpStage4VisionAIApplication(project, location, appID string) *visionaipb.Application {
	return &visionaipb.Application{
		Name:        fmt.Sprintf("projects/%s/locations/%s/applications/%s", project, location, appID),
		DisplayName: "Stackyard Application " + appID,
		Description: "Stackyard staged vision ai application",
		State:       visionaipb.Application_DEPLOYED,
		CreateTime:  timestamppb.New(gcpStage4ReferenceTime),
		UpdateTime:  timestamppb.New(gcpStage4ReferenceTime.Add(2 * time.Second)),
		Labels:      map[string]string{"env": "staged"},
	}
}

func gcpStage4VisionAIOperator(project, location, operatorID string) *visionaipb.Operator {
	return &visionaipb.Operator{
		Name:        fmt.Sprintf("projects/%s/locations/%s/operators/%s", project, location, operatorID),
		DockerImage: "gcr.io/stackyard/visionai/operator:latest",
		CreateTime:  timestamppb.New(gcpStage4ReferenceTime),
		UpdateTime:  timestamppb.New(gcpStage4ReferenceTime.Add(2 * time.Second)),
		Labels:      map[string]string{"source": "public"},
		OperatorDefinition: &visionaipb.OperatorDefinition{
			Operator: operatorID,
		},
	}
}

func gcpStage4VisionAICorpus(project, location, corpusID string) *visionaipb.Corpus {
	return &visionaipb.Corpus{
		Name:        fmt.Sprintf("projects/%s/locations/%s/corpora/%s", project, location, corpusID),
		DisplayName: "Stackyard Corpus " + corpusID,
		Description: "Stackyard staged corpus",
		Type:        visionaipb.Corpus_STREAM_VIDEO,
	}
}

func gcpStage4VisionAILease(leaseID, series, owner string) *visionaipb.Lease {
	return &visionaipb.Lease{
		Id:         leaseID,
		Series:     series,
		Owner:      owner,
		LeaseType:  visionaipb.LeaseType_LEASE_TYPE_READER,
		ExpireTime: timestamppb.New(gcpStage4ReferenceTime.Add(5 * time.Minute)),
	}
}

func gcpStage4VisionAIOperation(project, location, operationID, target, verb string, response proto.Message) *longrunningpb.Operation {
	if strings.TrimSpace(target) == "" {
		target = fmt.Sprintf("projects/%s/locations/%s", project, location)
	}
	if strings.TrimSpace(verb) == "" {
		verb = "mutate"
	}

	metadata, _ := anypb.New(&visionaipb.OperationMetadata{
		CreateTime: timestamppb.New(gcpStage4ReferenceTime),
		EndTime:    timestamppb.New(gcpStage4ReferenceTime.Add(2 * time.Second)),
		Target:     target,
		Verb:       verb,
		ApiVersion: "v1",
	})

	if response == nil {
		response = &emptypb.Empty{}
	}
	responseAny, _ := anypb.New(response)

	out := &longrunningpb.Operation{
		Name: fmt.Sprintf("projects/%s/locations/%s/operations/%s", project, location, operationID),
		Done: true,
	}
	if metadata != nil {
		out.Metadata = metadata
	}
	if responseAny != nil {
		out.Result = &longrunningpb.Operation_Response{Response: responseAny}
	}
	return out
}
