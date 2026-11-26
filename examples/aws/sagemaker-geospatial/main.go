// Command sagemaker-geospatial runs Amazon SageMaker geospatial capabilities
// against Stackyard: search Sentinel-2 scenes over an area of interest, run
// an NDVI earth observation job, fetch a tile of its output, export it to S3,
// and reverse-geocode a CSV of points with a vector enrichment job.
//
// Jobs go INITIALIZING -> IN_PROGRESS -> COMPLETED as they are read. The SDK
// has no SageMaker geospatial module in this repository's go.mod, so requests
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
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

const (
	role     = "arn:aws:iam::123456789012:role/geospatial"
	sentinel = "arn:aws:sagemaker-geospatial:us-west-2:378778860802:raster-data-collection/public/nmqj48dcu3g7ayw8"
)

func main() {
	c := newClient()
	fmt.Printf("Stackyard SageMaker geospatial client using %s\n", c.endpoint)
	run := fmt.Sprint(time.Now().UnixNano() % 1_000_000_000)
	bucket := "geospatial-example-" + run
	c.send(http.MethodPut, "/"+bucket, "", "", "s3", nil)
	c.send(http.MethodPut, "/"+bucket+"/points.csv", "text/csv", "", "s3", []byte("id,lon,lat\n1,-122.335,47.608\n2,-122.349,47.620\n"))

	aoi := map[string]any{"AreaOfInterestGeometry": map[string]any{"PolygonGeometry": map[string]any{"Coordinates": [][][]float64{
		{{-122.40, 47.55}, {-122.25, 47.55}, {-122.25, 47.68}, {-122.40, 47.68}, {-122.40, 47.55}}}}}}
	window := map[string]any{"StartTime": 1688169600, "EndTime": 1690848000} // July 2023
	found := c.call(http.MethodPost, "/search-raster-data-collection", map[string]any{"Arn": sentinel,
		"RasterDataCollectionQuery": map[string]any{"AreaOfInterest": aoi, "TimeRangeFilter": window,
			"PropertyFilters": map[string]any{"Properties": []any{map[string]any{"Property": map[string]any{
				"EoCloudCover": map[string]any{"LowerBound": 0, "UpperBound": 50}}}}}}})
	fmt.Printf("%v Sentinel-2 scenes under 50%% cloud cover\n", found["ApproximateResultCount"])

	job := c.call(http.MethodPost, "/earth-observation-jobs", map[string]any{"Name": "ndvi-" + run, "ExecutionRoleArn": role,
		"InputConfig": map[string]any{"RasterDataCollectionQuery": map[string]any{"RasterDataCollectionArn": sentinel,
			"AreaOfInterest": aoi, "TimeRangeFilter": window}},
		"JobConfig": map[string]any{"BandMathConfig": map[string]any{"PredefinedIndices": []string{"ndvi"}}}})["Arn"].(string)
	done := c.wait("/earth-observation-jobs/"+url.PathEscape(job), "Status", "COMPLETED")
	fmt.Printf("earth observation job %v completed with bands %v\n", done["Name"], done["OutputBands"])

	tile := c.send(http.MethodGet, "/tile/0/0/0?Arn="+url.QueryEscape(job)+"&ImageAssets=ndvi&Target=OUTPUT", "", "", "sagemaker-geospatial", nil)
	fmt.Printf("fetched a %d-byte PNG tile\n", len(tile))
	c.call(http.MethodPost, "/export-earth-observation-job", map[string]any{"Arn": job, "ExecutionRoleArn": role,
		"OutputConfig": map[string]any{"S3Data": map[string]any{"S3Uri": "s3://" + bucket + "/ndvi/"}}})
	c.wait("/earth-observation-jobs/"+url.PathEscape(job), "ExportStatus", "SUCCEEDED")
	fmt.Printf("exported to s3://%s/ndvi/\n", bucket)

	vec := c.call(http.MethodPost, "/vector-enrichment-jobs", map[string]any{"Name": "geocode-" + run, "ExecutionRoleArn": role,
		"InputConfig": map[string]any{"DocumentType": "CSV", "DataSourceConfig": map[string]any{"S3Data": map[string]any{"S3Uri": "s3://" + bucket + "/points.csv"}}},
		"JobConfig":   map[string]any{"ReverseGeocodingConfig": map[string]any{"XAttributeName": "lon", "YAttributeName": "lat"}}})["Arn"].(string)
	c.wait("/vector-enrichment-jobs/"+url.PathEscape(vec), "Status", "COMPLETED")
	c.call(http.MethodPost, "/export-vector-enrichment-jobs", map[string]any{"Arn": vec, "ExecutionRoleArn": role,
		"OutputConfig": map[string]any{"S3Data": map[string]any{"S3Uri": "s3://" + bucket + "/geocoded/"}}})
	c.wait("/vector-enrichment-jobs/"+url.PathEscape(vec), "ExportStatus", "SUCCEEDED")
	id := vec[strings.LastIndex(vec, "/")+1:]
	fmt.Printf("geocoded points:\n%s", c.send(http.MethodGet, "/"+bucket+"/geocoded/"+id+"/output.csv", "", "", "s3", nil))

	c.send(http.MethodDelete, "/earth-observation-jobs/"+url.PathEscape(job), "", "", "sagemaker-geospatial", nil)
	c.send(http.MethodDelete, "/vector-enrichment-jobs/"+url.PathEscape(vec), "", "", "sagemaker-geospatial", nil)
	fmt.Println("deleted both jobs")
}

func (c client) call(method, path string, payload any) map[string]any {
	return decode(c.send(method, path, "application/json", "", "sagemaker-geospatial", mustJSON(payload)))
}

// wait reads a job until key reports want.
func (c client) wait(path, key, want string) map[string]any {
	for i := 0; i < 20; i++ {
		out := decode(c.send(http.MethodGet, path, "", "", "sagemaker-geospatial", nil))
		if out[key] == want {
			return out
		}
		if out["Status"] == "FAILED" || out["ExportStatus"] == "FAILED" {
			exitf("%s failed: %v", path, out)
		}
		time.Sleep(200 * time.Millisecond)
	}
	exitf("%s never reached %s=%s", path, key, want)
	return nil
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

// send signs a request for service and exits unless it returns 2xx.
func (c client) send(method, path, contentType, target, service string, body []byte) []byte {
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, method, c.endpoint+path, bytes.NewReader(body))
	if err != nil {
		exitf("%s %s: %v", method, path, err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if target != "" {
		req.Header.Set("X-Amz-Target", target)
	}
	creds, err := c.creds.Retrieve(ctx)
	if err != nil {
		exitf("credentials: %v", err)
	}
	sum := sha256.Sum256(body)
	payloadHash := hex.EncodeToString(sum[:])
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)
	if err := v4.NewSigner().SignHTTP(ctx, creds, req, payloadHash, service, c.region, time.Now()); err != nil {
		exitf("sign: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		exitf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		exitf("%s %s %s: HTTP %d: %s", method, path, target, resp.StatusCode, raw)
	}
	return raw
}

func decode(raw []byte) map[string]any {
	out := map[string]any{}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			exitf("decode %q: %v", raw, err)
		}
	}
	return out
}

func mustJSON(v any) []byte {
	raw, err := json.Marshal(v)
	if err != nil {
		exitf("encode: %v", err)
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
