package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	publish "cloud.google.com/go/streetview/publish/apiv1"
	"cloud.google.com/go/streetview/publish/apiv1/publishpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type callSpec struct {
	name string
	call func(context.Context, *publish.StreetViewPublishClient) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	photoID := getenv("STACKYARD_GCP_STREETVIEW_PHOTO_ID", "photo-1")
	sequenceID := getenv("STACKYARD_GCP_STREETVIEW_SEQUENCE_ID", "sequence-1")

	photoUploadURL := fmt.Sprintf("https://streetviewpublish.googleapis.com/media/user/stackyard/photo/%s", photoID)
	sequenceUploadURL := fmt.Sprintf("https://streetviewpublish.googleapis.com/media/user/stackyard/photo/sequence-%s", sequenceID)

	fmt.Printf("Stackyard GCP Street View Publish apiv1 client using %s\n", apiEndpoint)

	httpClient := &http.Client{
		Transport: stackyardHeaderTransport{
			base:        http.DefaultTransport,
			serviceName: "streetview_publish",
		},
	}

	client, err := publish.NewStreetViewPublishRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create streetview publish client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "StartUpload",
			call: func(ctx context.Context, c *publish.StreetViewPublishClient) error {
				ref, err := c.StartUpload(ctx, &emptypb.Empty{})
				if err != nil {
					return err
				}
				if got := strings.TrimSpace(ref.GetUploadUrl()); got != "" {
					photoUploadURL = got
				}
				return nil
			},
		},
		{
			name: "CreatePhoto",
			call: func(ctx context.Context, c *publish.StreetViewPublishClient) error {
				_, err := c.CreatePhoto(ctx, &publishpb.CreatePhotoRequest{
					Photo: &publishpb.Photo{
						PhotoId: &publishpb.PhotoId{Id: photoID},
						UploadReference: &publishpb.UploadRef{
							FileSource: &publishpb.UploadRef_UploadUrl{UploadUrl: photoUploadURL},
						},
					},
				})
				return err
			},
		},
		{
			name: "GetPhoto",
			call: func(ctx context.Context, c *publish.StreetViewPublishClient) error {
				_, err := c.GetPhoto(ctx, &publishpb.GetPhotoRequest{
					PhotoId: photoID,
					View:    publishpb.PhotoView_INCLUDE_DOWNLOAD_URL,
				})
				return err
			},
		},
		{
			name: "BatchGetPhotos",
			call: func(ctx context.Context, c *publish.StreetViewPublishClient) error {
				_, err := c.BatchGetPhotos(ctx, &publishpb.BatchGetPhotosRequest{
					PhotoIds: []string{photoID, "missing-photo"},
					View:     publishpb.PhotoView_BASIC,
				})
				return err
			},
		},
		{
			name: "ListPhotos",
			call: func(ctx context.Context, c *publish.StreetViewPublishClient) error {
				it := c.ListPhotos(ctx, &publishpb.ListPhotosRequest{
					View:     publishpb.PhotoView_BASIC,
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
			name: "UpdatePhoto",
			call: func(ctx context.Context, c *publish.StreetViewPublishClient) error {
				_, err := c.UpdatePhoto(ctx, &publishpb.UpdatePhotoRequest{
					Photo: &publishpb.Photo{
						PhotoId: &publishpb.PhotoId{Id: photoID},
						Pose: &publishpb.Pose{
							Heading: 120,
						},
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"pose.heading"}},
				})
				return err
			},
		},
		{
			name: "BatchUpdatePhotos",
			call: func(ctx context.Context, c *publish.StreetViewPublishClient) error {
				_, err := c.BatchUpdatePhotos(ctx, &publishpb.BatchUpdatePhotosRequest{
					UpdatePhotoRequests: []*publishpb.UpdatePhotoRequest{
						{
							Photo: &publishpb.Photo{
								PhotoId: &publishpb.PhotoId{Id: photoID},
								Pose:    &publishpb.Pose{Heading: 135},
							},
							UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"pose.heading"}},
						},
					},
				})
				return err
			},
		},
		{
			name: "DeletePhoto",
			call: func(ctx context.Context, c *publish.StreetViewPublishClient) error {
				return c.DeletePhoto(ctx, &publishpb.DeletePhotoRequest{PhotoId: photoID})
			},
		},
		{
			name: "BatchDeletePhotos",
			call: func(ctx context.Context, c *publish.StreetViewPublishClient) error {
				_, err := c.BatchDeletePhotos(ctx, &publishpb.BatchDeletePhotosRequest{PhotoIds: []string{photoID, "missing-photo"}})
				return err
			},
		},
		{
			name: "StartPhotoSequenceUpload",
			call: func(ctx context.Context, c *publish.StreetViewPublishClient) error {
				ref, err := c.StartPhotoSequenceUpload(ctx, &emptypb.Empty{})
				if err != nil {
					return err
				}
				if got := strings.TrimSpace(ref.GetUploadUrl()); got != "" {
					sequenceUploadURL = got
				}
				return nil
			},
		},
		{
			name: "CreatePhotoSequence",
			call: func(ctx context.Context, c *publish.StreetViewPublishClient) error {
				op, err := c.CreatePhotoSequence(ctx, &publishpb.CreatePhotoSequenceRequest{
					PhotoSequence: &publishpb.PhotoSequence{
						Id: sequenceID,
						UploadReference: &publishpb.UploadRef{
							FileSource: &publishpb.UploadRef_UploadUrl{UploadUrl: sequenceUploadURL},
						},
					},
					InputType: publishpb.CreatePhotoSequenceRequest_VIDEO,
				})
				if err != nil {
					return err
				}
				if _, err := op.Poll(ctx); err != nil {
					return err
				}
				if name := strings.TrimSpace(op.Name()); name != "" {
					if parsed := strings.TrimPrefix(name, "operations/photoSequence."); parsed != name {
						sequenceID = parsed
					}
					_, err = c.CreatePhotoSequenceOperation(name).Poll(ctx)
					if err != nil {
						return err
					}
				}
				return nil
			},
		},
		{
			name: "GetPhotoSequence",
			call: func(ctx context.Context, c *publish.StreetViewPublishClient) error {
				op, err := c.GetPhotoSequence(ctx, &publishpb.GetPhotoSequenceRequest{
					SequenceId: sequenceID,
					Filter:     "published_status=PUBLISHED",
				})
				if err != nil {
					return err
				}
				if _, err := op.Poll(ctx); err != nil {
					return err
				}
				if name := strings.TrimSpace(op.Name()); name != "" {
					_, err = c.GetPhotoSequenceOperation(name).Poll(ctx)
					if err != nil {
						return err
					}
				}
				return nil
			},
		},
		{
			name: "ListPhotoSequences",
			call: func(ctx context.Context, c *publish.StreetViewPublishClient) error {
				it := c.ListPhotoSequences(ctx, &publishpb.ListPhotoSequencesRequest{
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
			name: "DeletePhotoSequence",
			call: func(ctx context.Context, c *publish.StreetViewPublishClient) error {
				return c.DeletePhotoSequence(ctx, &publishpb.DeletePhotoSequenceRequest{SequenceId: sequenceID})
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
		fmt.Fprintf(os.Stderr, "warning: close streetview publish client: %v\n", err)
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
