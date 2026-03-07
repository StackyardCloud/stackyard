package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	dialogflow "cloud.google.com/go/dialogflow/apiv2"
	"cloud.google.com/go/dialogflow/apiv2/dialogflowpb"
	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type callSpec struct {
	name string
	call func(context.Context) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	agentDisplayName := getenv("STACKYARD_GCP_DIALOGFLOW_AGENT_DISPLAY_NAME", "Stackyard Agent")
	intentID := getenv("STACKYARD_GCP_DIALOGFLOW_INTENT_ID", "intent-1")
	sessionID := getenv("STACKYARD_GCP_DIALOGFLOW_SESSION_ID", "session-1")
	operationID := getenv("STACKYARD_GCP_DIALOGFLOW_OPERATION_ID", "op-1")

	projectName := "projects/" + projectID
	searchParent := "projects/-"
	intentParent := projectName + "/agent"
	intentName := intentParent + "/intents/" + intentID
	sessionName := intentParent + "/sessions/" + sessionID
	operationName := projectName + "/operations/" + operationID

	fmt.Printf("Stackyard GCP Dialogflow apiv2 SDK clients using %s (example directory dialogflow-apiv1)\n", apiEndpoint)

	agentsClient, err := dialogflow.NewAgentsRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create dialogflow agents client: %v", err)
	}
	defer closeClient("dialogflow agents client", agentsClient.Close)

	intentsClient, err := dialogflow.NewIntentsRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create dialogflow intents client: %v", err)
	}
	defer closeClient("dialogflow intents client", intentsClient.Close)

	sessionsClient, err := dialogflow.NewSessionsRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create dialogflow sessions client: %v", err)
	}
	defer closeClient("dialogflow sessions client", sessionsClient.Close)

	calls := []callSpec{
		{
			name: "SearchAgents",
			call: func(ctx context.Context) error {
				it := agentsClient.SearchAgents(ctx, &dialogflowpb.SearchAgentsRequest{
					Parent:   searchParent,
					PageSize: 1,
				})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "GetAgent",
			call: func(ctx context.Context) error {
				_, err := agentsClient.GetAgent(ctx, &dialogflowpb.GetAgentRequest{
					Parent: projectName,
				})
				return err
			},
		},
		{
			name: "SetAgent",
			call: func(ctx context.Context) error {
				_, err := agentsClient.SetAgent(ctx, &dialogflowpb.SetAgentRequest{
					Agent:      newAgent(projectName, agentDisplayName),
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"display_name", "description"}},
				})
				return err
			},
		},
		{
			name: "GetValidationResult",
			call: func(ctx context.Context) error {
				_, err := agentsClient.GetValidationResult(ctx, &dialogflowpb.GetValidationResultRequest{
					Parent:       projectName,
					LanguageCode: "en",
				})
				return err
			},
		},
		{
			name: "TrainAgent",
			call: func(ctx context.Context) error {
				_, err := agentsClient.TrainAgent(ctx, &dialogflowpb.TrainAgentRequest{
					Parent: projectName,
				})
				return err
			},
		},
		{
			name: "ListIntents",
			call: func(ctx context.Context) error {
				it := intentsClient.ListIntents(ctx, &dialogflowpb.ListIntentsRequest{
					Parent:       intentParent,
					LanguageCode: "en",
					PageSize:     1,
				})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "CreateIntent",
			call: func(ctx context.Context) error {
				_, err := intentsClient.CreateIntent(ctx, &dialogflowpb.CreateIntentRequest{
					Parent:       intentParent,
					LanguageCode: "en",
					Intent: &dialogflowpb.Intent{
						Name:        intentName,
						DisplayName: "orders.intent",
					},
				})
				return err
			},
		},
		{
			name: "GetIntent",
			call: func(ctx context.Context) error {
				_, err := intentsClient.GetIntent(ctx, &dialogflowpb.GetIntentRequest{
					Name:         intentName,
					LanguageCode: "en",
				})
				return err
			},
		},
		{
			name: "UpdateIntent",
			call: func(ctx context.Context) error {
				_, err := intentsClient.UpdateIntent(ctx, &dialogflowpb.UpdateIntentRequest{
					LanguageCode: "en",
					UpdateMask:   &fieldmaskpb.FieldMask{Paths: []string{"display_name"}},
					Intent: &dialogflowpb.Intent{
						Name:        intentName,
						DisplayName: "orders.intent.updated",
					},
				})
				return err
			},
		},
		{
			name: "DeleteIntent",
			call: func(ctx context.Context) error {
				return intentsClient.DeleteIntent(ctx, &dialogflowpb.DeleteIntentRequest{
					Name: intentName,
				})
			},
		},
		{
			name: "DetectIntent",
			call: func(ctx context.Context) error {
				_, err := sessionsClient.DetectIntent(ctx, &dialogflowpb.DetectIntentRequest{
					Session: sessionName,
					QueryInput: &dialogflowpb.QueryInput{
						Input: &dialogflowpb.QueryInput_Text{
							Text: &dialogflowpb.TextInput{
								Text:         "hello from stackyard",
								LanguageCode: "en",
							},
						},
					},
				})
				return err
			},
		},
		{
			name: "GetOperation",
			call: func(ctx context.Context) error {
				_, err := sessionsClient.GetOperation(ctx, &longrunningpb.GetOperationRequest{
					Name: operationName,
				})
				return err
			},
		},
		{
			name: "ListOperations",
			call: func(ctx context.Context) error {
				it := sessionsClient.ListOperations(ctx, &longrunningpb.ListOperationsRequest{
					Name:     projectName,
					PageSize: 1,
				})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "CancelOperation",
			call: func(ctx context.Context) error {
				return sessionsClient.CancelOperation(ctx, &longrunningpb.CancelOperationRequest{
					Name: operationName,
				})
			},
		},
	}

	for _, call := range calls {
		err := call.call(ctx)
		switch {
		case err == nil:
			logf("%s succeeded", call.name)
		case isToleratedNotImplemented(err):
			logf("%s returned NotImplemented (expected in staged emulation)", call.name)
		default:
			exitf("%s failed: %v", call.name, err)
		}
	}

	fmt.Println("StreamingDetectIntent is not invoked in this example because the SDK REST transport does not support streaming RPCs.")
	fmt.Println("Done.")
}

func newAgent(parent, displayName string) *dialogflowpb.Agent {
	return &dialogflowpb.Agent{
		Parent:              parent,
		DisplayName:         displayName,
		DefaultLanguageCode: "en",
		TimeZone:            "America/New_York",
		Description:         "Dialogflow agent from Stackyard example",
	}
}

func isToleratedNotImplemented(err error) bool {
	if err == nil {
		return false
	}

	if grpcStatus, ok := status.FromError(err); ok && grpcStatus.Code() == codes.Unimplemented {
		return true
	}

	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) && apiErr.Code == 501 {
		return true
	}

	return strings.Contains(strings.ToLower(err.Error()), "notimplemented")
}

func closeClient(label string, closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close %s: %v\n", label, err)
	}
}

func getenv(key, fallback string) string {
	if val := strings.TrimSpace(os.Getenv(key)); val != "" {
		return val
	}
	return fallback
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func logf(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}
