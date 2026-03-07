package server

import (
	"encoding/binary"
	"net/http"
	"net/http/httptest"
	"testing"

	aiplatformpb "cloud.google.com/go/aiplatform/apiv1/aiplatformpb"
	"google.golang.org/protobuf/proto"
)

func TestKnownGRPCSuccessPayload_AiplatformListDatasets(t *testing.T) {
	t.Parallel()

	payload, ok := knownGRPCSuccessPayload("/google.cloud.aiplatform.v1.DatasetService/ListDatasets", nil)
	if !ok {
		t.Fatalf("expected known payload")
	}
	var resp aiplatformpb.ListDatasetsResponse
	if err := proto.Unmarshal(payload, &resp); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
}

func TestKnownGRPCSuccessPayload_UnknownPath(t *testing.T) {
	t.Parallel()

	_, ok := knownGRPCSuccessPayload("/google.cloud.aiplatform.v1.DatasetService/Unknown", nil)
	if ok {
		t.Fatalf("expected unknown path to return false")
	}
}

func TestMaybeWriteKnownGRPCSuccess_WritesFramedSuccess(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "/google.cloud.aiplatform.v1.DatasetService/ListDatasets", nil)
	rr := httptest.NewRecorder()
	rr.WriteHeader(http.StatusNotImplemented)

	out := httptest.NewRecorder()
	if !maybeWriteKnownGRPCSuccess(out, req, rr, nil) {
		t.Fatalf("expected known gRPC success mapping")
	}

	resp := out.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 status, got %d", resp.StatusCode)
	}
	if got := resp.Header.Get("Content-Type"); got != "application/grpc" {
		t.Fatalf("expected application/grpc content type, got %q", got)
	}

	body := out.Body.Bytes()
	if len(body) < 5 {
		t.Fatalf("expected gRPC frame bytes, got %d bytes", len(body))
	}
	frameLen := int(binary.BigEndian.Uint32(body[1:5]))
	if frameLen != len(body)-5 {
		t.Fatalf("frame length mismatch: header=%d actual=%d", frameLen, len(body)-5)
	}
}
