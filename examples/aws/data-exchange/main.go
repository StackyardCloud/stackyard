package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

type apiCall struct {
	Name   string
	Method string
	Path   string
	Body   []byte
}

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	region := getenv("AWS_REGION", "us-east-1")

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard Data Exchange advanced client using %s\n", endpoint)

	grantARN := url.QueryEscape("arn:aws:dataexchange:us-east-1:123456789012:data-grants/dg-000001")
	resourceARN := url.QueryEscape("arn:aws:dataexchange:us-east-1:123456789012:data-sets/ds-000001")

	calls := []apiCall{
		{Name: "CreateDataSet", Method: http.MethodPost, Path: "/v1/data-sets", Body: []byte(`{"Name":"example-data-set"}`)},
		{Name: "CreateRevision", Method: http.MethodPost, Path: "/v1/data-sets/ds-000001/revisions", Body: []byte(`{"Comment":"example-revision"}`)},
		{Name: "CreateJob", Method: http.MethodPost, Path: "/v1/jobs", Body: []byte(`{"Type":"IMPORT_ASSETS_FROM_S3"}`)},
		{Name: "StartJob", Method: http.MethodPatch, Path: "/v1/jobs/job-000001", Body: []byte(`{"State":"IN_PROGRESS"}`)},
		{Name: "GetJob", Method: http.MethodGet, Path: "/v1/jobs/job-000001"},
		{Name: "CreateEventAction", Method: http.MethodPost, Path: "/v1/event-actions", Body: []byte(`{"Name":"example-event-action"}`)},
		{Name: "CreateDataGrant", Method: http.MethodPost, Path: "/v1/data-grants", Body: []byte(`{}`)},
		{Name: "AcceptDataGrant", Method: http.MethodPost, Path: "/v1/data-grants/" + grantARN + "/accept", Body: []byte(`{}`)},
		{Name: "SendDataSetNotification", Method: http.MethodPost, Path: "/v1/data-sets/ds-000001/notification", Body: []byte(`{"Comment":"notify subscribers"}`)},
		{Name: "SendApiAsset", Method: http.MethodPost, Path: "/v1?assetId=asset-000001", Body: []byte(`{}`)},
		{Name: "ListDataSets", Method: http.MethodGet, Path: "/v1/data-sets?maxResults=10&origin=OWNED"},
		{Name: "ListDataSetRevisions", Method: http.MethodGet, Path: "/v1/data-sets/ds-000001/revisions?maxResults=10"},
		{Name: "ListJobs", Method: http.MethodGet, Path: "/v1/jobs?dataSetId=ds-000001&revisionId=rev-000001"},
		{Name: "TagResource", Method: http.MethodPost, Path: "/tags/" + resourceARN, Body: []byte(`{"Tags":{"env":"example","owner":"stackyard"}}`)},
		{Name: "ListTagsForResource", Method: http.MethodGet, Path: "/tags/" + resourceARN},
		{Name: "UntagResource", Method: http.MethodDelete, Path: "/tags/" + resourceARN + "?tagKeys=env"},
	}

	for _, call := range calls {
		status, body, err := dataExchangeRequest(ctx, endpoint, region, creds, call.Method, call.Path, call.Body)
		if err != nil {
			exitf("%s request failed: %v", call.Name, err)
		}
		if status != http.StatusOK {
			exitf("%s expected 200, got %d: %s", call.Name, status, strings.TrimSpace(string(body)))
		}
		if strings.Contains(string(body), "NotImplemented") {
			exitf("%s returned NotImplemented: %s", call.Name, strings.TrimSpace(string(body)))
		}
		logf("%s returned %d", call.Name, status)
		trimmed := strings.TrimSpace(string(body))
		if trimmed != "" {
			fmt.Println(trimmed)
		}
	}

	fmt.Println("Done.")
}

func dataExchangeRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	method, path string,
	body []byte,
) (int, []byte, error) {
	base, err := url.Parse(strings.TrimRight(endpoint, "/"))
	if err != nil {
		return 0, nil, err
	}
	cleanPath, rawQuery, _ := strings.Cut(path, "?")
	base.Path = cleanPath
	base.RawQuery = rawQuery

	if body == nil {
		body = []byte{}
	}
	req, err := http.NewRequestWithContext(ctx, method, base.String(), bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	if method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch {
		req.Header.Set("Content-Type", "application/json")
	}

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "dataexchange", region, time.Now()); err != nil {
		return 0, nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, err
	}
	return resp.StatusCode, respBody, nil
}

func hashSHA256(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func getenv(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return def
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func logf(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}

