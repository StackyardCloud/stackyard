package server

import (
	"net/http"
	"strings"
	"testing"

	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	visionpb "google.golang.org/genproto/googleapis/cloud/vision/v1"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

func TestGCPStage4GRPCParity_Vision(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)

	restBatchResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.cloud.vision.v1.ImageAnnotator/BatchAnnotateImages", []byte(`{
		"parent":"projects/stackyard/locations/us-central1",
		"requests":[{
			"image":{"source":{"imageUri":"gs://stackyard-inputs/image-1.jpg"}},
			"features":[{"type":"LABEL_DETECTION"}]
		}]
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "vision",
	})
	if restBatchResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest vision batch annotate images, got %d body=%s", restBatchResp.StatusCode, string(providerContractBody(t, restBatchResp)))
	}
	restBatchBody := providerContractJSONMap(t, restBatchResp)
	restResponses, ok := restBatchBody["responses"].([]any)
	if !ok || len(restResponses) == 0 {
		t.Fatalf("expected non-empty rest responses array, got %#v", restBatchBody["responses"])
	}

	var grpcBatchResp visionpb.BatchAnnotateImagesResponse
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpVisionBatchAnnotateImagesMethod, &visionpb.BatchAnnotateImagesRequest{
		Parent: "projects/stackyard/locations/us-central1",
		Requests: []*visionpb.AnnotateImageRequest{
			{
				Image: &visionpb.Image{
					Source: &visionpb.ImageSource{ImageUri: "gs://stackyard-inputs/image-1.jpg"},
				},
				Features: []*visionpb.Feature{{Type: visionpb.Feature_LABEL_DETECTION}},
			},
		},
	}, &grpcBatchResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for vision batch annotate images, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(grpcBatchResp.GetResponses()) != len(restResponses) {
		t.Fatalf("expected grpc responses len %d to match rest %d", len(grpcBatchResp.GetResponses()), len(restResponses))
	}
	if len(grpcBatchResp.GetResponses()[0].GetLabelAnnotations()) == 0 {
		t.Fatalf("expected grpc label annotations in first response")
	}

	var grpcAsyncOp longrunningpb.Operation
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpVisionAsyncBatchAnnotateImagesMethod, &visionpb.AsyncBatchAnnotateImagesRequest{
		Parent: "projects/stackyard/locations/us-central1",
		Requests: []*visionpb.AnnotateImageRequest{
			{
				Image: &visionpb.Image{
					Source: &visionpb.ImageSource{ImageUri: "gs://stackyard-inputs/image-1.jpg"},
				},
				Features: []*visionpb.Feature{{Type: visionpb.Feature_LABEL_DETECTION}},
			},
		},
		OutputConfig: &visionpb.OutputConfig{
			GcsDestination: &visionpb.GcsDestination{Uri: "gs://stackyard-outputs/vision/"},
		},
	}, &grpcAsyncOp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for async batch annotate images, got %q message=%q", grpcStatus, grpcMessage)
	}
	if !grpcAsyncOp.GetDone() {
		t.Fatalf("expected async operation done=true")
	}
	var asyncMetadata visionpb.OperationMetadata
	if err := grpcAsyncOp.GetMetadata().UnmarshalTo(&asyncMetadata); err != nil {
		t.Fatalf("expected typed async metadata, got error: %v", err)
	}
	var asyncResponse visionpb.AsyncBatchAnnotateImagesResponse
	if err := grpcAsyncOp.GetResponse().UnmarshalTo(&asyncResponse); err != nil {
		t.Fatalf("expected typed async response, got error: %v", err)
	}
	if strings.TrimSpace(asyncResponse.GetOutputConfig().GetGcsDestination().GetUri()) == "" {
		t.Fatalf("expected async response output config gcs destination uri")
	}

	restCreateProductSetResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.cloud.vision.v1.ProductSearch/CreateProductSet", []byte(`{
		"parent":"projects/stackyard/locations/us-central1",
		"productSetId":"product-set-1",
		"productSet":{"displayName":"Stackyard Product Set 1"}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "vision",
	})
	if restCreateProductSetResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest vision create product set, got %d body=%s", restCreateProductSetResp.StatusCode, string(providerContractBody(t, restCreateProductSetResp)))
	}
	restCreateProductSetBody := providerContractJSONMap(t, restCreateProductSetResp)
	restProductSetName, _ := restCreateProductSetBody["name"].(string)

	var grpcCreateProductSetResp visionpb.ProductSet
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpVisionCreateProductSetMethod, &visionpb.CreateProductSetRequest{
		Parent:       "projects/stackyard/locations/us-central1",
		ProductSetId: "product-set-1",
		ProductSet: &visionpb.ProductSet{
			DisplayName: "Stackyard Product Set 1",
		},
	}, &grpcCreateProductSetResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for create product set, got %q message=%q", grpcStatus, grpcMessage)
	}
	if grpcCreateProductSetResp.GetName() != restProductSetName {
		t.Fatalf("expected grpc product set name %q to match rest %q", grpcCreateProductSetResp.GetName(), restProductSetName)
	}

	var grpcImportProductSetsOp longrunningpb.Operation
	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpVisionImportProductSetsMethod, &visionpb.ImportProductSetsRequest{
		Parent: "projects/stackyard/locations/us-central1",
		InputConfig: &visionpb.ImportProductSetsInputConfig{
			Source: &visionpb.ImportProductSetsInputConfig_GcsSource{
				GcsSource: &visionpb.ImportProductSetsGcsSource{
					CsvFileUri: "gs://stackyard-inputs/vision-import.csv",
				},
			},
		},
	}, &grpcImportProductSetsOp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for import product sets, got %q message=%q", grpcStatus, grpcMessage)
	}
	var importMetadata visionpb.BatchOperationMetadata
	if err := grpcImportProductSetsOp.GetMetadata().UnmarshalTo(&importMetadata); err != nil {
		t.Fatalf("expected typed import metadata, got error: %v", err)
	}
	var importResponse visionpb.ImportProductSetsResponse
	if err := grpcImportProductSetsOp.GetResponse().UnmarshalTo(&importResponse); err != nil {
		t.Fatalf("expected typed import response, got error: %v", err)
	}
	if len(importResponse.GetReferenceImages()) == 0 {
		t.Fatalf("expected import response reference_images entries")
	}

	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpVisionBatchAnnotateImagesMethod, &visionpb.BatchAnnotateImagesRequest{
		Requests: []*visionpb.AnnotateImageRequest{
			{
				Image: &visionpb.Image{
					Source: &visionpb.ImageSource{ImageUri: "gs://stackyard-inputs/image-1.jpg"},
				},
			},
		},
	}, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "requests.features-required") {
		t.Fatalf("expected grpc invalid argument for missing features, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpVisionUpdateProductSetMethod, &visionpb.UpdateProductSetRequest{
		ProductSet: &visionpb.ProductSet{
			Name:        "projects/stackyard/locations/us-central1/productSets/product-set-1",
			DisplayName: "Updated Product Set",
		},
		UpdateMask: &fieldmaskpb.FieldMask{},
	}, nil)
	if grpcStatus != "3" || !strings.Contains(grpcMessage, "update_mask-required") {
		t.Fatalf("expected grpc invalid argument for empty update mask, got status=%q message=%q", grpcStatus, grpcMessage)
	}

	grpcStatus, grpcMessage = assertGCPStage4GRPCUnary(t, ts, gcpVisionGetProductSetMethod, &visionpb.GetProductSetRequest{
		Name: "projects/stackyard/locations/us-central1/productSets/missing-product-set",
	}, nil)
	if grpcStatus != "5" || !strings.Contains(grpcMessage, "product_set-not-found") {
		t.Fatalf("expected grpc not found for missing product set, got status=%q message=%q", grpcStatus, grpcMessage)
	}
}

