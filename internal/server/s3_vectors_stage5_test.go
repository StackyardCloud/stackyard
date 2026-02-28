package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestS3VectorsStage5PolicyLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	createBucket := []byte(`{"vectorBucketName":"demo-bucket"}`)
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/CreateVectorBucket", createBucket, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusOK)

	putReq := []byte(`{"vectorBucketArn":"arn:aws:s3vectors:us-east-1:123456789012:bucket/demo-bucket","policy":"{\"Version\":\"2012-10-17\",\"Statement\":[]}"}`)
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/PutVectorBucketPolicy", putReq, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusOK)

	getReq := []byte(`{"vectorBucketArn":"arn:aws:s3vectors:us-east-1:123456789012:bucket/demo-bucket"}`)
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/GetVectorBucketPolicy", getReq, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusOK)
	assertJSONEqual(t, mustBody(t, resp), []byte(`{"policy":"{\"Version\":\"2012-10-17\",\"Statement\":[]}"}`))

	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/DeleteVectorBucketPolicy", getReq, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/GetVectorBucketPolicy", getReq, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusNotFound)
}

func TestS3VectorsStage5TagLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	createBucket := []byte(`{"vectorBucketName":"demo-bucket"}`)
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/CreateVectorBucket", createBucket, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusOK)

	tagReq := []byte(`{"resourceArn":"arn:aws:s3vectors:us-east-1:123456789012:bucket/demo-bucket","tags":{"env":"dev","team":"search"}}`)
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/TagResource", tagReq, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusOK)

	listReq := []byte(`{"resourceArn":"arn:aws:s3vectors:us-east-1:123456789012:bucket/demo-bucket"}`)
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/ListTagsForResource", listReq, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusOK)
	assertJSONEqual(t, mustBody(t, resp), []byte(`{"tags":{"env":"dev","team":"search"}}`))

	untagReq := []byte(`{"resourceArn":"arn:aws:s3vectors:us-east-1:123456789012:bucket/demo-bucket","tagKeys":["team"]}`)
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/UntagResource", untagReq, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusOK)

	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/ListTagsForResource", listReq, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusOK)
	assertJSONEqual(t, mustBody(t, resp), []byte(`{"tags":{"env":"dev"}}`))
}

func TestS3VectorsStage5TagMissingResource(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	tagReq := []byte(`{"resourceArn":"arn:aws:s3vectors:us-east-1:123456789012:bucket/missing","tags":{"env":"dev"}}`)
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/TagResource", tagReq, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusNotFound)
}

func TestS3VectorsStage5InvalidJSON(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/PutVectorBucketPolicy", []byte("{"), map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusBadRequest)
	assertJSONEqual(t, mustBody(t, resp), []byte(`{"__type":"BadRequestException","message":"invalid JSON"}`))
}
