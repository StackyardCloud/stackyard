package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestS3VectorsStage4QueryOrderingAndFilter(t *testing.T) {
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

	putReq := []byte(`{"indexName":"demo-index","vectors":[{"key":"a","data":{"float32":[1,0,0]},"metadata":{"color":"blue"}},{"key":"b","data":{"float32":[0,1,0]},"metadata":{"color":"red"}},{"key":"c","data":{"float32":[0.5,0.5,0]},"metadata":{"color":"blue"}}]}`)
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/PutVectors", putReq, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusOK)

	queryReq := []byte(`{"indexName":"demo-index","queryVector":{"float32":[1,0,0]},"topK":2}`)
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/QueryVectors", queryReq, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusOK)
	assertJSONEqual(t, mustBody(t, resp), []byte(`{"matches":[{"key":"a","score":1,"metadata":{"color":"blue"}},{"key":"c","score":0.7071067811865475,"metadata":{"color":"blue"}}]}`))

	filterReq := []byte(`{"indexName":"demo-index","queryVector":{"float32":[1,0,0]},"topK":2,"metadataFilter":{"color":"red"}}`)
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/QueryVectors", filterReq, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusOK)
	assertJSONEqual(t, mustBody(t, resp), []byte(`{"matches":[{"key":"b","score":0,"metadata":{"color":"red"}}]}`))
}

func TestS3VectorsStage4QueryInvalidShape(t *testing.T) {
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

	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/QueryVectors", []byte(`{"indexName":"demo-index","queryVector":{"float32":[1,0]}}`), map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusBadRequest)
	assertJSONEqual(t, mustBody(t, resp), []byte(`{"__type":"ValidationException","message":"queryVector dimension mismatch"}`))

	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/QueryVectors", []byte(`{"indexName":"demo-index","queryVector":{"float32":[]}}`), map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusBadRequest)
	assertJSONEqual(t, mustBody(t, resp), []byte(`{"__type":"ValidationException","message":"queryVector must include float32"}`))
}
