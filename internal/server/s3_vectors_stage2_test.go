package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestS3VectorsStage2IndexLifecycle(t *testing.T) {
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
	assertJSONEqual(t, mustBody(t, resp), []byte(`{"indexArn":"arn:aws:s3vectors:us-east-1:123456789012:bucket/demo-bucket/index/demo-index"}`))

	getReq := []byte(`{"vectorBucketName":"demo-bucket","indexName":"demo-index"}`)
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/GetIndex", getReq, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusOK)
	assertJSONEqual(t, mustBody(t, resp), []byte(`{"indexName":"demo-index","indexArn":"arn:aws:s3vectors:us-east-1:123456789012:bucket/demo-bucket/index/demo-index","vectorBucketArn":"arn:aws:s3vectors:us-east-1:123456789012:bucket/demo-bucket","dimension":3,"distanceMetric":"COSINE"}`))

	listReq := []byte(`{"vectorBucketName":"demo-bucket"}`)
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/ListIndexes", listReq, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusOK)
	assertJSONEqual(t, mustBody(t, resp), []byte(`{"indexes":[{"indexName":"demo-index","indexArn":"arn:aws:s3vectors:us-east-1:123456789012:bucket/demo-bucket/index/demo-index","creationTime":1700000000,"dimension":3,"distanceMetric":"COSINE"}],"nextToken":"token-1"}`))

	deleteReq := []byte(`{"vectorBucketName":"demo-bucket","indexName":"demo-index"}`)
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/DeleteIndex", deleteReq, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/GetIndex", getReq, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusNotFound)
}

func TestS3VectorsStage2IndexValidation(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/CreateIndex", []byte(`{"indexName":"demo-index","dimension":3,"distanceMetric":"COSINE"}`), map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusBadRequest)
	assertJSONEqual(t, mustBody(t, resp), []byte(`{"__type":"ValidationException","message":"vectorBucketName is required"}`))

	createBucket := []byte(`{"vectorBucketName":"demo-bucket"}`)
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/CreateVectorBucket", createBucket, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/CreateIndex", []byte(`{"vectorBucketName":"demo-bucket","indexName":"demo-index","dimension":0,"distanceMetric":"COSINE"}`), map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusBadRequest)
	assertJSONEqual(t, mustBody(t, resp), []byte(`{"__type":"ValidationException","message":"dimension must be greater than 0"}`))

	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/CreateIndex", []byte(`{"vectorBucketName":"demo-bucket","indexName":"demo-index","dimension":3,"distanceMetric":"BAD"}`), map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusBadRequest)
	assertJSONEqual(t, mustBody(t, resp), []byte(`{"__type":"ValidationException","message":"distanceMetric is invalid"}`))
}

func TestS3VectorsStage2IndexPagination(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	createBucket := []byte(`{"vectorBucketName":"demo-bucket"}`)
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/CreateVectorBucket", createBucket, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusOK)

	for _, name := range []string{"a", "b", "c"} {
		req := []byte(fmt.Sprintf(`{"vectorBucketName":"demo-bucket","indexName":"%s","dimension":3,"distanceMetric":"COSINE"}`, name))
		resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/CreateIndex", req, map[string]string{
			"Content-Type": "application/json",
		}, "s3vectors")
		assertStatus(t, resp, http.StatusOK)
	}

	listReq := []byte(`{"vectorBucketName":"demo-bucket","maxResults":2}`)
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/ListIndexes", listReq, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusOK)
	assertJSONEqual(t, mustBody(t, resp), []byte(`{"indexes":[{"indexName":"a","indexArn":"arn:aws:s3vectors:us-east-1:123456789012:bucket/demo-bucket/index/a","creationTime":1700000000,"dimension":3,"distanceMetric":"COSINE"},{"indexName":"b","indexArn":"arn:aws:s3vectors:us-east-1:123456789012:bucket/demo-bucket/index/b","creationTime":1700000000,"dimension":3,"distanceMetric":"COSINE"}],"nextToken":"token-2"}`))

	listReq = []byte(`{"vectorBucketName":"demo-bucket","maxResults":2,"nextToken":"token-2"}`)
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/ListIndexes", listReq, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusOK)
	assertJSONEqual(t, mustBody(t, resp), []byte(`{"indexes":[{"indexName":"c","indexArn":"arn:aws:s3vectors:us-east-1:123456789012:bucket/demo-bucket/index/c","creationTime":1700000000,"dimension":3,"distanceMetric":"COSINE"}],"nextToken":"token-3"}`))
}
