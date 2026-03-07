package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	geminidataanalytics "cloud.google.com/go/geminidataanalytics/apiv1beta"
	"cloud.google.com/go/geminidataanalytics/apiv1beta/geminidataanalyticspb"
	iampb "cloud.google.com/go/iam/apiv1/iampb"
	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	locationpb "google.golang.org/genproto/googleapis/cloud/location"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type sdkClients struct {
	dataAgent *geminidataanalytics.DataAgentClient
	dataChat  *geminidataanalytics.DataChatClient
}

type callSpec struct {
	name string
	call func(context.Context, *sdkClients) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	agentID := getenv("STACKYARD_GCP_GEMINIDA_AGENT_ID", "analytics-agent")
	conversationID := getenv("STACKYARD_GCP_GEMINIDA_CONVERSATION_ID", "conv-1")
	operationID := getenv("STACKYARD_GCP_GEMINIDA_OPERATION_ID", "op-1")

	parent := fmt.Sprintf("projects/%s/locations/%s", projectID, locationID)
	agentName := parent + "/dataAgents/" + agentID
	conversationName := parent + "/conversations/" + conversationID
	operationName := parent + "/operations/" + operationID

	fmt.Printf("Stackyard GCP Data Analytics with Gemini client using %s\n", apiEndpoint)

	clients, err := newSDKClients(ctx, apiEndpoint)
	if err != nil {
		exitf("failed to create geminidataanalytics clients: %v", err)
	}
	defer closeClient("dataagent", clients.dataAgent.Close)
	defer closeClient("datachat", clients.dataChat.Close)

	calls := []callSpec{
		{
			name: "ListDataAgents",
			call: func(ctx context.Context, c *sdkClients) error {
				it := c.dataAgent.ListDataAgents(ctx, &geminidataanalyticspb.ListDataAgentsRequest{
					Parent:   parent,
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
			name: "ListAccessibleDataAgents",
			call: func(ctx context.Context, c *sdkClients) error {
				it := c.dataAgent.ListAccessibleDataAgents(ctx, &geminidataanalyticspb.ListAccessibleDataAgentsRequest{
					Parent:   parent,
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
			name: "GetDataAgent",
			call: func(ctx context.Context, c *sdkClients) error {
				_, err := c.dataAgent.GetDataAgent(ctx, &geminidataanalyticspb.GetDataAgentRequest{
					Name: agentName,
				})
				return err
			},
		},
		{
			name: "CreateDataAgent",
			call: func(ctx context.Context, c *sdkClients) error {
				_, err := c.dataAgent.CreateDataAgent(ctx, &geminidataanalyticspb.CreateDataAgentRequest{
					Parent:      parent,
					DataAgentId: agentID,
					DataAgent:   sampleDataAgent(agentName),
				})
				return err
			},
		},
		{
			name: "UpdateDataAgent",
			call: func(ctx context.Context, c *sdkClients) error {
				_, err := c.dataAgent.UpdateDataAgent(ctx, &geminidataanalyticspb.UpdateDataAgentRequest{
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"display_name"}},
					DataAgent: &geminidataanalyticspb.DataAgent{
						Name:        agentName,
						DisplayName: "analytics-agent-updated",
					},
				})
				return err
			},
		},
		{
			name: "DeleteDataAgent",
			call: func(ctx context.Context, c *sdkClients) error {
				_, err := c.dataAgent.DeleteDataAgent(ctx, &geminidataanalyticspb.DeleteDataAgentRequest{
					Name: agentName,
				})
				return err
			},
		},
		{
			name: "GetIamPolicy",
			call: func(ctx context.Context, c *sdkClients) error {
				_, err := c.dataAgent.GetIamPolicy(ctx, &iampb.GetIamPolicyRequest{
					Resource: agentName,
				})
				return err
			},
		},
		{
			name: "SetIamPolicy",
			call: func(ctx context.Context, c *sdkClients) error {
				_, err := c.dataAgent.SetIamPolicy(ctx, &iampb.SetIamPolicyRequest{
					Resource: agentName,
					Policy:   &iampb.Policy{},
				})
				return err
			},
		},
		{
			name: "Chat",
			call: func(ctx context.Context, c *sdkClients) error {
				stream, err := c.dataChat.Chat(ctx, &geminidataanalyticspb.ChatRequest{
					Parent: parent,
					ContextProvider: &geminidataanalyticspb.ChatRequest_DataAgentContext{
						DataAgentContext: &geminidataanalyticspb.DataAgentContext{
							DataAgent: agentName,
						},
					},
					Messages: []*geminidataanalyticspb.Message{
						{
							Kind: &geminidataanalyticspb.Message_UserMessage{
								UserMessage: &geminidataanalyticspb.UserMessage{
									Kind: &geminidataanalyticspb.UserMessage_Text{Text: "show monthly revenue"},
								},
							},
						},
					},
				})
				if err != nil {
					return err
				}
				_, err = stream.Recv()
				if errors.Is(err, io.EOF) {
					return nil
				}
				return err
			},
		},
		{
			name: "CreateConversation",
			call: func(ctx context.Context, c *sdkClients) error {
				_, err := c.dataChat.CreateConversation(ctx, &geminidataanalyticspb.CreateConversationRequest{
					Parent:         parent,
					ConversationId: conversationID,
					Conversation: &geminidataanalyticspb.Conversation{
						Name:   conversationName,
						Agents: []string{agentName},
					},
				})
				return err
			},
		},
		{
			name: "GetConversation",
			call: func(ctx context.Context, c *sdkClients) error {
				_, err := c.dataChat.GetConversation(ctx, &geminidataanalyticspb.GetConversationRequest{
					Name: conversationName,
				})
				return err
			},
		},
		{
			name: "ListConversations",
			call: func(ctx context.Context, c *sdkClients) error {
				it := c.dataChat.ListConversations(ctx, &geminidataanalyticspb.ListConversationsRequest{
					Parent:   parent,
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
			name: "ListMessages",
			call: func(ctx context.Context, c *sdkClients) error {
				it := c.dataChat.ListMessages(ctx, &geminidataanalyticspb.ListMessagesRequest{
					Parent:   conversationName,
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
			name: "GetLocation",
			call: func(ctx context.Context, c *sdkClients) error {
				_, err := c.dataAgent.GetLocation(ctx, &locationpb.GetLocationRequest{
					Name: parent,
				})
				return err
			},
		},
		{
			name: "ListLocations",
			call: func(ctx context.Context, c *sdkClients) error {
				it := c.dataAgent.ListLocations(ctx, &locationpb.ListLocationsRequest{
					Name:     fmt.Sprintf("projects/%s", projectID),
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
			name: "GetOperation",
			call: func(ctx context.Context, c *sdkClients) error {
				_, err := c.dataAgent.GetOperation(ctx, &longrunningpb.GetOperationRequest{Name: operationName})
				return err
			},
		},
		{
			name: "ListOperations",
			call: func(ctx context.Context, c *sdkClients) error {
				it := c.dataAgent.ListOperations(ctx, &longrunningpb.ListOperationsRequest{
					Name:     parent,
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
			call: func(ctx context.Context, c *sdkClients) error {
				return c.dataAgent.CancelOperation(ctx, &longrunningpb.CancelOperationRequest{Name: operationName})
			},
		},
		{
			name: "DeleteOperation",
			call: func(ctx context.Context, c *sdkClients) error {
				return c.dataAgent.DeleteOperation(ctx, &longrunningpb.DeleteOperationRequest{Name: operationName})
			},
		},
	}

	for _, call := range calls {
		err := call.call(ctx, clients)
		switch {
		case err == nil:
			logf("%s succeeded", call.name)
		case isToleratedNotImplemented(err):
			logf("%s returned NotImplemented (expected in staged emulation)", call.name)
		default:
			exitf("%s failed: %v", call.name, err)
		}
	}

	fmt.Println("Done.")
}

func newSDKClients(ctx context.Context, endpoint string) (*sdkClients, error) {
	dataAgentClient, err := geminidataanalytics.NewDataAgentRESTClient(ctx,
		option.WithEndpoint(endpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		return nil, err
	}

	dataChatClient, err := geminidataanalytics.NewDataChatRESTClient(ctx,
		option.WithEndpoint(endpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		closeClient("dataagent", dataAgentClient.Close)
		return nil, err
	}

	return &sdkClients{
		dataAgent: dataAgentClient,
		dataChat:  dataChatClient,
	}, nil
}

func sampleDataAgent(name string) *geminidataanalyticspb.DataAgent {
	return &geminidataanalyticspb.DataAgent{
		Name:        name,
		DisplayName: "analytics-agent",
		Description: "stackyard sample data agent",
		Type: &geminidataanalyticspb.DataAgent_DataAnalyticsAgent{
			DataAnalyticsAgent: &geminidataanalyticspb.DataAnalyticsAgent{},
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
		fmt.Fprintf(os.Stderr, "warning: close %s client: %v\n", name, err)
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
