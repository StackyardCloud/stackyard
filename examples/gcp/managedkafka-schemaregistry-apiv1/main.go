package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	schemaregistry "cloud.google.com/go/managedkafka/schemaregistry/apiv1"
	"cloud.google.com/go/managedkafka/schemaregistry/apiv1/schemaregistrypb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	locationpb "google.golang.org/genproto/googleapis/cloud/location"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type callSpec struct {
	name string
	call func(context.Context, *schemaregistry.ManagedSchemaRegistryClient) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	registryID := getenv("STACKYARD_GCP_MANAGEDKAFKA_SCHEMAREGISTRY_ID", "sr1")
	contextID := getenv("STACKYARD_GCP_MANAGEDKAFKA_SCHEMAREGISTRY_CONTEXT", "default")
	subjectID := getenv("STACKYARD_GCP_MANAGEDKAFKA_SCHEMAREGISTRY_SUBJECT", "orders-value")
	versionID := getenv("STACKYARD_GCP_MANAGEDKAFKA_SCHEMAREGISTRY_VERSION", "latest")
	schemaID := getenv("STACKYARD_GCP_MANAGEDKAFKA_SCHEMAREGISTRY_SCHEMA_ID", "1")
	operationID := getenv("STACKYARD_GCP_MANAGEDKAFKA_SCHEMAREGISTRY_OPERATION_ID", "op-1")

	projectName := fmt.Sprintf("projects/%s", projectID)
	locationName := fmt.Sprintf("%s/locations/%s", projectName, locationID)
	registryName := fmt.Sprintf("%s/schemaRegistries/%s", locationName, registryID)
	contextName := fmt.Sprintf("%s/contexts/%s", registryName, contextID)
	schemaName := fmt.Sprintf("%s/schemas/ids/%s", registryName, schemaID)
	subjectName := fmt.Sprintf("%s/subjects/%s", registryName, subjectID)
	versionName := fmt.Sprintf("%s/versions/%s", subjectName, versionID)
	configName := fmt.Sprintf("%s/config", registryName)
	subjectConfigName := fmt.Sprintf("%s/config/%s", registryName, subjectID)
	modeName := fmt.Sprintf("%s/mode/%s", registryName, subjectID)
	compatibilityName := fmt.Sprintf("%s/compatibility/subjects/%s/versions/%s", registryName, subjectID, versionID)
	operationName := fmt.Sprintf("%s/operations/%s", locationName, operationID)

	fmt.Printf("Stackyard GCP Managed Kafka Schema Registry apiv1 client using %s\n", apiEndpoint)

	client, err := schemaregistry.NewManagedSchemaRegistryRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create managedkafka schema registry client: %v", err)
	}
	defer closeClient("managedkafka schemaregistry", client.Close)

	avroType := schemaregistrypb.Schema_AVRO
	compatibility := schemaregistrypb.SchemaConfig_BACKWARD
	mode := schemaregistrypb.SchemaMode_READWRITE
	trueVal := true
	falseVal := false
	schemaPayload := `{"type":"record","name":"Order","fields":[{"name":"id","type":"string"}]}`

	calls := []callSpec{
		{
			name: "ListSchemaRegistries",
			call: func(ctx context.Context, c *schemaregistry.ManagedSchemaRegistryClient) error {
				_, err := c.ListSchemaRegistries(ctx, &schemaregistrypb.ListSchemaRegistriesRequest{Parent: locationName})
				return err
			},
		},
		{
			name: "GetSchemaRegistry",
			call: func(ctx context.Context, c *schemaregistry.ManagedSchemaRegistryClient) error {
				_, err := c.GetSchemaRegistry(ctx, &schemaregistrypb.GetSchemaRegistryRequest{Name: registryName})
				return err
			},
		},
		{
			name: "CreateSchemaRegistry",
			call: func(ctx context.Context, c *schemaregistry.ManagedSchemaRegistryClient) error {
				_, err := c.CreateSchemaRegistry(ctx, &schemaregistrypb.CreateSchemaRegistryRequest{
					Parent:           locationName,
					SchemaRegistryId: registryID,
					SchemaRegistry: &schemaregistrypb.SchemaRegistry{
						Name: registryName,
					},
				})
				return err
			},
		},
		{
			name: "DeleteSchemaRegistry",
			call: func(ctx context.Context, c *schemaregistry.ManagedSchemaRegistryClient) error {
				return c.DeleteSchemaRegistry(ctx, &schemaregistrypb.DeleteSchemaRegistryRequest{Name: registryName})
			},
		},
		{
			name: "GetContext",
			call: func(ctx context.Context, c *schemaregistry.ManagedSchemaRegistryClient) error {
				_, err := c.GetContext(ctx, &schemaregistrypb.GetContextRequest{Name: contextName})
				return err
			},
		},
		{
			name: "ListContexts",
			call: func(ctx context.Context, c *schemaregistry.ManagedSchemaRegistryClient) error {
				_, err := c.ListContexts(ctx, &schemaregistrypb.ListContextsRequest{Parent: registryName})
				return err
			},
		},
		{
			name: "GetSchema",
			call: func(ctx context.Context, c *schemaregistry.ManagedSchemaRegistryClient) error {
				_, err := c.GetSchema(ctx, &schemaregistrypb.GetSchemaRequest{Name: schemaName})
				return err
			},
		},
		{
			name: "GetRawSchema",
			call: func(ctx context.Context, c *schemaregistry.ManagedSchemaRegistryClient) error {
				_, err := c.GetRawSchema(ctx, &schemaregistrypb.GetSchemaRequest{Name: schemaName})
				return err
			},
		},
		{
			name: "ListSchemaVersions",
			call: func(ctx context.Context, c *schemaregistry.ManagedSchemaRegistryClient) error {
				_, err := c.ListSchemaVersions(ctx, &schemaregistrypb.ListSchemaVersionsRequest{Parent: schemaName})
				return err
			},
		},
		{
			name: "ListSchemaTypes",
			call: func(ctx context.Context, c *schemaregistry.ManagedSchemaRegistryClient) error {
				_, err := c.ListSchemaTypes(ctx, &schemaregistrypb.ListSchemaTypesRequest{Parent: registryName})
				return err
			},
		},
		{
			name: "ListSubjects",
			call: func(ctx context.Context, c *schemaregistry.ManagedSchemaRegistryClient) error {
				_, err := c.ListSubjects(ctx, &schemaregistrypb.ListSubjectsRequest{
					Parent:  registryName,
					Deleted: &falseVal,
				})
				return err
			},
		},
		{
			name: "ListSubjectsBySchemaId",
			call: func(ctx context.Context, c *schemaregistry.ManagedSchemaRegistryClient) error {
				_, err := c.ListSubjectsBySchemaId(ctx, &schemaregistrypb.ListSubjectsBySchemaIdRequest{
					Parent: schemaName,
				})
				return err
			},
		},
		{
			name: "DeleteSubject",
			call: func(ctx context.Context, c *schemaregistry.ManagedSchemaRegistryClient) error {
				_, err := c.DeleteSubject(ctx, &schemaregistrypb.DeleteSubjectRequest{
					Name:      subjectName,
					Permanent: &falseVal,
				})
				return err
			},
		},
		{
			name: "LookupVersion",
			call: func(ctx context.Context, c *schemaregistry.ManagedSchemaRegistryClient) error {
				_, err := c.LookupVersion(ctx, &schemaregistrypb.LookupVersionRequest{
					Parent:     subjectName,
					SchemaType: &avroType,
					Schema:     schemaPayload,
					Deleted:    &falseVal,
				})
				return err
			},
		},
		{
			name: "GetVersion",
			call: func(ctx context.Context, c *schemaregistry.ManagedSchemaRegistryClient) error {
				_, err := c.GetVersion(ctx, &schemaregistrypb.GetVersionRequest{Name: versionName})
				return err
			},
		},
		{
			name: "GetRawSchemaVersion",
			call: func(ctx context.Context, c *schemaregistry.ManagedSchemaRegistryClient) error {
				_, err := c.GetRawSchemaVersion(ctx, &schemaregistrypb.GetVersionRequest{Name: versionName})
				return err
			},
		},
		{
			name: "ListVersions",
			call: func(ctx context.Context, c *schemaregistry.ManagedSchemaRegistryClient) error {
				_, err := c.ListVersions(ctx, &schemaregistrypb.ListVersionsRequest{
					Parent:  subjectName,
					Deleted: &falseVal,
				})
				return err
			},
		},
		{
			name: "CreateVersion",
			call: func(ctx context.Context, c *schemaregistry.ManagedSchemaRegistryClient) error {
				_, err := c.CreateVersion(ctx, &schemaregistrypb.CreateVersionRequest{
					Parent:     subjectName,
					SchemaType: &avroType,
					Schema:     schemaPayload,
					Normalize:  &trueVal,
				})
				return err
			},
		},
		{
			name: "DeleteVersion",
			call: func(ctx context.Context, c *schemaregistry.ManagedSchemaRegistryClient) error {
				_, err := c.DeleteVersion(ctx, &schemaregistrypb.DeleteVersionRequest{
					Name:      versionName,
					Permanent: &falseVal,
				})
				return err
			},
		},
		{
			name: "ListReferencedSchemas",
			call: func(ctx context.Context, c *schemaregistry.ManagedSchemaRegistryClient) error {
				_, err := c.ListReferencedSchemas(ctx, &schemaregistrypb.ListReferencedSchemasRequest{Parent: versionName})
				return err
			},
		},
		{
			name: "CheckCompatibility",
			call: func(ctx context.Context, c *schemaregistry.ManagedSchemaRegistryClient) error {
				_, err := c.CheckCompatibility(ctx, &schemaregistrypb.CheckCompatibilityRequest{
					Name:       compatibilityName,
					SchemaType: &avroType,
					Schema:     schemaPayload,
					Verbose:    &trueVal,
				})
				return err
			},
		},
		{
			name: "GetSchemaConfig",
			call: func(ctx context.Context, c *schemaregistry.ManagedSchemaRegistryClient) error {
				_, err := c.GetSchemaConfig(ctx, &schemaregistrypb.GetSchemaConfigRequest{
					Name:            configName,
					DefaultToGlobal: &trueVal,
				})
				return err
			},
		},
		{
			name: "UpdateSchemaConfig",
			call: func(ctx context.Context, c *schemaregistry.ManagedSchemaRegistryClient) error {
				_, err := c.UpdateSchemaConfig(ctx, &schemaregistrypb.UpdateSchemaConfigRequest{
					Name:          subjectConfigName,
					Compatibility: &compatibility,
					Normalize:     &trueVal,
				})
				return err
			},
		},
		{
			name: "DeleteSchemaConfig",
			call: func(ctx context.Context, c *schemaregistry.ManagedSchemaRegistryClient) error {
				_, err := c.DeleteSchemaConfig(ctx, &schemaregistrypb.DeleteSchemaConfigRequest{Name: subjectConfigName})
				return err
			},
		},
		{
			name: "GetSchemaMode",
			call: func(ctx context.Context, c *schemaregistry.ManagedSchemaRegistryClient) error {
				_, err := c.GetSchemaMode(ctx, &schemaregistrypb.GetSchemaModeRequest{Name: modeName})
				return err
			},
		},
		{
			name: "UpdateSchemaMode",
			call: func(ctx context.Context, c *schemaregistry.ManagedSchemaRegistryClient) error {
				_, err := c.UpdateSchemaMode(ctx, &schemaregistrypb.UpdateSchemaModeRequest{
					Name: modeName,
					Mode: mode,
				})
				return err
			},
		},
		{
			name: "DeleteSchemaMode",
			call: func(ctx context.Context, c *schemaregistry.ManagedSchemaRegistryClient) error {
				_, err := c.DeleteSchemaMode(ctx, &schemaregistrypb.DeleteSchemaModeRequest{Name: modeName})
				return err
			},
		},
		{
			name: "GetLocation",
			call: func(ctx context.Context, c *schemaregistry.ManagedSchemaRegistryClient) error {
				_, err := c.GetLocation(ctx, &locationpb.GetLocationRequest{Name: locationName})
				return err
			},
		},
		{
			name: "ListLocations",
			call: func(ctx context.Context, c *schemaregistry.ManagedSchemaRegistryClient) error {
				it := c.ListLocations(ctx, &locationpb.ListLocationsRequest{Name: projectName})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetOperation",
			call: func(ctx context.Context, c *schemaregistry.ManagedSchemaRegistryClient) error {
				_, err := c.GetOperation(ctx, &longrunningpb.GetOperationRequest{Name: operationName})
				return err
			},
		},
		{
			name: "ListOperations",
			call: func(ctx context.Context, c *schemaregistry.ManagedSchemaRegistryClient) error {
				it := c.ListOperations(ctx, &longrunningpb.ListOperationsRequest{Name: locationName})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "CancelOperation",
			call: func(ctx context.Context, c *schemaregistry.ManagedSchemaRegistryClient) error {
				return c.CancelOperation(ctx, &longrunningpb.CancelOperationRequest{Name: operationName})
			},
		},
		{
			name: "DeleteOperation",
			call: func(ctx context.Context, c *schemaregistry.ManagedSchemaRegistryClient) error {
				return c.DeleteOperation(ctx, &longrunningpb.DeleteOperationRequest{Name: operationName})
			},
		},
	}

	for _, call := range calls {
		err := call.call(ctx, client)
		switch {
		case err == nil:
			logf("%s succeeded", call.name)
		case isToleratedFoundationError(err):
			logf("%s returned tolerated foundation error (expected in staged emulation): %v", call.name, err)
		default:
			exitf("%s failed: %v", call.name, err)
		}
	}

	fmt.Println("Done.")
}

func iteratorResult(err error) error {
	if errors.Is(err, iterator.Done) {
		return nil
	}
	return err
}

func isToleratedFoundationError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}

	if grpcStatus, ok := status.FromError(err); ok {
		switch grpcStatus.Code() {
		case codes.Unimplemented, codes.Unavailable, codes.DeadlineExceeded:
			return true
		}
	}

	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) && (apiErr.Code == 501 || apiErr.Code == 503) {
		return true
	}

	text := strings.ToLower(err.Error())
	return strings.Contains(text, "notimplemented") ||
		strings.Contains(text, "not implemented") ||
		strings.Contains(text, "context deadline exceeded") ||
		strings.Contains(text, "connection refused")
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

func logf(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
