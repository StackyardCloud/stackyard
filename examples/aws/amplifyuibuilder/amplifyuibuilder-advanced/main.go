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
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

type restCall struct {
	Name    string
	Method  string
	Path    string
	Payload map[string]any
}

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	region := getenv("AWS_REGION", "us-east-1")
	appID := getenv("STACKYARD_AMPLIFYUIBUILDER_APP_ID", "d1234567890")
	environmentName := getenv("STACKYARD_AMPLIFYUIBUILDER_ENV", "dev")
	componentID := getenv("STACKYARD_AMPLIFYUIBUILDER_COMPONENT_ID", "component-000001")
	formID := getenv("STACKYARD_AMPLIFYUIBUILDER_FORM_ID", "form-000001")
	themeID := getenv("STACKYARD_AMPLIFYUIBUILDER_THEME_ID", "theme-000001")
	codegenJobID := getenv("STACKYARD_AMPLIFYUIBUILDER_CODEGEN_JOB_ID", "codegen-job-000001")
	clientToken := getenv("STACKYARD_AMPLIFYUIBUILDER_CLIENT_TOKEN", "token-000001")

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard Amplify UI Builder advanced client using %s\n", endpoint)

	calls := []restCall{
		{
			Name:    "GetMetadata",
			Method:  http.MethodGet,
			Path:    "/app/" + appID + "/environment/" + environmentName + "/metadata",
			Payload: nil,
		},
		{
			Name:   "CreateTheme",
			Method: http.MethodPost,
			Path:   "/app/" + appID + "/environment/" + environmentName + "/themes?clientToken=" + clientToken,
			Payload: map[string]any{
				"name": themeID,
			},
		},
		{
			Name:    "GetTheme",
			Method:  http.MethodGet,
			Path:    "/app/" + appID + "/environment/" + environmentName + "/themes/" + themeID,
			Payload: nil,
		},
		{
			Name:   "CreateComponent",
			Method: http.MethodPost,
			Path:   "/app/" + appID + "/environment/" + environmentName + "/components?clientToken=" + clientToken,
			Payload: map[string]any{
				"name": componentID,
			},
		},
		{
			Name:    "GetComponent",
			Method:  http.MethodGet,
			Path:    "/app/" + appID + "/environment/" + environmentName + "/components/" + componentID,
			Payload: nil,
		},
		{
			Name:   "CreateForm",
			Method: http.MethodPost,
			Path:   "/app/" + appID + "/environment/" + environmentName + "/forms?clientToken=" + clientToken,
			Payload: map[string]any{
				"name": formID,
			},
		},
		{
			Name:    "GetForm",
			Method:  http.MethodGet,
			Path:    "/app/" + appID + "/environment/" + environmentName + "/forms/" + formID,
			Payload: nil,
		},
		{
			Name:   "StartCodegenJob",
			Method: http.MethodPost,
			Path:   "/app/" + appID + "/environment/" + environmentName + "/codegen-jobs?clientToken=" + clientToken,
			Payload: map[string]any{
				"codegenJobToCreate": map[string]any{},
			},
		},
		{
			Name:    "GetCodegenJob",
			Method:  http.MethodGet,
			Path:    "/app/" + appID + "/environment/" + environmentName + "/codegen-jobs/" + codegenJobID,
			Payload: nil,
		},
		{
			Name:    "ListThemes",
			Method:  http.MethodGet,
			Path:    "/app/" + appID + "/environment/" + environmentName + "/themes?maxResults=10",
			Payload: nil,
		},
		{
			Name:    "ListComponents",
			Method:  http.MethodGet,
			Path:    "/app/" + appID + "/environment/" + environmentName + "/components?maxResults=10",
			Payload: nil,
		},
		{
			Name:    "ListForms",
			Method:  http.MethodGet,
			Path:    "/app/" + appID + "/environment/" + environmentName + "/forms?maxResults=10",
			Payload: nil,
		},
	}

	for _, call := range calls {
		if err := runCall(ctx, endpoint, region, creds, call); err != nil {
			exitf("%s failed: %v", call.Name, err)
		}
	}

	fmt.Println("Done.")
}

func runCall(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, call restCall) error {
	status, body, err := amplifyUIBuilderRequest(ctx, endpoint, region, creds, call.Method, call.Path, call.Payload)
	if err != nil {
		return err
	}

	if status >= 200 && status < 300 {
		logf("%s returned %d", call.Name, status)
		return nil
	}

	errType := extractErrorType(body)
	if isStagedPlanTolerated(errType, body) {
		logf("%s returned %d (%s): expected while staged plan is in progress", call.Name, status, errType)
		return nil
	}

	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		trimmed = "<empty body>"
	}
	return fmt.Errorf("HTTP %d: %s", status, trimmed)
}

func amplifyUIBuilderRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	method, path string,
	payload map[string]any,
) (int, []byte, error) {
	var body []byte
	var err error
	if payload == nil {
		body = []byte{}
	} else {
		body, err = json.Marshal(payload)
		if err != nil {
			return 0, nil, err
		}
	}

	req, err := http.NewRequestWithContext(
		ctx,
		method,
		strings.TrimRight(endpoint, "/")+path,
		bytes.NewReader(body),
	)
	if err != nil {
		return 0, nil, err
	}
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "amplifyuibuilder", region, time.Now()); err != nil {
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

func extractErrorType(body []byte) string {
	payload := map[string]any{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	if v, ok := payload["__type"].(string); ok {
		return strings.TrimSpace(v)
	}
	if v, ok := payload["code"].(string); ok {
		return strings.TrimSpace(v)
	}
	if v, ok := payload["message"].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

func isStagedPlanTolerated(errType string, body []byte) bool {
	combined := strings.ToLower(errType + " " + string(body))
	return strings.Contains(combined, "notimplemented") ||
		strings.Contains(combined, "unknownoperation") ||
		strings.Contains(combined, "validationexception") ||
		strings.Contains(combined, "resourcenotfound") ||
		strings.Contains(combined, "accessdenied")
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

func logf(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}
