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

type rpcCall struct {
	Name    string
	Action  string
	Payload map[string]any
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

	fmt.Printf("Stackyard WorkMail advanced client using %s\n", endpoint)

	for _, call := range []rpcCall{
		{Name: "CreateOrganization", Action: "CreateOrganization", Payload: map[string]any{"Alias": "advanced-workmail"}},
		{Name: "ListOrganizations", Action: "ListOrganizations", Payload: map[string]any{"MaxResults": 10}},
		{Name: "RegisterMailDomain", Action: "RegisterMailDomain", Payload: map[string]any{"OrganizationId": "m-000001", "DomainName": "advanced-workmail.example.com"}},
		{Name: "UpdateDefaultMailDomain", Action: "UpdateDefaultMailDomain", Payload: map[string]any{"OrganizationId": "m-000001", "DomainName": "advanced-workmail.example.com"}},
		{Name: "CreateUser", Action: "CreateUser", Payload: map[string]any{"OrganizationId": "m-000001", "Name": "advanced-user", "DisplayName": "Advanced User"}},
		{Name: "CreateGroup", Action: "CreateGroup", Payload: map[string]any{"OrganizationId": "m-000001", "Name": "advanced-group"}},
		{Name: "AssociateMemberToGroup", Action: "AssociateMemberToGroup", Payload: map[string]any{"OrganizationId": "m-000001", "GroupId": "g-000001", "MemberId": "u-000001"}},
		{Name: "CreateResource", Action: "CreateResource", Payload: map[string]any{"OrganizationId": "m-000001", "Name": "advanced-room", "Type": "ROOM"}},
		{Name: "PutMailboxPermissions", Action: "PutMailboxPermissions", Payload: map[string]any{"OrganizationId": "m-000001", "EntityId": "u-000001", "GranteeId": "u-000001", "PermissionValues": []string{"FULL_ACCESS"}}},
		{Name: "PutAccessControlRule", Action: "PutAccessControlRule", Payload: map[string]any{"OrganizationId": "m-000001", "Name": "advanced-acl", "Effect": "ALLOW"}},
		{Name: "CreateMobileDeviceAccessRule", Action: "CreateMobileDeviceAccessRule", Payload: map[string]any{"OrganizationId": "m-000001", "Name": "advanced-mobile", "Effect": "ALLOW"}},
		{Name: "CreateImpersonationRole", Action: "CreateImpersonationRole", Payload: map[string]any{"OrganizationId": "m-000001", "Name": "advanced-role", "Type": "FULL_ACCESS"}},
		{Name: "PutIdentityProviderConfiguration", Action: "PutIdentityProviderConfiguration", Payload: map[string]any{"OrganizationId": "m-000001", "AuthenticationMode": "IDENTITY_PROVIDER_ONLY"}},
		{Name: "PutEmailMonitoringConfiguration", Action: "PutEmailMonitoringConfiguration", Payload: map[string]any{"OrganizationId": "m-000001"}},
		{Name: "StartMailboxExportJob", Action: "StartMailboxExportJob", Payload: map[string]any{"OrganizationId": "m-000001", "EntityId": "u-000001", "Description": "advanced export"}},
		{Name: "ListMailboxExportJobs", Action: "ListMailboxExportJobs", Payload: map[string]any{"OrganizationId": "m-000001"}},
		{Name: "TagResource", Action: "TagResource", Payload: map[string]any{"ResourceARN": "arn:aws:workmail:us-east-1:123456789012:organization/m-000001", "Tags": map[string]string{"env": "advanced", "owner": "qa"}}},
		{Name: "ListTagsForResource", Action: "ListTagsForResource", Payload: map[string]any{"ResourceARN": "arn:aws:workmail:us-east-1:123456789012:organization/m-000001"}},
		{Name: "UntagResource", Action: "UntagResource", Payload: map[string]any{"ResourceARN": "arn:aws:workmail:us-east-1:123456789012:organization/m-000001", "TagKeys": []string{"owner"}}},
	} {
		if err := runCall(ctx, endpoint, region, creds, call); err != nil {
			exitf("%s failed: %v", call.Name, err)
		}
	}

	fmt.Println("Done.")
}

func runCall(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, call rpcCall) error {
	status, body, err := workmailRequest(ctx, endpoint, region, creds, call.Action, call.Payload)
	if err != nil {
		return err
	}
	if status >= 200 && status < 300 {
		logf("%s returned %d", call.Name, status)
		return nil
	}
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		trimmed = "<empty body>"
	}
	return fmt.Errorf("HTTP %d: %s", status, trimmed)
}

func workmailRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	action string,
	payload map[string]any,
) (int, []byte, error) {
	var body []byte
	var err error
	if payload == nil {
		body = []byte("{}")
	} else {
		body, err = json.Marshal(payload)
		if err != nil {
			return 0, nil, err
		}
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		strings.TrimRight(endpoint, "/")+"/",
		bytes.NewReader(body),
	)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "WorkMailService."+action)

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "workmail", region, time.Now()); err != nil {
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
