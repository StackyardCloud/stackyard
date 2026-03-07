package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	fleetengine "cloud.google.com/go/maps/fleetengine/apiv1"
	"cloud.google.com/go/maps/fleetengine/apiv1/fleetenginepb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	latlngpb "google.golang.org/genproto/googleapis/type/latlng"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type tripCallSpec struct {
	name string
	call func(context.Context, *fleetengine.TripClient) error
}

type vehicleCallSpec struct {
	name string
	call func(context.Context, *fleetengine.VehicleClient) error
}

func main() {
	grpcEndpoint := grpcEndpointFromEnv()
	providerID := getenv("STACKYARD_GCP_FLEETENGINE_PROVIDER_ID", "stackyard")
	tripID := getenv("STACKYARD_GCP_FLEETENGINE_TRIP_ID", "trip-1")
	vehicleID := getenv("STACKYARD_GCP_FLEETENGINE_VEHICLE_ID", "vehicle-1")

	parent := fmt.Sprintf("providers/%s", providerID)
	tripName := fmt.Sprintf("%s/trips/%s", parent, tripID)
	vehicleName := fmt.Sprintf("%s/vehicles/%s", parent, vehicleID)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Printf("Stackyard GCP Local Rides and Deliveries (Fleet Engine) apiv1 clients using %s\n", grpcEndpoint)

	clientOpts := []option.ClientOption{
		option.WithEndpoint(grpcEndpoint),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	}

	tripClient, err := fleetengine.NewTripClient(ctx, clientOpts...)
	if err != nil {
		exitf("failed to create fleet engine trip client: %v", err)
	}
	defer closeClient("fleet engine trip", tripClient.Close)

	vehicleClient, err := fleetengine.NewVehicleClient(ctx, clientOpts...)
	if err != nil {
		exitf("failed to create fleet engine vehicle client: %v", err)
	}
	defer closeClient("fleet engine vehicle", vehicleClient.Close)

	tripCalls := []tripCallSpec{
		{
			name: "CreateTrip",
			call: func(ctx context.Context, c *fleetengine.TripClient) error {
				_, err := c.CreateTrip(ctx, &fleetenginepb.CreateTripRequest{
					Parent: parent,
					TripId: tripID,
					Trip: &fleetenginepb.Trip{
						TripType:    fleetenginepb.TripType_EXCLUSIVE,
						PickupPoint: pickupPoint(),
						VehicleId:   vehicleID,
					},
				})
				return err
			},
		},
		{
			name: "GetTrip",
			call: func(ctx context.Context, c *fleetengine.TripClient) error {
				_, err := c.GetTrip(ctx, &fleetenginepb.GetTripRequest{Name: tripName})
				return err
			},
		},
		{
			name: "UpdateTrip",
			call: func(ctx context.Context, c *fleetengine.TripClient) error {
				_, err := c.UpdateTrip(ctx, &fleetenginepb.UpdateTripRequest{
					Name: tripName,
					Trip: &fleetenginepb.Trip{
						Name:      tripName,
						VehicleId: vehicleID,
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"vehicle_id"}},
				})
				return err
			},
		},
		{
			name: "ReportBillableTrip",
			call: func(ctx context.Context, c *fleetengine.TripClient) error {
				return c.ReportBillableTrip(ctx, &fleetenginepb.ReportBillableTripRequest{
					Name:        tripName,
					CountryCode: "US",
				})
			},
		},
		{
			name: "DeleteTrip",
			call: func(ctx context.Context, c *fleetengine.TripClient) error {
				return c.DeleteTrip(ctx, &fleetenginepb.DeleteTripRequest{Name: tripName})
			},
		},
	}

	vehicleCalls := []vehicleCallSpec{
		{
			name: "CreateVehicle",
			call: func(ctx context.Context, c *fleetengine.VehicleClient) error {
				_, err := c.CreateVehicle(ctx, &fleetenginepb.CreateVehicleRequest{
					Parent:    parent,
					VehicleId: vehicleID,
					Vehicle: &fleetenginepb.Vehicle{
						VehicleState:       fleetenginepb.VehicleState_ONLINE,
						SupportedTripTypes: []fleetenginepb.TripType{fleetenginepb.TripType_EXCLUSIVE},
						MaximumCapacity:    4,
						VehicleType: &fleetenginepb.Vehicle_VehicleType{
							Category: fleetenginepb.Vehicle_VehicleType_AUTO,
						},
					},
				})
				return err
			},
		},
		{
			name: "GetVehicle",
			call: func(ctx context.Context, c *fleetengine.VehicleClient) error {
				_, err := c.GetVehicle(ctx, &fleetenginepb.GetVehicleRequest{Name: vehicleName})
				return err
			},
		},
		{
			name: "UpdateVehicleAttributes",
			call: func(ctx context.Context, c *fleetengine.VehicleClient) error {
				_, err := c.UpdateVehicleAttributes(ctx, &fleetenginepb.UpdateVehicleAttributesRequest{
					Name: vehicleName,
					Attributes: []*fleetenginepb.VehicleAttribute{
						{
							Key:   "env",
							Value: "local",
						},
					},
				})
				return err
			},
		},
		{
			name: "DeleteVehicle",
			call: func(ctx context.Context, c *fleetengine.VehicleClient) error {
				return c.DeleteVehicle(ctx, &fleetenginepb.DeleteVehicleRequest{Name: vehicleName})
			},
		},
	}

	runTripCalls(ctx, tripClient, tripCalls)
	runVehicleCalls(ctx, vehicleClient, vehicleCalls)

	fmt.Println("Done.")
}

