package server

import (
	"fmt"
	"strconv"
	"strings"

	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	livestreampb "cloud.google.com/go/video/livestream/apiv1/livestreampb"
	locationpb "google.golang.org/genproto/googleapis/cloud/location"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/emptypb"
)

const (
	gcpVideoLivestreamCreateChannelMethod     = "/google.cloud.video.livestream.v1.LivestreamService/CreateChannel"
	gcpVideoLivestreamListChannelsMethod      = "/google.cloud.video.livestream.v1.LivestreamService/ListChannels"
	gcpVideoLivestreamGetChannelMethod        = "/google.cloud.video.livestream.v1.LivestreamService/GetChannel"
	gcpVideoLivestreamDeleteChannelMethod     = "/google.cloud.video.livestream.v1.LivestreamService/DeleteChannel"
	gcpVideoLivestreamUpdateChannelMethod     = "/google.cloud.video.livestream.v1.LivestreamService/UpdateChannel"
	gcpVideoLivestreamStartChannelMethod      = "/google.cloud.video.livestream.v1.LivestreamService/StartChannel"
	gcpVideoLivestreamStopChannelMethod       = "/google.cloud.video.livestream.v1.LivestreamService/StopChannel"
	gcpVideoLivestreamStartDistributionMethod = "/google.cloud.video.livestream.v1.LivestreamService/StartDistribution"
	gcpVideoLivestreamStopDistributionMethod  = "/google.cloud.video.livestream.v1.LivestreamService/StopDistribution"
	gcpVideoLivestreamCreateInputMethod       = "/google.cloud.video.livestream.v1.LivestreamService/CreateInput"
	gcpVideoLivestreamListInputsMethod        = "/google.cloud.video.livestream.v1.LivestreamService/ListInputs"
	gcpVideoLivestreamGetInputMethod          = "/google.cloud.video.livestream.v1.LivestreamService/GetInput"
	gcpVideoLivestreamDeleteInputMethod       = "/google.cloud.video.livestream.v1.LivestreamService/DeleteInput"
	gcpVideoLivestreamUpdateInputMethod       = "/google.cloud.video.livestream.v1.LivestreamService/UpdateInput"
	gcpVideoLivestreamPreviewInputMethod      = "/google.cloud.video.livestream.v1.LivestreamService/PreviewInput"
	gcpVideoLivestreamCreateEventMethod       = "/google.cloud.video.livestream.v1.LivestreamService/CreateEvent"
	gcpVideoLivestreamListEventsMethod        = "/google.cloud.video.livestream.v1.LivestreamService/ListEvents"
	gcpVideoLivestreamGetEventMethod          = "/google.cloud.video.livestream.v1.LivestreamService/GetEvent"
	gcpVideoLivestreamDeleteEventMethod       = "/google.cloud.video.livestream.v1.LivestreamService/DeleteEvent"
	gcpVideoLivestreamListClipsMethod         = "/google.cloud.video.livestream.v1.LivestreamService/ListClips"
	gcpVideoLivestreamGetClipMethod           = "/google.cloud.video.livestream.v1.LivestreamService/GetClip"
	gcpVideoLivestreamCreateClipMethod        = "/google.cloud.video.livestream.v1.LivestreamService/CreateClip"
	gcpVideoLivestreamDeleteClipMethod        = "/google.cloud.video.livestream.v1.LivestreamService/DeleteClip"
	gcpVideoLivestreamCreateDvrSessionMethod  = "/google.cloud.video.livestream.v1.LivestreamService/CreateDvrSession"
	gcpVideoLivestreamListDvrSessionsMethod   = "/google.cloud.video.livestream.v1.LivestreamService/ListDvrSessions"
	gcpVideoLivestreamGetDvrSessionMethod     = "/google.cloud.video.livestream.v1.LivestreamService/GetDvrSession"
	gcpVideoLivestreamDeleteDvrSessionMethod  = "/google.cloud.video.livestream.v1.LivestreamService/DeleteDvrSession"
	gcpVideoLivestreamUpdateDvrSessionMethod  = "/google.cloud.video.livestream.v1.LivestreamService/UpdateDvrSession"
	gcpVideoLivestreamCreateAssetMethod       = "/google.cloud.video.livestream.v1.LivestreamService/CreateAsset"
	gcpVideoLivestreamDeleteAssetMethod       = "/google.cloud.video.livestream.v1.LivestreamService/DeleteAsset"
	gcpVideoLivestreamGetAssetMethod          = "/google.cloud.video.livestream.v1.LivestreamService/GetAsset"
	gcpVideoLivestreamListAssetsMethod        = "/google.cloud.video.livestream.v1.LivestreamService/ListAssets"
	gcpVideoLivestreamGetPoolMethod           = "/google.cloud.video.livestream.v1.LivestreamService/GetPool"
	gcpVideoLivestreamUpdatePoolMethod        = "/google.cloud.video.livestream.v1.LivestreamService/UpdatePool"
	gcpVideoLivestreamGetLocationMethod       = "/google.cloud.video.livestream.v1.LivestreamService/GetLocation"
	gcpVideoLivestreamListLocationsMethod     = "/google.cloud.video.livestream.v1.LivestreamService/ListLocations"
	gcpVideoLivestreamGetOperationMethod      = "/google.cloud.video.livestream.v1.LivestreamService/GetOperation"
	gcpVideoLivestreamListOperationsMethod    = "/google.cloud.video.livestream.v1.LivestreamService/ListOperations"
	gcpVideoLivestreamCancelOperationMethod   = "/google.cloud.video.livestream.v1.LivestreamService/CancelOperation"
	gcpVideoLivestreamDeleteOperationMethod   = "/google.cloud.video.livestream.v1.LivestreamService/DeleteOperation"
)

