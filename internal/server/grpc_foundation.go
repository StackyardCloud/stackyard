package server

import (
	"encoding/binary"
	"net/http"
	"net/http/httptest"
	"strings"

	aiplatformpb "cloud.google.com/go/aiplatform/apiv1/aiplatformpb"
	datastorepb "cloud.google.com/go/datastore/apiv1/datastorepb"
	"google.golang.org/protobuf/proto"
)

func maybeWriteKnownGRPCSuccess(w http.ResponseWriter, r *http.Request, rr *httptest.ResponseRecorder, grpcReqBody []byte) bool {
	if w == nil || r == nil || rr == nil {
		return false
	}
	payload, grpcStatus, grpcMessage, ok := knownGRPCResponse(rawRequestPath(r), grpcReqBody)
	if !ok {
		return false
	}
	writeGRPCUnaryResponse(w, payload, grpcStatus, grpcMessage)
	return true
}

func grpcFoundationNeedsRequestBody(path string) bool {
	normalized := normalizeGRPCFoundationPath(path)
	return strings.HasPrefix(normalized, "/google.")
}

func knownGRPCResponse(path string, grpcReqBody []byte) ([]byte, string, string, bool) {
	normalized := normalizeGRPCFoundationPath(path)
	if payload, grpcStatus, grpcMessage, ok := knownGCPStage4GRPCResponse(normalized, grpcReqBody); ok {
		return payload, grpcStatus, grpcMessage, true
	}
	switch normalized {
	case "/google.cloud.aiplatform.v1.DatasetService/ListDatasets":
		payload, ok := marshalProtoMessage(&aiplatformpb.ListDatasetsResponse{})
		return payload, "0", "", ok
	case "/google.cloud.aiplatform.v1.ModelService/ListModels":
		payload, ok := marshalProtoMessage(&aiplatformpb.ListModelsResponse{})
		return payload, "0", "", ok
	case "/google.cloud.aiplatform.v1.EndpointService/ListEndpoints":
		payload, ok := marshalProtoMessage(&aiplatformpb.ListEndpointsResponse{})
		return payload, "0", "", ok
	case "/google.cloud.aiplatform.v1.JobService/ListCustomJobs":
		payload, ok := marshalProtoMessage(&aiplatformpb.ListCustomJobsResponse{})
		return payload, "0", "", ok
	case "/google.datastore.v1.Datastore/Lookup":
		payload, ok := marshalProtoMessage(gcpFoundationDatastoreLookupResponse(grpcReqBody))
		return payload, "0", "", ok
	case "/google.datastore.v1.Datastore/RunQuery":
		payload, ok := marshalProtoMessage(gcpFoundationDatastoreRunQueryResponse())
		return payload, "0", "", ok
	case "/google.datastore.v1.Datastore/RunAggregationQuery":
		payload, ok := marshalProtoMessage(gcpFoundationDatastoreRunAggregationQueryResponse())
		return payload, "0", "", ok
	case "/google.datastore.v1.Datastore/BeginTransaction":
		payload, ok := marshalProtoMessage(&datastorepb.BeginTransactionResponse{Transaction: []byte("tx-1")})
		return payload, "0", "", ok
	case "/google.datastore.v1.Datastore/Commit":
		payload, ok := marshalProtoMessage(&datastorepb.CommitResponse{})
		return payload, "0", "", ok
	case "/google.datastore.v1.Datastore/Rollback":
		payload, ok := marshalProtoMessage(&datastorepb.RollbackResponse{})
		return payload, "0", "", ok
	case "/google.datastore.v1.Datastore/AllocateIds":
		payload, ok := marshalProtoMessage(&datastorepb.AllocateIdsResponse{})
		return payload, "0", "", ok
	case "/google.datastore.v1.Datastore/ReserveIds":
		payload, ok := marshalProtoMessage(&datastorepb.ReserveIdsResponse{})
		return payload, "0", "", ok
	default:
		return nil, "", "", false
	}
}

func knownGRPCSuccessPayload(path string, grpcReqBody []byte) ([]byte, bool) {
	payload, grpcStatus, _, ok := knownGRPCResponse(path, grpcReqBody)
	if !ok || grpcStatus != "0" {
		return nil, false
	}
	return payload, true
}

