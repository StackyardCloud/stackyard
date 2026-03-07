package server

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	gcpShoppingMerchantOrdertrackingAccountRe    = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
	gcpShoppingMerchantOrdertrackingSignalIDRe   = regexp.MustCompile(`^[A-Za-z0-9._~-]+$`)
	gcpShoppingMerchantOrdertrackingRegionCodeRe = regexp.MustCompile(`^[A-Z]{2}$`)
)

func (s *Server) handleGCPShoppingMerchantOrdertrackingRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_shoppingMerchantOrdertracking(w, r) {
		return true
	}

	path := normalizeGCPShoppingMerchantOrdertrackingPath(rawRequestPath(r))
	if !isGCPShoppingMerchantOrdertrackingPath(path, hasGCPShoppingMerchantOrdertrackingHint(r)) {
		return false
	}

	tail, ok := gcpShoppingMerchantOrdertrackingRESTTail(path)
	if !ok {
		if strings.HasPrefix(path, "/gcp/google.shopping.merchant.ordertracking.v1.") {
			return false
		}
		return false
	}

	switch r.Method {
	case http.MethodPost:
		if handleGCPShoppingMerchantOrdertrackingPOST(w, r, path, tail) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func normalizeGCPShoppingMerchantOrdertrackingPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "%3A", ":")
	path = strings.ReplaceAll(path, "%3a", ":")
	return path
}

func hasGCPShoppingMerchantOrdertrackingHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "shopping_merchant_ordertracking",
		"shopping-merchant-ordertracking",
		"shopping-merchant-ordertracking-apiv1",
		"shopping_merchant_ordertracking_apiv1",
		"merchant_ordertracking",
		"merchant-ordertracking",
		"merchantordertracking",
		"gcp-shopping-merchant-ordertracking":
		return true
	}
	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-shopping-merchant-ordertracking-apiv1") ||
		strings.Contains(ua, "cloud.google.com/go/shopping/merchant/ordertracking")
}

func isGCPShoppingMerchantOrdertrackingPath(path string, includeHint bool) bool {
	if strings.HasPrefix(path, "/gcp/ordertracking/v1/") {
		return true
	}
	if strings.HasPrefix(path, "/gcp/google.shopping.merchant.ordertracking.v1.") {
		return true
	}
	if includeHint && strings.HasPrefix(path, "/gcp/ordertracking/v1") {
		return true
	}
	return false
}

func gcpShoppingMerchantOrdertrackingRESTTail(path string) (string, bool) {
	if !strings.HasPrefix(path, "/gcp/ordertracking/v1/") {
		return "", false
	}
	tail := strings.Trim(strings.TrimSpace(strings.TrimPrefix(path, "/gcp/ordertracking/v1/")), "/")
	if tail == "" {
		return "", false
	}
	return tail, true
}

func handleGCPShoppingMerchantOrdertrackingPOST(w http.ResponseWriter, r *http.Request, path, tail string) bool {
	account, ok := parseGCPShoppingMerchantOrdertrackingCollectionPath(tail)
	if !ok {
		return false
	}
	body, ok := decodeGCPShoppingMerchantOrdertrackingJSONBodyRequired(w, r, path)
	if !ok {
		return true
	}
	fixture, errType, errMessage := buildGCPShoppingMerchantOrdertrackingCreateFixture(account, body)
	if errType != "" {
		switch errType {
		case "AlreadyExists":
			respondGCPShoppingMerchantOrdertrackingAlreadyExists(w, path, errMessage)
		case "NotFound":
			respondGCPShoppingMerchantOrdertrackingNotFound(w, path, errMessage)
		case "FailedPrecondition":
			respondGCPShoppingMerchantOrdertrackingFailedPrecondition(w, path, errMessage)
		default:
			respondGCPShoppingMerchantOrdertrackingInvalidArgument(w, path, errMessage)
		}
		return true
	}
	respondJSON(w, http.StatusOK, fixture)
	return true
}

