package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	identitytoolkit "cloud.google.com/go/identitytoolkit/apiv2"
	"cloud.google.com/go/identitytoolkit/apiv2/identitytoolkitpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type callSpec struct {
	name string
	call func(context.Context, *identitytoolkit.AccountManagementClient, *identitytoolkit.AuthenticationClient) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	fmt.Printf("Stackyard GCP Identity Toolkit apiv2 clients using %s\n", apiEndpoint)

	accountClient, err := identitytoolkit.NewAccountManagementRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create account management client: %v", err)
	}
	defer closeClient("account management", accountClient.Close)

	authClient, err := identitytoolkit.NewAuthenticationRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create authentication client: %v", err)
	}
	defer closeClient("authentication", authClient.Close)

	calls := []callSpec{
		{
			name: "StartMfaEnrollment",
			call: func(ctx context.Context, accountClient *identitytoolkit.AccountManagementClient, _ *identitytoolkit.AuthenticationClient) error {
				_, err := accountClient.StartMfaEnrollment(ctx, &identitytoolkitpb.StartMfaEnrollmentRequest{})
				return err
			},
		},
		{
			name: "FinalizeMfaEnrollment",
			call: func(ctx context.Context, accountClient *identitytoolkit.AccountManagementClient, _ *identitytoolkit.AuthenticationClient) error {
				_, err := accountClient.FinalizeMfaEnrollment(ctx, &identitytoolkitpb.FinalizeMfaEnrollmentRequest{})
				return err
			},
		},
		{
			name: "WithdrawMfa",
			call: func(ctx context.Context, accountClient *identitytoolkit.AccountManagementClient, _ *identitytoolkit.AuthenticationClient) error {
				_, err := accountClient.WithdrawMfa(ctx, &identitytoolkitpb.WithdrawMfaRequest{})
				return err
			},
		},
		{
			name: "StartMfaSignIn",
			call: func(ctx context.Context, _ *identitytoolkit.AccountManagementClient, authClient *identitytoolkit.AuthenticationClient) error {
				_, err := authClient.StartMfaSignIn(ctx, &identitytoolkitpb.StartMfaSignInRequest{})
				return err
			},
		},
		{
			name: "FinalizeMfaSignIn",
			call: func(ctx context.Context, _ *identitytoolkit.AccountManagementClient, authClient *identitytoolkit.AuthenticationClient) error {
				_, err := authClient.FinalizeMfaSignIn(ctx, &identitytoolkitpb.FinalizeMfaSignInRequest{})
				return err
			},
		},
	}

	for _, call := range calls {
		err := call.call(ctx, accountClient, authClient)
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

func closeClient(name string, closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close %s client: %v\n", name, err)
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
