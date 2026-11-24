package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
)

const educationAPIVersion = "2021-12-01-preview"
const educationPrefix = "/azure"

type educationClient struct {
	endpoint string
	pipeline runtime.Pipeline
}

type educationSharedKeyPolicy struct {
	account         string
	subscriptionKey string
}

func (p *educationSharedKeyPolicy) Do(req *policy.Request) (*http.Response, error) {
	req.Raw().Header.Set("Authorization", "SharedKey "+p.account+":signature")
	if strings.TrimSpace(p.subscriptionKey) != "" {
		req.Raw().Header.Set("Ocp-Apim-Subscription-Key", p.subscriptionKey)
	}
	return req.Next()
}

func newEducationClient(endpoint, account, subscriptionKey string) *educationClient {
	pipeline := runtime.NewPipeline(
		"stackyard",
		"education-2021-12-01-preview",
		runtime.PipelineOptions{
			PerRetry: []policy.Policy{
				&educationSharedKeyPolicy{
					account:         account,
					subscriptionKey: subscriptionKey,
				},
			},
		},
		&policy.ClientOptions{},
	)

	return &educationClient{
		endpoint: strings.TrimRight(endpoint, "/"),
		pipeline: pipeline,
	}
}

func (c *educationClient) doJSON(ctx context.Context, method, path string, payload any, expectedStatuses ...int) (map[string]any, int, error) {
	req, err := runtime.NewRequest(ctx, method, c.endpoint+path)
	if err != nil {
		return nil, 0, fmt.Errorf("create request %s %s: %w", method, path, err)
	}
	req.Raw().Header.Set("Accept", "application/json")
	if payload != nil {
		req.Raw().Header.Set("Content-Type", "application/json")
		if err := runtime.MarshalAsJSON(req, payload); err != nil {
			return nil, 0, fmt.Errorf("marshal payload %s %s: %w", method, path, err)
		}
	}

	resp, err := c.pipeline.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("execute request %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read response %s %s: %w", method, path, err)
	}
	if err := ensureStatus(resp.StatusCode, body, expectedStatuses...); err != nil {
		return nil, resp.StatusCode, err
	}
	if strings.TrimSpace(string(body)) == "" {
		return map[string]any{}, resp.StatusCode, nil
	}

	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, resp.StatusCode, fmt.Errorf("decode response %s %s: %w", method, path, err)
	}
	if decoded == nil {
		decoded = map[string]any{}
	}
	return decoded, resp.StatusCode, nil
}

func buildPath(template string, vars map[string]string) string {
	path := template
	for key, value := range vars {
		path = strings.ReplaceAll(path, "{"+key+"}", url.PathEscape(value))
	}
	return educationPrefix + path
}

