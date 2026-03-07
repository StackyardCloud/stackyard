package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	adapter "cloud.google.com/go/spanner/adapter/apiv1"
	"cloud.google.com/go/spanner/adapter/apiv1/adapterpb"
	"google.golang.org/api/option"
)

type callSpec struct {
	name string
	call func(context.Context, *adapter.Client) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	instanceID := getenv("STACKYARD_GCP_SPANNER_INSTANCE", "stackyard-instance")
	databaseID := getenv("STACKYARD_GCP_SPANNER_DATABASE", "stackyard-db")
	sessionID := getenv("STACKYARD_GCP_SPANNER_ADAPTER_SESSION_ID", "as-1")
	protocol := getenv("STACKYARD_GCP_SPANNER_ADAPTER_PROTOCOL", "pgwire")
	payloadText := getenv("STACKYARD_GCP_SPANNER_ADAPTER_PAYLOAD", "hello")
	location := getenv("STACKYARD_GCP_LOCATION", "us-central1")

	databaseName := fmt.Sprintf("projects/%s/instances/%s/databases/%s", projectID, instanceID, databaseID)
	sessionName := fmt.Sprintf("%s/sessions/%s", databaseName, sessionID)

	fmt.Printf("Stackyard GCP Spanner Adapter apiv1 client using %s\n", apiEndpoint)
	if err := waitForStackyardReady(ctx, apiEndpoint, projectID, location); err != nil {
		exitf("stackyard readiness check failed: %v", err)
	}

	httpClient := &http.Client{
		Transport: stackyardHeaderTransport{
			base:        http.DefaultTransport,
			serviceName: "spanner-adapter",
		},
	}

	client, err := adapter.NewRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create spanner adapter client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "CreateSession",
			call: func(ctx context.Context, c *adapter.Client) error {
				_, err := c.CreateSession(ctx, &adapterpb.CreateSessionRequest{
					Parent:  databaseName,
					Session: &adapterpb.Session{Name: sessionName},
				})
				return err
			},
		},
		{
			name: "AdaptMessage",
			call: func(ctx context.Context, c *adapter.Client) error {
				stream, err := c.AdaptMessage(ctx, &adapterpb.AdaptMessageRequest{
					Name:     sessionName,
					Protocol: protocol,
					Payload:  []byte(payloadText),
					Attachments: map[string]string{
						"trace": "stackyard",
					},
				})
				if err != nil {
					return err
				}
				for {
					resp, err := stream.Recv()
					if errors.Is(err, io.EOF) {
						return nil
					}
					if err != nil {
						return err
					}
					if len(resp.GetPayload()) == 0 {
						return fmt.Errorf("adapt message returned empty payload")
					}
					if resp.GetLast() {
						return nil
					}
				}
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
		fmt.Fprintf(os.Stderr, "warning: close spanner adapter client: %v\n", err)
	}
}

func waitForStackyardReady(ctx context.Context, apiEndpoint, projectID, location string) error {
	readyURL := fmt.Sprintf("%s/v1/projects/%s/locations/%s/spanner_adapter?stackyard_contract_probe=1&typedSuccess=1", strings.TrimRight(apiEndpoint, "/"), projectID, location)
	httpClient := &http.Client{Timeout: time.Second}
	deadline := time.Now().Add(20 * time.Second)
	var lastErr error

	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, readyURL, nil)
		if err != nil {
			return err
		}
		req.Header.Set("X-Stackyard-GCP-Service", "spanner-adapter")

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
