// Command ecr-public pushes an image to Stackyard's Amazon ECR Public
// registry: create a repository with gallery catalog data, upload a config
// blob and a layer, put the manifest, then read it back and clean up.
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
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

const repo = "stackyard/hello"

var (
	endpoint = getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	creds    = credentials.NewStaticCredentialsProvider(getenv("AWS_ACCESS_KEY_ID", "stackyard"), getenv("AWS_SECRET_ACCESS_KEY", "stackyard"), "")
)

func main() {
	fmt.Printf("Stackyard ECR Public client using %s\n", endpoint)

	created := call("CreateRepository", map[string]any{
		"repositoryName": repo,
		"catalogData":    map[string]any{"description": "hello world image", "architectures": []string{"x86-64"}, "operatingSystems": []string{"Linux"}},
	})
	fmt.Println("repository:", created["repository"].(map[string]any)["repositoryUri"])

	config := pushLayer([]byte(`{}`))
	layer := pushLayer([]byte("hello from stackyard"))
	manifest, _ := json.Marshal(map[string]any{
		"schemaVersion": 2,
		"mediaType":     "application/vnd.docker.distribution.manifest.v2+json",
		"config":        map[string]any{"mediaType": "application/vnd.docker.container.image.v1+json", "size": 2, "digest": config},
		"layers":        []any{map[string]any{"mediaType": "application/vnd.docker.image.rootfs.diff.tar.gzip", "size": 20, "digest": layer}},
	})
	image := call("PutImage", map[string]any{"repositoryName": repo, "imageManifest": string(manifest), "imageTag": "v1"})
	fmt.Println("pushed:", image["image"].(map[string]any)["imageId"])

	tags := call("DescribeImageTags", map[string]any{"repositoryName": repo})["imageTagDetails"].([]any)
	if len(tags) != 1 {
		exitf("expected one tag, got %v", tags)
	}
	registries := call("DescribeRegistries", map[string]any{})["registries"].([]any)
	fmt.Println("registry:", registries[0].(map[string]any)["registryUri"])

	call("DeleteRepository", map[string]any{"repositoryName": repo, "force": true})
	fmt.Println("ECR Public example finished")
}

// pushLayer uploads data as one layer and returns its digest.
func pushLayer(data []byte) string {
	id := call("InitiateLayerUpload", map[string]any{"repositoryName": repo})["uploadId"]
	call("UploadLayerPart", map[string]any{
		"repositoryName": repo, "uploadId": id, "partFirstByte": 0, "partLastByte": len(data) - 1,
		"layerPartBlob": base64.StdEncoding.EncodeToString(data),
	})
	sum := sha256.Sum256(data)
	digest := "sha256:" + hex.EncodeToString(sum[:])
	call("CompleteLayerUpload", map[string]any{"repositoryName": repo, "uploadId": id, "layerDigests": []string{digest}})
	return digest
}

// call sends a SigV4-signed ECR Public request (us-east-1 only) and fails on a non-2xx reply.
func call(op string, payload map[string]any) map[string]any {
	body, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, endpoint+"/", bytes.NewReader(body))
	if err != nil {
		exitf("%s: %v", op, err)
	}
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "SpencerFrontendService."+op)
	ctx := context.Background()
	c, err := creds.Retrieve(ctx)
	if err != nil {
		exitf("credentials: %v", err)
	}
	hash := sha256.Sum256(body)
	if err := v4.NewSigner().SignHTTP(ctx, aws.Credentials(c), req, hex.EncodeToString(hash[:]), "ecr-public", "us-east-1", time.Now()); err != nil {
		exitf("sign %s: %v", op, err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode/100 != 2 {
		exitf("%s: %d %s", op, resp.StatusCode, raw)
	}
	out := map[string]any{}
	_ = json.Unmarshal(raw, &out)
	return out
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