func parseGCPShoppingMerchantOrdertrackingCollectionPath(tail string) (account string, ok bool) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(tail), "/"), "/")
	if len(parts) != 3 || parts[0] != "accounts" || !strings.EqualFold(parts[2], "orderTrackingSignals") {
		return "", false
	}
	return parseGCPShoppingMerchantOrdertrackingParent(fmt.Sprintf("accounts/%s", strings.TrimSpace(parts[1])))
}

func parseGCPShoppingMerchantOrdertrackingParent(parent string) (account string, ok bool) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(parent), "/"), "/")
	if len(parts) != 2 || parts[0] != "accounts" {
		return "", false
	}
	account = strings.TrimSpace(parts[1])
	if !gcpShoppingMerchantOrdertrackingAccountRe.MatchString(account) {
		return "", false
	}
	return account, true
}

func decodeGCPShoppingMerchantOrdertrackingJSONBodyRequired(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r == nil || r.Body == nil {
		respondGCPShoppingMerchantOrdertrackingInvalidArgument(w, path, "request body is required")
		return nil, false
	}
	payload, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		respondGCPShoppingMerchantOrdertrackingInvalidArgument(w, path, "unable to read request body")
		return nil, false
	}
	if len(strings.TrimSpace(string(payload))) == 0 {
		respondGCPShoppingMerchantOrdertrackingInvalidArgument(w, path, "request body is required")
		return nil, false
	}
	var body map[string]any
	if err := json.Unmarshal(payload, &body); err != nil {
		respondGCPShoppingMerchantOrdertrackingInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if len(body) == 0 {
		respondGCPShoppingMerchantOrdertrackingInvalidArgument(w, path, "request body is required")
		return nil, false
	}
	return body, true
}

