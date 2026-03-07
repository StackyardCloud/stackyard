package server

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	longrunningpb "cloud.google.com/go/longrunning/autogen/longrunningpb"
	visionpb "google.golang.org/genproto/googleapis/cloud/vision/v1"
	statuspb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	gcpVisionBatchAnnotateImagesMethod      = "/google.cloud.vision.v1.ImageAnnotator/BatchAnnotateImages"
	gcpVisionBatchAnnotateFilesMethod       = "/google.cloud.vision.v1.ImageAnnotator/BatchAnnotateFiles"
	gcpVisionAsyncBatchAnnotateImagesMethod = "/google.cloud.vision.v1.ImageAnnotator/AsyncBatchAnnotateImages"
	gcpVisionAsyncBatchAnnotateFilesMethod  = "/google.cloud.vision.v1.ImageAnnotator/AsyncBatchAnnotateFiles"

	gcpVisionCreateProductSetMethod = "/google.cloud.vision.v1.ProductSearch/CreateProductSet"
	gcpVisionListProductSetsMethod  = "/google.cloud.vision.v1.ProductSearch/ListProductSets"
	gcpVisionGetProductSetMethod    = "/google.cloud.vision.v1.ProductSearch/GetProductSet"
	gcpVisionUpdateProductSetMethod = "/google.cloud.vision.v1.ProductSearch/UpdateProductSet"
	gcpVisionDeleteProductSetMethod = "/google.cloud.vision.v1.ProductSearch/DeleteProductSet"

	gcpVisionCreateProductMethod = "/google.cloud.vision.v1.ProductSearch/CreateProduct"
	gcpVisionListProductsMethod  = "/google.cloud.vision.v1.ProductSearch/ListProducts"
	gcpVisionGetProductMethod    = "/google.cloud.vision.v1.ProductSearch/GetProduct"
	gcpVisionUpdateProductMethod = "/google.cloud.vision.v1.ProductSearch/UpdateProduct"
	gcpVisionDeleteProductMethod = "/google.cloud.vision.v1.ProductSearch/DeleteProduct"

	gcpVisionCreateReferenceImageMethod = "/google.cloud.vision.v1.ProductSearch/CreateReferenceImage"
	gcpVisionDeleteReferenceImageMethod = "/google.cloud.vision.v1.ProductSearch/DeleteReferenceImage"
	gcpVisionListReferenceImagesMethod  = "/google.cloud.vision.v1.ProductSearch/ListReferenceImages"
	gcpVisionGetReferenceImageMethod    = "/google.cloud.vision.v1.ProductSearch/GetReferenceImage"

	gcpVisionAddProductToProductSetMethod      = "/google.cloud.vision.v1.ProductSearch/AddProductToProductSet"
	gcpVisionRemoveProductFromProductSetMethod = "/google.cloud.vision.v1.ProductSearch/RemoveProductFromProductSet"
	gcpVisionListProductsInProductSetMethod    = "/google.cloud.vision.v1.ProductSearch/ListProductsInProductSet"

	gcpVisionImportProductSetsMethod = "/google.cloud.vision.v1.ProductSearch/ImportProductSets"
	gcpVisionPurgeProductsMethod     = "/google.cloud.vision.v1.ProductSearch/PurgeProducts"
)

