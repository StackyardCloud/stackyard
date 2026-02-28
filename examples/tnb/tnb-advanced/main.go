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

	fmt.Printf("Stackyard AWS TNB advanced client using %s\n", endpoint)

	nsInstanceID := "ns-instance-000001"
	nsPackageID := "nsd-000001"
	vnfPackageID := "vnfpkg-000001"
	operationID := "ns-lcm-op-000001"
	resourceArn := url.PathEscape("arn:aws:tnb:us-east-1:123456789012:nsd/nsd-000001")

	calls := []apiCall{
		{Name: "CreateSolNetworkPackage", Method: http.MethodPost, Path: "/sol/nsd/v1/ns_descriptors", Body: []byte(`{}`)},
		{Name: "ListSolNetworkPackages", Method: http.MethodGet, Path: "/sol/nsd/v1/ns_descriptors?max_results=10"},
		{Name: "GetSolNetworkPackage", Method: http.MethodGet, Path: "/sol/nsd/v1/ns_descriptors/" + url.PathEscape(nsPackageID)},
		{Name: "PutSolNetworkPackageContent", Method: http.MethodPut, Path: "/sol/nsd/v1/ns_descriptors/" + url.PathEscape(nsPackageID) + "/nsd_content", Body: []byte(`{}`)},
		{Name: "ValidateSolNetworkPackageContent", Method: http.MethodPut, Path: "/sol/nsd/v1/ns_descriptors/" + url.PathEscape(nsPackageID) + "/nsd_content/validate", Body: []byte(`{}`)},
		{Name: "CreateSolNetworkInstance", Method: http.MethodPost, Path: "/sol/nslcm/v1/ns_instances", Body: []byte(`{}`)},
		{Name: "InstantiateSolNetworkInstance", Method: http.MethodPost, Path: "/sol/nslcm/v1/ns_instances/" + url.PathEscape(nsInstanceID) + "/instantiate?dry_run=false", Body: []byte(`{}`)},
		{Name: "GetSolNetworkOperation", Method: http.MethodGet, Path: "/sol/nslcm/v1/ns_lcm_op_occs/" + url.PathEscape(operationID)},
		{Name: "CancelSolNetworkOperation", Method: http.MethodPost, Path: "/sol/nslcm/v1/ns_lcm_op_occs/" + url.PathEscape(operationID) + "/cancel", Body: []byte(`{}`)},
		{Name: "CreateSolFunctionPackage", Method: http.MethodPost, Path: "/sol/vnfpkgm/v1/vnf_packages", Body: []byte(`{}`)},
		{Name: "GetSolFunctionPackage", Method: http.MethodGet, Path: "/sol/vnfpkgm/v1/vnf_packages/" + url.PathEscape(vnfPackageID)},
		{Name: "UpdateSolFunctionPackage", Method: http.MethodPatch, Path: "/sol/vnfpkgm/v1/vnf_packages/" + url.PathEscape(vnfPackageID), Body: []byte(`{}`)},
		{Name: "TagResource", Method: http.MethodPost, Path: "/tags/" + resourceArn, Body: []byte(`{"tags":{"env":"example"}}`)},
		{Name: "ListTagsForResource", Method: http.MethodGet, Path: "/tags/" + resourceArn},
		{Name: "UntagResource", Method: http.MethodDelete, Path: "/tags/" + resourceArn + "?tagKeys=env"},
		{Name: "TerminateSolNetworkInstance", Method: http.MethodPost, Path: "/sol/nslcm/v1/ns_instances/" + url.PathEscape(nsInstanceID) + "/terminate", Body: []byte(`{}`)},
	}

	for _, call := range calls {
		if err := runCall(ctx, endpoint, region, creds, call); err != nil {
			exitf("%s failed: %v", call.Name, err)
		}
	}

	fmt.Println("Done.")
}

func runCall(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	call apiCall,
) error {
	status, body, err := tnbRequest(ctx, endpoint, region, creds, call.Method, call.Path, call.Body)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("expected 200, got %d: %s", status, strings.TrimSpace(string(body)))
	}
	if strings.Contains(string(body), "NotImplemented") {
		return fmt.Errorf("returned NotImplemented: %s", strings.TrimSpace(string(body)))
	}
	logf("%s returned %d", call.Name, status)
	return nil
}

func tnbRequest(
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
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "tnb", region, time.Now()); err != nil {
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
