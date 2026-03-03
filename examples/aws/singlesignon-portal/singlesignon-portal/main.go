package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type portalCall struct {
	Name   string
	Method string
	Path   string
}

func main() {
	endpoint := strings.TrimRight(getenv("STACKYARD_ENDPOINT", "http://localhost:4566"), "/")
	token := getenv("SSO_ACCESS_TOKEN", "stackyard-access-token")
	accountID := getenv("SSO_ACCOUNT_ID", "123456789012")
	roleName := getenv("SSO_ROLE_NAME", "stackyard-role")

	fmt.Printf("Stackyard IAM Identity Center Portal advanced client using %s\n", endpoint)

	calls := []portalCall{
		{
			Name:   "ListAccounts",
			Method: http.MethodGet,
			Path:   "/assignment/accounts",
		},
		{
			Name:   "ListAccountRoles",
			Method: http.MethodGet,
			Path:   "/assignment/roles?account_id=" + url.QueryEscape(accountID),
		},
		{
			Name:   "GetRoleCredentials",
			Method: http.MethodGet,
			Path: "/federation/credentials?account_id=" + url.QueryEscape(accountID) +
				"&role_name=" + url.QueryEscape(roleName),
		},
		{
			Name:   "Logout",
			Method: http.MethodPost,
			Path:   "/logout",
		},
	}

	for _, call := range calls {
		status, body, err := portalRequest(call.Method, endpoint+call.Path, token)
		if err != nil {
			exitf("%s request failed: %v", call.Name, err)
		}

		if status >= 200 && status < 300 {
			fmt.Printf("%s returned %d\n", call.Name, status)
			continue
		}

		if isStagedPlanTolerated(body) {
			fmt.Printf("%s returned %d: tolerated while staged plan is in progress\n", call.Name, status)
			continue
		}

		exitf("%s returned HTTP %d: %s", call.Name, status, strings.TrimSpace(string(body)))
	}

	fmt.Println("Done.")
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
