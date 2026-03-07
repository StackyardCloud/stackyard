package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	shell "cloud.google.com/go/shell/apiv1"
	"cloud.google.com/go/shell/apiv1/shellpb"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type callSpec struct {
	name string
	call func(context.Context) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	environmentName := getenv("STACKYARD_GCP_SHELL_ENV", "users/me/environments/default")
	newPublicKey := getenv("STACKYARD_GCP_SHELL_PUBLIC_KEY", "ssh-rsa c3RhY2t5YXJkLW5ldy1rZXk= stackyard@example.com")

	fmt.Printf("Stackyard GCP Cloud Shell apiv1 client using %s\n", apiEndpoint)
	if err := waitForStackyardReady(ctx, apiEndpoint); err != nil {
		exitf("stackyard readiness check failed: %v", err)
	}

	httpClient := &http.Client{
		Transport: stackyardHeaderTransport{
			base:        http.DefaultTransport,
			serviceName: "shell",
		},
	}

	client, err := shell.NewCloudShellRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create shell client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "GetEnvironment",
			call: func(ctx context.Context) error {
				_, err := client.GetEnvironment(ctx, &shellpb.GetEnvironmentRequest{
					Name: environmentName,
				})
				return err
			},
		},
		{
			name: "StartEnvironment",
			call: func(ctx context.Context) error {
				op, err := client.StartEnvironment(ctx, &shellpb.StartEnvironmentRequest{
					Name:       environmentName,
					PublicKeys: []string{newPublicKey},
				})
				if err != nil {
					return err
				}
				_, err = op.Wait(ctx)
				return err
			},
		},
		{
			name: "AuthorizeEnvironment",
			call: func(ctx context.Context) error {
				op, err := client.AuthorizeEnvironment(ctx, &shellpb.AuthorizeEnvironmentRequest{
					Name:        environmentName,
					AccessToken: "ya29.stackyard",
					IdToken:     "stackyard-id-token",
					ExpireTime:  timestamppb.New(time.Now().UTC().Add(time.Hour)),
				})
				if err != nil {
					return err
				}
				_, err = op.Wait(ctx)
				return err
			},
		},
		{
			name: "AddPublicKey",
			call: func(ctx context.Context) error {
				op, err := client.AddPublicKey(ctx, &shellpb.AddPublicKeyRequest{
					Environment: environmentName,
					Key:         newPublicKey,
				})
				if err != nil {
					return err
				}
				_, err = op.Wait(ctx)
				return err
			},
		},
		{
			name: "RemovePublicKey",
			call: func(ctx context.Context) error {
				op, err := client.RemovePublicKey(ctx, &shellpb.RemovePublicKeyRequest{
					Environment: environmentName,
					Key:         newPublicKey,
				})
				if err != nil {
					return err
				}
				_, err = op.Wait(ctx)
				return err
			},
		},
	}

	for _, call := range calls {
		err := call.call(ctx)
		if err == nil {
			logf("%s succeeded", call.name)
		} else {
			exitf("%s failed: %v", call.name, err)
		}
	}

	fmt.Println("Done.")
}

func closeClient(closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close shell client: %v\n", err)
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

func waitForStackyardReady(ctx context.Context, apiEndpoint string) error {
	readyURL := strings.TrimRight(apiEndpoint, "/") + "/v1/users/me/environments/default"
	httpClient := &http.Client{Timeout: time.Second}
	deadline := time.Now().Add(20 * time.Second)
	var lastErr error

	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, readyURL, nil)
		if err != nil {
			return err
		}
		req.Header.Set("X-Stackyard-GCP-Service", "shell")
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
