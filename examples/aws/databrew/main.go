// Command databrew walks an AWS Glue DataBrew flow against Stackyard: upload
// a CSV to S3, register it as a dataset, write and publish a recipe, preview a
// step in a project session, then run a recipe job and a profile job and read
// their output back from S3.
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
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const role = "arn:aws:iam::123456789012:role/databrew"

type client struct {
	endpoint, region string
	creds            aws.CredentialsProvider
}

func main() {
	ctx := context.Background()
	c := client{
		endpoint: strings.TrimRight(getenv("STACKYARD_ENDPOINT", "http://localhost:4566"), "/"),
		region:   getenv("AWS_REGION", "us-east-1"),
		creds: credentials.NewStaticCredentialsProvider(
			getenv("AWS_ACCESS_KEY_ID", "stackyard"), getenv("AWS_SECRET_ACCESS_KEY", "stackyard"), ""),
	}
	fmt.Printf("Stackyard Glue DataBrew client using %s\n", c.endpoint)

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(c.region), config.WithCredentialsProvider(c.creds))
	if err != nil {
		exitf("load aws config: %v", err)
	}
	s3c := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(c.endpoint)
		o.UsePathStyle = true
	})
	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%1_000_000_000)
	bucket := "databrew-example-" + suffix
	if _, err := s3c.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(bucket)}); err != nil {
		exitf("CreateBucket: %v", err)
	}
	csv := "name,city,score\nalice,paris,3\nbob,,5\ncarol,rome,4\n"
	if _, err := s3c.PutObject(ctx, &s3.PutObjectInput{Bucket: aws.String(bucket), Key: aws.String("in/people.csv"), Body: strings.NewReader(csv)}); err != nil {
		exitf("PutObject: %v", err)
	}

	dataset, recipe, project := "people-"+suffix, "clean-"+suffix, "explore-"+suffix
	c.call("CreateDataset", http.MethodPost, "/datasets", map[string]any{
		"Name": dataset, "Format": "CSV", "Input": map[string]any{"S3InputDefinition": map[string]any{"Bucket": bucket, "Key": "in/people.csv"}},
	})
	c.call("CreateRecipe", http.MethodPost, "/recipes", map[string]any{"Name": recipe, "Steps": []any{
		map[string]any{"Action": map[string]any{"Operation": "UPPER_CASE", "Parameters": map[string]string{"sourceColumn": "name"}}},
		map[string]any{"Action": map[string]any{"Operation": "FILL_WITH_CUSTOM", "Parameters": map[string]string{"sourceColumn": "city", "value": "unknown"}}},
	}})
	c.call("PublishRecipe", http.MethodPost, "/recipes/"+recipe+"/publishRecipe", map[string]any{"Description": "first"})
	fmt.Println("recipe version:", c.call("DescribeRecipe", http.MethodGet, "/recipes/"+recipe, nil)["RecipeVersion"])

	// A project session previews a step on the dataset sample.
	scratch := "scratch-" + suffix
	c.call("CreateRecipe", http.MethodPost, "/recipes", map[string]any{"Name": scratch, "Steps": []any{
		map[string]any{"Action": map[string]any{"Operation": "LOWER_CASE", "Parameters": map[string]string{"sourceColumn": "city"}}},
	}})
	c.call("CreateProject", http.MethodPost, "/projects", map[string]any{"Name": project, "DatasetName": dataset, "RecipeName": scratch, "RoleArn": role})
	session := c.call("StartProjectSession", http.MethodPut, "/projects/"+project+"/startProjectSession", map[string]any{})
	preview := c.call("SendProjectSessionAction", http.MethodPut, "/projects/"+project+"/sendProjectSessionAction", map[string]any{
		"Preview": true, "ClientSessionId": session["ClientSessionId"],
		"RecipeStep": map[string]any{"Action": map[string]any{"Operation": "RENAME", "Parameters": map[string]string{"sourceColumn": "score", "targetColumn": "points"}}},
	})
	fmt.Println("preview:", preview["Result"])

	// A recipe job writes the cleaned CSV back to S3.
	job := "clean-job-" + suffix
	c.call("CreateRecipeJob", http.MethodPost, "/recipeJobs", map[string]any{
		"Name": job, "DatasetName": dataset, "RoleArn": role, "RecipeReference": map[string]any{"Name": recipe},
		"Outputs": []any{map[string]any{"Location": map[string]any{"Bucket": bucket, "Key": "out/"}}},
	})
	run := c.call("StartJobRun", http.MethodPost, "/jobs/"+job+"/startJobRun", nil)["RunId"].(string)
	fmt.Println("recipe job:", waitRun(c, job, run))
	fmt.Print(readFirst(ctx, s3c, bucket, "out/"))

	// A profile job writes per-column statistics.
	profile := "profile-job-" + suffix
	c.call("CreateProfileJob", http.MethodPost, "/profileJobs", map[string]any{
		"Name": profile, "DatasetName": dataset, "RoleArn": role, "OutputLocation": map[string]any{"Bucket": bucket, "Key": "profile/"},
	})
	prun := c.call("StartJobRun", http.MethodPost, "/jobs/"+profile+"/startJobRun", nil)["RunId"].(string)
	fmt.Println("profile job:", waitRun(c, profile, prun))
	var stats struct {
		Columns []map[string]any `json:"columns"`
	}
	_ = json.Unmarshal([]byte(readFirst(ctx, s3c, bucket, "profile/")), &stats)
	for _, col := range stats.Columns {
		fmt.Printf("  %v: %v distinct, %v missing\n", col["name"], col["distinctValuesCount"], col["missingValuesCount"])
	}
	fmt.Println("Glue DataBrew example completed")
}

// waitRun reads a job run until it finishes (reads settle the run state).
func waitRun(c client, job, run string) string {
	for i := 0; i < 5; i++ {
		state := c.call("DescribeJobRun", http.MethodGet, "/jobs/"+job+"/jobRun/"+run, nil)["State"].(string)
		if state == "SUCCEEDED" || state == "FAILED" || state == "STOPPED" {
			if state != "SUCCEEDED" {
				exitf("job %s run %s ended %s", job, run, state)
			}
			return state
		}
	}
	exitf("job %s run %s never finished", job, run)
	return ""
}

func readFirst(ctx context.Context, c *s3.Client, bucket, prefix string) string {
	list, err := c.ListObjectsV2(ctx, &s3.ListObjectsV2Input{Bucket: aws.String(bucket), Prefix: aws.String(prefix)})
	if err != nil || len(list.Contents) == 0 {
		exitf("no output under s3://%s/%s: %v", bucket, prefix, err)
	}
	obj, err := c.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(bucket), Key: list.Contents[0].Key})
	if err != nil {
		exitf("GetObject: %v", err)
	}
	defer obj.Body.Close()
	raw, _ := io.ReadAll(obj.Body)
	return string(raw)
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
	if err := v4.NewSigner().SignHTTP(ctx, creds, req, hex.EncodeToString(sum[:]), "databrew", c.region, time.Now()); err != nil {
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
