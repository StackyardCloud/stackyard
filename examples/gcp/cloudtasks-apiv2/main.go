package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	cloudtasks "cloud.google.com/go/cloudtasks/apiv2"
	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type callSpec struct {
	name string
	call func(context.Context, *cloudtasks.Client) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	queueID := getenv("STACKYARD_GCP_TASKS_QUEUE_ID", "team-queue")
	taskID := getenv("STACKYARD_GCP_TASKS_TASK_ID", "task-1")

	locationName := fmt.Sprintf("projects/%s/locations/%s", projectID, locationID)
	queueName := locationName + "/queues/" + queueID
	taskName := queueName + "/tasks/" + taskID

	fmt.Printf("Stackyard GCP Cloud Tasks apiv2 client using %s\n", apiEndpoint)

	client, err := cloudtasks.NewRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create cloudtasks client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "ListQueues",
			call: func(ctx context.Context, c *cloudtasks.Client) error {
				it := c.ListQueues(ctx, &cloudtaskspb.ListQueuesRequest{
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
			name: "GetQueue",
			call: func(ctx context.Context, c *cloudtasks.Client) error {
				_, err := c.GetQueue(ctx, &cloudtaskspb.GetQueueRequest{Name: queueName})
				return err
			},
		},
		{
			name: "CreateQueue",
			call: func(ctx context.Context, c *cloudtasks.Client) error {
				_, err := c.CreateQueue(ctx, &cloudtaskspb.CreateQueueRequest{
					Parent: locationName,
					Queue:  &cloudtaskspb.Queue{Name: queueName},
				})
				return err
			},
		},
		{
			name: "UpdateQueue",
			call: func(ctx context.Context, c *cloudtasks.Client) error {
				_, err := c.UpdateQueue(ctx, &cloudtaskspb.UpdateQueueRequest{
					Queue: &cloudtaskspb.Queue{Name: queueName},
				})
				return err
			},
		},
		{
			name: "PauseQueue",
			call: func(ctx context.Context, c *cloudtasks.Client) error {
				_, err := c.PauseQueue(ctx, &cloudtaskspb.PauseQueueRequest{Name: queueName})
				return err
			},
		},
		{
			name: "ResumeQueue",
			call: func(ctx context.Context, c *cloudtasks.Client) error {
				_, err := c.ResumeQueue(ctx, &cloudtaskspb.ResumeQueueRequest{Name: queueName})
				return err
			},
		},
		{
			name: "PurgeQueue",
			call: func(ctx context.Context, c *cloudtasks.Client) error {
				_, err := c.PurgeQueue(ctx, &cloudtaskspb.PurgeQueueRequest{Name: queueName})
				return err
			},
		},
		{
			name: "ListTasks",
			call: func(ctx context.Context, c *cloudtasks.Client) error {
				it := c.ListTasks(ctx, &cloudtaskspb.ListTasksRequest{
					Parent:   queueName,
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
			name: "GetTask",
			call: func(ctx context.Context, c *cloudtasks.Client) error {
				_, err := c.GetTask(ctx, &cloudtaskspb.GetTaskRequest{
					Name:         taskName,
					ResponseView: cloudtaskspb.Task_BASIC,
				})
				return err
			},
		},
		{
			name: "CreateTask",
			call: func(ctx context.Context, c *cloudtasks.Client) error {
				_, err := c.CreateTask(ctx, &cloudtaskspb.CreateTaskRequest{
					Parent: queueName,
					Task: &cloudtaskspb.Task{
						Name: taskName,
						MessageType: &cloudtaskspb.Task_HttpRequest{
							HttpRequest: &cloudtaskspb.HttpRequest{
								Url:        "https://example.com/stackyard/tasks",
								HttpMethod: cloudtaskspb.HttpMethod_POST,
								Headers: map[string]string{
									"content-type": "application/json",
								},
								Body: []byte(`{"event":"created"}`),
							},
						},
					},
					ResponseView: cloudtaskspb.Task_BASIC,
				})
				return err
			},
		},
		{
			name: "RunTask",
			call: func(ctx context.Context, c *cloudtasks.Client) error {
				_, err := c.RunTask(ctx, &cloudtaskspb.RunTaskRequest{
					Name:         taskName,
					ResponseView: cloudtaskspb.Task_BASIC,
				})
				return err
			},
		},
		{
			name: "DeleteTask",
			call: func(ctx context.Context, c *cloudtasks.Client) error {
				return c.DeleteTask(ctx, &cloudtaskspb.DeleteTaskRequest{Name: taskName})
			},
		},
		{
			name: "DeleteQueue",
			call: func(ctx context.Context, c *cloudtasks.Client) error {
				return c.DeleteQueue(ctx, &cloudtaskspb.DeleteQueueRequest{Name: queueName})
			},
		},
	}

	for _, call := range calls {
		err := call.call(ctx, client)
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

func closeClient(closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close cloudtasks client: %v\n", err)
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
