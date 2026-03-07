package server

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	stitcherpb "cloud.google.com/go/video/stitcher/apiv1/stitcherpb"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	gcpVideoStitcherCreateCdnKeyMethod         = "/google.cloud.video.stitcher.v1.VideoStitcherService/CreateCdnKey"
	gcpVideoStitcherListCdnKeysMethod          = "/google.cloud.video.stitcher.v1.VideoStitcherService/ListCdnKeys"
	gcpVideoStitcherGetCdnKeyMethod            = "/google.cloud.video.stitcher.v1.VideoStitcherService/GetCdnKey"
	gcpVideoStitcherDeleteCdnKeyMethod         = "/google.cloud.video.stitcher.v1.VideoStitcherService/DeleteCdnKey"
	gcpVideoStitcherUpdateCdnKeyMethod         = "/google.cloud.video.stitcher.v1.VideoStitcherService/UpdateCdnKey"
	gcpVideoStitcherCreateVodSessionMethod     = "/google.cloud.video.stitcher.v1.VideoStitcherService/CreateVodSession"
	gcpVideoStitcherGetVodSessionMethod        = "/google.cloud.video.stitcher.v1.VideoStitcherService/GetVodSession"
	gcpVideoStitcherListVodStitchDetailsMethod = "/google.cloud.video.stitcher.v1.VideoStitcherService/ListVodStitchDetails"
	gcpVideoStitcherGetVodStitchDetailMethod   = "/google.cloud.video.stitcher.v1.VideoStitcherService/GetVodStitchDetail"
	gcpVideoStitcherListVodAdTagDetailsMethod  = "/google.cloud.video.stitcher.v1.VideoStitcherService/ListVodAdTagDetails"
	gcpVideoStitcherGetVodAdTagDetailMethod    = "/google.cloud.video.stitcher.v1.VideoStitcherService/GetVodAdTagDetail"
	gcpVideoStitcherListLiveAdTagDetailsMethod = "/google.cloud.video.stitcher.v1.VideoStitcherService/ListLiveAdTagDetails"
	gcpVideoStitcherGetLiveAdTagDetailMethod   = "/google.cloud.video.stitcher.v1.VideoStitcherService/GetLiveAdTagDetail"
	gcpVideoStitcherCreateSlateMethod          = "/google.cloud.video.stitcher.v1.VideoStitcherService/CreateSlate"
	gcpVideoStitcherListSlatesMethod           = "/google.cloud.video.stitcher.v1.VideoStitcherService/ListSlates"
	gcpVideoStitcherGetSlateMethod             = "/google.cloud.video.stitcher.v1.VideoStitcherService/GetSlate"
	gcpVideoStitcherUpdateSlateMethod          = "/google.cloud.video.stitcher.v1.VideoStitcherService/UpdateSlate"
	gcpVideoStitcherDeleteSlateMethod          = "/google.cloud.video.stitcher.v1.VideoStitcherService/DeleteSlate"
	gcpVideoStitcherCreateLiveSessionMethod    = "/google.cloud.video.stitcher.v1.VideoStitcherService/CreateLiveSession"
	gcpVideoStitcherGetLiveSessionMethod       = "/google.cloud.video.stitcher.v1.VideoStitcherService/GetLiveSession"
	gcpVideoStitcherCreateLiveConfigMethod     = "/google.cloud.video.stitcher.v1.VideoStitcherService/CreateLiveConfig"
	gcpVideoStitcherListLiveConfigsMethod      = "/google.cloud.video.stitcher.v1.VideoStitcherService/ListLiveConfigs"
	gcpVideoStitcherGetLiveConfigMethod        = "/google.cloud.video.stitcher.v1.VideoStitcherService/GetLiveConfig"
	gcpVideoStitcherDeleteLiveConfigMethod     = "/google.cloud.video.stitcher.v1.VideoStitcherService/DeleteLiveConfig"
	gcpVideoStitcherUpdateLiveConfigMethod     = "/google.cloud.video.stitcher.v1.VideoStitcherService/UpdateLiveConfig"
	gcpVideoStitcherCreateVodConfigMethod      = "/google.cloud.video.stitcher.v1.VideoStitcherService/CreateVodConfig"
	gcpVideoStitcherListVodConfigsMethod       = "/google.cloud.video.stitcher.v1.VideoStitcherService/ListVodConfigs"
	gcpVideoStitcherGetVodConfigMethod         = "/google.cloud.video.stitcher.v1.VideoStitcherService/GetVodConfig"
	gcpVideoStitcherDeleteVodConfigMethod      = "/google.cloud.video.stitcher.v1.VideoStitcherService/DeleteVodConfig"
	gcpVideoStitcherUpdateVodConfigMethod      = "/google.cloud.video.stitcher.v1.VideoStitcherService/UpdateVodConfig"
)

