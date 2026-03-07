package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	gcpVisionImageAnnotatorGRPCPrefix = "/gcp/google.cloud.vision.v1.ImageAnnotator/"
	gcpVisionProductSearchGRPCPrefix  = "/gcp/google.cloud.vision.v1.ProductSearch/"
)

var gcpVisionReferenceTime = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

var gcpVisionFeatureByNumber = map[int]string{
	1:  "FACE_DETECTION",
	2:  "LANDMARK_DETECTION",
	3:  "LOGO_DETECTION",
	4:  "LABEL_DETECTION",
	5:  "TEXT_DETECTION",
	6:  "SAFE_SEARCH_DETECTION",
	7:  "IMAGE_PROPERTIES",
	9:  "CROP_HINTS",
	10: "WEB_DETECTION",
	11: "DOCUMENT_TEXT_DETECTION",
	12: "PRODUCT_SEARCH",
	19: "OBJECT_LOCALIZATION",
}

func (s *Server) handleGCPVisionRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_vision(w, r) {
		return true
	}

	path := normalizeGCPVisionPath(rawRequestPath(r))
	if isGCPVisionLocationRequest(r, path) {
		if r.Method != http.MethodGet {
			return false
		}
		if handleGCPVisionListLocations(w, r, path) {
			return true
		}
		if handleGCPVisionGetLocation(w, path) {
			return true
		}
		return false
	}

	if strings.HasPrefix(path, gcpVisionImageAnnotatorGRPCPrefix) {
		if r.Method != http.MethodPost {
			return false
		}
		body, ok := decodeGCPVisionJSONBody(w, r, path)
		if !ok {
			return true
		}
		method := strings.TrimSpace(strings.TrimPrefix(path, gcpVisionImageAnnotatorGRPCPrefix))
		if method == "" {
			respondProviderNotImplemented(w, providerGCP, path)
			return true
		}
		if handleGCPVisionImageAnnotatorMethod(w, path, method, body) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	}

	if strings.HasPrefix(path, gcpVisionProductSearchGRPCPrefix) {
		if r.Method != http.MethodPost {
			return false
		}
		body, ok := decodeGCPVisionJSONBody(w, r, path)
		if !ok {
			return true
		}
		method := strings.TrimSpace(strings.TrimPrefix(path, gcpVisionProductSearchGRPCPrefix))
		if method == "" {
			respondProviderNotImplemented(w, providerGCP, path)
			return true
		}
		if handleGCPVisionProductSearchMethod(w, path, method, body) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	}

	if !isGCPVisionPath(path, hasGCPVisionHint(r)) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPVisionListOperations(w, r, path) {
			return true
		}
		if handleGCPVisionGetOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPVisionCancelOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPVisionDeleteOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func normalizeGCPVisionPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "%3A", ":")
	path = strings.ReplaceAll(path, "%3a", ":")
	return path
}

func hasGCPVisionHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "vision", "vision-apiv1", "vision_apiv1", "cloud-vision", "cloud_vision", "cloudvision", "gcp-cloud-vision",
		"vision-v2", "vision_v2", "vision-v2-apiv1", "vision_v2_apiv1", "cloud-vision-v2", "cloud_vision_v2", "gcp-cloud-vision-v2":
		return true
	}
	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-vision-apiv1") ||
		strings.Contains(ua, "cloud.google.com/go/vision/apiv1") ||
		strings.Contains(ua, "stackyard-vision-v2-apiv1") ||
		strings.Contains(ua, "cloud.google.com/go/vision/v2/apiv1")
}

func isGCPVisionLocationRequest(r *http.Request, path string) bool {
	if !hasGCPVisionHint(r) {
		return false
	}
	_, _, _, ok := parseGCPProjectLocationPath(path)
	return ok
}

func isGCPVisionPath(path string, includeHint bool) bool {
	if strings.HasPrefix(path, gcpVisionImageAnnotatorGRPCPrefix) || strings.HasPrefix(path, gcpVisionProductSearchGRPCPrefix) {
		return true
	}
	if !includeHint {
		return false
	}
	if _, _, ok := parseGCPVisionOperationCollectionPath(path); ok {
		return true
	}
	if _, _, _, ok := parseGCPVisionOperationPath(path); ok {
		return true
	}
	if _, _, _, action, ok := parseGCPVisionOperationActionPath(path); ok {
		return action == "cancel"
	}
	return false
}

func decodeGCPVisionJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		return map[string]any{}, true
	}
	defer r.Body.Close()

	dec := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	dec.UseNumber()

	var body map[string]any
	if err := dec.Decode(&body); err != nil {
		if err == io.EOF {
			return map[string]any{}, true
		}
		respondGCPVisionInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func handleGCPVisionImageAnnotatorMethod(w http.ResponseWriter, path, method string, body map[string]any) bool {
	switch method {
	case "BatchAnnotateImages":
		return handleGCPVisionBatchAnnotateImages(w, path, body)
	case "BatchAnnotateFiles":
		return handleGCPVisionBatchAnnotateFiles(w, path, body)
	case "AsyncBatchAnnotateImages":
		return handleGCPVisionAsyncBatchAnnotateImages(w, path, body)
	case "AsyncBatchAnnotateFiles":
		return handleGCPVisionAsyncBatchAnnotateFiles(w, path, body)
	default:
		return false
	}
}

func handleGCPVisionProductSearchMethod(w http.ResponseWriter, path, method string, body map[string]any) bool {
	switch method {
	case "CreateProductSet":
		return handleGCPVisionCreateProductSet(w, path, body)
	case "ListProductSets":
		return handleGCPVisionListProductSets(w, path, body)
	case "GetProductSet":
		return handleGCPVisionGetProductSet(w, path, body)
	case "UpdateProductSet":
		return handleGCPVisionUpdateProductSet(w, path, body)
	case "DeleteProductSet":
		return handleGCPVisionDeleteProductSet(w, path, body)
	case "CreateProduct":
		return handleGCPVisionCreateProduct(w, path, body)
	case "ListProducts":
		return handleGCPVisionListProducts(w, path, body)
	case "GetProduct":
		return handleGCPVisionGetProduct(w, path, body)
	case "UpdateProduct":
		return handleGCPVisionUpdateProduct(w, path, body)
	case "DeleteProduct":
		return handleGCPVisionDeleteProduct(w, path, body)
	case "CreateReferenceImage":
		return handleGCPVisionCreateReferenceImage(w, path, body)
	case "DeleteReferenceImage":
		return handleGCPVisionDeleteReferenceImage(w, path, body)
	case "ListReferenceImages":
		return handleGCPVisionListReferenceImages(w, path, body)
	case "GetReferenceImage":
		return handleGCPVisionGetReferenceImage(w, path, body)
	case "AddProductToProductSet":
		return handleGCPVisionAddProductToProductSet(w, path, body)
	case "RemoveProductFromProductSet":
		return handleGCPVisionRemoveProductFromProductSet(w, path, body)
	case "ListProductsInProductSet":
		return handleGCPVisionListProductsInProductSet(w, path, body)
	case "ImportProductSets":
		return handleGCPVisionImportProductSets(w, path, body)
	case "PurgeProducts":
		return handleGCPVisionPurgeProducts(w, path, body)
	default:
		return false
	}
}

func handleGCPVisionListLocations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, _, list, ok := parseGCPProjectLocationPath(path)
	if !ok || !list {
		return false
	}
	pageSize, start, valid := parseGCPVisionQueryPagination(w, r, path, 100, 1000)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpVisionLocation(project, "us-central1"),
		gcpVisionLocation(project, "us-east1"),
		gcpVisionLocation(project, "global"),
	}
	return respondGCPVisionList(w, "locations", items, pageSize, start, path)
}

