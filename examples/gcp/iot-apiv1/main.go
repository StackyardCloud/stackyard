package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	iampb "cloud.google.com/go/iam/apiv1/iampb"
	iot "cloud.google.com/go/iot/apiv1"
	"cloud.google.com/go/iot/apiv1/iotpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type callSpec struct {
	name string
	call func(context.Context, *iot.DeviceManagerClient) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	registryID := getenv("STACKYARD_GCP_IOT_REGISTRY_ID", "team-registry")
	deviceID := getenv("STACKYARD_GCP_IOT_DEVICE_ID", "sensor-1")
	gatewayID := getenv("STACKYARD_GCP_IOT_GATEWAY_ID", "gateway-1")

	locationName := fmt.Sprintf("projects/%s/locations/%s", projectID, locationID)
	registryName := locationName + "/registries/" + registryID
	deviceName := registryName + "/devices/" + deviceID

	fmt.Printf("Stackyard GCP Cloud IoT apiv1 client using %s\n", apiEndpoint)

	client, err := iot.NewDeviceManagerRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create iot client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "ListDeviceRegistries",
			call: func(ctx context.Context, c *iot.DeviceManagerClient) error {
				it := c.ListDeviceRegistries(ctx, &iotpb.ListDeviceRegistriesRequest{
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
			name: "GetDeviceRegistry",
			call: func(ctx context.Context, c *iot.DeviceManagerClient) error {
				_, err := c.GetDeviceRegistry(ctx, &iotpb.GetDeviceRegistryRequest{Name: registryName})
				return err
			},
		},
		{
			name: "CreateDeviceRegistry",
			call: func(ctx context.Context, c *iot.DeviceManagerClient) error {
				_, err := c.CreateDeviceRegistry(ctx, &iotpb.CreateDeviceRegistryRequest{
					Parent: locationName,
					DeviceRegistry: &iotpb.DeviceRegistry{
						Name: registryName,
						Id:   registryID,
					},
				})
				return err
			},
		},
		{
			name: "UpdateDeviceRegistry",
			call: func(ctx context.Context, c *iot.DeviceManagerClient) error {
				_, err := c.UpdateDeviceRegistry(ctx, &iotpb.UpdateDeviceRegistryRequest{
					DeviceRegistry: &iotpb.DeviceRegistry{
						Name:     registryName,
						LogLevel: iotpb.LogLevel_DEBUG,
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"log_level"}},
				})
				return err
			},
		},
		{
			name: "DeleteDeviceRegistry",
			call: func(ctx context.Context, c *iot.DeviceManagerClient) error {
				return c.DeleteDeviceRegistry(ctx, &iotpb.DeleteDeviceRegistryRequest{Name: registryName})
			},
		},
		{
			name: "ListDevices",
			call: func(ctx context.Context, c *iot.DeviceManagerClient) error {
				it := c.ListDevices(ctx, &iotpb.ListDevicesRequest{
					Parent:   registryName,
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
			name: "GetDevice",
			call: func(ctx context.Context, c *iot.DeviceManagerClient) error {
				_, err := c.GetDevice(ctx, &iotpb.GetDeviceRequest{Name: deviceName})
				return err
			},
		},
		{
			name: "CreateDevice",
			call: func(ctx context.Context, c *iot.DeviceManagerClient) error {
				_, err := c.CreateDevice(ctx, &iotpb.CreateDeviceRequest{
					Parent: registryName,
					Device: &iotpb.Device{
						Id: deviceID,
					},
				})
				return err
			},
		},
		{
			name: "UpdateDevice",
			call: func(ctx context.Context, c *iot.DeviceManagerClient) error {
				_, err := c.UpdateDevice(ctx, &iotpb.UpdateDeviceRequest{
					Device: &iotpb.Device{
						Name:    deviceName,
						Blocked: true,
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"blocked"}},
				})
				return err
			},
		},
		{
			name: "DeleteDevice",
			call: func(ctx context.Context, c *iot.DeviceManagerClient) error {
				return c.DeleteDevice(ctx, &iotpb.DeleteDeviceRequest{Name: deviceName})
			},
		},
		{
			name: "ModifyCloudToDeviceConfig",
			call: func(ctx context.Context, c *iot.DeviceManagerClient) error {
				_, err := c.ModifyCloudToDeviceConfig(ctx, &iotpb.ModifyCloudToDeviceConfigRequest{
					Name:       deviceName,
					BinaryData: []byte(`{"desired":"enabled"}`),
				})
				return err
			},
		},
		{
			name: "ListDeviceConfigVersions",
			call: func(ctx context.Context, c *iot.DeviceManagerClient) error {
				_, err := c.ListDeviceConfigVersions(ctx, &iotpb.ListDeviceConfigVersionsRequest{
					Name:        deviceName,
					NumVersions: 1,
				})
				return err
			},
		},
		{
			name: "ListDeviceStates",
			call: func(ctx context.Context, c *iot.DeviceManagerClient) error {
				_, err := c.ListDeviceStates(ctx, &iotpb.ListDeviceStatesRequest{
					Name:      deviceName,
					NumStates: 1,
				})
				return err
			},
		},
		{
			name: "SetIamPolicy",
			call: func(ctx context.Context, c *iot.DeviceManagerClient) error {
				_, err := c.SetIamPolicy(ctx, &iampb.SetIamPolicyRequest{
					Resource: registryName,
					Policy:   &iampb.Policy{},
				})
				return err
			},
		},
		{
			name: "GetIamPolicy",
			call: func(ctx context.Context, c *iot.DeviceManagerClient) error {
				_, err := c.GetIamPolicy(ctx, &iampb.GetIamPolicyRequest{Resource: registryName})
				return err
			},
		},
		{
			name: "TestIamPermissions",
			call: func(ctx context.Context, c *iot.DeviceManagerClient) error {
				_, err := c.TestIamPermissions(ctx, &iampb.TestIamPermissionsRequest{
					Resource:    registryName,
					Permissions: []string{"cloudiot.registries.get"},
				})
				return err
			},
		},
		{
			name: "SendCommandToDevice",
			call: func(ctx context.Context, c *iot.DeviceManagerClient) error {
				_, err := c.SendCommandToDevice(ctx, &iotpb.SendCommandToDeviceRequest{
					Name:       deviceName,
					BinaryData: []byte(`{"command":"ping"}`),
				})
				return err
			},
		},
		{
			name: "BindDeviceToGateway",
			call: func(ctx context.Context, c *iot.DeviceManagerClient) error {
				_, err := c.BindDeviceToGateway(ctx, &iotpb.BindDeviceToGatewayRequest{
					Parent:    registryName,
					GatewayId: gatewayID,
					DeviceId:  deviceID,
				})
				return err
			},
		},
		{
			name: "UnbindDeviceFromGateway",
			call: func(ctx context.Context, c *iot.DeviceManagerClient) error {
				_, err := c.UnbindDeviceFromGateway(ctx, &iotpb.UnbindDeviceFromGatewayRequest{
					Parent:    registryName,
					GatewayId: gatewayID,
					DeviceId:  deviceID,
				})
				return err
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
		fmt.Fprintf(os.Stderr, "warning: close iot client: %v\n", err)
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
