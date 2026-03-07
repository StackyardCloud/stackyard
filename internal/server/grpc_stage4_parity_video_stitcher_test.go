package server

import (
	"net/http"
	"strings"
	"testing"

	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	stitcherpb "cloud.google.com/go/video/stitcher/apiv1/stitcherpb"
)

func TestGCPStage4GRPCParity_VideoStitcher(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)

	parent := "projects/stackyard/locations/us-central1"
	cdnKeyName := parent + "/cdnKeys/cdn-key-1"
	vodSessionName := parent + "/vodSessions/vod-session-1"

	restListResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/cdnKeys?pageSize=1", nil, map[string]string{
		"X-Stackyard-GCP-Service": "video-stitcher",
	})
	if restListResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest video stitcher list cdn keys, got %d body=%s", restListResp.StatusCode, string(providerContractBody(t, restListResp)))
	}
	restListBody := providerContractJSONMap(t, restListResp)
	restCdnKeys, ok := restListBody["cdnKeys"].([]any)
	if !ok || len(restCdnKeys) == 0 {
		t.Fatalf("expected cdnKeys list in rest payload, got %#v", restListBody["cdnKeys"])
	}
	restCdnKey, _ := restCdnKeys[0].(map[string]any)
	restCdnKeyName, _ := restCdnKey["name"].(string)

	var listCdnKeysResp stitcherpb.ListCdnKeysResponse
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpVideoStitcherListCdnKeysMethod, &stitcherpb.ListCdnKeysRequest{
		Parent:   parent,
		PageSize: 1,
	}, &listCdnKeysResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for list cdn keys, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(listCdnKeysResp.GetCdnKeys()) != 1 {
		t.Fatalf("expected one grpc cdn key, got %d", len(listCdnKeysResp.GetCdnKeys()))
	}
	if listCdnKeysResp.GetCdnKeys()[0].GetName() != restCdnKeyName {
		t.Fatalf("expected grpc cdn key name %q to match rest %q", listCdnKeysResp.GetCdnKeys()[0].GetName(), restCdnKeyName)
	}

	restCreateResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/cdnKeys?cdnKeyId=cdn-key-1", []byte(`{"cdnKey":{"name":"projects/stackyard/locations/us-central1/cdnKeys/cdn-key-1"}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "video-stitcher",
	})
	if restCreateResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest video stitcher create cdn key, got %d body=%s", restCreateResp.StatusCode, string(providerContractBody(t, restCreateResp)))
	}
	restCreateBody := providerContractJSONMap(t, restCreateResp)
	restOperationName, _ := restCreateBody["name"].(string)
	if strings.TrimSpace(restOperationName) == "" {
		t.Fatalf("expected operation name from rest create cdn key")
	}

	var createOp longrunningpb.Operation
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpVideoStitcherCreateCdnKeyMethod, &stitcherpb.CreateCdnKeyRequest{
		Parent:   parent,
		CdnKeyId: "cdn-key-1",
		CdnKey: &stitcherpb.CdnKey{
			Name: cdnKeyName,
		},
	}, &createOp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for create cdn key, got %q message=%q", grpcStatus, grpcMessage)
	}
	if createOp.GetName() != restOperationName {
		t.Fatalf("expected grpc operation name %q to match rest %q", createOp.GetName(), restOperationName)
	}

	restVodSessionResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/vodSessions/vod-session-1", nil, map[string]string{
		"X-Stackyard-GCP-Service": "video-stitcher",
	})
	if restVodSessionResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest video stitcher get vod session, got %d body=%s", restVodSessionResp.StatusCode, string(providerContractBody(t, restVodSessionResp)))
	}
	restVodSessionBody := providerContractJSONMap(t, restVodSessionResp)
	restPlayURI, _ := restVodSessionBody["playUri"].(string)
	if strings.TrimSpace(restPlayURI) == "" {
		t.Fatalf("expected playUri in rest vod session payload")
	}

	var grpcVodSession stitcherpb.VodSession
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpVideoStitcherGetVodSessionMethod, &stitcherpb.GetVodSessionRequest{
		Name: vodSessionName,
	}, &grpcVodSession)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for get vod session, got %q message=%q", grpcStatus, grpcMessage)
	}
	if grpcVodSession.GetPlayUri() != restPlayURI {
		t.Fatalf("expected grpc playUri %q to match rest %q", grpcVodSession.GetPlayUri(), restPlayURI)
	}

	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpVideoStitcherCreateCdnKeyMethod, &stitcherpb.CreateCdnKeyRequest{
		CdnKeyId: "cdn-key-2",
		CdnKey: &stitcherpb.CdnKey{
			Name: parent + "/cdnKeys/cdn-key-2",
		},
	}, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "parent-required") {
		t.Fatalf("expected grpc invalid argument for create cdn key missing parent, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpVideoStitcherGetCdnKeyMethod, &stitcherpb.GetCdnKeyRequest{
		Name: parent + "/cdnKeys/missing-cdn-key",
	}, nil)
	if grpcStatus != "5" || !strings.Contains(grpcMessage, "cdn_key-not-found") {
		t.Fatalf("expected grpc not found for get cdn key missing resource, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}
