package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	oslogin "cloud.google.com/go/oslogin/apiv1"
	"cloud.google.com/go/oslogin/apiv1/osloginpb"
	commonpb "cloud.google.com/go/oslogin/common/commonpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type callSpec struct {
	name string
	call func(context.Context, *oslogin.Client) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	user := getenv("STACKYARD_GCP_OSLOGIN_USER", "alice@example.com")
	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	fingerprint := getenv("STACKYARD_GCP_OSLOGIN_FINGERPRINT", "fingerprint-1")

	userName := fmt.Sprintf("users/%s", user)
	sshKeyName := fmt.Sprintf("%s/sshPublicKeys/%s", userName, fingerprint)
	posixAccountName := fmt.Sprintf("%s/projects/%s", userName, projectID)

	expiryUsec := time.Now().Add(24 * time.Hour).UnixMicro()
	sampleKey := &commonpb.SshPublicKey{
		Key:                "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIMockStackyardKey stackyard@example",
		ExpirationTimeUsec: expiryUsec,
	}

	fmt.Printf("Stackyard GCP OS Login apiv1 client using %s\n", apiEndpoint)

	client, err := oslogin.NewRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create oslogin client: %v", err)
	}
	defer closeClient("oslogin", client.Close)

	calls := []callSpec{
		{
			name: "CreateSshPublicKey",
			call: func(ctx context.Context, c *oslogin.Client) error {
				_, err := c.CreateSshPublicKey(ctx, &osloginpb.CreateSshPublicKeyRequest{
					Parent:       userName,
					SshPublicKey: sampleKey,
				})
				return err
			},
		},
		{
			name: "GetLoginProfile",
			call: func(ctx context.Context, c *oslogin.Client) error {
				_, err := c.GetLoginProfile(ctx, &osloginpb.GetLoginProfileRequest{
					Name:      userName,
					ProjectId: projectID,
				})
				return err
			},
		},
		{
			name: "GetSshPublicKey",
			call: func(ctx context.Context, c *oslogin.Client) error {
				_, err := c.GetSshPublicKey(ctx, &osloginpb.GetSshPublicKeyRequest{Name: sshKeyName})
				return err
			},
		},
		{
			name: "ImportSshPublicKey",
			call: func(ctx context.Context, c *oslogin.Client) error {
				_, err := c.ImportSshPublicKey(ctx, &osloginpb.ImportSshPublicKeyRequest{
					Parent:       userName,
					SshPublicKey: sampleKey,
					ProjectId:    projectID,
					Regions:      []string{"us-central1"},
				})
				return err
			},
		},
		{
			name: "UpdateSshPublicKey",
			call: func(ctx context.Context, c *oslogin.Client) error {
				_, err := c.UpdateSshPublicKey(ctx, &osloginpb.UpdateSshPublicKeyRequest{
					Name:         sshKeyName,
					SshPublicKey: sampleKey,
					UpdateMask: &fieldmaskpb.FieldMask{
						Paths: []string{"expiration_time_usec"},
					},
				})
				return err
			},
		},
		{
			name: "DeleteSshPublicKey",
			call: func(ctx context.Context, c *oslogin.Client) error {
				return c.DeleteSshPublicKey(ctx, &osloginpb.DeleteSshPublicKeyRequest{Name: sshKeyName})
			},
		},
		{
			name: "DeletePosixAccount",
			call: func(ctx context.Context, c *oslogin.Client) error {
				return c.DeletePosixAccount(ctx, &osloginpb.DeletePosixAccountRequest{Name: posixAccountName})
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

	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) && apiErr.Code == 501 {
		return true
	}

	return strings.Contains(strings.ToLower(err.Error()), "notimplemented") ||
		strings.Contains(strings.ToLower(err.Error()), "not implemented")
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
