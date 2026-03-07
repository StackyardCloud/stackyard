package main

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/compute/metadata"
)

type callSpec struct {
	name string
	call func(context.Context, *metadata.Client) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	metadataHost := ensureMetadataHost(endpoint)

	projectAttr := getenv("STACKYARD_GCP_METADATA_PROJECT_ATTRIBUTE", "env")
	instanceAttr := getenv("STACKYARD_GCP_METADATA_INSTANCE_ATTRIBUTE", "role")

	fmt.Printf("Stackyard GCP Compute Metadata client using host %s\n", metadataHost)

	client := metadata.NewWithOptions(&metadata.Options{UseDefaultClient: true})
	if client == nil {
		exitf("failed to create compute metadata client")
	}

	onGCECtx, onGCECancel := context.WithTimeout(ctx, 2*time.Second)
	onGCE := client.OnGCEWithContext(onGCECtx)
	onGCECancel()
	logf("OnGCEWithContext=%t", onGCE)

	calls := []callSpec{
		{
			name: "Get(project/project-id)",
			call: func(ctx context.Context, c *metadata.Client) error {
				_, err := c.GetWithContext(ctx, "project/project-id")
				return err
			},
		},
		{
			name: "ProjectIDWithContext",
			call: func(ctx context.Context, c *metadata.Client) error {
				_, err := c.ProjectIDWithContext(ctx)
				return err
			},
		},
		{
			name: "NumericProjectIDWithContext",
			call: func(ctx context.Context, c *metadata.Client) error {
				_, err := c.NumericProjectIDWithContext(ctx)
				return err
			},
		},
		{
			name: "InstanceIDWithContext",
			call: func(ctx context.Context, c *metadata.Client) error {
				_, err := c.InstanceIDWithContext(ctx)
				return err
			},
		},
		{
			name: "InstanceNameWithContext",
			call: func(ctx context.Context, c *metadata.Client) error {
				_, err := c.InstanceNameWithContext(ctx)
				return err
			},
		},
		{
			name: "ZoneWithContext",
			call: func(ctx context.Context, c *metadata.Client) error {
				_, err := c.ZoneWithContext(ctx)
				return err
			},
		},
		{
			name: "HostnameWithContext",
			call: func(ctx context.Context, c *metadata.Client) error {
				_, err := c.HostnameWithContext(ctx)
				return err
			},
		},
		{
			name: "InternalIPWithContext",
			call: func(ctx context.Context, c *metadata.Client) error {
				_, err := c.InternalIPWithContext(ctx)
				return err
			},
		},
		{
			name: "ExternalIPWithContext",
			call: func(ctx context.Context, c *metadata.Client) error {
				_, err := c.ExternalIPWithContext(ctx)
				return err
			},
		},
		{
			name: "EmailWithContext(default)",
			call: func(ctx context.Context, c *metadata.Client) error {
				_, err := c.EmailWithContext(ctx, "default")
				return err
			},
		},
		{
			name: "ScopesWithContext(default)",
			call: func(ctx context.Context, c *metadata.Client) error {
				_, err := c.ScopesWithContext(ctx, "default")
				return err
			},
		},
		{
			name: "ProjectAttributesWithContext",
			call: func(ctx context.Context, c *metadata.Client) error {
				_, err := c.ProjectAttributesWithContext(ctx)
				return err
			},
		},
		{
			name: "ProjectAttributeValueWithContext(" + projectAttr + ")",
			call: func(ctx context.Context, c *metadata.Client) error {
				_, err := c.ProjectAttributeValueWithContext(ctx, projectAttr)
				return err
			},
		},
		{
			name: "InstanceAttributesWithContext",
			call: func(ctx context.Context, c *metadata.Client) error {
				_, err := c.InstanceAttributesWithContext(ctx)
				return err
			},
		},
		{
			name: "InstanceAttributeValueWithContext(" + instanceAttr + ")",
			call: func(ctx context.Context, c *metadata.Client) error {
				_, err := c.InstanceAttributeValueWithContext(ctx, instanceAttr)
				return err
			},
		},
		{
			name: "InstanceTagsWithContext",
			call: func(ctx context.Context, c *metadata.Client) error {
				_, err := c.InstanceTagsWithContext(ctx)
				return err
			},
		},
	}

	for _, call := range calls {
		callCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err := call.call(callCtx, client)
		cancel()

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

func ensureMetadataHost(endpoint string) string {
	if current := strings.TrimSpace(os.Getenv("GCE_METADATA_HOST")); current != "" {
		return current
	}
	host := metadataHostFromEndpoint(endpoint)
	_ = os.Setenv("GCE_METADATA_HOST", host)
	return host
}

func metadataHostFromEndpoint(endpoint string) string {
	raw := strings.TrimSpace(endpoint)
	if raw == "" {
		return "localhost:4566"
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return strings.TrimPrefix(strings.TrimPrefix(raw, "http://"), "https://")
	}
	if parsed.Host != "" {
		return parsed.Host
	}
	if parsed.Path != "" {
		return parsed.Path
	}
	return "localhost:4566"
}

func isToleratedNotImplemented(err error) bool {
	if err == nil {
		return false
	}

	var metadataErr *metadata.Error
	if errors.As(err, &metadataErr) && metadataErr.Code == 501 {
		return true
	}

	return strings.Contains(strings.ToLower(err.Error()), "notimplemented")
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
