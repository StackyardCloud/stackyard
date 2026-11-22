// Command eks-auth exchanges an EKS Pod Identity service-account token for
// temporary credentials against Stackyard: it creates an EKS cluster and a pod
// identity association, then calls AssumeRoleForPodIdentity.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
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

func main() {
	endpoint := strings.TrimRight(getenv("STACKYARD_ENDPOINT", "http://localhost:4566"), "/")
	region := getenv("AWS_REGION", "us-east-1")
	cluster := getenv("STACKYARD_EKS_CLUSTER", "eks-auth-demo")
	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(getenv("AWS_ACCESS_KEY_ID", "stackyard"), getenv("AWS_SECRET_ACCESS_KEY", "stackyard"), "")

	call(ctx, creds, region, "eks", http.MethodPost, endpoint+"/clusters", map[string]any{
		"name":               cluster,
		"roleArn":            "arn:aws:iam::123456789012:role/eks-cluster",
		"resourcesVpcConfig": map[string]any{"subnetIds": []string{"subnet-12345678"}},
	}, nil)
	call(ctx, creds, region, "eks", http.MethodPost, endpoint+"/clusters/"+cluster+"/pod-identity-associations", map[string]any{
		"namespace":      "apps",
		"serviceAccount": "web",
		"roleArn":        "arn:aws:iam::123456789012:role/web-pods",
	}, nil)

	var out struct {
		Subject struct {
			Namespace      string `json:"namespace"`
			ServiceAccount string `json:"serviceAccount"`
		} `json:"subject"`
		AssumedRoleUser struct {
			Arn string `json:"arn"`
		} `json:"assumedRoleUser"`
		Credentials struct {
			AccessKeyID string  `json:"accessKeyId"`
			Expiration  float64 `json:"expiration"`
		} `json:"credentials"`
	}
	call(ctx, creds, region, "eks-auth", http.MethodPost, endpoint+"/clusters/"+cluster+"/assume-role-for-pod-identity", map[string]any{
		"token": token("apps", "web"),
	}, &out)
	if out.Subject.ServiceAccount != "web" || out.Credentials.AccessKeyID == "" {
		exitf("unexpected AssumeRoleForPodIdentity response: %+v", out)
	}
	fmt.Printf("assumed %s for %s/%s (access key %s, expires %s)\n", out.AssumedRoleUser.Arn,
		out.Subject.Namespace, out.Subject.ServiceAccount, out.Credentials.AccessKeyID,
		time.Unix(int64(out.Credentials.Expiration), 0).UTC().Format(time.RFC3339))
}

// token builds an unsigned projected service-account token; Stackyard checks
// its claims, not its signature.
func token(namespace, serviceAccount string) string {
	enc := func(v any) string {
		b, _ := json.Marshal(v)
		return base64.RawURLEncoding.EncodeToString(b)
	}
	return enc(map[string]string{"alg": "RS256", "typ": "JWT"}) + "." + enc(map[string]any{
		"aud": []string{"pods.eks.amazonaws.com"},
		"exp": time.Now().Add(time.Hour).Unix(),
		"sub": "system:serviceaccount:" + namespace + ":" + serviceAccount,
	}) + ".c2lnbmF0dXJl"
}

func call(ctx context.Context, creds aws.CredentialsProvider, region, service, method, url string, payload, out any) {
	body, err := json.Marshal(payload)
	if err != nil {
		exitf("encode: %v", err)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		exitf("request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	value, err := creds.Retrieve(ctx)
	if err != nil {
		exitf("credentials: %v", err)
	}
	sum := sha256.Sum256(body)
	if err := v4.NewSigner().SignHTTP(ctx, value, req, hex.EncodeToString(sum[:]), service, region, time.Now()); err != nil {
		exitf("sign: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		exitf("%s %s: %v", method, url, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode/100 != 2 {
		exitf("%s %s: HTTP %d %s", method, url, resp.StatusCode, raw)
	}
	if out != nil {
		if err := json.Unmarshal(raw, out); err != nil {
			exitf("decode %s: %v", raw, err)
		}
	}
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
