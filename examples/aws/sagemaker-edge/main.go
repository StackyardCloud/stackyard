// Command sagemaker-edge runs the Amazon SageMaker Edge Manager agent API against
// Stackyard: register a device fleet and devices with the SageMaker control
// plane, start an edge deployment stage, and act as a device's agent: check
// its registration, fetch its deployment and report the result in a
// heartbeat, which the deployment plan then counts.
//
// The SDK has no SageMaker Edge module in this repository's go.mod, so requests are
// signed by hand.
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
)

func main() {
	c := newClient()
	fmt.Printf("Stackyard SageMaker Edge client using %s\n", c.endpoint)
	run := fmt.Sprintf("%d", time.Now().UnixNano()%1_000_000_000)
	fleet, plan := "cameras-"+run, "detector-rollout-"+run
	c.sagemaker("CreateDeviceFleet", map[string]any{"DeviceFleetName": fleet,
		"OutputConfig": map[string]any{"S3OutputLocation": "s3://edge-example/" + fleet}})
	c.sagemaker("RegisterDevices", map[string]any{"DeviceFleetName": fleet, "Devices": []any{
		map[string]any{"DeviceName": "door-cam-" + run}, map[string]any{"DeviceName": "yard-cam-" + run}}})
	c.sagemaker("CreateEdgeDeploymentPlan", map[string]any{"EdgeDeploymentPlanName": plan, "DeviceFleetName": fleet,
		"ModelConfigs": []any{map[string]any{"ModelHandle": "detector", "EdgePackagingJobName": "detector-pkg"}},
		"Stages": []any{map[string]any{"StageName": "door", "DeviceSelectionConfig": map[string]any{
			"DeviceSubsetType": "NAMECONTAINS", "DeviceNameContains": "door"}}}})
	c.sagemaker("StartEdgeDeploymentStage", map[string]any{"EdgeDeploymentPlanName": plan, "StageName": "door"})
	logf("fleet %s with two devices; stage door of %s started", fleet, plan)

	device := map[string]any{"DeviceName": "door-cam-" + run, "DeviceFleetName": fleet}
	reg := c.restOK(http.MethodPost, "/GetDeviceRegistration", device)
	logf("registration: %v (cache %vs)", reg["DeviceRegistration"], reg["CacheTTL"])
	deployments := c.restOK(http.MethodPost, "/GetDeployments", device)["Deployments"].([]any)
	if len(deployments) != 1 {
		exitf("expected one deployment, got %v", deployments)
	}
	logf("deployment: %v", deployments[0])
	other := c.restOK(http.MethodPost, "/GetDeployments", map[string]any{"DeviceName": "yard-cam-" + run, "DeviceFleetName": fleet})
	logf("yard camera deployments (not in the stage): %v", other["Deployments"])

	heartbeat := map[string]any{"DeviceName": device["DeviceName"], "DeviceFleetName": fleet, "AgentVersion": "1.20",
		"Models": []any{map[string]any{"ModelName": "detector", "ModelVersion": "1", "LatestInference": float64(time.Now().Unix())}},
		"DeploymentResult": map[string]any{"DeploymentName": plan, "DeploymentStatus": "SUCCESS",
			"DeploymentModels": []any{map[string]any{"ModelHandle": "detector", "Status": "SUCCESS"}}}}
	c.restOK(http.MethodPost, "/SendHeartbeat", heartbeat)
	status := c.sagemaker("DescribeEdgeDeploymentPlan", map[string]any{"EdgeDeploymentPlanName": plan})
	logf("plan after the heartbeat: success=%v pending=%v stages=%v", status["EdgeDeploymentSuccess"], status["EdgeDeploymentPending"], status["Stages"])
	desc := c.sagemaker("DescribeDevice", device)
	logf("device agent=%v models=%v", desc["AgentVersion"], desc["Models"])

	c.sagemaker("DeleteEdgeDeploymentPlan", map[string]any{"EdgeDeploymentPlanName": plan})
	c.sagemaker("DeregisterDevices", map[string]any{"DeviceFleetName": fleet, "DeviceNames": []string{"door-cam-" + run, "yard-cam-" + run}})
	c.sagemaker("DeleteDeviceFleet", map[string]any{"DeviceFleetName": fleet})
	fmt.Println("SageMaker Edge example completed")
}

type client struct {
	endpoint, region string
	creds            aws.CredentialsProvider
}

func newClient() client {
	return client{
		endpoint: strings.TrimRight(getenv("STACKYARD_ENDPOINT", "http://localhost:4566"), "/"),
		region:   getenv("AWS_REGION", "us-east-1"),
		creds: credentials.NewStaticCredentialsProvider(
			getenv("AWS_ACCESS_KEY_ID", "stackyard"), getenv("AWS_SECRET_ACCESS_KEY", "stackyard"), ""),
	}
}

// sagemaker calls the SageMaker control plane (JSON 1.1).
func (c client) sagemaker(action string, payload any) map[string]any {
	status, body := c.do(http.MethodPost, "/", payload, map[string]string{
		"Content-Type": "application/x-amz-json-1.1", "X-Amz-Target": "SageMaker." + action,
	})
	return expectOK(action, status, body)
}

// rest calls a REST JSON SageMaker data plane; payload may be nil.
func (c client) rest(method, path string, payload any) (int, map[string]any) {
	status, body := c.do(method, path, payload, map[string]string{"Content-Type": "application/json"})
	out := map[string]any{}
	if len(bytes.TrimSpace(body)) > 0 {
		if err := json.Unmarshal(body, &out); err != nil {
			exitf("%s %s: decode %s: %v", method, path, body, err)
		}
	}
	return status, out
}

func (c client) restOK(method, path string, payload any) map[string]any {
	status, out := c.rest(method, path, payload)
	if status != http.StatusOK {
		exitf("%s %s failed: status=%d %v", method, path, status, out)
	}
	return out
}

func (c client) do(method, path string, payload any, headers map[string]string) (int, []byte) {
	var body []byte
	if payload != nil {
		var err error
		if body, err = json.Marshal(payload); err != nil {
			exitf("%s %s: %v", method, path, err)
		}
	}
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, method, c.endpoint+path, bytes.NewReader(body))
	if err != nil {
		exitf("%s %s: %v", method, path, err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	creds, err := c.creds.Retrieve(ctx)
	if err != nil {
		exitf("credentials: %v", err)
	}
	sum := sha256.Sum256(body)
	hash := hex.EncodeToString(sum[:])
	req.Header.Set("X-Amz-Content-Sha256", hash)
	if err := v4.NewSigner().SignHTTP(ctx, creds, req, hash, "sagemaker", c.region, time.Now()); err != nil {
		exitf("sign: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		exitf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	out, err := io.ReadAll(resp.Body)
	if err != nil {
		exitf("read %s %s: %v", method, path, err)
	}
	return resp.StatusCode, out
}

func expectOK(action string, status int, body []byte) map[string]any {
	if status != http.StatusOK {
		exitf("%s failed: status=%d body=%s", action, status, strings.TrimSpace(string(body)))
	}
	out := map[string]any{}
	if len(bytes.TrimSpace(body)) > 0 {
		_ = json.Unmarshal(body, &out)
	}
	return out
}

func getenv(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func logf(format string, args ...any) {
	fmt.Printf("  "+format+"\n", args...)
}
