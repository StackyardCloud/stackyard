package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	region := getenv("AWS_REGION", "us-east-1")

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard Storage Gateway advanced client using %s\n", endpoint)

	actions := []string{
		"ActivateGateway",
		"AddCache",
		"AddTagsToResource",
		"AddUploadBuffer",
		"AddWorkingStorage",
		"AssignTapePool",
		"AssociateFileSystem",
		"AttachVolume",
		"CancelArchival",
		"CancelCacheReport",
		"CancelRetrieval",
		"CreateCachediSCSIVolume",
		"CreateNFSFileShare",
		"CreateSMBFileShare",
		"CreateSnapshot",
		"CreateSnapshotFromVolumeRecoveryPoint",
		"CreateStorediSCSIVolume",
		"CreateTapePool",
		"CreateTapeWithBarcode",
		"CreateTapes",
		"DeleteAutomaticTapeCreationPolicy",
		"DeleteBandwidthRateLimit",
		"DeleteCacheReport",
		"DeleteChapCredentials",
		"DeleteFileShare",
		"DeleteGateway",
		"DeleteSnapshotSchedule",
		"DeleteTape",
		"DeleteTapeArchive",
		"DeleteTapePool",
		"DeleteVolume",
		"DescribeAvailabilityMonitorTest",
		"DescribeBandwidthRateLimit",
		"DescribeBandwidthRateLimitSchedule",
		"DescribeCache",
		"DescribeCacheReport",
		"DescribeCachediSCSIVolumes",
		"DescribeChapCredentials",
		"DescribeFileSystemAssociations",
		"DescribeGatewayInformation",
		"DescribeMaintenanceStartTime",
		"DescribeNFSFileShares",
		"DescribeSMBFileShares",
		"DescribeSMBSettings",
		"DescribeSnapshotSchedule",
		"DescribeStorediSCSIVolumes",
		"DescribeTapeArchives",
		"DescribeTapeRecoveryPoints",
		"DescribeTapes",
		"DescribeUploadBuffer",
		"DescribeVTLDevices",
		"DescribeWorkingStorage",
		"DetachVolume",
		"DisableGateway",
		"DisassociateFileSystem",
		"EvictFilesFailingUpload",
		"JoinDomain",
		"ListAutomaticTapeCreationPolicies",
		"ListCacheReports",
		"ListFileShares",
		"ListFileSystemAssociations",
		"ListGateways",
		"ListLocalDisks",
		"ListTagsForResource",
		"ListTapePools",
		"ListTapes",
		"ListVolumeInitiators",
		"ListVolumeRecoveryPoints",
		"ListVolumes",
		"NotifyWhenUploaded",
		"RefreshCache",
		"RemoveTagsFromResource",
		"ResetCache",
		"RetrieveTapeArchive",
		"RetrieveTapeRecoveryPoint",
		"SetLocalConsolePassword",
		"SetSMBGuestPassword",
		"ShutdownGateway",
		"StartAvailabilityMonitorTest",
		"StartCacheReport",
		"StartGateway",
		"UpdateAutomaticTapeCreationPolicy",
		"UpdateBandwidthRateLimit",
		"UpdateBandwidthRateLimitSchedule",
		"UpdateChapCredentials",
		"UpdateFileSystemAssociation",
		"UpdateGatewayInformation",
		"UpdateGatewaySoftwareNow",
		"UpdateMaintenanceStartTime",
		"UpdateNFSFileShare",
		"UpdateSMBFileShare",
		"UpdateSMBFileShareVisibility",
		"UpdateSMBLocalGroups",
		"UpdateSMBSecurityStrategy",
		"UpdateSnapshotSchedule",
		"UpdateVTLDeviceType",
	}

	for _, action := range actions {
		status, body, err := storageGatewayRequest(ctx, endpoint, region, creds, action, map[string]any{})
		mustSuccess(action, status, body, err)
		fmt.Printf("%s returned %d\n", action, status)
	}

	fmt.Println("Done.")
}

func storageGatewayRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	action string,
	payload map[string]any,
) (int, []byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return 0, nil, err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		strings.TrimRight(endpoint, "/")+"/",
		bytes.NewReader(body),
	)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "StorageGateway_20130630."+action)

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "storagegateway", region, time.Now()); err != nil {
		return 0, nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, err
	}
	return resp.StatusCode, respBody, nil
}

func mustSuccess(action string, status int, body []byte, err error) {
	if err != nil {
		exitf("%s request failed: %v", action, err)
	}
	if status < 200 || status >= 300 {
		exitf("%s returned HTTP %d: %s", action, status, strings.TrimSpace(string(body)))
	}
}

func hashSHA256(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func getenv(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return def
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
