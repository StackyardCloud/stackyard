package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestS3VectorsStage3VectorDataPlane(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	createBucket := []byte(`{"vectorBucketName":"demo-bucket"}`)
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/CreateVectorBucket", createBucket, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusOK)

	createIndex := []byte(`{"vectorBucketName":"demo-bucket","indexName":"demo-index","dimension":3,"distanceMetric":"COSINE"}`)
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/CreateIndex", createIndex, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusOK)

	putReq := []byte(`{"indexName":"demo-index","vectors":[{"key":"vec-1","data":{"float32":[0.1,0.2,0.3]},"metadata":{"color":"blue"}},{"key":"vec-2","data":{"float32":[0.2,0.3,0.4]},"metadata":{"color":"red"}}]}`)
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/PutVectors", putReq, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusOK)

	getReq := []byte(`{"indexName":"demo-index","keys":["vec-1","vec-2"]}`)
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/GetVectors", getReq, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusOK)
	assertJSONEqual(t, mustBody(t, resp), []byte(`{"vectors":[{"key":"vec-1","data":{"float32":[0.1,0.2,0.3]},"metadata":{"color":"blue"},"createdAt":1700000000},{"key":"vec-2","data":{"float32":[0.2,0.3,0.4]},"metadata":{"color":"red"},"createdAt":1700000000}]}`))

	listReq := []byte(`{"indexName":"demo-index","metadataFilter":{"color":"blue"}}`)
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/ListVectors", listReq, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusOK)
	assertJSONEqual(t, mustBody(t, resp), []byte(`{"vectors":[{"key":"vec-1","data":{"float32":[0.1,0.2,0.3]},"metadata":{"color":"blue"},"createdAt":1700000000}],"nextToken":"token-1"}`))

	deleteReq := []byte(`{"indexName":"demo-index","keys":["vec-1"]}`)
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/DeleteVectors", deleteReq, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusOK)

	getReq = []byte(`{"indexName":"demo-index","keys":["vec-1"]}`)
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/GetVectors", getReq, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusOK)
	assertJSONEqual(t, mustBody(t, resp), []byte(`{"vectors":[]}`))
}

func TestS3VectorsStage3VectorPaginationAndSegments(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	createBucket := []byte(`{"vectorBucketName":"demo-bucket"}`)
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/CreateVectorBucket", createBucket, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusOK)

	createIndex := []byte(`{"vectorBucketName":"demo-bucket","indexName":"demo-index","dimension":3,"distanceMetric":"COSINE"}`)
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/CreateIndex", createIndex, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusOK)

	for _, name := range []string{"a", "b", "c"} {
		req := []byte(fmt.Sprintf(`{"indexName":"demo-index","vectors":[{"key":"%s","data":{"float32":[0.1,0.2,0.3]}}]}`, name))
		resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/PutVectors", req, map[string]string{
			"Content-Type": "application/json",
		}, "s3vectors")
		assertStatus(t, resp, http.StatusOK)
	}

	listReq := []byte(`{"indexName":"demo-index","maxResults":2}`)
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/ListVectors", listReq, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusOK)
	assertJSONEqual(t, mustBody(t, resp), []byte(`{"vectors":[{"key":"a","data":{"float32":[0.1,0.2,0.3]},"createdAt":1700000000},{"key":"b","data":{"float32":[0.1,0.2,0.3]},"createdAt":1700000000}],"nextToken":"token-2"}`))

	listReq = []byte(`{"indexName":"demo-index","maxResults":2,"nextToken":"token-2"}`)
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/ListVectors", listReq, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusOK)
	assertJSONEqual(t, mustBody(t, resp), []byte(`{"vectors":[{"key":"c","data":{"float32":[0.1,0.2,0.3]},"createdAt":1700000000}],"nextToken":"token-3"}`))

	segmentReq := []byte(`{"indexName":"demo-index","segmentCount":2,"segmentIndex":0}`)
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/ListVectors", segmentReq, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusOK)
	respBody := mustBody(t, resp)
	var parsed s3VectorsListVectorsResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		t.Fatalf("decode segment response: %v", err)
	}
	for _, vector := range parsed.Vectors {
		if s3VectorsSegmentForKey(vector.Key, 2) != 0 {
			t.Fatalf("vector %s not in segment 0", vector.Key)
		}
	}
}

func TestS3VectorsStage3VectorInvalidPayloads(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	createBucket := []byte(`{"vectorBucketName":"demo-bucket"}`)
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/CreateVectorBucket", createBucket, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusOK)

	createIndex := []byte(`{"vectorBucketName":"demo-bucket","indexName":"demo-index","dimension":3,"distanceMetric":"COSINE"}`)
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/CreateIndex", createIndex, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/PutVectors", []byte(`{"indexName":"demo-index","vectors":[{"key":"vec-1","data":{"float32":["bad"]}}]}`), map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusBadRequest)
	assertJSONEqual(t, mustBody(t, resp), []byte(`{"__type":"BadRequestException","message":"invalid JSON"}`))

	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/PutVectors", []byte(`{"indexName":"demo-index","vectors":[{"key":"vec-1","data":{"float32":[0.1,0.2]}}]}`), map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusBadRequest)
	assertJSONEqual(t, mustBody(t, resp), []byte(`{"__type":"ValidationException","message":"vector dimension mismatch"}`))

	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/PutVectors", []byte(`{"indexName":"demo-index","vectors":[{"key":"vec-1","data":{"float32":[0.1,0.2,0.3]},"metadata":{"color":1}}]}`), map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusBadRequest)
	assertJSONEqual(t, mustBody(t, resp), []byte(`{"__type":"BadRequestException","message":"invalid JSON"}`))
}