func buildGCPShoppingMerchantOrdertrackingCreateFixture(account string, body map[string]any) (map[string]any, string, string) {
	if strings.Contains(strings.ToLower(account), "missing") {
		return nil, "NotFound", "merchant account not found"
	}

	requestSignalID := strings.TrimSpace(gcpShoppingMerchantOrdertrackingString(body, "orderTrackingSignalId", "order_tracking_signal_id"))
	if requestSignalID != "" && !gcpShoppingMerchantOrdertrackingSignalIDRe.MatchString(requestSignalID) {
		return nil, "InvalidArgument", "orderTrackingSignalId must contain only URL-safe characters"
	}

	signalBody := gcpShoppingMerchantOrdertrackingMap(body, "orderTrackingSignal", "order_tracking_signal")
	if len(signalBody) == 0 {
		return nil, "InvalidArgument", "orderTrackingSignal is required"
	}

	orderCreatedTime, ok := gcpShoppingMerchantOrdertrackingNormalizeDateTime(signalBody, "orderCreatedTime", "order_created_time")
	if !ok {
		return nil, "InvalidArgument", "orderTrackingSignal.orderCreatedTime is required"
	}

	orderID := strings.TrimSpace(gcpShoppingMerchantOrdertrackingString(signalBody, "orderId", "order_id"))
	if orderID == "" {
		return nil, "InvalidArgument", "orderTrackingSignal.orderId is required"
	}
	if strings.Contains(strings.ToLower(orderID), "existing") || strings.Contains(strings.ToLower(orderID), "duplicate") ||
		strings.Contains(strings.ToLower(requestSignalID), "existing") || strings.Contains(strings.ToLower(requestSignalID), "duplicate") {
		return nil, "AlreadyExists", "order tracking signal already exists"
	}

	merchantID := gcpShoppingMerchantOrdertrackingInt64(signalBody, "merchantId", "merchant_id")
	if merchantID <= 0 {
		merchantID = gcpShoppingMerchantOrdertrackingNumericAccountID(account, 123456)
	}

	shippingRaw := gcpShoppingMerchantOrdertrackingSlice(signalBody, "shippingInfo", "shipping_info")
	if len(shippingRaw) == 0 {
		return nil, "InvalidArgument", "orderTrackingSignal.shippingInfo must include at least one shipment"
	}

	knownShipments := make(map[string]struct{})
	shippingInfo := make([]map[string]any, 0, len(shippingRaw))
	for i, item := range shippingRaw {
		shippingMap, ok := item.(map[string]any)
		if !ok {
			return nil, "InvalidArgument", fmt.Sprintf("orderTrackingSignal.shippingInfo[%d] must be an object", i)
		}
		shipmentID := strings.TrimSpace(gcpShoppingMerchantOrdertrackingString(shippingMap, "shipmentId", "shipment_id"))
		if shipmentID == "" {
			return nil, "InvalidArgument", fmt.Sprintf("orderTrackingSignal.shippingInfo[%d].shipmentId is required", i)
		}
		shippingStatus, ok := gcpShoppingMerchantOrdertrackingShippingState(shippingMap)
		if !ok || shippingStatus == 0 {
			return nil, "InvalidArgument", fmt.Sprintf("orderTrackingSignal.shippingInfo[%d].shippingStatus is required and must be valid", i)
		}
		originPostalCode := strings.TrimSpace(gcpShoppingMerchantOrdertrackingString(shippingMap, "originPostalCode", "origin_postal_code"))
		if originPostalCode == "" {
			return nil, "InvalidArgument", fmt.Sprintf("orderTrackingSignal.shippingInfo[%d].originPostalCode is required", i)
		}
		originRegionCode := strings.ToUpper(strings.TrimSpace(gcpShoppingMerchantOrdertrackingString(shippingMap, "originRegionCode", "origin_region_code")))
		if !gcpShoppingMerchantOrdertrackingRegionCodeRe.MatchString(originRegionCode) {
			return nil, "InvalidArgument", fmt.Sprintf("orderTrackingSignal.shippingInfo[%d].originRegionCode must be a CLDR territory code", i)
		}

		trackingID := strings.TrimSpace(gcpShoppingMerchantOrdertrackingString(shippingMap, "trackingId", "tracking_id"))
		carrier := strings.TrimSpace(gcpShoppingMerchantOrdertrackingString(shippingMap, "carrier"))
		shippedTime, hasShippedTime := gcpShoppingMerchantOrdertrackingNormalizeDateTime(shippingMap, "shippedTime", "shipped_time")
		earliestPromise, hasEarliestPromise := gcpShoppingMerchantOrdertrackingNormalizeDateTime(shippingMap, "earliestDeliveryPromiseTime", "earliest_delivery_promise_time")
		latestPromise, hasLatestPromise := gcpShoppingMerchantOrdertrackingNormalizeDateTime(shippingMap, "latestDeliveryPromiseTime", "latest_delivery_promise_time")
		actualDelivery, hasActualDelivery := gcpShoppingMerchantOrdertrackingNormalizeDateTime(shippingMap, "actualDeliveryTime", "actual_delivery_time")

		hasPromiseOrActual := hasEarliestPromise || hasLatestPromise || hasActualDelivery
		if !hasPromiseOrActual && (trackingID == "" || carrier == "") {
			return nil, "InvalidArgument", fmt.Sprintf("orderTrackingSignal.shippingInfo[%d] requires trackingId and carrier when promise/delivery times are omitted", i)
		}
		if shippingStatus == 2 && !hasActualDelivery {
			return nil, "FailedPrecondition", fmt.Sprintf("orderTrackingSignal.shippingInfo[%d] with DELIVERED status requires actualDeliveryTime", i)
		}

		knownShipments[shipmentID] = struct{}{}
		out := map[string]any{
			"shipmentId":       gcpShoppingMerchantOrdertrackingMaskedShipmentID(shipmentID),
			"shippingStatus":   shippingStatus,
			"originPostalCode": gcpShoppingMerchantOrdertrackingMaskedPostalCode(originPostalCode),
			"originRegionCode": originRegionCode,
		}
		if trackingID != "" {
			out["trackingId"] = trackingID
		}
		if carrier != "" {
			out["carrier"] = carrier
		}
		if carrierService := strings.TrimSpace(gcpShoppingMerchantOrdertrackingString(shippingMap, "carrierService", "carrier_service")); carrierService != "" {
			out["carrierService"] = carrierService
		}
		if hasShippedTime {
			out["shippedTime"] = shippedTime
		}
		if hasEarliestPromise {
			out["earliestDeliveryPromiseTime"] = earliestPromise
		}
		if hasLatestPromise {
			out["latestDeliveryPromiseTime"] = latestPromise
		}
		if hasActualDelivery {
			out["actualDeliveryTime"] = actualDelivery
		}
		shippingInfo = append(shippingInfo, out)
	}

	lineItemsRaw := gcpShoppingMerchantOrdertrackingSlice(signalBody, "lineItems", "line_items")
	if len(lineItemsRaw) == 0 {
		return nil, "InvalidArgument", "orderTrackingSignal.lineItems must include at least one line item"
	}
	knownLineItems := make(map[string]struct{})
	lineItems := make([]map[string]any, 0, len(lineItemsRaw))
	for i, item := range lineItemsRaw {
		lineItemMap, ok := item.(map[string]any)
		if !ok {
			return nil, "InvalidArgument", fmt.Sprintf("orderTrackingSignal.lineItems[%d] must be an object", i)
		}
		lineItemID := strings.TrimSpace(gcpShoppingMerchantOrdertrackingString(lineItemMap, "lineItemId", "line_item_id"))
		if lineItemID == "" {
			return nil, "InvalidArgument", fmt.Sprintf("orderTrackingSignal.lineItems[%d].lineItemId is required", i)
		}
		productID := strings.TrimSpace(gcpShoppingMerchantOrdertrackingString(lineItemMap, "productId", "product_id"))
		if !gcpShoppingMerchantOrdertrackingValidProductID(productID) {
			return nil, "InvalidArgument", fmt.Sprintf("orderTrackingSignal.lineItems[%d].productId must be in channel:contentLanguage:targetCountry:offerId format", i)
		}
		quantity := gcpShoppingMerchantOrdertrackingInt64(lineItemMap, "quantity")
		if quantity <= 0 {
			return nil, "InvalidArgument", fmt.Sprintf("orderTrackingSignal.lineItems[%d].quantity must be > 0", i)
		}
		knownLineItems[lineItemID] = struct{}{}
		out := map[string]any{
			"lineItemId": lineItemID,
			"productId":  productID,
			"quantity":   strconv.FormatInt(quantity, 10),
		}
		if gtins := gcpShoppingMerchantOrdertrackingStringSlice(lineItemMap, "gtins"); len(gtins) > 0 {
			out["gtins"] = gtins
		}
		if mpn := strings.TrimSpace(gcpShoppingMerchantOrdertrackingString(lineItemMap, "mpn")); mpn != "" {
			out["mpn"] = mpn
		}
		if productTitle := strings.TrimSpace(gcpShoppingMerchantOrdertrackingString(lineItemMap, "productTitle", "product_title")); productTitle != "" {
			out["productTitle"] = productTitle
		}
		if brand := strings.TrimSpace(gcpShoppingMerchantOrdertrackingString(lineItemMap, "brand")); brand != "" {
			out["brand"] = brand
		}
		lineItems = append(lineItems, out)
	}

	mappingRaw := gcpShoppingMerchantOrdertrackingSlice(signalBody, "shipmentLineItemMapping", "shipment_line_item_mapping")
	mapping := make([]map[string]any, 0, len(mappingRaw))
	for i, item := range mappingRaw {
		mappingMap, ok := item.(map[string]any)
		if !ok {
			return nil, "InvalidArgument", fmt.Sprintf("orderTrackingSignal.shipmentLineItemMapping[%d] must be an object", i)
		}
		shipmentID := strings.TrimSpace(gcpShoppingMerchantOrdertrackingString(mappingMap, "shipmentId", "shipment_id"))
		lineItemID := strings.TrimSpace(gcpShoppingMerchantOrdertrackingString(mappingMap, "lineItemId", "line_item_id"))
		quantity := gcpShoppingMerchantOrdertrackingInt64(mappingMap, "quantity")
		if shipmentID == "" || lineItemID == "" || quantity <= 0 {
			return nil, "InvalidArgument", fmt.Sprintf("orderTrackingSignal.shipmentLineItemMapping[%d] requires shipmentId, lineItemId, and quantity > 0", i)
		}
		if _, ok := knownShipments[shipmentID]; !ok {
			return nil, "FailedPrecondition", fmt.Sprintf("orderTrackingSignal.shipmentLineItemMapping[%d].shipmentId must reference shippingInfo", i)
		}
		if _, ok := knownLineItems[lineItemID]; !ok {
			return nil, "FailedPrecondition", fmt.Sprintf("orderTrackingSignal.shipmentLineItemMapping[%d].lineItemId must reference lineItems", i)
		}
		mapping = append(mapping, map[string]any{
			"shipmentId": gcpShoppingMerchantOrdertrackingMaskedShipmentID(shipmentID),
			"lineItemId": lineItemID,
			"quantity":   strconv.FormatInt(quantity, 10),
		})
	}

	customerShippingFee := gcpShoppingMerchantOrdertrackingMap(signalBody, "customerShippingFee", "customer_shipping_fee")
	customerShippingFeeOut := map[string]any(nil)
	if len(customerShippingFee) > 0 {
		currencyCode := strings.ToUpper(strings.TrimSpace(gcpShoppingMerchantOrdertrackingString(customerShippingFee, "currencyCode", "currency_code")))
		amountMicros := gcpShoppingMerchantOrdertrackingInt64(customerShippingFee, "amountMicros", "amount_micros")
		if !gcpShoppingMerchantOrdertrackingCurrencyCode(currencyCode) || amountMicros < 0 {
			return nil, "InvalidArgument", "orderTrackingSignal.customerShippingFee must include valid currencyCode and non-negative amountMicros"
		}
		customerShippingFeeOut = map[string]any{
			"currencyCode": currencyCode,
			"amountMicros": strconv.FormatInt(amountMicros, 10),
		}
	}

	deliveryPostalCode := strings.TrimSpace(gcpShoppingMerchantOrdertrackingString(signalBody, "deliveryPostalCode", "delivery_postal_code"))
	deliveryRegionCode := strings.ToUpper(strings.TrimSpace(gcpShoppingMerchantOrdertrackingString(signalBody, "deliveryRegionCode", "delivery_region_code")))
	if deliveryRegionCode != "" && !gcpShoppingMerchantOrdertrackingRegionCodeRe.MatchString(deliveryRegionCode) {
		return nil, "InvalidArgument", "orderTrackingSignal.deliveryRegionCode must be a CLDR territory code"
	}

	signalID := gcpShoppingMerchantOrdertrackingStableNumber(account + "|" + orderID)
	if requestSignalID != "" {
		if parsed, err := strconv.ParseInt(requestSignalID, 10, 64); err == nil && parsed > 0 {
			signalID = parsed
		}
	}

	out := map[string]any{
		"orderTrackingSignalId": strconv.FormatInt(signalID, 10),
		"merchantId":            strconv.FormatInt(merchantID, 10),
		"orderCreatedTime":      orderCreatedTime,
		"orderId":               gcpShoppingMerchantOrdertrackingMaskedOrderID(orderID),
		"shippingInfo":          shippingInfo,
		"lineItems":             lineItems,
	}
	if len(mapping) > 0 {
		out["shipmentLineItemMapping"] = mapping
	}
	if customerShippingFeeOut != nil {
		out["customerShippingFee"] = customerShippingFeeOut
	}
	if deliveryPostalCode != "" {
		out["deliveryPostalCode"] = gcpShoppingMerchantOrdertrackingMaskedPostalCode(deliveryPostalCode)
	}
	if deliveryRegionCode != "" {
		out["deliveryRegionCode"] = deliveryRegionCode
	}
	return out, "", ""
}

