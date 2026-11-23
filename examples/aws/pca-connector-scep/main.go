// Command pca-connector-scep walks a Private CA Connector for SCEP lifecycle
// against Stackyard: create a Private CA, a SCEP connector on it and a
// challenge password, read them back, tag the connector, then tear down.
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
	fmt.Printf("Stackyard Private CA Connector for SCEP client using %s\n", c.endpoint)

	caBody, _ := json.Marshal(map[string]any{
		"CertificateAuthorityType": "ROOT",
		"CertificateAuthorityConfiguration": map[string]any{
			"KeyAlgorithm": "RSA_2048", "SigningAlgorithm": "SHA256WITHRSA",
			"Subject": map[string]any{"CommonName": "stackyard-scep-example"},
		},
	})
	var ca struct{ CertificateAuthorityArn string }
	if err := json.Unmarshal(c.send("CreateCertificateAuthority", "acm-pca", http.MethodPost, c.endpoint+"/", caBody,
		"application/x-amz-json-1.1", "X-Amz-Target", "ACMPrivateCA.CreateCertificateAuthority"), &ca); err != nil {
		exitf("CreateCertificateAuthority: %v", err)
	}
	fmt.Println("CA:", ca.CertificateAuthorityArn)

	token := fmt.Sprintf("stackyard-%d", time.Now().UnixNano())
	conn := c.call("CreateConnector", http.MethodPost, "/connectors",
		map[string]any{"CertificateAuthorityArn": ca.CertificateAuthorityArn, "ClientToken": token, "Tags": map[string]string{"env": "dev"}})["ConnectorArn"].(string)
	path := "/connectors/" + url.PathEscape(conn)
	fmt.Println("Connector:", conn)
	got := c.call("GetConnector", http.MethodGet, path, nil)["Connector"].(map[string]any)
	fmt.Println("Status:", got["Status"], "Type:", got["Type"], "Endpoint:", got["Endpoint"])

	ch := c.call("CreateChallenge", http.MethodPost, "/challenges", map[string]any{"ConnectorArn": conn})["Challenge"].(map[string]any)
	chArn := ch["Arn"].(string)
	fmt.Println("Challenge:", chArn)
	pw := c.call("GetChallengePassword", http.MethodGet, "/challengePasswords/"+url.PathEscape(chArn), nil)["Password"]
	fmt.Println("Password matches:", pw == ch["Password"])
	list := c.call("ListChallengeMetadata", http.MethodGet, "/challengeMetadata?ConnectorArn="+url.QueryEscape(conn), nil)["Challenges"].([]any)
	fmt.Println("Challenges on connector:", len(list))

	c.call("TagResource", http.MethodPost, "/tags/"+url.PathEscape(conn), map[string]any{"Tags": map[string]string{"team": "pki"}})
	fmt.Println("Tags:", c.call("ListTagsForResource", http.MethodGet, "/tags/"+url.PathEscape(conn), nil)["Tags"])

	c.call("DeleteChallenge", http.MethodDelete, "/challenges/"+url.PathEscape(chArn), nil)
	c.call("DeleteConnector", http.MethodDelete, path, nil)
	fmt.Println("Deleted connector and challenge")
}

func (c client) call(op, method, path string, payload any) map[string]any {
	var body []byte
	if payload != nil {
		var err error
		if body, err = json.Marshal(payload); err != nil {
			exitf("%s: %v", op, err)
		}
	}
	raw := c.send(op, "pca-connector-scep", method, c.endpoint+path, body, "application/json")
	out := map[string]any{}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			exitf("%s: decode: %v", op, err)
		}
	}
	return out
}

// send signs a request for service and exits unless it returns 2xx. Extra
// headers come in name/value pairs.
func (c client) send(op, service, method, target string, body []byte, contentType string, headers ...string) []byte {
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
