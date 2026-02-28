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

	fmt.Printf("Stackyard MediaConnect advanced client using %s\n", endpoint)

	status, body, err := apiRequest(ctx, endpoint, region, creds, http.MethodPost, "/v1/flows", map[string]any{
		"Name": "stackyard-flow",
	})
	mustSuccess(status, body, err, "CreateFlow")
	flowArn := extractString(body, []string{"FlowArn", "Flow.FlowArn"}, "arn:aws:mediaconnect:us-east-1:123456789012:flow/flow-00000001")

	status, body, err = apiRequest(ctx, endpoint, region, creds, http.MethodPost, "/v1/bridges", map[string]any{"Name": "stackyard-bridge"})
	mustSuccess(status, body, err, "CreateBridge")
	bridgeArn := extractString(body, []string{"BridgeArn", "Bridge.BridgeArn"}, "arn:aws:mediaconnect:us-east-1:123456789012:bridge/bridge-00000001")

	status, body, err = apiRequest(ctx, endpoint, region, creds, http.MethodPost, "/v1/gateways", map[string]any{"Name": "stackyard-gateway"})
	mustSuccess(status, body, err, "CreateGateway")
	gatewayArn := extractString(body, []string{"GatewayArn", "Gateway.GatewayArn"}, "arn:aws:mediaconnect:us-east-1:123456789012:gateway/gateway-00000001")

	status, body, err = apiRequest(ctx, endpoint, region, creds, http.MethodPost, "/v1/routerInput", map[string]any{"Name": "stackyard-router-input"})
	mustSuccess(status, body, err, "CreateRouterInput")
	routerInputArn := extractString(body, []string{"RouterInputArn", "RouterInput.RouterInputArn"}, "arn:aws:mediaconnect:us-east-1:123456789012:router-input/router-input-00000001")

	status, body, err = apiRequest(ctx, endpoint, region, creds, http.MethodPost, "/v1/routerOutput", map[string]any{"Name": "stackyard-router-output"})
	mustSuccess(status, body, err, "CreateRouterOutput")
	routerOutputArn := extractString(body, []string{"RouterOutputArn", "RouterOutput.RouterOutputArn"}, "arn:aws:mediaconnect:us-east-1:123456789012:router-output/router-output-00000001")

	status, body, err = apiRequest(ctx, endpoint, region, creds, http.MethodPost, "/v1/routerNetworkInterface", map[string]any{"Name": "stackyard-router-interface"})
	mustSuccess(status, body, err, "CreateRouterNetworkInterface")
	routerIfaceArn := extractString(body, []string{"RouterNetworkInterfaceArn", "RouterNetworkInterface.RouterNetworkInterfaceArn"}, "arn:aws:mediaconnect:us-east-1:123456789012:router-interface/router-interface-00000001")

	tagPath := "/tags/" + url.PathEscape(flowArn)
	globalTagPath := "/tags/global/" + url.PathEscape(flowArn)

	calls := []apiCall{
		{Method: http.MethodPost, Path: "/v1/flows/" + url.PathEscape(flowArn) + "/outputs", Payload: map[string]any{}},
		{Method: http.MethodPost, Path: "/v1/flows/" + url.PathEscape(flowArn) + "/source", Payload: map[string]any{}},
		{Method: http.MethodPost, Path: "/v1/flows/" + url.PathEscape(flowArn) + "/mediaStreams", Payload: map[string]any{}},
		{Method: http.MethodPost, Path: "/v1/flows/" + url.PathEscape(flowArn) + "/vpcInterfaces", Payload: map[string]any{}},
		{Method: http.MethodPost, Path: "/v1/bridges/" + url.PathEscape(bridgeArn) + "/outputs", Payload: map[string]any{}},
		{Method: http.MethodPost, Path: "/v1/bridges/" + url.PathEscape(bridgeArn) + "/sources", Payload: map[string]any{}},
		{Method: http.MethodGet, Path: "/v1/flows/" + url.PathEscape(flowArn)},
		{Method: http.MethodGet, Path: "/v1/bridges/" + url.PathEscape(bridgeArn)},
		{Method: http.MethodGet, Path: "/v1/gateways/" + url.PathEscape(gatewayArn)},
		{Method: http.MethodGet, Path: "/v1/routerInput/" + url.PathEscape(routerInputArn)},
		{Method: http.MethodGet, Path: "/v1/routerOutput/" + url.PathEscape(routerOutputArn)},
		{Method: http.MethodGet, Path: "/v1/routerNetworkInterface/" + url.PathEscape(routerIfaceArn)},
		{Method: http.MethodGet, Path: "/v1/flows"},
		{Method: http.MethodGet, Path: "/v1/bridges"},
		{Method: http.MethodGet, Path: "/v1/gateways"},
		{Method: http.MethodGet, Path: "/v1/routerInputs"},
		{Method: http.MethodGet, Path: "/v1/routerOutputs"},
		{Method: http.MethodGet, Path: "/v1/routerNetworkInterfaces"},
		{Method: http.MethodPost, Path: "/v1/flows/" + url.PathEscape(flowArn) + "/entitlements", Payload: map[string]any{}},
		{Method: http.MethodGet, Path: "/v1/entitlements"},
		{Method: http.MethodGet, Path: "/v1/offerings"},
		{Method: http.MethodGet, Path: "/v1/reservations"},
		{Method: http.MethodPost, Path: tagPath, Payload: map[string]any{"Tags": map[string]any{"env": "test", "team": "stackyard"}}},
		{Method: http.MethodGet, Path: tagPath},
		{Method: http.MethodDelete, Path: tagPath + "?tagKeys=team"},
		{Method: http.MethodPost, Path: globalTagPath, Payload: map[string]any{"Tags": map[string]any{"scope": "global"}}},
		{Method: http.MethodGet, Path: globalTagPath},
		{Method: http.MethodDelete, Path: globalTagPath + "?tagKeys=scope"},
		{Method: http.MethodPost, Path: "/v1/flows/start/" + url.PathEscape(flowArn), Payload: map[string]any{}},
		{Method: http.MethodPost, Path: "/v1/flows/stop/" + url.PathEscape(flowArn), Payload: map[string]any{}},
		{Method: http.MethodPost, Path: "/v1/routerInput/start/" + url.PathEscape(routerInputArn), Payload: map[string]any{}},
		{Method: http.MethodPost, Path: "/v1/routerInput/restart/" + url.PathEscape(routerInputArn), Payload: map[string]any{}},
		{Method: http.MethodPut, Path: "/v1/routerOutput/takeRouterInput/" + url.PathEscape(routerOutputArn), Payload: map[string]any{}},
		{Method: http.MethodPost, Path: "/v1/routerInput/stop/" + url.PathEscape(routerInputArn), Payload: map[string]any{}},
		{Method: http.MethodPost, Path: "/v1/routerOutput/start/" + url.PathEscape(routerOutputArn), Payload: map[string]any{}},
		{Method: http.MethodPost, Path: "/v1/routerOutput/restart/" + url.PathEscape(routerOutputArn), Payload: map[string]any{}},
		{Method: http.MethodPost, Path: "/v1/routerOutput/stop/" + url.PathEscape(routerOutputArn), Payload: map[string]any{}},
		{Method: http.MethodDelete, Path: "/v1/routerNetworkInterface/" + url.PathEscape(routerIfaceArn)},
		{Method: http.MethodDelete, Path: "/v1/routerOutput/" + url.PathEscape(routerOutputArn)},
		{Method: http.MethodDelete, Path: "/v1/routerInput/" + url.PathEscape(routerInputArn)},
		{Method: http.MethodDelete, Path: "/v1/gateways/" + url.PathEscape(gatewayArn)},
		{Method: http.MethodDelete, Path: "/v1/bridges/" + url.PathEscape(bridgeArn)},
		{Method: http.MethodDelete, Path: "/v1/flows/" + url.PathEscape(flowArn)},
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
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "mediaconnect", region, time.Now()); err != nil {
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
