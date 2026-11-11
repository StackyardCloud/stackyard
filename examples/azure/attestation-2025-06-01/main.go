package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/attestation/2020-10-01/attestation"
	"github.com/Azure/go-autorest/autorest"
	"github.com/stackyard/stackyard/examples/azure/internal/azsdkshim"
)

func main() {
	ctx := context.Background()
	endpoint := strings.TrimRight(getenv("STACKYARD_ENDPOINT", "http://localhost:4566"), "/")
	account := getenv("STACKYARD_AZURE_ATTESTATION_ACCOUNT", "devstoreaccount1")
	subscriptionKey := getenv("STACKYARD_AZURE_ATTESTATION_SUBSCRIPTION_KEY", "stackyard-local-subscription-key")
	instanceURL := strings.TrimRight(getenv("STACKYARD_AZURE_ATTESTATION_INSTANCE_URL", endpoint+"/attestation"), "/")

	fmt.Printf("Stackyard Azure Attestation typed SDK example using %s\n", instanceURL)

	transport := azsdkshim.NewTransport("/azure", account, subscriptionKey)

	metadataClient := attestation.NewMetadataConfigurationClient()
	signingCertificatesClient := attestation.NewSigningCertificatesClient()
	policyClient := attestation.NewPolicyClient()
	policyCertificatesClient := attestation.NewPolicyCertificatesClient()
	attestationClient := attestation.NewClient()

	metadataClient.Sender = transport
	signingCertificatesClient.Sender = transport
	policyClient.Sender = transport
	policyCertificatesClient.Sender = transport
	attestationClient.Sender = transport

	calls := []struct {
		name string
		run  func() error
	}{
		{
			name: "GetMetadataConfiguration",
			run: func() error {
				_, err := metadataClient.Get(ctx, instanceURL)
				return err
			},
		},
		{
			name: "GetSigningCertificates",
			run: func() error {
				_, err := signingCertificatesClient.Get(ctx, instanceURL)
				return err
			},
		},
		{
			name: "GetPolicy",
			run: func() error {
				_, err := policyClient.Get(ctx, instanceURL, attestation.Type("SgxEnclave"))
				return err
			},
		},
		{
			name: "SetPolicy",
			run: func() error {
				_, err := policyClient.Set(ctx, instanceURL, attestation.Type("SgxEnclave"), "eyJhbGciOiJub25lIn0.eyJBdHRlc3RhdGlvblBvbGljeSI6InZlcnNpb249MS4wOyBhdXRob3JpemF0aW9ucnVsZXMgeyA9PiBwZXJtaXQoKTsgfTsifQ.")
				return err
			},
		},
		{
			name: "ResetPolicy",
			run: func() error {
				_, err := policyClient.Reset(ctx, instanceURL, attestation.Type("SgxEnclave"), "eyJhbGciOiJub25lIn0.e30.")
				return err
			},
		},
		{
			name: "GetPolicyCertificates",
			run: func() error {
				_, err := policyCertificatesClient.Get(ctx, instanceURL)
				return err
			},
		},
		{
			name: "AddPolicyCertificate",
			run: func() error {
				_, err := policyCertificatesClient.Add(ctx, instanceURL, "-----BEGIN CERTIFICATE-----\nU1RBQ0tZQVJELUFUVEVTVEFUSU9OLUNFUlQtMg==\n-----END CERTIFICATE-----")
				return err
			},
		},
		{
			name: "RemovePolicyCertificate",
			run: func() error {
				_, err := policyCertificatesClient.Remove(ctx, instanceURL, "-----BEGIN CERTIFICATE-----\nU1RBQ0tZQVJELUFUVEVTVEFUSU9OLUNFUlQtMg==\n-----END CERTIFICATE-----")
				return err
			},
		},
		{
			name: "AttestOpenEnclave",
			run: func() error {
				_, err := attestationClient.AttestOpenEnclave(ctx, instanceURL, attestation.AttestOpenEnclaveRequest{
					Report: strPtr("b3BlbmVuY2xhdmUtcmVwb3J0"),
				})
				return err
			},
		},
		{
			name: "AttestSgxEnclave",
			run: func() error {
				_, err := attestationClient.AttestSgxEnclave(ctx, instanceURL, attestation.AttestSgxEnclaveRequest{
					Quote: strPtr("c2d4LXF1b3Rl"),
				})
				return err
			},
		},
		{
			name: "AttestTpm",
			run: func() error {
				_, err := attestationClient.AttestTpm(ctx, instanceURL, attestation.TpmAttestationRequest{
					Data: strPtr("dHBtLWRhdGE"),
				})
				return err
			},
		},
	}

	for _, call := range calls {
		if err := call.run(); err != nil {
			if isNotImplemented(err) {
				fmt.Printf("%s: not implemented in emulator yet\n", call.name)
				continue
			}
			exitf("%s failed: %v", call.name, err)
		}
		fmt.Printf("%s: ok\n", call.name)
	}
}

func isNotImplemented(err error) bool {
	if err == nil {
		return false
	}
	if strings.Contains(err.Error(), "StatusCode=501") || strings.Contains(err.Error(), "NotImplemented") {
		return true
	}
	var detailed autorest.DetailedError
	if errors.As(err, &detailed) && detailed.StatusCode == http.StatusNotImplemented {
		return true
	}
	return false
}

func strPtr(v string) *string {
	return &v
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
