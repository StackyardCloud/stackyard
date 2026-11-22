package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	dynamodbsvc "github.com/stackyard/stackyard/internal/services/dynamodb"
)

const dynamodbstreamsTargetPrefix = "DynamoDBStreams_20120810."

// respondDynamoDBStreamsJSON writes an awsJson1_0 success body.
func respondDynamoDBStreamsJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.0")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// handleDynamoDBStreamsJSONRouter serves the DynamoDB Streams API. It must run
// before handleDynamoDBJSONRouter, which claims every awsJson request signed
// "dynamodb" (Streams signs with the same service name).
func (s *Server) handleDynamoDBStreamsJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if r.Method != http.MethodPost || !strings.HasPrefix(target, dynamodbstreamsTargetPrefix) {
		return false
	}
	ok, status, code, msg, _ := s.validateSigV4WithService(r, "dynamodb")
	if !ok {
		respondDynamoDBError(w, status, code, msg)
		return true
	}
	action := strings.TrimPrefix(target, dynamodbstreamsTargetPrefix)
	if _, known := dynamodbstreamsOperationByName[action]; !known {
		respondDynamoDBError(w, http.StatusBadRequest, "UnknownOperationException", "unknown action "+action)
		return true
	}
	payload, err := parseDynamoDBPayload(r)
	if err != nil {
		respondDynamoDBError(w, http.StatusBadRequest, "SerializationException", "invalid JSON body")
		return true
	}
	limit, ok := dynamodbstreamsLimit(payload)
	if !ok {
		respondDynamoDBError(w, http.StatusBadRequest, "ValidationException", "Limit must be an integer")
		return true
	}

	switch action {
	case "ListStreams":
		streams, last, err := s.dynamodb.ListStreams(dynamodbString(payload["TableName"]), limit, dynamodbString(payload["ExclusiveStartStreamArn"]))
		if err != nil {
			respondDynamoDBStreamsErrorForErr(w, err)
			return true
		}
		out := map[string]any{"Streams": streams}
		if last != "" {
			out["LastEvaluatedStreamArn"] = last
		}
		respondDynamoDBStreamsJSON(w, http.StatusOK, out)
	case "DescribeStream":
		arn := dynamodbString(payload["StreamArn"])
		if arn == "" {
			respondDynamoDBError(w, http.StatusBadRequest, "ValidationException", "StreamArn is required")
			return true
		}
		desc, err := s.dynamodb.DescribeStream(arn, limit, dynamodbString(payload["ExclusiveStartShardId"]))
		if err != nil {
			respondDynamoDBStreamsErrorForErr(w, err)
			return true
		}
		respondDynamoDBStreamsJSON(w, http.StatusOK, map[string]any{"StreamDescription": desc})
	case "GetShardIterator":
		arn, shard, kind := dynamodbString(payload["StreamArn"]), dynamodbString(payload["ShardId"]), dynamodbString(payload["ShardIteratorType"])
		if arn == "" || shard == "" || kind == "" {
			respondDynamoDBError(w, http.StatusBadRequest, "ValidationException", "StreamArn, ShardId and ShardIteratorType are required")
			return true
		}
		iterator, err := s.dynamodb.GetShardIterator(arn, shard, kind, dynamodbString(payload["SequenceNumber"]))
		if err != nil {
			respondDynamoDBStreamsErrorForErr(w, err)
			return true
		}
		respondDynamoDBStreamsJSON(w, http.StatusOK, map[string]any{"ShardIterator": iterator})
	case "GetRecords":
		iterator := dynamodbString(payload["ShardIterator"])
		if iterator == "" {
			respondDynamoDBError(w, http.StatusBadRequest, "ValidationException", "ShardIterator is required")
			return true
		}
		records, next, err := s.dynamodb.GetRecords(iterator, limit)
		if err != nil {
			respondDynamoDBStreamsErrorForErr(w, err)
			return true
		}
		out := map[string]any{"Records": records}
		if next != "" {
			out["NextShardIterator"] = next
		}
		respondDynamoDBStreamsJSON(w, http.StatusOK, out)
	}
	return true
}

// dynamodbstreamsLimit reads the optional Limit; 0 means unset.
func dynamodbstreamsLimit(payload map[string]any) (int, bool) {
	if payload["Limit"] == nil {
		return 0, true
	}
	limit, ok := dynamodbInt(payload["Limit"])
	if ok && limit == 0 {
		limit = -1 // an explicit 0 is out of range, not "unset"
	}
	return limit, ok
}

func respondDynamoDBStreamsErrorForErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, dynamodbsvc.ErrExpiredIterator):
		respondDynamoDBError(w, http.StatusBadRequest, "ExpiredIteratorException", err.Error())
	case errors.Is(err, dynamodbsvc.ErrTrimmedDataAccess):
		respondDynamoDBError(w, http.StatusBadRequest, "TrimmedDataAccessException", err.Error())
	default:
		respondDynamoDBErrorForErr(w, err)
	}
}
