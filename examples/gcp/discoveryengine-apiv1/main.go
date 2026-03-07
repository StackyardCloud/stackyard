package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	discoveryengine "cloud.google.com/go/discoveryengine/apiv1"
	"cloud.google.com/go/discoveryengine/apiv1/discoveryenginepb"
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
	locationID := getenv("STACKYARD_GCP_LOCATION", "global")
	collectionID := getenv("STACKYARD_GCP_DISCOVERYENGINE_COLLECTION_ID", "default_collection")
	engineID := getenv("STACKYARD_GCP_DISCOVERYENGINE_ENGINE_ID", "orders-engine")
	dataStoreID := getenv("STACKYARD_GCP_DISCOVERYENGINE_DATASTORE_ID", "orders-store")
	servingConfigID := getenv("STACKYARD_GCP_DISCOVERYENGINE_SERVING_CONFIG_ID", "default_serving_config")
	conversationID := getenv("STACKYARD_GCP_DISCOVERYENGINE_CONVERSATION_ID", "conv-1")
	sessionID := getenv("STACKYARD_GCP_DISCOVERYENGINE_SESSION_ID", "session-1")
	operationID := getenv("STACKYARD_GCP_DISCOVERYENGINE_OPERATION_ID", "op-1")

	collectionName := fmt.Sprintf("projects/%s/locations/%s/collections/%s", projectID, locationID, collectionID)
	engineName := collectionName + "/engines/" + engineID
	dataStoreName := collectionName + "/dataStores/" + dataStoreID
	servingConfigName := engineName + "/servingConfigs/" + servingConfigID
	conversationName := dataStoreName + "/conversations/" + conversationID
	sessionName := dataStoreName + "/sessions/" + sessionID
	operationName := collectionName + "/operations/" + operationID

	fmt.Printf("Stackyard GCP Discovery Engine apiv1 clients using %s\n", apiEndpoint)

	engineClient, err := discoveryengine.NewEngineRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create discoveryengine engine client: %v", err)
	}
	defer closeClient("discoveryengine engine client", engineClient.Close)

	dataStoreClient, err := discoveryengine.NewDataStoreRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create discoveryengine data store client: %v", err)
	}
	defer closeClient("discoveryengine data store client", dataStoreClient.Close)

	searchClient, err := discoveryengine.NewSearchRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create discoveryengine search client: %v", err)
	}
	defer closeClient("discoveryengine search client", searchClient.Close)

	completionClient, err := discoveryengine.NewCompletionRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create discoveryengine completion client: %v", err)
	}
	defer closeClient("discoveryengine completion client", completionClient.Close)

	conversationalClient, err := discoveryengine.NewConversationalSearchRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create discoveryengine conversational search client: %v", err)
	}
	defer closeClient("discoveryengine conversational search client", conversationalClient.Close)

	calls := []callSpec{
		{
			name: "ListEngines",
			call: func(ctx context.Context) error {
				it := engineClient.ListEngines(ctx, &discoveryenginepb.ListEnginesRequest{
					Parent:   collectionName,
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
			name: "GetEngine",
			call: func(ctx context.Context) error {
				_, err := engineClient.GetEngine(ctx, &discoveryenginepb.GetEngineRequest{Name: engineName})
				return err
			},
		},
		{
			name: "CreateEngine",
			call: func(ctx context.Context) error {
				_, err := engineClient.CreateEngine(ctx, &discoveryenginepb.CreateEngineRequest{
					Parent:   collectionName,
					EngineId: engineID,
					Engine:   newEngine(engineName, dataStoreID),
				})
				return err
			},
		},
		{
			name: "UpdateEngine",
			call: func(ctx context.Context) error {
				_, err := engineClient.UpdateEngine(ctx, &discoveryenginepb.UpdateEngineRequest{
					Engine:     newEngine(engineName, dataStoreID),
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"display_name"}},
				})
				return err
			},
		},
		{
			name: "DeleteEngine",
			call: func(ctx context.Context) error {
				_, err := engineClient.DeleteEngine(ctx, &discoveryenginepb.DeleteEngineRequest{Name: engineName})
				return err
			},
		},
		{
			name: "ListDataStores",
			call: func(ctx context.Context) error {
				it := dataStoreClient.ListDataStores(ctx, &discoveryenginepb.ListDataStoresRequest{
					Parent:   collectionName,
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
			name: "GetDataStore",
			call: func(ctx context.Context) error {
				_, err := dataStoreClient.GetDataStore(ctx, &discoveryenginepb.GetDataStoreRequest{Name: dataStoreName})
				return err
			},
		},
		{
			name: "CreateDataStore",
			call: func(ctx context.Context) error {
				_, err := dataStoreClient.CreateDataStore(ctx, &discoveryenginepb.CreateDataStoreRequest{
					Parent:      collectionName,
					DataStoreId: dataStoreID,
					DataStore:   newDataStore(dataStoreName),
				})
				return err
			},
		},
		{
			name: "UpdateDataStore",
			call: func(ctx context.Context) error {
				_, err := dataStoreClient.UpdateDataStore(ctx, &discoveryenginepb.UpdateDataStoreRequest{
					DataStore:  newDataStore(dataStoreName),
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"display_name"}},
				})
				return err
			},
		},
		{
			name: "DeleteDataStore",
			call: func(ctx context.Context) error {
				_, err := dataStoreClient.DeleteDataStore(ctx, &discoveryenginepb.DeleteDataStoreRequest{Name: dataStoreName})
				return err
			},
		},
		{
			name: "Search",
			call: func(ctx context.Context) error {
				it := searchClient.Search(ctx, &discoveryenginepb.SearchRequest{
					ServingConfig: servingConfigName,
					Query:         "latest order status",
					PageSize:      1,
				})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "SearchLite",
			call: func(ctx context.Context) error {
				it := searchClient.SearchLite(ctx, &discoveryenginepb.SearchRequest{
					ServingConfig: servingConfigName,
					Query:         "orders lite search",
					PageSize:      1,
				})
				_, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "CompleteQuery",
			call: func(ctx context.Context) error {
				_, err := completionClient.CompleteQuery(ctx, &discoveryenginepb.CompleteQueryRequest{
					DataStore: dataStoreName,
					Query:     "order",
				})
				return err
			},
		},
		{
			name: "CreateConversation",
			call: func(ctx context.Context) error {
				_, err := conversationalClient.CreateConversation(ctx, &discoveryenginepb.CreateConversationRequest{
					Parent:       dataStoreName,
					Conversation: newConversation(conversationName),
				})
				return err
			},
		},
		{
			name: "GetConversation",
			call: func(ctx context.Context) error {
				_, err := conversationalClient.GetConversation(ctx, &discoveryenginepb.GetConversationRequest{
					Name: conversationName,
				})
				return err
			},
		},
		{
			name: "ListConversations",
			call: func(ctx context.Context) error {
				it := conversationalClient.ListConversations(ctx, &discoveryenginepb.ListConversationsRequest{
					Parent:   dataStoreName,
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
			name: "UpdateConversation",
			call: func(ctx context.Context) error {
				_, err := conversationalClient.UpdateConversation(ctx, &discoveryenginepb.UpdateConversationRequest{
					Conversation: newConversation(conversationName),
					UpdateMask:   &fieldmaskpb.FieldMask{Paths: []string{"user_pseudo_id"}},
				})
				return err
			},
		},
		{
			name: "ConverseConversation",
			call: func(ctx context.Context) error {
				_, err := conversationalClient.ConverseConversation(ctx, &discoveryenginepb.ConverseConversationRequest{
					Name:          conversationName,
					ServingConfig: servingConfigName,
					Query: &discoveryenginepb.TextInput{
						Input: "show me recent orders",
					},
				})
				return err
			},
		},
		{
			name: "AnswerQuery",
			call: func(ctx context.Context) error {
				_, err := conversationalClient.AnswerQuery(ctx, &discoveryenginepb.AnswerQueryRequest{
					ServingConfig: servingConfigName,
					Session:       sessionName,
					Query: &discoveryenginepb.Query{
						Content: &discoveryenginepb.Query_Text{
							Text: "what changed in order o-1?",
						},
					},
				})
				return err
			},
		},
		{
			name: "CreateSession",
			call: func(ctx context.Context) error {
				_, err := conversationalClient.CreateSession(ctx, &discoveryenginepb.CreateSessionRequest{
					Parent:  dataStoreName,
					Session: newSession(sessionName),
				})
				return err
			},
		},
		{
			name: "GetSession",
			call: func(ctx context.Context) error {
				_, err := conversationalClient.GetSession(ctx, &discoveryenginepb.GetSessionRequest{
					Name: sessionName,
				})
				return err
			},
		},
		{
			name: "ListSessions",
			call: func(ctx context.Context) error {
				it := conversationalClient.ListSessions(ctx, &discoveryenginepb.ListSessionsRequest{
					Parent:   dataStoreName,
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
			name: "UpdateSession",
			call: func(ctx context.Context) error {
				_, err := conversationalClient.UpdateSession(ctx, &discoveryenginepb.UpdateSessionRequest{
					Session:    newSession(sessionName),
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"display_name"}},
				})
				return err
			},
		},
		{
			name: "DeleteSession",
			call: func(ctx context.Context) error {
				return conversationalClient.DeleteSession(ctx, &discoveryenginepb.DeleteSessionRequest{
					Name: sessionName,
				})
			},
		},
		{
			name: "DeleteConversation",
			call: func(ctx context.Context) error {
				return conversationalClient.DeleteConversation(ctx, &discoveryenginepb.DeleteConversationRequest{
					Name: conversationName,
				})
			},
		},
		{
			name: "GetOperation",
			call: func(ctx context.Context) error {
				_, err := engineClient.GetOperation(ctx, &longrunningpb.GetOperationRequest{
					Name: operationName,
				})
				return err
			},
		},
		{
			name: "ListOperations",
			call: func(ctx context.Context) error {
				it := engineClient.ListOperations(ctx, &longrunningpb.ListOperationsRequest{
					Name:     collectionName,
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
				return engineClient.CancelOperation(ctx, &longrunningpb.CancelOperationRequest{
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

	fmt.Println("StreamAnswerQuery is not invoked in this example because the SDK REST transport does not support server streaming RPCs.")
	fmt.Println("Done.")
}

func newEngine(name, dataStoreID string) *discoveryenginepb.Engine {
	return &discoveryenginepb.Engine{
		Name:             name,
		DisplayName:      "orders-engine",
		DataStoreIds:     []string{dataStoreID},
		SolutionType:     discoveryenginepb.SolutionType_SOLUTION_TYPE_SEARCH,
		IndustryVertical: discoveryenginepb.IndustryVertical_GENERIC,
	}
}

func newDataStore(name string) *discoveryenginepb.DataStore {
	return &discoveryenginepb.DataStore{
		Name:             name,
		DisplayName:      "orders-store",
		IndustryVertical: discoveryenginepb.IndustryVertical_GENERIC,
		ContentConfig:    discoveryenginepb.DataStore_NO_CONTENT,
	}
}

func newConversation(name string) *discoveryenginepb.Conversation {
	return &discoveryenginepb.Conversation{
		Name:         name,
		UserPseudoId: "user-1",
	}
}

func newSession(name string) *discoveryenginepb.Session {
	return &discoveryenginepb.Session{
		Name:         name,
		DisplayName:  "orders-session",
		UserPseudoId: "user-1",
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
