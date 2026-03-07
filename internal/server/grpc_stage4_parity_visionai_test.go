package server

import (
	"net/http"
	"strings"
	"testing"

	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	visionaipb "cloud.google.com/go/visionai/apiv1/visionaipb"
)

func TestGCPStage4GRPCParity_VisionAI(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)
	headers := map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "visionai",
	}

	restHealthResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.cloud.visionai.v1.HealthCheckService/HealthCheck", []byte(`{
		"cluster":"projects/stackyard/locations/us-central1/clusters/cluster-1"
	}`), headers)
	if restHealthResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest visionai healthcheck, got %d body=%s", restHealthResp.StatusCode, string(providerContractBody(t, restHealthResp)))
	}
	restHealthBody := providerContractJSONMap(t, restHealthResp)
	restHealthy, _ := restHealthBody["healthy"].(bool)
	restClusterInfo, _ := restHealthBody["clusterInfo"].(map[string]any)
	restStreamsCount, _ := restClusterInfo["streamsCount"].(float64)

	var grpcHealthResp visionaipb.HealthCheckResponse
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpVisionAIHealthCheckMethod, &visionaipb.HealthCheckRequest{
		Cluster: "projects/stackyard/locations/us-central1/clusters/cluster-1",
	}, &grpcHealthResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for visionai healthcheck, got %q message=%q", grpcStatus, grpcMessage)
	}
	if grpcHealthResp.GetHealthy() != restHealthy {
		t.Fatalf("expected grpc healthy=%t to match rest %t", grpcHealthResp.GetHealthy(), restHealthy)
	}
	if float64(grpcHealthResp.GetClusterInfo().GetStreamsCount()) != restStreamsCount {
		t.Fatalf("expected grpc streams_count=%d to match rest %.0f", grpcHealthResp.GetClusterInfo().GetStreamsCount(), restStreamsCount)
	}

	restListStreamsResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.cloud.visionai.v1.StreamsService/ListStreams", []byte(`{
		"parent":"projects/stackyard/locations/us-central1",
		"pageSize":1
	}`), headers)
	if restListStreamsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest visionai list streams, got %d body=%s", restListStreamsResp.StatusCode, string(providerContractBody(t, restListStreamsResp)))
	}
	restListStreamsBody := providerContractJSONMap(t, restListStreamsResp)
	restStreams, ok := restListStreamsBody["streams"].([]any)
	if !ok || len(restStreams) == 0 {
		t.Fatalf("expected non-empty rest streams array, got %#v", restListStreamsBody["streams"])
	}
	restFirstStream, _ := restStreams[0].(map[string]any)
	restStreamName, _ := restFirstStream["name"].(string)

	var grpcListStreamsResp visionaipb.ListStreamsResponse
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpVisionAIListStreamsMethod, &visionaipb.ListStreamsRequest{
		Parent:   "projects/stackyard/locations/us-central1",
		PageSize: 1,
	}, &grpcListStreamsResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for visionai list streams, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(grpcListStreamsResp.GetStreams()) != 1 {
		t.Fatalf("expected one grpc stream for page size 1, got %d", len(grpcListStreamsResp.GetStreams()))
	}
	if grpcListStreamsResp.GetStreams()[0].GetName() != restStreamName {
		t.Fatalf("expected grpc stream name %q to match rest %q", grpcListStreamsResp.GetStreams()[0].GetName(), restStreamName)
	}

	restCreateStreamResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.cloud.visionai.v1.StreamsService/CreateStream", []byte(`{
		"parent":"projects/stackyard/locations/us-central1",
		"streamId":"stream-1",
		"stream":{"displayName":"Stream One"}
	}`), headers)
	if restCreateStreamResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest visionai create stream, got %d body=%s", restCreateStreamResp.StatusCode, string(providerContractBody(t, restCreateStreamResp)))
	}
	restCreateStreamBody := providerContractJSONMap(t, restCreateStreamResp)
	restCreateStreamOperationName, _ := restCreateStreamBody["name"].(string)

	var grpcCreateStreamOp longrunningpb.Operation
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpVisionAICreateStreamMethod, &visionaipb.CreateStreamRequest{
		Parent:   "projects/stackyard/locations/us-central1",
		StreamId: "stream-1",
		Stream: &visionaipb.Stream{
			DisplayName: "Stream One",
		},
	}, &grpcCreateStreamOp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for visionai create stream, got %q message=%q", grpcStatus, grpcMessage)
	}
	if grpcCreateStreamOp.GetName() != restCreateStreamOperationName {
		t.Fatalf("expected grpc operation name %q to match rest %q", grpcCreateStreamOp.GetName(), restCreateStreamOperationName)
	}
	if !grpcCreateStreamOp.GetDone() {
		t.Fatalf("expected grpc create stream operation done=true")
	}

	var streamOpMetadata visionaipb.OperationMetadata
	if err := grpcCreateStreamOp.GetMetadata().UnmarshalTo(&streamOpMetadata); err != nil {
		t.Fatalf("expected typed create stream operation metadata, got error: %v", err)
	}
	var streamOpResponse visionaipb.Stream
	if err := grpcCreateStreamOp.GetResponse().UnmarshalTo(&streamOpResponse); err != nil {
		t.Fatalf("expected typed create stream operation response, got error: %v", err)
	}
	if !strings.Contains(streamOpResponse.GetName(), "/streams/stream-1") {
		t.Fatalf("expected create stream response stream name to include stream-1, got %q", streamOpResponse.GetName())
	}

	var grpcListApplicationsResp visionaipb.ListApplicationsResponse
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpVisionAIListApplicationsMethod, &visionaipb.ListApplicationsRequest{
		Parent:   "projects/stackyard/locations/us-central1",
		PageSize: 1,
	}, &grpcListApplicationsResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for visionai list applications, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(grpcListApplicationsResp.GetApplications()) != 1 {
		t.Fatalf("expected one grpc application for page size 1, got %d", len(grpcListApplicationsResp.GetApplications()))
	}

	var grpcListPublicOperatorsResp visionaipb.ListPublicOperatorsResponse
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpVisionAIListPublicOperatorsMethod, &visionaipb.ListPublicOperatorsRequest{
		Parent:   "projects/stackyard/locations/us-central1",
		PageSize: 1,
	}, &grpcListPublicOperatorsResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for visionai list public operators, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(grpcListPublicOperatorsResp.GetOperators()) != 1 {
		t.Fatalf("expected one grpc operator for page size 1, got %d", len(grpcListPublicOperatorsResp.GetOperators()))
	}

	var grpcCreateCorpusOp longrunningpb.Operation
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpVisionAICreateCorpusMethod, &visionaipb.CreateCorpusRequest{
		Parent: "projects/stackyard/locations/us-central1",
		Corpus: &visionaipb.Corpus{
			DisplayName: "Corpus One",
		},
	}, &grpcCreateCorpusOp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for visionai create corpus, got %q message=%q", grpcStatus, grpcMessage)
	}
	var corpusOpResponse visionaipb.Corpus
	if err := grpcCreateCorpusOp.GetResponse().UnmarshalTo(&corpusOpResponse); err != nil {
		t.Fatalf("expected typed create corpus operation response, got error: %v", err)
	}
	if !strings.Contains(corpusOpResponse.GetName(), "/corpora/") {
		t.Fatalf("expected typed corpus response name, got %q", corpusOpResponse.GetName())
	}

	var grpcListCorporaResp visionaipb.ListCorporaResponse
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpVisionAIListCorporaMethod, &visionaipb.ListCorporaRequest{
		Parent:   "projects/stackyard/locations/us-central1",
		PageSize: 1,
	}, &grpcListCorporaResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for visionai list corpora, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(grpcListCorporaResp.GetCorpora()) != 1 {
		t.Fatalf("expected one grpc corpus for page size 1, got %d", len(grpcListCorporaResp.GetCorpora()))
	}

	var grpcGetCorpusResp visionaipb.Corpus
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpVisionAIGetCorpusMethod, &visionaipb.GetCorpusRequest{
		Name: "projects/stackyard/locations/us-central1/corpora/corpus-1",
	}, &grpcGetCorpusResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for visionai get corpus, got %q message=%q", grpcStatus, grpcMessage)
	}
	if !strings.Contains(grpcGetCorpusResp.GetName(), "/corpora/corpus-1") {
		t.Fatalf("expected grpc corpus name to include corpus-1, got %q", grpcGetCorpusResp.GetName())
	}

	var grpcAcquireLeaseResp visionaipb.Lease
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpVisionAIAcquireLeaseMethod, &visionaipb.AcquireLeaseRequest{
		Series: "projects/stackyard/locations/us-central1/series/series-1",
		Owner:  "owner-1",
	}, &grpcAcquireLeaseResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for visionai acquire lease, got %q message=%q", grpcStatus, grpcMessage)
	}
	if grpcAcquireLeaseResp.GetId() == "" {
		t.Fatalf("expected grpc lease id")
	}

	var grpcRenewLeaseResp visionaipb.Lease
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpVisionAIRenewLeaseMethod, &visionaipb.RenewLeaseRequest{
		Id:     "lease-1",
		Series: "projects/stackyard/locations/us-central1/series/series-1",
		Owner:  "owner-1",
	}, &grpcRenewLeaseResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for visionai renew lease, got %q message=%q", grpcStatus, grpcMessage)
	}
	if grpcRenewLeaseResp.GetId() != "lease-1" {
		t.Fatalf("expected renewed lease id lease-1, got %q", grpcRenewLeaseResp.GetId())
	}

	var grpcReleaseLeaseResp visionaipb.ReleaseLeaseResponse
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpVisionAIReleaseLeaseMethod, &visionaipb.ReleaseLeaseRequest{
		Id:     "lease-1",
		Series: "projects/stackyard/locations/us-central1/series/series-1",
		Owner:  "owner-1",
	}, &grpcReleaseLeaseResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for visionai release lease, got %q message=%q", grpcStatus, grpcMessage)
	}

	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpVisionAIListStreamsMethod, &visionaipb.ListStreamsRequest{}, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "parent-required") {
		t.Fatalf("expected grpc invalid argument for list streams missing parent, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpVisionAIGetStreamMethod, &visionaipb.GetStreamRequest{
		Name: "projects/stackyard/locations/us-central1/streams/missing-stream",
	}, nil)
	if grpcStatus != "5" || !strings.Contains(grpcMessage, "stream-not-found") {
		t.Fatalf("expected grpc not found for missing stream, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpVisionAICreateCorpusMethod, &visionaipb.CreateCorpusRequest{
		Parent: "projects/stackyard/locations/us-central1",
		Corpus: &visionaipb.Corpus{},
	}, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "corpus.display_name-required") {
		t.Fatalf("expected grpc invalid argument for create corpus missing display name, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpVisionAIAcquireLeaseMethod, &visionaipb.AcquireLeaseRequest{
		Series: "projects/stackyard/locations/us-central1/series/series-1",
	}, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "series-owner-required") {
		t.Fatalf("expected grpc invalid argument for acquire lease missing owner, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}