func gcpStage4GRPCVision(path string, grpcReqBody []byte) ([]byte, string, string, bool) {
	switch path {
	case gcpVisionBatchAnnotateImagesMethod:
		return gcpStage4GRPCVisionBatchAnnotateImages(grpcReqBody)
	case gcpVisionBatchAnnotateFilesMethod:
		return gcpStage4GRPCVisionBatchAnnotateFiles(grpcReqBody)
	case gcpVisionAsyncBatchAnnotateImagesMethod:
		return gcpStage4GRPCVisionAsyncBatchAnnotateImages(grpcReqBody)
	case gcpVisionAsyncBatchAnnotateFilesMethod:
		return gcpStage4GRPCVisionAsyncBatchAnnotateFiles(grpcReqBody)
	case gcpVisionCreateProductSetMethod:
		return gcpStage4GRPCVisionCreateProductSet(grpcReqBody)
	case gcpVisionListProductSetsMethod:
		return gcpStage4GRPCVisionListProductSets(grpcReqBody)
	case gcpVisionGetProductSetMethod:
		return gcpStage4GRPCVisionGetProductSet(grpcReqBody)
	case gcpVisionUpdateProductSetMethod:
		return gcpStage4GRPCVisionUpdateProductSet(grpcReqBody)
	case gcpVisionDeleteProductSetMethod:
		return gcpStage4GRPCVisionDeleteProductSet(grpcReqBody)
	case gcpVisionCreateProductMethod:
		return gcpStage4GRPCVisionCreateProduct(grpcReqBody)
	case gcpVisionListProductsMethod:
		return gcpStage4GRPCVisionListProducts(grpcReqBody)
	case gcpVisionGetProductMethod:
		return gcpStage4GRPCVisionGetProduct(grpcReqBody)
	case gcpVisionUpdateProductMethod:
		return gcpStage4GRPCVisionUpdateProduct(grpcReqBody)
	case gcpVisionDeleteProductMethod:
		return gcpStage4GRPCVisionDeleteProduct(grpcReqBody)
	case gcpVisionCreateReferenceImageMethod:
		return gcpStage4GRPCVisionCreateReferenceImage(grpcReqBody)
	case gcpVisionDeleteReferenceImageMethod:
		return gcpStage4GRPCVisionDeleteReferenceImage(grpcReqBody)
	case gcpVisionListReferenceImagesMethod:
		return gcpStage4GRPCVisionListReferenceImages(grpcReqBody)
	case gcpVisionGetReferenceImageMethod:
		return gcpStage4GRPCVisionGetReferenceImage(grpcReqBody)
	case gcpVisionAddProductToProductSetMethod:
		return gcpStage4GRPCVisionAddProductToProductSet(grpcReqBody)
	case gcpVisionRemoveProductFromProductSetMethod:
		return gcpStage4GRPCVisionRemoveProductFromProductSet(grpcReqBody)
	case gcpVisionListProductsInProductSetMethod:
		return gcpStage4GRPCVisionListProductsInProductSet(grpcReqBody)
	case gcpVisionImportProductSetsMethod:
		return gcpStage4GRPCVisionImportProductSets(grpcReqBody)
	case gcpVisionPurgeProductsMethod:
		return gcpStage4GRPCVisionPurgeProducts(grpcReqBody)
	default:
		return grpcUnimplemented("method-not-implemented")
	}
}

func gcpStage4GRPCVisionBatchAnnotateImages(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &visionpb.BatchAnnotateImagesRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if len(req.GetRequests()) == 0 {
		return grpcInvalidArgument("requests-required")
	}

	project, location := gcpStage4VisionProjectLocationFromParent(req.GetParent())
	responses := make([]*visionpb.AnnotateImageResponse, 0, len(req.GetRequests()))
	for _, item := range req.GetRequests() {
		if item == nil {
			return grpcInvalidArgument("requests-entry-invalid")
		}
		features, ok := gcpStage4VisionValidateFeatures(item.GetFeatures())
		if !ok {
			return grpcInvalidArgument("requests.features-required")
		}
		imageURI := gcpStage4VisionImageURI(item.GetImage())
		if imageURI == "" {
			return grpcInvalidArgument("requests.image-required")
		}
		responses = append(responses, gcpStage4VisionAnnotateImageResponse(project, location, imageURI, features))
	}

	return grpcProtoSuccess(&visionpb.BatchAnnotateImagesResponse{Responses: responses})
}

func gcpStage4GRPCVisionBatchAnnotateFiles(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &visionpb.BatchAnnotateFilesRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if len(req.GetRequests()) == 0 {
		return grpcInvalidArgument("requests-required")
	}

	responses := make([]*visionpb.AnnotateFileResponse, 0, len(req.GetRequests()))
	for _, item := range req.GetRequests() {
		if item == nil {
			return grpcInvalidArgument("requests-entry-invalid")
		}
		features, ok := gcpStage4VisionValidateFeatures(item.GetFeatures())
		if !ok {
			return grpcInvalidArgument("requests.features-required")
		}
		fileURI := strings.TrimSpace(item.GetInputConfig().GetGcsSource().GetUri())
		if fileURI == "" || !isGCPVisionGCSURI(fileURI) {
			return grpcInvalidArgument("requests.input_config.gcs_source.uri-invalid")
		}
		if len(item.GetPages()) > 5 {
			return grpcInvalidArgument("requests.pages-too-many")
		}
		responses = append(responses, &visionpb.AnnotateFileResponse{
			InputConfig: item.GetInputConfig(),
			Responses: []*visionpb.AnnotateImageResponse{
				gcpStage4VisionAnnotateImageResponse("stackyard", "us-central1", fileURI, features),
			},
			TotalPages: 1,
		})
	}

	return grpcProtoSuccess(&visionpb.BatchAnnotateFilesResponse{Responses: responses})
}