func handleGCPVisionGetLocation(w http.ResponseWriter, path string) bool {
	project, location, list, ok := parseGCPProjectLocationPath(path)
	if !ok || list {
		return false
	}
	respondJSON(w, http.StatusOK, gcpVisionLocation(project, location))
	return true
}

func handleGCPVisionBatchAnnotateImages(w http.ResponseWriter, path string, body map[string]any) bool {
	project, location := gcpVisionProjectLocationFromParent(gcpVisionString(body, "parent"), "stackyard", "us-central1")
	requests := gcpVisionBodyArray(body, "requests")
	if len(requests) == 0 {
		respondGCPVisionInvalidArgument(w, path, "requests is required")
		return true
	}

	responses := make([]any, 0, len(requests))
	for i, item := range requests {
		req, ok := item.(map[string]any)
		if !ok {
			respondGCPVisionInvalidArgument(w, path, "requests entries must be objects")
			return true
		}
		features, errMessage := gcpVisionParseFeatureNames(req)
		if errMessage != "" {
			respondGCPVisionInvalidArgument(w, path, errMessage)
			return true
		}
		image := gcpVisionBodyMap(req, "image")
		if len(image) == 0 {
			respondGCPVisionInvalidArgument(w, path, "requests.image is required")
			return true
		}
		if !gcpVisionImageInputPresent(image) {
			respondGCPVisionInvalidArgument(w, path, "requests.image requires content or source.imageUri")
			return true
		}
		imageURI := gcpVisionImageURIFromMap(image)
		if imageURI == "" {
			imageURI = fmt.Sprintf("inline://image-%d", i+1)
		}
		responses = append(responses, gcpVisionAnnotateImageResponseMap(project, location, imageURI, features))
	}

	respondJSON(w, http.StatusOK, map[string]any{"responses": responses})
	return true
}

func handleGCPVisionBatchAnnotateFiles(w http.ResponseWriter, path string, body map[string]any) bool {
	requests := gcpVisionBodyArray(body, "requests")
	if len(requests) == 0 {
		respondGCPVisionInvalidArgument(w, path, "requests is required")
		return true
	}

	responses := make([]any, 0, len(requests))
	for _, item := range requests {
		req, ok := item.(map[string]any)
		if !ok {
			respondGCPVisionInvalidArgument(w, path, "requests entries must be objects")
			return true
		}
		features, errMessage := gcpVisionParseFeatureNames(req)
		if errMessage != "" {
			respondGCPVisionInvalidArgument(w, path, errMessage)
			return true
		}
		inputConfig := gcpVisionBodyMap(req, "inputConfig", "input_config")
		if len(inputConfig) == 0 {
			respondGCPVisionInvalidArgument(w, path, "requests.inputConfig is required")
			return true
		}
		fileURI := gcpVisionInputConfigURI(inputConfig)
		if fileURI == "" || !isGCPVisionGCSURI(fileURI) {
			respondGCPVisionInvalidArgument(w, path, "requests.inputConfig.gcsSource.uri must be a valid gs:// URI")
			return true
		}
		if pages := gcpVisionIntSlice(req["pages"]); len(pages) > 5 {
			respondGCPVisionInvalidArgument(w, path, "requests.pages must contain at most 5 entries")
			return true
		}
		responses = append(responses, map[string]any{
			"inputConfig": map[string]any{
				"gcsSource": map[string]any{"uri": fileURI},
				"mimeType":  gcpVisionInputConfigMimeType(inputConfig),
			},
			"responses":  []any{gcpVisionAnnotateImageResponseMap("stackyard", "us-central1", fileURI, features)},
			"totalPages": 1,
		})
	}

	respondJSON(w, http.StatusOK, map[string]any{"responses": responses})
	return true
}

func handleGCPVisionAsyncBatchAnnotateImages(w http.ResponseWriter, path string, body map[string]any) bool {
	project, location := gcpVisionProjectLocationFromParent(gcpVisionString(body, "parent"), "stackyard", "us-central1")
	requests := gcpVisionBodyArray(body, "requests")
	if len(requests) == 0 {
		respondGCPVisionInvalidArgument(w, path, "requests is required")
		return true
	}
	for _, item := range requests {
		req, ok := item.(map[string]any)
		if !ok {
			respondGCPVisionInvalidArgument(w, path, "requests entries must be objects")
			return true
		}
		features, errMessage := gcpVisionParseFeatureNames(req)
		if errMessage != "" {
			respondGCPVisionInvalidArgument(w, path, errMessage)
			return true
		}
		_ = features
		image := gcpVisionBodyMap(req, "image")
		if len(image) == 0 || !gcpVisionImageInputPresent(image) {
			respondGCPVisionInvalidArgument(w, path, "requests.image requires content or source.imageUri")
			return true
		}
	}
	outputConfig := gcpVisionBodyMap(body, "outputConfig", "output_config")
	outputURI := gcpVisionOutputConfigURI(outputConfig)
	if outputURI == "" || !isGCPVisionGCSOutputURI(outputURI) {
		respondGCPVisionInvalidArgument(w, path, "outputConfig.gcsDestination.uri must be a valid gs:// URI")
		return true
	}

	opID := "asyncBatchAnnotateImages.op-1"
	operation := gcpVisionOperationFixture(project, location, opID, true,
		map[string]any{
			"@type":      "type.googleapis.com/google.cloud.vision.v1.OperationMetadata",
			"state":      "DONE",
			"createTime": gcpVisionReferenceTime.Format(time.RFC3339Nano),
			"updateTime": gcpVisionReferenceTime.Add(2 * time.Second).Format(time.RFC3339Nano),
		},
		map[string]any{
			"@type": "type.googleapis.com/google.cloud.vision.v1.AsyncBatchAnnotateImagesResponse",
			"outputConfig": map[string]any{
				"gcsDestination": map[string]any{"uri": outputURI},
				"batchSize":      gcpVisionOutputBatchSize(outputConfig),
			},
		},
	)
	respondJSON(w, http.StatusOK, operation)
	return true
}