func gcpShoppingMerchantOrdertrackingMaskedOrderID(orderID string) string {
	return "order-" + gcpShoppingMerchantOrdertrackingToken(orderID)
}

func gcpShoppingMerchantOrdertrackingMaskedShipmentID(shipmentID string) string {
	return "shipment-" + gcpShoppingMerchantOrdertrackingToken(shipmentID)
}

func gcpShoppingMerchantOrdertrackingMaskedPostalCode(postalCode string) string {
	return "postal-" + gcpShoppingMerchantOrdertrackingToken(postalCode)
}

func gcpShoppingMerchantOrdertrackingToken(value string) string {
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(strings.ToLower(strings.TrimSpace(value))))
	return fmt.Sprintf("%08x", hasher.Sum32())
}

func gcpShoppingMerchantOrdertrackingStableNumber(value string) int64 {
	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte(strings.ToLower(strings.TrimSpace(value))))
	return int64(hasher.Sum64()%900000000000) + 1000
}

func gcpShoppingMerchantOrdertrackingAny(m map[string]any, keys ...string) (any, bool) {
	for _, key := range keys {
		if value, ok := m[key]; ok {
			return value, true
		}
	}
	return nil, false
}

func gcpShoppingMerchantOrdertrackingMap(m map[string]any, keys ...string) map[string]any {
	raw, ok := gcpShoppingMerchantOrdertrackingAny(m, keys...)
	if !ok {
		return nil
	}
	out, _ := raw.(map[string]any)
	return out
}

