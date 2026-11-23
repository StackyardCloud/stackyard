// Command serverlessrepo walks an AWS Serverless Application Repository
// lifecycle against Stackyard: publish a SAM application, add a version,
// share it, generate a CloudFormation template and a change set, then delete
// the application.
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
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

const template = `{
  "Transform": "AWS::Serverless-2016-10-31",
  "Parameters": {"Stage": {"Type": "String", "Default": "dev", "AllowedValues": ["dev", "prod"]}},
  "Resources": {"Topic": {"Type": "AWS::SNS::Topic", "Properties": {"TopicName": {"Ref": "Stage"}}}}
}`

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
	fmt.Printf("Stackyard Serverless Application Repository client using %s\n", c.endpoint)

	name := fmt.Sprintf("stackyard-hello-%d", time.Now().UnixNano())
	app := c.call("CreateApplication", http.MethodPost, "/applications", map[string]any{
		"name": name, "author": "stackyard", "description": "Hello from Stackyard",
		"semanticVersion": "1.0.0", "templateBody": template, "readmeBody": "# Hello\n",
	})
	id := app["applicationId"].(string)
	path := "/applications/" + url.PathEscape(id)
	version := app["version"].(map[string]any)
	fmt.Println("Application:", id)
	fmt.Println("Parameters:", len(version["parameterDefinitions"].([]any)), "capabilities:", version["requiredCapabilities"])

	c.call("CreateApplicationVersion", http.MethodPut, path+"/versions/1.1.0", map[string]any{"templateBody": template})
	versions := c.call("ListApplicationVersions", http.MethodGet, path+"/versions", nil)["versions"].([]any)
	fmt.Println("Versions:", len(versions))

	c.call("PutApplicationPolicy", http.MethodPut, path+"/policy", map[string]any{
		"statements": []map[string]any{{"actions": []string{"Deploy"}, "principals": []string{"123456789012"}}},
	})
	fmt.Println("Policy statements:", len(c.call("GetApplicationPolicy", http.MethodGet, path+"/policy", nil)["statements"].([]any)))

	tpl := c.call("CreateCloudFormationTemplate", http.MethodPost, path+"/templates", map[string]any{"semanticVersion": "1.1.0"})
	tplID := tpl["templateId"].(string)
	c.call("GetCloudFormationTemplate", http.MethodGet, path+"/templates/"+tplID, nil)
	ready := c.call("GetCloudFormationTemplate", http.MethodGet, path+"/templates/"+tplID, nil)
	fmt.Println("Template:", ready["status"], ready["templateUrl"])

	cs := c.call("CreateCloudFormationChangeSet", http.MethodPost, path+"/changesets", map[string]any{
		"stackName": "hello", "templateId": tplID, "semanticVersion": "1.1.0", "capabilities": []string{"CAPABILITY_AUTO_EXPAND"},
		"parameterOverrides": []map[string]string{{"name": "Stage", "value": "prod"}},
	})
	fmt.Println("Change set:", cs["changeSetId"])
	fmt.Println("Stack:", cs["stackId"])

	c.call("DeleteApplication", http.MethodDelete, path, nil)
	fmt.Println("Deleted:", id)
}

func (c client) call(op, method, path string, payload any) map[string]any {
	var body []byte
	if payload != nil {
		var err error
		if body, err = json.Marshal(payload); err != nil {
			exitf("%s: %v", op, err)
		}
	}
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, method, c.endpoint+path, bytes.NewReader(body))
	if err != nil {
		exitf("%s: %v", op, err)
	}
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	creds, err := c.creds.Retrieve(ctx)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	sum := sha256.Sum256(body)
	if err := v4.NewSigner().SignHTTP(ctx, creds, req, hex.EncodeToString(sum[:]), "serverlessrepo", c.region, time.Now()); err != nil {
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
	out := map[string]any{}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			exitf("%s: decode: %v", op, err)
		}
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