func gcpStage4GRPCVideoLivestream(path string, grpcReqBody []byte) ([]byte, string, string, bool) {
	switch path {
	case gcpVideoLivestreamCreateChannelMethod:
		return gcpStage4GRPCVideoLivestreamCreateChannel(grpcReqBody)
	case gcpVideoLivestreamListChannelsMethod:
		return gcpStage4GRPCVideoLivestreamListChannels(grpcReqBody)
	case gcpVideoLivestreamGetChannelMethod:
		return gcpStage4GRPCVideoLivestreamGetChannel(grpcReqBody)
	case gcpVideoLivestreamDeleteChannelMethod:
		return gcpStage4GRPCVideoLivestreamDeleteChannel(grpcReqBody)
	case gcpVideoLivestreamUpdateChannelMethod:
		return gcpStage4GRPCVideoLivestreamUpdateChannel(grpcReqBody)
	case gcpVideoLivestreamStartChannelMethod:
		return gcpStage4GRPCVideoLivestreamStartChannel(grpcReqBody)
	case gcpVideoLivestreamStopChannelMethod:
		return gcpStage4GRPCVideoLivestreamStopChannel(grpcReqBody)
	case gcpVideoLivestreamStartDistributionMethod:
		return gcpStage4GRPCVideoLivestreamStartDistribution(grpcReqBody)
	case gcpVideoLivestreamStopDistributionMethod:
		return gcpStage4GRPCVideoLivestreamStopDistribution(grpcReqBody)
	case gcpVideoLivestreamCreateInputMethod:
		return gcpStage4GRPCVideoLivestreamCreateInput(grpcReqBody)
	case gcpVideoLivestreamListInputsMethod:
		return gcpStage4GRPCVideoLivestreamListInputs(grpcReqBody)
	case gcpVideoLivestreamGetInputMethod:
		return gcpStage4GRPCVideoLivestreamGetInput(grpcReqBody)
	case gcpVideoLivestreamDeleteInputMethod:
		return gcpStage4GRPCVideoLivestreamDeleteInput(grpcReqBody)
	case gcpVideoLivestreamUpdateInputMethod:
		return gcpStage4GRPCVideoLivestreamUpdateInput(grpcReqBody)
	case gcpVideoLivestreamPreviewInputMethod:
		return gcpStage4GRPCVideoLivestreamPreviewInput(grpcReqBody)
	case gcpVideoLivestreamCreateEventMethod:
		return gcpStage4GRPCVideoLivestreamCreateEvent(grpcReqBody)
	case gcpVideoLivestreamListEventsMethod:
		return gcpStage4GRPCVideoLivestreamListEvents(grpcReqBody)
	case gcpVideoLivestreamGetEventMethod:
		return gcpStage4GRPCVideoLivestreamGetEvent(grpcReqBody)
	case gcpVideoLivestreamDeleteEventMethod:
		return gcpStage4GRPCVideoLivestreamDeleteEvent(grpcReqBody)
	case gcpVideoLivestreamListClipsMethod:
		return gcpStage4GRPCVideoLivestreamListClips(grpcReqBody)
	case gcpVideoLivestreamGetClipMethod:
		return gcpStage4GRPCVideoLivestreamGetClip(grpcReqBody)
	case gcpVideoLivestreamCreateClipMethod:
		return gcpStage4GRPCVideoLivestreamCreateClip(grpcReqBody)
	case gcpVideoLivestreamDeleteClipMethod:
		return gcpStage4GRPCVideoLivestreamDeleteClip(grpcReqBody)
	case gcpVideoLivestreamCreateDvrSessionMethod:
		return gcpStage4GRPCVideoLivestreamCreateDvrSession(grpcReqBody)
	case gcpVideoLivestreamListDvrSessionsMethod:
		return gcpStage4GRPCVideoLivestreamListDvrSessions(grpcReqBody)
	case gcpVideoLivestreamGetDvrSessionMethod:
		return gcpStage4GRPCVideoLivestreamGetDvrSession(grpcReqBody)
	case gcpVideoLivestreamDeleteDvrSessionMethod:
		return gcpStage4GRPCVideoLivestreamDeleteDvrSession(grpcReqBody)
	case gcpVideoLivestreamUpdateDvrSessionMethod:
		return gcpStage4GRPCVideoLivestreamUpdateDvrSession(grpcReqBody)
	case gcpVideoLivestreamCreateAssetMethod:
		return gcpStage4GRPCVideoLivestreamCreateAsset(grpcReqBody)
	case gcpVideoLivestreamDeleteAssetMethod:
		return gcpStage4GRPCVideoLivestreamDeleteAsset(grpcReqBody)
	case gcpVideoLivestreamGetAssetMethod:
		return gcpStage4GRPCVideoLivestreamGetAsset(grpcReqBody)
	case gcpVideoLivestreamListAssetsMethod:
		return gcpStage4GRPCVideoLivestreamListAssets(grpcReqBody)
	case gcpVideoLivestreamGetPoolMethod:
		return gcpStage4GRPCVideoLivestreamGetPool(grpcReqBody)
	case gcpVideoLivestreamUpdatePoolMethod:
		return gcpStage4GRPCVideoLivestreamUpdatePool(grpcReqBody)
	case gcpVideoLivestreamGetLocationMethod:
		return gcpStage4GRPCVideoLivestreamGetLocation(grpcReqBody)
	case gcpVideoLivestreamListLocationsMethod:
		return gcpStage4GRPCVideoLivestreamListLocations(grpcReqBody)
	case gcpVideoLivestreamGetOperationMethod:
		return gcpStage4GRPCVideoLivestreamGetOperation(grpcReqBody)
	case gcpVideoLivestreamListOperationsMethod:
		return gcpStage4GRPCVideoLivestreamListOperations(grpcReqBody)
	case gcpVideoLivestreamCancelOperationMethod:
		return gcpStage4GRPCVideoLivestreamCancelOperation(grpcReqBody)
	case gcpVideoLivestreamDeleteOperationMethod:
		return gcpStage4GRPCVideoLivestreamDeleteOperation(grpcReqBody)
	default:
		return grpcUnimplemented("method-not-implemented")
	}
}

