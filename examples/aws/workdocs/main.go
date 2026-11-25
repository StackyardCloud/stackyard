// Command workdocs runs Amazon WorkDocs against Stackyard: create a user and a
// folder, upload a document through the pre-signed URL, add a second
// version, comment on it, label and share it, search for it, read the
// activity feed, download it, and clean up.
//
// The SDK has no WorkDocs module in this repository's go.mod, so requests are
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
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

const signingName = "workdocs"

func main() {
	c := newClient()
	fmt.Printf("Stackyard WorkDocs client using %s\n", c.endpoint)
	run := time.Now().UnixNano()

	admin := c.rest(http.MethodGet, "/api/v1/users?query=stackyard-admin", nil)["Users"].([]any)[0].(map[string]any)
	root := admin["RootFolderId"].(string)
	me := c.restAs(admin["Id"].(string), http.MethodGet, "/api/v1/me", nil)["User"].(map[string]any)
	fmt.Printf("acting as %v (%v)\n", me["Username"], me["Type"])
	user := c.rest(http.MethodPost, "/api/v1/users", map[string]any{
		"Username": fmt.Sprintf("analyst%d", run), "GivenName": "Ana", "Surname": "Lyst", "Password": "Correct-Horse-9",
	})["User"].(map[string]any)

	folder := c.rest(http.MethodPost, "/api/v1/folders", map[string]any{"Name": fmt.Sprintf("Reports %d", run), "ParentFolderId": root})["Metadata"].(map[string]any)["Id"].(string)
	doc, v1 := c.upload(map[string]any{"Name": "q3-summary.txt", "ParentFolderId": folder}, "Q3 revenue: 1.2M\n")
	_, v2 := c.upload(map[string]any{"Id": doc}, "Q3 revenue: 1.25M (restated)\n")
	versions := c.rest(http.MethodGet, "/api/v1/documents/"+doc+"/versions", nil)["DocumentVersions"].([]any)
	fmt.Printf("document %s has %d versions (latest %s, first %s)\n", doc, len(versions), v2, v1)

	comment := c.rest(http.MethodPost, "/api/v1/documents/"+doc+"/versions/"+v2+"/comment", map[string]any{"Text": "Restated after audit"})["Comment"].(map[string]any)
	c.rest(http.MethodPost, "/api/v1/documents/"+doc+"/versions/"+v2+"/comment", map[string]any{"Text": "Thanks", "ParentId": comment["CommentId"]})
	c.rest(http.MethodPut, "/api/v1/resources/"+doc+"/labels", map[string]any{"Labels": []string{"finance", "q3"}})
	c.rest(http.MethodPut, "/api/v1/resources/"+doc+"/customMetadata", map[string]any{"CustomMetadata": map[string]string{"owner": "finance"}})
	share := c.rest(http.MethodPost, "/api/v1/resources/"+doc+"/permissions", map[string]any{
		"Principals": []any{map[string]any{"Id": user["Id"], "Type": "USER", "Role": "VIEWER"}},
	})["ShareResults"].([]any)
	fmt.Printf("shared with %v: %v\n", user["Username"], share[0].(map[string]any)["Status"])

	items := c.rest(http.MethodPost, "/api/v1/search", map[string]any{"QueryText": "q3", "Filters": map[string]any{"Labels": []string{"finance"}}})["Items"].([]any)
	fmt.Printf("search found %d item(s)\n", len(items))

	src := c.rest(http.MethodGet, "/api/v1/documents/"+doc+"/versions/"+v2+"?fields=SOURCE", nil)["Metadata"].(map[string]any)["Source"].(map[string]any)["ORIGINAL"].(string)
	resp, err := http.Get(src)
	if err != nil {
		exitf("download: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	fmt.Printf("downloaded %q\n", strings.TrimSpace(string(body)))

	acts := c.rest(http.MethodGet, "/api/v1/activities?resourceId="+url.QueryEscape(doc), nil)["UserActivities"].([]any)
	for _, a := range acts {
		fmt.Printf("activity: %v\n", a.(map[string]any)["Type"])
	}

	c.rest(http.MethodDelete, "/api/v1/folders/"+folder, nil)
	c.rest(http.MethodDelete, "/api/v1/users/"+user["Id"].(string), nil)
	fmt.Println("cleaned up")
}

// upload starts a document version, PUTs the content and activates it.
func (c client) upload(args map[string]any, content string) (string, string) {
	args["ContentType"] = "text/plain"
	args["DocumentSizeInBytes"] = len(content)
	out := c.rest(http.MethodPost, "/api/v1/documents", args)
	doc := out["Metadata"].(map[string]any)["Id"].(string)
	up := out["UploadMetadata"].(map[string]any)
	req, err := http.NewRequest(http.MethodPut, up["UploadUrl"].(string), strings.NewReader(content))
	if err != nil {
		exitf("upload: %v", err)
	}
	for k, v := range up["SignedHeaders"].(map[string]any) {
		req.Header.Set(k, fmt.Sprint(v))
	}
	do("upload", req)
	pending := ""
	for _, v := range c.rest(http.MethodGet, "/api/v1/documents/"+doc+"/versions?include=INITIALIZED", nil)["DocumentVersions"].([]any) {
		if vm := v.(map[string]any); vm["Status"] == "INITIALIZED" {
			pending = vm["Id"].(string)
		}
	}
	c.rest(http.MethodPatch, "/api/v1/documents/"+doc+"/versions/"+pending, map[string]any{"VersionStatus": "ACTIVE"})
	return doc, pending
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

func (c client) rest(method, path string, payload any) map[string]any {
	return c.restAs("", method, path, payload)
}

// restAs sends one signed REST JSON request (as token's user when set) and
// exits unless it returns 2xx.
func (c client) restAs(token, method, path string, payload any) map[string]any {
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
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authentication", token)
	}
	creds, err := c.creds.Retrieve(ctx)
	if err != nil {
		exitf("%s %s: %v", method, path, err)
	}
	sum := sha256.Sum256(body)
	hash := hex.EncodeToString(sum[:])
	req.Header.Set("X-Amz-Content-Sha256", hash)
	if err := v4.NewSigner().SignHTTP(ctx, creds, req, hash, signingName, c.region, time.Now()); err != nil {
		exitf("%s %s: %v", method, path, err)
	}
	return do(method+" "+path, req)
}

func do(op string, req *http.Request) map[string]any {
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
	if len(bytes.TrimSpace(raw)) > 0 && json.Valid(raw) {
		_ = json.Unmarshal(raw, &out)
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