func normalizeGRPCFoundationPath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "/"
	}
	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + trimmed
	}
	if strings.HasPrefix(trimmed, "/gcp/") {
		trimmed = "/" + strings.TrimPrefix(trimmed, "/gcp/")
	}
	return trimmed
}

func gcpFoundationDatastoreLookupResponse(grpcReqBody []byte) *datastorepb.LookupResponse {
	req := &datastorepb.LookupRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) || len(req.GetKeys()) == 0 {
		return &datastorepb.LookupResponse{
			Found: []*datastorepb.EntityResult{
				{
					Entity: &datastorepb.Entity{
						Key: gcpFoundationDatastoreDefaultKey(),
					},
				},
			},
		}
	}

	found := make([]*datastorepb.EntityResult, 0, len(req.GetKeys()))
	for _, key := range req.GetKeys() {
		if key == nil {
			key = gcpFoundationDatastoreDefaultKey()
		}
		found = append(found, &datastorepb.EntityResult{
			Entity: &datastorepb.Entity{
				Key: key,
			},
		})
	}
	return &datastorepb.LookupResponse{Found: found}
}

func gcpFoundationDatastoreRunQueryResponse() *datastorepb.RunQueryResponse {
	return &datastorepb.RunQueryResponse{
		Batch: &datastorepb.QueryResultBatch{
			EntityResultType: datastorepb.EntityResult_FULL,
			EntityResults: []*datastorepb.EntityResult{
				{
					Entity: &datastorepb.Entity{
						Key: gcpFoundationDatastoreDefaultKey(),
					},
				},
			},
			MoreResults: datastorepb.QueryResultBatch_NO_MORE_RESULTS,
		},
	}
}

func gcpFoundationDatastoreRunAggregationQueryResponse() *datastorepb.RunAggregationQueryResponse {
	return &datastorepb.RunAggregationQueryResponse{
		Batch: &datastorepb.AggregationResultBatch{
			AggregationResults: []*datastorepb.AggregationResult{
				{
					AggregateProperties: map[string]*datastorepb.Value{
						"total_orders": {
							ValueType: &datastorepb.Value_IntegerValue{IntegerValue: 1},
						},
					},
				},
			},
			MoreResults: datastorepb.QueryResultBatch_NO_MORE_RESULTS,
		},
	}
}

func gcpFoundationDatastoreDefaultKey() *datastorepb.Key {
	return &datastorepb.Key{
		Path: []*datastorepb.Key_PathElement{
			{
				Kind:   "Order",
				IdType: &datastorepb.Key_PathElement_Name{Name: "order-1"},
			},
		},
	}
}

func decodeGRPCUnaryProtoRequest(grpcReqBody []byte, msg proto.Message) bool {
	if msg == nil || len(grpcReqBody) < 5 {
		return false
	}
	// Unary gRPC over h2c: 1-byte compression flag + 4-byte message length + protobuf payload.
	if grpcReqBody[0] != 0 {
		return false
	}
	payloadLen := int(binary.BigEndian.Uint32(grpcReqBody[1:5]))
	if payloadLen < 0 || len(grpcReqBody) < 5+payloadLen {
		return false
	}
	return proto.Unmarshal(grpcReqBody[5:5+payloadLen], msg) == nil
}

func marshalProtoMessage(msg proto.Message) ([]byte, bool) {
	if msg == nil {
		return nil, false
	}
	payload, err := proto.Marshal(msg)
	if err != nil {
		return nil, false
	}
	return payload, true
}

func writeGRPCUnaryResponse(w http.ResponseWriter, payload []byte, grpcStatus, grpcMessage string) {
	frame := make([]byte, 5+len(payload))
	frame[0] = 0 // uncompressed
	binary.BigEndian.PutUint32(frame[1:5], uint32(len(payload)))
	copy(frame[5:], payload)

	header := w.Header()
	header.Set("Content-Type", "application/grpc")
	header.Set("Trailer", "Grpc-Status,Grpc-Message")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(frame)

	// Declared trailers are set after writing the response body.
	header.Set("Grpc-Status", grpcStatus)
	header.Set("Grpc-Message", grpcMessage)
}
