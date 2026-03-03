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

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	region := getenv("AWS_REGION", "us-east-1")
	clusterName := getenv("STACKYARD_CLUSTER_NAME", "eks-cluster")
	nodegroupName := getenv("STACKYARD_NODEGROUP_NAME", "eks-nodegroup")
	fargateProfileName := getenv("STACKYARD_FARGATE_PROFILE_NAME", "eks-fargate")
	addonName := getenv("STACKYARD_ADDON_NAME", "vpc-cni")
	identityProviderConfigName := getenv("STACKYARD_IDP_CONFIG_NAME", "eks-idp")
	accessPrincipalARN := getenv("STACKYARD_ACCESS_PRINCIPAL_ARN", "arn:aws:iam::123456789012:role/stackyard-eks-access")
	accessPolicyARN := getenv("STACKYARD_ACCESS_POLICY_ARN", "arn:aws:eks::aws:cluster-access-policy/AmazonEKSViewPolicy")
	roleARN := getenv("STACKYARD_ROLE_ARN", "arn:aws:iam::123456789012:role/stackyard-eks")

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard EKS advanced client using %s\n", endpoint)

	requests := []requestCase{
		{
			Action: "CreateCluster",
			Method: http.MethodPost,
			Path:   "/clusters",
			Payload: map[string]any{
				"name":    clusterName,
				"roleArn": roleARN,
				"resourcesVpcConfig": map[string]any{
					"subnetIds":            []string{"subnet-12345678"},
					"endpointPublicAccess": true,
				},
			},
		},
		{
			Action: "UpdateClusterConfig",
			Method: http.MethodPost,
			Path:   "/clusters/" + clusterName + "/update-config",
			Payload: map[string]any{
				"resourcesVpcConfig": map[string]any{
					"endpointPublicAccess": false,
				},
			},
		},
		{
			Action: "UpdateClusterVersion",
			Method: http.MethodPost,
			Path:   "/clusters/" + clusterName + "/updates",
			Payload: map[string]any{
				"version": "1.30",
			},
		},
		{
			Action: "CreateNodegroup",
			Method: http.MethodPost,
			Path:   "/clusters/" + clusterName + "/node-groups",
			Payload: map[string]any{
				"nodegroupName": nodegroupName,
				"nodeRole":      roleARN,
				"subnets":       []string{"subnet-12345678"},
			},
		},
		{
			Action: "UpdateNodegroupConfig",
			Method: http.MethodPost,
			Path:   "/clusters/" + clusterName + "/node-groups/" + nodegroupName + "/update-config",
			Payload: map[string]any{
				"labels": map[string]string{"env": "dev"},
			},
		},
		{
			Action: "UpdateNodegroupVersion",
			Method: http.MethodPost,
			Path:   "/clusters/" + clusterName + "/node-groups/" + nodegroupName + "/update-version",
			Payload: map[string]any{
				"version": "1.30",
			},
		},
		{
			Action: "CreateFargateProfile",
			Method: http.MethodPost,
			Path:   "/clusters/" + clusterName + "/fargate-profiles",
			Payload: map[string]any{
				"fargateProfileName":  fargateProfileName,
				"podExecutionRoleArn": roleARN,
				"subnets":             []string{"subnet-12345678"},
				"selectors": []map[string]any{
					{"namespace": "default"},
				},
			},
		},
		{
			Action: "CreateAddon",
			Method: http.MethodPost,
			Path:   "/clusters/" + clusterName + "/addons",
			Payload: map[string]any{
				"addonName":             addonName,
				"addonVersion":          "latest",
				"serviceAccountRoleArn": roleARN,
			},
		},
		{
			Action: "AssociateIdentityProviderConfig",
			Method: http.MethodPost,
			Path:   "/clusters/" + clusterName + "/identity-provider-configs/associate",
			Payload: map[string]any{
				"oidc": map[string]any{
					"identityProviderConfigName": identityProviderConfigName,
					"issuerUrl":                  "https://issuer.example.com",
					"clientId":                   "sts.amazonaws.com",
				},
			},
		},
		{
			Action: "CreateAccessEntry",
			Method: http.MethodPost,
			Path:   "/clusters/" + clusterName + "/access-entries",
			Payload: map[string]any{
				"principalArn": accessPrincipalARN,
				"type":         "STANDARD",
			},
		},
		{
			Action: "AssociateAccessPolicy",
			Method: http.MethodPost,
			Path:   "/clusters/" + clusterName + "/access-entries/" + url.PathEscape(accessPrincipalARN) + "/access-policies",
			Payload: map[string]any{
				"policyArn": accessPolicyARN,
				"accessScope": map[string]any{
					"type": "cluster",
				},
			},
		},
		{Action: "ListAddons", Method: http.MethodGet, Path: "/clusters/" + clusterName + "/addons"},
		{Action: "DescribeAddonVersions", Method: http.MethodGet, Path: "/addons/supported-versions?addonName=" + addonName},
		{Action: "ListIdentityProviderConfigs", Method: http.MethodGet, Path: "/clusters/" + clusterName + "/identity-provider-configs"},
		{Action: "ListAccessPolicies", Method: http.MethodGet, Path: "/access-policies"},
		{Action: "ListAccessEntries", Method: http.MethodGet, Path: "/clusters/" + clusterName + "/access-entries"},
		{Action: "ListNodegroups", Method: http.MethodGet, Path: "/clusters/" + clusterName + "/node-groups"},
		{Action: "ListFargateProfiles", Method: http.MethodGet, Path: "/clusters/" + clusterName + "/fargate-profiles"},
		{Action: "ListUpdates", Method: http.MethodGet, Path: "/clusters/" + clusterName + "/updates"},
	}

	var lastUpdateID string
	for _, reqCase := range requests {
		status, body, err := eksRequest(ctx, endpoint, region, creds, reqCase.Method, reqCase.Path, reqCase.Payload)
		if err != nil {
			exitf("%s request failed: %v", reqCase.Action, err)
		}
		expectStatus(reqCase.Action, status, body, http.StatusOK)
		logf("%s returned %d", reqCase.Action, status)

		if reqCase.Action == "UpdateClusterVersion" || reqCase.Action == "UpdateNodegroupVersion" || reqCase.Action == "UpdateNodegroupConfig" {
			if id := extractUpdateID(body); id != "" {
				lastUpdateID = id
			}
		}
	}

	if lastUpdateID != "" {
		status, body, err := eksRequest(ctx, endpoint, region, creds, http.MethodGet, "/clusters/"+clusterName+"/updates/"+lastUpdateID, nil)
		if err != nil {
			exitf("DescribeUpdate request failed: %v", err)
		}
		expectStatus("DescribeUpdate", status, body, http.StatusOK)
		logf("DescribeUpdate returned %d", status)
	}

	teardown := []requestCase{
		{Action: "DisassociateAccessPolicy", Method: http.MethodDelete, Path: "/clusters/" + clusterName + "/access-entries/" + url.PathEscape(accessPrincipalARN) + "/access-policies?policyArn=" + url.QueryEscape(accessPolicyARN)},
		{Action: "DeleteAccessEntry", Method: http.MethodDelete, Path: "/clusters/" + clusterName + "/access-entries/" + url.PathEscape(accessPrincipalARN)},
		{Action: "DisassociateIdentityProviderConfig", Method: http.MethodPost, Path: "/clusters/" + clusterName + "/identity-provider-configs/disassociate", Payload: map[string]any{"identityProviderConfig": map[string]any{"type": "oidc", "name": identityProviderConfigName}}},
		{Action: "DeleteAddon", Method: http.MethodDelete, Path: "/clusters/" + clusterName + "/addons/" + addonName},
		{Action: "DeleteFargateProfile", Method: http.MethodDelete, Path: "/clusters/" + clusterName + "/fargate-profiles/" + fargateProfileName},
		{Action: "DeleteNodegroup", Method: http.MethodDelete, Path: "/clusters/" + clusterName + "/node-groups/" + nodegroupName},
		{Action: "DeleteCluster", Method: http.MethodDelete, Path: "/clusters/" + clusterName},
	}
	for _, reqCase := range teardown {
		status, body, err := eksRequest(ctx, endpoint, region, creds, reqCase.Method, reqCase.Path, reqCase.Payload)
		if err != nil {
			exitf("%s request failed: %v", reqCase.Action, err)
		}
		expectStatus(reqCase.Action, status, body, http.StatusOK)
		logf("%s returned %d", reqCase.Action, status)
	}

	fmt.Println("Done.")
}

type requestCase struct {
	Action  string
	Method  string
	Path    string
	Payload map[string]any
}

func extractUpdateID(body []byte) string {
	var parsed struct {
		Update struct {
			ID string `json:"id"`
		} `json:"update"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return ""
	}
	return strings.TrimSpace(parsed.Update.ID)
}

func eksRequest(
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

	requestURL := strings.TrimRight(endpoint, "/") + path
	req, err := http.NewRequestWithContext(ctx, method, requestURL, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	credentialValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credentialValue, req, hashSHA256(body), "eks", region, time.Now()); err != nil {
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

func expectStatus(action string, status int, body []byte, expected int) {
	if status != expected {
		exitf("expected %s to return %d, got %d: %s", action, expected, status, strings.TrimSpace(string(body)))
	}
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
