package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

type requestCase struct {
	Action  string
	Method  string
	Path    string
	Payload map[string]any
}

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	region := getenv("AWS_REGION", "us-east-1")
	meshName := getenv("STACKYARD_MESH_NAME", "stackyard-mesh")
	virtualNodeName := getenv("STACKYARD_VIRTUAL_NODE_NAME", "stackyard-node")
	virtualRouterName := getenv("STACKYARD_VIRTUAL_ROUTER_NAME", "stackyard-router")
	routeName := getenv("STACKYARD_ROUTE_NAME", "stackyard-route")
	virtualServiceName := getenv("STACKYARD_VIRTUAL_SERVICE_NAME", "stackyard.local")

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard App Mesh advanced client using %s\n", endpoint)

	meshARN := fmt.Sprintf("arn:aws:appmesh:%s:%s:mesh/%s", region, "123456789012", meshName)

	requests := []requestCase{
		{
			Action: "CreateMesh",
			Method: http.MethodPut,
			Path:   "/v20190125/meshes",
			Payload: map[string]any{
				"meshName": meshName,
			},
		},
		{
			Action: "CreateVirtualNode",
			Method: http.MethodPut,
			Path:   "/v20190125/meshes/" + meshName + "/virtualNodes",
			Payload: map[string]any{
				"virtualNodeName": virtualNodeName,
			},
		},
		{
			Action: "CreateVirtualRouter",
			Method: http.MethodPut,
			Path:   "/v20190125/meshes/" + meshName + "/virtualRouters",
			Payload: map[string]any{
				"virtualRouterName": virtualRouterName,
			},
		},
		{
			Action: "CreateRoute",
			Method: http.MethodPut,
			Path:   "/v20190125/meshes/" + meshName + "/virtualRouter/" + virtualRouterName + "/routes",
			Payload: map[string]any{
				"routeName": routeName,
			},
		},
		{
			Action: "CreateVirtualService",
			Method: http.MethodPut,
			Path:   "/v20190125/meshes/" + meshName + "/virtualServices",
			Payload: map[string]any{
				"virtualServiceName": virtualServiceName,
			},
		},
		{
			Action: "TagResource",
			Method: http.MethodPut,
			Path:   "/v20190125/tag",
			Payload: map[string]any{
				"resourceArn": meshARN,
				"tags": map[string]string{
					"env": "dev",
				},
			},
		},
		{
			Action:  "ListMeshes",
			Method:  http.MethodGet,
			Path:    "/v20190125/meshes",
			Payload: nil,
		},
		{
			Action:  "DescribeVirtualService",
			Method:  http.MethodGet,
			Path:    "/v20190125/meshes/" + meshName + "/virtualServices/" + virtualServiceName,
			Payload: nil,
		},
		{
			Action:  "DeleteVirtualService",
			Method:  http.MethodDelete,
			Path:    "/v20190125/meshes/" + meshName + "/virtualServices/" + virtualServiceName,
			Payload: nil,
		},
		{
			Action:  "DeleteRoute",
			Method:  http.MethodDelete,
			Path:    "/v20190125/meshes/" + meshName + "/virtualRouter/" + virtualRouterName + "/routes/" + routeName,
			Payload: nil,
		},
		{
			Action:  "DeleteVirtualRouter",
			Method:  http.MethodDelete,
			Path:    "/v20190125/meshes/" + meshName + "/virtualRouters/" + virtualRouterName,
			Payload: nil,
		},
		{
			Action:  "DeleteVirtualNode",
			Method:  http.MethodDelete,
			Path:    "/v20190125/meshes/" + meshName + "/virtualNodes/" + virtualNodeName,
			Payload: nil,
		},
		{
			Action:  "DeleteMesh",
			Method:  http.MethodDelete,
			Path:    "/v20190125/meshes/" + meshName,
			Payload: nil,
		},
	}

	for _, reqCase := range requests {
		status, body, err := appMeshRequest(ctx, endpoint, region, creds, reqCase.Method, reqCase.Path, reqCase.Payload)
		if err != nil {
			exitf("%s request failed: %v", reqCase.Action, err)
		}
		if status < 200 || status >= 300 {
			exitf("%s returned HTTP %d: %s", reqCase.Action, status, strings.TrimSpace(string(body)))
		}
		logf("%s returned %d", reqCase.Action, status)
	}

	fmt.Println("Done.")
}

func appMeshRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	method, path string,
	payload map[string]any,
) (int, []byte, error) {
	var body []byte
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return 0, nil, err
		}
		body = encoded
	}

	url := strings.TrimRight(endpoint, "/") + path
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "appmesh", region, time.Now()); err != nil {
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
