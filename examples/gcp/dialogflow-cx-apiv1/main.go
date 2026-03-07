package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	dialogflowcx "cloud.google.com/go/dialogflow/cx/apiv3"
	"cloud.google.com/go/dialogflow/cx/apiv3/cxpb"
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
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	agentID := getenv("STACKYARD_GCP_DIALOGFLOW_CX_AGENT_ID", "agent-1")
	flowID := getenv("STACKYARD_GCP_DIALOGFLOW_CX_FLOW_ID", "flow-1")
	sessionID := getenv("STACKYARD_GCP_DIALOGFLOW_CX_SESSION_ID", "session-1")
	operationID := getenv("STACKYARD_GCP_DIALOGFLOW_CX_OPERATION_ID", "op-1")

	locationName := fmt.Sprintf("projects/%s/locations/%s", projectID, locationID)
	agentName := locationName + "/agents/" + agentID
	flowName := agentName + "/flows/" + flowID
	sessionName := agentName + "/sessions/" + sessionID
	operationName := locationName + "/operations/" + operationID

	fmt.Printf("Stackyard GCP Dialogflow CX apiv3 SDK clients using %s (example directory dialogflow-cx-apiv1)\n", apiEndpoint)

	agentsClient, err := dialogflowcx.NewAgentsRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create dialogflow cx agents client: %v", err)
	}
	defer closeClient("dialogflow cx agents client", agentsClient.Close)

	flowsClient, err := dialogflowcx.NewFlowsRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create dialogflow cx flows client: %v", err)
	}
	defer closeClient("dialogflow cx flows client", flowsClient.Close)

	sessionsClient, err := dialogflowcx.NewSessionsRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create dialogflow cx sessions client: %v", err)
	}
	defer closeClient("dialogflow cx sessions client", sessionsClient.Close)

	calls := []callSpec{
		{
			name: "ListAgents",
			call: func(ctx context.Context) error {
				it := agentsClient.ListAgents(ctx, &cxpb.ListAgentsRequest{
					Parent:   locationName,
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
				_, err := agentsClient.GetAgent(ctx, &cxpb.GetAgentRequest{Name: agentName})
				return err
			},
		},
		{
			name: "CreateAgent",
			call: func(ctx context.Context) error {
				_, err := agentsClient.CreateAgent(ctx, &cxpb.CreateAgentRequest{
					Parent: locationName,
					Agent:  newAgent(agentName),
				})
				return err
			},
		},
		{
			name: "UpdateAgent",
			call: func(ctx context.Context) error {
				_, err := agentsClient.UpdateAgent(ctx, &cxpb.UpdateAgentRequest{
					Agent:      newAgent(agentName),
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"display_name", "description"}},
				})
				return err
			},
		},
		{
			name: "ValidateAgent",
			call: func(ctx context.Context) error {
				_, err := agentsClient.ValidateAgent(ctx, &cxpb.ValidateAgentRequest{
					Name:         agentName,
					LanguageCode: "en",
				})
				return err
			},
		},
		{
			name: "GetAgentValidationResult",
			call: func(ctx context.Context) error {
				_, err := agentsClient.GetAgentValidationResult(ctx, &cxpb.GetAgentValidationResultRequest{
					Name:         agentName + "/validationResult",
					LanguageCode: "en",
				})
				return err
			},
		},
		{
			name: "ExportAgent",
			call: func(ctx context.Context) error {
				_, err := agentsClient.ExportAgent(ctx, &cxpb.ExportAgentRequest{
					Name: agentName,
				})
				return err
			},
		},
		{
			name: "RestoreAgent",
			call: func(ctx context.Context) error {
				_, err := agentsClient.RestoreAgent(ctx, &cxpb.RestoreAgentRequest{
					Name: agentName,
				})
				return err
			},
		},
		{
			name: "ListFlows",
			call: func(ctx context.Context) error {
				it := flowsClient.ListFlows(ctx, &cxpb.ListFlowsRequest{
					Parent:   agentName,
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
			name: "GetFlow",
			call: func(ctx context.Context) error {
				_, err := flowsClient.GetFlow(ctx, &cxpb.GetFlowRequest{Name: flowName})
				return err
			},
		},
		{
			name: "CreateFlow",
			call: func(ctx context.Context) error {
				_, err := flowsClient.CreateFlow(ctx, &cxpb.CreateFlowRequest{
					Parent: agentName,
					Flow:   newFlow(flowName),
				})
				return err
			},
		},
		{
			name: "UpdateFlow",
			call: func(ctx context.Context) error {
				_, err := flowsClient.UpdateFlow(ctx, &cxpb.UpdateFlowRequest{
					Flow:       newFlow(flowName),
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"display_name", "description"}},
				})
				return err
			},
		},
		{
			name: "ValidateFlow",
			call: func(ctx context.Context) error {
				_, err := flowsClient.ValidateFlow(ctx, &cxpb.ValidateFlowRequest{
					Name:         flowName,
					LanguageCode: "en",
				})
				return err
			},
		},
		{
			name: "GetFlowValidationResult",
			call: func(ctx context.Context) error {
				_, err := flowsClient.GetFlowValidationResult(ctx, &cxpb.GetFlowValidationResultRequest{
					Name:         flowName + "/validationResult",
					LanguageCode: "en",
				})
				return err
			},
		},
		{
			name: "TrainFlow",
			call: func(ctx context.Context) error {
				_, err := flowsClient.TrainFlow(ctx, &cxpb.TrainFlowRequest{
					Name: flowName,
				})
				return err
			},
		},
		{
			name: "DetectIntent",
			call: func(ctx context.Context) error {
				_, err := sessionsClient.DetectIntent(ctx, &cxpb.DetectIntentRequest{
					Session:    sessionName,
					QueryInput: newQueryInput("hello from stackyard cx"),
				})
				return err
			},
		},
		{
			name: "MatchIntent",
			call: func(ctx context.Context) error {
				_, err := sessionsClient.MatchIntent(ctx, &cxpb.MatchIntentRequest{
					Session:    sessionName,
					QueryInput: newQueryInput("match this intent"),
				})
				return err
			},
		},
		{
			name: "FulfillIntent",
			call: func(ctx context.Context) error {
				_, err := sessionsClient.FulfillIntent(ctx, &cxpb.FulfillIntentRequest{
					MatchIntentRequest: &cxpb.MatchIntentRequest{
						Session:    sessionName,
						QueryInput: newQueryInput("fulfill this intent"),
					},
					Match: &cxpb.Match{},
				})
				return err
			},
		},
		{
			name: "GetOperation",
			call: func(ctx context.Context) error {
				_, err := agentsClient.GetOperation(ctx, &longrunningpb.GetOperationRequest{
					Name: operationName,
				})
				return err
			},
		},
		{
			name: "ListOperations",
			call: func(ctx context.Context) error {
				it := agentsClient.ListOperations(ctx, &longrunningpb.ListOperationsRequest{
					Name:     locationName,
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
				return agentsClient.CancelOperation(ctx, &longrunningpb.CancelOperationRequest{
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

	fmt.Println("StreamingDetectIntent is not invoked in this example because the SDK REST transport does not support bidirectional streaming RPCs.")
	fmt.Println("Done.")
}

func newAgent(name string) *cxpb.Agent {
	return &cxpb.Agent{
		Name:                name,
		DisplayName:         "stackyard-cx-agent",
		DefaultLanguageCode: "en",
		TimeZone:            "America/New_York",
		Description:         "Dialogflow CX agent from Stackyard example",
	}
}

func newFlow(name string) *cxpb.Flow {
	return &cxpb.Flow{
		Name:        name,
		DisplayName: "stackyard-cx-flow",
		Description: "Flow managed by Stackyard CX example",
	}
}

func newQueryInput(text string) *cxpb.QueryInput {
	return &cxpb.QueryInput{
		LanguageCode: "en",
		Input: &cxpb.QueryInput_Text{
			Text: &cxpb.TextInput{
				Text: text,
			},
		},
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

func closeClient(name string, closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close %s: %v\n", name, err)
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