func gcpShoppingMerchantOrdertrackingSlice(m map[string]any, keys ...string) []any {
	raw, ok := gcpShoppingMerchantOrdertrackingAny(m, keys...)
	if !ok {
		return nil
	}
	switch typed := raw.(type) {
	case []any:
		return typed
	case []map[string]any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, item)
		}
		return out
	default:
		return nil
	}
}

func gcpShoppingMerchantOrdertrackingString(m map[string]any, keys ...string) string {
	raw, ok := gcpShoppingMerchantOrdertrackingAny(m, keys...)
	if !ok {
		return ""
	}
	switch typed := raw.(type) {
	case string:
		return typed
	case float64:
		return strconv.FormatInt(int64(typed), 10)
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	default:
		return ""
	}
}

func gcpShoppingMerchantOrdertrackingInt64(m map[string]any, keys ...string) int64 {
	raw, ok := gcpShoppingMerchantOrdertrackingAny(m, keys...)
	if !ok {
		return 0
	}
	switch typed := raw.(type) {
	case float64:
		return int64(typed)
	case int:
		return int64(typed)
	case int64:
		return typed
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
		if err != nil {
			return 0
		}
		return parsed
	default:
		return 0
	}
}

func gcpShoppingMerchantOrdertrackingStringSlice(m map[string]any, keys ...string) []string {
	raw := gcpShoppingMerchantOrdertrackingSlice(m, keys...)
	if len(raw) == 0 {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
			out = append(out, strings.TrimSpace(s))
		}
	}
	return out
}

