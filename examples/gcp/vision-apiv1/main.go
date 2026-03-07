package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	vision "cloud.google.com/go/vision/apiv1"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	visionpb "google.golang.org/genproto/googleapis/cloud/vision/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

func main() {
	ctx := context.Background()
	endpoint := strings.TrimRight(getenv("STACKYARD_ENDPOINT", "http://localhost:4566"), "/")
	apiEndpoint := endpoint + "/gcp"
	grpcEndpoint := strings.TrimPrefix(strings.TrimPrefix(endpoint, "http://"), "https://")

	projectID := getenv("STACKYARD_GCP_PROJECT_ID", "stackyard")
	locationID := getenv("STACKYARD_GCP_LOCATION", "us-central1")
	parent := fmt.Sprintf("projects/%s/locations/%s", projectID, locationID)

	productSetID := getenv("STACKYARD_GCP_VISION_PRODUCT_SET_ID", "product-set-1")
	productID := getenv("STACKYARD_GCP_VISION_PRODUCT_ID", "product-1")
	referenceImageID := getenv("STACKYARD_GCP_VISION_REFERENCE_IMAGE_ID", "reference-image-1")
	imageURI := getenv("STACKYARD_GCP_VISION_IMAGE_URI", "gs://stackyard-inputs/image-1.jpg")
	fileURI := getenv("STACKYARD_GCP_VISION_FILE_URI", "gs://stackyard-inputs/file-1.pdf")
	outputURI := getenv("STACKYARD_GCP_VISION_OUTPUT_URI", "gs://stackyard-outputs/vision/")
	importCSVURI := getenv("STACKYARD_GCP_VISION_IMPORT_CSV_URI", "gs://stackyard-inputs/vision-import.csv")

	productSetName := parent + "/productSets/" + productSetID
	productName := parent + "/products/" + productID
	referenceImageName := productName + "/referenceImages/" + referenceImageID

	fmt.Printf("Stackyard GCP Vision vision/apiv1 clients using %s (grpc=%s)\n", apiEndpoint, grpcEndpoint)
	if err := waitForStackyardReady(ctx, apiEndpoint, projectID, locationID); err != nil {
		exitf("stackyard readiness check failed: %v", err)
	}

	imageClient, err := vision.NewImageAnnotatorClient(ctx,
		option.WithEndpoint(grpcEndpoint),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create vision image annotator client: %v", err)
	}
	defer closeClient("vision image annotator", imageClient.Close)

	productClient, err := vision.NewProductSearchClient(ctx,
		option.WithEndpoint(grpcEndpoint),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
		option.WithoutAuthentication(),
	)
	if err != nil {
		exitf("failed to create vision product search client: %v", err)
	}
	defer closeClient("vision product search", productClient.Close)

	if err := runImageAnnotatorCalls(ctx, imageClient, parent, imageURI, fileURI, outputURI); err != nil {
		exitf("image annotator calls failed: %v", err)
	}
	if err := runProductSearchCalls(ctx, productClient, parent, productSetID, productSetName, productID, productName, referenceImageID, referenceImageName, imageURI, importCSVURI); err != nil {
		exitf("product search calls failed: %v", err)
	}

	fmt.Println("Done.")
}

