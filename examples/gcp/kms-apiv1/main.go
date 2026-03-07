package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	iampb "cloud.google.com/go/iam/apiv1/iampb"
	kms "cloud.google.com/go/kms/apiv1"
	"cloud.google.com/go/kms/apiv1/kmspb"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	locationpb "google.golang.org/genproto/googleapis/cloud/location"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type callSpec struct {
	name string
	call func(context.Context, *kms.KeyManagementClient) error
}

func main() {
	ctx := context.Background()
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	apiEndpoint := strings.TrimRight(endpoint, "/") + "/gcp"

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	keyRingID := getenv("STACKYARD_GCP_KMS_KEY_RING_ID", "team-ring")
	cryptoKeyID := getenv("STACKYARD_GCP_KMS_CRYPTO_KEY_ID", "app-key")
	cryptoKeyVersionID := getenv("STACKYARD_GCP_KMS_CRYPTO_KEY_VERSION_ID", "1")
	importJobID := getenv("STACKYARD_GCP_KMS_IMPORT_JOB_ID", "import-1")

	locationName := fmt.Sprintf("projects/%s/locations/%s", projectID, locationID)
	projectName := fmt.Sprintf("projects/%s", projectID)
	keyRingName := locationName + "/keyRings/" + keyRingID
	cryptoKeyName := keyRingName + "/cryptoKeys/" + cryptoKeyID
	cryptoKeyVersionName := cryptoKeyName + "/cryptoKeyVersions/" + cryptoKeyVersionID
	importJobName := keyRingName + "/importJobs/" + importJobID

	fmt.Printf("Stackyard GCP Cloud KMS apiv1 client using %s\n", apiEndpoint)

	client, err := kms.NewKeyManagementRESTClient(ctx,
		option.WithEndpoint(apiEndpoint),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create kms client: %v", err)
	}
	defer closeClient(client.Close)

	calls := []callSpec{
		{
			name: "ListKeyRings",
			call: func(ctx context.Context, c *kms.KeyManagementClient) error {
				it := c.ListKeyRings(ctx, &kmspb.ListKeyRingsRequest{
					Parent:   locationName,
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
			name: "CreateKeyRing",
			call: func(ctx context.Context, c *kms.KeyManagementClient) error {
				_, err := c.CreateKeyRing(ctx, &kmspb.CreateKeyRingRequest{
					Parent:    locationName,
					KeyRingId: keyRingID,
					KeyRing:   &kmspb.KeyRing{},
				})
				return err
			},
		},
		{
			name: "GetKeyRing",
			call: func(ctx context.Context, c *kms.KeyManagementClient) error {
				_, err := c.GetKeyRing(ctx, &kmspb.GetKeyRingRequest{Name: keyRingName})
				return err
			},
		},
		{
			name: "ListCryptoKeys",
			call: func(ctx context.Context, c *kms.KeyManagementClient) error {
				it := c.ListCryptoKeys(ctx, &kmspb.ListCryptoKeysRequest{
					Parent:   keyRingName,
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
			name: "CreateCryptoKey",
			call: func(ctx context.Context, c *kms.KeyManagementClient) error {
				_, err := c.CreateCryptoKey(ctx, &kmspb.CreateCryptoKeyRequest{
					Parent:      keyRingName,
					CryptoKeyId: cryptoKeyID,
					CryptoKey: &kmspb.CryptoKey{
						Purpose: kmspb.CryptoKey_ENCRYPT_DECRYPT,
					},
				})
				return err
			},
		},
		{
			name: "GetCryptoKey",
			call: func(ctx context.Context, c *kms.KeyManagementClient) error {
				_, err := c.GetCryptoKey(ctx, &kmspb.GetCryptoKeyRequest{Name: cryptoKeyName})
				return err
			},
		},
		{
			name: "UpdateCryptoKey",
			call: func(ctx context.Context, c *kms.KeyManagementClient) error {
				_, err := c.UpdateCryptoKey(ctx, &kmspb.UpdateCryptoKeyRequest{
					CryptoKey: &kmspb.CryptoKey{
						Name:   cryptoKeyName,
						Labels: map[string]string{"owner": "stackyard"},
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"labels"}},
				})
				return err
			},
		},
		{
			name: "ListCryptoKeyVersions",
			call: func(ctx context.Context, c *kms.KeyManagementClient) error {
				it := c.ListCryptoKeyVersions(ctx, &kmspb.ListCryptoKeyVersionsRequest{
					Parent:   cryptoKeyName,
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
			name: "CreateCryptoKeyVersion",
			call: func(ctx context.Context, c *kms.KeyManagementClient) error {
				_, err := c.CreateCryptoKeyVersion(ctx, &kmspb.CreateCryptoKeyVersionRequest{
					Parent:           cryptoKeyName,
					CryptoKeyVersion: &kmspb.CryptoKeyVersion{},
				})
				return err
			},
		},
		{
			name: "GetCryptoKeyVersion",
			call: func(ctx context.Context, c *kms.KeyManagementClient) error {
				_, err := c.GetCryptoKeyVersion(ctx, &kmspb.GetCryptoKeyVersionRequest{Name: cryptoKeyVersionName})
				return err
			},
		},
		{
			name: "UpdateCryptoKeyVersion",
			call: func(ctx context.Context, c *kms.KeyManagementClient) error {
				_, err := c.UpdateCryptoKeyVersion(ctx, &kmspb.UpdateCryptoKeyVersionRequest{
					CryptoKeyVersion: &kmspb.CryptoKeyVersion{
						Name:  cryptoKeyVersionName,
						State: kmspb.CryptoKeyVersion_DISABLED,
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"state"}},
				})
				return err
			},
		},
		{
			name: "UpdateCryptoKeyPrimaryVersion",
			call: func(ctx context.Context, c *kms.KeyManagementClient) error {
				_, err := c.UpdateCryptoKeyPrimaryVersion(ctx, &kmspb.UpdateCryptoKeyPrimaryVersionRequest{
					Name:               cryptoKeyName,
					CryptoKeyVersionId: cryptoKeyVersionID,
				})
				return err
			},
		},
		{
			name: "DestroyCryptoKeyVersion",
			call: func(ctx context.Context, c *kms.KeyManagementClient) error {
				_, err := c.DestroyCryptoKeyVersion(ctx, &kmspb.DestroyCryptoKeyVersionRequest{Name: cryptoKeyVersionName})
				return err
			},
		},
		{
			name: "RestoreCryptoKeyVersion",
			call: func(ctx context.Context, c *kms.KeyManagementClient) error {
				_, err := c.RestoreCryptoKeyVersion(ctx, &kmspb.RestoreCryptoKeyVersionRequest{Name: cryptoKeyVersionName})
				return err
			},
		},
		{
			name: "GetPublicKey",
			call: func(ctx context.Context, c *kms.KeyManagementClient) error {
				_, err := c.GetPublicKey(ctx, &kmspb.GetPublicKeyRequest{Name: cryptoKeyVersionName})
				return err
			},
		},
		{
			name: "ListImportJobs",
			call: func(ctx context.Context, c *kms.KeyManagementClient) error {
				it := c.ListImportJobs(ctx, &kmspb.ListImportJobsRequest{
					Parent:   keyRingName,
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
			name: "CreateImportJob",
			call: func(ctx context.Context, c *kms.KeyManagementClient) error {
				_, err := c.CreateImportJob(ctx, &kmspb.CreateImportJobRequest{
					Parent:      keyRingName,
					ImportJobId: importJobID,
					ImportJob: &kmspb.ImportJob{
						ImportMethod:    kmspb.ImportJob_RSA_OAEP_3072_SHA1_AES_256,
						ProtectionLevel: kmspb.ProtectionLevel_HSM,
					},
				})
				return err
			},
		},
		{
			name: "GetImportJob",
			call: func(ctx context.Context, c *kms.KeyManagementClient) error {
				_, err := c.GetImportJob(ctx, &kmspb.GetImportJobRequest{Name: importJobName})
				return err
			},
		},
		{
			name: "ImportCryptoKeyVersion",
			call: func(ctx context.Context, c *kms.KeyManagementClient) error {
				_, err := c.ImportCryptoKeyVersion(ctx, &kmspb.ImportCryptoKeyVersionRequest{
					Parent:    cryptoKeyName,
					Algorithm: kmspb.CryptoKeyVersion_RSA_SIGN_PSS_2048_SHA256,
					ImportJob: importJobName,
					WrappedKey: []byte(
						"wrapped-key-material",
					),
				})
				return err
			},
		},
		{
			name: "Encrypt",
			call: func(ctx context.Context, c *kms.KeyManagementClient) error {
				_, err := c.Encrypt(ctx, &kmspb.EncryptRequest{
					Name:      cryptoKeyName,
					Plaintext: []byte("hello"),
				})
				return err
			},
		},
		{
			name: "Decrypt",
			call: func(ctx context.Context, c *kms.KeyManagementClient) error {
				_, err := c.Decrypt(ctx, &kmspb.DecryptRequest{
					Name:       cryptoKeyName,
					Ciphertext: []byte("ciphertext"),
				})
				return err
			},
		},
		{
			name: "RawEncrypt",
			call: func(ctx context.Context, c *kms.KeyManagementClient) error {
				_, err := c.RawEncrypt(ctx, &kmspb.RawEncryptRequest{
					Name:      cryptoKeyVersionName,
					Plaintext: []byte("plaintext"),
				})
				return err
			},
		},
		{
			name: "RawDecrypt",
			call: func(ctx context.Context, c *kms.KeyManagementClient) error {
				_, err := c.RawDecrypt(ctx, &kmspb.RawDecryptRequest{
					Name:                 cryptoKeyVersionName,
					Ciphertext:           []byte("ciphertext"),
					InitializationVector: []byte("0123456789abcdef"),
				})
				return err
			},
		},
		{
			name: "AsymmetricSign",
			call: func(ctx context.Context, c *kms.KeyManagementClient) error {
				_, err := c.AsymmetricSign(ctx, &kmspb.AsymmetricSignRequest{
					Name: cryptoKeyVersionName,
					Data: []byte("payload"),
				})
				return err
			},
		},
		{
			name: "AsymmetricDecrypt",
			call: func(ctx context.Context, c *kms.KeyManagementClient) error {
				_, err := c.AsymmetricDecrypt(ctx, &kmspb.AsymmetricDecryptRequest{
					Name:       cryptoKeyVersionName,
					Ciphertext: []byte("ciphertext"),
				})
				return err
			},
		},
		{
			name: "MacSign",
			call: func(ctx context.Context, c *kms.KeyManagementClient) error {
				_, err := c.MacSign(ctx, &kmspb.MacSignRequest{
					Name: cryptoKeyVersionName,
					Data: []byte("payload"),
				})
				return err
			},
		},
		{
			name: "MacVerify",
			call: func(ctx context.Context, c *kms.KeyManagementClient) error {
				_, err := c.MacVerify(ctx, &kmspb.MacVerifyRequest{
					Name: cryptoKeyVersionName,
					Data: []byte("payload"),
					Mac:  []byte("mac"),
				})
				return err
			},
		},
		{
			name: "GenerateRandomBytes",
			call: func(ctx context.Context, c *kms.KeyManagementClient) error {
				_, err := c.GenerateRandomBytes(ctx, &kmspb.GenerateRandomBytesRequest{
					Location:    locationName,
					LengthBytes: 32,
				})
				return err
			},
		},
		{
			name: "GetLocation",
			call: func(ctx context.Context, c *kms.KeyManagementClient) error {
				_, err := c.GetLocation(ctx, &locationpb.GetLocationRequest{Name: locationName})
				return err
			},
		},
		{
			name: "ListLocations",
			call: func(ctx context.Context, c *kms.KeyManagementClient) error {
				it := c.ListLocations(ctx, &locationpb.ListLocationsRequest{
					Name:     projectName,
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
			name: "GetIamPolicy",
			call: func(ctx context.Context, c *kms.KeyManagementClient) error {
				_, err := c.GetIamPolicy(ctx, &iampb.GetIamPolicyRequest{Resource: cryptoKeyName})
				return err
			},
		},
		{
			name: "SetIamPolicy",
			call: func(ctx context.Context, c *kms.KeyManagementClient) error {
				_, err := c.SetIamPolicy(ctx, &iampb.SetIamPolicyRequest{
					Resource: cryptoKeyName,
					Policy:   &iampb.Policy{},
				})
				return err
			},
		},
		{
			name: "TestIamPermissions",
			call: func(ctx context.Context, c *kms.KeyManagementClient) error {
				_, err := c.TestIamPermissions(ctx, &iampb.TestIamPermissionsRequest{
					Resource:    cryptoKeyName,
					Permissions: []string{"cloudkms.cryptoKeys.get"},
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
		fmt.Fprintf(os.Stderr, "warning: close kms client: %v\n", err)
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
