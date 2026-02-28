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
	"net/url"
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
	serviceID := getenv("STACKYARD_SERVICE_ID", "svc-00000000000000001")
	listenerID := getenv("STACKYARD_LISTENER_ID", "listener-00000000000000001")
	targetGroupID := getenv("STACKYARD_TARGET_GROUP_ID", "tg-00000000000000001")
	serviceNetworkID := getenv("STACKYARD_SERVICE_NETWORK_ID", "sn-00000000000000001")

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard VPC Lattice advanced client using %s\n", endpoint)

	resourceArn := url.PathEscape("arn:aws:vpc-lattice:us-east-1:123456789012:service/" + serviceID)

	requests := []requestCase{
		{
			Action: "CreateService",
			Method: http.MethodPost,
			Path:   "/services",
			Payload: map[string]any{
				"name":     "stackyard-service",
				"authType": "AWS_IAM",
			},
		},
		{
			Action: "UpdateService",
			Method: http.MethodPatch,
			Path:   "/services/" + serviceID,
			Payload: map[string]any{
				"name": "stackyard-service-updated",
			},
		},
		{
			Action: "CreateListener",
			Method: http.MethodPost,
			Path:   "/services/" + serviceID + "/listeners",
			Payload: map[string]any{
				"name":     "stackyard-listener",
				"protocol": "HTTP",
				"port":     80,
			},
		},
		{
			Action: "BatchUpdateRule",
			Method: http.MethodPatch,
			Path:   "/services/" + serviceID + "/listeners/" + listenerID + "/rules",
			Payload: map[string]any{
				"rules": []any{
					map[string]any{"name": "stackyard-rule", "priority": 10},
				},
			},
		},
		{
			Action: "CreateTargetGroup",
			Method: http.MethodPost,
			Path:   "/targetgroups",
			Payload: map[string]any{
				"name": "stackyard-target-group",
				"type": "IP",
			},
		},
		{
			Action: "RegisterTargets",
			Method: http.MethodPost,
			Path:   "/targetgroups/" + targetGroupID + "/registertargets",
			Payload: map[string]any{
				"targets": []any{map[string]any{"id": "10.0.0.11", "port": 80}},
			},
		},
		{
			Action:  "ListTargets",
			Method:  http.MethodPost,
			Path:    "/targetgroups/" + targetGroupID + "/listtargets?maxResults=10",
			Payload: map[string]any{},
		},
		{
			Action: "CreateServiceNetwork",
			Method: http.MethodPost,
			Path:   "/servicenetworks",
			Payload: map[string]any{
				"name": "stackyard-service-network",
			},
		},
		{
			Action: "CreateServiceNetworkServiceAssociation",
			Method: http.MethodPost,
			Path:   "/servicenetworkserviceassociations",
			Payload: map[string]any{
				"serviceIdentifier":        serviceID,
				"serviceNetworkIdentifier": serviceNetworkID,
			},
		},
		{
			Action: "PutAuthPolicy",
			Method: http.MethodPut,
			Path:   "/authpolicy/" + serviceID,
			Payload: map[string]any{
				"policy": `{"Version":"2012-10-17","Statement":[]}`,
			},
		},
		{
			Action: "TagResource",
			Method: http.MethodPost,
			Path:   "/tags/" + resourceArn,
			Payload: map[string]any{
				"tags": map[string]string{"env": "dev", "team": "stackyard"},
			},
		},
		{
			Action: "ListTagsForResource",
			Method: http.MethodGet,
			Path:   "/tags/" + resourceArn,
		},
		{
			Action: "ListServices",
			Method: http.MethodGet,
			Path:   "/services?maxResults=10",
		},
		{
			Action: "UntagResource",
			Method: http.MethodDelete,
			Path:   "/tags/" + resourceArn + "?tagKeys=team",
		},
	}

	for _, reqCase := range requests {
		status, body, err := vpcLatticeRequest(ctx, endpoint, region, creds, reqCase.Method, reqCase.Path, reqCase.Payload)
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

func vpcLatticeRequest(
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
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "vpc-lattice", region, time.Now()); err != nil {
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
