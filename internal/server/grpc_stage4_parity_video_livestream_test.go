package server

import (
	"net/http"
	"strings"
	"testing"

	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	livestreampb "cloud.google.com/go/video/livestream/apiv1/livestreampb"
)

func TestGCPStage4GRPCParity_VideoLivestream(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)

	parent := "projects/stackyard/locations/us-central1"
	channelName := parent + "/channels/channel-1"
	inputName := parent + "/inputs/input-1"

	restListResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/channels?pageSize=1", nil, map[string]string{
		"X-Stackyard-GCP-Service": "video-livestream",
	})
	if restListResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest video livestream list channels, got %d body=%s", restListResp.StatusCode, string(providerContractBody(t, restListResp)))
	}
	restListBody := providerContractJSONMap(t, restListResp)
	restChannels, ok := restListBody["channels"].([]any)
	if !ok || len(restChannels) == 0 {
		t.Fatalf("expected channels list in rest payload, got %#v", restListBody["channels"])
	}
	restChannel, _ := restChannels[0].(map[string]any)
	restChannelName, _ := restChannel["name"].(string)

	var listChannelsResp livestreampb.ListChannelsResponse
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpVideoLivestreamListChannelsMethod, &livestreampb.ListChannelsRequest{
		Parent:   parent,
		PageSize: 1,
	}, &listChannelsResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for list channels, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(listChannelsResp.GetChannels()) != 1 {
		t.Fatalf("expected one grpc channel, got %d", len(listChannelsResp.GetChannels()))
	}
	if listChannelsResp.GetChannels()[0].GetName() != restChannelName {
		t.Fatalf("expected grpc channel name %q to match rest %q", listChannelsResp.GetChannels()[0].GetName(), restChannelName)
	}

	restCreateResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/channels?channelId=channel-1", []byte(`{"channel":{"name":"projects/stackyard/locations/us-central1/channels/channel-1"}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "video-livestream",
	})
	if restCreateResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest video livestream create channel, got %d body=%s", restCreateResp.StatusCode, string(providerContractBody(t, restCreateResp)))
	}
	restCreateBody := providerContractJSONMap(t, restCreateResp)
	restOperationName, _ := restCreateBody["name"].(string)
	if strings.TrimSpace(restOperationName) == "" {
		t.Fatalf("expected operation name from rest create channel")
	}

	var createOp longrunningpb.Operation
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpVideoLivestreamCreateChannelMethod, &livestreampb.CreateChannelRequest{
		Parent:    parent,
		ChannelId: "channel-1",
		Channel: &livestreampb.Channel{
			Name: channelName,
		},
	}, &createOp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for create channel, got %q message=%q", grpcStatus, grpcMessage)
	}
	if createOp.GetName() != restOperationName {
		t.Fatalf("expected grpc operation name %q to match rest %q", createOp.GetName(), restOperationName)
	}

	restPreviewResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/inputs/input-1:preview", []byte(`{}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "video-livestream",
	})
	if restPreviewResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest video livestream preview input, got %d body=%s", restPreviewResp.StatusCode, string(providerContractBody(t, restPreviewResp)))
	}
	restPreviewBody := providerContractJSONMap(t, restPreviewResp)
	restPreviewURI, _ := restPreviewBody["uri"].(string)
	if strings.TrimSpace(restPreviewURI) == "" {
		t.Fatalf("expected preview uri in rest payload")
	}

	var previewResp livestreampb.PreviewInputResponse
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpVideoLivestreamPreviewInputMethod, &livestreampb.PreviewInputRequest{
		Name: inputName,
	}, &previewResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for preview input, got %q message=%q", grpcStatus, grpcMessage)
	}
	if previewResp.GetUri() != restPreviewURI {
		t.Fatalf("expected grpc preview uri %q to match rest %q", previewResp.GetUri(), restPreviewURI)
	}

	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpVideoLivestreamCreateChannelMethod, &livestreampb.CreateChannelRequest{
		ChannelId: "channel-2",
		Channel: &livestreampb.Channel{
			Name: parent + "/channels/channel-2",
		},
	}, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "parent-required") {
		t.Fatalf("expected grpc invalid argument for create channel missing parent, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpVideoLivestreamGetChannelMethod, &livestreampb.GetChannelRequest{
		Name: parent + "/channels/missing-channel",
	}, nil)
	if grpcStatus != "5" || !strings.Contains(grpcMessage, "channel-not-found") {
		t.Fatalf("expected grpc not found for get channel missing resource, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpVideoLivestreamStartChannelMethod, &livestreampb.StartChannelRequest{
		Name: parent + "/channels/running-channel",
	}, nil)
	if grpcStatus != "9" || !strings.Contains(grpcMessage, "channel-already-running") {
		t.Fatalf("expected grpc failed precondition for start channel running resource, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}