func gcpShoppingMerchantOrdertrackingNormalizeDateTime(m map[string]any, keys ...string) (map[string]any, bool) {
	raw, ok := gcpShoppingMerchantOrdertrackingAny(m, keys...)
	if !ok {
		return nil, false
	}
	switch typed := raw.(type) {
	case map[string]any:
		year := gcpShoppingMerchantOrdertrackingInt64(typed, "year")
		if year <= 0 {
			return nil, false
		}
		out := map[string]any{
			"year": year,
		}
		if month := gcpShoppingMerchantOrdertrackingInt64(typed, "month"); month > 0 {
			out["month"] = month
		}
		if day := gcpShoppingMerchantOrdertrackingInt64(typed, "day"); day > 0 {
			out["day"] = day
		}
		if hours := gcpShoppingMerchantOrdertrackingInt64(typed, "hours"); hours >= 0 {
			out["hours"] = hours
		}
		if minutes := gcpShoppingMerchantOrdertrackingInt64(typed, "minutes"); minutes >= 0 {
			out["minutes"] = minutes
		}
		if seconds := gcpShoppingMerchantOrdertrackingInt64(typed, "seconds"); seconds >= 0 {
			out["seconds"] = seconds
		}
		if nanos := gcpShoppingMerchantOrdertrackingInt64(typed, "nanos"); nanos > 0 {
			out["nanos"] = nanos
		}
		if tz := gcpShoppingMerchantOrdertrackingMap(typed, "timeZone", "time_zone"); len(tz) > 0 {
			out["timeZone"] = tz
		} else {
			out["timeZone"] = map[string]any{"id": "UTC"}
		}
		return out, true
	case string:
		parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(typed))
		if err != nil {
			return nil, false
		}
		return map[string]any{
			"year":    parsed.Year(),
			"month":   int64(parsed.Month()),
			"day":     parsed.Day(),
			"hours":   parsed.Hour(),
			"minutes": parsed.Minute(),
			"seconds": parsed.Second(),
			"nanos":   parsed.Nanosecond(),
			"timeZone": map[string]any{
				"id": "UTC",
			},
		}, true
	default:
		return nil, false
	}
}

