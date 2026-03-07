package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	spanner "cloud.google.com/go/spanner/apiv1"
	"cloud.google.com/go/spanner/apiv1/spannerpb"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/types/known/structpb"
)

type callSpec struct {
	name string
	call func(context.Context, *spanner.Client) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	instanceID := getenv("STACKYARD_GCP_SPANNER_INSTANCE", "stackyard-instance")
	databaseID := getenv("STACKYARD_GCP_SPANNER_DATABASE", "stackyard-db")
	sessionID := getenv("STACKYARD_GCP_SPANNER_SESSION_ID", "s-1")
	readTable := getenv("STACKYARD_GCP_SPANNER_TABLE", "Users")

	databaseName := fmt.Sprintf("projects/%s/instances/%s/databases/%s", projectID, instanceID, databaseID)
	sessionName := fmt.Sprintf("%s/sessions/%s", databaseName, sessionID)

	fmt.Printf("Stackyard GCP Cloud Spanner apiv1 client using %s\n", apiEndpoint)
	if err := waitForStackyardReady(ctx, apiEndpoint, sessionName); err != nil {
		exitf("stackyard readiness check failed: %v", err)
	}

	httpClient := &http.Client{
		Transport: stackyardHeaderTransport{
			base:        http.DefaultTransport,
			serviceName: "spanner",
		},
	}

	client, err := spanner.NewRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create spanner client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "CreateSession",
			call: func(ctx context.Context, c *spanner.Client) error {
				_, err := c.CreateSession(ctx, &spannerpb.CreateSessionRequest{
					Database: databaseName,
					Session:  &spannerpb.Session{Name: sessionName},
				})
				return err
			},
		},
		{
			name: "GetSession",
			call: func(ctx context.Context, c *spanner.Client) error {
				_, err := c.GetSession(ctx, &spannerpb.GetSessionRequest{Name: sessionName})
				return err
			},
		},
		{
			name: "ListSessions",
			call: func(ctx context.Context, c *spanner.Client) error {
				it := c.ListSessions(ctx, &spannerpb.ListSessionsRequest{
					Database: databaseName,
					PageSize: 1,
				})
				_, err := it.Next()
				if err == iterator.Done {
					return nil
				}
				return err
			},
		},
		{
			name: "ExecuteSql",
			call: func(ctx context.Context, c *spanner.Client) error {
				_, err := c.ExecuteSql(ctx, &spannerpb.ExecuteSqlRequest{
					Session: sessionName,
					Sql:     "SELECT 1",
				})
				return err
			},
		},
		{
			name: "Read",
			call: func(ctx context.Context, c *spanner.Client) error {
				_, err := c.Read(ctx, &spannerpb.ReadRequest{
					Session: sessionName,
					Table:   readTable,
					Columns: []string{"UserId", "Name"},
					KeySet:  &spannerpb.KeySet{All: true},
				})
				return err
			},
		},
		{
			name: "BeginTransaction",
			call: func(ctx context.Context, c *spanner.Client) error {
				tx, err := c.BeginTransaction(ctx, &spannerpb.BeginTransactionRequest{
					Session: sessionName,
					Options: &spannerpb.TransactionOptions{
						Mode: &spannerpb.TransactionOptions_ReadWrite_{
							ReadWrite: &spannerpb.TransactionOptions_ReadWrite{},
						},
					},
				})
				if err != nil {
					return err
				}
				if len(tx.GetId()) > 0 {
					return nil
				}
				return fmt.Errorf("begin transaction returned empty id")
			},
		},
		{
			name: "Commit",
			call: func(ctx context.Context, c *spanner.Client) error {
				_, err := c.Commit(ctx, &spannerpb.CommitRequest{
					Session: sessionName,
					Transaction: &spannerpb.CommitRequest_TransactionId{
						TransactionId: []byte("tx-s-1"),
					},
					Mutations: []*spannerpb.Mutation{
						{
							Operation: &spannerpb.Mutation_Insert{
								Insert: &spannerpb.Mutation_Write{
									Table:   readTable,
									Columns: []string{"UserId", "Name"},
									Values: []*structpb.ListValue{
										{
											Values: []*structpb.Value{
												structpb.NewStringValue("1"),
												structpb.NewStringValue("Stackyard"),
											},
										},
									},
								},
							},
						},
					},
				})
				return err
			},
		},
		{
			name: "PartitionQuery",
			call: func(ctx context.Context, c *spanner.Client) error {
				_, err := c.PartitionQuery(ctx, &spannerpb.PartitionQueryRequest{
					Session: sessionName,
					Transaction: &spannerpb.TransactionSelector{
						Selector: &spannerpb.TransactionSelector_Id{Id: []byte("tx-s-1")},
					},
					Sql: "SELECT * FROM Users",
				})
				return err
			},
		},
		{
			name: "PartitionRead",
			call: func(ctx context.Context, c *spanner.Client) error {
				_, err := c.PartitionRead(ctx, &spannerpb.PartitionReadRequest{
					Session: sessionName,
					Transaction: &spannerpb.TransactionSelector{
						Selector: &spannerpb.TransactionSelector_Id{Id: []byte("tx-s-1")},
					},
					Table:   readTable,
					Columns: []string{"UserId", "Name"},
					KeySet:  &spannerpb.KeySet{All: true},
				})
				return err
			},
		},
		{
			name: "DeleteSession",
			call: func(ctx context.Context, c *spanner.Client) error {
				return c.DeleteSession(ctx, &spannerpb.DeleteSessionRequest{Name: sessionName})
			},
		},
	}

	for _, call := range calls {
		if err := call.call(ctx, client); err != nil {
			exitf("%s failed: %v", call.name, err)
		}
		logf("%s succeeded", call.name)
	}

	fmt.Println("Done.")
}

func closeClient(closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close spanner client: %v\n", err)
	}
}

func waitForStackyardReady(ctx context.Context, apiEndpoint, sessionName string) error {
	readyURL := strings.TrimRight(apiEndpoint, "/") + "/v1/" + sessionName
	httpClient := &http.Client{Timeout: time.Second}
	deadline := time.Now().Add(20 * time.Second)
	var lastErr error

	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, readyURL, nil)
		if err != nil {
			return err
		}
		req.Header.Set("X-Stackyard-GCP-Service", "spanner")

		resp, err := httpClient.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
			lastErr = fmt.Errorf("ready probe status %d", resp.StatusCode)
		} else {
			lastErr = err
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for %s: %w", readyURL, lastErr)
		}
		time.Sleep(300 * time.Millisecond)
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

type stackyardHeaderTransport struct {
	base        http.RoundTripper
	serviceName string
}

func (t stackyardHeaderTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	clone := req.Clone(req.Context())
	clone.Header.Set("X-Stackyard-GCP-Service", t.serviceName)
	return base.RoundTrip(clone)
}
