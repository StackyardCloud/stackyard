// Command cloud9 creates an AWS Cloud9 EC2 environment on Stackyard, waits
// for it to become ready, shares it with a teammate, stops its EC2 instance
// through the EC2 API to show the status follow it, and deletes it.
//
// The SDK has no Cloud9 module in this repository's go.mod, so Cloud9
// requests are signed by hand; EC2 uses the SDK.
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
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type client struct {
	endpoint, region string
	creds            aws.CredentialsProvider
}

func main() {
	c := client{
		endpoint: strings.TrimRight(getenv("STACKYARD_ENDPOINT", "http://localhost:4566"), "/"),
		region:   getenv("AWS_REGION", "us-east-1"),
		creds: credentials.NewStaticCredentialsProvider(
			getenv("AWS_ACCESS_KEY_ID", "stackyard"), getenv("AWS_SECRET_ACCESS_KEY", "stackyard"), ""),
	}
	fmt.Printf("Stackyard Cloud9 client using %s\n", c.endpoint)
	name := fmt.Sprintf("dev-%d", time.Now().UnixNano()%1_000_000_000)

	id := c.call("CreateEnvironmentEC2", map[string]any{
		"name": name, "instanceType": "t3.small", "imageId": "amazonlinux-2023-x86_64",
		"automaticStopTimeMinutes": 30, "connectionType": "CONNECT_SSM",
		"tags": []any{map[string]any{"Key": "team", "Value": "platform"}},
	})["environmentId"].(string)
	fmt.Printf("environment %s created\n", id)
	c.waitStatus(id, "ready")

	member := "arn:aws:iam::123456789012:user/teammate"
	c.call("CreateEnvironmentMembership", map[string]any{"environmentId": id, "userArn": member, "permissions": "read-write"})
	members := c.call("DescribeEnvironmentMemberships", map[string]any{"environmentId": id})["memberships"].([]any)
	for _, m := range members {
		mm := m.(map[string]any)
		fmt.Printf("member %s: %s\n", mm["userArn"], mm["permissions"])
	}

	// The environment runs on an EC2 instance; stopping it shows in Cloud9.
	ctx := context.Background()
	ec := ec2.NewFromConfig(aws.Config{Region: c.region, Credentials: c.creds}, func(o *ec2.Options) {
		o.BaseEndpoint = aws.String(c.endpoint)
	})
	out, err := ec.DescribeInstances(ctx, &ec2.DescribeInstancesInput{Filters: []ec2types.Filter{
		{Name: aws.String("tag:aws:cloud9:environment"), Values: []string{id}},
	}})
	if err != nil {
		exitf("DescribeInstances: %v", err)
	}
	var instance string
	for _, r := range out.Reservations {
		for _, inst := range r.Instances {
			for _, tag := range inst.Tags {
				if aws.ToString(tag.Key) == "aws:cloud9:environment" && aws.ToString(tag.Value) == id {
					instance = aws.ToString(inst.InstanceId)
				}
			}
		}
	}
	if instance == "" {
		exitf("no EC2 instance is tagged for environment %s", id)
	}
	if _, err := ec.StopInstances(ctx, &ec2.StopInstancesInput{InstanceIds: []string{instance}}); err != nil {
		exitf("StopInstances: %v", err)
	}
	fmt.Printf("stopped instance %s\n", instance)
	c.waitStatus(id, "stopped")

	c.call("DeleteEnvironment", map[string]any{"environmentId": id})
	c.waitStatus(id, "deleting")
	fmt.Println("environment deleted")
}

// waitStatus polls DescribeEnvironmentStatus until it reports want.
func (c client) waitStatus(id, want string) {
	for i := 0; i < 30; i++ {
		out := c.call("DescribeEnvironmentStatus", map[string]any{"environmentId": id})
		if out["status"] == want {
			fmt.Printf("status %s: %s\n", want, out["message"])
			return
		}
		time.Sleep(250 * time.Millisecond)
	}
	exitf("environment %s never reported %s", id, want)
}

func (c client) call(op string, payload any) map[string]any {
	body, err := json.Marshal(payload)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"/", bytes.NewReader(body))
	if err != nil {
		exitf("%s: %v", op, err)
	}
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "AWSCloud9WorkspaceManagementService."+op)
	creds, err := c.creds.Retrieve(ctx)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	sum := sha256.Sum256(body)
	payloadHash := hex.EncodeToString(sum[:])
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)
	if err := v4.NewSigner().SignHTTP(ctx, creds, req, payloadHash, "cloud9", c.region, time.Now()); err != nil {
		exitf("%s: %v", op, err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		exitf("%s: HTTP %d: %s", op, resp.StatusCode, raw)
	}
	out := map[string]any{}
	if err := json.Unmarshal(raw, &out); err != nil {
		exitf("%s: decode: %v", op, err)
	}
	return out
}

func getenv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