func runImageAnnotatorCalls(ctx context.Context, client *vision.ImageAnnotatorClient, parent, imageURI, fileURI, outputURI string) error {
	batchImagesResp, err := client.BatchAnnotateImages(ctx, &visionpb.BatchAnnotateImagesRequest{
		Parent: parent,
		Requests: []*visionpb.AnnotateImageRequest{
			{
				Image: &visionpb.Image{
					Source: &visionpb.ImageSource{ImageUri: imageURI},
				},
				Features: []*visionpb.Feature{{Type: visionpb.Feature_LABEL_DETECTION}},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("BatchAnnotateImages: %w", err)
	}
	if len(batchImagesResp.GetResponses()) == 0 || len(batchImagesResp.GetResponses()[0].GetLabelAnnotations()) == 0 {
		return errors.New("BatchAnnotateImages returned no label annotations")
	}
	logf("BatchAnnotateImages succeeded")

	batchFilesResp, err := client.BatchAnnotateFiles(ctx, &visionpb.BatchAnnotateFilesRequest{
		Requests: []*visionpb.AnnotateFileRequest{
			{
				InputConfig: &visionpb.InputConfig{
					GcsSource: &visionpb.GcsSource{Uri: fileURI},
					MimeType:  "application/pdf",
				},
				Features: []*visionpb.Feature{{Type: visionpb.Feature_DOCUMENT_TEXT_DETECTION}},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("BatchAnnotateFiles: %w", err)
	}
	if len(batchFilesResp.GetResponses()) == 0 || len(batchFilesResp.GetResponses()[0].GetResponses()) == 0 {
		return errors.New("BatchAnnotateFiles returned no page responses")
	}
	logf("BatchAnnotateFiles succeeded")

	asyncImagesOp, err := client.AsyncBatchAnnotateImages(ctx, &visionpb.AsyncBatchAnnotateImagesRequest{
		Parent: parent,
		Requests: []*visionpb.AnnotateImageRequest{
			{
				Image: &visionpb.Image{
					Source: &visionpb.ImageSource{ImageUri: imageURI},
				},
				Features: []*visionpb.Feature{{Type: visionpb.Feature_LABEL_DETECTION}},
			},
		},
		OutputConfig: &visionpb.OutputConfig{
			GcsDestination: &visionpb.GcsDestination{Uri: outputURI},
		},
	})
	if err != nil {
		return fmt.Errorf("AsyncBatchAnnotateImages: %w", err)
	}
	if strings.TrimSpace(asyncImagesOp.Name()) == "" {
		return errors.New("AsyncBatchAnnotateImages returned empty operation name")
	}
	metadata, err := asyncImagesOp.Metadata()
	if err != nil {
		return fmt.Errorf("AsyncBatchAnnotateImages Metadata: %w", err)
	}
	if metadata == nil {
		return errors.New("AsyncBatchAnnotateImages Metadata returned nil")
	}
	waitCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	asyncImagesResp, err := asyncImagesOp.Wait(waitCtx)
	if err != nil {
		return fmt.Errorf("AsyncBatchAnnotateImages Wait: %w", err)
	}
	if strings.TrimSpace(asyncImagesResp.GetOutputConfig().GetGcsDestination().GetUri()) == "" {
		return errors.New("AsyncBatchAnnotateImages Wait returned empty output config uri")
	}
	logf("AsyncBatchAnnotateImages succeeded")

	asyncFilesOp, err := client.AsyncBatchAnnotateFiles(ctx, &visionpb.AsyncBatchAnnotateFilesRequest{
		Parent: parent,
		Requests: []*visionpb.AsyncAnnotateFileRequest{
			{
				InputConfig: &visionpb.InputConfig{
					GcsSource: &visionpb.GcsSource{Uri: fileURI},
					MimeType:  "application/pdf",
				},
				Features: []*visionpb.Feature{{Type: visionpb.Feature_DOCUMENT_TEXT_DETECTION}},
				OutputConfig: &visionpb.OutputConfig{
					GcsDestination: &visionpb.GcsDestination{Uri: outputURI},
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("AsyncBatchAnnotateFiles: %w", err)
	}
	if strings.TrimSpace(asyncFilesOp.Name()) == "" {
		return errors.New("AsyncBatchAnnotateFiles returned empty operation name")
	}
	waitCtxFiles, cancelFiles := context.WithTimeout(ctx, 15*time.Second)
	defer cancelFiles()
	asyncFilesResp, err := asyncFilesOp.Wait(waitCtxFiles)
	if err != nil {
		return fmt.Errorf("AsyncBatchAnnotateFiles Wait: %w", err)
	}
	if len(asyncFilesResp.GetResponses()) == 0 {
		return errors.New("AsyncBatchAnnotateFiles Wait returned no responses")
	}
	logf("AsyncBatchAnnotateFiles succeeded")

	helperLabels, err := client.DetectLabels(ctx, vision.NewImageFromURI(imageURI), nil, 1)
	if err != nil {
		return fmt.Errorf("DetectLabels: %w", err)
	}
	if len(helperLabels) == 0 {
		return errors.New("DetectLabels returned no labels")
	}
	logf("DetectLabels helper succeeded")

	_, err = client.BatchAnnotateImages(ctx, &visionpb.BatchAnnotateImagesRequest{
		Parent: parent,
		Requests: []*visionpb.AnnotateImageRequest{
			{
				Image: &visionpb.Image{
					Source: &visionpb.ImageSource{ImageUri: imageURI},
				},
			},
		},
	})
	if err == nil {
		return errors.New("BatchAnnotateImages missing features unexpectedly succeeded")
	}
	if !isInvalidArgument(err) {
		return fmt.Errorf("BatchAnnotateImages missing features returned unexpected error: %w", err)
	}
	logf("BatchAnnotateImages invalid request returned InvalidArgument as expected")

	return nil
}

func runProductSearchCalls(
	ctx context.Context,
	client *vision.ProductSearchClient,
	parent string,
	productSetID string,
	productSetName string,
	productID string,
	productName string,
	referenceImageID string,
	referenceImageName string,
	imageURI string,
	importCSVURI string,
) error {
	createdProductSet, err := client.CreateProductSet(ctx, &visionpb.CreateProductSetRequest{
		Parent:       parent,
		ProductSetId: productSetID,
		ProductSet: &visionpb.ProductSet{
			DisplayName: "Stackyard Product Set 1",
		},
	})
	if err != nil {
		return fmt.Errorf("CreateProductSet: %w", err)
	}
	if strings.TrimSpace(createdProductSet.GetName()) == "" {
		return errors.New("CreateProductSet returned empty name")
	}
	logf("CreateProductSet succeeded")

	productSetIt := client.ListProductSets(ctx, &visionpb.ListProductSetsRequest{Parent: parent, PageSize: 1})
	_, err = productSetIt.Next()
	if err != nil && !errors.Is(err, iterator.Done) {
		return fmt.Errorf("ListProductSets: %w", err)
	}
	logf("ListProductSets succeeded")

	gotProductSet, err := client.GetProductSet(ctx, &visionpb.GetProductSetRequest{Name: productSetName})
	if err != nil {
		return fmt.Errorf("GetProductSet: %w", err)
	}
	if gotProductSet.GetName() != productSetName {
		return fmt.Errorf("GetProductSet returned unexpected name %q", gotProductSet.GetName())
	}
	logf("GetProductSet succeeded")

	updatedProductSet, err := client.UpdateProductSet(ctx, &visionpb.UpdateProductSetRequest{
		ProductSet: &visionpb.ProductSet{
			Name:        productSetName,
			DisplayName: "Stackyard Product Set Updated",
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"display_name"}},
	})
	if err != nil {
		return fmt.Errorf("UpdateProductSet: %w", err)
	}
	if strings.TrimSpace(updatedProductSet.GetDisplayName()) == "" {
		return errors.New("UpdateProductSet returned empty display_name")
	}
	logf("UpdateProductSet succeeded")

	createdProduct, err := client.CreateProduct(ctx, &visionpb.CreateProductRequest{
		Parent:    parent,
		ProductId: productID,
		Product: &visionpb.Product{
			DisplayName:     "Stackyard Product 1",
			ProductCategory: "general-v1",
		},
	})
	if err != nil {
		return fmt.Errorf("CreateProduct: %w", err)
	}
	if strings.TrimSpace(createdProduct.GetName()) == "" {
		return errors.New("CreateProduct returned empty name")
	}
	logf("CreateProduct succeeded")

	productIt := client.ListProducts(ctx, &visionpb.ListProductsRequest{Parent: parent, PageSize: 1})
	_, err = productIt.Next()
	if err != nil && !errors.Is(err, iterator.Done) {
		return fmt.Errorf("ListProducts: %w", err)
	}
	logf("ListProducts succeeded")

	gotProduct, err := client.GetProduct(ctx, &visionpb.GetProductRequest{Name: productName})
	if err != nil {
		return fmt.Errorf("GetProduct: %w", err)
	}
	if gotProduct.GetName() != productName {
		return fmt.Errorf("GetProduct returned unexpected name %q", gotProduct.GetName())
	}
	logf("GetProduct succeeded")

	updatedProduct, err := client.UpdateProduct(ctx, &visionpb.UpdateProductRequest{
		Product: &visionpb.Product{
			Name:            productName,
			DisplayName:     "Stackyard Product Updated",
			ProductCategory: "general-v1",
		},
		UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"display_name"}},
	})
	if err != nil {
		return fmt.Errorf("UpdateProduct: %w", err)
	}
	if strings.TrimSpace(updatedProduct.GetDisplayName()) == "" {
		return errors.New("UpdateProduct returned empty display_name")
	}
	logf("UpdateProduct succeeded")

	createdRef, err := client.CreateReferenceImage(ctx, &visionpb.CreateReferenceImageRequest{
		Parent:           productName,
		ReferenceImageId: referenceImageID,
		ReferenceImage: &visionpb.ReferenceImage{
			Uri: imageURI,
		},
	})
	if err != nil {
		return fmt.Errorf("CreateReferenceImage: %w", err)
	}
	if strings.TrimSpace(createdRef.GetName()) == "" {
		return errors.New("CreateReferenceImage returned empty name")
	}
	logf("CreateReferenceImage succeeded")

	refIt := client.ListReferenceImages(ctx, &visionpb.ListReferenceImagesRequest{Parent: productName, PageSize: 1})
	_, err = refIt.Next()
	if err != nil && !errors.Is(err, iterator.Done) {
		return fmt.Errorf("ListReferenceImages: %w", err)
	}
	logf("ListReferenceImages succeeded")

	gotRef, err := client.GetReferenceImage(ctx, &visionpb.GetReferenceImageRequest{Name: referenceImageName})
	if err != nil {
		return fmt.Errorf("GetReferenceImage: %w", err)
	}
	if gotRef.GetName() != referenceImageName {
		return fmt.Errorf("GetReferenceImage returned unexpected name %q", gotRef.GetName())
	}
	logf("GetReferenceImage succeeded")

	if err := client.AddProductToProductSet(ctx, &visionpb.AddProductToProductSetRequest{
		Name:    productSetName,
		Product: productName,
	}); err != nil {
		return fmt.Errorf("AddProductToProductSet: %w", err)
	}
	logf("AddProductToProductSet succeeded")

	productInSetIt := client.ListProductsInProductSet(ctx, &visionpb.ListProductsInProductSetRequest{
		Name:     productSetName,
		PageSize: 1,
	})
	_, err = productInSetIt.Next()
	if err != nil && !errors.Is(err, iterator.Done) {
		return fmt.Errorf("ListProductsInProductSet: %w", err)
	}
	logf("ListProductsInProductSet succeeded")

	if err := client.RemoveProductFromProductSet(ctx, &visionpb.RemoveProductFromProductSetRequest{
		Name:    productSetName,
		Product: productName,
	}); err != nil {
		return fmt.Errorf("RemoveProductFromProductSet: %w", err)
	}
	logf("RemoveProductFromProductSet succeeded")

	importOp, err := client.ImportProductSets(ctx, &visionpb.ImportProductSetsRequest{
		Parent: parent,
		InputConfig: &visionpb.ImportProductSetsInputConfig{
			Source: &visionpb.ImportProductSetsInputConfig_GcsSource{
				GcsSource: &visionpb.ImportProductSetsGcsSource{
					CsvFileUri: importCSVURI,
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("ImportProductSets: %w", err)
	}
	if strings.TrimSpace(importOp.Name()) == "" {
		return errors.New("ImportProductSets returned empty operation name")
	}
	importWaitCtx, importCancel := context.WithTimeout(ctx, 15*time.Second)
	defer importCancel()
	importResp, err := importOp.Wait(importWaitCtx)
	if err != nil {
		return fmt.Errorf("ImportProductSets Wait: %w", err)
	}
	if len(importResp.GetStatuses()) == 0 {
		return errors.New("ImportProductSets Wait returned empty statuses")
	}
	logf("ImportProductSets succeeded")

	purgeOp, err := client.PurgeProducts(ctx, &visionpb.PurgeProductsRequest{
		Parent: parent,
		Target: &visionpb.PurgeProductsRequest_ProductSetPurgeConfig{
			ProductSetPurgeConfig: &visionpb.ProductSetPurgeConfig{ProductSetId: productSetID},
		},
	})
	if err != nil {
		return fmt.Errorf("PurgeProducts: %w", err)
	}
	if strings.TrimSpace(purgeOp.Name()) == "" {
		return errors.New("PurgeProducts returned empty operation name")
	}
	purgeWaitCtx, purgeCancel := context.WithTimeout(ctx, 15*time.Second)
	defer purgeCancel()
	if err := purgeOp.Wait(purgeWaitCtx); err != nil {
		return fmt.Errorf("PurgeProducts Wait: %w", err)
	}
	logf("PurgeProducts succeeded")

	if err := client.DeleteReferenceImage(ctx, &visionpb.DeleteReferenceImageRequest{Name: referenceImageName}); err != nil {
		return fmt.Errorf("DeleteReferenceImage: %w", err)
	}
	logf("DeleteReferenceImage succeeded")

	if err := client.DeleteProduct(ctx, &visionpb.DeleteProductRequest{Name: productName}); err != nil {
		return fmt.Errorf("DeleteProduct: %w", err)
	}
	logf("DeleteProduct succeeded")

	if err := client.DeleteProductSet(ctx, &visionpb.DeleteProductSetRequest{Name: productSetName}); err != nil {
		return fmt.Errorf("DeleteProductSet: %w", err)
	}
	logf("DeleteProductSet succeeded")

	_, err = client.GetProductSet(ctx, &visionpb.GetProductSetRequest{
		Name: strings.Replace(productSetName, productSetID, "missing-product-set", 1),
	})
	if err == nil {
		return errors.New("GetProductSet missing resource unexpectedly succeeded")
	}
	if !isNotFound(err) {
		return fmt.Errorf("GetProductSet missing resource returned unexpected error: %w", err)
	}
	logf("GetProductSet missing resource returned NotFound as expected")

	return nil
}

func waitForStackyardReady(ctx context.Context, apiEndpoint, projectID, locationID string) error {
	readyURL := fmt.Sprintf(
		"%s/v1/projects/%s/locations/%s/vision?stackyard_contract_probe=1&typedSuccess=1",
		strings.TrimRight(apiEndpoint, "/"),
		projectID,
		locationID,
	)
	httpClient := &http.Client{Timeout: time.Second}
	deadline := time.Now().Add(20 * time.Second)
	var lastErr error

	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, readyURL, nil)
		if err != nil {
			return err
		}
		req.Header.Set("X-Stackyard-GCP-Service", "vision")

		resp, err := httpClient.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
			lastErr = fmt.Errorf("ready probe status %d", resp.StatusCode)
		} else {
			lastErr = err
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for %s: %w", readyURL, lastErr)
		}
		time.Sleep(300 * time.Millisecond)
	}
}

func isInvalidArgument(err error) bool {
	if err == nil {
		return false
	}

	if grpcStatus, ok := status.FromError(err); ok && grpcStatus.Code() == codes.InvalidArgument {
		return true
	}

	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) && apiErr.Code == http.StatusBadRequest {
		return true
	}

	return strings.Contains(strings.ToLower(err.Error()), "invalidargument")
}

func isNotFound(err error) bool {
	if err == nil {
		return false
	}

	if grpcStatus, ok := status.FromError(err); ok && grpcStatus.Code() == codes.NotFound {
		return true
	}

	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) && apiErr.Code == http.StatusNotFound {
		return true
	}

	return strings.Contains(strings.ToLower(err.Error()), "notfound")
}

func closeClient(name string, closeFn func() error) {
	if err := closeFn(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: close %s client: %v\n", name, err)
	}
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

func logf(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}