func gcpStage4GRPCVisionAsyncBatchAnnotateImages(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &visionpb.AsyncBatchAnnotateImagesRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if len(req.GetRequests()) == 0 {
		return grpcInvalidArgument("requests-required")
	}
	for _, item := range req.GetRequests() {
		if item == nil {
			return grpcInvalidArgument("requests-entry-invalid")
		}
		if _, ok := gcpStage4VisionValidateFeatures(item.GetFeatures()); !ok {
			return grpcInvalidArgument("requests.features-required")
		}
		if gcpStage4VisionImageURI(item.GetImage()) == "" {
			return grpcInvalidArgument("requests.image-required")
		}
	}
	outputURI := strings.TrimSpace(req.GetOutputConfig().GetGcsDestination().GetUri())
	if outputURI == "" || !isGCPVisionGCSOutputURI(outputURI) {
		return grpcInvalidArgument("output_config.gcs_destination.uri-invalid")
	}

	project, location := gcpStage4VisionProjectLocationFromParent(req.GetParent())
	return grpcProtoSuccess(gcpStage4VisionOperation(
		project,
		location,
		"asyncBatchAnnotateImages.op-1",
		&visionpb.OperationMetadata{
			State:      visionpb.OperationMetadata_DONE,
			CreateTime: timestamppb.New(gcpStage4ReferenceTime),
			UpdateTime: timestamppb.New(gcpStage4ReferenceTime.Add(2 * time.Second)),
		},
		&visionpb.AsyncBatchAnnotateImagesResponse{
			OutputConfig: req.GetOutputConfig(),
		},
	))
}

func gcpStage4GRPCVisionAsyncBatchAnnotateFiles(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &visionpb.AsyncBatchAnnotateFilesRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if len(req.GetRequests()) == 0 {
		return grpcInvalidArgument("requests-required")
	}

	responses := make([]*visionpb.AsyncAnnotateFileResponse, 0, len(req.GetRequests()))
	for _, item := range req.GetRequests() {
		if item == nil {
			return grpcInvalidArgument("requests-entry-invalid")
		}
		if _, ok := gcpStage4VisionValidateFeatures(item.GetFeatures()); !ok {
			return grpcInvalidArgument("requests.features-required")
		}
		fileURI := strings.TrimSpace(item.GetInputConfig().GetGcsSource().GetUri())
		if fileURI == "" || !isGCPVisionGCSURI(fileURI) {
			return grpcInvalidArgument("requests.input_config.gcs_source.uri-invalid")
		}
		outputURI := strings.TrimSpace(item.GetOutputConfig().GetGcsDestination().GetUri())
		if outputURI == "" || !isGCPVisionGCSOutputURI(outputURI) {
			return grpcInvalidArgument("requests.output_config.gcs_destination.uri-invalid")
		}
		responses = append(responses, &visionpb.AsyncAnnotateFileResponse{
			OutputConfig: item.GetOutputConfig(),
		})
	}

	project, location := gcpStage4VisionProjectLocationFromParent(req.GetParent())
	return grpcProtoSuccess(gcpStage4VisionOperation(
		project,
		location,
		"asyncBatchAnnotateFiles.op-1",
		&visionpb.OperationMetadata{
			State:      visionpb.OperationMetadata_DONE,
			CreateTime: timestamppb.New(gcpStage4ReferenceTime),
			UpdateTime: timestamppb.New(gcpStage4ReferenceTime.Add(2 * time.Second)),
		},
		&visionpb.AsyncBatchAnnotateFilesResponse{
			Responses: responses,
		},
	))
}