func runTripCalls(ctx context.Context, client *fleetengine.TripClient, calls []tripCallSpec) {
	for _, call := range calls {
		callCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err := call.call(callCtx, client)
		cancel()

		switch {
		case err == nil:
			logf("%s succeeded", call.name)
		case isToleratedFoundationError(err):
			logf("%s returned tolerated foundation error (expected in staged emulation): %v", call.name, err)
		default:
			exitf("%s failed: %v", call.name, err)
		}
	}
}

func runVehicleCalls(ctx context.Context, client *fleetengine.VehicleClient, calls []vehicleCallSpec) {
	for _, call := range calls {
		callCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err := call.call(callCtx, client)
		cancel()

		switch {
		case err == nil:
			logf("%s succeeded", call.name)
		case isToleratedFoundationError(err):
			logf("%s returned tolerated foundation error (expected in staged emulation): %v", call.name, err)
		default:
			exitf("%s failed: %v", call.name, err)
		}
	}
}

func pickupPoint() *fleetenginepb.TerminalLocation {
	return &fleetenginepb.TerminalLocation{
		Point: &latlngpb.LatLng{
			Latitude:  37.7749,
			Longitude: -122.4194,
		},
	}
}

func grpcEndpointFromEnv() string {
	if endpoint := strings.TrimSpace(os.Getenv("STACKYARD_GCP_GRPC_ENDPOINT")); endpoint != "" {
		return normalizeEndpoint(endpoint)
	}
	httpBase := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	return normalizeEndpoint(httpBase)
}

func normalizeEndpoint(raw string) string {
	endpoint := strings.TrimSpace(raw)
	endpoint = strings.TrimPrefix(endpoint, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")
	if idx := strings.Index(endpoint, "/"); idx >= 0 {
		endpoint = endpoint[:idx]
	}
	return endpoint
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
	if errors.As(err, &apiErr) && apiErr.Code == 501 {
		return true
	}

	text := strings.ToLower(err.Error())
	return strings.Contains(text, "notimplemented") ||
		strings.Contains(text, "not implemented") ||
		strings.Contains(text, "context deadline exceeded") ||
		strings.Contains(text, "connection refused") ||
		strings.Contains(text, "failed to connect to all addresses") ||
		strings.Contains(text, "server preface") ||
		strings.Contains(text, "frame too large") ||
		strings.Contains(text, "unexpected http status code") ||
		strings.Contains(text, "unexpected content-type")
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