func gcpStage4GRPCVideoLivestreamCreateChannel(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &livestreampb.CreateChannelRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := gcpVideoLivestreamProjectLocationFromParent(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetChannel() == nil {
		return grpcInvalidArgument("channel-required")
	}
	channelID := strings.TrimSpace(req.GetChannelId())
	if channelID == "" {
		if name := strings.TrimSpace(req.GetChannel().GetName()); name != "" {
			_, parsedID, ok := gcpVideoLivestreamParseChannelName(name)
			if !ok {
				return grpcInvalidArgument("channel-name-invalid")
			}
			channelID = parsedID
		}
	}
	if channelID == "" {
		return grpcInvalidArgument("channel_id-required")
	}
	if !gcpVideoLivestreamIDPattern.MatchString(channelID) {
		return grpcInvalidArgument("channel_id-invalid")
	}
	parent := fmt.Sprintf("projects/%s/locations/%s", project, location)
	expectedName := parent + "/channels/" + channelID
	if name := strings.TrimSpace(req.GetChannel().GetName()); name != "" && name != expectedName {
		return grpcInvalidArgument("channel-name-mismatch")
	}
	return grpcProtoSuccess(gcpStage4VideoLivestreamOperation(parent, "createChannel."+channelID))
}

func gcpStage4GRPCVideoLivestreamListChannels(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &livestreampb.ListChannelsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := gcpVideoLivestreamProjectLocationFromParent(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	items := []*livestreampb.Channel{
		{Name: fmt.Sprintf("projects/%s/locations/%s/channels/channel-1", project, location)},
		{Name: fmt.Sprintf("projects/%s/locations/%s/channels/channel-2", project, location)},
	}
	start, end, nextPageToken, reason, ok := gcpStage4VideoLivestreamPageWindow(req.GetPageSize(), req.GetPageToken(), 1000, len(items))
	if !ok {
		return grpcInvalidArgument(reason)
	}
	return grpcProtoSuccess(&livestreampb.ListChannelsResponse{Channels: items[start:end], NextPageToken: nextPageToken})
}

func gcpStage4GRPCVideoLivestreamGetChannel(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &livestreampb.GetChannelRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	_, channelID, ok := gcpVideoLivestreamParseChannelName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPVideoLivestreamMissingID(channelID) {
		return grpcNotFound("channel-not-found")
	}
	return grpcProtoSuccess(&livestreampb.Channel{Name: strings.TrimSpace(req.GetName())})
}

func gcpStage4GRPCVideoLivestreamDeleteChannel(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &livestreampb.DeleteChannelRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent, channelID, ok := gcpVideoLivestreamParseChannelName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPVideoLivestreamMissingID(channelID) {
		return grpcNotFound("channel-not-found")
	}
	return grpcProtoSuccess(gcpStage4VideoLivestreamOperation(parent, "deleteChannel."+channelID))
}

func gcpStage4GRPCVideoLivestreamUpdateChannel(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &livestreampb.UpdateChannelRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if req.GetChannel() == nil {
		return grpcInvalidArgument("channel-required")
	}
	if req.GetUpdateMask() == nil {
		return grpcInvalidArgument("update_mask-required")
	}
	parent, channelID, ok := gcpVideoLivestreamParseChannelName(strings.TrimSpace(req.GetChannel().GetName()))
	if !ok {
		return grpcInvalidArgument("channel-name-required")
	}
	return grpcProtoSuccess(gcpStage4VideoLivestreamOperation(parent, "updateChannel."+channelID))
}

func gcpStage4GRPCVideoLivestreamStartChannel(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &livestreampb.StartChannelRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent, channelID, ok := gcpVideoLivestreamParseChannelName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if strings.Contains(strings.ToLower(channelID), "running") {
		return grpcFailedPrecondition("channel-already-running")
	}
	if isGCPVideoLivestreamMissingID(channelID) {
		return grpcNotFound("channel-not-found")
	}
	return grpcProtoSuccess(gcpStage4VideoLivestreamOperation(parent, "startChannel."+channelID))
}

func gcpStage4GRPCVideoLivestreamStopChannel(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &livestreampb.StopChannelRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent, channelID, ok := gcpVideoLivestreamParseChannelName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if strings.Contains(strings.ToLower(channelID), "stopped") {
		return grpcFailedPrecondition("channel-already-stopped")
	}
	if isGCPVideoLivestreamMissingID(channelID) {
		return grpcNotFound("channel-not-found")
	}
	return grpcProtoSuccess(gcpStage4VideoLivestreamOperation(parent, "stopChannel."+channelID))
}

func gcpStage4GRPCVideoLivestreamStartDistribution(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &livestreampb.StartDistributionRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent, channelID, distributionID, ok := gcpVideoLivestreamParseDistributionName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPVideoLivestreamMissingID(channelID) || isGCPVideoLivestreamMissingID(distributionID) {
		return grpcNotFound("distribution-not-found")
	}
	return grpcProtoSuccess(gcpStage4VideoLivestreamOperation(parent, "startDistribution."+channelID+"."+distributionID))
}

func gcpStage4GRPCVideoLivestreamStopDistribution(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &livestreampb.StopDistributionRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent, channelID, distributionID, ok := gcpVideoLivestreamParseDistributionName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPVideoLivestreamMissingID(channelID) || isGCPVideoLivestreamMissingID(distributionID) {
		return grpcNotFound("distribution-not-found")
	}
	return grpcProtoSuccess(gcpStage4VideoLivestreamOperation(parent, "stopDistribution."+channelID+"."+distributionID))
}

func gcpStage4GRPCVideoLivestreamCreateInput(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &livestreampb.CreateInputRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := gcpVideoLivestreamProjectLocationFromParent(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetInput() == nil {
		return grpcInvalidArgument("input-required")
	}
	inputID := strings.TrimSpace(req.GetInputId())
	if inputID == "" {
		if name := strings.TrimSpace(req.GetInput().GetName()); name != "" {
			_, parsedID, ok := gcpVideoLivestreamParseInputName(name)
			if !ok {
				return grpcInvalidArgument("input-name-invalid")
			}
			inputID = parsedID
		}
	}
	if inputID == "" {
		return grpcInvalidArgument("input_id-required")
	}
	if !gcpVideoLivestreamIDPattern.MatchString(inputID) {
		return grpcInvalidArgument("input_id-invalid")
	}
	parent := fmt.Sprintf("projects/%s/locations/%s", project, location)
	expectedName := parent + "/inputs/" + inputID
	if name := strings.TrimSpace(req.GetInput().GetName()); name != "" && name != expectedName {
		return grpcInvalidArgument("input-name-mismatch")
	}
	return grpcProtoSuccess(gcpStage4VideoLivestreamOperation(parent, "createInput."+inputID))
}

func gcpStage4GRPCVideoLivestreamListInputs(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &livestreampb.ListInputsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := gcpVideoLivestreamProjectLocationFromParent(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	items := []*livestreampb.Input{
		{Name: fmt.Sprintf("projects/%s/locations/%s/inputs/input-1", project, location)},
		{Name: fmt.Sprintf("projects/%s/locations/%s/inputs/input-2", project, location)},
	}
	start, end, nextPageToken, reason, ok := gcpStage4VideoLivestreamPageWindow(req.GetPageSize(), req.GetPageToken(), 1000, len(items))
	if !ok {
		return grpcInvalidArgument(reason)
	}
	return grpcProtoSuccess(&livestreampb.ListInputsResponse{Inputs: items[start:end], NextPageToken: nextPageToken})
}

func gcpStage4GRPCVideoLivestreamGetInput(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &livestreampb.GetInputRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	_, inputID, ok := gcpVideoLivestreamParseInputName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPVideoLivestreamMissingID(inputID) {
		return grpcNotFound("input-not-found")
	}
	return grpcProtoSuccess(&livestreampb.Input{Name: strings.TrimSpace(req.GetName())})
}

func gcpStage4GRPCVideoLivestreamDeleteInput(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &livestreampb.DeleteInputRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent, inputID, ok := gcpVideoLivestreamParseInputName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPVideoLivestreamMissingID(inputID) {
		return grpcNotFound("input-not-found")
	}
	return grpcProtoSuccess(gcpStage4VideoLivestreamOperation(parent, "deleteInput."+inputID))
}

func gcpStage4GRPCVideoLivestreamUpdateInput(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &livestreampb.UpdateInputRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if req.GetInput() == nil {
		return grpcInvalidArgument("input-required")
	}
	if req.GetUpdateMask() == nil {
		return grpcInvalidArgument("update_mask-required")
	}
	parent, inputID, ok := gcpVideoLivestreamParseInputName(strings.TrimSpace(req.GetInput().GetName()))
	if !ok {
		return grpcInvalidArgument("input-name-required")
	}
	return grpcProtoSuccess(gcpStage4VideoLivestreamOperation(parent, "updateInput."+inputID))
}

func gcpStage4GRPCVideoLivestreamPreviewInput(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &livestreampb.PreviewInputRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent, inputID, ok := gcpVideoLivestreamParseInputName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPVideoLivestreamMissingID(inputID) {
		return grpcNotFound("input-not-found")
	}
	project, location, _ := gcpVideoLivestreamProjectLocationFromParent(parent)
	return grpcProtoSuccess(&livestreampb.PreviewInputResponse{
		Uri:         fmt.Sprintf("https://preview.example.com/projects/%s/locations/%s/inputs/%s/manifest.m3u8", project, location, inputID),
		BearerToken: "stackyard-preview-token",
	})
}

func gcpStage4GRPCVideoLivestreamCreateEvent(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &livestreampb.CreateEventRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent, channelID, ok := gcpVideoLivestreamParseChannelName(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetEvent() == nil {
		return grpcInvalidArgument("event-required")
	}
	eventID := strings.TrimSpace(req.GetEventId())
	if eventID == "" {
		if name := strings.TrimSpace(req.GetEvent().GetName()); name != "" {
			_, _, parsedID, ok := gcpVideoLivestreamParseEventName(name)
			if !ok {
				return grpcInvalidArgument("event-name-invalid")
			}
			eventID = parsedID
		}
	}
	if eventID == "" {
		return grpcInvalidArgument("event_id-required")
	}
	if !gcpVideoLivestreamIDPattern.MatchString(eventID) {
		return grpcInvalidArgument("event_id-invalid")
	}
	return grpcProtoSuccess(&livestreampb.Event{Name: parent + "/channels/" + channelID + "/events/" + eventID})
}

func gcpStage4GRPCVideoLivestreamListEvents(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &livestreampb.ListEventsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent, channelID, ok := gcpVideoLivestreamParseChannelName(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	items := []*livestreampb.Event{
		{Name: parent + "/channels/" + channelID + "/events/event-1"},
		{Name: parent + "/channels/" + channelID + "/events/event-2"},
	}
	start, end, nextPageToken, reason, ok := gcpStage4VideoLivestreamPageWindow(req.GetPageSize(), req.GetPageToken(), 1000, len(items))
	if !ok {
		return grpcInvalidArgument(reason)
	}
	return grpcProtoSuccess(&livestreampb.ListEventsResponse{Events: items[start:end], NextPageToken: nextPageToken})
}

func gcpStage4GRPCVideoLivestreamGetEvent(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &livestreampb.GetEventRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	_, _, eventID, ok := gcpVideoLivestreamParseEventName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPVideoLivestreamMissingID(eventID) {
		return grpcNotFound("event-not-found")
	}
	return grpcProtoSuccess(&livestreampb.Event{Name: strings.TrimSpace(req.GetName())})
}

func gcpStage4GRPCVideoLivestreamDeleteEvent(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &livestreampb.DeleteEventRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	_, _, eventID, ok := gcpVideoLivestreamParseEventName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPVideoLivestreamMissingID(eventID) {
		return grpcNotFound("event-not-found")
	}
	return grpcProtoSuccess(&emptypb.Empty{})
}

func gcpStage4GRPCVideoLivestreamListClips(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &livestreampb.ListClipsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent, channelID, ok := gcpVideoLivestreamParseChannelName(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	items := []*livestreampb.Clip{
		{Name: parent + "/channels/" + channelID + "/clips/clip-1"},
		{Name: parent + "/channels/" + channelID + "/clips/clip-2"},
	}
	start, end, nextPageToken, reason, ok := gcpStage4VideoLivestreamPageWindow(req.GetPageSize(), req.GetPageToken(), 1000, len(items))
	if !ok {
		return grpcInvalidArgument(reason)
	}
	return grpcProtoSuccess(&livestreampb.ListClipsResponse{Clips: items[start:end], NextPageToken: nextPageToken})
}

func gcpStage4GRPCVideoLivestreamGetClip(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &livestreampb.GetClipRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	_, _, clipID, ok := gcpVideoLivestreamParseClipName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPVideoLivestreamMissingID(clipID) {
		return grpcNotFound("clip-not-found")
	}
	return grpcProtoSuccess(&livestreampb.Clip{Name: strings.TrimSpace(req.GetName())})
}

func gcpStage4GRPCVideoLivestreamCreateClip(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &livestreampb.CreateClipRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent, channelID, ok := gcpVideoLivestreamParseChannelName(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetClip() == nil {
		return grpcInvalidArgument("clip-required")
	}
	clipID := strings.TrimSpace(req.GetClipId())
	if clipID == "" {
		if name := strings.TrimSpace(req.GetClip().GetName()); name != "" {
			_, _, parsedID, ok := gcpVideoLivestreamParseClipName(name)
			if !ok {
				return grpcInvalidArgument("clip-name-invalid")
			}
			clipID = parsedID
		}
	}
	if clipID == "" {
		return grpcInvalidArgument("clip_id-required")
	}
	if !gcpVideoLivestreamIDPattern.MatchString(clipID) {
		return grpcInvalidArgument("clip_id-invalid")
	}
	return grpcProtoSuccess(gcpStage4VideoLivestreamOperation(parent, "createClip."+channelID+"."+clipID))
}

func gcpStage4GRPCVideoLivestreamDeleteClip(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &livestreampb.DeleteClipRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent, _, clipID, ok := gcpVideoLivestreamParseClipName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPVideoLivestreamMissingID(clipID) {
		return grpcNotFound("clip-not-found")
	}
	return grpcProtoSuccess(gcpStage4VideoLivestreamOperation(parent, "deleteClip."+clipID))
}

func gcpStage4GRPCVideoLivestreamCreateDvrSession(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &livestreampb.CreateDvrSessionRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent, channelID, ok := gcpVideoLivestreamParseChannelName(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetDvrSession() == nil {
		return grpcInvalidArgument("dvr_session-required")
	}
	dvrSessionID := strings.TrimSpace(req.GetDvrSessionId())
	if dvrSessionID == "" {
		if name := strings.TrimSpace(req.GetDvrSession().GetName()); name != "" {
			_, _, parsedID, ok := gcpVideoLivestreamParseDvrSessionName(name)
			if !ok {
				return grpcInvalidArgument("dvr_session-name-invalid")
			}
			dvrSessionID = parsedID
		}
	}
	if dvrSessionID == "" {
		return grpcInvalidArgument("dvr_session_id-required")
	}
	if !gcpVideoLivestreamIDPattern.MatchString(dvrSessionID) {
		return grpcInvalidArgument("dvr_session_id-invalid")
	}
	return grpcProtoSuccess(gcpStage4VideoLivestreamOperation(parent, "createDvrSession."+channelID+"."+dvrSessionID))
}

func gcpStage4GRPCVideoLivestreamListDvrSessions(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &livestreampb.ListDvrSessionsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent, channelID, ok := gcpVideoLivestreamParseChannelName(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	items := []*livestreampb.DvrSession{
		{Name: parent + "/channels/" + channelID + "/dvrSessions/dvr-session-1"},
		{Name: parent + "/channels/" + channelID + "/dvrSessions/dvr-session-2"},
	}
	start, end, nextPageToken, reason, ok := gcpStage4VideoLivestreamPageWindow(req.GetPageSize(), req.GetPageToken(), 1000, len(items))
	if !ok {
		return grpcInvalidArgument(reason)
	}
	return grpcProtoSuccess(&livestreampb.ListDvrSessionsResponse{DvrSessions: items[start:end], NextPageToken: nextPageToken})
}

func gcpStage4GRPCVideoLivestreamGetDvrSession(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &livestreampb.GetDvrSessionRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	_, _, dvrSessionID, ok := gcpVideoLivestreamParseDvrSessionName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPVideoLivestreamMissingID(dvrSessionID) {
		return grpcNotFound("dvr_session-not-found")
	}
	return grpcProtoSuccess(&livestreampb.DvrSession{Name: strings.TrimSpace(req.GetName())})
}

func gcpStage4GRPCVideoLivestreamDeleteDvrSession(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &livestreampb.DeleteDvrSessionRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent, _, dvrSessionID, ok := gcpVideoLivestreamParseDvrSessionName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPVideoLivestreamMissingID(dvrSessionID) {
		return grpcNotFound("dvr_session-not-found")
	}
	return grpcProtoSuccess(gcpStage4VideoLivestreamOperation(parent, "deleteDvrSession."+dvrSessionID))
}

func gcpStage4GRPCVideoLivestreamUpdateDvrSession(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &livestreampb.UpdateDvrSessionRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if req.GetDvrSession() == nil {
		return grpcInvalidArgument("dvr_session-required")
	}
	if req.GetUpdateMask() == nil {
		return grpcInvalidArgument("update_mask-required")
	}
	parent, _, dvrSessionID, ok := gcpVideoLivestreamParseDvrSessionName(strings.TrimSpace(req.GetDvrSession().GetName()))
	if !ok {
		return grpcInvalidArgument("dvr_session-name-required")
	}
	return grpcProtoSuccess(gcpStage4VideoLivestreamOperation(parent, "updateDvrSession."+dvrSessionID))
}

func gcpStage4GRPCVideoLivestreamCreateAsset(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &livestreampb.CreateAssetRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := gcpVideoLivestreamProjectLocationFromParent(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetAsset() == nil {
		return grpcInvalidArgument("asset-required")
	}
	assetID := strings.TrimSpace(req.GetAssetId())
	if assetID == "" {
		if name := strings.TrimSpace(req.GetAsset().GetName()); name != "" {
			_, parsedID, ok := gcpVideoLivestreamParseAssetName(name)
			if !ok {
				return grpcInvalidArgument("asset-name-invalid")
			}
			assetID = parsedID
		}
	}
	if assetID == "" {
		return grpcInvalidArgument("asset_id-required")
	}
	if !gcpVideoLivestreamIDPattern.MatchString(assetID) {
		return grpcInvalidArgument("asset_id-invalid")
	}
	parent := fmt.Sprintf("projects/%s/locations/%s", project, location)
	return grpcProtoSuccess(gcpStage4VideoLivestreamOperation(parent, "createAsset."+assetID))
}

func gcpStage4GRPCVideoLivestreamDeleteAsset(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &livestreampb.DeleteAssetRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent, assetID, ok := gcpVideoLivestreamParseAssetName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPVideoLivestreamMissingID(assetID) {
		return grpcNotFound("asset-not-found")
	}
	return grpcProtoSuccess(gcpStage4VideoLivestreamOperation(parent, "deleteAsset."+assetID))
}

func gcpStage4GRPCVideoLivestreamGetAsset(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &livestreampb.GetAssetRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	_, assetID, ok := gcpVideoLivestreamParseAssetName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPVideoLivestreamMissingID(assetID) {
		return grpcNotFound("asset-not-found")
	}
	return grpcProtoSuccess(&livestreampb.Asset{Name: strings.TrimSpace(req.GetName())})
}

func gcpStage4GRPCVideoLivestreamListAssets(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &livestreampb.ListAssetsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := gcpVideoLivestreamProjectLocationFromParent(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	items := []*livestreampb.Asset{
		{Name: fmt.Sprintf("projects/%s/locations/%s/assets/asset-1", project, location)},
		{Name: fmt.Sprintf("projects/%s/locations/%s/assets/asset-2", project, location)},
	}
	start, end, nextPageToken, reason, ok := gcpStage4VideoLivestreamPageWindow(req.GetPageSize(), req.GetPageToken(), 1000, len(items))
	if !ok {
		return grpcInvalidArgument(reason)
	}
	return grpcProtoSuccess(&livestreampb.ListAssetsResponse{Assets: items[start:end], NextPageToken: nextPageToken})
}

func gcpStage4GRPCVideoLivestreamGetPool(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &livestreampb.GetPoolRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	_, poolID, ok := gcpVideoLivestreamParsePoolName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPVideoLivestreamMissingID(poolID) {
		return grpcNotFound("pool-not-found")
	}
	return grpcProtoSuccess(&livestreampb.Pool{Name: strings.TrimSpace(req.GetName())})
}

func gcpStage4GRPCVideoLivestreamUpdatePool(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &livestreampb.UpdatePoolRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if req.GetPool() == nil {
		return grpcInvalidArgument("pool-required")
	}
	if req.GetUpdateMask() == nil {
		return grpcInvalidArgument("update_mask-required")
	}
	parent, poolID, ok := gcpVideoLivestreamParsePoolName(strings.TrimSpace(req.GetPool().GetName()))
	if !ok {
		return grpcInvalidArgument("pool-name-required")
	}
	return grpcProtoSuccess(gcpStage4VideoLivestreamOperation(parent, "updatePool."+poolID))
}

func gcpStage4GRPCVideoLivestreamGetLocation(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &locationpb.GetLocationRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := gcpVideoLivestreamProjectLocationFromName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	return grpcProtoSuccess(&locationpb.Location{Name: fmt.Sprintf("projects/%s/locations/%s", project, location), LocationId: location})
}

func gcpStage4GRPCVideoLivestreamListLocations(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &locationpb.ListLocationsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, ok := gcpVideoLivestreamProjectFromName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	items := []*locationpb.Location{
		{Name: fmt.Sprintf("projects/%s/locations/us-central1", project), LocationId: "us-central1"},
		{Name: fmt.Sprintf("projects/%s/locations/global", project), LocationId: "global"},
	}
	start, end, nextPageToken, reason, ok := gcpStage4VideoLivestreamPageWindow(req.GetPageSize(), req.GetPageToken(), 1000, len(items))
	if !ok {
		return grpcInvalidArgument(reason)
	}
	return grpcProtoSuccess(&locationpb.ListLocationsResponse{Locations: items[start:end], NextPageToken: nextPageToken})
}

func gcpStage4GRPCVideoLivestreamGetOperation(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &longrunningpb.GetOperationRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent, operationID, ok := gcpVideoLivestreamParseOperationName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPVideoLivestreamMissingID(operationID) {
		return grpcNotFound("operation-not-found")
	}
	return grpcProtoSuccess(gcpStage4VideoLivestreamOperation(parent, operationID))
}

func gcpStage4GRPCVideoLivestreamListOperations(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &longrunningpb.ListOperationsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	name := strings.TrimSpace(req.GetName())
	if strings.HasSuffix(name, "/operations") {
		name = strings.TrimSuffix(name, "/operations")
	}
	project, location, ok := gcpVideoLivestreamProjectLocationFromParent(name)
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	parent := fmt.Sprintf("projects/%s/locations/%s", project, location)
	items := []*longrunningpb.Operation{
		gcpStage4VideoLivestreamOperation(parent, "createChannel.channel-1"),
		gcpStage4VideoLivestreamOperation(parent, "createInput.input-1"),
	}
	start, end, nextPageToken, reason, ok := gcpStage4VideoLivestreamPageWindow(req.GetPageSize(), req.GetPageToken(), 1000, len(items))
	if !ok {
		return grpcInvalidArgument(reason)
	}
	return grpcProtoSuccess(&longrunningpb.ListOperationsResponse{Operations: items[start:end], NextPageToken: nextPageToken})
}

func gcpStage4GRPCVideoLivestreamCancelOperation(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &longrunningpb.CancelOperationRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	_, operationID, ok := gcpVideoLivestreamParseOperationName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPVideoLivestreamMissingID(operationID) {
		return grpcNotFound("operation-not-found")
	}
	return grpcProtoSuccess(&emptypb.Empty{})
}

func gcpStage4GRPCVideoLivestreamDeleteOperation(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &longrunningpb.DeleteOperationRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	_, operationID, ok := gcpVideoLivestreamParseOperationName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPVideoLivestreamMissingID(operationID) {
		return grpcNotFound("operation-not-found")
	}
	return grpcProtoSuccess(&emptypb.Empty{})
}

func gcpStage4VideoLivestreamOperation(parent, operationID string) *longrunningpb.Operation {
	responseAny, err := anypb.New(&emptypb.Empty{})
	if err != nil {
		responseAny = nil
	}
	out := &longrunningpb.Operation{
		Name: fmt.Sprintf("%s/operations/%s", strings.TrimSpace(parent), strings.TrimSpace(operationID)),
		Done: true,
	}
	if responseAny != nil {
		out.Result = &longrunningpb.Operation_Response{Response: responseAny}
	}
	return out
}

func gcpStage4VideoLivestreamPageWindow(pageSize int32, pageToken string, max, total int) (start, end int, nextPageToken, reason string, ok bool) {
	if pageSize < 0 {
		return 0, 0, "", "page_size-negative", false
	}
	if pageSize > int32(max) {
		return 0, 0, "", "page_size-too-large", false
	}
	start = 0
	if strings.TrimSpace(pageToken) != "" {
		parsed, err := strconv.Atoi(strings.TrimSpace(pageToken))
		if err != nil || parsed < 0 {
			return 0, 0, "", "page_token-invalid", false
		}
		start = parsed
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