func gcpStage4GRPCVisionCreateProductSet(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &visionpb.CreateProductSetRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := parseGCPVisionParentName(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if !isGCPVisionResourceID(req.GetProductSetId()) {
		return grpcInvalidArgument("product_set_id-required")
	}
	if req.GetProductSet() == nil {
		return grpcInvalidArgument("product_set-required")
	}
	if strings.Contains(strings.ToLower(req.GetProductSetId()), "existing") {
		return grpcAlreadyExists("product_set-already-exists")
	}
	expectedName := fmt.Sprintf("projects/%s/locations/%s/productSets/%s", project, location, req.GetProductSetId())
	if name := strings.TrimSpace(req.GetProductSet().GetName()); name != "" && name != expectedName {
		return grpcInvalidArgument("product_set.name-mismatch")
	}
	displayName := strings.TrimSpace(req.GetProductSet().GetDisplayName())
	if displayName == "" {
		displayName = "Stackyard Product Set " + req.GetProductSetId()
	}
	return grpcProtoSuccess(gcpStage4VisionProductSet(project, location, req.GetProductSetId(), displayName))
}

func gcpStage4GRPCVisionListProductSets(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &visionpb.ListProductSetsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := parseGCPVisionParentName(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	start, end, next, reason, ok := gcpStage4VisionPageWindow(req.GetPageSize(), req.GetPageToken(), 100, 2)
	if !ok {
		return grpcInvalidArgument(reason)
	}
	items := []*visionpb.ProductSet{
		gcpStage4VisionProductSet(project, location, "product-set-1", "Stackyard Product Set 1"),
		gcpStage4VisionProductSet(project, location, "product-set-2", "Stackyard Product Set 2"),
	}
	return grpcProtoSuccess(&visionpb.ListProductSetsResponse{
		ProductSets:   items[start:end],
		NextPageToken: next,
	})
}

func gcpStage4GRPCVisionGetProductSet(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &visionpb.GetProductSetRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, productSetID, ok := parseGCPVisionProductSetName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPVisionMissingID(productSetID) {
		return grpcNotFound("product_set-not-found")
	}
	return grpcProtoSuccess(gcpStage4VisionProductSet(project, location, productSetID, "Stackyard Product Set "+productSetID))
}

func gcpStage4GRPCVisionUpdateProductSet(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &visionpb.UpdateProductSetRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if req.GetProductSet() == nil {
		return grpcInvalidArgument("product_set-required")
	}
	if req.GetUpdateMask() == nil || len(req.GetUpdateMask().GetPaths()) == 0 {
		return grpcInvalidArgument("update_mask-required")
	}
	project, location, productSetID, ok := parseGCPVisionProductSetName(strings.TrimSpace(req.GetProductSet().GetName()))
	if !ok {
		return grpcInvalidArgument("product_set.name-required")
	}
	displayName := strings.TrimSpace(req.GetProductSet().GetDisplayName())
	if displayName == "" {
		displayName = "Stackyard Product Set " + productSetID
	}
	return grpcProtoSuccess(gcpStage4VisionProductSet(project, location, productSetID, displayName))
}

func gcpStage4GRPCVisionDeleteProductSet(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &visionpb.DeleteProductSetRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	_, _, productSetID, ok := parseGCPVisionProductSetName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPVisionMissingID(productSetID) {
		return grpcNotFound("product_set-not-found")
	}
	return grpcProtoSuccess(&emptypb.Empty{})
}

func gcpStage4GRPCVisionCreateProduct(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &visionpb.CreateProductRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := parseGCPVisionParentName(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if !isGCPVisionResourceID(req.GetProductId()) {
		return grpcInvalidArgument("product_id-required")
	}
	if req.GetProduct() == nil {
		return grpcInvalidArgument("product-required")
	}
	if strings.Contains(strings.ToLower(req.GetProductId()), "existing") {
		return grpcAlreadyExists("product-already-exists")
	}
	expectedName := fmt.Sprintf("projects/%s/locations/%s/products/%s", project, location, req.GetProductId())
	if name := strings.TrimSpace(req.GetProduct().GetName()); name != "" && name != expectedName {
		return grpcInvalidArgument("product.name-mismatch")
	}
	displayName := strings.TrimSpace(req.GetProduct().GetDisplayName())
	if displayName == "" {
		displayName = "Stackyard Product " + req.GetProductId()
	}
	category := strings.TrimSpace(req.GetProduct().GetProductCategory())
	if category == "" {
		return grpcInvalidArgument("product.product_category-required")
	}
	return grpcProtoSuccess(gcpStage4VisionProduct(project, location, req.GetProductId(), displayName, category))
}

func gcpStage4GRPCVisionListProducts(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &visionpb.ListProductsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := parseGCPVisionParentName(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	start, end, next, reason, ok := gcpStage4VisionPageWindow(req.GetPageSize(), req.GetPageToken(), 100, 2)
	if !ok {
		return grpcInvalidArgument(reason)
	}
	items := []*visionpb.Product{
		gcpStage4VisionProduct(project, location, "product-1", "Stackyard Product 1", "general-v1"),
		gcpStage4VisionProduct(project, location, "product-2", "Stackyard Product 2", "general-v1"),
	}
	return grpcProtoSuccess(&visionpb.ListProductsResponse{
		Products:      items[start:end],
		NextPageToken: next,
	})
}

func gcpStage4GRPCVisionGetProduct(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &visionpb.GetProductRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, productID, ok := parseGCPVisionProductName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPVisionMissingID(productID) {
		return grpcNotFound("product-not-found")
	}
	return grpcProtoSuccess(gcpStage4VisionProduct(project, location, productID, "Stackyard Product "+productID, "general-v1"))
}

func gcpStage4GRPCVisionUpdateProduct(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &visionpb.UpdateProductRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	if req.GetProduct() == nil {
		return grpcInvalidArgument("product-required")
	}
	if req.GetUpdateMask() == nil || len(req.GetUpdateMask().GetPaths()) == 0 {
		return grpcInvalidArgument("update_mask-required")
	}
	project, location, productID, ok := parseGCPVisionProductName(strings.TrimSpace(req.GetProduct().GetName()))
	if !ok {
		return grpcInvalidArgument("product.name-required")
	}
	displayName := strings.TrimSpace(req.GetProduct().GetDisplayName())
	if displayName == "" {
		displayName = "Stackyard Product " + productID
	}
	category := strings.TrimSpace(req.GetProduct().GetProductCategory())
	if category == "" {
		category = "general-v1"
	}
	return grpcProtoSuccess(gcpStage4VisionProduct(project, location, productID, displayName, category))
}

func gcpStage4GRPCVisionDeleteProduct(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &visionpb.DeleteProductRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	_, _, productID, ok := parseGCPVisionProductName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPVisionMissingID(productID) {
		return grpcNotFound("product-not-found")
	}
	return grpcProtoSuccess(&emptypb.Empty{})
}

func gcpStage4GRPCVisionCreateReferenceImage(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &visionpb.CreateReferenceImageRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, productID, ok := parseGCPVisionProductName(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	if !isGCPVisionResourceID(req.GetReferenceImageId()) {
		return grpcInvalidArgument("reference_image_id-required")
	}
	if req.GetReferenceImage() == nil {
		return grpcInvalidArgument("reference_image-required")
	}
	uri := strings.TrimSpace(req.GetReferenceImage().GetUri())
	if !isGCPVisionGCSURI(uri) {
		return grpcInvalidArgument("reference_image.uri-invalid")
	}
	return grpcProtoSuccess(gcpStage4VisionReferenceImage(project, location, productID, req.GetReferenceImageId(), uri))
}

func gcpStage4GRPCVisionDeleteReferenceImage(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &visionpb.DeleteReferenceImageRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	_, _, _, refID, ok := parseGCPVisionReferenceImageName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPVisionMissingID(refID) {
		return grpcNotFound("reference_image-not-found")
	}
	return grpcProtoSuccess(&emptypb.Empty{})
}

func gcpStage4GRPCVisionListReferenceImages(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &visionpb.ListReferenceImagesRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, productID, ok := parseGCPVisionProductName(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	start, end, next, reason, ok := gcpStage4VisionPageWindow(req.GetPageSize(), req.GetPageToken(), 100, 2)
	if !ok {
		return grpcInvalidArgument(reason)
	}
	items := []*visionpb.ReferenceImage{
		gcpStage4VisionReferenceImage(project, location, productID, "reference-image-1", "gs://stackyard-inputs/reference-image-1.jpg"),
		gcpStage4VisionReferenceImage(project, location, productID, "reference-image-2", "gs://stackyard-inputs/reference-image-2.jpg"),
	}
	return grpcProtoSuccess(&visionpb.ListReferenceImagesResponse{
		ReferenceImages: items[start:end],
		NextPageToken:   next,
	})
}

func gcpStage4GRPCVisionGetReferenceImage(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &visionpb.GetReferenceImageRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, productID, refID, ok := parseGCPVisionReferenceImageName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	if isGCPVisionMissingID(refID) {
		return grpcNotFound("reference_image-not-found")
	}
	return grpcProtoSuccess(gcpStage4VisionReferenceImage(project, location, productID, refID, "gs://stackyard-inputs/"+refID+".jpg"))
}

func gcpStage4GRPCVisionAddProductToProductSet(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &visionpb.AddProductToProductSetRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, _, ok := parseGCPVisionProductSetName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	pproject, plocation, _, ok := parseGCPVisionProductName(strings.TrimSpace(req.GetProduct()))
	if !ok {
		return grpcInvalidArgument("product-required")
	}
	if project != pproject || location != plocation {
		return grpcInvalidArgument("product-project-location-mismatch")
	}
	return grpcProtoSuccess(&emptypb.Empty{})
}

func gcpStage4GRPCVisionRemoveProductFromProductSet(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &visionpb.RemoveProductFromProductSetRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, _, ok := parseGCPVisionProductSetName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	pproject, plocation, _, ok := parseGCPVisionProductName(strings.TrimSpace(req.GetProduct()))
	if !ok {
		return grpcInvalidArgument("product-required")
	}
	if project != pproject || location != plocation {
		return grpcInvalidArgument("product-project-location-mismatch")
	}
	return grpcProtoSuccess(&emptypb.Empty{})
}

func gcpStage4GRPCVisionListProductsInProductSet(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &visionpb.ListProductsInProductSetRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, _, ok := parseGCPVisionProductSetName(strings.TrimSpace(req.GetName()))
	if !ok {
		return grpcInvalidArgument("name-required")
	}
	start, end, next, reason, ok := gcpStage4VisionPageWindow(req.GetPageSize(), req.GetPageToken(), 100, 2)
	if !ok {
		return grpcInvalidArgument(reason)
	}
	items := []*visionpb.Product{
		gcpStage4VisionProduct(project, location, "product-1", "Stackyard Product 1", "general-v1"),
		gcpStage4VisionProduct(project, location, "product-2", "Stackyard Product 2", "general-v1"),
	}
	return grpcProtoSuccess(&visionpb.ListProductsInProductSetResponse{
		Products:      items[start:end],
		NextPageToken: next,
	})
}

func gcpStage4GRPCVisionImportProductSets(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &visionpb.ImportProductSetsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := parseGCPVisionParentName(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	csvURI := strings.TrimSpace(req.GetInputConfig().GetGcsSource().GetCsvFileUri())
	if csvURI == "" || !isGCPVisionGCSURI(csvURI) {
		return grpcInvalidArgument("input_config.gcs_source.csv_file_uri-invalid")
	}
	return grpcProtoSuccess(gcpStage4VisionOperation(
		project,
		location,
		"importProductSets.op-1",
		&visionpb.BatchOperationMetadata{
			State:      visionpb.BatchOperationMetadata_SUCCESSFUL,
			SubmitTime: timestamppb.New(gcpStage4ReferenceTime),
			EndTime:    timestamppb.New(gcpStage4ReferenceTime.Add(2 * time.Second)),
		},
		&visionpb.ImportProductSetsResponse{
			ReferenceImages: []*visionpb.ReferenceImage{
				gcpStage4VisionReferenceImage(project, location, "product-1", "reference-image-1", "gs://stackyard-inputs/reference-image-1.jpg"),
			},
			Statuses: []*statuspb.Status{{Code: 0, Message: "ok"}},
		},
	))
}

func gcpStage4GRPCVisionPurgeProducts(grpcReqBody []byte) ([]byte, string, string, bool) {
	req := &visionpb.PurgeProductsRequest{}
	if !decodeGRPCUnaryProtoRequest(grpcReqBody, req) {
		return grpcInvalidArgument("request-body-invalid")
	}
	project, location, ok := parseGCPVisionParentName(strings.TrimSpace(req.GetParent()))
	if !ok {
		return grpcInvalidArgument("parent-required")
	}
	hasPurgeConfig := req.GetProductSetPurgeConfig() != nil
	hasDeleteOrphan := req.GetDeleteOrphanProducts()
	if !hasPurgeConfig && !hasDeleteOrphan {
		return grpcInvalidArgument("purge_target-required")
	}
	if hasPurgeConfig && strings.TrimSpace(req.GetProductSetPurgeConfig().GetProductSetId()) == "" {
		return grpcInvalidArgument("product_set_purge_config.product_set_id-required")
	}
	return grpcProtoSuccess(gcpStage4VisionOperation(
		project,
		location,
		"purgeProducts.op-1",
		&visionpb.BatchOperationMetadata{
			State:      visionpb.BatchOperationMetadata_SUCCESSFUL,
			SubmitTime: timestamppb.New(gcpStage4ReferenceTime),
			EndTime:    timestamppb.New(gcpStage4ReferenceTime.Add(2 * time.Second)),
		},
		&emptypb.Empty{},
	))
}

func gcpStage4VisionProjectLocationFromParent(parent string) (project, location string) {
	if p, l, ok := parseGCPVisionParentName(strings.TrimSpace(parent)); ok {
		return p, l
	}
	return "stackyard", "us-central1"
}

func gcpStage4VisionValidateFeatures(features []*visionpb.Feature) ([]visionpb.Feature_Type, bool) {
	if len(features) == 0 {
		return nil, false
	}
	out := make([]visionpb.Feature_Type, 0, len(features))
	for _, feature := range features {
		if feature == nil || feature.GetType() == visionpb.Feature_TYPE_UNSPECIFIED {
			return nil, false
		}
		out = append(out, feature.GetType())
	}
	return out, true
}

func gcpStage4VisionImageURI(image *visionpb.Image) string {
	if image == nil {
		return ""
	}
	source := image.GetSource()
	if source != nil {
		if uri := strings.TrimSpace(source.GetImageUri()); uri != "" {
			return uri
		}
		if uri := strings.TrimSpace(source.GetGcsImageUri()); uri != "" {
			return uri
		}
	}
	if len(image.GetContent()) > 0 {
		return "inline://image"
	}
	return ""
}

func gcpStage4VisionOperation(project, location, operationID string, metadata proto.Message, response proto.Message) *longrunningpb.Operation {
	out := &longrunningpb.Operation{
		Name: fmt.Sprintf("projects/%s/locations/%s/operations/%s", project, location, operationID),
		Done: true,
	}
	if metadata != nil {
		if packed, err := anypb.New(metadata); err == nil {
			out.Metadata = packed
		}
	}
	if response != nil {
		if packed, err := anypb.New(response); err == nil {
			out.Result = &longrunningpb.Operation_Response{Response: packed}
		}
	}
	return out
}

func gcpStage4VisionPageWindow(pageSize int32, pageToken string, max, total int) (start, end int, nextPageToken, reason string, ok bool) {
	if pageSize < 0 {
		return 0, 0, "", "page_size-negative", false
	}
	if pageSize > int32(max) {
		return 0, 0, "", "page_size-too-large", false
	}
	start = 0
	if strings.TrimSpace(pageToken) != "" {
		parsed, err := strconv.Atoi(strings.TrimSpace(pageToken))
		if err != nil || parsed < 0 {
			return 0, 0, "", "page_token-invalid", false
		}
		start = parsed
	}
	if start > total {
		return 0, 0, "", "page_token-out-of-range", false
	}
	end = total
	if pageSize > 0 && start+int(pageSize) < end {
		end = start + int(pageSize)
	}
	nextPageToken = ""
	if end < total {
		nextPageToken = strconv.Itoa(end)
	}
	return start, end, nextPageToken, "", true
}

func gcpStage4VisionProductSet(project, location, productSetID, displayName string) *visionpb.ProductSet {
	return &visionpb.ProductSet{
		Name:        fmt.Sprintf("projects/%s/locations/%s/productSets/%s", project, location, productSetID),
		DisplayName: displayName,
		IndexTime:   timestamppb.New(gcpStage4ReferenceTime),
	}
}

func gcpStage4VisionProduct(project, location, productID, displayName, category string) *visionpb.Product {
	return &visionpb.Product{
		Name:            fmt.Sprintf("projects/%s/locations/%s/products/%s", project, location, productID),
		DisplayName:     displayName,
		Description:     "Stackyard emulated product",
		ProductCategory: category,
		ProductLabels: []*visionpb.Product_KeyValue{
			{Key: "env", Value: "test"},
			{Key: "service", Value: "vision"},
		},
	}
}

func gcpStage4VisionReferenceImage(project, location, productID, imageID, uri string) *visionpb.ReferenceImage {
	return &visionpb.ReferenceImage{
		Name: fmt.Sprintf("projects/%s/locations/%s/products/%s/referenceImages/%s", project, location, productID, imageID),
		Uri:  uri,
		BoundingPolys: []*visionpb.BoundingPoly{
			{
				Vertices: []*visionpb.Vertex{
					{X: 0, Y: 0},
					{X: 512, Y: 0},
					{X: 512, Y: 512},
					{X: 0, Y: 512},
				},
			},
		},
	}
}

func gcpStage4VisionAnnotateImageResponse(project, location, imageURI string, features []visionpb.Feature_Type) *visionpb.AnnotateImageResponse {
	featureSet := map[visionpb.Feature_Type]struct{}{}
	for _, feature := range features {
		featureSet[feature] = struct{}{}
	}
	if len(featureSet) == 0 {
		featureSet[visionpb.Feature_LABEL_DETECTION] = struct{}{}
	}

	out := &visionpb.AnnotateImageResponse{}
	if _, ok := featureSet[visionpb.Feature_LABEL_DETECTION]; ok {
		out.LabelAnnotations = []*visionpb.EntityAnnotation{
			{Mid: "/m/stackyard", Description: "stackyard-label", Score: 0.98},
		}
	}
	if _, ok := featureSet[visionpb.Feature_TEXT_DETECTION]; ok {
		out.TextAnnotations = []*visionpb.EntityAnnotation{
			{Locale: "en", Description: "Stackyard Vision Text"},
		}
	}
	if _, ok := featureSet[visionpb.Feature_DOCUMENT_TEXT_DETECTION]; ok {
		out.FullTextAnnotation = &visionpb.TextAnnotation{
			Text: "Stackyard Vision Document",
			Pages: []*visionpb.Page{
				{Width: 1000, Height: 1000},
			},
		}
	}
	if _, ok := featureSet[visionpb.Feature_SAFE_SEARCH_DETECTION]; ok {
		out.SafeSearchAnnotation = &visionpb.SafeSearchAnnotation{
			Adult:    visionpb.Likelihood_VERY_UNLIKELY,
			Spoof:    visionpb.Likelihood_VERY_UNLIKELY,
			Medical:  visionpb.Likelihood_VERY_UNLIKELY,
			Violence: visionpb.Likelihood_VERY_UNLIKELY,
			Racy:     visionpb.Likelihood_VERY_UNLIKELY,
		}
	}
	if _, ok := featureSet[visionpb.Feature_WEB_DETECTION]; ok {
		out.WebDetection = &visionpb.WebDetection{
			BestGuessLabels: []*visionpb.WebDetection_WebLabel{{Label: "stackyard-web"}},
			WebEntities: []*visionpb.WebDetection_WebEntity{
				{EntityId: "stackyard", Score: 0.75, Description: "Stackyard"},
			},
		}
	}
	if _, ok := featureSet[visionpb.Feature_OBJECT_LOCALIZATION]; ok {
		out.LocalizedObjectAnnotations = []*visionpb.LocalizedObjectAnnotation{
			{Name: "Object", Score: 0.88},
		}
	}
	if _, ok := featureSet[visionpb.Feature_PRODUCT_SEARCH]; ok {
		out.ProductSearchResults = &visionpb.ProductSearchResults{
			Results: []*visionpb.ProductSearchResults_Result{
				{
					Score:   0.91,
					Product: gcpStage4VisionProduct(project, location, "product-1", "Stackyard Product 1", "general-v1"),
					Image:   fmt.Sprintf("projects/%s/locations/%s/products/product-1/referenceImages/reference-image-1", project, location),
				},
			},
		}
	}
	if _, ok := featureSet[visionpb.Feature_IMAGE_PROPERTIES]; ok {
		out.ImagePropertiesAnnotation = &visionpb.ImageProperties{
			DominantColors: &visionpb.DominantColorsAnnotation{
				Colors: []*visionpb.ColorInfo{{Score: 0.77}},
			},
		}
	}
	if _, ok := featureSet[visionpb.Feature_CROP_HINTS]; ok {
		out.CropHintsAnnotation = &visionpb.CropHintsAnnotation{
			CropHints: []*visionpb.CropHint{{Confidence: 0.85}},
		}
	}
	if _, ok := featureSet[visionpb.Feature_FACE_DETECTION]; ok {
		out.FaceAnnotations = []*visionpb.FaceAnnotation{{DetectionConfidence: 0.76}}
	}
	if _, ok := featureSet[visionpb.Feature_LANDMARK_DETECTION]; ok {
		out.LandmarkAnnotations = []*visionpb.EntityAnnotation{
			{Description: "Stackyard Landmark", Score: 0.81},
		}
	}
	if _, ok := featureSet[visionpb.Feature_LOGO_DETECTION]; ok {
		out.LogoAnnotations = []*visionpb.EntityAnnotation{
			{Description: "Stackyard Logo", Score: 0.82},
		}
	}
	if strings.TrimSpace(imageURI) != "" {
		out.Context = &visionpb.ImageAnnotationContext{Uri: imageURI, PageNumber: 1}
	}
	return out
}
