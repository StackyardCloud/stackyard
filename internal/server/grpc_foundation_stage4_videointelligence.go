package server

import (
	"fmt"
	"strings"
	"time"

	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	videointelligencepb "cloud.google.com/go/videointelligence/apiv1/videointelligencepb"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const gcpVideoIntelligenceAnnotateMethod = "/google.cloud.videointelligence.v1.VideoIntelligenceService/AnnotateVideo"

func gcpStage4GRPCVideoIntelligence(path string, grpcReqBody []byte) ([]byte, string, string, bool) {
	switch path {
	case gcpVideoIntelligenceAnnotateMethod:
		return gcpStage4GRPCVideoIntelligenceAnnotateVideo(grpcReqBody)
	default:
		return grpcUnimplemented("method-not-implemented")
	}
}

func gcpStage4GRPCVideoIntelligenceAnnotateVideo(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &videointelligencepb.AnnotateVideoRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}

	features := req.GetFeatures()
	if len(features) == 0 {
		return grpcInvalidArgument("features-required")
	}
	for _, feature := range features {
		if feature == videointelligencepb.Feature_FEATURE_UNSPECIFIED {
			return grpcInvalidArgument("feature-unspecified")
		}
	}

	inputURI := strings.TrimSpace(req.GetInputUri())
	hasInputContent := len(req.GetInputContent()) > 0
	switch {
	case inputURI == "" && !hasInputContent:
		return grpcInvalidArgument("input-required")
	case inputURI != "" && hasInputContent:
		return grpcInvalidArgument("input-mutually-exclusive")
	}
	if inputURI != "" && !isGCPVideoIntelligenceGCSURI(inputURI) {
		return grpcInvalidArgument("input_uri-invalid")
	}

	outputURI := strings.TrimSpace(req.GetOutputUri())
	if outputURI != "" && !isGCPVideoIntelligenceGCSURI(outputURI) {
		return grpcInvalidArgument("output_uri-invalid")
	}

	locationID := strings.ToLower(strings.TrimSpace(req.GetLocationId()))
	if locationID != "" {
		if _, ok := gcpVideoIntelligenceAllowedLocations[locationID]; !ok {
			return grpcInvalidArgument("location_id-invalid")
		}
	} else {
		locationID = "us-east1"
	}

	projectID := "stackyard"
	operationID := gcpVideoIntelligenceOperationID(inputURI, "input-content")
	return grpcProtoSuccess(gcpStage4VideoIntelligenceOperation(projectID, locationID, operationID, inputURI, features))
}

func gcpStage4VideoIntelligenceOperation(project, location, operationID, inputURI string, features []videointelligencepb.Feature) *longrunningpb.Operation {
	if strings.TrimSpace(inputURI) == "" {
		inputURI = "inline://input-content"
	}
	feature := videointelligencepb.Feature_SHOT_CHANGE_DETECTION
	if len(features) > 0 {
		feature = features[0]
	}

	metadata, _ := anypb.New(&videointelligencepb.AnnotateVideoProgress{
		AnnotationProgress: []*videointelligencepb.VideoAnnotationProgress{
			{
				InputUri:        inputURI,
				ProgressPercent: 100,
				StartTime:       timestamppb.New(gcpStage4ReferenceTime),
				UpdateTime:      timestamppb.New(gcpStage4ReferenceTime.Add(2 * time.Second)),
				Feature:         feature,
				Segment:         gcpStage4VideoIntelligenceSegment(0, 30*time.Second),
			},
		},
	})
	response, _ := anypb.New(&videointelligencepb.AnnotateVideoResponse{
		AnnotationResults: []*videointelligencepb.VideoAnnotationResults{
			{
				InputUri: inputURI,
				Segment:  gcpStage4VideoIntelligenceSegment(0, 30*time.Second),
				ShotAnnotations: []*videointelligencepb.VideoSegment{
					gcpStage4VideoIntelligenceSegment(0, 10*time.Second),
					gcpStage4VideoIntelligenceSegment(10*time.Second, 20*time.Second),
				},
			},
		},
	})

	out := &longrunningpb.Operation{
		Name: fmt.Sprintf("projects/%s/locations/%s/operations/%s", project, location, operationID),
		Done: true,
	}
	if metadata != nil {
		out.Metadata = metadata
	}
	if response != nil {
		out.Result = &longrunningpb.Operation_Response{Response: response}
	}
	return out
}

func gcpStage4VideoIntelligenceSegment(start, end time.Duration) *videointelligencepb.VideoSegment {
	return &videointelligencepb.VideoSegment{
		StartTimeOffset: durationpb.New(start),
		EndTimeOffset:   durationpb.New(end),
	}
}
