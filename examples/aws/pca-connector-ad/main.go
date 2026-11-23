// Command pca-connector-ad walks a Private CA Connector for Active Directory
// lifecycle against Stackyard: create a Private CA, register the seeded
// directory, create a connector and its service principal name, add a
// certificate template with a group access control entry, then tear it all
// down.
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

const definition = `{"TemplateV2":{
 "CertificateValidity":{"ValidityPeriod":{"Period":1,"PeriodType":"YEARS"},"RenewalPeriod":{"Period":6,"PeriodType":"WEEKS"}},
 "EnrollmentFlags":{"IncludeSymmetricAlgorithms":true},
 "Extensions":{"KeyUsage":{"UsageFlags":{"DigitalSignature":true,"KeyEncipherment":true}},"ApplicationPolicies":{"Policies":[{"PolicyType":"SERVER_AUTHENTICATION"}]}},
 "GeneralFlags":{"AutoEnrollment":true,"MachineType":true},
 "PrivateKeyAttributes":{"KeySpec":"KEY_EXCHANGE","MinimalKeyLength":2048},
 "PrivateKeyFlags":{"ClientVersion":"WINDOWS_SERVER_2016"},
 "SubjectNameFlags":{"SanRequireDns":true}}}`

func main() {
	c := client{
		endpoint: strings.TrimRight(getenv("STACKYARD_ENDPOINT", "http://localhost:4566"), "/"),
		region:   getenv("AWS_REGION", "us-east-1"),
		creds: credentials.NewStaticCredentialsProvider(
			getenv("AWS_ACCESS_KEY_ID", "stackyard"), getenv("AWS_SECRET_ACCESS_KEY", "stackyard"), ""),
	}
	fmt.Printf("Stackyard Private CA Connector for AD client using %s\n", c.endpoint)

	// A connector needs an ACTIVE Private CA.
	caBody, _ := json.Marshal(map[string]any{
		"CertificateAuthorityConfiguration": map[string]any{
			"KeyAlgorithm": "RSA_2048", "SigningAlgorithm": "SHA256WITHRSA",
			"Subject": map[string]any{"CommonName": "stackyard-ad"},
		},
		"CertificateAuthorityType": "ROOT",
	})
	var ca struct{ CertificateAuthorityArn string }
	if err := json.Unmarshal(c.send("CreateCertificateAuthority", "acm-pca", http.MethodPost, c.endpoint+"/", caBody,
		"application/x-amz-json-1.1", "X-Amz-Target", "ACMPrivateCA.CreateCertificateAuthority"), &ca); err != nil {
		exitf("CreateCertificateAuthority: %v", err)
	}
	fmt.Println("Certificate authority:", ca.CertificateAuthorityArn)

	// Register the seeded directory (another run may have registered it already).
	directory := "d-0000000000"
	reg := "arn:aws:pca-connector-ad:" + c.region + ":123456789012:directory-registration/" + directory
	if _, status := c.try("CreateDirectoryRegistration", http.MethodPost, "/directoryRegistrations", map[string]any{"DirectoryId": directory}); status != http.StatusAccepted && status != http.StatusConflict {
		exitf("CreateDirectoryRegistration: HTTP %d", status)
	}
	fmt.Println("Directory registration:", reg)

	conn := c.call("CreateConnector", http.MethodPost, "/connectors", map[string]any{
		"CertificateAuthorityArn": ca.CertificateAuthorityArn, "DirectoryId": directory,
		"VpcInformation": map[string]any{"SecurityGroupIds": []string{"sg-00000000"}},
	})["ConnectorArn"].(string)
	got := c.call("GetConnector", http.MethodGet, "/connectors/"+url.PathEscape(conn), nil)["Connector"].(map[string]any)
	fmt.Println("Connector:", conn, got["Status"])
	got = c.call("GetConnector", http.MethodGet, "/connectors/"+url.PathEscape(conn), nil)["Connector"].(map[string]any)
	fmt.Println("Connector after read:", got["Status"], got["CertificateEnrollmentPolicyServerEndpoint"])

	spn := "/directoryRegistrations/" + url.PathEscape(reg) + "/servicePrincipalNames/" + url.PathEscape(conn)
	c.call("CreateServicePrincipalName", http.MethodPost, spn, map[string]any{})
	fmt.Println("Service principal name:", c.call("GetServicePrincipalName", http.MethodGet, spn, nil)["ServicePrincipalName"].(map[string]any)["Status"])

	var def map[string]any
	_ = json.Unmarshal([]byte(definition), &def)
	tmpl := c.call("CreateTemplate", http.MethodPost, "/templates", map[string]any{
		"ConnectorArn": conn, "Name": fmt.Sprintf("WebServer-%d", time.Now().UnixNano()), "Definition": def,
	})["TemplateArn"].(string)
	tp := "/templates/" + url.PathEscape(tmpl)
	c.call("UpdateTemplate", http.MethodPatch, tp, map[string]any{"ReenrollAllCertificateHolders": true})
	t := c.call("GetTemplate", http.MethodGet, tp, nil)["Template"].(map[string]any)
	fmt.Println("Template:", t["Name"], "schema", t["PolicySchema"], "revision", t["Revision"])

	sid := "S-1-5-21-1004336348-1177238915-682003330-512"
	c.call("CreateTemplateGroupAccessControlEntry", http.MethodPost, tp+"/accessControlEntries", map[string]any{
		"GroupSecurityIdentifier": sid, "GroupDisplayName": "Domain Admins",
		"AccessRights": map[string]any{"Enroll": "ALLOW", "AutoEnroll": "ALLOW"},
	})
	entries := c.call("ListTemplateGroupAccessControlEntries", http.MethodGet, tp+"/accessControlEntries", nil)["AccessControlEntries"].([]any)
	fmt.Println("Access control entries:", len(entries))

	c.call("TagResource", http.MethodPost, "/tags/"+url.PathEscape(conn), map[string]any{"Tags": map[string]string{"env": "dev"}})
	fmt.Println("Tags:", c.call("ListTagsForResource", http.MethodGet, "/tags/"+url.PathEscape(conn), nil)["Tags"])

	c.call("DeleteTemplateGroupAccessControlEntry", http.MethodDelete, tp+"/accessControlEntries/"+sid, nil)
	c.call("DeleteTemplate", http.MethodDelete, tp, nil)
	c.call("DeleteServicePrincipalName", http.MethodDelete, spn, nil)
	c.call("DeleteConnector", http.MethodDelete, "/connectors/"+url.PathEscape(conn), nil)
	fmt.Println("Deleted the template, service principal name and connector")
}