func handleGCPVisionAsyncBatchAnnotateFiles(w http.ResponseWriter, path string, body map[string]any) bool {
	project, location := gcpVisionProjectLocationFromParent(gcpVisionString(body, "parent"), "stackyard", "us-central1")
	requests := gcpVisionBodyArray(body, "requests")
	if len(requests) == 0 {
		respondGCPVisionInvalidArgument(w, path, "requests is required")
		return true
	}

	responses := make([]any, 0, len(requests))
	for _, item := range requests {
		req, ok := item.(map[string]any)
		if !ok {
			respondGCPVisionInvalidArgument(w, path, "requests entries must be objects")
			return true
		}
		inputConfig := gcpVisionBodyMap(req, "inputConfig", "input_config")
		if len(inputConfig) == 0 {
			respondGCPVisionInvalidArgument(w, path, "requests.inputConfig is required")
			return true
		}
		if uri := gcpVisionInputConfigURI(inputConfig); uri == "" || !isGCPVisionGCSURI(uri) {
			respondGCPVisionInvalidArgument(w, path, "requests.inputConfig.gcsSource.uri must be a valid gs:// URI")
			return true
		}
		features, errMessage := gcpVisionParseFeatureNames(req)
		if errMessage != "" {
			respondGCPVisionInvalidArgument(w, path, errMessage)
			return true
		}
		_ = features
		outputConfig := gcpVisionBodyMap(req, "outputConfig", "output_config")
		outputURI := gcpVisionOutputConfigURI(outputConfig)
		if outputURI == "" || !isGCPVisionGCSOutputURI(outputURI) {
			respondGCPVisionInvalidArgument(w, path, "requests.outputConfig.gcsDestination.uri must be a valid gs:// URI")
			return true
		}
		responses = append(responses, map[string]any{
			"outputConfig": map[string]any{
				"gcsDestination": map[string]any{"uri": outputURI},
				"batchSize":      gcpVisionOutputBatchSize(outputConfig),
			},
		})
	}

	opID := "asyncBatchAnnotateFiles.op-1"
	operation := gcpVisionOperationFixture(project, location, opID, true,
		map[string]any{
			"@type":      "type.googleapis.com/google.cloud.vision.v1.OperationMetadata",
			"state":      "DONE",
			"createTime": gcpVisionReferenceTime.Format(time.RFC3339Nano),
			"updateTime": gcpVisionReferenceTime.Add(2 * time.Second).Format(time.RFC3339Nano),
		},
		map[string]any{
			"@type":     "type.googleapis.com/google.cloud.vision.v1.AsyncBatchAnnotateFilesResponse",
			"responses": responses,
		},
	)
	respondJSON(w, http.StatusOK, operation)
	return true
}

func handleGCPVisionCreateProductSet(w http.ResponseWriter, path string, body map[string]any) bool {
	project, location, ok := parseGCPVisionParentName(gcpVisionString(body, "parent"))
	if !ok {
		respondGCPVisionInvalidArgument(w, path, "parent is required")
		return true
	}
	productSetID := strings.TrimSpace(gcpVisionString(body, "productSetId", "product_set_id"))
	if !isGCPVisionResourceID(productSetID) {
		respondGCPVisionInvalidArgument(w, path, "productSetId is required")
		return true
	}
	if strings.Contains(strings.ToLower(productSetID), "existing") {
		respondGCPVisionAlreadyExists(w, path, "product set already exists")
		return true
	}
	productSetReq := gcpVisionBodyMap(body, "productSet", "product_set")
	if len(productSetReq) == 0 {
		respondGCPVisionInvalidArgument(w, path, "productSet is required")
		return true
	}
	expectedName := fmt.Sprintf("projects/%s/locations/%s/productSets/%s", project, location, productSetID)
	if name := strings.TrimSpace(gcpVisionString(productSetReq, "name")); name != "" && name != expectedName {
		respondGCPVisionInvalidArgument(w, path, "productSet.name must match parent and productSetId")
		return true
	}
	displayName := strings.TrimSpace(gcpVisionString(productSetReq, "displayName", "display_name"))
	if displayName == "" {
		displayName = "Stackyard Product Set " + productSetID
	}
	respondJSON(w, http.StatusOK, gcpVisionProductSetFixture(project, location, productSetID, displayName))
	return true
}

func handleGCPVisionListProductSets(w http.ResponseWriter, path string, body map[string]any) bool {
	project, location, ok := parseGCPVisionParentName(gcpVisionString(body, "parent"))
	if !ok {
		respondGCPVisionInvalidArgument(w, path, "parent is required")
		return true
	}
	pageSize, start, valid := parseGCPVisionBodyPagination(w, path, body, 10, 100)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpVisionProductSetFixture(project, location, "product-set-1", "Stackyard Product Set 1"),
		gcpVisionProductSetFixture(project, location, "product-set-2", "Stackyard Product Set 2"),
	}
	return respondGCPVisionList(w, "productSets", items, pageSize, start, path)
}

