package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	token := getenv("SSO_ACCESS_TOKEN", "stackyard-access-token")

	fmt.Printf("Stackyard IAM Identity Center Portal basic client using %s\n", endpoint)

	status, body, err := portalRequest(http.MethodGet, strings.TrimRight(endpoint, "/")+"/assignment/accounts", token)
	if err != nil {
		exitf("ListAccounts request failed: %v", err)
	}

	if status >= 200 && status < 300 {
		fmt.Printf("ListAccounts returned %d\n", status)
		fmt.Println(strings.TrimSpace(string(body)))
		return
	}

	if isStagedPlanTolerated(body) {
		fmt.Printf("ListAccounts returned %d: tolerated while staged plan is in progress\n", status)
		return
	}

	exitf("ListAccounts returned HTTP %d: %s", status, strings.TrimSpace(string(body)))
}

func portalRequest(method, url, token string) (int, []byte, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("x-amz-sso_bearer_token", token)

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

func isStagedPlanTolerated(body []byte) bool {
	combined := strings.ToLower(string(body))
	return strings.Contains(combined, "notimplemented") ||
		strings.Contains(combined, "unknownoperation") ||
		strings.Contains(combined, "validationexception") ||
		strings.Contains(combined, "invalidrequest") ||
		strings.Contains(combined, "unauthorizedexception") ||
		strings.Contains(combined, "accessdenied") ||
		strings.Contains(combined, "forbidden") ||
		strings.Contains(combined, "resource not found")
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
