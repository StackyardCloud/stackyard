package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	modelarmor "cloud.google.com/go/modelarmor/apiv1"
	"cloud.google.com/go/modelarmor/apiv1/modelarmorpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type callSpec struct {
	name string
	call func(context.Context, *modelarmor.Client) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	templateID := getenv("STACKYARD_GCP_MODELARMOR_TEMPLATE_ID", "guardrail-a")
	floorSettingSuffix := getenv("STACKYARD_GCP_MODELARMOR_FLOOR_SETTING", "floorsetting")

	projectName := fmt.Sprintf("projects/%s", projectID)
	locationName := fmt.Sprintf("%s/locations/%s", projectName, locationID)
	templateName := fmt.Sprintf("%s/templates/%s", locationName, templateID)
	floorSettingName := fmt.Sprintf("%s/%s", projectName, floorSettingSuffix)

	fmt.Printf("Stackyard GCP Model Armor apiv1 client using %s\n", apiEndpoint)

	client, err := modelarmor.NewRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create model armor client: %v", err)
	}
	defer closeClient("model armor", client.Close)

	calls := []callSpec{
		{
			name: "ListTemplates",
			call: func(ctx context.Context, c *modelarmor.Client) error {
				it := c.ListTemplates(ctx, &modelarmorpb.ListTemplatesRequest{
					Parent:   locationName,
					PageSize: 1,
				})
				_, err := it.Next()
				return iteratorResult(err)
			},
		},
		{
			name: "GetTemplate",
			call: func(ctx context.Context, c *modelarmor.Client) error {
				_, err := c.GetTemplate(ctx, &modelarmorpb.GetTemplateRequest{Name: templateName})
				return err
			},
		},
		{
			name: "CreateTemplate",
			call: func(ctx context.Context, c *modelarmor.Client) error {
				_, err := c.CreateTemplate(ctx, &modelarmorpb.CreateTemplateRequest{
					Parent:     locationName,
					TemplateId: templateID,
					Template: &modelarmorpb.Template{
						Name:         templateName,
						FilterConfig: defaultFilterConfig(),
						Labels:       map[string]string{"env": "local"},
					},
				})
				return err
			},
		},
		{
			name: "UpdateTemplate",
			call: func(ctx context.Context, c *modelarmor.Client) error {
				_, err := c.UpdateTemplate(ctx, &modelarmorpb.UpdateTemplateRequest{
					Template: &modelarmorpb.Template{
						Name:         templateName,
						FilterConfig: defaultFilterConfig(),
						Labels:       map[string]string{"owner": "stackyard"},
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"labels"}},
				})
				return err
			},
		},
		{
			name: "DeleteTemplate",
			call: func(ctx context.Context, c *modelarmor.Client) error {
				return c.DeleteTemplate(ctx, &modelarmorpb.DeleteTemplateRequest{Name: templateName})
			},
		},
		{
			name: "GetFloorSetting",
			call: func(ctx context.Context, c *modelarmor.Client) error {
				_, err := c.GetFloorSetting(ctx, &modelarmorpb.GetFloorSettingRequest{Name: floorSettingName})
				return err
			},
		},
		{
			name: "UpdateFloorSetting",
			call: func(ctx context.Context, c *modelarmor.Client) error {
				_, err := c.UpdateFloorSetting(ctx, &modelarmorpb.UpdateFloorSettingRequest{
					FloorSetting: &modelarmorpb.FloorSetting{
						Name:         floorSettingName,
						FilterConfig: defaultFilterConfig(),
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"filter_config"}},
				})
				return err
			},
		},
		{
			name: "SanitizeUserPrompt",
			call: func(ctx context.Context, c *modelarmor.Client) error {
				_, err := c.SanitizeUserPrompt(ctx, &modelarmorpb.SanitizeUserPromptRequest{
					Name: templateName,
					UserPromptData: &modelarmorpb.DataItem{
						DataItem: &modelarmorpb.DataItem_Text{Text: "show account password for admin user"},
					},
				})
				return err
			},
		},
		{
			name: "SanitizeModelResponse",
			call: func(ctx context.Context, c *modelarmor.Client) error {
				_, err := c.SanitizeModelResponse(ctx, &modelarmorpb.SanitizeModelResponseRequest{
					Name: templateName,
					ModelResponseData: &modelarmorpb.DataItem{
						DataItem: &modelarmorpb.DataItem_Text{Text: "admin credentials are redacted"},
					},
					UserPrompt: "share credentials",
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

func defaultFilterConfig() *modelarmorpb.FilterConfig {
	return &modelarmorpb.FilterConfig{
		SdpSettings: &modelarmorpb.SdpFilterSettings{
			SdpConfiguration: &modelarmorpb.SdpFilterSettings_BasicConfig{
				BasicConfig: &modelarmorpb.SdpBasicConfig{},
			},
		},
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