func gcpStage4GRPCVideoStitcher(path string, grpcReqBody []byte) ([]byte, string, string, bool) {
	switch path {
	case gcpVideoStitcherCreateCdnKeyMethod:
		return gcpStage4GRPCVideoStitcherCreateCdnKey(grpcReqBody)
	case gcpVideoStitcherListCdnKeysMethod:
		return gcpStage4GRPCVideoStitcherListCdnKeys(grpcReqBody)
	case gcpVideoStitcherGetCdnKeyMethod:
		return gcpStage4GRPCVideoStitcherGetCdnKey(grpcReqBody)
	case gcpVideoStitcherDeleteCdnKeyMethod:
		return gcpStage4GRPCVideoStitcherDeleteCdnKey(grpcReqBody)
	case gcpVideoStitcherUpdateCdnKeyMethod:
		return gcpStage4GRPCVideoStitcherUpdateCdnKey(grpcReqBody)
	case gcpVideoStitcherCreateVodSessionMethod:
		return gcpStage4GRPCVideoStitcherCreateVodSession(grpcReqBody)
	case gcpVideoStitcherGetVodSessionMethod:
		return gcpStage4GRPCVideoStitcherGetVodSession(grpcReqBody)
	case gcpVideoStitcherListVodStitchDetailsMethod:
		return gcpStage4GRPCVideoStitcherListVodStitchDetails(grpcReqBody)
	case gcpVideoStitcherGetVodStitchDetailMethod:
		return gcpStage4GRPCVideoStitcherGetVodStitchDetail(grpcReqBody)
	case gcpVideoStitcherListVodAdTagDetailsMethod:
		return gcpStage4GRPCVideoStitcherListVodAdTagDetails(grpcReqBody)
	case gcpVideoStitcherGetVodAdTagDetailMethod:
		return gcpStage4GRPCVideoStitcherGetVodAdTagDetail(grpcReqBody)
	case gcpVideoStitcherListLiveAdTagDetailsMethod:
		return gcpStage4GRPCVideoStitcherListLiveAdTagDetails(grpcReqBody)
	case gcpVideoStitcherGetLiveAdTagDetailMethod:
		return gcpStage4GRPCVideoStitcherGetLiveAdTagDetail(grpcReqBody)
	case gcpVideoStitcherCreateSlateMethod:
		return gcpStage4GRPCVideoStitcherCreateSlate(grpcReqBody)
	case gcpVideoStitcherListSlatesMethod:
		return gcpStage4GRPCVideoStitcherListSlates(grpcReqBody)
	case gcpVideoStitcherGetSlateMethod:
		return gcpStage4GRPCVideoStitcherGetSlate(grpcReqBody)
	case gcpVideoStitcherUpdateSlateMethod:
		return gcpStage4GRPCVideoStitcherUpdateSlate(grpcReqBody)
	case gcpVideoStitcherDeleteSlateMethod:
		return gcpStage4GRPCVideoStitcherDeleteSlate(grpcReqBody)
	case gcpVideoStitcherCreateLiveSessionMethod:
		return gcpStage4GRPCVideoStitcherCreateLiveSession(grpcReqBody)
	case gcpVideoStitcherGetLiveSessionMethod:
		return gcpStage4GRPCVideoStitcherGetLiveSession(grpcReqBody)
	case gcpVideoStitcherCreateLiveConfigMethod:
		return gcpStage4GRPCVideoStitcherCreateLiveConfig(grpcReqBody)
	case gcpVideoStitcherListLiveConfigsMethod:
		return gcpStage4GRPCVideoStitcherListLiveConfigs(grpcReqBody)
	case gcpVideoStitcherGetLiveConfigMethod:
		return gcpStage4GRPCVideoStitcherGetLiveConfig(grpcReqBody)
	case gcpVideoStitcherDeleteLiveConfigMethod:
		return gcpStage4GRPCVideoStitcherDeleteLiveConfig(grpcReqBody)
	case gcpVideoStitcherUpdateLiveConfigMethod:
		return gcpStage4GRPCVideoStitcherUpdateLiveConfig(grpcReqBody)
	case gcpVideoStitcherCreateVodConfigMethod:
		return gcpStage4GRPCVideoStitcherCreateVodConfig(grpcReqBody)
	case gcpVideoStitcherListVodConfigsMethod:
		return gcpStage4GRPCVideoStitcherListVodConfigs(grpcReqBody)
	case gcpVideoStitcherGetVodConfigMethod:
		return gcpStage4GRPCVideoStitcherGetVodConfig(grpcReqBody)
	case gcpVideoStitcherDeleteVodConfigMethod:
		return gcpStage4GRPCVideoStitcherDeleteVodConfig(grpcReqBody)
	case gcpVideoStitcherUpdateVodConfigMethod:
		return gcpStage4GRPCVideoStitcherUpdateVodConfig(grpcReqBody)
	default:
		return grpcUnimplemented("method-not-implemented")
	}
}

