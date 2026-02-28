package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func readS3VectorsFixture(t *testing.T, name string) []byte {
	t.Helper()
	path := filepath.Join("testdata", "s3vectors", name)
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return body
}

func TestS3VectorsStage0VectorBucketFixtures(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	createReq := readS3VectorsFixture(t, "create-vector-bucket-request.json")
	createResp := readS3VectorsFixture(t, "create-vector-bucket-response.json")
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/CreateVectorBucket", createReq, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusOK)
	assertJSONEqual(t, mustBody(t, resp), createResp)

	getReq := readS3VectorsFixture(t, "get-vector-bucket-request.json")
	getResp := readS3VectorsFixture(t, "get-vector-bucket-response.json")
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/GetVectorBucket", getReq, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusOK)
	assertJSONEqual(t, mustBody(t, resp), getResp)

	listResp := readS3VectorsFixture(t, "list-vector-buckets-response.json")
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/ListVectorBuckets", nil, nil, "s3vectors")
	assertStatus(t, resp, http.StatusOK)
	assertJSONEqual(t, mustBody(t, resp), listResp)

	deleteReq := readS3VectorsFixture(t, "delete-vector-bucket-request.json")
	deleteResp := readS3VectorsFixture(t, "delete-vector-bucket-response.json")
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/DeleteVectorBucket", deleteReq, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusOK)
	assertJSONEqual(t, mustBody(t, resp), deleteResp)
}

func TestS3VectorsStage0IndexFixtures(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	createBucketReq := readS3VectorsFixture(t, "create-vector-bucket-request.json")
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/CreateVectorBucket", createBucketReq, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusOK)

	createReq := readS3VectorsFixture(t, "create-index-request.json")
	createResp := readS3VectorsFixture(t, "create-index-response.json")
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/CreateIndex", createReq, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusOK)
	assertJSONEqual(t, mustBody(t, resp), createResp)

	getReq := readS3VectorsFixture(t, "get-index-request.json")
	getResp := readS3VectorsFixture(t, "get-index-response.json")
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/GetIndex", getReq, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusOK)
	assertJSONEqual(t, mustBody(t, resp), getResp)

	listResp := readS3VectorsFixture(t, "list-indexes-response.json")
	listReq := []byte(`{"vectorBucketName":"demo-bucket"}`)
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/ListIndexes", listReq, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusOK)
	assertJSONEqual(t, mustBody(t, resp), listResp)

	deleteReq := readS3VectorsFixture(t, "delete-index-request.json")
	deleteResp := readS3VectorsFixture(t, "delete-index-response.json")
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/DeleteIndex", deleteReq, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusOK)
	assertJSONEqual(t, mustBody(t, resp), deleteResp)
}

func TestS3VectorsStage0VectorOpsFixtures(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	createBucketReq := readS3VectorsFixture(t, "create-vector-bucket-request.json")
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/CreateVectorBucket", createBucketReq, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusOK)

	createIndexReq := readS3VectorsFixture(t, "create-index-request.json")
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/CreateIndex", createIndexReq, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusOK)

	putReq := readS3VectorsFixture(t, "put-vectors-request.json")
	putResp := readS3VectorsFixture(t, "put-vectors-response.json")
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/PutVectors", putReq, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusOK)
	assertJSONEqual(t, mustBody(t, resp), putResp)

	getReq := readS3VectorsFixture(t, "get-vectors-request.json")
	getResp := readS3VectorsFixture(t, "get-vectors-response.json")
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/GetVectors", getReq, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusOK)
	assertJSONEqual(t, mustBody(t, resp), getResp)

	listReq := readS3VectorsFixture(t, "list-vectors-request.json")
	listResp := readS3VectorsFixture(t, "list-vectors-response.json")
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/ListVectors", listReq, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusOK)
	assertJSONEqual(t, mustBody(t, resp), listResp)

	queryReq := readS3VectorsFixture(t, "query-vectors-request.json")
	queryResp := readS3VectorsFixture(t, "query-vectors-response.json")
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/QueryVectors", queryReq, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusOK)
	assertJSONEqual(t, mustBody(t, resp), queryResp)

	deleteReq := readS3VectorsFixture(t, "delete-vectors-request.json")
	deleteResp := readS3VectorsFixture(t, "delete-vectors-response.json")
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/DeleteVectors", deleteReq, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusOK)
	assertJSONEqual(t, mustBody(t, resp), deleteResp)
}

func TestS3VectorsStage0PolicyAndTaggingFixtures(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	createBucketReq := readS3VectorsFixture(t, "create-vector-bucket-request.json")
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/CreateVectorBucket", createBucketReq, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusOK)

	putPolicyReq := readS3VectorsFixture(t, "put-vector-bucket-policy-request.json")
	putPolicyResp := readS3VectorsFixture(t, "put-vector-bucket-policy-response.json")
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/PutVectorBucketPolicy", putPolicyReq, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusOK)
	assertJSONEqual(t, mustBody(t, resp), putPolicyResp)

	getPolicyReq := readS3VectorsFixture(t, "get-vector-bucket-policy-request.json")
	getPolicyResp := readS3VectorsFixture(t, "get-vector-bucket-policy-response.json")
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/GetVectorBucketPolicy", getPolicyReq, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusOK)
	assertJSONEqual(t, mustBody(t, resp), getPolicyResp)

	deletePolicyReq := readS3VectorsFixture(t, "delete-vector-bucket-policy-request.json")
	deletePolicyResp := readS3VectorsFixture(t, "delete-vector-bucket-policy-response.json")
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/DeleteVectorBucketPolicy", deletePolicyReq, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusOK)
	assertJSONEqual(t, mustBody(t, resp), deletePolicyResp)

	tagReq := readS3VectorsFixture(t, "tag-resource-request.json")
	tagResp := readS3VectorsFixture(t, "tag-resource-response.json")
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/TagResource", tagReq, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusOK)
	assertJSONEqual(t, mustBody(t, resp), tagResp)

	listReq := readS3VectorsFixture(t, "list-tags-request.json")
	listResp := readS3VectorsFixture(t, "list-tags-response.json")
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/ListTagsForResource", listReq, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusOK)
	assertJSONEqual(t, mustBody(t, resp), listResp)

	untagReq := readS3VectorsFixture(t, "untag-resource-request.json")
	untagResp := readS3VectorsFixture(t, "untag-resource-response.json")
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/UntagResource", untagReq, map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusOK)
	assertJSONEqual(t, mustBody(t, resp), untagResp)
}

func TestS3VectorsStage0InvalidJSON(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	want := readS3VectorsFixture(t, "error-invalid-json.json")
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/CreateVectorBucket", []byte("{"), map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusBadRequest)
	assertJSONEqual(t, mustBody(t, resp), want)
}

func TestS3VectorsStage0MissingRequired(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	want := readS3VectorsFixture(t, "error-missing-vector-bucket.json")
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/CreateVectorBucket", []byte("{}"), map[string]string{
		"Content-Type": "application/json",
	}, "s3vectors")
	assertStatus(t, resp, http.StatusBadRequest)
	assertJSONEqual(t, mustBody(t, resp), want)
}

func TestS3VectorsStage0RequiresS3VectorsSigning(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/ListVectorBuckets", nil, nil, "s3tables")
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for wrong signing service, got %d", resp.StatusCode)
	}
}
