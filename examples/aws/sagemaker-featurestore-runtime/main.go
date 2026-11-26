// Command sagemaker-featurestore-runtime runs Amazon SageMaker Feature Store Runtime against Stackyard:
// create an online feature group with the SageMaker control plane, put,
// update, read, list and delete records, and batch-write and batch-read them.
//
// The SDK has no SageMaker Feature Store Runtime module in this repository's go.mod, so requests are
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
	fmt.Printf("Stackyard SageMaker Feature Store Runtime client using %s\n", c.endpoint)
	group := fmt.Sprintf("customers-%d", time.Now().UnixNano()%1_000_000_000)
	c.sagemaker("CreateFeatureGroup", map[string]any{
		"FeatureGroupName": group, "RecordIdentifierFeatureName": "customer_id", "EventTimeFeatureName": "event_time",
		"FeatureDefinitions": []any{
			map[string]any{"FeatureName": "customer_id", "FeatureType": "String"},
			map[string]any{"FeatureName": "event_time", "FeatureType": "Fractional"},
			map[string]any{"FeatureName": "lifetime_value", "FeatureType": "Fractional"},
			map[string]any{"FeatureName": "segments", "FeatureType": "String", "CollectionType": "Set"},
		},
		"OnlineStoreConfig": map[string]any{"EnableOnlineStore": true, "StorageType": "InMemory"},
	})
	logf("created feature group %s", group)

	now := float64(time.Now().Unix())
	record := func(id string, at float64, value string) []any {
		return []any{
			map[string]any{"FeatureName": "customer_id", "ValueAsString": id},
			map[string]any{"FeatureName": "event_time", "ValueAsString": fmt.Sprintf("%.0f", at)},
			map[string]any{"FeatureName": "lifetime_value", "ValueAsString": value},
		}
	}
	path := "/FeatureGroup/" + group
	c.restOK(http.MethodPut, path, map[string]any{"Record": append(record("c-1", now, "120.5"),
		map[string]any{"FeatureName": "segments", "ValueAsStringList": []string{"loyal", "web"}})})
	// An older event never replaces the online record.
	c.restOK(http.MethodPut, path, map[string]any{"Record": record("c-1", now-3600, "1")})
	got := c.restOK(http.MethodGet, path+"?RecordIdentifierValueAsString=c-1", nil)
	logf("GetRecord c-1: %v", got["Record"])
	if !strings.Contains(fmt.Sprint(got["Record"]), "120.5") {
		exitf("the older event replaced the record: %v", got)
	}
	if status, out := c.rest(http.MethodPut, path, map[string]any{"Record": append(record("c-2", now, "5"),
		map[string]any{"FeatureName": "favorite_color", "ValueAsString": "blue"})}); status != http.StatusBadRequest {
		exitf("an undefined feature was accepted: %d %v", status, out)
	} else {
		logf("undefined feature rejected: %v", out["message"])
	}

	c.restOK(http.MethodPost, path+"/Record", map[string]any{"RecordIdentifierValueAsString": "c-1",
		"Features": []any{map[string]any{"FeatureName": "lifetime_value", "ValueAsString": "150"}}})
	got = c.restOK(http.MethodGet, path+"?RecordIdentifierValueAsString=c-1&FeatureName=lifetime_value", nil)
	logf("after UpdateRecord: %v", got["Record"])

	batch := c.restOK(http.MethodPost, "/BatchWriteRecord", map[string]any{"Entries": []any{
		map[string]any{"FeatureGroupName": group, "Record": record("c-2", now, "42")},
		map[string]any{"FeatureGroupName": group, "Record": record("c-3", now, "7")},
	}})
	logf("BatchWriteRecord errors: %v", batch["Errors"])
	read := c.restOK(http.MethodPost, "/BatchGetRecord", map[string]any{"Identifiers": []any{map[string]any{
		"FeatureGroupName": group, "RecordIdentifiersValueAsString": []string{"c-1", "c-2", "c-9"}}}})
	logf("BatchGetRecord: %d records, %d errors", len(read["Records"].([]any)), len(read["Errors"].([]any)))

	c.restOK(http.MethodDelete, fmt.Sprintf("%s?RecordIdentifierValueAsString=c-3&EventTime=%.0f", path, now+1), nil)
	list := c.restOK(http.MethodPost, path+"/ListRecords", map[string]any{})
	logf("ListRecords after deleting c-3: %v", list["RecordIdentifiers"])
	if fmt.Sprint(list["RecordIdentifiers"]) != "[c-1 c-2]" {
		exitf("unexpected record identifiers %v", list["RecordIdentifiers"])
	}

	c.sagemaker("DeleteFeatureGroup", map[string]any{"FeatureGroupName": group})
	fmt.Println("SageMaker Feature Store Runtime example completed")
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