func gcpStage4GRPCVideoStitcherCreateCdnKey(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &stitcherpb.CreateCdnKeyRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := gcpVideoStitcherProjectLocationFromParent(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetCdnKey() == nil {
		return grpcInvalidArgument("cdn_key-required")
	}
	cdnKeyID := strings.TrimSpace(req.GetCdnKeyId())
	if cdnKeyID == "" {
		if name := strings.TrimSpace(req.GetCdnKey().GetName()); name != "" {
			_, parsedID, ok := gcpVideoStitcherParseManagedResourceName(name, "cdnKeys")
			if !ok {
				return grpcInvalidArgument("cdn_key-name-invalid")
			}
			cdnKeyID = parsedID
		}
	}
	if cdnKeyID == "" {
		return grpcInvalidArgument("cdn_key_id-required")
	}
	if !gcpVideoStitcherIDPattern.MatchString(cdnKeyID) {
		return grpcInvalidArgument("cdn_key_id-invalid")
	}
	parent := fmt.Sprintf("projects/%s/locations/%s", project, location)
	expectedName := parent + "/cdnKeys/" + cdnKeyID
	if name := strings.TrimSpace(req.GetCdnKey().GetName()); name != "" && name != expectedName {
		return grpcInvalidArgument("cdn_key-name-mismatch")
	}
	return grpcProtoSuccess(gcpStage4VideoStitcherOperation(parent, "createCdnKey."+cdnKeyID, expectedName, "create"))
}

func gcpStage4GRPCVideoStitcherListCdnKeys(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &stitcherpb.ListCdnKeysRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := gcpVideoStitcherProjectLocationFromParent(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	items := []*stitcherpb.CdnKey{
		gcpStage4VideoStitcherCdnKey(project, location, "cdn-key-1"),
		gcpStage4VideoStitcherCdnKey(project, location, "cdn-key-2"),
	}
	start, end, nextPageToken, reason, ok := gcpStage4VideoStitcherPageWindow(req.GetPageSize(), req.GetPageToken(), 1000, len(items))
	if !ok {
		return grpcInvalidArgument(reason)
	}
	return grpcProtoSuccess(&stitcherpb.ListCdnKeysResponse{
		CdnKeys:       items[start:end],
		NextPageToken: nextPageToken,
	})
}

func gcpStage4GRPCVideoStitcherGetCdnKey(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &stitcherpb.GetCdnKeyRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent, cdnKeyID, ok := gcpVideoStitcherParseManagedResourceName(strings.TrimSpace(req.GetName()), "cdnKeys")
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPVideoStitcherMissingID(cdnKeyID) {
		return grpcNotFound("cdn_key-not-found")
	}
	project, location, _ := gcpVideoStitcherProjectLocationFromParent(parent)
	return grpcProtoSuccess(gcpStage4VideoStitcherCdnKey(project, location, cdnKeyID))
}

func gcpStage4GRPCVideoStitcherDeleteCdnKey(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &stitcherpb.DeleteCdnKeyRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent, cdnKeyID, ok := gcpVideoStitcherParseManagedResourceName(strings.TrimSpace(req.GetName()), "cdnKeys")
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPVideoStitcherMissingID(cdnKeyID) {
		return grpcNotFound("cdn_key-not-found")
	}
	return grpcProtoSuccess(gcpStage4VideoStitcherOperation(parent, "deleteCdnKey."+cdnKeyID, strings.TrimSpace(req.GetName()), "delete"))
}

func gcpStage4GRPCVideoStitcherUpdateCdnKey(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &stitcherpb.UpdateCdnKeyRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if req.GetCdnKey() == nil {
		return grpcInvalidArgument("cdn_key-required")
	}
	if req.GetUpdateMask() == nil || len(req.GetUpdateMask().GetPaths()) == 0 {
		return grpcInvalidArgument("update_mask-required")
	}
	parent, cdnKeyID, ok := gcpVideoStitcherParseManagedResourceName(strings.TrimSpace(req.GetCdnKey().GetName()), "cdnKeys")
	if !ok {
		return grpcInvalidArgument("cdn_key-name-required")
	}
	if isGCPVideoStitcherMissingID(cdnKeyID) {
		return grpcNotFound("cdn_key-not-found")
	}
	return grpcProtoSuccess(gcpStage4VideoStitcherOperation(parent, "updateCdnKey."+cdnKeyID, req.GetCdnKey().GetName(), "update"))
}

func gcpStage4GRPCVideoStitcherCreateVodSession(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &stitcherpb.CreateVodSessionRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := gcpVideoStitcherProjectLocationFromParent(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetVodSession() == nil {
		return grpcInvalidArgument("vod_session-required")
	}
	vodConfig := strings.TrimSpace(req.GetVodSession().GetVodConfig())
	sourceURI := strings.TrimSpace(req.GetVodSession().GetSourceUri())
	if vodConfig == "" && sourceURI == "" {
		return grpcInvalidArgument("vod_session-vod_config-or-source_uri-required")
	}
	sessionID := "vod-session-1"
	if name := strings.TrimSpace(req.GetVodSession().GetName()); name != "" {
		parent, parsedID, ok := gcpVideoStitcherParseVodSessionName(name)
		if !ok {
			return grpcInvalidArgument("vod_session-name-invalid")
		}
		if parent != strings.TrimSpace(req.GetParent()) {
			return grpcInvalidArgument("vod_session-name-mismatch")
		}
		sessionID = parsedID
	}
	resp := gcpStage4VideoStitcherVodSession(project, location, sessionID)
	if vodConfig != "" {
		resp.VodConfig = vodConfig
	}
	if sourceURI != "" {
		resp.SourceUri = sourceURI
	}
	if adTagURI := strings.TrimSpace(req.GetVodSession().GetAdTagUri()); adTagURI != "" {
		resp.AdTagUri = adTagURI
	}
	return grpcProtoSuccess(resp)
}

func gcpStage4GRPCVideoStitcherGetVodSession(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &stitcherpb.GetVodSessionRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent, sessionID, ok := gcpVideoStitcherParseVodSessionName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPVideoStitcherMissingID(sessionID) {
		return grpcNotFound("vod_session-not-found")
	}
	project, location, _ := gcpVideoStitcherProjectLocationFromParent(parent)
	return grpcProtoSuccess(gcpStage4VideoStitcherVodSession(project, location, sessionID))
}

func gcpStage4GRPCVideoStitcherListVodStitchDetails(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &stitcherpb.ListVodStitchDetailsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent, sessionID, ok := gcpVideoStitcherParseVodSessionName(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	project, location, _ := gcpVideoStitcherProjectLocationFromParent(parent)
	items := []*stitcherpb.VodStitchDetail{
		gcpStage4VideoStitcherVodStitchDetail(project, location, sessionID, "stitch-1"),
		gcpStage4VideoStitcherVodStitchDetail(project, location, sessionID, "stitch-2"),
	}
	start, end, nextPageToken, reason, ok := gcpStage4VideoStitcherPageWindow(req.GetPageSize(), req.GetPageToken(), 1000, len(items))
	if !ok {
		return grpcInvalidArgument(reason)
	}
	return grpcProtoSuccess(&stitcherpb.ListVodStitchDetailsResponse{
		VodStitchDetails: items[start:end],
		NextPageToken:    nextPageToken,
	})
}

func gcpStage4GRPCVideoStitcherGetVodStitchDetail(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &stitcherpb.GetVodStitchDetailRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent, sessionID, detailID, ok := gcpVideoStitcherParseVodStitchDetailName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPVideoStitcherMissingID(detailID) {
		return grpcNotFound("vod_stitch_detail-not-found")
	}
	project, location, _ := gcpVideoStitcherProjectLocationFromParent(parent)
	return grpcProtoSuccess(gcpStage4VideoStitcherVodStitchDetail(project, location, sessionID, detailID))
}

func gcpStage4GRPCVideoStitcherListVodAdTagDetails(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &stitcherpb.ListVodAdTagDetailsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent, sessionID, ok := gcpVideoStitcherParseVodSessionName(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	project, location, _ := gcpVideoStitcherProjectLocationFromParent(parent)
	items := []*stitcherpb.VodAdTagDetail{
		gcpStage4VideoStitcherVodAdTagDetail(project, location, sessionID, "adtag-1"),
		gcpStage4VideoStitcherVodAdTagDetail(project, location, sessionID, "adtag-2"),
	}
	start, end, nextPageToken, reason, ok := gcpStage4VideoStitcherPageWindow(req.GetPageSize(), req.GetPageToken(), 1000, len(items))
	if !ok {
		return grpcInvalidArgument(reason)
	}
	return grpcProtoSuccess(&stitcherpb.ListVodAdTagDetailsResponse{
		VodAdTagDetails: items[start:end],
		NextPageToken:   nextPageToken,
	})
}

func gcpStage4GRPCVideoStitcherGetVodAdTagDetail(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &stitcherpb.GetVodAdTagDetailRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent, sessionID, detailID, ok := gcpVideoStitcherParseVodAdTagDetailName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPVideoStitcherMissingID(detailID) {
		return grpcNotFound("vod_ad_tag_detail-not-found")
	}
	project, location, _ := gcpVideoStitcherProjectLocationFromParent(parent)
	return grpcProtoSuccess(gcpStage4VideoStitcherVodAdTagDetail(project, location, sessionID, detailID))
}

func gcpStage4GRPCVideoStitcherListLiveAdTagDetails(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &stitcherpb.ListLiveAdTagDetailsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent, sessionID, ok := gcpVideoStitcherParseLiveSessionName(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	project, location, _ := gcpVideoStitcherProjectLocationFromParent(parent)
	items := []*stitcherpb.LiveAdTagDetail{
		gcpStage4VideoStitcherLiveAdTagDetail(project, location, sessionID, "adtag-1"),
		gcpStage4VideoStitcherLiveAdTagDetail(project, location, sessionID, "adtag-2"),
	}
	start, end, nextPageToken, reason, ok := gcpStage4VideoStitcherPageWindow(req.GetPageSize(), req.GetPageToken(), 1000, len(items))
	if !ok {
		return grpcInvalidArgument(reason)
	}
	return grpcProtoSuccess(&stitcherpb.ListLiveAdTagDetailsResponse{
		LiveAdTagDetails: items[start:end],
		NextPageToken:    nextPageToken,
	})
}

func gcpStage4GRPCVideoStitcherGetLiveAdTagDetail(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &stitcherpb.GetLiveAdTagDetailRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent, sessionID, detailID, ok := gcpVideoStitcherParseLiveAdTagDetailName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPVideoStitcherMissingID(detailID) {
		return grpcNotFound("live_ad_tag_detail-not-found")
	}
	project, location, _ := gcpVideoStitcherProjectLocationFromParent(parent)
	return grpcProtoSuccess(gcpStage4VideoStitcherLiveAdTagDetail(project, location, sessionID, detailID))
}

func gcpStage4GRPCVideoStitcherCreateSlate(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &stitcherpb.CreateSlateRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := gcpVideoStitcherProjectLocationFromParent(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetSlate() == nil {
		return grpcInvalidArgument("slate-required")
	}
	if strings.TrimSpace(req.GetSlate().GetUri()) == "" {
		return grpcInvalidArgument("slate-uri-required")
	}
	slateID := strings.TrimSpace(req.GetSlateId())
	if slateID == "" {
		if name := strings.TrimSpace(req.GetSlate().GetName()); name != "" {
			_, parsedID, ok := gcpVideoStitcherParseManagedResourceName(name, "slates")
			if !ok {
				return grpcInvalidArgument("slate-name-invalid")
			}
			slateID = parsedID
		}
	}
	if slateID == "" {
		return grpcInvalidArgument("slate_id-required")
	}
	if !gcpVideoStitcherIDPattern.MatchString(slateID) {
		return grpcInvalidArgument("slate_id-invalid")
	}
	parent := fmt.Sprintf("projects/%s/locations/%s", project, location)
	expectedName := parent + "/slates/" + slateID
	if name := strings.TrimSpace(req.GetSlate().GetName()); name != "" && name != expectedName {
		return grpcInvalidArgument("slate-name-mismatch")
	}
	return grpcProtoSuccess(gcpStage4VideoStitcherOperation(parent, "createSlate."+slateID, expectedName, "create"))
}

func gcpStage4GRPCVideoStitcherListSlates(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &stitcherpb.ListSlatesRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := gcpVideoStitcherProjectLocationFromParent(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	items := []*stitcherpb.Slate{
		gcpStage4VideoStitcherSlate(project, location, "slate-1"),
		gcpStage4VideoStitcherSlate(project, location, "slate-2"),
	}
	start, end, nextPageToken, reason, ok := gcpStage4VideoStitcherPageWindow(req.GetPageSize(), req.GetPageToken(), 1000, len(items))
	if !ok {
		return grpcInvalidArgument(reason)
	}
	return grpcProtoSuccess(&stitcherpb.ListSlatesResponse{
		Slates:        items[start:end],
		NextPageToken: nextPageToken,
	})
}

func gcpStage4GRPCVideoStitcherGetSlate(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &stitcherpb.GetSlateRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent, slateID, ok := gcpVideoStitcherParseManagedResourceName(strings.TrimSpace(req.GetName()), "slates")
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPVideoStitcherMissingID(slateID) {
		return grpcNotFound("slate-not-found")
	}
	project, location, _ := gcpVideoStitcherProjectLocationFromParent(parent)
	return grpcProtoSuccess(gcpStage4VideoStitcherSlate(project, location, slateID))
}

func gcpStage4GRPCVideoStitcherUpdateSlate(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &stitcherpb.UpdateSlateRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if req.GetSlate() == nil {
		return grpcInvalidArgument("slate-required")
	}
	if req.GetUpdateMask() == nil || len(req.GetUpdateMask().GetPaths()) == 0 {
		return grpcInvalidArgument("update_mask-required")
	}
	parent, slateID, ok := gcpVideoStitcherParseManagedResourceName(strings.TrimSpace(req.GetSlate().GetName()), "slates")
	if !ok {
		return grpcInvalidArgument("slate-name-required")
	}
	if isGCPVideoStitcherMissingID(slateID) {
		return grpcNotFound("slate-not-found")
	}
	return grpcProtoSuccess(gcpStage4VideoStitcherOperation(parent, "updateSlate."+slateID, req.GetSlate().GetName(), "update"))
}

func gcpStage4GRPCVideoStitcherDeleteSlate(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &stitcherpb.DeleteSlateRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent, slateID, ok := gcpVideoStitcherParseManagedResourceName(strings.TrimSpace(req.GetName()), "slates")
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPVideoStitcherMissingID(slateID) {
		return grpcNotFound("slate-not-found")
	}
	return grpcProtoSuccess(gcpStage4VideoStitcherOperation(parent, "deleteSlate."+slateID, req.GetName(), "delete"))
}

func gcpStage4GRPCVideoStitcherCreateLiveSession(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &stitcherpb.CreateLiveSessionRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := gcpVideoStitcherProjectLocationFromParent(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetLiveSession() == nil {
		return grpcInvalidArgument("live_session-required")
	}
	liveConfig := strings.TrimSpace(req.GetLiveSession().GetLiveConfig())
	if liveConfig == "" {
		return grpcInvalidArgument("live_session-live_config-required")
	}
	sessionID := "live-session-1"
	if name := strings.TrimSpace(req.GetLiveSession().GetName()); name != "" {
		parent, parsedID, ok := gcpVideoStitcherParseLiveSessionName(name)
		if !ok {
			return grpcInvalidArgument("live_session-name-invalid")
		}
		if parent != strings.TrimSpace(req.GetParent()) {
			return grpcInvalidArgument("live_session-name-mismatch")
		}
		sessionID = parsedID
	}
	resp := gcpStage4VideoStitcherLiveSession(project, location, sessionID)
	resp.LiveConfig = liveConfig
	return grpcProtoSuccess(resp)
}

func gcpStage4GRPCVideoStitcherGetLiveSession(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &stitcherpb.GetLiveSessionRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent, sessionID, ok := gcpVideoStitcherParseLiveSessionName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPVideoStitcherMissingID(sessionID) {
		return grpcNotFound("live_session-not-found")
	}
	project, location, _ := gcpVideoStitcherProjectLocationFromParent(parent)
	return grpcProtoSuccess(gcpStage4VideoStitcherLiveSession(project, location, sessionID))
}

func gcpStage4GRPCVideoStitcherCreateLiveConfig(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &stitcherpb.CreateLiveConfigRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := gcpVideoStitcherProjectLocationFromParent(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetLiveConfig() == nil {
		return grpcInvalidArgument("live_config-required")
	}
	if strings.TrimSpace(req.GetLiveConfig().GetSourceUri()) == "" {
		return grpcInvalidArgument("live_config-source_uri-required")
	}
	liveConfigID := strings.TrimSpace(req.GetLiveConfigId())
	if liveConfigID == "" {
		if name := strings.TrimSpace(req.GetLiveConfig().GetName()); name != "" {
			_, parsedID, ok := gcpVideoStitcherParseManagedResourceName(name, "liveConfigs")
			if !ok {
				return grpcInvalidArgument("live_config-name-invalid")
			}
			liveConfigID = parsedID
		}
	}
	if liveConfigID == "" {
		return grpcInvalidArgument("live_config_id-required")
	}
	if !gcpVideoStitcherIDPattern.MatchString(liveConfigID) {
		return grpcInvalidArgument("live_config_id-invalid")
	}
	parent := fmt.Sprintf("projects/%s/locations/%s", project, location)
	expectedName := parent + "/liveConfigs/" + liveConfigID
	if name := strings.TrimSpace(req.GetLiveConfig().GetName()); name != "" && name != expectedName {
		return grpcInvalidArgument("live_config-name-mismatch")
	}
	return grpcProtoSuccess(gcpStage4VideoStitcherOperation(parent, "createLiveConfig."+liveConfigID, expectedName, "create"))
}

func gcpStage4GRPCVideoStitcherListLiveConfigs(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &stitcherpb.ListLiveConfigsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := gcpVideoStitcherProjectLocationFromParent(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	items := []*stitcherpb.LiveConfig{
		gcpStage4VideoStitcherLiveConfig(project, location, "live-config-1"),
		gcpStage4VideoStitcherLiveConfig(project, location, "live-config-2"),
	}
	start, end, nextPageToken, reason, ok := gcpStage4VideoStitcherPageWindow(req.GetPageSize(), req.GetPageToken(), 1000, len(items))
	if !ok {
		return grpcInvalidArgument(reason)
	}
	return grpcProtoSuccess(&stitcherpb.ListLiveConfigsResponse{
		LiveConfigs:   items[start:end],
		NextPageToken: nextPageToken,
	})
}

func gcpStage4GRPCVideoStitcherGetLiveConfig(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &stitcherpb.GetLiveConfigRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent, liveConfigID, ok := gcpVideoStitcherParseManagedResourceName(strings.TrimSpace(req.GetName()), "liveConfigs")
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPVideoStitcherMissingID(liveConfigID) {
		return grpcNotFound("live_config-not-found")
	}
	project, location, _ := gcpVideoStitcherProjectLocationFromParent(parent)
	return grpcProtoSuccess(gcpStage4VideoStitcherLiveConfig(project, location, liveConfigID))
}

func gcpStage4GRPCVideoStitcherDeleteLiveConfig(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &stitcherpb.DeleteLiveConfigRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent, liveConfigID, ok := gcpVideoStitcherParseManagedResourceName(strings.TrimSpace(req.GetName()), "liveConfigs")
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPVideoStitcherMissingID(liveConfigID) {
		return grpcNotFound("live_config-not-found")
	}
	return grpcProtoSuccess(gcpStage4VideoStitcherOperation(parent, "deleteLiveConfig."+liveConfigID, req.GetName(), "delete"))
}

func gcpStage4GRPCVideoStitcherUpdateLiveConfig(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &stitcherpb.UpdateLiveConfigRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if req.GetLiveConfig() == nil {
		return grpcInvalidArgument("live_config-required")
	}
	if req.GetUpdateMask() == nil || len(req.GetUpdateMask().GetPaths()) == 0 {
		return grpcInvalidArgument("update_mask-required")
	}
	parent, liveConfigID, ok := gcpVideoStitcherParseManagedResourceName(strings.TrimSpace(req.GetLiveConfig().GetName()), "liveConfigs")
	if !ok {
		return grpcInvalidArgument("live_config-name-required")
	}
	if isGCPVideoStitcherMissingID(liveConfigID) {
		return grpcNotFound("live_config-not-found")
	}
	return grpcProtoSuccess(gcpStage4VideoStitcherOperation(parent, "updateLiveConfig."+liveConfigID, req.GetLiveConfig().GetName(), "update"))
}

func gcpStage4GRPCVideoStitcherCreateVodConfig(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &stitcherpb.CreateVodConfigRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := gcpVideoStitcherProjectLocationFromParent(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if req.GetVodConfig() == nil {
		return grpcInvalidArgument("vod_config-required")
	}
	if strings.TrimSpace(req.GetVodConfig().GetSourceUri()) == "" {
		return grpcInvalidArgument("vod_config-source_uri-required")
	}
	if strings.TrimSpace(req.GetVodConfig().GetAdTagUri()) == "" {
		return grpcInvalidArgument("vod_config-ad_tag_uri-required")
	}
	vodConfigID := strings.TrimSpace(req.GetVodConfigId())
	if vodConfigID == "" {
		if name := strings.TrimSpace(req.GetVodConfig().GetName()); name != "" {
			_, parsedID, ok := gcpVideoStitcherParseManagedResourceName(name, "vodConfigs")
			if !ok {
				return grpcInvalidArgument("vod_config-name-invalid")
			}
			vodConfigID = parsedID
		}
	}
	if vodConfigID == "" {
		return grpcInvalidArgument("vod_config_id-required")
	}
	if !gcpVideoStitcherIDPattern.MatchString(vodConfigID) {
		return grpcInvalidArgument("vod_config_id-invalid")
	}
	parent := fmt.Sprintf("projects/%s/locations/%s", project, location)
	expectedName := parent + "/vodConfigs/" + vodConfigID
	if name := strings.TrimSpace(req.GetVodConfig().GetName()); name != "" && name != expectedName {
		return grpcInvalidArgument("vod_config-name-mismatch")
	}
	return grpcProtoSuccess(gcpStage4VideoStitcherOperation(parent, "createVodConfig."+vodConfigID, expectedName, "create"))
}

func gcpStage4GRPCVideoStitcherListVodConfigs(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &stitcherpb.ListVodConfigsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := gcpVideoStitcherProjectLocationFromParent(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	items := []*stitcherpb.VodConfig{
		gcpStage4VideoStitcherVodConfig(project, location, "vod-config-1"),
		gcpStage4VideoStitcherVodConfig(project, location, "vod-config-2"),
	}
	start, end, nextPageToken, reason, ok := gcpStage4VideoStitcherPageWindow(req.GetPageSize(), req.GetPageToken(), 1000, len(items))
	if !ok {
		return grpcInvalidArgument(reason)
	}
	return grpcProtoSuccess(&stitcherpb.ListVodConfigsResponse{
		VodConfigs:    items[start:end],
		NextPageToken: nextPageToken,
	})
}

func gcpStage4GRPCVideoStitcherGetVodConfig(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &stitcherpb.GetVodConfigRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent, vodConfigID, ok := gcpVideoStitcherParseManagedResourceName(strings.TrimSpace(req.GetName()), "vodConfigs")
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPVideoStitcherMissingID(vodConfigID) {
		return grpcNotFound("vod_config-not-found")
	}
	project, location, _ := gcpVideoStitcherProjectLocationFromParent(parent)
	return grpcProtoSuccess(gcpStage4VideoStitcherVodConfig(project, location, vodConfigID))
}

func gcpStage4GRPCVideoStitcherDeleteVodConfig(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &stitcherpb.DeleteVodConfigRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	parent, vodConfigID, ok := gcpVideoStitcherParseManagedResourceName(strings.TrimSpace(req.GetName()), "vodConfigs")
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPVideoStitcherMissingID(vodConfigID) {
		return grpcNotFound("vod_config-not-found")
	}
	return grpcProtoSuccess(gcpStage4VideoStitcherOperation(parent, "deleteVodConfig."+vodConfigID, req.GetName(), "delete"))
}

func gcpStage4GRPCVideoStitcherUpdateVodConfig(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &stitcherpb.UpdateVodConfigRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if req.GetVodConfig() == nil {
		return grpcInvalidArgument("vod_config-required")
	}
	if req.GetUpdateMask() == nil || len(req.GetUpdateMask().GetPaths()) == 0 {
		return grpcInvalidArgument("update_mask-required")
	}
	parent, vodConfigID, ok := gcpVideoStitcherParseManagedResourceName(strings.TrimSpace(req.GetVodConfig().GetName()), "vodConfigs")
	if !ok {
		return grpcInvalidArgument("vod_config-name-required")
	}
	if isGCPVideoStitcherMissingID(vodConfigID) {
		return grpcNotFound("vod_config-not-found")
	}
	return grpcProtoSuccess(gcpStage4VideoStitcherOperation(parent, "updateVodConfig."+vodConfigID, req.GetVodConfig().GetName(), "update"))
}

func gcpStage4VideoStitcherCdnKey(project, location, id string) *stitcherpb.CdnKey {
	return &stitcherpb.CdnKey{
		Name:     fmt.Sprintf("projects/%s/locations/%s/cdnKeys/%s", project, location, id),
		Hostname: fmt.Sprintf("%s.example.com", id),
		CdnKeyConfig: &stitcherpb.CdnKey_GoogleCdnKey{
			GoogleCdnKey: &stitcherpb.GoogleCdnKey{
				KeyName: "key-" + id,
			},
		},
	}
}

func gcpStage4VideoStitcherSlate(project, location, id string) *stitcherpb.Slate {
	return &stitcherpb.Slate{
		Name: fmt.Sprintf("projects/%s/locations/%s/slates/%s", project, location, id),
		Uri:  fmt.Sprintf("https://cdn.example.com/slates/%s.mp4", id),
	}
}

func gcpStage4VideoStitcherLiveConfig(project, location, id string) *stitcherpb.LiveConfig {
	return &stitcherpb.LiveConfig{
		Name:       fmt.Sprintf("projects/%s/locations/%s/liveConfigs/%s", project, location, id),
		SourceUri:  fmt.Sprintf("https://origin.example.com/live/%s.m3u8", id),
		AdTagUri:   fmt.Sprintf("https://ads.example.com/live/%s", id),
		State:      stitcherpb.LiveConfig_READY,
		AdTracking: stitcherpb.AdTracking_SERVER,
	}
}

func gcpStage4VideoStitcherVodConfig(project, location, id string) *stitcherpb.VodConfig {
	return &stitcherpb.VodConfig{
		Name:      fmt.Sprintf("projects/%s/locations/%s/vodConfigs/%s", project, location, id),
		SourceUri: fmt.Sprintf("https://origin.example.com/vod/%s.m3u8", id),
		AdTagUri:  fmt.Sprintf("https://ads.example.com/vod/%s", id),
		State:     stitcherpb.VodConfig_READY,
	}
}

func gcpStage4VideoStitcherVodSession(project, location, id string) *stitcherpb.VodSession {
	return &stitcherpb.VodSession{
		Name:       fmt.Sprintf("projects/%s/locations/%s/vodSessions/%s", project, location, id),
		PlayUri:    fmt.Sprintf("https://play.example.com/vod/%s/master.m3u8", id),
		VodConfig:  fmt.Sprintf("projects/%s/locations/%s/vodConfigs/vod-config-1", project, location),
		SourceUri:  "https://origin.example.com/vod/source.m3u8",
		AdTagUri:   "https://ads.example.com/vod/default",
		AdTracking: stitcherpb.AdTracking_SERVER,
	}
}

func gcpStage4VideoStitcherLiveSession(project, location, id string) *stitcherpb.LiveSession {
	return &stitcherpb.LiveSession{
		Name:       fmt.Sprintf("projects/%s/locations/%s/liveSessions/%s", project, location, id),
		PlayUri:    fmt.Sprintf("https://play.example.com/live/%s/master.m3u8", id),
		LiveConfig: fmt.Sprintf("projects/%s/locations/%s/liveConfigs/live-config-1", project, location),
		AdTracking: stitcherpb.AdTracking_SERVER,
	}
}

func gcpStage4VideoStitcherVodStitchDetail(project, location, sessionID, detailID string) *stitcherpb.VodStitchDetail {
	return &stitcherpb.VodStitchDetail{
		Name: fmt.Sprintf("projects/%s/locations/%s/vodSessions/%s/vodStitchDetails/%s", project, location, sessionID, detailID),
		AdStitchDetails: []*stitcherpb.AdStitchDetail{
			{
				AdBreakId: "break-1",
				AdId:      "ad-1",
			},
		},
	}
}

func gcpStage4VideoStitcherVodAdTagDetail(project, location, sessionID, detailID string) *stitcherpb.VodAdTagDetail {
	return &stitcherpb.VodAdTagDetail{
		Name: fmt.Sprintf("projects/%s/locations/%s/vodSessions/%s/vodAdTagDetails/%s", project, location, sessionID, detailID),
		AdRequests: []*stitcherpb.AdRequest{
			{
				Uri: "https://ads.example.com/vod/request",
			},
		},
	}
}

func gcpStage4VideoStitcherLiveAdTagDetail(project, location, sessionID, detailID string) *stitcherpb.LiveAdTagDetail {
	return &stitcherpb.LiveAdTagDetail{
		Name: fmt.Sprintf("projects/%s/locations/%s/liveSessions/%s/liveAdTagDetails/%s", project, location, sessionID, detailID),
		AdRequests: []*stitcherpb.AdRequest{
			{
				Uri: "https://ads.example.com/live/request",
			},
		},
	}
}

func gcpStage4VideoStitcherOperation(parent, operationID, target, verb string) *longrunningpb.Operation {
	metadataAny, err := anypb.New(&stitcherpb.OperationMetadata{
		CreateTime: timestamppb.New(gcpVideoStitcherReferenceTime),
		EndTime:    timestamppb.New(gcpVideoStitcherReferenceTime.Add(2 * time.Second)),
		Target:     target,
		Verb:       verb,
	})
	if err != nil {
		metadataAny = nil
	}
	responseAny, err := anypb.New(&emptypb.Empty{})
	if err != nil {
		responseAny = nil
	}
	out := &longrunningpb.Operation{
		Name: fmt.Sprintf("%s/operations/%s", parent, operationID),
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

func gcpStage4VideoStitcherPageWindow(pageSize int32, pageToken string, max, total int) (start, end int, nextPageToken, reason string, ok bool) {
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
