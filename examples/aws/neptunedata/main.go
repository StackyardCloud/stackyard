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

type requestCase struct {
	Action  string
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

	fmt.Printf("Stackyard Neptune Data API advanced client using %s\n", endpoint)

	stage1Requests := []requestCase{
		{
			Action: "GetEngineStatus",
			Method: http.MethodGet,
			Path:   "/status",
		},
		{
			Action: "ListGremlinQueries",
			Method: http.MethodGet,
			Path:   "/gremlin/status",
		},
		{
			Action: "ListOpenCypherQueries",
			Method: http.MethodGet,
			Path:   "/opencypher/status",
		},
		{
			Action: "GetPropertygraphSummary",
			Method: http.MethodGet,
			Path:   "/propertygraph/statistics/summary",
		},
		{
			Action: "GetRDFGraphSummary",
			Method: http.MethodGet,
			Path:   "/rdf/statistics/summary",
		},
	}

	for _, reqCase := range stage1Requests {
		status, body, err := neptuneDataRequest(ctx, endpoint, region, creds, reqCase.Method, reqCase.Path, reqCase.Payload)
		if err != nil {
			exitf("%s request failed: %v", reqCase.Action, err)
		}
		if err := expectSuccess(reqCase.Action, status, body); err != nil {
			exitf("%s response validation failed: %v", reqCase.Action, err)
		}
		logf("%s succeeded (%d)", reqCase.Action, status)
	}

	gremlinStatus, gremlinBody, err := neptuneDataRequest(
		ctx,
		endpoint,
		region,
		creds,
		http.MethodPost,
		"/gremlin",
		map[string]any{"gremlin": "g.V().limit(1)"},
	)
	if err != nil {
		exitf("ExecuteGremlinQuery request failed: %v", err)
	}
	if err := expectSuccess("ExecuteGremlinQuery", gremlinStatus, gremlinBody); err != nil {
		exitf("ExecuteGremlinQuery response validation failed: %v", err)
	}
	gremlinQueryID := jsonTagValue(string(gremlinBody), "queryId")
	if gremlinQueryID == "" {
		exitf("ExecuteGremlinQuery response missing queryId: %s", strings.TrimSpace(string(gremlinBody)))
	}
	logf("ExecuteGremlinQuery succeeded (%d)", gremlinStatus)

	openCypherStatus, openCypherBody, err := neptuneDataRequest(
		ctx,
		endpoint,
		region,
		creds,
		http.MethodPost,
		"/opencypher",
		map[string]any{"query": "MATCH (n) RETURN n LIMIT 1"},
	)
	if err != nil {
		exitf("ExecuteOpenCypherQuery request failed: %v", err)
	}
	if err := expectSuccess("ExecuteOpenCypherQuery", openCypherStatus, openCypherBody); err != nil {
		exitf("ExecuteOpenCypherQuery response validation failed: %v", err)
	}
	openCypherQueryID := jsonTagValue(string(openCypherBody), "queryId")
	if openCypherQueryID == "" {
		exitf("ExecuteOpenCypherQuery response missing queryId: %s", strings.TrimSpace(string(openCypherBody)))
	}
	logf("ExecuteOpenCypherQuery succeeded (%d)", openCypherStatus)

	stage2Requests := []requestCase{
		{
			Action: "ExecuteGremlinExplainQuery",
			Method: http.MethodPost,
			Path:   "/gremlin/explain",
			Payload: map[string]any{
				"gremlin": "g.V().limit(1)",
			},
		},
		{
			Action: "ExecuteGremlinProfileQuery",
			Method: http.MethodPost,
			Path:   "/gremlin/profile",
			Payload: map[string]any{
				"gremlin": "g.V().limit(1)",
			},
		},
		{
			Action: "GetGremlinQueryStatus",
			Method: http.MethodGet,
			Path:   "/gremlin/status/" + gremlinQueryID,
		},
		{
			Action: "CancelGremlinQuery",
			Method: http.MethodDelete,
			Path:   "/gremlin/status/" + gremlinQueryID,
		},
		{
			Action: "ExecuteOpenCypherExplainQuery",
			Method: http.MethodPost,
			Path:   "/opencypher/explain",
			Payload: map[string]any{
				"query":   "MATCH (n) RETURN n LIMIT 1",
				"explain": "details",
			},
		},
		{
			Action: "GetOpenCypherQueryStatus",
			Method: http.MethodGet,
			Path:   "/opencypher/status/" + openCypherQueryID,
		},
		{
			Action: "CancelOpenCypherQuery",
			Method: http.MethodDelete,
			Path:   "/opencypher/status/" + openCypherQueryID,
		},
	}

	for _, reqCase := range stage2Requests {
		status, body, err := neptuneDataRequest(ctx, endpoint, region, creds, reqCase.Method, reqCase.Path, reqCase.Payload)
		if err != nil {
			exitf("%s request failed: %v", reqCase.Action, err)
		}
		if err := expectSuccess(reqCase.Action, status, body); err != nil {
			exitf("%s response validation failed: %v", reqCase.Action, err)
		}
		logf("%s succeeded (%d)", reqCase.Action, status)
	}

	stage3StartLoader := requestCase{
		Action: "StartLoaderJob",
		Method: http.MethodPost,
		Path:   "/loader",
		Payload: map[string]any{
			"source":         "s3://stackyard-neptunedata/advanced/input.csv",
			"format":         "csv",
			"s3BucketRegion": "us-east-1",
			"iamRoleArn":     "arn:aws:iam::123456789012:role/stackyard-neptunedata",
		},
	}
	stage3Status, stage3Body, err := neptuneDataRequest(
		ctx,
		endpoint,
		region,
		creds,
		stage3StartLoader.Method,
		stage3StartLoader.Path,
		stage3StartLoader.Payload,
	)
	if err != nil {
		exitf("%s request failed: %v", stage3StartLoader.Action, err)
	}
	if err := expectSuccess(stage3StartLoader.Action, stage3Status, stage3Body); err != nil {
		exitf("%s response validation failed: %v", stage3StartLoader.Action, err)
	}
	loaderID := jsonTagValue(string(stage3Body), "loadId")
	if loaderID == "" {
		exitf("StartLoaderJob response missing loadId: %s", strings.TrimSpace(string(stage3Body)))
	}
	logf("%s succeeded (%d)", stage3StartLoader.Action, stage3Status)

	stage3Requests := []requestCase{
		{
			Action: "ListLoaderJobs",
			Method: http.MethodGet,
			Path:   "/loader?limit=10&includeQueuedLoads=false",
		},
		{
			Action: "GetLoaderJobStatus",
			Method: http.MethodGet,
			Path:   "/loader/" + loaderID + "?details=true&errors=true&page=1&errorsPerPage=10",
		},
		{
			Action: "CancelLoaderJob",
			Method: http.MethodDelete,
			Path:   "/loader/" + loaderID,
		},
	}
	for _, reqCase := range stage3Requests {
		status, body, err := neptuneDataRequest(ctx, endpoint, region, creds, reqCase.Method, reqCase.Path, reqCase.Payload)
		if err != nil {
			exitf("%s request failed: %v", reqCase.Action, err)
		}
		if err := expectSuccess(reqCase.Action, status, body); err != nil {
			exitf("%s response validation failed: %v", reqCase.Action, err)
		}
		logf("%s succeeded (%d)", reqCase.Action, status)
	}

	stage4Requests := []requestCase{
		{
			Action: "GetPropertygraphStatistics",
			Method: http.MethodGet,
			Path:   "/propertygraph/statistics",
		},
		{
			Action: "ManagePropertygraphStatistics",
			Method: http.MethodPost,
			Path:   "/propertygraph/statistics",
			Payload: map[string]any{
				"mode": "refresh",
			},
		},
		{
			Action: "GetPropertygraphStream",
			Method: http.MethodGet,
			Path:   "/propertygraph/stream?limit=5&iteratorType=LATEST&commitNum=1&opNum=1",
		},
		{
			Action: "DeletePropertygraphStatistics",
			Method: http.MethodDelete,
			Path:   "/propertygraph/statistics",
		},
		{
			Action: "GetSparqlStatistics",
			Method: http.MethodGet,
			Path:   "/sparql/statistics",
		},
		{
			Action: "ManageSparqlStatistics",
			Method: http.MethodPost,
			Path:   "/sparql/statistics",
			Payload: map[string]any{
				"mode": "enableAutoCompute",
			},
		},
		{
			Action: "GetSparqlStream",
			Method: http.MethodGet,
			Path:   "/sparql/stream?limit=5&iteratorType=LATEST&commitNum=1&opNum=1",
		},
		{
			Action: "DeleteSparqlStatistics",
			Method: http.MethodDelete,
			Path:   "/sparql/statistics",
		},
	}
	for _, reqCase := range stage4Requests {
		status, body, err := neptuneDataRequest(ctx, endpoint, region, creds, reqCase.Method, reqCase.Path, reqCase.Payload)
		if err != nil {
			exitf("%s request failed: %v", reqCase.Action, err)
		}
		if err := expectSuccess(reqCase.Action, status, body); err != nil {
			exitf("%s response validation failed: %v", reqCase.Action, err)
		}
		logf("%s succeeded (%d)", reqCase.Action, status)
	}

	stage5StartDataProcessing := requestCase{
		Action: "StartMLDataProcessingJob",
		Method: http.MethodPost,
		Path:   "/ml/dataprocessing",
		Payload: map[string]any{
			"id":                      "advanced-dataproc",
			"inputDataS3Location":     "s3://stackyard-neptunedata/advanced/raw",
			"processedDataS3Location": "s3://stackyard-neptunedata/advanced/processed",
		},
	}
	stage5Status, stage5Body, err := neptuneDataRequest(
		ctx,
		endpoint,
		region,
		creds,
		stage5StartDataProcessing.Method,
		stage5StartDataProcessing.Path,
		stage5StartDataProcessing.Payload,
	)
	if err != nil {
		exitf("%s request failed: %v", stage5StartDataProcessing.Action, err)
	}
	if err := expectSuccess(stage5StartDataProcessing.Action, stage5Status, stage5Body); err != nil {
		exitf("%s response validation failed: %v", stage5StartDataProcessing.Action, err)
	}
	dataProcessingID := jsonTagValue(string(stage5Body), "id")
	if dataProcessingID == "" {
		exitf("StartMLDataProcessingJob response missing id: %s", strings.TrimSpace(string(stage5Body)))
	}
	logf("%s succeeded (%d)", stage5StartDataProcessing.Action, stage5Status)

	stage5StartTraining := requestCase{
		Action: "StartMLModelTrainingJob",
		Method: http.MethodPost,
		Path:   "/ml/modeltraining",
		Payload: map[string]any{
			"id":                   "advanced-training",
			"dataProcessingJobId":  dataProcessingID,
			"trainModelS3Location": "s3://stackyard-neptunedata/advanced/model",
		},
	}
	stage5TrainStatus, stage5TrainBody, err := neptuneDataRequest(
		ctx,
		endpoint,
		region,
		creds,
		stage5StartTraining.Method,
		stage5StartTraining.Path,
		stage5StartTraining.Payload,
	)
	if err != nil {
		exitf("%s request failed: %v", stage5StartTraining.Action, err)
	}
	if err := expectSuccess(stage5StartTraining.Action, stage5TrainStatus, stage5TrainBody); err != nil {
		exitf("%s response validation failed: %v", stage5StartTraining.Action, err)
	}
	trainingID := jsonTagValue(string(stage5TrainBody), "id")
	if trainingID == "" {
		exitf("StartMLModelTrainingJob response missing id: %s", strings.TrimSpace(string(stage5TrainBody)))
	}
	logf("%s succeeded (%d)", stage5StartTraining.Action, stage5TrainStatus)

	stage5StartTransform := requestCase{
		Action: "StartMLModelTransformJob",
		Method: http.MethodPost,
		Path:   "/ml/modeltransform",
		Payload: map[string]any{
			"id":                             "advanced-transform",
			"modelTransformOutputS3Location": "s3://stackyard-neptunedata/advanced/transform",
		},
	}
	stage5TransformStatus, stage5TransformBody, err := neptuneDataRequest(
		ctx,
		endpoint,
		region,
		creds,
		stage5StartTransform.Method,
		stage5StartTransform.Path,
		stage5StartTransform.Payload,
	)
	if err != nil {
		exitf("%s request failed: %v", stage5StartTransform.Action, err)
	}
	if err := expectSuccess(stage5StartTransform.Action, stage5TransformStatus, stage5TransformBody); err != nil {
		exitf("%s response validation failed: %v", stage5StartTransform.Action, err)
	}
	transformID := jsonTagValue(string(stage5TransformBody), "id")
	if transformID == "" {
		exitf("StartMLModelTransformJob response missing id: %s", strings.TrimSpace(string(stage5TransformBody)))
	}
	logf("%s succeeded (%d)", stage5StartTransform.Action, stage5TransformStatus)

	stage5CreateEndpoint := requestCase{
		Action: "CreateMLEndpoint",
		Method: http.MethodPost,
		Path:   "/ml/endpoints",
		Payload: map[string]any{
			"id":                   "advanced-endpoint",
			"mlModelTrainingJobId": trainingID,
			"modelName":            "advanced-model",
		},
	}
	stage5EndpointStatus, stage5EndpointBody, err := neptuneDataRequest(
		ctx,
		endpoint,
		region,
		creds,
		stage5CreateEndpoint.Method,
		stage5CreateEndpoint.Path,
		stage5CreateEndpoint.Payload,
	)
	if err != nil {
		exitf("%s request failed: %v", stage5CreateEndpoint.Action, err)
	}
	if err := expectSuccess(stage5CreateEndpoint.Action, stage5EndpointStatus, stage5EndpointBody); err != nil {
		exitf("%s response validation failed: %v", stage5CreateEndpoint.Action, err)
	}
	endpointID := jsonTagValue(string(stage5EndpointBody), "id")
	if endpointID == "" {
		exitf("CreateMLEndpoint response missing id: %s", strings.TrimSpace(string(stage5EndpointBody)))
	}
	logf("%s succeeded (%d)", stage5CreateEndpoint.Action, stage5EndpointStatus)

	stage5Requests := []requestCase{
		{Action: "ListMLDataProcessingJobs", Method: http.MethodGet, Path: "/ml/dataprocessing?maxItems=10"},
		{Action: "GetMLDataProcessingJob", Method: http.MethodGet, Path: "/ml/dataprocessing/" + dataProcessingID},
		{Action: "CancelMLDataProcessingJob", Method: http.MethodDelete, Path: "/ml/dataprocessing/" + dataProcessingID},
		{Action: "ListMLModelTrainingJobs", Method: http.MethodGet, Path: "/ml/modeltraining?maxItems=10"},
		{Action: "GetMLModelTrainingJob", Method: http.MethodGet, Path: "/ml/modeltraining/" + trainingID},
		{Action: "CancelMLModelTrainingJob", Method: http.MethodDelete, Path: "/ml/modeltraining/" + trainingID},
		{Action: "ListMLModelTransformJobs", Method: http.MethodGet, Path: "/ml/modeltransform?maxItems=10"},
		{Action: "GetMLModelTransformJob", Method: http.MethodGet, Path: "/ml/modeltransform/" + transformID},
		{Action: "CancelMLModelTransformJob", Method: http.MethodDelete, Path: "/ml/modeltransform/" + transformID},
		{Action: "ListMLEndpoints", Method: http.MethodGet, Path: "/ml/endpoints?maxItems=10"},
		{Action: "GetMLEndpoint", Method: http.MethodGet, Path: "/ml/endpoints/" + endpointID},
		{Action: "DeleteMLEndpoint", Method: http.MethodDelete, Path: "/ml/endpoints/" + endpointID},
	}
	for _, reqCase := range stage5Requests {
		status, body, err := neptuneDataRequest(ctx, endpoint, region, creds, reqCase.Method, reqCase.Path, reqCase.Payload)
		if err != nil {
			exitf("%s request failed: %v", reqCase.Action, err)
		}
		if err := expectSuccess(reqCase.Action, status, body); err != nil {
			exitf("%s response validation failed: %v", reqCase.Action, err)
		}
		logf("%s succeeded (%d)", reqCase.Action, status)
	}

	stage6Requests := []requestCase{
		{
			Action: "ExecuteFastReset(initiate)",
			Method: http.MethodPost,
			Path:   "/system",
			Payload: map[string]any{
				"action": "initiateDatabaseReset",
			},
		},
		{
			Action: "ExecuteFastReset(perform)",
			Method: http.MethodPost,
			Path:   "/system",
			Payload: map[string]any{
				"action": "performDatabaseReset",
			},
		},
	}
	for _, reqCase := range stage6Requests {
		status, body, err := neptuneDataRequest(ctx, endpoint, region, creds, reqCase.Method, reqCase.Path, reqCase.Payload)
		if err != nil {
			exitf("%s request failed: %v", reqCase.Action, err)
		}
		if err := expectSuccess(reqCase.Action, status, body); err != nil {
			exitf("%s response validation failed: %v", reqCase.Action, err)
		}
		logf("%s succeeded (%d)", reqCase.Action, status)
	}

	fmt.Println("Done.")
}

func neptuneDataRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	method, path string,
	payload map[string]any,
) (int, []byte, error) {
	var body []byte
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return 0, nil, err
		}
		body = encoded
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
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "neptune-db", region, time.Now()); err != nil {
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

func expectSuccess(action string, status int, body []byte) error {
	if status != http.StatusOK {
		return fmt.Errorf("expected %s to return %d, got %d: %s", action, http.StatusOK, status, strings.TrimSpace(string(body)))
	}
	if strings.Contains(string(body), "NotImplemented") {
		return fmt.Errorf("expected %s to be implemented, got: %s", action, strings.TrimSpace(string(body)))
	}
	return nil
}

func jsonTagValue(payload, key string) string {
	marker := `"` + key + `":"`
	start := strings.Index(payload, marker)
	if start == -1 {
		return ""
	}
	start += len(marker)
	end := strings.Index(payload[start:], `"`)
	if end == -1 {
		return ""
	}
	return payload[start : start+end]
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