func main() {
	ctx := context.Background()

	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	account := getenv("STACKYARD_AZURE_EDUCATION_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_EDUCATION_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	billingAccountName := getenv("STACKYARD_AZURE_EDUCATION_BILLING_ACCOUNT_NAME", "123456")
	billingProfileName := getenv("STACKYARD_AZURE_EDUCATION_BILLING_PROFILE_NAME", "bp-0001")
	invoiceSectionName := getenv("STACKYARD_AZURE_EDUCATION_INVOICE_SECTION_NAME", "is-0001")
	studentAlias := getenv("STACKYARD_AZURE_EDUCATION_STUDENT_ALIAS", "student1")

	client := newEducationClient(endpoint, account, subscriptionKey)

	vars := map[string]string{
		"billingAccountName": billingAccountName,
		"billingProfileName": billingProfileName,
		"invoiceSectionName": invoiceSectionName,
		"studentAlias":       studentAlias,
	}

	labCreatePayload := map[string]any{
		"properties": map[string]any{
			"description": "example lab description",
			"budgetPerStudent": map[string]any{
				"currency": "USD",
				"value":    100.0,
			},
			"displayName":    "example lab",
			"expirationDate": "2027-12-09T22:11:29.422Z",
		},
	}

	inviteCodePayload := map[string]any{
		"maxStudentCount": 10.0,
	}

	studentCreatePayload := map[string]any{
		"properties": map[string]any{
			"budget": map[string]any{
				"currency": "USD",
				"value":    100.0,
			},
			"email":          "test@contoso.com",
			"expirationDate": "2027-11-09T22:13:21.795Z",
			"firstName":      "test",
			"lastName":       "user",
			"role":           "Student",
		},
	}

	run := func(name, method, template string, payload any, statuses ...int) map[string]any {
		response, status, err := client.doJSON(ctx, method, buildPath(template, vars), payload, statuses...)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s failed: %v\n", name, err)
			os.Exit(1)
		}
		fmt.Printf("%s -> %d\n", name, status)
		if len(response) > 0 {
			encoded, err := json.MarshalIndent(response, "", "  ")
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s marshal response: %v\n", name, err)
				os.Exit(1)
			}
			fmt.Println(string(encoded))
		}
		return response
	}
	names := func(list map[string]any) []string {
		var out []string
		for _, item := range asSlice(list["value"]) {
			if m, ok := item.(map[string]any); ok {
				out = append(out, fmt.Sprint(m["name"]))
			}
		}
		return out
	}

	profile := "/providers/Microsoft.Billing/billingAccounts/{billingAccountName}/billingProfiles/{billingProfileName}"
	lab := profile + "/invoiceSections/{invoiceSectionName}/providers/Microsoft.Education/labs/default"
	q := "?api-version=" + educationAPIVersion

	run("operations_list", http.MethodGet, "/providers/Microsoft.Education/operations"+q, nil, 200)
	run("labs_create_or_update", http.MethodPut, lab+q, labCreatePayload, 200, 201)
	run("labs_get", http.MethodGet, lab+q+"&includeBudget=true", nil, 200)
	run("labs_list_all", http.MethodGet, profile+"/providers/Microsoft.Education/labs"+q, nil, 200)
	run("labs_list", http.MethodGet, profile+"/invoiceSections/{invoiceSectionName}/providers/Microsoft.Education/labs"+q, nil, 200)
	invite := run("labs_generate_invite_code", http.MethodPost, lab+"/generateInviteCode"+q, inviteCodePayload, 200)
	code := fmt.Sprint(asMap(invite["properties"])["invitationCode"])

	// Two students redeem the code: one request is approved, one denied.
	for _, who := range [][2]string{{"test", "user"}, {"other", "user"}} {
		run("redeem_invitation_code", http.MethodPost, "/providers/Microsoft.Education/redeemInvitationCode"+q,
			map[string]any{"firstName": who[0], "lastName": who[1], "redeemCode": code}, 200)
	}
	requests := names(run("join_requests_list", http.MethodGet, lab+"/joinRequests"+q, nil, 200))
	if len(requests) != 2 {
		fmt.Fprintf(os.Stderr, "expected 2 join requests, got %d\n", len(requests))
		os.Exit(1)
	}
	vars["joinRequestName"] = requests[0]
	run("join_requests_get", http.MethodGet, lab+"/joinRequests/{joinRequestName}"+q, nil, 200)
	run("join_requests_approve", http.MethodPost, lab+"/joinRequests/{joinRequestName}/approve"+q, nil, 200)
	vars["joinRequestName"] = requests[1]
	run("join_requests_deny", http.MethodPost, lab+"/joinRequests/{joinRequestName}/deny"+q, nil, 200)

	run("students_create_or_update", http.MethodPut, lab+"/students/{studentAlias}"+q, studentCreatePayload, 200, 201)
	run("students_get", http.MethodGet, lab+"/students/{studentAlias}"+q, nil, 200)
	run("students_list", http.MethodGet, lab+"/students"+q, nil, 200)

	run("grants_list_all", http.MethodGet, "/providers/Microsoft.Education/grants"+q, nil, 200)
	run("grants_list", http.MethodGet, profile+"/providers/Microsoft.Education/grants"+q, nil, 200)
	run("grants_get", http.MethodGet, profile+"/providers/Microsoft.Education/grants/default"+q+"&includeAllocatedBudget=true", nil, 200)
	studentLabs := names(run("student_labs_list_all", http.MethodGet, "/providers/Microsoft.Education/studentLabs"+q, nil, 200))
	if len(studentLabs) == 0 {
		fmt.Fprintln(os.Stderr, "expected a student lab")
		os.Exit(1)
	}
	vars["studentLabName"] = studentLabs[0]
	run("student_labs_get", http.MethodGet, "/providers/Microsoft.Education/studentLabs/{studentLabName}"+q, nil, 200)

	run("students_delete", http.MethodDelete, lab+"/students/{studentAlias}"+q, nil, 200, 204)
	run("labs_delete", http.MethodDelete, lab+q, nil, 200, 204)
}

func asMap(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func asSlice(v any) []any {
	s, _ := v.([]any)
	return s
}

func ensureStatus(actual int, body []byte, expected ...int) error {
	for _, code := range expected {
		if actual == code {
			return nil
		}
	}
	return fmt.Errorf("unexpected status %d, expected one of %v, body: %s", actual, expected, strings.TrimSpace(string(body)))
}

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
