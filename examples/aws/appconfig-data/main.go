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
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

type client struct {
	ctx      context.Context
	endpoint string
	region   string
	creds    aws.CredentialsProvider
}

type reply struct {
	status int
	header http.Header
	body   []byte
}

// main deploys two hosted configuration versions with AppConfig and reads
// them back through AppConfig Data, the way an application polls for flags.
func main() {
	c := client{
		ctx:      context.Background(),
		endpoint: strings.TrimRight(getenv("STACKYARD_ENDPOINT", "http://localhost:4566"), "/"),
		region:   getenv("AWS_REGION", "us-east-1"),
		creds: credentials.NewStaticCredentialsProvider(
			getenv("AWS_ACCESS_KEY_ID", "stackyard"),
			getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
			"",
		),
	}
	appName := getenv("STACKYARD_APPCONFIG_APPLICATION", "orders")
	fmt.Printf("Stackyard AppConfig Data client using %s\n", c.endpoint)

	appID := c.id("CreateApplication", "/applications", map[string]any{"Name": appName})
	envID := c.id("CreateEnvironment", "/applications/"+appID+"/environments", map[string]any{"Name": "prod"})
	profileID := c.id("CreateConfigurationProfile", "/applications/"+appID+"/configurationprofiles", map[string]any{"Name": "feature-flags", "LocationUri": "hosted"})

	deploy := func(content string) {
		r := c.do(http.MethodPost, "/applications/"+appID+"/configurationprofiles/"+profileID+"/hostedconfigurationversions",
			[]byte(content), map[string]string{"Content-Type": "application/json"})
		expect2xx("CreateHostedConfigurationVersion", r)
		version := r.header.Get("Version-Number")
		r = c.json(http.MethodPost, "/applications/"+appID+"/environments/"+envID+"/deployments", map[string]any{
			"ConfigurationProfileId": profileID,
			"ConfigurationVersion":   version,
			"DeploymentStrategyId":   "ds-000001",
		})
		expect2xx("StartDeployment", r)
		logf("deployed version %s", version)
	}
	deploy(`{"checkout":{"enabled":false}}`)

	// Sessions accept names as well as IDs.
	r := c.json(http.MethodPost, "/configurationsessions", map[string]any{
		"ApplicationIdentifier":          appName,
		"EnvironmentIdentifier":          "prod",
		"ConfigurationProfileIdentifier": "feature-flags",
	})
	expect2xx("StartConfigurationSession", r)
	var session struct{ InitialConfigurationToken string }
	if err := json.Unmarshal(r.body, &session); err != nil || session.InitialConfigurationToken == "" {
		exitf("StartConfigurationSession returned no token: %s", r.body)
	}

	token := session.InitialConfigurationToken
	poll := func(want string) {
		r := c.do(http.MethodGet, "/configuration?configuration_token="+url.QueryEscape(token), nil, nil)
		expect2xx("GetLatestConfiguration", r)
		if got := string(r.body); got != want {
			exitf("GetLatestConfiguration: expected %q, got %q", want, got)
		}
		token = r.header.Get("Next-Poll-Configuration-Token")
		logf("configuration=%q next poll in %ss", r.body, r.header.Get("Next-Poll-Interval-In-Seconds"))
	}
	poll(`{"checkout":{"enabled":false}}`)
	poll("") // unchanged: empty body
	deploy(`{"checkout":{"enabled":true}}`)
	poll(`{"checkout":{"enabled":true}}`)
	fmt.Println("AppConfig Data example completed")
}

func (c client) id(action, path string, payload map[string]any) string {
	r := c.json(http.MethodPost, path, payload)
	expect2xx(action, r)
	var out struct{ Id string }
	if err := json.Unmarshal(r.body, &out); err != nil || out.Id == "" {
		exitf("%s returned no Id: %s", action, r.body)
	}
	logf("%s id=%s", action, out.Id)
	return out.Id
}

func (c client) json(method, path string, payload any) reply {
	body, err := json.Marshal(payload)
	if err != nil {
		exitf("marshal %s: %v", path, err)
	}
	return c.do(method, path, body, map[string]string{"Content-Type": "application/json"})
}

// do sends a request signed for "appconfig", the signing name of both
// AppConfig and AppConfig Data.
func (c client) do(method, path string, body []byte, headers map[string]string) reply {
	req, err := http.NewRequestWithContext(c.ctx, method, c.endpoint+path, bytes.NewReader(body))
	if err != nil {
		exitf("build %s %s: %v", method, path, err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	credValue, err := c.creds.Retrieve(c.ctx)
	if err != nil {
		exitf("credentials: %v", err)
	}
	sum := sha256.Sum256(body)
	if err := v4.NewSigner().SignHTTP(c.ctx, credValue, req, hex.EncodeToString(sum[:]), "appconfig", c.region, time.Now()); err != nil {
		exitf("sign %s %s: %v", method, path, err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		exitf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		exitf("read %s %s: %v", method, path, err)
	}
	return reply{status: resp.StatusCode, header: resp.Header, body: respBody}
}

func expect2xx(action string, r reply) {
	if r.status >= 200 && r.status < 300 {
		logf("%s returned %d", action, r.status)
		return
	}
	exitf("expected %s to return 2xx, got %d: %s", action, r.status, strings.TrimSpace(string(r.body)))
}

func getenv(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return def
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func logf(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}
