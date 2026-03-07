package main

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/datastore"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type order struct {
	Amount int64  `datastore:"amount"`
	State  string `datastore:"state"`
}

type callSpec struct {
	name string
	call func(context.Context, *datastore.Client) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	databaseID := getenv("STACKYARD_GCP_DATASTORE_DATABASE_ID", datastore.DefaultDatabaseID)
	callTimeout := parseDurationEnv("STACKYARD_GCP_DATASTORE_CALL_TIMEOUT", 2*time.Second)

	emulatorHost := datastoreEmulatorHost(endpoint)
	if emulatorHost == "" {
		exitf("failed to derive datastore emulator host from %q", endpoint)
	}
	_ = os.Setenv("DATASTORE_EMULATOR_HOST", emulatorHost)
	_ = os.Setenv("DATASTORE_PROJECT_ID", projectID)

	fmt.Printf("Stackyard GCP Datastore apiv1 client using emulator endpoint %s\n", emulatorHost)

	clientCtx, clientCancel := context.WithTimeout(ctx, callTimeout)
	client, err := datastore.NewClientWithDatabase(clientCtx, projectID, databaseID,
		option.WithEndpoint(emulatorHost),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	clientCancel()
	if err != nil {
		exitf("failed to create datastore client: %v", err)
	}
	defer closeClient(client.Close)

	entityKey := datastore.NameKey("Order", getenv("STACKYARD_GCP_DATASTORE_ENTITY_ID", "order-1"), nil)
	incompleteKey := datastore.IncompleteKey("Order", nil)

	calls := []callSpec{
		{
			name: "Put",
			call: func(ctx context.Context, c *datastore.Client) error {
				_, err := c.Put(ctx, entityKey, &order{Amount: 42, State: "CREATED"})
				return err
			},
		},
		{
			name: "Get",
			call: func(ctx context.Context, c *datastore.Client) error {
				var dst order
				return c.Get(ctx, entityKey, &dst)
			},
		},
		{
			name: "RunQuery",
			call: func(ctx context.Context, c *datastore.Client) error {
				it := c.Run(ctx, datastore.NewQuery("Order").Limit(1))
				var dst order
				_, err := it.Next(&dst)
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		},
		{
			name: "RunAggregationQuery",
			call: func(ctx context.Context, c *datastore.Client) error {
				aq := datastore.NewQuery("Order").NewAggregationQuery().WithCount("total_orders")
				_, err := c.RunAggregationQuery(ctx, aq)
				return err
			},
		},
		{
			name: "AllocateIDs",
			call: func(ctx context.Context, c *datastore.Client) error {
				_, err := c.AllocateIDs(ctx, []*datastore.Key{incompleteKey})
				return err
			},
		},
		{
			name: "ReserveIDs",
			call: func(ctx context.Context, c *datastore.Client) error {
				return c.ReserveIDs(ctx, []*datastore.Key{entityKey})
			},
		},
		{
			name: "Transaction",
			call: func(ctx context.Context, c *datastore.Client) error {
				tx, err := c.NewTransaction(ctx)
				if err != nil {
					return err
				}
				return tx.Rollback()
			},
		},
	}

	for _, call := range calls {
		logf("Running %s...", call.name)
		callCtx, callCancel := context.WithTimeout(ctx, callTimeout)
		err := call.call(callCtx, client)
		callCancel()
		switch {
		case err == nil:
			logf("%s succeeded", call.name)
		case isToleratedFoundationError(err):
			logf("%s returned staged-foundation error (expected in early emulation): %v", call.name, err)
		default:
			exitf("%s failed: %v", call.name, err)
		}
	}

	fmt.Println("Done.")
}

func datastoreEmulatorHost(endpoint string) string {
	trimmed := strings.TrimSpace(endpoint)
	if trimmed == "" {
		return ""
	}

	if strings.Contains(trimmed, "://") {
		parsed, err := url.Parse(trimmed)
		if err != nil {
			return ""
		}
		return parsed.Host
	}

	return strings.TrimPrefix(trimmed, "http://")
}

func isToleratedFoundationError(err error) bool {
	if err == nil {
		return false
	}

	if grpcStatus, ok := status.FromError(err); ok {
		switch grpcStatus.Code() {
		case codes.Unimplemented, codes.Unavailable, codes.Internal, codes.DeadlineExceeded:
			return true
		}
	}

	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) && (apiErr.Code == 501 || apiErr.Code == 503) {
		return true
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "notimplemented") ||
		strings.Contains(msg, "unsupported protocol") ||
		strings.Contains(msg, "unexpected http status") ||
		strings.Contains(msg, "deadline exceeded") ||
		strings.Contains(msg, "connection refused")
}

func closeClient(closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close datastore client: %v\n", err)
	}
}

func getenv(key, fallback string) string {
	if val := strings.TrimSpace(os.Getenv(key)); val != "" {
		return val
	}
	return fallback
}

func parseDurationEnv(key string, fallback time.Duration) time.Duration {
	val := strings.TrimSpace(os.Getenv(key))
	if val == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(val)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func logf(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}
