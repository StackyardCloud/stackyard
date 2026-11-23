// Command servicecatalog-appregistry walks an AWS Service Catalog AppRegistry
// lifecycle against Stackyard: create a CloudFormation stack, an application
// and an attribute group, associate both with the application, read them back,
// tag the application, then tear everything down.
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
	fmt.Printf("Stackyard Service Catalog AppRegistry client using %s\n", c.endpoint)
	suffix := fmt.Sprint(time.Now().UnixNano())

	stack := "stackyard-appregistry-" + suffix
	c.send("CreateStack", "cloudformation", http.MethodPost, c.endpoint+"/",
		[]byte(url.Values{"Action": {"CreateStack"}, "Version": {"2010-05-15"}, "StackName": {stack}, "TemplateBody": {"{}"}}.Encode()),
		"application/x-www-form-urlencoded")
	fmt.Println("Stack:", stack)

	app := c.call("CreateApplication", http.MethodPost, "/applications", map[string]any{
		"name": "shop-" + suffix, "description": "Storefront", "clientToken": "app-" + suffix, "tags": map[string]string{"env": "dev"},
	})["application"].(map[string]any)
	appID, appARN := app["id"].(string), app["arn"].(string)
	fmt.Println("Application:", appID)

	group := c.call("CreateAttributeGroup", http.MethodPost, "/attribute-groups", map[string]any{
		"name": "meta-" + suffix, "attributes": `{"owner":"team-shop","tier":"gold"}`, "clientToken": "group-" + suffix,
	})["attributeGroup"].(map[string]any)
	groupID := group["id"].(string)
	fmt.Println("Attribute group:", groupID)

	c.call("AssociateAttributeGroup", http.MethodPut, "/applications/"+appID+"/attribute-groups/"+groupID, nil)
	c.call("AssociateResource", http.MethodPut, "/applications/"+appID+"/resources/CFN_STACK/"+stack, map[string]any{"options": []string{"APPLY_APPLICATION_TAG"}})

	got := c.call("GetApplication", http.MethodGet, "/applications/"+appID, nil)
	fmt.Println("Associated resources:", got["associatedResourceCount"])
	fmt.Println("Resource group:", got["integrations"].(map[string]any)["resourceGroup"].(map[string]any)["arn"])
	res := c.call("GetAssociatedResource", http.MethodGet, "/applications/"+appID+"/resources/CFN_STACK/"+stack, nil)
	fmt.Println("Stack ARN:", res["resource"].(map[string]any)["arn"])
	fmt.Println("Sync:", c.call("SyncResource", http.MethodPost, "/sync/CFN_STACK/"+stack, nil)["actionTaken"])

	tagPath := "/tags/" + url.PathEscape(appARN)
	c.call("TagResource", http.MethodPost, tagPath, map[string]any{"tags": map[string]string{"team": "shop"}})
	fmt.Println("Tags:", c.call("ListTagsForResource", http.MethodGet, tagPath, nil)["tags"])

	c.call("DisassociateResource", http.MethodDelete, "/applications/"+appID+"/resources/CFN_STACK/"+stack, nil)
	c.call("DisassociateAttributeGroup", http.MethodDelete, "/applications/"+appID+"/attribute-groups/"+groupID, nil)
	c.call("DeleteAttributeGroup", http.MethodDelete, "/attribute-groups/"+groupID, nil)
	deleted := c.call("DeleteApplication", http.MethodDelete, "/applications/"+appID, nil)["application"].(map[string]any)
	fmt.Println("Deleted:", deleted["name"])
}

func (c client) call(op, method, path string, payload any) map[string]any {
	var body []byte
	if payload != nil {
		var err error
		if body, err = json.Marshal(payload); err != nil {
			exitf("%s: %v", op, err)
		}
	}
	raw := c.send(op, "servicecatalog", method, c.endpoint+path, body, "application/json")
	out := map[string]any{}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			exitf("%s: decode: %v", op, err)
		}
	}
	return out
}

// send signs a request for service and exits unless it returns 2xx.
func (c client) send(op, service, method, target string, body []byte, contentType string) []byte {
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, method, target, bytes.NewReader(body))
	if err != nil {
		exitf("%s: %v", op, err)
	}
	if len(body) > 0 {
		req.Header.Set("Content-Type", contentType)
	}
	creds, err := c.creds.Retrieve(ctx)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	sum := sha256.Sum256(body)
	if err := v4.NewSigner().SignHTTP(ctx, creds, req, hex.EncodeToString(sum[:]), service, c.region, time.Now()); err != nil {
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
