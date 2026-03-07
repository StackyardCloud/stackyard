package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	delivery "cloud.google.com/go/maps/fleetengine/delivery/apiv1"
	"cloud.google.com/go/maps/fleetengine/delivery/apiv1/deliverypb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	latlngpb "google.golang.org/genproto/googleapis/type/latlng"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
)

type callSpec struct {
	name string
	call func(context.Context, *delivery.Client) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	providerID := getenv("STACKYARD_GCP_FLEETENGINE_DELIVERY_PROVIDER_ID", "stackyard")
	deliveryVehicleID := getenv("STACKYARD_GCP_FLEETENGINE_DELIVERY_VEHICLE_ID", "delivery-vehicle-1")
	taskID := getenv("STACKYARD_GCP_FLEETENGINE_DELIVERY_TASK_ID", "task-1")
	batchTaskID := getenv("STACKYARD_GCP_FLEETENGINE_DELIVERY_BATCH_TASK_ID", "task-batch-1")
	trackingID := getenv("STACKYARD_GCP_FLEETENGINE_DELIVERY_TRACKING_ID", "tracking-1")
	batchTrackingID := getenv("STACKYARD_GCP_FLEETENGINE_DELIVERY_BATCH_TRACKING_ID", "tracking-batch-1")

	parent := fmt.Sprintf("providers/%s", providerID)
	deliveryVehicleName := fmt.Sprintf("%s/deliveryVehicles/%s", parent, deliveryVehicleID)
	taskName := fmt.Sprintf("%s/tasks/%s", parent, taskID)
	taskTrackingInfoName := fmt.Sprintf("%s/taskTrackingInfo/%s", parent, trackingID)

	fmt.Printf("Stackyard GCP Last Mile Fleet Delivery Solution apiv1 client using %s\n", apiEndpoint)

	client, err := delivery.NewRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create fleetengine delivery client: %v", err)
	}
	defer closeClient("fleetengine delivery", client.Close)

	calls := []callSpec{
		{
			name: "CreateDeliveryVehicle",
			call: func(ctx context.Context, c *delivery.Client) error {
				_, err := c.CreateDeliveryVehicle(ctx, &deliverypb.CreateDeliveryVehicleRequest{
					Parent:            parent,
					DeliveryVehicleId: deliveryVehicleID,
					DeliveryVehicle: &deliverypb.DeliveryVehicle{
						Type: deliverypb.DeliveryVehicle_AUTO,
					},
				})
				return err
			},
		},
		{
			name: "GetDeliveryVehicle",
			call: func(ctx context.Context, c *delivery.Client) error {
				_, err := c.GetDeliveryVehicle(ctx, &deliverypb.GetDeliveryVehicleRequest{Name: deliveryVehicleName})
				return err
			},
		},
		{
			name: "ListDeliveryVehicles",
			call: func(ctx context.Context, c *delivery.Client) error {
				it := c.ListDeliveryVehicles(ctx, &deliverypb.ListDeliveryVehiclesRequest{
					Parent:   parent,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "CreateTask",
			call: func(ctx context.Context, c *delivery.Client) error {
				_, err := c.CreateTask(ctx, &deliverypb.CreateTaskRequest{
					Parent: parent,
					TaskId: taskID,
					Task:   buildTask(trackingID),
				})
				return err
			},
		},
		{
			name: "BatchCreateTasks",
			call: func(ctx context.Context, c *delivery.Client) error {
				_, err := c.BatchCreateTasks(ctx, &deliverypb.BatchCreateTasksRequest{
					Parent: parent,
					Requests: []*deliverypb.CreateTaskRequest{
						{
							TaskId: batchTaskID,
							Task:   buildTask(batchTrackingID),
						},
					},
				})
				return err
			},
		},
		{
			name: "GetTask",
			call: func(ctx context.Context, c *delivery.Client) error {
				_, err := c.GetTask(ctx, &deliverypb.GetTaskRequest{Name: taskName})
				return err
			},
		},
		{
			name: "ListTasks",
			call: func(ctx context.Context, c *delivery.Client) error {
				it := c.ListTasks(ctx, &deliverypb.ListTasksRequest{
					Parent:   parent,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetTaskTrackingInfo",
			call: func(ctx context.Context, c *delivery.Client) error {
				_, err := c.GetTaskTrackingInfo(ctx, &deliverypb.GetTaskTrackingInfoRequest{
					Name: taskTrackingInfoName,
				})
				return err
			},
		},
		{
			name: "DeleteTask",
			call: func(ctx context.Context, c *delivery.Client) error {
				return c.DeleteTask(ctx, &deliverypb.DeleteTaskRequest{Name: taskName})
			},
		},
		{
			name: "DeleteDeliveryVehicle",
			call: func(ctx context.Context, c *delivery.Client) error {
				return c.DeleteDeliveryVehicle(ctx, &deliverypb.DeleteDeliveryVehicleRequest{Name: deliveryVehicleName})
			},
		},
	}

	for _, c := range calls {
		err := c.call(ctx, client)
		switch {
		case err == nil:
			logf("%s succeeded", c.name)
		case isToleratedNotImplemented(err):
			logf("%s returned NotImplemented (expected in staged emulation)", c.name)
		default:
			exitf("%s failed: %v", c.name, err)
		}
	}

	fmt.Println("Done.")
}

func buildTask(trackingID string) *deliverypb.Task {
	return &deliverypb.Task{
		Type:       deliverypb.Task_PICKUP,
		State:      deliverypb.Task_OPEN,
		TrackingId: trackingID,
		PlannedLocation: &deliverypb.LocationInfo{
			Point: &latlngpb.LatLng{
				Latitude:  37.7749,
				Longitude: -122.4194,
			},
		},
		TaskDuration: durationpb.New(5 * time.Minute),
	}
}

func iteratorResult(err error) error {
	if errors.Is(err, iterator.Done) {
		return nil
	}
	return err
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
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
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
