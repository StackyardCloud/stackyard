// Command servicecatalog walks an AWS Service Catalog lifecycle against
// Stackyard: upload a CloudFormation template to S3, create a portfolio and a
// product from it, add the product to the portfolio with a launch constraint,
// provision it, follow the record to SUCCEEDED, read the outputs (the stack is
// registered in Stackyard's CloudFormation), update it and terminate it.
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
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const target = "AWS242ServiceCatalogService."

const template = `{"Parameters":{"Env":{"Type":"String","AllowedValues":["dev","prod"]}},
"Resources":{"Bucket":{"Type":"AWS::S3::Bucket"}},
"Outputs":{"Environment":{"Description":"The environment","Value":{"Ref":"Env"}}}}`

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
	fmt.Printf("Stackyard Service Catalog client using %s\n", c.endpoint)
	suffix := fmt.Sprint(time.Now().UnixNano() % 1_000_000_000)

	// Provisioning artifacts load their template from S3.
	bucket := "stackyard-sc-" + suffix
	s3c := s3.New(s3.Options{Region: c.region, Credentials: c.creds, BaseEndpoint: aws.String(c.endpoint), UsePathStyle: true})
	ctx := context.Background()
	if _, err := s3c.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(bucket)}); err != nil {
		exitf("CreateBucket: %v", err)
	}
	if _, err := s3c.PutObject(ctx, &s3.PutObjectInput{Bucket: aws.String(bucket), Key: aws.String("app.json"), Body: strings.NewReader(template)}); err != nil {
		exitf("PutObject: %v", err)
	}

	port := c.call("CreatePortfolio", map[string]any{"DisplayName": "Platform", "ProviderName": "IT", "IdempotencyToken": "port-" + suffix})
	portID := dig(port, "PortfolioDetail", "Id")
	prod := c.call("CreateProduct", map[string]any{
		"Name": "Web app", "Owner": "IT", "ProductType": "CLOUD_FORMATION_TEMPLATE", "IdempotencyToken": "prod-" + suffix,
		"ProvisioningArtifactParameters": map[string]any{"Name": "v1", "Info": map[string]any{
			"LoadTemplateFromURL": "https://" + bucket + ".s3.amazonaws.com/app.json"}},
	})
	prodID := dig(prod, "ProductViewDetail", "ProductViewSummary", "ProductId")
	paID := dig(prod, "ProvisioningArtifactDetail", "Id")
	fmt.Printf("portfolio %s, product %s, artifact %s\n", portID, prodID, paID)

	c.call("AssociateProductWithPortfolio", map[string]any{"ProductId": prodID, "PortfolioId": portID})
	c.call("CreateConstraint", map[string]any{"PortfolioId": portID, "ProductId": prodID, "Type": "LAUNCH", "IdempotencyToken": "cons-" + suffix,
		"Parameters": `{"RoleArn":"arn:aws:iam::123456789012:role/sc-launch"}`})
	params := c.call("DescribeProvisioningParameters", map[string]any{"ProductId": prodID, "ProvisioningArtifactId": paID})
	fmt.Printf("parameters: %v\n", params["ProvisioningArtifactParameters"])

	rec := c.call("ProvisionProduct", map[string]any{"ProductId": prodID, "ProvisioningArtifactId": paID, "ProvisionedProductName": "web-" + suffix,
		"ProvisionToken": "prov-" + suffix, "ProvisioningParameters": []map[string]any{{"Key": "Env", "Value": "dev"}}})
	recID, ppID := dig(rec, "RecordDetail", "RecordId"), dig(rec, "RecordDetail", "ProvisionedProductId")
	desc := waitRecord(c, recID)
	fmt.Printf("provisioned %s: outputs %v\n", ppID, desc["RecordOutputs"])

	up := c.call("UpdateProvisionedProduct", map[string]any{"ProvisionedProductId": ppID, "UpdateToken": "upd-" + suffix,
		"ProvisioningParameters": []map[string]any{{"Key": "Env", "Value": "prod"}}})
	waitRecord(c, dig(up, "RecordDetail", "RecordId"))
	outputs := c.call("GetProvisionedProductOutputs", map[string]any{"ProvisionedProductId": ppID})
	fmt.Printf("after update: %v\n", outputs["Outputs"])

	term := c.call("TerminateProvisionedProduct", map[string]any{"ProvisionedProductId": ppID, "TerminateToken": "term-" + suffix})
	waitRecord(c, dig(term, "RecordDetail", "RecordId"))
	fmt.Printf("terminated %s\n", ppID)
}

// waitRecord reads a record until it finishes.
func waitRecord(c client, id string) map[string]any {
	for i := 0; i < 10; i++ {
		out := c.call("DescribeRecord", map[string]any{"Id": id})
		switch status := dig(out, "RecordDetail", "Status"); status {
		case "SUCCEEDED":
			return out
		case "FAILED", "IN_PROGRESS_IN_ERROR":
			exitf("record %s: %s", id, status)
		}
	}
	exitf("record %s did not finish", id)
	return nil
}

func dig(m map[string]any, keys ...string) string {
	var cur any = m
	for _, k := range keys {
		next, _ := cur.(map[string]any)
		cur = next[k]
	}
	s, _ := cur.(string)
	return s
}

func (c client) call(op string, payload any) map[string]any {
	body, err := json.Marshal(payload)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"/", bytes.NewReader(body))
	if err != nil {
		exitf("%s: %v", op, err)
	}
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", target+op)
	creds, err := c.creds.Retrieve(ctx)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	sum := sha256.Sum256(body)
	if err := v4.NewSigner().SignHTTP(ctx, creds, req, hex.EncodeToString(sum[:]), "servicecatalog", c.region, time.Now()); err != nil {
		exitf("%s: %v", op, err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
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