func handleGCPVisionGetProductSet(w http.ResponseWriter, path string, body map[string]any) bool {
	project, location, productSetID, ok := parseGCPVisionProductSetName(gcpVisionString(body, "name"))
	if !ok {
		respondGCPVisionInvalidArgument(w, path, "name is required")
		return true
	}
	if isGCPVisionMissingID(productSetID) {
		respondGCPVisionNotFound(w, path, "product set not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpVisionProductSetFixture(project, location, productSetID, "Stackyard Product Set "+productSetID))
	return true
}

func handleGCPVisionUpdateProductSet(w http.ResponseWriter, path string, body map[string]any) bool {
	productSetReq := gcpVisionBodyMap(body, "productSet", "product_set")
	if len(productSetReq) == 0 {
		respondGCPVisionInvalidArgument(w, path, "productSet is required")
		return true
	}
	project, location, productSetID, ok := parseGCPVisionProductSetName(gcpVisionString(productSetReq, "name"))
	if !ok {
		respondGCPVisionInvalidArgument(w, path, "productSet.name is required")
		return true
	}
	if !gcpVisionHasUpdateMask(body, "updateMask", "update_mask") {
		respondGCPVisionInvalidArgument(w, path, "updateMask is required")
		return true
	}
	displayName := strings.TrimSpace(gcpVisionString(productSetReq, "displayName", "display_name"))
	if displayName == "" {
		displayName = "Stackyard Product Set " + productSetID
	}
	respondJSON(w, http.StatusOK, gcpVisionProductSetFixture(project, location, productSetID, displayName))
	return true
}

func handleGCPVisionDeleteProductSet(w http.ResponseWriter, path string, body map[string]any) bool {
	_, _, productSetID, ok := parseGCPVisionProductSetName(gcpVisionString(body, "name"))
	if !ok {
		respondGCPVisionInvalidArgument(w, path, "name is required")
		return true
	}
	if isGCPVisionMissingID(productSetID) {
		respondGCPVisionNotFound(w, path, "product set not found")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPVisionCreateProduct(w http.ResponseWriter, path string, body map[string]any) bool {
	project, location, ok := parseGCPVisionParentName(gcpVisionString(body, "parent"))
	if !ok {
		respondGCPVisionInvalidArgument(w, path, "parent is required")
		return true
	}
	productID := strings.TrimSpace(gcpVisionString(body, "productId", "product_id"))
	if !isGCPVisionResourceID(productID) {
		respondGCPVisionInvalidArgument(w, path, "productId is required")
		return true
	}
	if strings.Contains(strings.ToLower(productID), "existing") {
		respondGCPVisionAlreadyExists(w, path, "product already exists")
		return true
	}
	productReq := gcpVisionBodyMap(body, "product")
	if len(productReq) == 0 {
		respondGCPVisionInvalidArgument(w, path, "product is required")
		return true
	}
	expectedName := fmt.Sprintf("projects/%s/locations/%s/products/%s", project, location, productID)
	if name := strings.TrimSpace(gcpVisionString(productReq, "name")); name != "" && name != expectedName {
		respondGCPVisionInvalidArgument(w, path, "product.name must match parent and productId")
		return true
	}
	displayName := strings.TrimSpace(gcpVisionString(productReq, "displayName", "display_name"))
	if displayName == "" {
		displayName = "Stackyard Product " + productID
	}
	productCategory := strings.TrimSpace(gcpVisionString(productReq, "productCategory", "product_category"))
	if productCategory == "" {
		respondGCPVisionInvalidArgument(w, path, "product.productCategory is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpVisionProductFixture(project, location, productID, displayName, productCategory))
	return true
}

func handleGCPVisionListProducts(w http.ResponseWriter, path string, body map[string]any) bool {
	project, location, ok := parseGCPVisionParentName(gcpVisionString(body, "parent"))
	if !ok {
		respondGCPVisionInvalidArgument(w, path, "parent is required")
		return true
	}
	pageSize, start, valid := parseGCPVisionBodyPagination(w, path, body, 10, 100)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpVisionProductFixture(project, location, "product-1", "Stackyard Product 1", "general-v1"),
		gcpVisionProductFixture(project, location, "product-2", "Stackyard Product 2", "general-v1"),
	}
	return respondGCPVisionList(w, "products", items, pageSize, start, path)
}

func handleGCPVisionGetProduct(w http.ResponseWriter, path string, body map[string]any) bool {
	project, location, productID, ok := parseGCPVisionProductName(gcpVisionString(body, "name"))
	if !ok {
		respondGCPVisionInvalidArgument(w, path, "name is required")
		return true
	}
	if isGCPVisionMissingID(productID) {
		respondGCPVisionNotFound(w, path, "product not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpVisionProductFixture(project, location, productID, "Stackyard Product "+productID, "general-v1"))
	return true
}

func handleGCPVisionUpdateProduct(w http.ResponseWriter, path string, body map[string]any) bool {
	productReq := gcpVisionBodyMap(body, "product")
	if len(productReq) == 0 {
		respondGCPVisionInvalidArgument(w, path, "product is required")
		return true
	}
	project, location, productID, ok := parseGCPVisionProductName(gcpVisionString(productReq, "name"))
	if !ok {
		respondGCPVisionInvalidArgument(w, path, "product.name is required")
		return true
	}
	if !gcpVisionHasUpdateMask(body, "updateMask", "update_mask") {
		respondGCPVisionInvalidArgument(w, path, "updateMask is required")
		return true
	}
	displayName := strings.TrimSpace(gcpVisionString(productReq, "displayName", "display_name"))
	if displayName == "" {
		displayName = "Stackyard Product " + productID
	}
	category := strings.TrimSpace(gcpVisionString(productReq, "productCategory", "product_category"))
	if category == "" {
		category = "general-v1"
	}
	respondJSON(w, http.StatusOK, gcpVisionProductFixture(project, location, productID, displayName, category))
	return true
}

func handleGCPVisionDeleteProduct(w http.ResponseWriter, path string, body map[string]any) bool {
	_, _, productID, ok := parseGCPVisionProductName(gcpVisionString(body, "name"))
	if !ok {
		respondGCPVisionInvalidArgument(w, path, "name is required")
		return true
	}
	if isGCPVisionMissingID(productID) {
		respondGCPVisionNotFound(w, path, "product not found")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPVisionCreateReferenceImage(w http.ResponseWriter, path string, body map[string]any) bool {
	project, location, productID, ok := parseGCPVisionProductName(gcpVisionString(body, "parent"))
	if !ok {
		respondGCPVisionInvalidArgument(w, path, "parent is required")
		return true
	}
	refID := strings.TrimSpace(gcpVisionString(body, "referenceImageId", "reference_image_id"))
	if !isGCPVisionResourceID(refID) {
		respondGCPVisionInvalidArgument(w, path, "referenceImageId is required")
		return true
	}
	refReq := gcpVisionBodyMap(body, "referenceImage", "reference_image")
	if len(refReq) == 0 {
		respondGCPVisionInvalidArgument(w, path, "referenceImage is required")
		return true
	}
	uri := strings.TrimSpace(gcpVisionString(refReq, "uri"))
	if !isGCPVisionGCSURI(uri) {
		respondGCPVisionInvalidArgument(w, path, "referenceImage.uri must be a valid gs:// URI")
		return true
	}
	respondJSON(w, http.StatusOK, gcpVisionReferenceImageFixture(project, location, productID, refID, uri))
	return true
}

func handleGCPVisionDeleteReferenceImage(w http.ResponseWriter, path string, body map[string]any) bool {
	_, _, _, refID, ok := parseGCPVisionReferenceImageName(gcpVisionString(body, "name"))
	if !ok {
		respondGCPVisionInvalidArgument(w, path, "name is required")
		return true
	}
	if isGCPVisionMissingID(refID) {
		respondGCPVisionNotFound(w, path, "reference image not found")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPVisionListReferenceImages(w http.ResponseWriter, path string, body map[string]any) bool {
	project, location, productID, ok := parseGCPVisionProductName(gcpVisionString(body, "parent"))
	if !ok {
		respondGCPVisionInvalidArgument(w, path, "parent is required")
		return true
	}
	pageSize, start, valid := parseGCPVisionBodyPagination(w, path, body, 10, 100)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpVisionReferenceImageFixture(project, location, productID, "reference-image-1", "gs://stackyard-inputs/reference-image-1.jpg"),
		gcpVisionReferenceImageFixture(project, location, productID, "reference-image-2", "gs://stackyard-inputs/reference-image-2.jpg"),
	}
	return respondGCPVisionList(w, "referenceImages", items, pageSize, start, path)
}

func handleGCPVisionGetReferenceImage(w http.ResponseWriter, path string, body map[string]any) bool {
	project, location, productID, refID, ok := parseGCPVisionReferenceImageName(gcpVisionString(body, "name"))
	if !ok {
		respondGCPVisionInvalidArgument(w, path, "name is required")
		return true
	}
	if isGCPVisionMissingID(refID) {
		respondGCPVisionNotFound(w, path, "reference image not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpVisionReferenceImageFixture(project, location, productID, refID, "gs://stackyard-inputs/"+refID+".jpg"))
	return true
}

func handleGCPVisionAddProductToProductSet(w http.ResponseWriter, path string, body map[string]any) bool {
	project, location, _, ok := parseGCPVisionProductSetName(gcpVisionString(body, "name"))
	if !ok {
		respondGCPVisionInvalidArgument(w, path, "name is required")
		return true
	}
	pproject, plocation, _, ok := parseGCPVisionProductName(gcpVisionString(body, "product"))
	if !ok {
		respondGCPVisionInvalidArgument(w, path, "product is required")
		return true
	}
	if project != pproject || location != plocation {
		respondGCPVisionInvalidArgument(w, path, "product must match product set project/location")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPVisionRemoveProductFromProductSet(w http.ResponseWriter, path string, body map[string]any) bool {
	project, location, _, ok := parseGCPVisionProductSetName(gcpVisionString(body, "name"))
	if !ok {
		respondGCPVisionInvalidArgument(w, path, "name is required")
		return true
	}
	pproject, plocation, _, ok := parseGCPVisionProductName(gcpVisionString(body, "product"))
	if !ok {
		respondGCPVisionInvalidArgument(w, path, "product is required")
		return true
	}
	if project != pproject || location != plocation {
		respondGCPVisionInvalidArgument(w, path, "product must match product set project/location")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPVisionListProductsInProductSet(w http.ResponseWriter, path string, body map[string]any) bool {
	project, location, _, ok := parseGCPVisionProductSetName(gcpVisionString(body, "name"))
	if !ok {
		respondGCPVisionInvalidArgument(w, path, "name is required")
		return true
	}
	pageSize, start, valid := parseGCPVisionBodyPagination(w, path, body, 10, 100)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpVisionProductFixture(project, location, "product-1", "Stackyard Product 1", "general-v1"),
		gcpVisionProductFixture(project, location, "product-2", "Stackyard Product 2", "general-v1"),
	}
	return respondGCPVisionList(w, "products", items, pageSize, start, path)
}

func handleGCPVisionImportProductSets(w http.ResponseWriter, path string, body map[string]any) bool {
	project, location, ok := parseGCPVisionParentName(gcpVisionString(body, "parent"))
	if !ok {
		respondGCPVisionInvalidArgument(w, path, "parent is required")
		return true
	}
	inputConfig := gcpVisionBodyMap(body, "inputConfig", "input_config")
	if len(inputConfig) == 0 {
		respondGCPVisionInvalidArgument(w, path, "inputConfig is required")
		return true
	}
	gcsSource := gcpVisionBodyMap(inputConfig, "gcsSource", "gcs_source")
	csvURI := strings.TrimSpace(gcpVisionString(gcsSource, "csvFileUri", "csv_file_uri"))
	if !isGCPVisionGCSURI(csvURI) {
		respondGCPVisionInvalidArgument(w, path, "inputConfig.gcsSource.csvFileUri must be a valid gs:// URI")
		return true
	}
	opID := "importProductSets.op-1"
	operation := gcpVisionOperationFixture(project, location, opID, true,
		map[string]any{
			"@type":      "type.googleapis.com/google.cloud.vision.v1.BatchOperationMetadata",
			"state":      "SUCCESSFUL",
			"submitTime": gcpVisionReferenceTime.Format(time.RFC3339Nano),
			"endTime":    gcpVisionReferenceTime.Add(2 * time.Second).Format(time.RFC3339Nano),
		},
		map[string]any{
			"@type": "type.googleapis.com/google.cloud.vision.v1.ImportProductSetsResponse",
			"referenceImages": []any{
				gcpVisionReferenceImageFixture(project, location, "product-1", "reference-image-1", "gs://stackyard-inputs/reference-image-1.jpg"),
			},
			"statuses": []any{map[string]any{"code": 0, "message": "ok"}},
		},
	)
	respondJSON(w, http.StatusOK, operation)
	return true
}

func handleGCPVisionPurgeProducts(w http.ResponseWriter, path string, body map[string]any) bool {
	project, location, ok := parseGCPVisionParentName(gcpVisionString(body, "parent"))
	if !ok {
		respondGCPVisionInvalidArgument(w, path, "parent is required")
		return true
	}
	purgeConfig := gcpVisionBodyMap(body, "productSetPurgeConfig", "product_set_purge_config")
	deleteOrphan, _ := gcpVisionBool(body, "deleteOrphanProducts", "delete_orphan_products")
	if len(purgeConfig) == 0 && !deleteOrphan {
		respondGCPVisionInvalidArgument(w, path, "one purge target is required")
		return true
	}
	if len(purgeConfig) > 0 {
		if strings.TrimSpace(gcpVisionString(purgeConfig, "productSetId", "product_set_id")) == "" {
			respondGCPVisionInvalidArgument(w, path, "productSetPurgeConfig.productSetId is required")
			return true
		}
	}
	opID := "purgeProducts.op-1"
	operation := gcpVisionOperationFixture(project, location, opID, true,
		map[string]any{
			"@type":      "type.googleapis.com/google.cloud.vision.v1.BatchOperationMetadata",
			"state":      "SUCCESSFUL",
			"submitTime": gcpVisionReferenceTime.Format(time.RFC3339Nano),
			"endTime":    gcpVisionReferenceTime.Add(2 * time.Second).Format(time.RFC3339Nano),
		},
		nil,
	)
	respondJSON(w, http.StatusOK, operation)
	return true
}

func handleGCPVisionListOperations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, ok := parseGCPVisionOperationCollectionPath(path)
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPVisionQueryPagination(w, r, path, 100, 1000)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpVisionOperationFixture(project, location, "vision-op-1", true, map[string]any{"@type": "type.googleapis.com/google.cloud.vision.v1.OperationMetadata", "state": "DONE"}, map[string]any{"@type": "type.googleapis.com/google.protobuf.Empty"}),
		gcpVisionOperationFixture(project, location, "vision-op-2", true, map[string]any{"@type": "type.googleapis.com/google.cloud.vision.v1.BatchOperationMetadata", "state": "SUCCESSFUL"}, nil),
	}
	return respondGCPVisionList(w, "operations", items, pageSize, start, path)
}

func handleGCPVisionGetOperation(w http.ResponseWriter, path string) bool {
	project, location, operationID, ok := parseGCPVisionOperationPath(path)
	if !ok {
		return false
	}
	if isGCPVisionMissingID(operationID) {
		respondGCPVisionNotFound(w, path, "operation not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpVisionOperationFixture(project, location, operationID, true, map[string]any{"@type": "type.googleapis.com/google.cloud.vision.v1.OperationMetadata", "state": "DONE"}, map[string]any{"@type": "type.googleapis.com/google.protobuf.Empty"}))
	return true
}

func handleGCPVisionCancelOperation(w http.ResponseWriter, path string) bool {
	_, _, operationID, action, ok := parseGCPVisionOperationActionPath(path)
	if !ok || action != "cancel" {
		return false
	}
	if isGCPVisionMissingID(operationID) {
		respondGCPVisionNotFound(w, path, "operation not found")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPVisionDeleteOperation(w http.ResponseWriter, path string) bool {
	_, _, operationID, ok := parseGCPVisionOperationPath(path)
	if !ok {
		return false
	}
	if isGCPVisionMissingID(operationID) {
		respondGCPVisionNotFound(w, path, "operation not found")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func gcpVisionLocation(project, location string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("projects/%s/locations/%s", project, location),
		"locationId":  location,
		"displayName": strings.ToUpper(location),
		"labels": map[string]string{
			"service":  "vision",
			"provider": providerGCP,
		},
	}
}

func gcpVisionProductSetFixture(project, location, productSetID, displayName string) map[string]any {
	if strings.TrimSpace(displayName) == "" {
		displayName = "Stackyard Product Set " + productSetID
	}
	return map[string]any{
		"name":        fmt.Sprintf("projects/%s/locations/%s/productSets/%s", project, location, productSetID),
		"displayName": displayName,
		"indexTime":   gcpVisionReferenceTime.Format(time.RFC3339Nano),
	}
}

func gcpVisionProductFixture(project, location, productID, displayName, productCategory string) map[string]any {
	if strings.TrimSpace(displayName) == "" {
		displayName = "Stackyard Product " + productID
	}
	if strings.TrimSpace(productCategory) == "" {
		productCategory = "general-v1"
	}
	return map[string]any{
		"name":            fmt.Sprintf("projects/%s/locations/%s/products/%s", project, location, productID),
		"displayName":     displayName,
		"description":     "Stackyard emulated product",
		"productCategory": productCategory,
		"productLabels": []any{
			map[string]any{"key": "env", "value": "test"},
			map[string]any{"key": "service", "value": "vision"},
		},
	}
}

func gcpVisionReferenceImageFixture(project, location, productID, imageID, uri string) map[string]any {
	if strings.TrimSpace(uri) == "" {
		uri = "gs://stackyard-inputs/" + imageID + ".jpg"
	}
	return map[string]any{
		"name": fmt.Sprintf("projects/%s/locations/%s/products/%s/referenceImages/%s", project, location, productID, imageID),
		"uri":  uri,
		"boundingPolys": []any{
			map[string]any{
				"vertices": []any{
					map[string]any{"x": 0, "y": 0},
					map[string]any{"x": 512, "y": 0},
					map[string]any{"x": 512, "y": 512},
					map[string]any{"x": 0, "y": 512},
				},
			},
		},
	}
}

func gcpVisionOperationFixture(project, location, operationID string, done bool, metadata map[string]any, response map[string]any) map[string]any {
	op := map[string]any{
		"name": fmt.Sprintf("projects/%s/locations/%s/operations/%s", project, location, operationID),
		"done": done,
	}
	if metadata != nil {
		op["metadata"] = metadata
	}
	if response != nil {
		op["response"] = response
	}
	return op
}

func gcpVisionAnnotateImageResponseMap(project, location, imageURI string, features []string) map[string]any {
	featureSet := map[string]struct{}{}
	for _, feature := range features {
		featureSet[strings.ToUpper(strings.TrimSpace(feature))] = struct{}{}
	}
	if len(featureSet) == 0 {
		featureSet["LABEL_DETECTION"] = struct{}{}
	}
	resp := map[string]any{}

	if _, ok := featureSet["LABEL_DETECTION"]; ok {
		resp["labelAnnotations"] = []any{
			map[string]any{"mid": "/m/stackyard", "description": "stackyard-label", "score": 0.98},
		}
	}
	if _, ok := featureSet["TEXT_DETECTION"]; ok {
		resp["textAnnotations"] = []any{
			map[string]any{"locale": "en", "description": "Stackyard Vision Text"},
		}
	}
	if _, ok := featureSet["DOCUMENT_TEXT_DETECTION"]; ok {
		resp["fullTextAnnotation"] = map[string]any{
			"text": "Stackyard Vision Document",
			"pages": []any{
				map[string]any{"width": 1000, "height": 1000},
			},
		}
	}
	if _, ok := featureSet["SAFE_SEARCH_DETECTION"]; ok {
		resp["safeSearchAnnotation"] = map[string]any{
			"adult":    "VERY_UNLIKELY",
			"spoof":    "VERY_UNLIKELY",
			"medical":  "VERY_UNLIKELY",
			"violence": "VERY_UNLIKELY",
			"racy":     "VERY_UNLIKELY",
		}
	}
	if _, ok := featureSet["WEB_DETECTION"]; ok {
		resp["webDetection"] = map[string]any{
			"bestGuessLabels": []any{map[string]any{"label": "stackyard-web"}},
			"webEntities":     []any{map[string]any{"entityId": "stackyard", "score": 0.75, "description": "Stackyard"}},
		}
	}
	if _, ok := featureSet["OBJECT_LOCALIZATION"]; ok {
		resp["localizedObjectAnnotations"] = []any{
			map[string]any{"name": "Object", "score": 0.88},
		}
	}
	if _, ok := featureSet["PRODUCT_SEARCH"]; ok {
		resp["productSearchResults"] = map[string]any{
			"results": []any{
				map[string]any{
					"score":   0.91,
					"product": gcpVisionProductFixture(project, location, "product-1", "Stackyard Product 1", "general-v1"),
					"image":   gcpVisionReferenceImageFixture(project, location, "product-1", "reference-image-1", "gs://stackyard-inputs/reference-image-1.jpg"),
				},
			},
		}
	}
	if _, ok := featureSet["IMAGE_PROPERTIES"]; ok {
		resp["imagePropertiesAnnotation"] = map[string]any{"dominantColors": map[string]any{"colors": []any{map[string]any{"score": 0.77}}}}
	}
	if _, ok := featureSet["CROP_HINTS"]; ok {
		resp["cropHintsAnnotation"] = map[string]any{"cropHints": []any{map[string]any{"confidence": 0.85}}}
	}
	if _, ok := featureSet["FACE_DETECTION"]; ok {
		resp["faceAnnotations"] = []any{map[string]any{"detectionConfidence": 0.76}}
	}
	if _, ok := featureSet["LANDMARK_DETECTION"]; ok {
		resp["landmarkAnnotations"] = []any{map[string]any{"description": "Stackyard Landmark", "score": 0.81}}
	}
	if _, ok := featureSet["LOGO_DETECTION"]; ok {
		resp["logoAnnotations"] = []any{map[string]any{"description": "Stackyard Logo", "score": 0.82}}
	}

	resp["context"] = map[string]any{"uri": imageURI, "pageNumber": 1}
	return resp
}

func parseGCPVisionParentName(parent string) (project, location string, ok bool) {
	parts := strings.Split(strings.Trim(parent, "/"), "/")
	if len(parts) != 4 || parts[0] != "projects" || parts[2] != "locations" {
		return "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	if project == "" || location == "" {
		return "", "", false
	}
	return project, location, true
}

func parseGCPVisionProductSetName(name string) (project, location, productSetID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "productSets" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	productSetID = strings.TrimSpace(parts[5])
	if project == "" || location == "" || productSetID == "" {
		return "", "", "", false
	}
	return project, location, productSetID, true
}

func parseGCPVisionProductName(name string) (project, location, productID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "products" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	productID = strings.TrimSpace(parts[5])
	if project == "" || location == "" || productID == "" {
		return "", "", "", false
	}
	return project, location, productID, true
}

func parseGCPVisionReferenceImageName(name string) (project, location, productID, imageID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 8 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "products" || parts[6] != "referenceImages" {
		return "", "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	productID = strings.TrimSpace(parts[5])
	imageID = strings.TrimSpace(parts[7])
	if project == "" || location == "" || productID == "" || imageID == "" {
		return "", "", "", "", false
	}
	return project, location, productID, imageID, true
}

func parseGCPVisionOperationCollectionPath(path string) (project, location string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 7 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "operations" {
		return "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	if project == "" || location == "" {
		return "", "", false
	}
	return project, location, true
}

func parseGCPVisionOperationPath(path string) (project, location, operationID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 8 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "operations" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	operationID = strings.TrimSpace(parts[7])
	if project == "" || location == "" || operationID == "" {
		return "", "", "", false
	}
	if strings.Contains(operationID, ":") {
		return "", "", "", false
	}
	return project, location, operationID, true
}

func parseGCPVisionOperationActionPath(path string) (project, location, operationID, action string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 8 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "operations" {
		return "", "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	operationAndAction := strings.TrimSpace(parts[7])
	operationID, action, ok = strings.Cut(operationAndAction, ":")
	if !ok {
		return "", "", "", "", false
	}
	operationID = strings.TrimSpace(operationID)
	action = strings.TrimSpace(action)
	if project == "" || location == "" || operationID == "" || action == "" {
		return "", "", "", "", false
	}
	return project, location, operationID, action, true
}

func parseGCPVisionQueryPagination(w http.ResponseWriter, r *http.Request, path string, defaultPageSize, maxPageSize int) (pageSize, start int, ok bool) {
	pageSize = defaultPageSize
	if raw := strings.TrimSpace(r.URL.Query().Get("pageSize")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 0 {
			respondGCPVisionInvalidArgument(w, path, "pageSize must be a non-negative integer")
			return 0, 0, false
		}
		if value > maxPageSize {
			respondGCPVisionInvalidArgument(w, path, fmt.Sprintf("pageSize must be <= %d", maxPageSize))
			return 0, 0, false
		}
		if value > 0 {
			pageSize = value
		}
	}
	start = 0
	if raw := strings.TrimSpace(r.URL.Query().Get("pageToken")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 0 {
			respondGCPVisionInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
			return 0, 0, false
		}
		start = value
	}
	return pageSize, start, true
}

func parseGCPVisionBodyPagination(w http.ResponseWriter, path string, body map[string]any, defaultPageSize, maxPageSize int) (pageSize, start int, ok bool) {
	pageSize = defaultPageSize
	if raw, exists := gcpVisionRaw(body, "pageSize", "page_size"); exists {
		value, valid := gcpVisionInt(raw)
		if !valid || value < 0 {
			respondGCPVisionInvalidArgument(w, path, "pageSize must be a non-negative integer")
			return 0, 0, false
		}
		if value > maxPageSize {
			respondGCPVisionInvalidArgument(w, path, fmt.Sprintf("pageSize must be <= %d", maxPageSize))
			return 0, 0, false
		}
		if value > 0 {
			pageSize = value
		}
	}
	start = 0
	if raw, exists := gcpVisionRaw(body, "pageToken", "page_token"); exists {
		value, valid := gcpVisionInt(raw)
		if !valid || value < 0 {
			respondGCPVisionInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
			return 0, 0, false
		}
		start = value
	}
	return pageSize, start, true
}

func respondGCPVisionList(w http.ResponseWriter, field string, items []map[string]any, pageSize, start int, path string) bool {
	if start > len(items) {
		respondGCPVisionInvalidArgument(w, path, "pageToken is out of range")
		return false
	}
	end := len(items)
	if pageSize > 0 && start+pageSize < end {
		end = start + pageSize
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	respondJSON(w, http.StatusOK, map[string]any{
		field:           items[start:end],
		"nextPageToken": next,
	})
	return true
}

func gcpVisionString(body map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := body[key]; ok {
			switch typed := value.(type) {
			case string:
				if strings.TrimSpace(typed) != "" {
					return typed
				}
			case json.Number:
				return typed.String()
			}
		}
	}
	return ""
}

func gcpVisionRaw(body map[string]any, keys ...string) (any, bool) {
	for _, key := range keys {
		if value, ok := body[key]; ok {
			return value, true
		}
	}
	return nil, false
}

func gcpVisionBodyMap(body map[string]any, keys ...string) map[string]any {
	for _, key := range keys {
		if value, ok := body[key]; ok {
			if out, ok := value.(map[string]any); ok {
				return out
			}
		}
	}
	return map[string]any{}
}

func gcpVisionBodyArray(body map[string]any, keys ...string) []any {
	for _, key := range keys {
		if value, ok := body[key]; ok {
			if out, ok := value.([]any); ok {
				return out
			}
		}
	}
	return nil
}

func gcpVisionInt(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int32:
		return int(typed), true
	case int64:
		return int(typed), true
	case float64:
		if typed != float64(int(typed)) {
			return 0, false
		}
		return int(typed), true
	case json.Number:
		v, err := typed.Int64()
		if err != nil {
			return 0, false
		}
		return int(v), true
	case string:
		v, err := strconv.Atoi(strings.TrimSpace(typed))
		if err != nil {
			return 0, false
		}
		return v, true
	default:
		return 0, false
	}
}

func gcpVisionBool(body map[string]any, keys ...string) (bool, bool) {
	for _, key := range keys {
		if value, ok := body[key]; ok {
			switch typed := value.(type) {
			case bool:
				return typed, true
			case string:
				switch strings.ToLower(strings.TrimSpace(typed)) {
				case "true":
					return true, true
				case "false":
					return false, true
				}
			}
			return false, false
		}
	}
	return false, false
}

func gcpVisionIntSlice(value any) []int {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]int, 0, len(items))
	for _, item := range items {
		if v, ok := gcpVisionInt(item); ok {
			out = append(out, v)
		}
	}
	return out
}

func gcpVisionParseFeatureNames(req map[string]any) ([]string, string) {
	features := gcpVisionBodyArray(req, "features")
	if len(features) == 0 {
		return nil, "requests.features is required"
	}
	out := make([]string, 0, len(features))
	for _, raw := range features {
		featureMap, ok := raw.(map[string]any)
		if !ok {
			return nil, "requests.features entries must be objects"
		}
		featureName := strings.TrimSpace(gcpVisionString(featureMap, "type"))
		if featureName == "" {
			if number, ok := gcpVisionRaw(featureMap, "type"); ok {
				if num, valid := gcpVisionInt(number); valid {
					featureName = gcpVisionFeatureByNumber[num]
				}
			}
		}
		featureName = strings.ToUpper(strings.TrimSpace(featureName))
		if featureName == "" || featureName == "TYPE_UNSPECIFIED" {
			return nil, "requests.features.type must be specified"
		}
		out = append(out, featureName)
	}
	return out, ""
}

func gcpVisionImageInputPresent(image map[string]any) bool {
	if strings.TrimSpace(gcpVisionString(image, "content")) != "" {
		return true
	}
	source := gcpVisionBodyMap(image, "source")
	if len(source) == 0 {
		return false
	}
	uri := strings.TrimSpace(gcpVisionString(source, "imageUri", "image_uri", "gcsImageUri", "gcs_image_uri"))
	return uri != ""
}

func gcpVisionImageURIFromMap(image map[string]any) string {
	source := gcpVisionBodyMap(image, "source")
	if len(source) == 0 {
		return ""
	}
	return strings.TrimSpace(gcpVisionString(source, "imageUri", "image_uri", "gcsImageUri", "gcs_image_uri"))
}

func gcpVisionInputConfigURI(inputConfig map[string]any) string {
	gcsSource := gcpVisionBodyMap(inputConfig, "gcsSource", "gcs_source")
	if len(gcsSource) > 0 {
		return strings.TrimSpace(gcpVisionString(gcsSource, "uri"))
	}
	return ""
}

func gcpVisionInputConfigMimeType(inputConfig map[string]any) string {
	mime := strings.TrimSpace(gcpVisionString(inputConfig, "mimeType", "mime_type"))
	if mime == "" {
		mime = "application/pdf"
	}
	return mime
}

func gcpVisionOutputConfigURI(outputConfig map[string]any) string {
	gcsDestination := gcpVisionBodyMap(outputConfig, "gcsDestination", "gcs_destination")
	if len(gcsDestination) == 0 {
		return ""
	}
	return strings.TrimSpace(gcpVisionString(gcsDestination, "uri"))
}

func gcpVisionOutputBatchSize(outputConfig map[string]any) int {
	if value, exists := gcpVisionRaw(outputConfig, "batchSize", "batch_size"); exists {
		if parsed, ok := gcpVisionInt(value); ok && parsed > 0 {
			return parsed
		}
	}
	return 20
}

func gcpVisionHasUpdateMask(body map[string]any, keys ...string) bool {
	for _, key := range keys {
		if value, ok := body[key]; ok {
			switch typed := value.(type) {
			case string:
				return strings.TrimSpace(typed) != ""
			case map[string]any:
				paths := gcpVisionBodyArray(typed, "paths")
				return len(paths) > 0
			}
		}
	}
	return false
}

func gcpVisionProjectLocationFromParent(parent, fallbackProject, fallbackLocation string) (project, location string) {
	if p, l, ok := parseGCPVisionParentName(strings.TrimSpace(parent)); ok {
		return p, l
	}
	return fallbackProject, fallbackLocation
}

func isGCPVisionGCSURI(value string) bool {
	trimmed := strings.TrimSpace(value)
	if !strings.HasPrefix(trimmed, "gs://") {
		return false
	}
	trimmed = strings.TrimPrefix(trimmed, "gs://")
	return strings.Contains(trimmed, "/") && !strings.HasSuffix(trimmed, "/")
}

func isGCPVisionGCSOutputURI(value string) bool {
	trimmed := strings.TrimSpace(value)
	if !strings.HasPrefix(trimmed, "gs://") {
		return false
	}
	trimmed = strings.TrimPrefix(trimmed, "gs://")
	return strings.Contains(trimmed, "/")
}

func isGCPVisionResourceID(value string) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || len(trimmed) > 128 {
		return false
	}
	return !strings.Contains(trimmed, "/")
}

func isGCPVisionMissingID(id string) bool {
	lower := strings.ToLower(strings.TrimSpace(id))
	return strings.Contains(lower, "missing") || strings.Contains(lower, "not-found") || strings.Contains(lower, "does-not-exist")
}

func respondGCPVisionInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func respondGCPVisionNotFound(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "NotFound",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func respondGCPVisionAlreadyExists(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "AlreadyExists",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func handleGCPContractProbe_vision(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "vision") &&
		!isGCPContractProbeRequestForService(r, path, "vision-apiv1") &&
		!isGCPContractProbeRequestForService(r, path, "cloud-vision") &&
		!isGCPContractProbeRequestForService(r, path, "vision-v2") &&
		!isGCPContractProbeRequestForService(r, path, "vision-v2-apiv1") &&
		!isGCPContractProbeRequestForService(r, path, "cloud-vision-v2") {
		return false
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("pageSize")); raw != "" {
		if _, err := parseOptionalNonNegativeInt(raw); err != nil {
			respondGCPVisionInvalidArgument(w, path, "pageSize must be a non-negative integer")
			return true
		}
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"name":         "projects/stackyard/locations/us-central1/vision/sample",
		"service":      "vision",
		"provider":     providerGCP,
		"typedSuccess": true,
		"resource":     "projects/stackyard/locations/us-central1/productSets/product-set-1",
		"operation":    "projects/stackyard/locations/us-central1/operations/vision-op-1",
	})
	return true
}
