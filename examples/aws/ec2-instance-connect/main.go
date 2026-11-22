// EC2 Instance Connect example: launches an instance, pushes an SSH public key
// to it, enables serial console access and pushes a serial console key, then
// reads the pushed keys back from Stackyard's inspection endpoint.
package main

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	region := getenv("AWS_REGION", "us-east-1")
	creds := credentials.NewStaticCredentialsProvider(getenv("AWS_ACCESS_KEY_ID", "stackyard"), getenv("AWS_SECRET_ACCESS_KEY", "stackyard"), "")
	ctx := context.Background()
	fmt.Printf("Stackyard EC2 Instance Connect client using %s\n", endpoint)

	do := func(service, method, path, contentType string, headers map[string]string, body []byte) []byte {
		req, err := http.NewRequestWithContext(ctx, method, endpoint+path, bytes.NewReader(body))
		if err != nil {
			exitf("%s %s: %v", method, path, err)
		}
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		sum := sha256.Sum256(body)
		hash := hex.EncodeToString(sum[:])
		req.Header.Set("X-Amz-Content-Sha256", hash)
		c, err := creds.Retrieve(ctx)
		if err != nil {
			exitf("credentials: %v", err)
		}
		if err := v4.NewSigner().SignHTTP(ctx, aws.Credentials(c), req, hash, service, region, time.Now()); err != nil {
			exitf("sign: %v", err)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			exitf("%s %s: %v", method, path, err)
		}
		defer resp.Body.Close()
		raw, _ := io.ReadAll(resp.Body)
		if resp.StatusCode/100 != 2 {
			exitf("%s %s returned %d: %s", method, path, resp.StatusCode, raw)
		}
		return raw
	}
	ec2 := func(action string, params url.Values) []byte {
		params.Set("Action", action)
		params.Set("Version", "2016-11-15")
		return do("ec2", http.MethodPost, "/", "application/x-www-form-urlencoded", nil, []byte(params.Encode()))
	}
	eic := func(op string, payload map[string]any) map[string]any {
		body, _ := json.Marshal(payload)
		raw := do("ec2-instance-connect", http.MethodPost, "/", "application/x-amz-json-1.1", map[string]string{"X-Amz-Target": "AWSEC2InstanceConnectService." + op}, body)
		out := map[string]any{}
		_ = json.Unmarshal(raw, &out)
		return out
	}

	run := ec2("RunInstances", url.Values{"ImageId": {"ami-12345678"}, "MinCount": {"1"}, "MaxCount": {"1"}, "InstanceType": {"t3.micro"}})
	m := regexp.MustCompile(`<instanceId>(i-[0-9a-f]+)</instanceId>`).FindSubmatch(run)
	if m == nil {
		exitf("RunInstances returned no instance id: %s", run)
	}
	instanceID := string(m[1])
	fmt.Printf("launched %s\n", instanceID)

	key := publicKey()
	if out := eic("SendSSHPublicKey", map[string]any{"InstanceId": instanceID, "InstanceOSUser": "ec2-user", "SSHPublicKey": key}); out["Success"] != true {
		exitf("SendSSHPublicKey: %v", out)
	}
	fmt.Println("pushed an SSH key for ec2-user")

	ec2("EnableSerialConsoleAccess", url.Values{})
	if out := eic("SendSerialConsoleSSHPublicKey", map[string]any{"InstanceId": instanceID, "SSHPublicKey": key, "SerialPort": 0}); out["Success"] != true {
		exitf("SendSerialConsoleSSHPublicKey: %v", out)
	}
	fmt.Println("pushed a serial console key")

	var keys struct{ Keys []map[string]any }
	raw := do("ec2-instance-connect", http.MethodGet, "/_stackyard/ec2instanceconnect/instances/"+instanceID+"/keys", "", nil, nil)
	if err := json.Unmarshal(raw, &keys); err != nil || len(keys.Keys) != 2 {
		exitf("pushed keys: %v %s", err, raw)
	}
	fmt.Printf("instance %s holds %d keys for the next 60 seconds\n", instanceID, len(keys.Keys))

	ec2("TerminateInstances", url.Values{"InstanceId.1": {instanceID}})
	fmt.Println("done")
}

// publicKey makes a fresh OpenSSH ssh-ed25519 public key line.
func publicKey() string {
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		exitf("generate key: %v", err)
	}
	blob := binary.BigEndian.AppendUint32(nil, 11)
	blob = append(blob, "ssh-ed25519"...)
	blob = binary.BigEndian.AppendUint32(blob, uint32(len(pub)))
	blob = append(blob, pub...)
	return "ssh-ed25519 " + base64.StdEncoding.EncodeToString(blob) + " example@stackyard"
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
