// Command efs walks an Amazon EFS lifecycle against Stackyard: create a file
// system, add a mount target in the default subnet and an access point, set
// policies, then tear it all down.
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
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

type client struct {
	endpoint, region string
	creds            aws.CredentialsProvider
}

func main() {
	c := client{
		endpoint: strings.TrimRight(getenv("STACKYARD_ENDPOINT", "http://localhost:4566"), "/"),
		region:   getenv("AWS_REGION", "us-east-1"),
		creds: credentials.NewStaticCredentialsProvider(
			getenv("AWS_ACCESS_KEY_ID", "stackyard"), getenv("AWS_SECRET_ACCESS_KEY", "stackyard"), ""),
	}
	subnet := getenv("STACKYARD_EFS_SUBNET_ID", "subnet-00000001")
	fmt.Printf("Stackyard EFS client using %s\n", c.endpoint)

	fs := c.call("CreateFileSystem", http.MethodPost, "/file-systems", map[string]any{
		"CreationToken":  fmt.Sprintf("stackyard-efs-%d", time.Now().UnixNano()),
		"Encrypted":      true,
		"ThroughputMode": "elastic",
		"Tags":           []map[string]string{{"Key": "Name", "Value": "stackyard-example"}},
	})
	fsID, _ := fs["FileSystemId"].(string)
	fmt.Printf("created file system %s (%v)\n", fsID, fs["LifeCycleState"])

	mt := c.call("CreateMountTarget", http.MethodPost, "/mount-targets", map[string]any{"FileSystemId": fsID, "SubnetId": subnet})
	mtID, _ := mt["MountTargetId"].(string)
	fmt.Printf("mount target %s at %v in %v\n", mtID, mt["IpAddress"], mt["AvailabilityZoneName"])

	ap := c.call("CreateAccessPoint", http.MethodPost, "/access-points", map[string]any{
		"ClientToken":  fmt.Sprintf("stackyard-ap-%d", time.Now().UnixNano()),
		"FileSystemId": fsID,
		"PosixUser":    map[string]any{"Uid": 1000, "Gid": 1000},
		"RootDirectory": map[string]any{"Path": "/app", "CreationInfo": map[string]any{
			"OwnerUid": 1000, "OwnerGid": 1000, "Permissions": "750"}},
	})
	apID, _ := ap["AccessPointId"].(string)
	fmt.Printf("access point %s rooted at /app\n", apID)

	c.call("PutLifecycleConfiguration", http.MethodPut, "/file-systems/"+fsID+"/lifecycle-configuration", map[string]any{
		"LifecyclePolicies": []map[string]string{{"TransitionToIA": "AFTER_30_DAYS"}, {"TransitionToArchive": "AFTER_90_DAYS"}},
	})
	c.call("PutBackupPolicy", http.MethodPut, "/file-systems/"+fsID+"/backup-policy", map[string]any{"BackupPolicy": map[string]string{"Status": "ENABLED"}})
	policy := `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"AWS":"*"},"Action":["elasticfilesystem:ClientMount","elasticfilesystem:ClientWrite"],"Condition":{"Bool":{"aws:SecureTransport":"true"}}}]}`
	c.call("PutFileSystemPolicy", http.MethodPut, "/file-systems/"+fsID+"/policy", map[string]any{"Policy": policy})

	listed := c.call("DescribeMountTargets", http.MethodGet, "/mount-targets?FileSystemId="+fsID, nil)
	fmt.Printf("file system has %d mount target(s)\n", len(listed["MountTargets"].([]any)))

	c.call("DeleteAccessPoint", http.MethodDelete, "/access-points/"+apID, nil)
	c.call("DeleteMountTarget", http.MethodDelete, "/mount-targets/"+mtID, nil)
	c.call("DeleteFileSystem", http.MethodDelete, "/file-systems/"+fsID, nil)
	fmt.Println("EFS example completed")
}

// call sends a signed EFS request and exits unless it returns 2xx.
func (c client) call(op, method, path string, payload any) map[string]any {
	var body []byte
	if payload != nil {
		var err error
		if body, err = json.Marshal(payload); err != nil {
			exitf("%s: %v", op, err)
		}
	}
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, method, c.endpoint+"/2015-02-01"+path, bytes.NewReader(body))
	if err != nil {
		exitf("%s: %v", op, err)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	creds, err := c.creds.Retrieve(ctx)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	sum := sha256.Sum256(body)
	if err := v4.NewSigner().SignHTTP(ctx, creds, req, hex.EncodeToString(sum[:]), "elasticfilesystem", c.region, time.Now()); err != nil {
		exitf("%s: %v", op, err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		exitf("%s: HTTP %d: %s", op, resp.StatusCode, raw)
	}
	out := map[string]any{}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			exitf("%s: decode: %v", op, err)
		}
	}
	return out
}

func getenv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