func gcpShoppingMerchantOrdertrackingShippingState(m map[string]any) (int64, bool) {
	raw, ok := gcpShoppingMerchantOrdertrackingAny(m, "shippingStatus", "shipping_status")
	if !ok {
		return 0, false
	}
	switch typed := raw.(type) {
	case float64:
		state := int64(typed)
		return state, state >= 0 && state <= 2
	case int:
		state := int64(typed)
		return state, state >= 0 && state <= 2
	case int64:
		return typed, typed >= 0 && typed <= 2
	case string:
		value := strings.ToUpper(strings.TrimSpace(typed))
		switch value {
		case "1", "SHIPPED":
			return 1, true
		case "2", "DELIVERED":
			return 2, true
		case "0", "SHIPPING_STATE_UNSPECIFIED":
			return 0, true
		default:
			return 0, false
		}
	default:
		return 0, false
	}
}

func gcpShoppingMerchantOrdertrackingValidProductID(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	parts := strings.Split(value, ":")
	if len(parts) != 4 {
		return false
	}
	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			return false
		}
	}
	return true
}

func gcpShoppingMerchantOrdertrackingCurrencyCode(value string) bool {
	return len(value) == 3 && value == strings.ToUpper(value)
}

func gcpShoppingMerchantOrdertrackingNumericAccountID(account string, fallback int64) int64 {
	parsed, err := strconv.ParseInt(strings.TrimSpace(account), 10, 64)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func respondGCPShoppingMerchantOrdertrackingInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func respondGCPShoppingMerchantOrdertrackingAlreadyExists(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusConflict, map[string]any{
		"error":    "AlreadyExists",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func respondGCPShoppingMerchantOrdertrackingNotFound(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusNotFound, map[string]any{
		"error":    "NotFound",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func respondGCPShoppingMerchantOrdertrackingFailedPrecondition(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "FailedPrecondition",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func handleGCPContractProbe_shoppingMerchantOrdertracking(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "shopping_merchant_ordertracking") {
		return false
	}
	if handleGCPContractProbeGeneric(w, r) {
		return true
	}
	if r.URL.Query().Get("typedSuccess") == "1" {
		respondJSON(w, http.StatusOK, map[string]any{
			"name":      "projects/stackyard/locations/us-central1/shopping_merchant_ordertracking/sample",
			"service":   "shopping_merchant_ordertracking",
			"provider":  providerGCP,
			"path":      path,
			"timestamp": time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339),
		})
		return true
	}
	return false
}
