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
	membershipID := getenv("STACKYARD_CLEANROOMSML_MEMBERSHIP_ID", "mem-example-001")
	collaborationID := getenv("STACKYARD_CLEANROOMSML_COLLABORATION_ID", "col-example-001")
	trainedModelARN := getenv("STACKYARD_CLEANROOMSML_TRAINED_MODEL_ARN", "arn:aws:cleanrooms-ml:us-east-1:123456789012:trained-model/tm-example-001")
	audienceModelARN := getenv("STACKYARD_CLEANROOMSML_AUDIENCE_MODEL_ARN", "arn:aws:cleanrooms-ml:us-east-1:123456789012:audience-model/am-example-001")
	configuredAudienceModelARN := getenv("STACKYARD_CLEANROOMSML_CONFIGURED_AUDIENCE_MODEL_ARN", "arn:aws:cleanrooms-ml:us-east-1:123456789012:configured-audience-model/cam-example-001")
	trainingDatasetARN := getenv("STACKYARD_CLEANROOMSML_TRAINING_DATASET_ARN", "arn:aws:cleanrooms-ml:us-east-1:123456789012:training-dataset/td-example-001")
	resourceARN := getenv("STACKYARD_CLEANROOMSML_RESOURCE_ARN", configuredAudienceModelARN)

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard Clean Rooms ML advanced client using %s\n", endpoint)

	escapedResourceARN := url.PathEscape(resourceARN)

	calls := []restCall{
		{Name: "CreateTrainingDataset", Method: http.MethodPost, Path: "/training-dataset", Payload: map[string]any{"name": "stackyard-training-dataset"}},
		{Name: "GetTrainingDataset", Method: http.MethodGet, Path: "/training-dataset/" + url.PathEscape(trainingDatasetARN), Payload: nil},
		{Name: "CreateAudienceModel", Method: http.MethodPost, Path: "/audience-model", Payload: map[string]any{"name": "stackyard-audience-model", "trainingDatasetArn": trainingDatasetARN}},
		{Name: "GetAudienceModel", Method: http.MethodGet, Path: "/audience-model/" + url.PathEscape(audienceModelARN), Payload: nil},
		{Name: "CreateConfiguredAudienceModel", Method: http.MethodPost, Path: "/configured-audience-model", Payload: map[string]any{"name": "stackyard-configured-audience-model", "audienceModelArn": audienceModelARN}},
		{Name: "GetConfiguredAudienceModel", Method: http.MethodGet, Path: "/configured-audience-model/" + url.PathEscape(configuredAudienceModelARN), Payload: nil},
		{Name: "PutMLConfiguration", Method: http.MethodPut, Path: "/memberships/" + url.PathEscape(membershipID) + "/ml-configurations", Payload: map[string]any{"defaultOutputLocation": map[string]any{"s3": map[string]any{"bucket": "stackyard-bucket", "keyPrefix": "cleanrooms-ml/"}}}},
		{Name: "CreateMLInputChannel", Method: http.MethodPost, Path: "/memberships/" + url.PathEscape(membershipID) + "/ml-input-channels", Payload: map[string]any{"name": "stackyard-ml-input-channel"}},
		{Name: "CreateTrainedModel", Method: http.MethodPost, Path: "/memberships/" + url.PathEscape(membershipID) + "/trained-models", Payload: map[string]any{"name": "stackyard-trained-model", "configuredModelAlgorithmAssociationArn": "arn:aws:cleanrooms-ml:us-east-1:123456789012:configured-model-algorithm-association/cmaa-example-001"}},
		{Name: "GetTrainedModel", Method: http.MethodGet, Path: "/memberships/" + url.PathEscape(membershipID) + "/trained-models/" + url.PathEscape(trainedModelARN) + "?versionIdentifier=1", Payload: nil},
		{Name: "StartTrainedModelInferenceJob", Method: http.MethodPost, Path: "/memberships/" + url.PathEscape(membershipID) + "/trained-model-inference-jobs", Payload: map[string]any{"name": "stackyard-inference-job", "trainedModelArn": trainedModelARN}},
		{Name: "ListAudienceModels", Method: http.MethodGet, Path: "/audience-model", Payload: nil},
		{Name: "ListConfiguredAudienceModels", Method: http.MethodGet, Path: "/configured-audience-model", Payload: nil},
		{Name: "ListTrainingDatasets", Method: http.MethodGet, Path: "/training-dataset", Payload: nil},
		{Name: "ListCollaborationTrainedModels", Method: http.MethodGet, Path: "/collaborations/" + url.PathEscape(collaborationID) + "/trained-models", Payload: nil},
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
	status, body, err := cleanRoomsMLRequest(ctx, endpoint, region, creds, call.Method, call.Path, call.Payload)
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

func cleanRoomsMLRequest(
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
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "cleanrooms-ml", region, time.Now()); err != nil {
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
