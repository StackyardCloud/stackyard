// Command iot-things-graph runs AWS IoT Things Graph against Stackyard:
// upload a camera device model and device, associate the device with a thing,
// create a flow and a system that uses it, deploy a cloud system instance,
// read its flow execution, then undeploy and clean up.
//
// The SDK has no Things Graph module in this repository's go.mod, so requests
// are signed by hand.
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

const (
	signingName  = "iotthingsgraph"
	targetPrefix = "IotThingsGraphFrontEndService."
	contentType  = "application/x-amz-json-1.1"
	ns           = "urn:tdm:us-east-1/123456789012/default:"
)

func graphql(text string) map[string]any { return map[string]any{"language": "GRAPHQL", "text": text} }

func main() {
	c := newClient()
	fmt.Printf("Stackyard IoT Things Graph client using %s\n", c.endpoint)
	run := fmt.Sprintf("%d", time.Now().UnixNano())

	entities := "type Camera @deviceModel(id: \"" + ns + "deviceModel:Camera\", capability: \"" + ns + "capability:Capture\") {ignore:void}\n" +
		"type Capture @capability(id: \"" + ns + "capability:Capture\") {ignore:void}\n" +
		"type LobbyCamera @device(id: \"" + ns + "device:LobbyCamera\", deviceModel: \"" + ns + "deviceModel:Camera\") {ignore:void}\n"
	up := c.call("UploadEntityDefinitions", map[string]any{"document": graphql(entities)})["uploadId"]
	for i := 0; i < 2; i++ {
		fmt.Println("upload:", c.call("GetUploadStatus", map[string]any{"uploadId": up})["uploadStatus"])
	}
	fmt.Println("namespace version:", c.call("DescribeNamespace", map[string]any{})["namespaceVersion"])

	thing := "lobby-cam-" + run
	c.call("AssociateEntityToThing", map[string]any{"thingName": thing, "entityId": ns + "device:LobbyCamera"})
	for _, t := range c.call("SearchThings", map[string]any{"entityId": ns + "deviceModel:Camera"})["things"].([]any) {
		fmt.Println("camera thing:", t.(map[string]any)["thingName"])
	}

	flow := ns + "Workflow:Snap" + run
	c.call("CreateFlowTemplate", map[string]any{"definition": graphql("{ query Snap @workflowType(id: \"" + flow +
		"\") { variables { cam @property(id: \"" + ns + "deviceModel:Camera\") } } }")})
	system := ns + "System:Lobby" + run
	c.call("CreateSystemTemplate", map[string]any{"definition": graphql("{ type Lobby @systemType(id: \"" + system +
		"\") { camera: Camera @thing(id: \"" + ns + "deviceModel:Camera\") snap: Workflow @workflow(id: \"" + flow + "\") } }")})
	inst := ns + "Deployment:Lobby" + run
	c.call("CreateSystemInstance", map[string]any{
		"definition": graphql("query Lobby @deployment(id: \"" + inst + "\", systemId: \"" + system + "\") { camera(deviceId: \"" + thing + "\") }"),
		"target":     "CLOUD", "flowActionsRoleArn": "arn:aws:iam::123456789012:role/things-graph",
	})
	fmt.Println("deploy:", c.call("DeploySystemInstance", map[string]any{"id": inst})["summary"].(map[string]any)["status"])
	desc := c.call("GetSystemInstance", map[string]any{"id": inst})["description"].(map[string]any)
	fmt.Println("instance:", desc["summary"].(map[string]any)["status"])

	for _, e := range c.call("SearchFlowExecutions", map[string]any{"systemInstanceId": inst})["summaries"].([]any) {
		exec := e.(map[string]any)
		fmt.Printf("execution %v: %v\n", exec["flowExecutionId"], exec["status"])
		for _, m := range c.call("ListFlowExecutionMessages", map[string]any{"flowExecutionId": exec["flowExecutionId"]})["messages"].([]any) {
			fmt.Println("  ", m.(map[string]any)["eventType"])
		}
	}

	c.call("UndeploySystemInstance", map[string]any{"id": inst})
	fmt.Println("undeploy:", c.call("GetSystemInstance", map[string]any{"id": inst})["description"].(map[string]any)["summary"].(map[string]any)["status"])
	c.call("DeleteSystemInstance", map[string]any{"id": inst})
	c.call("DeleteSystemTemplate", map[string]any{"id": system})
	c.call("DeleteFlowTemplate", map[string]any{"id": flow})
	c.call("DissociateEntityFromThing", map[string]any{"thingName": thing, "entityType": "DEVICE"})
	fmt.Println("cleaned up", inst)
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

// call sends one awsJson operation and exits unless it returns 2xx.
func (c client) call(op string, payload any) map[string]any {
	body, err := json.Marshal(payload)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	raw := c.send(op, http.MethodPost, signingName, c.endpoint+"/", body, map[string]string{
		"Content-Type": contentType, "X-Amz-Target": targetPrefix + op,
	})
	out := map[string]any{}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			exitf("%s: decode: %v", op, err)
		}
	}
	return out
}

// send signs a request for service and exits unless it returns 2xx.
func (c client) send(op, method, service, target string, body []byte, headers map[string]string) []byte {
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, method, target, bytes.NewReader(body))
	if err != nil {
		exitf("%s: %v", op, err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	creds, err := c.creds.Retrieve(ctx)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	sum := sha256.Sum256(body)
	payloadHash := hex.EncodeToString(sum[:])
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)
	if err := v4.NewSigner().SignHTTP(ctx, creds, req, payloadHash, service, c.region, time.Now()); err != nil {
		exitf("%s: %v", op, err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		exitf("%s: HTTP %d: %s", op, resp.StatusCode, raw)
	}
	return raw
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
