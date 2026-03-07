package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	stitcher "cloud.google.com/go/video/stitcher/apiv1"
	stitcherpb "cloud.google.com/go/video/stitcher/apiv1/stitcherpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type callSpec struct {
	name string
	call func(context.Context) error
}

func main() {
	ctx := context.Background()

	endpoint := strings.TrimRight(getenv("STACKYARD_ENDPOINT", "http://localhost:4566"), "/")
	apiEndpoint := endpoint + "/gcp"
	grpcEndpoint := strings.TrimPrefix(strings.TrimPrefix(endpoint, "http://"), "https://")

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1")

	parent := fmt.Sprintf("projects/%s/locations/%s", projectID, locationID)
	cdnKeyName := parent + "/cdnKeys/cdn-key-1"
	slateName := parent + "/slates/slate-1"
	liveConfigName := parent + "/liveConfigs/live-config-1"
	vodConfigName := parent + "/vodConfigs/vod-config-1"
	vodSessionName := parent + "/vodSessions/vod-session-1"
	liveSessionName := parent + "/liveSessions/live-session-1"

	fmt.Printf("Stackyard GCP Video Stitcher video/stitcher/apiv1 client using %s (grpc=%s)\n", apiEndpoint, grpcEndpoint)
	if err := waitForStackyardReady(ctx, apiEndpoint, projectID, locationID); err != nil {
		exitf("stackyard readiness check failed: %v", err)
	}

	client, err := stitcher.NewVideoStitcherClient(ctx,
		option.WithEndpoint(grpcEndpoint),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create video stitcher client: %v", err)
	}
	defer closeClient("video stitcher", client.Close)

	operationName := parent + "/operations/createCdnKey.cdn-key-1"

	calls := []callSpec{
		{
			name: "CreateCdnKey",
			call: func(ctx context.Context) error {
				op, err := client.CreateCdnKey(ctx, &stitcherpb.CreateCdnKeyRequest{
					Parent:   parent,
					CdnKeyId: "cdn-key-1",
					CdnKey: &stitcherpb.CdnKey{
						Name: cdnKeyName,
					},
				})
				if err != nil {
					return err
				}
				if name := strings.TrimSpace(op.Name()); name != "" {
					operationName = name
				}
				return nil
			},
		},
		{
			name: "ListCdnKeys",
			call: func(ctx context.Context) error {
				it := client.ListCdnKeys(ctx, &stitcherpb.ListCdnKeysRequest{
					Parent:   parent,
					PageSize: 1,
				})
				cdnKey, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return errors.New("ListCdnKeys returned no items")
				}
				if err != nil {
					return err
				}
				if strings.TrimSpace(cdnKey.GetName()) == "" {
					return errors.New("ListCdnKeys returned cdn key without name")
				}
				return nil
			},
		},
		{
			name: "GetCdnKey",
			call: func(ctx context.Context) error {
				cdnKey, err := client.GetCdnKey(ctx, &stitcherpb.GetCdnKeyRequest{Name: cdnKeyName})
				if err != nil {
					return err
				}
				if cdnKey.GetName() != cdnKeyName {
					return fmt.Errorf("GetCdnKey returned unexpected name: %q", cdnKey.GetName())
				}
				return nil
			},
		},
		{
			name: "CreateSlate",
			call: func(ctx context.Context) error {
				op, err := client.CreateSlate(ctx, &stitcherpb.CreateSlateRequest{
					Parent:  parent,
					SlateId: "slate-1",
					Slate: &stitcherpb.Slate{
						Name: slateName,
						Uri:  "https://cdn.example.com/slates/slate-1.mp4",
					},
				})
				if err != nil {
					return err
				}
				if strings.TrimSpace(op.Name()) == "" {
					return errors.New("CreateSlate returned empty operation name")
				}
				return nil
			},
		},
		{
			name: "ListSlates",
			call: func(ctx context.Context) error {
				it := client.ListSlates(ctx, &stitcherpb.ListSlatesRequest{
					Parent:   parent,
					PageSize: 1,
				})
				slate, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return errors.New("ListSlates returned no items")
				}
				if err != nil {
					return err
				}
				if strings.TrimSpace(slate.GetName()) == "" {
					return errors.New("ListSlates returned slate without name")
				}
				return nil
			},
		},
		{
			name: "CreateLiveConfig",
			call: func(ctx context.Context) error {
				op, err := client.CreateLiveConfig(ctx, &stitcherpb.CreateLiveConfigRequest{
					Parent:       parent,
					LiveConfigId: "live-config-1",
					LiveConfig: &stitcherpb.LiveConfig{
						Name:      liveConfigName,
						SourceUri: "https://origin.example.com/live/live-config-1.m3u8",
					},
				})
				if err != nil {
					return err
				}
				if strings.TrimSpace(op.Name()) == "" {
					return errors.New("CreateLiveConfig returned empty operation name")
				}
				return nil
			},
		},
		{
			name: "ListLiveConfigs",
			call: func(ctx context.Context) error {
				it := client.ListLiveConfigs(ctx, &stitcherpb.ListLiveConfigsRequest{
					Parent:   parent,
					PageSize: 1,
				})
				liveConfig, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return errors.New("ListLiveConfigs returned no items")
				}
				if err != nil {
					return err
				}
				if strings.TrimSpace(liveConfig.GetName()) == "" {
					return errors.New("ListLiveConfigs returned live config without name")
				}
				return nil
			},
		},
		{
			name: "CreateVodConfig",
			call: func(ctx context.Context) error {
				op, err := client.CreateVodConfig(ctx, &stitcherpb.CreateVodConfigRequest{
					Parent:      parent,
					VodConfigId: "vod-config-1",
					VodConfig: &stitcherpb.VodConfig{
						Name:      vodConfigName,
						SourceUri: "https://origin.example.com/vod/vod-config-1.m3u8",
						AdTagUri:  "https://ads.example.com/vod/vod-config-1",
					},
				})
				if err != nil {
					return err
				}
				if strings.TrimSpace(op.Name()) == "" {
					return errors.New("CreateVodConfig returned empty operation name")
				}
				return nil
			},
		},
		{
			name: "ListVodConfigs",
			call: func(ctx context.Context) error {
				it := client.ListVodConfigs(ctx, &stitcherpb.ListVodConfigsRequest{
					Parent:   parent,
					PageSize: 1,
				})
				vodConfig, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return errors.New("ListVodConfigs returned no items")
				}
				if err != nil {
					return err
				}
				if strings.TrimSpace(vodConfig.GetName()) == "" {
					return errors.New("ListVodConfigs returned vod config without name")
				}
				return nil
			},
		},
		{
			name: "CreateVodSession",
			call: func(ctx context.Context) error {
				vodSession, err := client.CreateVodSession(ctx, &stitcherpb.CreateVodSessionRequest{
					Parent: parent,
					VodSession: &stitcherpb.VodSession{
						Name:      vodSessionName,
						VodConfig: vodConfigName,
					},
				})
				if err != nil {
					return err
				}
				if strings.TrimSpace(vodSession.GetName()) == "" {
					return errors.New("CreateVodSession returned empty name")
				}
				return nil
			},
		},
		{
			name: "GetVodSession",
			call: func(ctx context.Context) error {
				vodSession, err := client.GetVodSession(ctx, &stitcherpb.GetVodSessionRequest{Name: vodSessionName})
				if err != nil {
					return err
				}
				if vodSession.GetName() != vodSessionName {
					return fmt.Errorf("GetVodSession returned unexpected name: %q", vodSession.GetName())
				}
				if strings.TrimSpace(vodSession.GetPlayUri()) == "" {
					return errors.New("GetVodSession returned empty play uri")
				}
				return nil
			},
		},
		{
			name: "CreateLiveSession",
			call: func(ctx context.Context) error {
				liveSession, err := client.CreateLiveSession(ctx, &stitcherpb.CreateLiveSessionRequest{
					Parent: parent,
					LiveSession: &stitcherpb.LiveSession{
						Name:       liveSessionName,
						LiveConfig: liveConfigName,
					},
				})
				if err != nil {
					return err
				}
				if strings.TrimSpace(liveSession.GetName()) == "" {
					return errors.New("CreateLiveSession returned empty name")
				}
				return nil
			},
		},
		{
			name: "GetLiveSession",
			call: func(ctx context.Context) error {
				liveSession, err := client.GetLiveSession(ctx, &stitcherpb.GetLiveSessionRequest{Name: liveSessionName})
				if err != nil {
					return err
				}
				if liveSession.GetName() != liveSessionName {
					return fmt.Errorf("GetLiveSession returned unexpected name: %q", liveSession.GetName())
				}
				return nil
			},
		},
		{
			name: "UpdateCdnKey",
			call: func(ctx context.Context) error {
				op, err := client.UpdateCdnKey(ctx, &stitcherpb.UpdateCdnKeyRequest{
					CdnKey: &stitcherpb.CdnKey{
						Name: cdnKeyName,
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"hostname"}},
				})
				if err != nil {
					return err
				}
				if strings.TrimSpace(op.Name()) == "" {
					return errors.New("UpdateCdnKey returned empty operation name")
				}
				return nil
			},
		},
		{
			name: "GetOperation",
			call: func(ctx context.Context) error {
				operation, err := client.GetOperation(ctx, &longrunningpb.GetOperationRequest{Name: operationName})
				if err != nil {
					return err
				}
				if strings.TrimSpace(operation.GetName()) == "" {
					return errors.New("GetOperation returned empty name")
				}
				return nil
			},
		},
		{
			name: "ListOperations",
			call: func(ctx context.Context) error {
				it := client.ListOperations(ctx, &longrunningpb.ListOperationsRequest{
					Name:     parent,
					PageSize: 1,
				})
				operation, err := it.Next()
				if errors.Is(err, iterator.Done) {
					return errors.New("ListOperations returned no items")
				}
				if err != nil {
					return err
				}
				if strings.TrimSpace(operation.GetName()) == "" {
					return errors.New("ListOperations returned operation without name")
				}
				return nil
			},
		},
		{
			name: "NegativeValidation_CreateCdnKeyMissingParent",
			call: func(ctx context.Context) error {
				_, err := client.CreateCdnKey(ctx, &stitcherpb.CreateCdnKeyRequest{
					CdnKeyId: "invalid-no-parent",
					CdnKey: &stitcherpb.CdnKey{
						Name: parent + "/cdnKeys/invalid-no-parent",
					},
				})
				if isInvalidArgument(err) {
					return nil
				}
				if err == nil {
					return errors.New("expected invalid argument for create cdn key missing parent")
				}
				return fmt.Errorf("expected invalid argument, got: %w", err)
			},
		},
	}

	for _, call := range calls {
		if err := call.call(ctx); err != nil {
			exitf("%s failed: %v", call.name, err)
		}
		logf("%s succeeded", call.name)
	}

	fmt.Println("Done.")
}

func waitForStackyardReady(ctx context.Context, apiEndpoint, projectID, locationID string) error {
	client := &http.Client{Timeout: 2 * time.Second}
	probeURL := fmt.Sprintf("%s/v1/projects/%s/locations/%s/video_stitcher?stackyard_contract_probe=1&typedSuccess=1", apiEndpoint, projectID, locationID)

	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, probeURL, nil)
		if err != nil {
			return err
		}
		req.Header.Set("X-Stackyard-GCP-Service", "video-stitcher")

		resp, err := client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("video stitcher contract probe did not become ready: %s", probeURL)
}

func isInvalidArgument(err error) bool {
	if err == nil {
		return false
	}
	if grpcStatus, ok := status.FromError(err); ok && grpcStatus.Code() == codes.InvalidArgument {
		return true
	}
	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) && apiErr.Code == http.StatusBadRequest {
		return true
	}
	return strings.Contains(strings.ToLower(err.Error()), "invalidargument")
}

func closeClient(label string, closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close %s client: %v\n", label, err)
	}
}

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
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
