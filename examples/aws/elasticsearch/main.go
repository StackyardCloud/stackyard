// Command elasticsearch drives the legacy Amazon Elasticsearch Service API
// (es 2015-01-01) against Stackyard and checks the domain is also visible
// through the OpenSearch Service API.
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
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	region := getenv("AWS_REGION", "us-east-1")
	domain := getenv("STACKYARD_DOMAIN_NAME", "es-example")
	arn := "arn:aws:es:" + region + ":123456789012:domain/" + domain
	creds, err := credentials.NewStaticCredentialsProvider(getenv("AWS_ACCESS_KEY_ID", "stackyard"), getenv("AWS_SECRET_ACCESS_KEY", "stackyard"), "").Retrieve(context.Background())
	if err != nil {
		exitf("credentials: %v", err)
	}
	call := func(method, path string, payload any) map[string]any {
		var body []byte
		if payload != nil {
			body, _ = json.Marshal(payload)
		}
		req, err := http.NewRequest(method, endpoint+path, bytes.NewReader(body))
		if err != nil {
			exitf("%s %s: %v", method, path, err)
		}
		if payload != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		sum := sha256.Sum256(body)
		if err := v4.NewSigner().SignHTTP(context.Background(), aws.Credentials(creds), req, hex.EncodeToString(sum[:]), "es", region, time.Now()); err != nil {
			exitf("sign: %v", err)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			exitf("%s %s: %v", method, path, err)
		}
		defer resp.Body.Close()
		data, _ := io.ReadAll(resp.Body)
		if resp.StatusCode/100 != 2 {
			exitf("%s %s: %d %s", method, path, resp.StatusCode, data)
		}
		out := map[string]any{}
		_ = json.Unmarshal(data, &out)
		fmt.Printf("%s %s -> %d\n", method, path, resp.StatusCode)
		return out
	}

	call(http.MethodPost, "/2015-01-01/es/domain", map[string]any{
		"DomainName": domain, "ElasticsearchVersion": "7.9",
		"ElasticsearchClusterConfig": map[string]any{"InstanceType": "r5.large.elasticsearch", "InstanceCount": 2},
		"EBSOptions":                 map[string]any{"EBSEnabled": true, "VolumeSize": 20},
	})
	call(http.MethodGet, "/2015-01-01/es/domain/"+domain, nil)
	call(http.MethodPost, "/2015-01-01/es/domain/"+domain+"/config", map[string]any{"ElasticsearchClusterConfig": map[string]any{"InstanceCount": 4}})
	call(http.MethodPost, "/2015-01-01/tags", map[string]any{"ARN": arn, "TagList": []any{map[string]any{"Key": "env", "Value": "dev"}}})
	call(http.MethodGet, "/2015-01-01/tags/?arn="+url.QueryEscape(arn), nil)
	call(http.MethodPost, "/2015-01-01/es/upgradeDomain", map[string]any{"DomainName": domain, "TargetVersion": "7.10"})
	call(http.MethodGet, "/2015-01-01/es/upgradeDomain/"+domain+"/status", nil)

	// The OpenSearch API sees the same domain.
	status, _ := call(http.MethodGet, "/2021-01-01/opensearch/domain/"+domain, nil)["DomainStatus"].(map[string]any)
	if status["EngineVersion"] != "Elasticsearch_7.10" {
		exitf("OpenSearch view: EngineVersion %v, want Elasticsearch_7.10", status["EngineVersion"])
	}

	call(http.MethodDelete, "/2015-01-01/es/domain/"+domain, nil)
	fmt.Println("elasticsearch example completed")
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
