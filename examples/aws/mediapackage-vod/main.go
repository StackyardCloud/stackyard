// Command mediapackage-vod packages a VOD asset with AWS Elemental MediaPackage
// VOD against Stackyard: upload an HLS source to S3, create a packaging group
// with an HLS packaging configuration, ingest the asset, read its egress
// endpoints, tag it, then tear everything down.
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

func main() {
	c := newClient()
	fmt.Printf("Stackyard MediaPackage VOD client using %s\n", c.endpoint)
	suffix := fmt.Sprint(time.Now().UnixNano())
	bucket, group, config, asset := "stackyard-vod-"+suffix, "group-"+suffix, "hls-"+suffix, "movie-"+suffix

	c.send("CreateBucket", "s3", http.MethodPut, c.endpoint+"/"+bucket, nil)
	c.send("PutObject", "s3", http.MethodPut, c.endpoint+"/"+bucket+"/movie/index.m3u8", []byte("#EXTM3U\n#EXT-X-ENDLIST\n"))

	g := c.vod("CreatePackagingGroup", http.MethodPost, "/packaging_groups", map[string]any{"id": group, "egressAccessLogs": map[string]any{}})
	fmt.Println("Packaging group:", g["arn"], g["domainName"])
	c.vod("CreatePackagingConfiguration", http.MethodPost, "/packaging_configurations", map[string]any{
		"id": config, "packagingGroupId": group,
		"hlsPackage": map[string]any{"hlsManifests": []map[string]any{{"manifestName": "index", "adMarkers": "NONE"}}, "segmentDurationSeconds": 6},
	})
	a := c.vod("CreateAsset", http.MethodPost, "/assets", map[string]any{
		"id": asset, "packagingGroupId": group, "sourceArn": "arn:aws:s3:::" + bucket + "/movie/index.m3u8",
		"sourceRoleArn": "arn:aws:iam::123456789012:role/mediapackage-vod", "tags": map[string]string{"env": "dev"},
	})
	fmt.Println("Asset:", a["arn"])
	for _, ep := range c.vod("DescribeAsset", http.MethodGet, "/assets/"+asset, nil)["egressEndpoints"].([]any) {
		e := ep.(map[string]any)
		fmt.Println("Egress:", e["packagingConfigurationId"], e["status"], e["url"])
	}
	fmt.Println("Assets in group:", c.vod("DescribePackagingGroup", http.MethodGet, "/packaging_groups/"+group, nil)["approximateAssetCount"])

	tags := "/tags/" + url.PathEscape(a["arn"].(string))
	c.vod("TagResource", http.MethodPost, tags, map[string]any{"tags": map[string]string{"team": "video"}})
	fmt.Println("Tags:", c.vod("ListTagsForResource", http.MethodGet, tags, nil)["tags"])

	c.vod("DeleteAsset", http.MethodDelete, "/assets/"+asset, nil)
	c.vod("DeletePackagingConfiguration", http.MethodDelete, "/packaging_configurations/"+config, nil)
	c.vod("DeletePackagingGroup", http.MethodDelete, "/packaging_groups/"+group, nil)
	fmt.Println("Deleted asset, configuration and group")
}

// vod calls a MediaPackage VOD (REST JSON) operation.
func (c client) vod(op, method, path string, payload any) map[string]any {
	var body []byte
	if payload != nil {
		body, _ = json.Marshal(payload)
	}
	_, raw := c.send(op, "mediapackage-vod", method, c.endpoint+path, body, "Content-Type", "application/json")
	return decode(op, raw)
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

// send signs a request for service and exits unless it returns 2xx. Extra
// headers come in name/value pairs.
func (c client) send(op, service, method, target string, body []byte, headers ...string) (http.Header, []byte) {
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, method, target, bytes.NewReader(body))
	if err != nil {
		exitf("%s: %v", op, err)
	}
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
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
	return resp.Header, raw
}

func decode(op string, raw []byte) map[string]any {
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