func TestGCPStage4GRPCParity_VisionV2HintedREST(t *testing.T) {
	t.Parallel()

	ts := newGCPStage4GRPCContractServer(t)

	restBatchResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/google.cloud.vision.v1.ImageAnnotator/BatchAnnotateImages", []byte(`{
		"parent":"projects/stackyard/locations/us-central1",
		"requests":[{
			"image":{"source":{"imageUri":"gs://stackyard-inputs/image-1.jpg"}},
			"features":[{"type":"LABEL_DETECTION"}]
		}]
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "vision-v2-apiv1",
	})
	if restBatchResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from rest vision v2 batch annotate images, got %d body=%s", restBatchResp.StatusCode, string(providerContractBody(t, restBatchResp)))
	}
	restBatchBody := providerContractJSONMap(t, restBatchResp)
	restResponses, ok := restBatchBody["responses"].([]any)
	if !ok || len(restResponses) == 0 {
		t.Fatalf("expected non-empty rest responses array, got %#v", restBatchBody["responses"])
	}

	var grpcBatchResp visionpb.BatchAnnotateImagesResponse
	grpcStatus, grpcMessage := assertGCPStage4GRPCUnary(t, ts, gcpVisionBatchAnnotateImagesMethod, &visionpb.BatchAnnotateImagesRequest{
		Parent: "projects/stackyard/locations/us-central1",
		Requests: []*visionpb.AnnotateImageRequest{
			{
				Image: &visionpb.Image{
					Source: &visionpb.ImageSource{ImageUri: "gs://stackyard-inputs/image-1.jpg"},
				},
				Features: []*visionpb.Feature{{Type: visionpb.Feature_LABEL_DETECTION}},
			},
		},
	}, &grpcBatchResp)
	if grpcStatus != "0" {
		t.Fatalf("expected grpc status 0 for vision batch annotate images, got %q message=%q", grpcStatus, grpcMessage)
	}
	if len(grpcBatchResp.GetResponses()) != len(restResponses) {
		t.Fatalf("expected grpc responses len %d to match rest %d", len(grpcBatchResp.GetResponses()), len(restResponses))
	}
}
