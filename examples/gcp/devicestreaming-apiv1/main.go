package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	devicestreaming "cloud.google.com/go/devicestreaming/apiv1"
	"cloud.google.com/go/devicestreaming/apiv1/devicestreamingpb"
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
	sessionID := getenv("STACKYARD_GCP_DEVICESTREAMING_SESSION_ID", "session-1")

	projectName := "projects/" + projectID
	sessionName := projectName + "/deviceSessions/" + sessionID

	fmt.Printf("Stackyard GCP Device Streaming apiv1 client using %s\n", apiEndpoint)

	client, err := devicestreaming.NewDirectAccessRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create device streaming client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "ListDeviceSessions",
			call: func(ctx context.Context) error {
				it := client.ListDeviceSessions(ctx, &devicestreamingpb.ListDeviceSessionsRequest{
					Parent:   projectName,
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
			name: "GetDeviceSession",
			call: func(ctx context.Context) error {
				_, err := client.GetDeviceSession(ctx, &devicestreamingpb.GetDeviceSessionRequest{
					Name: sessionName,
				})
				return err
			},
		},
		{
			name: "CreateDeviceSession",
			call: func(ctx context.Context) error {
				_, err := client.CreateDeviceSession(ctx, &devicestreamingpb.CreateDeviceSessionRequest{
					Parent:          projectName,
					DeviceSessionId: sessionID,
					DeviceSession:   newDeviceSession(sessionName),
				})
				return err
			},
		},
		{
			name: "UpdateDeviceSession",
			call: func(ctx context.Context) error {
				_, err := client.UpdateDeviceSession(ctx, &devicestreamingpb.UpdateDeviceSessionRequest{
					DeviceSession: newDeviceSession(sessionName),
					UpdateMask:    &fieldmaskpb.FieldMask{Paths: []string{"ttl"}},
				})
				return err
			},
		},
		{
			name: "CancelDeviceSession",
			call: func(ctx context.Context) error {
				return client.CancelDeviceSession(ctx, &devicestreamingpb.CancelDeviceSessionRequest{
					Name: sessionName,
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

	fmt.Println("AdbConnect is not invoked in this example because the SDK REST transport does not support it.")
	fmt.Println("Done.")
}

func newDeviceSession(name string) *devicestreamingpb.DeviceSession {
	return &devicestreamingpb.DeviceSession{
		Name: name,
		AndroidDevice: &devicestreamingpb.AndroidDevice{
			AndroidModelId:   "pixel_7",
			AndroidVersionId: "android_34",
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

func closeClient(closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close client: %v\n", err)
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
