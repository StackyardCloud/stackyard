package server

import (
	"net/http"
	"strings"
	"testing"

	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	videointelligencepb "cloud.google.com/go/videointelligence/apiv1/videointelligencepb"
)

func TestGCPStage4GRPCParity_VideoIntelligence(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)

	restResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/videos:annotate", []byte(`{
		"inputUri":"gs://stackyard-inputs/video-1.mp4",
		"features":["SHOT_CHANGE_DETECTION"],
		"locationId":"us-east1"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "videointelligence",
	})
	if restResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest video intelligence annotate, got %d body=%s", restResp.StatusCode, string(providerContractBody(t, restResp)))
	}
	restBody := providerContractJSONMap(t, restResp)
	restOperationName, _ := restBody["name"].(string)
	if strings.TrimSpace(restOperationName) == "" {
		t.Fatalf("expected operation name in rest annotate response")
	}

	var annotateOp longrunningpb.Operation
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpVideoIntelligenceAnnotateMethod, &videointelligencepb.AnnotateVideoRequest{
		InputUri:   "gs://stackyard-inputs/video-1.mp4",
		Features:   []videointelligencepb.Feature{videointelligencepb.Feature_SHOT_CHANGE_DETECTION},
		LocationId: "us-east1",
		OutputUri:  "gs://stackyard-outputs/video-1.json",
	}, &annotateOp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for video intelligence annotate, got %q message=%q", grpcStatus, grpcMessage)
	}
	if annotateOp.GetName() != restOperationName {
		t.Fatalf("expected grpc operation name %q to match rest %q", annotateOp.GetName(), restOperationName)
	}
	if !annotateOp.GetDone() {
		t.Fatalf("expected grpc annotate operation done=true")
	}

	var metadata videointelligencepb.AnnotateVideoProgress
	if err := annotateOp.GetMetadata().UnmarshalTo(&metadata); err != nil {
		t.Fatalf("expected typed annotate metadata, got error: %v", err)
	}
	if len(metadata.GetAnnotationProgress()) == 0 {
		t.Fatalf("expected annotate metadata annotation_progress entries")
	}

	responseAny := annotateOp.GetResponse()
	if responseAny == nil {
		t.Fatalf("expected annotate operation response Any")
	}
	var response videointelligencepb.AnnotateVideoResponse
	if err := responseAny.UnmarshalTo(&response); err != nil {
		t.Fatalf("expected typed annotate response, got error: %v", err)
	}
	if len(response.GetAnnotationResults()) == 0 {
		t.Fatalf("expected annotate response annotation_results entries")
	}

	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpVideoIntelligenceAnnotateMethod, &videointelligencepb.AnnotateVideoRequest{
		InputUri: "gs://stackyard-inputs/video-1.mp4",
	}, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "features-required") {
		t.Fatalf("expected grpc invalid argument for missing features, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpVideoIntelligenceAnnotateMethod, &videointelligencepb.AnnotateVideoRequest{
		InputUri:     "gs://stackyard-inputs/video-1.mp4",
		InputContent: []byte("stackyard"),
		Features:     []videointelligencepb.Feature{videointelligencepb.Feature_SHOT_CHANGE_DETECTION},
	}, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "input-mutually-exclusive") {
		t.Fatalf("expected grpc invalid argument for mutually exclusive input, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpVideoIntelligenceAnnotateMethod, &videointelligencepb.AnnotateVideoRequest{
		InputUri:   "gs://stackyard-inputs/video-1.mp4",
		Features:   []videointelligencepb.Feature{videointelligencepb.Feature_SHOT_CHANGE_DETECTION},
		LocationId: "us-central1",
	}, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "location_id-invalid") {
		t.Fatalf("expected grpc invalid argument for invalid location id, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}
