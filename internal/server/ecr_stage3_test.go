package server

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awsecr "github.com/aws/aws-sdk-go-v2/service/ecr"
	awsecrtypes "github.com/aws/aws-sdk-go-v2/service/ecr/types"
)

func TestECRStage3ImplementedActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	const repositoryName = "stage3-raw"
	layerBlob := []byte("hello-layer")
	layerDigest := sha256HexDigest(layerBlob)
	manifest := `{"schemaVersion":2}`

	resp := ecrRequest(t, ts, "CreateRepository", []byte(`{"repositoryName":"`+repositoryName+`"}`))
	assertStatus(t, resp, http.StatusOK)

	resp = ecrRequest(t, ts, "InitiateLayerUpload", []byte(`{"repositoryName":"`+repositoryName+`"}`))
	assertStatus(t, resp, http.StatusOK)
	var initiateOut struct {
		UploadID string `json:"uploadId"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &initiateOut); err != nil {
		t.Fatalf("unmarshal initiate layer upload: %v", err)
	}
	if initiateOut.UploadID == "" {
		t.Fatalf("expected upload id")
	}

	uploadBody := fmt.Sprintf(`{"repositoryName":"%s","uploadId":"%s","partFirstByte":0,"partLastByte":%d,"layerPartBlob":"%s"}`,
		repositoryName,
		initiateOut.UploadID,
		len(layerBlob)-1,
		base64.StdEncoding.EncodeToString(layerBlob),
	)
	resp = ecrRequest(t, ts, "UploadLayerPart", []byte(uploadBody))
	assertStatus(t, resp, http.StatusOK)

	completeBody := fmt.Sprintf(`{"repositoryName":"%s","uploadId":"%s","layerDigests":["%s"]}`,
		repositoryName,
		initiateOut.UploadID,
		layerDigest,
	)
	resp = ecrRequest(t, ts, "CompleteLayerUpload", []byte(completeBody))
	assertStatus(t, resp, http.StatusOK)

	putImagePayload, err := json.Marshal(map[string]any{
		"repositoryName": repositoryName,
		"imageManifest":  manifest,
		"imageTag":       "v1",
	})
	if err != nil {
		t.Fatalf("marshal put image payload: %v", err)
	}
	resp = ecrRequest(t, ts, "PutImage", putImagePayload)
	assertStatus(t, resp, http.StatusOK)

	actions := []struct {
		name string
		body []byte
	}{
		{name: "BatchCheckLayerAvailability", body: []byte(`{"repositoryName":"` + repositoryName + `","layerDigests":["` + layerDigest + `"]}`)},
		{name: "BatchGetImage", body: []byte(`{"repositoryName":"` + repositoryName + `","imageIds":[{"imageTag":"v1"}]}`)},
		{name: "GetDownloadUrlForLayer", body: []byte(`{"repositoryName":"` + repositoryName + `","layerDigest":"` + layerDigest + `"}`)},
		{name: "ListImages", body: []byte(`{"repositoryName":"` + repositoryName + `","filter":{"tagStatus":"TAGGED"}}`)},
	}
	for _, action := range actions {
		resp = ecrRequest(t, ts, action.name, action.body)
		if resp.StatusCode == http.StatusNotImplemented {
			t.Fatalf("action %s returned not implemented", action.name)
		}
		assertStatus(t, resp, http.StatusOK)
	}
}

func TestECRStage3SDKPushPullPrimitives(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(testRegion),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(testAccessKey, testSecretKey, "")),
	)
	if err != nil {
		t.Fatalf("load aws config: %v", err)
	}

	client := awsecr.NewFromConfig(cfg, func(o *awsecr.Options) {
		o.BaseEndpoint = aws.String(ts.URL)
	})

	const repositoryName = "stage3-sdk"
	if _, err := client.CreateRepository(ctx, &awsecr.CreateRepositoryInput{RepositoryName: aws.String(repositoryName)}); err != nil {
		t.Fatalf("create repository: %v", err)
	}

	layerBlob := []byte("sdk-layer-content")
	layerDigest := sha256HexDigest(layerBlob)

	initiateOut, err := client.InitiateLayerUpload(ctx, &awsecr.InitiateLayerUploadInput{RepositoryName: aws.String(repositoryName)})
	if err != nil {
		t.Fatalf("initiate layer upload: %v", err)
	}
	if initiateOut.UploadId == nil || *initiateOut.UploadId == "" {
		t.Fatalf("expected upload id")
	}

	partLastByte := int64(len(layerBlob) - 1)
	uploadOut, err := client.UploadLayerPart(ctx, &awsecr.UploadLayerPartInput{
		RepositoryName: aws.String(repositoryName),
		UploadId:       initiateOut.UploadId,
		PartFirstByte:  aws.Int64(0),
		PartLastByte:   aws.Int64(partLastByte),
		LayerPartBlob:  layerBlob,
	})
	if err != nil {
		t.Fatalf("upload layer part: %v", err)
	}
	if uploadOut.LastByteReceived == nil || *uploadOut.LastByteReceived != partLastByte {
		t.Fatalf("unexpected last byte received: %v", uploadOut.LastByteReceived)
	}

	completeOut, err := client.CompleteLayerUpload(ctx, &awsecr.CompleteLayerUploadInput{
		RepositoryName: aws.String(repositoryName),
		UploadId:       initiateOut.UploadId,
		LayerDigests:   []string{layerDigest},
	})
	if err != nil {
		t.Fatalf("complete layer upload: %v", err)
	}
	if aws.ToString(completeOut.LayerDigest) != layerDigest {
		t.Fatalf("unexpected completed layer digest: %q", aws.ToString(completeOut.LayerDigest))
	}

	checkOut, err := client.BatchCheckLayerAvailability(ctx, &awsecr.BatchCheckLayerAvailabilityInput{
		RepositoryName: aws.String(repositoryName),
		LayerDigests:   []string{layerDigest, "sha256:1111111111111111111111111111111111111111111111111111111111111111"},
	})
	if err != nil {
		t.Fatalf("batch check layer availability: %v", err)
	}
	if len(checkOut.Layers) != 2 {
		t.Fatalf("expected two layer entries, got %d", len(checkOut.Layers))
	}
	if checkOut.Layers[0].LayerAvailability != awsecrtypes.LayerAvailabilityAvailable {
		t.Fatalf("expected first layer to be available")
	}

	manifest := `{"schemaVersion":2,"mediaType":"application/vnd.docker.distribution.manifest.v2+json"}`
	putImageOut, err := client.PutImage(ctx, &awsecr.PutImageInput{
		RepositoryName: aws.String(repositoryName),
		ImageManifest:  aws.String(manifest),
		ImageTag:       aws.String("v1"),
	})
	if err != nil {
		t.Fatalf("put image: %v", err)
	}
	if putImageOut.Image == nil || putImageOut.Image.ImageId == nil || putImageOut.Image.ImageId.ImageDigest == nil {
		t.Fatalf("expected image digest in put image output")
	}

	batchImageOut, err := client.BatchGetImage(ctx, &awsecr.BatchGetImageInput{
		RepositoryName: aws.String(repositoryName),
		ImageIds: []awsecrtypes.ImageIdentifier{
			{ImageTag: aws.String("v1")},
		},
	})
	if err != nil {
		t.Fatalf("batch get image: %v", err)
	}
	if len(batchImageOut.Images) != 1 {
		t.Fatalf("expected one image, got %d", len(batchImageOut.Images))
	}

	downloadOut, err := client.GetDownloadUrlForLayer(ctx, &awsecr.GetDownloadUrlForLayerInput{
		RepositoryName: aws.String(repositoryName),
		LayerDigest:    aws.String(layerDigest),
	})
	if err != nil {
		t.Fatalf("get download url for layer: %v", err)
	}
	if !strings.Contains(aws.ToString(downloadOut.DownloadUrl), "/blobs/") {
		t.Fatalf("expected blob URL, got %q", aws.ToString(downloadOut.DownloadUrl))
	}

	listOut, err := client.ListImages(ctx, &awsecr.ListImagesInput{
		RepositoryName: aws.String(repositoryName),
		Filter:         &awsecrtypes.ListImagesFilter{TagStatus: awsecrtypes.TagStatusTagged},
	})
	if err != nil {
		t.Fatalf("list images: %v", err)
	}
	if len(listOut.ImageIds) == 0 {
		t.Fatalf("expected tagged image ids")
	}
}

func sha256HexDigest(payload []byte) string {
	sum := sha256.Sum256(payload)
	return "sha256:" + hex.EncodeToString(sum[:])
}
