package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	database "cloud.google.com/go/spanner/admin/database/apiv1"
	"cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	iampb "google.golang.org/genproto/googleapis/iam/v1"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type callSpec struct {
	name string
	call func(context.Context, *database.DatabaseAdminClient) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	instanceID := getenv("STACKYARD_GCP_SPANNER_INSTANCE", "stackyard-instance")
	databaseID := getenv("STACKYARD_GCP_SPANNER_DATABASE", "stackyard-db")
	backupID := getenv("STACKYARD_GCP_SPANNER_BACKUP", "backup-1")
	scheduleID := getenv("STACKYARD_GCP_SPANNER_BACKUP_SCHEDULE", "daily-full")
	location := getenv("STACKYARD_GCP_LOCATION", "us-central1")

	instanceName := fmt.Sprintf("projects/%s/instances/%s", projectID, instanceID)
	databaseName := fmt.Sprintf("%s/databases/%s", instanceName, databaseID)
	backupName := fmt.Sprintf("%s/backups/%s", instanceName, backupID)
	scheduleName := fmt.Sprintf("%s/backupSchedules/%s", databaseName, scheduleID)

	fmt.Printf("Stackyard GCP Spanner Admin Database apiv1 client using %s\n", apiEndpoint)
	if err := waitForStackyardReady(ctx, apiEndpoint, projectID, location); err != nil {
		exitf("stackyard readiness check failed: %v", err)
	}

	httpClient := &http.Client{
		Transport: stackyardHeaderTransport{
			base:        http.DefaultTransport,
			serviceName: "spanner-admin-database",
		},
	}

	client, err := database.NewDatabaseAdminRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create spanner admin database client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "ListDatabases",
			call: func(ctx context.Context, c *database.DatabaseAdminClient) error {
				it := c.ListDatabases(ctx, &databasepb.ListDatabasesRequest{Parent: instanceName, PageSize: 1})
				_, err := it.Next()
				if err == iterator.Done {
					return nil
				}
				return err
			},
		},
		{
			name: "CreateDatabaseAndGetOperation",
			call: func(ctx context.Context, c *database.DatabaseAdminClient) error {
				op, err := c.CreateDatabase(ctx, &databasepb.CreateDatabaseRequest{
					Parent:          instanceName,
					CreateStatement: fmt.Sprintf("CREATE DATABASE `%s`", databaseID),
				})
				if err != nil {
					return err
				}
				if strings.TrimSpace(op.Name()) == "" {
					return fmt.Errorf("create database returned empty operation name")
				}
				_, err = c.GetOperation(ctx, &longrunningpb.GetOperationRequest{Name: op.Name()})
				return err
			},
		},
		{
			name: "GetDatabase",
			call: func(ctx context.Context, c *database.DatabaseAdminClient) error {
				_, err := c.GetDatabase(ctx, &databasepb.GetDatabaseRequest{Name: databaseName})
				return err
			},
		},
		{
			name: "GetDatabaseDdl",
			call: func(ctx context.Context, c *database.DatabaseAdminClient) error {
				_, err := c.GetDatabaseDdl(ctx, &databasepb.GetDatabaseDdlRequest{Database: databaseName})
				return err
			},
		},
		{
			name: "CreateBackupAndGetOperation",
			call: func(ctx context.Context, c *database.DatabaseAdminClient) error {
				expireTime := timestamppb.New(time.Now().UTC().Add(24 * time.Hour))
				op, err := c.CreateBackup(ctx, &databasepb.CreateBackupRequest{
					Parent:   instanceName,
					BackupId: backupID,
					Backup: &databasepb.Backup{
						Database:   databaseName,
						ExpireTime: expireTime,
					},
				})
				if err != nil {
					return err
				}
				if strings.TrimSpace(op.Name()) == "" {
					return fmt.Errorf("create backup returned empty operation name")
				}
				_, err = c.GetOperation(ctx, &longrunningpb.GetOperationRequest{Name: op.Name()})
				return err
			},
		},
		{
			name: "ListBackups",
			call: func(ctx context.Context, c *database.DatabaseAdminClient) error {
				it := c.ListBackups(ctx, &databasepb.ListBackupsRequest{Parent: instanceName, PageSize: 1})
				_, err := it.Next()
				if err == iterator.Done {
					return nil
				}
				return err
			},
		},
		{
			name: "CreateBackupSchedule",
			call: func(ctx context.Context, c *database.DatabaseAdminClient) error {
				_, err := c.CreateBackupSchedule(ctx, &databasepb.CreateBackupScheduleRequest{
					Parent:           databaseName,
					BackupScheduleId: scheduleID,
					BackupSchedule: &databasepb.BackupSchedule{
						RetentionDuration: durationpb.New(48 * time.Hour),
						Spec: &databasepb.BackupScheduleSpec{
							ScheduleSpec: &databasepb.BackupScheduleSpec_CronSpec{
								CronSpec: &databasepb.CrontabSpec{Text: "0 */6 * * *"},
							},
						},
						BackupTypeSpec: &databasepb.BackupSchedule_FullBackupSpec{
							FullBackupSpec: &databasepb.FullBackupSpec{},
						},
					},
				})
				return err
			},
		},
		{
			name: "ListBackupSchedules",
			call: func(ctx context.Context, c *database.DatabaseAdminClient) error {
				it := c.ListBackupSchedules(ctx, &databasepb.ListBackupSchedulesRequest{Parent: databaseName, PageSize: 1})
				_, err := it.Next()
				if err == iterator.Done {
					return nil
				}
				return err
			},
		},
		{
			name: "TestIamPermissions",
			call: func(ctx context.Context, c *database.DatabaseAdminClient) error {
				_, err := c.TestIamPermissions(ctx, &iampb.TestIamPermissionsRequest{
					Resource:    databaseName,
					Permissions: []string{"spanner.databases.get", "resourcemanager.projects.get"},
				})
				return err
			},
		},
		{
			name: "GetBackupSchedule",
			call: func(ctx context.Context, c *database.DatabaseAdminClient) error {
				_, err := c.GetBackupSchedule(ctx, &databasepb.GetBackupScheduleRequest{Name: scheduleName})
				return err
			},
		},
		{
			name: "GetBackup",
			call: func(ctx context.Context, c *database.DatabaseAdminClient) error {
				_, err := c.GetBackup(ctx, &databasepb.GetBackupRequest{Name: backupName})
				return err
			},
		},
		{
			name: "DropDatabase",
			call: func(ctx context.Context, c *database.DatabaseAdminClient) error {
				return c.DropDatabase(ctx, &databasepb.DropDatabaseRequest{Database: databaseName})
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
		fmt.Fprintf(os.Stderr, "warning: close spanner admin database client: %v\n", err)
	}
}

func waitForStackyardReady(ctx context.Context, apiEndpoint, projectID, location string) error {
	readyURL := fmt.Sprintf("%s/v1/projects/%s/locations/%s/spanner_admin_database?stackyard_contract_probe=1&typedSuccess=1", strings.TrimRight(apiEndpoint, "/"), projectID, location)
	httpClient := &http.Client{Timeout: time.Second}
	deadline := time.Now().Add(20 * time.Second)
	var lastErr error

	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, readyURL, nil)
		if err != nil {
			return err
		}
		req.Header.Set("X-Stackyard-GCP-Service", "spanner-admin-database")

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
