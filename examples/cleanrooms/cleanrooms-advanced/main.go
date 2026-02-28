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

type restCall struct {
	Name    string
	Method  string
	Path    string
	Payload map[string]any
}

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	region := getenv("AWS_REGION", "us-east-1")
	collaborationID := getenv("STACKYARD_CLEANROOMS_COLLABORATION_ID", "col-example-001")
	membershipID := getenv("STACKYARD_CLEANROOMS_MEMBERSHIP_ID", "mem-example-001")
	configuredTableID := getenv("STACKYARD_CLEANROOMS_CONFIGURED_TABLE_ID", "tbl-example-001")
	configuredTableAssociationID := getenv("STACKYARD_CLEANROOMS_CONFIGURED_TABLE_ASSOCIATION_ID", "cta-example-001")
	analysisTemplateID := getenv("STACKYARD_CLEANROOMS_ANALYSIS_TEMPLATE_ID", "at-example-001")
	privacyBudgetTemplateID := getenv("STACKYARD_CLEANROOMS_PRIVACY_BUDGET_TEMPLATE_ID", "pbt-example-001")
	protectedQueryID := getenv("STACKYARD_CLEANROOMS_PROTECTED_QUERY_ID", "pq-example-001")
	protectedJobID := getenv("STACKYARD_CLEANROOMS_PROTECTED_JOB_ID", "pj-example-001")
	idMappingTableID := getenv("STACKYARD_CLEANROOMS_ID_MAPPING_TABLE_ID", "imt-example-001")
	idNamespaceAssociationID := getenv("STACKYARD_CLEANROOMS_ID_NAMESPACE_ASSOCIATION_ID", "ina-example-001")

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard Clean Rooms advanced client using %s\n", endpoint)

	resourceARN := "arn:aws:cleanrooms:us-east-1:123456789012:collaboration/" + collaborationID
	escapedResourceARN := url.PathEscape(resourceARN)

	calls := []restCall{
		{Name: "CreateCollaboration", Method: http.MethodPost, Path: "/collaborations", Payload: map[string]any{"collaborationIdentifier": collaborationID, "name": "cleanrooms-example-collaboration"}},
		{Name: "GetCollaboration", Method: http.MethodGet, Path: "/collaborations/" + url.PathEscape(collaborationID), Payload: nil},
		{Name: "ListCollaborations", Method: http.MethodGet, Path: "/collaborations", Payload: nil},
		{Name: "CreateMembership", Method: http.MethodPost, Path: "/memberships", Payload: map[string]any{"membershipIdentifier": membershipID, "collaborationIdentifier": collaborationID}},
		{Name: "GetMembership", Method: http.MethodGet, Path: "/memberships/" + url.PathEscape(membershipID), Payload: nil},
		{Name: "CreateConfiguredTable", Method: http.MethodPost, Path: "/configuredTables", Payload: map[string]any{"configuredTableIdentifier": configuredTableID, "name": "cleanrooms-configured-table"}},
		{Name: "CreateConfiguredTableAssociation", Method: http.MethodPost, Path: "/memberships/" + url.PathEscape(membershipID) + "/configuredTableAssociations", Payload: map[string]any{"configuredTableAssociationIdentifier": configuredTableAssociationID, "configuredTableIdentifier": configuredTableID}},
		{Name: "CreateAnalysisTemplate", Method: http.MethodPost, Path: "/memberships/" + url.PathEscape(membershipID) + "/analysistemplates", Payload: map[string]any{"analysisTemplateIdentifier": analysisTemplateID, "name": "cleanrooms-analysis-template"}},
		{Name: "CreatePrivacyBudgetTemplate", Method: http.MethodPost, Path: "/memberships/" + url.PathEscape(membershipID) + "/privacybudgettemplates", Payload: map[string]any{"privacyBudgetTemplateIdentifier": privacyBudgetTemplateID}},
		{Name: "StartProtectedQuery", Method: http.MethodPost, Path: "/memberships/" + url.PathEscape(membershipID) + "/protectedQueries", Payload: map[string]any{"protectedQueryIdentifier": protectedQueryID}},
		{Name: "StartProtectedJob", Method: http.MethodPost, Path: "/memberships/" + url.PathEscape(membershipID) + "/protectedJobs", Payload: map[string]any{"protectedJobIdentifier": protectedJobID}},
		{Name: "CreateIdMappingTable", Method: http.MethodPost, Path: "/memberships/" + url.PathEscape(membershipID) + "/idmappingtables", Payload: map[string]any{"idMappingTableIdentifier": idMappingTableID}},
		{Name: "PopulateIdMappingTable", Method: http.MethodPost, Path: "/memberships/" + url.PathEscape(membershipID) + "/idmappingtables/" + url.PathEscape(idMappingTableID) + "/populate", Payload: map[string]any{}},
		{Name: "CreateIdNamespaceAssociation", Method: http.MethodPost, Path: "/memberships/" + url.PathEscape(membershipID) + "/idnamespaceassociations", Payload: map[string]any{"idNamespaceAssociationIdentifier": idNamespaceAssociationID}},
		{Name: "BatchGetSchema", Method: http.MethodPost, Path: "/collaborations/" + url.PathEscape(collaborationID) + "/batch-schema", Payload: map[string]any{"names": []string{"orders"}}},
		{Name: "ListProtectedQueries", Method: http.MethodGet, Path: "/memberships/" + url.PathEscape(membershipID) + "/protectedQueries", Payload: nil},
		{Name: "ListProtectedJobs", Method: http.MethodGet, Path: "/memberships/" + url.PathEscape(membershipID) + "/protectedJobs", Payload: nil},
		{Name: "PreviewPrivacyImpact", Method: http.MethodPost, Path: "/memberships/" + url.PathEscape(membershipID) + "/previewprivacyimpact", Payload: map[string]any{"privacyBudgetType": "DIFFERENTIAL_PRIVACY"}},
		{Name: "TagResource", Method: http.MethodPost, Path: "/tags/" + escapedResourceARN, Payload: map[string]any{"tags": map[string]any{"env": "dev", "owner": "stackyard"}}},
		{Name: "ListTagsForResource", Method: http.MethodGet, Path: "/tags/" + escapedResourceARN, Payload: nil},
		{Name: "UntagResource", Method: http.MethodDelete, Path: "/tags/" + escapedResourceARN + "?tagKeys=owner", Payload: nil},
	}

	for _, call := range calls {
		if err := runCall(ctx, endpoint, region, creds, call); err != nil {
			exitf("%s failed: %v", call.Name, err)
		}
	}

	fmt.Println("Done.")
}

func runCall(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, call restCall) error {
	status, body, err := cleanRoomsRequest(ctx, endpoint, region, creds, call.Method, call.Path, call.Payload)
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

func cleanRoomsRequest(
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
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "cleanrooms", region, time.Now()); err != nil {
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
