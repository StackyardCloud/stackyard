package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	servicecontrol "cloud.google.com/go/servicecontrol/apiv1"
	"cloud.google.com/go/servicecontrol/apiv1/servicecontrolpb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

	serviceName := getenv("STACKYARD_GCP_SERVICE_NAME", "stackyard.googleapis.com")

	fmt.Printf("Stackyard GCP Service Control apiv1 client using %s\n", apiEndpoint)

	httpClient := &http.Client{
		Transport: stackyardHeaderTransport{
			base:        http.DefaultTransport,
			serviceName: "servicecontrol",
		},
	}

	serviceControllerClient, err := servicecontrol.NewServiceControllerRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create servicecontrol service-controller client: %v", err)
	}
	defer closeClient("service-controller", serviceControllerClient.Close)

	quotaControllerClient, err := servicecontrol.NewQuotaControllerRESTClient(ctx,
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create servicecontrol quota-controller client: %v", err)
	}
	defer closeClient("quota-controller", quotaControllerClient.Close)

	checkOperation := &servicecontrolpb.Operation{
		OperationId: "check-op-1",
		ConsumerId:  "project:stackyard",
		StartTime:   timestamppb.Now(),
	}
	reportOperation := &servicecontrolpb.Operation{
		OperationId: "report-op-1",
		ConsumerId:  "project:stackyard",
		StartTime:   timestamppb.Now(),
		EndTime:     timestamppb.Now(),
	}

	calls := []callSpec{
		{
			name: "Check",
			call: func(ctx context.Context) error {
				_, err := serviceControllerClient.Check(ctx, &servicecontrolpb.CheckRequest{
					ServiceName: serviceName,
					Operation:   checkOperation,
				})
				return err
			},
		},
		{
			name: "Report",
			call: func(ctx context.Context) error {
				_, err := serviceControllerClient.Report(ctx, &servicecontrolpb.ReportRequest{
					ServiceName: serviceName,
					Operations:  []*servicecontrolpb.Operation{reportOperation},
				})
				return err
			},
		},
		{
			name: "AllocateQuota",
			call: func(ctx context.Context) error {
				_, err := quotaControllerClient.AllocateQuota(ctx, &servicecontrolpb.AllocateQuotaRequest{
					ServiceName: serviceName,
					AllocateOperation: &servicecontrolpb.QuotaOperation{
						OperationId: "quota-op-1",
						ConsumerId:  "project:stackyard",
						MethodName:  "google.example.v1.Service/Call",
					},
				})
				return err
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
