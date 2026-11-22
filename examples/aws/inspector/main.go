// Command inspector runs an Amazon Inspector Classic assessment against
// Stackyard: launch a tagged EC2 instance, target it through a resource
// group, run a CVE assessment until it completes, read its findings and
// report, then scan a CycloneDX SBOM with Inspector Scan.
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
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

const cvePackage = "arn:aws:inspector:us-east-1:316112463485:rulespackage/0-gEjTy7T7"

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
	fmt.Printf("Stackyard Inspector client using %s\n", c.endpoint)
	suffix := fmt.Sprint(time.Now().UnixNano())

	instance := c.runInstance("inspector-example-" + suffix)
	fmt.Printf("launched instance %s\n", instance)

	group := c.inspector("CreateResourceGroup", map[string]any{
		"resourceGroupTags": []map[string]string{{"key": "app", "value": "inspector-example-" + suffix}},
	})["resourceGroupArn"]
	target := c.inspector("CreateAssessmentTarget", map[string]any{"assessmentTargetName": "example-" + suffix, "resourceGroupArn": group})["assessmentTargetArn"]
	template := c.inspector("CreateAssessmentTemplate", map[string]any{
		"assessmentTargetArn": target, "assessmentTemplateName": "cve-" + suffix,
		"durationInSeconds": 3600, "rulesPackageArns": []string{cvePackage},
	})["assessmentTemplateArn"]
	run := c.inspector("StartAssessmentRun", map[string]any{"assessmentTemplateArn": template})["assessmentRunArn"]
	fmt.Printf("started run %v\n", run)

	// Runs settle one phase per DescribeAssessmentRuns call.
	for state := ""; state != "COMPLETED"; {
		runs := c.inspector("DescribeAssessmentRuns", map[string]any{"assessmentRunArns": []any{run}})["assessmentRuns"].([]any)
		state = runs[0].(map[string]any)["state"].(string)
		fmt.Printf("run is %s\n", state)
	}

	findings := c.inspector("ListFindings", map[string]any{"assessmentRunArns": []any{run}})["findingArns"].([]any)
	if len(findings) == 0 {
		exitf("expected findings for instance %s", instance)
	}
	described := c.inspector("DescribeFindings", map[string]any{"findingArns": findings})["findings"].([]any)
	for _, f := range described {
		m := f.(map[string]any)
		fmt.Printf("finding [%s] %s\n", m["severity"], m["title"])
	}
	report := c.inspector("GetAssessmentReport", map[string]any{"assessmentRunArn": run, "reportFileFormat": "HTML", "reportType": "FINDING"})
	fmt.Printf("report %v at %v\n", report["status"], report["url"])

	sbom := map[string]any{"bomFormat": "CycloneDX", "specVersion": "1.5", "components": []map[string]string{
		{"bom-ref": "log4j", "type": "library", "name": "log4j-core", "version": "2.14.1", "purl": "pkg:maven/org.apache.logging.log4j/log4j-core@2.14.1"},
	}}
	body, _ := json.Marshal(map[string]any{"sbom": sbom, "outputFormat": "CYCLONE_DX_1_5"})
	var scanned map[string]any
	if err := json.Unmarshal(c.send("ScanSbom", "inspector-scan", c.endpoint+"/scan/sbom", body, "application/json", nil), &scanned); err != nil {
		exitf("ScanSbom: decode: %v", err)
	}
	vulns := scanned["sbom"].(map[string]any)["vulnerabilities"].([]any)
	fmt.Printf("SBOM scan found %d vulnerabilit(ies)\n", len(vulns))

	c.inspector("DeleteAssessmentTarget", map[string]any{"assessmentTargetArn": target})
	fmt.Println("Inspector example completed")
}

func (c client) inspector(op string, payload any) map[string]any {
	body, err := json.Marshal(payload)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	raw := c.send(op, "inspector", c.endpoint+"/", body, "application/x-amz-json-1.1", map[string]string{"X-Amz-Target": "InspectorService." + op})
	out := map[string]any{}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			exitf("%s: decode: %v", op, err)
		}
	}
	return out
}

var instanceIDPattern = regexp.MustCompile(`<instanceId>(i-[0-9a-f]+)</instanceId>`)

func (c client) runInstance(app string) string {
	params := url.Values{
		"Action": {"RunInstances"}, "Version": {"2016-11-15"}, "ImageId": {"ami-12345678"}, "InstanceType": {"t3.micro"},
		"MinCount": {"1"}, "MaxCount": {"1"}, "TagSpecification.1.ResourceType": {"instance"},
		"TagSpecification.1.Tag.1.Key": {"app"}, "TagSpecification.1.Tag.1.Value": {app},
	}
	raw := c.send("RunInstances", "ec2", c.endpoint+"/", []byte(params.Encode()), "application/x-www-form-urlencoded", nil)
	m := instanceIDPattern.FindSubmatch(raw)
	if m == nil {
		exitf("RunInstances: no instance id in %s", raw)
	}
	return string(m[1])
}

// send signs a POST for service and exits unless it returns 2xx.
func (c client) send(op, service, target string, body []byte, contentType string, headers map[string]string) []byte {
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target, bytes.NewReader(body))
	if err != nil {
		exitf("%s: %v", op, err)
	}
	req.Header.Set("Content-Type", contentType)
	for k, v := range headers {
		req.Header.Set(k, v)
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