func (c client) call(op, method, path string, payload any) map[string]any {
	out, status := c.try(op, method, path, payload)
	if status < 200 || status > 299 {
		exitf("%s: HTTP %d: %v", op, status, out)
	}
	return out
}

// try sends a pca-connector-ad request and returns the decoded body and status.
func (c client) try(op, method, path string, payload any) (map[string]any, int) {
	var body []byte
	if payload != nil {
		var err error
		if body, err = json.Marshal(payload); err != nil {
			exitf("%s: %v", op, err)
		}
	}
	raw, status := c.do(op, "pca-connector-ad", method, c.endpoint+path, body, "application/json")
	out := map[string]any{}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			exitf("%s: decode: %v", op, err)
		}
	}
	return out, status
}

// send signs a request for service and exits unless it returns 2xx. Extra
// headers come in name/value pairs.
func (c client) send(op, service, method, target string, body []byte, contentType string, headers ...string) []byte {
	raw, status := c.do(op, service, method, target, body, contentType, headers...)
	if status < 200 || status > 299 {
		exitf("%s: HTTP %d: %s", op, status, raw)
	}
	return raw
}

func (c client) do(op, service, method, target string, body []byte, contentType string, headers ...string) ([]byte, int) {
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, method, target, bytes.NewReader(body))
	if err != nil {
		exitf("%s: %v", op, err)
	}
	if len(body) > 0 {
		req.Header.Set("Content-Type", contentType)
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
	return raw, resp.StatusCode
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
