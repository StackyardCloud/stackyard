// Command backup-gateway runs AWS Backup gateway against Stackyard: activate
// a gateway, import a VMware hypervisor, wait for it to come ONLINE and sync
// its virtual machines, associate the gateway, set a bandwidth schedule, then
// back up a VM with AWS Backup and read its LastBackupDate.
//
// Hypervisor credentials (omitted here) are never stored or returned. The
// SDK has no Backup gateway module in this repository's go.mod, so requests
// are signed by hand.
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

const signingName = "backup-gateway"

func main() {
	c := newClient()
	fmt.Printf("Stackyard Backup gateway client using %s\n", c.endpoint)
	run := time.Now().UnixNano()

	gw := c.call("CreateGateway", map[string]any{"ActivationKey": fmt.Sprintf("ACT-%d", run), "GatewayDisplayName": "branch-gw", "GatewayType": "BACKUP_VM"})["GatewayArn"].(string)
	fmt.Println("gateway", gw)
	hv := c.call("ImportHypervisorConfiguration", map[string]any{
		"Name": "vcenter", "Host": fmt.Sprintf("vcenter-%d.example.com", run),
	})["HypervisorArn"].(string)
	for {
		h := c.call("GetHypervisor", map[string]any{"HypervisorArn": hv})["Hypervisor"].(map[string]any)
		fmt.Printf("hypervisor %v, metadata sync %v\n", h["State"], h["LatestMetadataSyncStatus"])
		if h["LatestMetadataSyncStatus"] == "SUCCEEDED" {
			break
		}
	}
	c.call("AssociateGatewayToServer", map[string]any{"GatewayArn": gw, "ServerArn": hv})
	c.call("PutBandwidthRateLimitSchedule", map[string]any{"GatewayArn": gw, "BandwidthRateLimitIntervals": []any{map[string]any{
		"AverageUploadRateLimitInBitsPerSec": 104857600, "StartHourOfDay": 9, "StartMinuteOfHour": 0, "EndHourOfDay": 17, "EndMinuteOfHour": 0,
		"DaysOfWeek": []any{1, 2, 3, 4, 5}}}})
	g := c.call("GetGateway", map[string]any{"GatewayArn": gw})["Gateway"].(map[string]any)
	fmt.Printf("gateway serves hypervisor %v, software %v\n", g["HypervisorId"], g["SoftwareVersion"])

	vms := c.call("ListVirtualMachines", map[string]any{"HypervisorArn": hv})["VirtualMachines"].([]any)
	vm := vms[0].(map[string]any)
	fmt.Printf("%d VMs; backing up %v\n", len(vms), vm["Name"])

	vault := fmt.Sprintf("vms-%d", run%1000000)
	c.backup("CreateBackupVault", http.MethodPut, "/backup-vaults/"+vault, map[string]any{})
	job := c.backup("StartBackupJob", http.MethodPut, "/backup-jobs", map[string]any{
		"BackupVaultName": vault, "IamRoleArn": "arn:aws:iam::123456789012:role/backup", "ResourceArn": vm["ResourceArn"],
	})["BackupJobId"].(string)
	for {
		state := c.backup("DescribeBackupJob", http.MethodGet, "/backup-jobs/"+job, nil)["State"]
		fmt.Println("backup job:", state)
		if state == "COMPLETED" {
			break
		}
	}
	got := c.call("GetVirtualMachine", map[string]any{"ResourceArn": vm["ResourceArn"]})["VirtualMachine"].(map[string]any)
	fmt.Printf("%v last backed up at %v\n", got["Name"], got["LastBackupDate"])

	c.call("DisassociateGatewayFromServer", map[string]any{"GatewayArn": gw})
	c.call("DeleteHypervisor", map[string]any{"HypervisorArn": hv})
	c.call("DeleteGateway", map[string]any{"GatewayArn": gw})
	fmt.Println("cleaned up")
}

// call sends one Backup gateway JSON 1.0 operation.
func (c client) call(op string, payload any) map[string]any {
	body, err := json.Marshal(payload)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	return decode(op, c.send(op, http.MethodPost, signingName, c.endpoint+"/", body, map[string]string{
		"Content-Type": "application/x-amz-json-1.0", "X-Amz-Target": "BackupOnPremises_v20210101." + op,
	}))
}

// backup sends one AWS Backup REST operation.
func (c client) backup(op, method, path string, payload any) map[string]any {
	var body []byte
	if payload != nil {
		body, _ = json.Marshal(payload)
	}
	return decode(op, c.send(op, method, "backup", c.endpoint+path, body, map[string]string{"Content-Type": "application/json"}))
}

func decode(op string, raw []byte) map[string]any {
	out := map[string]any{}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			exitf("%s: decode: %v", op, err)
		}
	}
	return out
}

type client struct {
	endpoint, region string
	creds            aws.CredentialsProvider
}

func newClient() client {
	return client{
		endpoint: strings.TrimRight(getenv("STACKYARD_ENDPOINT", "http://localhost:4566"), "/"),
		region:   getenv("AWS_REGION", "us-east-1"),
		creds: credentials.NewStaticCredentialsProvider(
			getenv("AWS_ACCESS_KEY_ID", "stackyard"), getenv("AWS_SECRET_ACCESS_KEY", "stackyard"), ""),
	}
}

// send signs a request for service and exits unless it returns 2xx.
func (c client) send(op, method, service, target string, body []byte, headers map[string]string) []byte {
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, method, target, bytes.NewReader(body))
	if err != nil {
		exitf("%s: %v", op, err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	creds, err := c.creds.Retrieve(ctx)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	sum := sha256.Sum256(body)
	payloadHash := hex.EncodeToString(sum[:])
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)
	if err := v4.NewSigner().SignHTTP(ctx, creds, req, payloadHash, service, c.region, time.Now()); err != nil {
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
	return raw
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
