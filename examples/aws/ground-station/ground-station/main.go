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

type apiCall struct {
	Method  string
	Path    string
	Payload map[string]any
}

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	region := getenv("AWS_REGION", "us-east-1")

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard Ground Station advanced client using %s\n", endpoint)

	status, body, err := apiRequest(ctx, endpoint, region, creds, http.MethodPost, "/config", map[string]any{
		"name":       "stackyard-config",
		"configType": "antenna-downlink",
		"configData": map[string]any{"antennaDownlinkConfig": map[string]any{}},
	})
	mustSuccess(status, body, err, "CreateConfig")
	configID := extractString(body, []string{"configId"}, "cfg-00000001")
	configType := extractString(body, []string{"configType"}, "antenna-downlink")

	status, body, err = apiRequest(ctx, endpoint, region, creds, http.MethodPost, "/dataflowEndpointGroup", map[string]any{
		"endpointDetails": []any{},
	})
	mustSuccess(status, body, err, "CreateDataflowEndpointGroup")
	dataflowEndpointGroupID := extractString(body, []string{"dataflowEndpointGroupId"}, "deg-00000001")

	status, body, err = apiRequest(ctx, endpoint, region, creds, http.MethodPost, "/ephemeris", map[string]any{"name": "stackyard-ephemeris"})
	mustSuccess(status, body, err, "CreateEphemeris")
	ephemerisID := extractString(body, []string{"ephemerisId"}, "eph-00000001")

	status, body, err = apiRequest(ctx, endpoint, region, creds, http.MethodPost, "/missionprofile", map[string]any{"name": "stackyard-mission"})
	mustSuccess(status, body, err, "CreateMissionProfile")
	missionProfileID := extractString(body, []string{"missionProfileId"}, "mp-00000001")

	status, body, err = apiRequest(ctx, endpoint, region, creds, http.MethodPost, "/agent", map[string]any{"agentStatus": "ACTIVE"})
	mustSuccess(status, body, err, "RegisterAgent")
	agentID := extractString(body, []string{"agentId"}, "agent-00000001")

	status, body, err = apiRequest(ctx, endpoint, region, creds, http.MethodPost, "/contact", map[string]any{"missionProfileId": missionProfileID})
	mustSuccess(status, body, err, "ReserveContact")
	contactID := extractString(body, []string{"contactId"}, "contact-00000001")

	resourceARN := "arn:aws:groundstation:us-east-1:123456789012:mission-profile/" + missionProfileID
	tagPath := "/tags/" + url.PathEscape(resourceARN)

	calls := []apiCall{
		{Method: http.MethodPost, Path: "/dataflowEndpointGroupV2", Payload: map[string]any{"endpointDetails": []any{}}},
		{Method: http.MethodGet, Path: "/config"},
		{Method: http.MethodGet, Path: "/config/" + configType + "/" + configID},
		{Method: http.MethodPut, Path: "/config/" + configType + "/" + configID, Payload: map[string]any{"name": "stackyard-config-updated"}},
		{Method: http.MethodGet, Path: "/dataflowEndpointGroup"},
		{Method: http.MethodGet, Path: "/dataflowEndpointGroup/" + dataflowEndpointGroupID},
		{Method: http.MethodPost, Path: "/ephemerides", Payload: map[string]any{}},
		{Method: http.MethodGet, Path: "/ephemeris/" + ephemerisID},
		{Method: http.MethodPut, Path: "/ephemeris/" + ephemerisID, Payload: map[string]any{"name": "stackyard-ephemeris-updated"}},
		{Method: http.MethodGet, Path: "/missionprofile"},
		{Method: http.MethodGet, Path: "/missionprofile/" + missionProfileID},
		{Method: http.MethodPut, Path: "/missionprofile/" + missionProfileID, Payload: map[string]any{"name": "stackyard-mission-updated"}},
		{Method: http.MethodPost, Path: "/minute-usage", Payload: map[string]any{}},
		{Method: http.MethodPost, Path: "/contacts", Payload: map[string]any{}},
		{Method: http.MethodGet, Path: "/contact/" + contactID},
		{Method: http.MethodDelete, Path: "/contact/" + contactID},
		{Method: http.MethodGet, Path: "/agent/" + agentID + "/configuration"},
		{Method: http.MethodGet, Path: "/agentResponseUrl/" + agentID},
		{Method: http.MethodPut, Path: "/agent/" + agentID, Payload: map[string]any{"agentStatus": "ACTIVE"}},
		{Method: http.MethodGet, Path: "/groundstation"},
		{Method: http.MethodGet, Path: "/satellite"},
		{Method: http.MethodGet, Path: "/satellite/25544"},
		{Method: http.MethodPost, Path: tagPath, Payload: map[string]any{"tags": map[string]any{"env": "test", "team": "stackyard"}}},
		{Method: http.MethodGet, Path: tagPath},
		{Method: http.MethodDelete, Path: tagPath + "?tagKeys=team"},
		{Method: http.MethodDelete, Path: "/ephemeris/" + ephemerisID},
		{Method: http.MethodDelete, Path: "/dataflowEndpointGroup/" + dataflowEndpointGroupID},
		{Method: http.MethodDelete, Path: "/config/" + configType + "/" + configID},
		{Method: http.MethodDelete, Path: "/missionprofile/" + missionProfileID},
	}

	for _, call := range calls {
		status, body, err = apiRequest(ctx, endpoint, region, creds, call.Method, call.Path, call.Payload)
		mustSuccess(status, body, err, call.Method+" "+call.Path)
		fmt.Printf("%s %s returned %d\n", call.Method, call.Path, status)
	}

	fmt.Println("Done.")
}

func extractString(body []byte, keys []string, fallback string) string {
	payload := map[string]any{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return fallback
	}
	for _, key := range keys {
		parts := strings.Split(key, ".")
		var cur any = payload
		ok := true
		for _, part := range parts {
			m, good := cur.(map[string]any)
			if !good {
				ok = false
				break
			}
			cur, good = m[part]
			if !good {
				ok = false
				break
			}
		}
		if ok {
			if s, good := cur.(string); good && strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s)
			}
		}
	}
	return fallback
}

func mustSuccess(status int, body []byte, err error, action string) {
	if err != nil {
		exitf("%s request failed: %v", action, err)
	}
	if status < 200 || status >= 300 {
		exitf("%s returned HTTP %d: %s", action, status, strings.TrimSpace(string(body)))
	}
}

func apiRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	method, requestPath string,
	payload map[string]any,
) (int, []byte, error) {
	var body []byte
	var err error
	if payload != nil {
		body, err = json.Marshal(payload)
		if err != nil {
			return 0, nil, err
		}
	}

	url := strings.TrimRight(endpoint, "/") + requestPath
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Accept", "application/json")
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "groundstation", region, time.Now()); err != nil {
		return 0, nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, err
	}
	return resp.StatusCode, respBody, nil
}

func hashSHA256(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
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
