package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestS3VectorsStage1VectorBucketCRUD(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	createReq := []byte(`{"vectorBucketName":"alpha"}`)
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/CreateVectorBucket", createReq, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusOK)
	assertJSONEqual(t, mustBody(t, resp), []byte(`{"vectorBucketArn":"arn:aws:s3vectors:us-east-1:123456789012:bucket/alpha"}`))

	getReq := []byte(`{"vectorBucketName":"alpha"}`)
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/GetVectorBucket", getReq, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusOK)
	assertJSONEqual(t, mustBody(t, resp), []byte(`{"vectorBucketName":"alpha","vectorBucketArn":"arn:aws:s3vectors:us-east-1:123456789012:bucket/alpha","creationTime":1700000000}`))

	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/ListVectorBuckets", nil, nil, "s3vectors")
	assertStatus(t, resp, http.StatusOK)
	assertJSONEqual(t, mustBody(t, resp), []byte(`{"vectorBuckets":[{"vectorBucketName":"alpha","vectorBucketArn":"arn:aws:s3vectors:us-east-1:123456789012:bucket/alpha","creationTime":1700000000}],"nextToken":"token-1"}`))

	deleteReq := []byte(`{"vectorBucketName":"alpha"}`)
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/DeleteVectorBucket", deleteReq, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusOK)
	assertJSONEqual(t, mustBody(t, resp), []byte(`{}`))

	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/GetVectorBucket", getReq, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusNotFound)
}

func TestS3VectorsStage1VectorBucketValidation(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/CreateVectorBucket", []byte("{"), map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusBadRequest)
	assertJSONEqual(t, mustBody(t, resp), []byte(`{"__type":"BadRequestException","message":"invalid JSON"}`))

	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/CreateVectorBucket", []byte(`{}`), map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusBadRequest)
	assertJSONEqual(t, mustBody(t, resp), []byte(`{"__type":"ValidationException","message":"vectorBucketName is required"}`))

	createReq := []byte(`{"vectorBucketName":"dup"}`)
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/CreateVectorBucket", createReq, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/CreateVectorBucket", createReq, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusConflict)
	assertJSONEqual(t, mustBody(t, resp), []byte(`{"__type":"ConflictException","message":"vector bucket already exists"}`))
}

func TestS3VectorsStage1VectorBucketPagination(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, name := range []string{"a", "b", "c"} {
		req := []byte(fmt.Sprintf(`{"vectorBucketName":"%s"}`, name))
		resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/CreateVectorBucket", req, map[string]string{
			"Content-Type": "application/json",
		}, "s3vectors")
		assertStatus(t, resp, http.StatusOK)
	}

	listReq := []byte(`{"maxResults":2}`)
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/ListVectorBuckets", listReq, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusOK)
	assertJSONEqual(t, mustBody(t, resp), []byte(`{"vectorBuckets":[{"vectorBucketName":"a","vectorBucketArn":"arn:aws:s3vectors:us-east-1:123456789012:bucket/a","creationTime":1700000000},{"vectorBucketName":"b","vectorBucketArn":"arn:aws:s3vectors:us-east-1:123456789012:bucket/b","creationTime":1700000000}],"nextToken":"token-2"}`))

	listReq = []byte(`{"maxResults":2,"nextToken":"token-2"}`)
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/ListVectorBuckets", listReq, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusOK)
	assertJSONEqual(t, mustBody(t, resp), []byte(`{"vectorBuckets":[{"vectorBucketName":"c","vectorBucketArn":"arn:aws:s3vectors:us-east-1:123456789012:bucket/c","creationTime":1700000000}],"nextToken":"token-3"}`))
}
