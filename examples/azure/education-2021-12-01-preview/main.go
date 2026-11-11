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

type educationCall struct {
	name     string
	method   string
	template string
	payload  any
	statuses []int
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
	joinRequestName := getenv("STACKYARD_AZURE_EDUCATION_JOIN_REQUEST_NAME", "joinrequest1")
	studentAlias := getenv("STACKYARD_AZURE_EDUCATION_STUDENT_ALIAS", "student1")
	studentLabName := getenv("STACKYARD_AZURE_EDUCATION_STUDENT_LAB_NAME", "studentlab1")
	redeemCode := getenv("STACKYARD_AZURE_EDUCATION_REDEEM_CODE", "exampleRedeemCode")

	client := newEducationClient(endpoint, account, subscriptionKey)

	vars := map[string]string{
		"billingAccountName": billingAccountName,
		"billingProfileName": billingProfileName,
		"invoiceSectionName": invoiceSectionName,
		"joinRequestName":    joinRequestName,
		"studentAlias":       studentAlias,
		"studentLabName":     studentLabName,
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

	redeemPayload := map[string]any{
		"firstName":  "test",
		"lastName":   "user",
		"redeemCode": redeemCode,
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

	calls := []educationCall{
		{
			name:     "operations_list",
			method:   http.MethodGet,
			template: "/providers/Microsoft.Education/operations?api-version=2021-12-01-preview",
			statuses: []int{200},
		},
		{
			name:     "grants_list_all",
			method:   http.MethodGet,
			template: "/providers/Microsoft.Education/grants?api-version=2021-12-01-preview",
			statuses: []int{200},
		},
		{
			name:     "grants_list",
			method:   http.MethodGet,
			template: "/providers/Microsoft.Billing/billingAccounts/{billingAccountName}/billingProfiles/{billingProfileName}/providers/Microsoft.Education/grants?api-version=2021-12-01-preview",
			statuses: []int{200},
		},
		{
			name:     "grants_get",
			method:   http.MethodGet,
			template: "/providers/Microsoft.Billing/billingAccounts/{billingAccountName}/billingProfiles/{billingProfileName}/providers/Microsoft.Education/grants/default?api-version=2021-12-01-preview",
			statuses: []int{200},
		},
		{
			name:     "labs_list_all",
			method:   http.MethodGet,
			template: "/providers/Microsoft.Billing/billingAccounts/{billingAccountName}/billingProfiles/{billingProfileName}/providers/Microsoft.Education/labs?api-version=2021-12-01-preview",
			statuses: []int{200},
		},
		{
			name:     "labs_list",
			method:   http.MethodGet,
			template: "/providers/Microsoft.Billing/billingAccounts/{billingAccountName}/billingProfiles/{billingProfileName}/invoiceSections/{invoiceSectionName}/providers/Microsoft.Education/labs?api-version=2021-12-01-preview",
			statuses: []int{200},
		},
		{
			name:     "labs_create_or_update",
			method:   http.MethodPut,
			template: "/providers/Microsoft.Billing/billingAccounts/{billingAccountName}/billingProfiles/{billingProfileName}/invoiceSections/{invoiceSectionName}/providers/Microsoft.Education/labs/default?api-version=2021-12-01-preview",
			payload:  labCreatePayload,
			statuses: []int{200, 201},
		},
		{
			name:     "labs_get",
			method:   http.MethodGet,
			template: "/providers/Microsoft.Billing/billingAccounts/{billingAccountName}/billingProfiles/{billingProfileName}/invoiceSections/{invoiceSectionName}/providers/Microsoft.Education/labs/default?api-version=2021-12-01-preview",
			statuses: []int{200},
		},
		{
			name:     "labs_generate_invite_code",
			method:   http.MethodPost,
			template: "/providers/Microsoft.Billing/billingAccounts/{billingAccountName}/billingProfiles/{billingProfileName}/invoiceSections/{invoiceSectionName}/providers/Microsoft.Education/labs/default/generateInviteCode?api-version=2021-12-01-preview",
			payload:  inviteCodePayload,
			statuses: []int{200},
		},
		{
			name:     "join_requests_list",
			method:   http.MethodGet,
			template: "/providers/Microsoft.Billing/billingAccounts/{billingAccountName}/billingProfiles/{billingProfileName}/invoiceSections/{invoiceSectionName}/providers/Microsoft.Education/labs/default/joinRequests?api-version=2021-12-01-preview",
			statuses: []int{200},
		},
		{
			name:     "join_requests_get",
			method:   http.MethodGet,
			template: "/providers/Microsoft.Billing/billingAccounts/{billingAccountName}/billingProfiles/{billingProfileName}/invoiceSections/{invoiceSectionName}/providers/Microsoft.Education/labs/default/joinRequests/{joinRequestName}?api-version=2021-12-01-preview",
			statuses: []int{200},
		},
		{
			name:     "join_requests_approve",
			method:   http.MethodPost,
			template: "/providers/Microsoft.Billing/billingAccounts/{billingAccountName}/billingProfiles/{billingProfileName}/invoiceSections/{invoiceSectionName}/providers/Microsoft.Education/labs/default/joinRequests/{joinRequestName}/approve?api-version=2021-12-01-preview",
			statuses: []int{200},
		},
		{
			name:     "join_requests_deny",
			method:   http.MethodPost,
			template: "/providers/Microsoft.Billing/billingAccounts/{billingAccountName}/billingProfiles/{billingProfileName}/invoiceSections/{invoiceSectionName}/providers/Microsoft.Education/labs/default/joinRequests/{joinRequestName}/deny?api-version=2021-12-01-preview",
			statuses: []int{200},
		},
		{
			name:     "students_list",
			method:   http.MethodGet,
			template: "/providers/Microsoft.Billing/billingAccounts/{billingAccountName}/billingProfiles/{billingProfileName}/invoiceSections/{invoiceSectionName}/providers/Microsoft.Education/labs/default/students?api-version=2021-12-01-preview",
			statuses: []int{200},
		},
		{
			name:     "students_create_or_update",
			method:   http.MethodPut,
			template: "/providers/Microsoft.Billing/billingAccounts/{billingAccountName}/billingProfiles/{billingProfileName}/invoiceSections/{invoiceSectionName}/providers/Microsoft.Education/labs/default/students/{studentAlias}?api-version=2021-12-01-preview",
			payload:  studentCreatePayload,
			statuses: []int{200, 201},
		},
		{
			name:     "students_get",
			method:   http.MethodGet,
			template: "/providers/Microsoft.Billing/billingAccounts/{billingAccountName}/billingProfiles/{billingProfileName}/invoiceSections/{invoiceSectionName}/providers/Microsoft.Education/labs/default/students/{studentAlias}?api-version=2021-12-01-preview",
			statuses: []int{200},
		},
		{
			name:     "student_labs_list_all",
			method:   http.MethodGet,
			template: "/providers/Microsoft.Education/studentLabs?api-version=2021-12-01-preview",
			statuses: []int{200},
		},
		{
			name:     "student_labs_get",
			method:   http.MethodGet,
			template: "/providers/Microsoft.Education/studentLabs/{studentLabName}?api-version=2021-12-01-preview",
			statuses: []int{200},
		},
		{
			name:     "redeem_invitation_code",
			method:   http.MethodPost,
			template: "/providers/Microsoft.Education/redeemInvitationCode?api-version=2021-12-01-preview",
			payload:  redeemPayload,
			statuses: []int{200},
		},
		{
			name:     "students_delete",
			method:   http.MethodDelete,
			template: "/providers/Microsoft.Billing/billingAccounts/{billingAccountName}/billingProfiles/{billingProfileName}/invoiceSections/{invoiceSectionName}/providers/Microsoft.Education/labs/default/students/{studentAlias}?api-version=2021-12-01-preview",
			statuses: []int{200, 204},
		},
		{
			name:     "labs_delete",
			method:   http.MethodDelete,
			template: "/providers/Microsoft.Billing/billingAccounts/{billingAccountName}/billingProfiles/{billingProfileName}/invoiceSections/{invoiceSectionName}/providers/Microsoft.Education/labs/default?api-version=2021-12-01-preview",
			statuses: []int{200, 204},
		},
	}

	for _, call := range calls {
		path := buildPath(call.template, vars)
		response, status, err := client.doJSON(ctx, call.method, path, call.payload, call.statuses...)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s failed: %v\n", call.name, err)
			os.Exit(1)
		}
		fmt.Printf("%s -> %d\n", call.name, status)
		if len(response) > 0 {
			encoded, err := json.MarshalIndent(response, "", "  ")
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s marshal response: %v\n", call.name, err)
				os.Exit(1)
			}
			fmt.Println(string(encoded))
		}
	}
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
